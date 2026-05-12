// Package app provides application lifecycle management for the Firefly framework.
// It implements the App struct which manages service startup, shutdown, and signal handling.
package app

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"time"

	"github.com/zhangpeihaoks/firefly/internal/transport"
	"golang.org/x/sync/errgroup"
)

// defaultSigs is the list of default shutdown signals.
// Platform-specific files (app_restart_unix.go) may add signals.
var defaultSigs = []os.Signal{os.Interrupt}

// App is the application lifecycle manager.
// It manages service startup, shutdown, and signal handling.
type App struct {
	ctx        context.Context
	cancel     context.CancelFunc
	opts       options
	lifecycles []Lifecycle
}

// Lifecycle is the interface for components that need start/stop lifecycle management.
// Components implementing this interface are automatically started after servers
// and stopped before servers during App.Run().
type Lifecycle interface {
	// Start is called after all servers have started successfully.
	Start(ctx context.Context) error
	// Stop is called before servers are stopped during shutdown.
	Stop(ctx context.Context) error
}

// Option is the application configuration option function type.
type Option func(o *options)

// options contains all application configuration.
type options struct {
	ctx          context.Context
	name         string
	metadata     map[string]string
	logger       *slog.Logger
	servers      []transport.Server
	started      func()
	sigs         []os.Signal
	stopTimeout  time.Duration
	restartable  bool
	restartSig   os.Signal
}

// New creates a new App instance with the given options.
func New(opts ...Option) *App {
	ctx, cancel := context.WithCancel(context.Background())
	o := options{
		ctx:         ctx,
		stopTimeout: 5 * time.Second,
		sigs:        append([]os.Signal{}, defaultSigs...),
	}

	// SIGTERM is Unix-only; for cross-platform, Interrupt (Ctrl+C) is universal
	// Applications can add SIGTERM via app.Signal() if desired

	for _, opt := range opts {
		opt(&o)
	}

	return &App{
		ctx:    ctx,
		cancel: cancel,
		opts:   o,
	}
}

// Name returns the service name.
func (a *App) Name() string {
	return a.opts.name
}

// Metadata returns the service metadata.
func (a *App) Metadata() map[string]string {
	return a.opts.metadata
}

// Name sets the service name option.
func Name(name string) Option {
	return func(o *options) {
		o.name = name
	}
}

// Metadata sets the service metadata option.
func Metadata(md map[string]string) Option {
	return func(o *options) {
		o.metadata = md
	}
}

// Logger sets the logger option.
func Logger(logger *slog.Logger) Option {
	return func(o *options) {
		o.logger = logger
	}
}

// Server sets the transport servers option.
func Server(srv ...transport.Server) Option {
	return func(o *options) {
		o.servers = append(o.servers, srv...)
	}
}

// StartedFunc sets the startup callback function option.
func StartedFunc(f func()) Option {
	return func(o *options) {
		o.started = f
	}
}

// Signal sets the exit signals option.
func Signal(sigs ...os.Signal) Option {
	return func(o *options) {
		o.sigs = sigs
	}
}

// StopTimeout sets the stop timeout option.
func StopTimeout(t time.Duration) Option {
	return func(o *options) {
		o.stopTimeout = t
	}
}

// EnableRestart enables graceful restart via SIGUSR2 signal.
// When received, the app spawns a new process with inherited file descriptors
// and gracefully shuts down the old one. On Windows, it falls back to
// stop-and-restart.
func EnableRestart() Option {
	return func(o *options) {
		o.restartable = true
	}
}

// RegisterLifecycle registers a component for lifecycle management.
// Components are started after all servers (in registration order) and
// stopped before servers (in reverse order).
func (a *App) RegisterLifecycle(lc Lifecycle) {
	a.lifecycles = append(a.lifecycles, lc)
}

// Run starts the application and manages the lifecycle of all servers.
// It starts all servers concurrently using errgroup, waits for them to start,
// and handles graceful shutdown on context cancellation or signal.
// Returns exit code (0 for success, 1 for error) and any error encountered.
// **Validates: Requirements 1.2, 1.3, 1.4, 1.5, 1.7**
func (a *App) Run() (int, error) {
	// Create a context that can be cancelled on signal
	ctx, cancel := context.WithCancel(a.ctx)
	defer cancel()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, a.opts.sigs...)
	defer signal.Stop(sigChan)

	// Goroutine to cancel context on shutdown signal
	go func() {
		select {
		case sig := <-sigChan:
			if a.opts.logger != nil {
				a.opts.logger.Info("received shutdown signal, initiating graceful shutdown", "signal", sig.String())
			}
			cancel()
		case <-ctx.Done():
		}
	}()

	// Handle restart signal if enabled
	if a.opts.restartable {
		rs := a.opts.restartSig
		if rs == nil {
			rs = restartSignal() // platform-specific default
		}
		if rs != nil {
			restartChan := make(chan os.Signal, 1)
			signal.Notify(restartChan, rs)
			defer signal.Stop(restartChan)

			go func() {
				select {
				case <-restartChan:
					if a.opts.logger != nil {
						a.opts.logger.Info("received restart signal, initiating graceful restart")
					}
					if err := a.restart(ctx); err != nil {
						if a.opts.logger != nil {
							a.opts.logger.Error("restart failed, falling back to shutdown", "error", err)
						}
						cancel()
					}
				case <-ctx.Done():
				}
			}()
		}
	}

	// Create errgroup with the cancellable context
	g, ctx := errgroup.WithContext(ctx)

	// Start all servers concurrently
	for _, srv := range a.opts.servers {
		srv := srv // capture range variable
		g.Go(func() error {
			return srv.Start(ctx)
		})
	}

	// Wait for all servers to start
	if err := g.Wait(); err != nil {
		if a.opts.logger != nil {
			a.opts.logger.Error("server startup failed", "error", err)
		}
		return 1, err
	}

	// All servers started successfully, call the started callback if set
	if a.opts.started != nil {
		a.opts.started()
	}

	if a.opts.logger != nil {
		a.opts.logger.Info("all servers started successfully")
	}

	// Start lifecycle components
	for _, lc := range a.lifecycles {
		if err := lc.Start(ctx); err != nil {
			if a.opts.logger != nil {
				a.opts.logger.Error("lifecycle start failed", "error", err)
			}
			return 1, err
		}
	}

	// Wait for context cancellation (signal or error)
	<-ctx.Done()

	// Stop lifecycle components (reverse order)
	for i := len(a.lifecycles) - 1; i >= 0; i-- {
		if err := a.lifecycles[i].Stop(ctx); err != nil {
			if a.opts.logger != nil {
				a.opts.logger.Error("lifecycle stop failed", "error", err)
			}
		}
	}

	// Handle graceful shutdown with configurable timeout
	if a.opts.stopTimeout == 0 {
		// Immediate shutdown - stop servers without waiting
		if a.opts.logger != nil {
			a.opts.logger.Info("performing immediate shutdown (stopTimeout=0)")
		}
		var stopGroup errgroup.Group
		for _, srv := range a.opts.servers {
			srv := srv // capture range variable
			stopGroup.Go(func() error {
				return srv.Stop(context.Background())
			})
		}
		if err := stopGroup.Wait(); err != nil {
			if a.opts.logger != nil {
				a.opts.logger.Error("server stop failed", "error", err)
			}
			return 1, err
		}
	} else {
		// Graceful shutdown with timeout
		if a.opts.logger != nil {
			a.opts.logger.Info("performing graceful shutdown", "timeout", a.opts.stopTimeout.String())
		}
		stopCtx, stopCancel := context.WithTimeout(context.Background(), a.opts.stopTimeout)
		defer stopCancel()

		// Stop all servers concurrently
		var stopGroup errgroup.Group
		for _, srv := range a.opts.servers {
			srv := srv // capture range variable
			stopGroup.Go(func() error {
				return srv.Stop(stopCtx)
			})
		}

		// Wait for all servers to stop
		if err := stopGroup.Wait(); err != nil {
			if a.opts.logger != nil {
				a.opts.logger.Error("server stop failed", "error", err)
			}
			return 1, err
		}
	}

	if a.opts.logger != nil {
		a.opts.logger.Info("all servers stopped successfully")
	}
	return 0, nil
}

// restart performs a graceful restart by spawning a new process.
// File descriptors for active listeners are extracted and passed to the child.
// On platforms without fd-passing support (Windows), it logs a notice.
func (a *App) restart(ctx context.Context) error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("restart: cannot find executable: %w", err)
	}

	// Collect listener file descriptors from all servers
	var extraFiles []*os.File
	var listenerFds []uintptr

	for _, srv := range a.opts.servers {
		if lp, ok := srv.(interface{ Listener() net.Listener }); ok {
			ln := lp.Listener()
			if ln != nil {
				if fl, ok := ln.(interface{ File() (*os.File, error) }); ok {
					f, err := fl.File()
					if err != nil {
						if a.opts.logger != nil {
							a.opts.logger.Warn("restart: cannot dup listener fd", "error", err)
						}
						continue
					}
					extraFiles = append(extraFiles, f)
					listenerFds = append(listenerFds, f.Fd())
				}
			}
		}
	}

	// Build environment with fd info for child process
	env := os.Environ()
	if len(listenerFds) > 0 {
		env = append(env, fmt.Sprintf("FIREFLY_LISTEN_FDS=%d", len(listenerFds)))
	}

	// Spawn new process
	proc, err := os.StartProcess(execPath, os.Args, &os.ProcAttr{
		Env:        env,
		Files:      append([]*os.File{os.Stdin, os.Stdout, os.Stderr}, extraFiles...),
		Sys:        getSysProcAttr(),
	})
	if err != nil {
		return fmt.Errorf("restart: cannot start new process: %w", err)
	}

	if a.opts.logger != nil {
		a.opts.logger.Info("restart: new process spawned", "pid", proc.Pid)
	}

	// Release the new process - parent will continue with graceful shutdown
	proc.Release()
	return nil
}
