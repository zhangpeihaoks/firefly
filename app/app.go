// Package app provides application lifecycle management for the Firefly framework.
// It implements the App struct which manages service startup, shutdown, and signal handling.
package app

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zhangpeihaoks/firefly/internal/transport"
	"golang.org/x/sync/errgroup"
)

// App is the application lifecycle manager.
// It manages service startup, shutdown, and signal handling.
type App struct {
	ctx    context.Context
	cancel context.CancelFunc
	opts   options
}

// Option is the application configuration option function type.
type Option func(o *options)

// options contains all application configuration.
type options struct {
	ctx         context.Context
	name        string
	metadata    map[string]string
	logger      *slog.Logger
	servers     []transport.Server
	started     func()
	sigs        []os.Signal
	stopTimeout time.Duration
}

// New creates a new App instance with the given options.
// It initializes the application context and applies all provided options.
func New(opts ...Option) *App {
	ctx, cancel := context.WithCancel(context.Background())
	o := options{
		ctx:         ctx,
		stopTimeout: 5 * time.Second,
		sigs:        []os.Signal{os.Interrupt, syscall.SIGTERM},
	}

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

	// Goroutine to cancel context on signal
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

	// Wait for context cancellation (signal or error)
	<-ctx.Done()

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
