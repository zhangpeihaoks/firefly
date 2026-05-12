// Package app provides application lifecycle management for the Firefly framework.
package app

import (
	"context"
	"time"

	"github.com/zhangpeihaoks/firefly/internal/middleware"
	httptransport "github.com/zhangpeihaoks/firefly/internal/transport/http"
)

// QuickStart creates a pre-configured App with sensible defaults:
//   - HTTP server on :8080 with 30s timeout
//   - Recovery, RequestID, and Logging middleware
//   - GET /health endpoint returning {"status": "ok"}
//
// This is ideal for rapid prototyping, demos, and services that don't
// need custom configuration. The returned App can be further customized
// via app.Option before calling Run().
//
// Example:
//
//	application, err := app.QuickStart()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	application.Run()
func QuickStart() (*App, error) {
	srv := httptransport.NewServer(
		httptransport.Address(":8080"),
		httptransport.Timeout(30*time.Second),
		httptransport.Middleware(
			middleware.Recovery(),
			middleware.RequestID(),
			middleware.Logging(),
		),
	)

	srv.Route("GET", "/health", func(ctx context.Context, req any) (any, error) {
		return map[string]string{"status": "ok"}, nil
	})

	application := New(
		Name("firefly"),
		Server(srv),
	)

	return application, nil
}
