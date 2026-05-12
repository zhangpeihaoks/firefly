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
