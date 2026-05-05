// Package main provides a complete example of starting a Firefly application
// with multiple servers (HTTP and gRPC).
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/zhangpeihaoks/firefly/app"
	"github.com/zhangpeihaoks/firefly/internal/log"
	"github.com/zhangpeihaoks/firefly/internal/middleware"
	httpserver "github.com/zhangpeihaoks/firefly/internal/transport/http"
)

func main() {
	// Example 1: Basic Application Startup
	exampleBasicStartup()

	// Example 2: Application with Configuration
	// exampleWithConfig()

	// Example 3: Application with Graceful Shutdown
	// exampleGracefulShutdown()
}

// exampleBasicStartup demonstrates the simplest way to start a Firefly application.
func exampleBasicStartup() {
	fmt.Println("=== Example 1: Basic Startup ===")

	// Initialize logger with file output
	cleanup := log.New(&log.Config{
		FileName:   "app.log",
		MaxSize:    100,    // 100MB max file size
		MaxBackups: 5,      // Keep 5 backup files
		MaxAge:     7,      // Keep files for 7 days
		Level:      "info", // Log level: debug, info, warn, error
		JSONFormat: true,   // Use JSON format for structured logging
		Location:   true,   // Include file location in logs
	})
	defer cleanup()

	// Create HTTP server
	httpServer := httpserver.NewServer(
		httpserver.Address(":8080"),
		httpserver.Timeout(30*time.Second),
		httpserver.Middleware(
			middleware.Recovery(),
			middleware.Logging(),
		),
	)

	// Register routes
	httpServer.Route("GET", "/health", func(ctx context.Context, req any) (any, error) {
		return map[string]string{
			"status": "healthy",
			"time":   time.Now().Format(time.RFC3339),
		}, nil
	})

	// Create application with single server
	application := app.New(
		app.Name("my-service"),
		app.Logger(log.L()),
		app.Server(httpServer),
		app.StopTimeout(10*time.Second), // Graceful shutdown timeout
	)

	// Run application (blocking)
	fmt.Println("Starting service on :8080...")
	exitCode, err := application.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Application error: %v\n", err)
		os.Exit(exitCode)
	}
}
