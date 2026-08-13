// Package main is the entry point for the Firefly framework.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/zhangpeihaoks/firefly/app"
	"github.com/zhangpeihaoks/firefly/conf"
	"github.com/zhangpeihaoks/firefly/internal/admin"
	"github.com/zhangpeihaoks/firefly/internal/health"
	"github.com/zhangpeihaoks/firefly/internal/log"
	"github.com/zhangpeihaoks/firefly/internal/middleware"
	"github.com/zhangpeihaoks/firefly/internal/registry"
	"github.com/zhangpeihaoks/firefly/internal/registry/consul"
	httptransport "github.com/zhangpeihaoks/firefly/internal/transport/http"
	"github.com/zhangpeihaoks/firefly/pkg/config"
)

func main() {
	configPath := flag.String("config", "./config/config.yaml", "configuration file path")
	flag.Parse()

	cfg := conf.DefaultBootstrap()
	c := config.New(config.WithEnvPrefix("FIREFLY"))
	if err := c.Load(*configPath, cfg); err != nil {
		fmt.Printf("failed to load config: %v\n", err)
		os.Exit(1)
	}

	cleanup := log.New(&log.Config{
		FileName:   cfg.Log.FileName,
		MaxSize:    cfg.Log.MaxSize,
		MaxBackups: cfg.Log.MaxBackups,
		MaxAge:     cfg.Log.MaxAge,
		Level:      cfg.Log.Level,
		JSONFormat: cfg.Log.JSONFormat,
		Location:   cfg.Log.Location,
	})
	defer cleanup()

	logger := slog.Default()
	logger.Info("firefly server starting", "name", cfg.Name, "version", cfg.Version)

	// Enable config file hot-reload
	c.Watch(cfg)

	appOpts := buildAppOptions(cfg, logger)
	firefly := app.New(appOpts...)

	// Register with the service registry (e.g., Consul) if configured.
	if lc, err := buildRegistryLifecycle(cfg, logger); err != nil {
		logger.Error("failed to create service registry", "error", err)
		os.Exit(1)
	} else if lc != nil {
		firefly.RegisterLifecycle(lc)
	}

	if _, err := firefly.Run(); err != nil {
		logger.Error("firefly server stopped with error", "error", err)
		os.Exit(1)
	}
	logger.Info("firefly server stopped gracefully")
}

// buildAppOptions assembles application options from configuration.
func buildAppOptions(cfg *conf.Bootstrap, logger *slog.Logger) []app.Option {
	opts := []app.Option{
		app.Name(cfg.Name),
		app.Metadata(cfg.Metadata),
		app.Logger(logger),
		app.StopTimeout(10 * time.Second),
	}

	if srv := createHTTPServer(cfg, logger); srv != nil {
		opts = append(opts, app.Server(srv))
	}

	// Future: add gRPC server, admin server, etc.
	return opts
}

// createHTTPServer builds and returns an HTTP server if configured, or nil.
func createHTTPServer(cfg *conf.Bootstrap, logger *slog.Logger) *httptransport.Server {
	if cfg.HTTP == nil || cfg.HTTP.Address == "" {
		return nil
	}

	srv := httptransport.NewServer(
		httptransport.Address(cfg.HTTP.Address),
		httptransport.Timeout(cfg.HTTP.Timeout),
		httptransport.Logger(logger),
		httptransport.Middleware(
			middleware.Recovery(middleware.WithRecoveryLogger(logger)),
			middleware.RequestID(),
			middleware.Logging(middleware.WithLoggingLogger(logger)),
			middleware.RateLimit(middleware.WithRateLimiter(middleware.NewTokenBucketLimiter(100, 200))),
			middleware.Timeout(cfg.HTTP.Timeout, middleware.WithTimeoutLogger(logger)),
		),
	)

	srv.Route("GET", "/health", healthHandler)
	logger.Info("HTTP server configured", "address", cfg.HTTP.Address)

	// Mount admin endpoints
	healthChecker := health.NewChecker()
	admin.Mount(srv.Router(), admin.WithHealthChecker(healthChecker), admin.WithConfigProvider(func() any { return cfg }))
	return srv
}

// healthHandler returns a simple health check response.
func healthHandler(_ context.Context, _ any) (any, error) {
	return map[string]string{"status": "ok"}, nil
}

// buildRegistryLifecycle creates a lifecycle component that registers the
// service with the configured registry on start and deregisters on stop.
// Returns nil when no registry is configured.
func buildRegistryLifecycle(cfg *conf.Bootstrap, logger *slog.Logger) (app.Lifecycle, error) {
	if cfg.Registry == nil || cfg.Registry.Address == "" {
		return nil, nil
	}

	var registrar registry.Registrar
	switch cfg.Registry.Type {
	case "consul":
		reg, err := consul.NewRegistrar(&consul.RegistrarConfig{
			Address: cfg.Registry.Address,
			Timeout: cfg.Registry.Timeout,
		})
		if err != nil {
			return nil, fmt.Errorf("create consul registrar: %w", err)
		}
		registrar = reg
	default:
		logger.Warn("unsupported registry type, skipping registration", "type", cfg.Registry.Type)
		return nil, nil
	}

	instance := registry.NewServiceInstance(
		registry.WithID(fmt.Sprintf("%s-%d", cfg.Name, os.Getpid())),
		registry.WithName(cfg.Name),
		registry.WithVersion(cfg.Version),
		registry.WithMetadata(cfg.Registry.Metadata),
	)
	if cfg.HTTP != nil && cfg.HTTP.Address != "" {
		instance.Endpoints = append(instance.Endpoints, "http://"+cfg.HTTP.Address)
	}
	if cfg.GRPC != nil && cfg.GRPC.Address != "" {
		instance.Endpoints = append(instance.Endpoints, "grpc://"+cfg.GRPC.Address)
	}

	return &registryLifecycle{
		registrar: registrar,
		instance:  instance,
		logger:    logger,
	}, nil
}

// registryLifecycle registers the service instance on Start and deregisters
// it on Stop. It implements app.Lifecycle: Start runs after all servers are
// up, Stop runs before servers shut down.
type registryLifecycle struct {
	registrar registry.Registrar
	instance  *registry.ServiceInstance
	logger    *slog.Logger
}

func (l *registryLifecycle) Start(ctx context.Context) error {
	if err := l.registrar.Register(ctx, l.instance); err != nil {
		return fmt.Errorf("register service: %w", err)
	}
	l.logger.Info("service registered",
		"name", l.instance.Name,
		"id", l.instance.ID,
		"endpoints", l.instance.Endpoints,
	)
	return nil
}

func (l *registryLifecycle) Stop(ctx context.Context) error {
	if err := l.registrar.Deregister(ctx, l.instance); err != nil {
		l.logger.Warn("failed to deregister service", "id", l.instance.ID, "error", err)
		return err
	}
	l.logger.Info("service deregistered", "name", l.instance.Name, "id", l.instance.ID)
	return nil
}
