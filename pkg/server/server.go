// Package server exposes an HTTP server that services outside this module can
// build on.
//
// A service that lives in its own repository cannot import
// internal/transport/http, which is where Firefly's HTTP transport lives, so
// the transport was previously reachable only from examples inside this
// module. This package is the supported entry point: it wraps the internal
// server, keeps its lifecycle (Start/Stop) so it can be handed to app.Server,
// and re-exports the handler and middleware types a service needs.
//
// Typical use:
//
//	srv := server.New(
//	    server.Address(":8080"),
//	    server.Timeout(15*time.Second),
//	    server.Middleware(middleware.Recovery(), middleware.Logging()),
//	)
//	srv.Route(http.MethodGet, "/healthz", func(ctx context.Context, _ any) (any, error) {
//	    return map[string]string{"status": "ok"}, nil
//	})
//	application := app.New(app.Name("my-service"), app.Server(srv))
//	code, err := application.Run()
//
// Handlers receive the transport request as the req argument. For HTTP that
// value is the original *http.Request, so a handler that needs the raw body
// or query string type-asserts it; handlers that only produce a value can
// ignore it.
package server

import (
	"context"
	"net/url"
	"time"

	internal "github.com/zhangpeihaoks/firefly/internal/transport/http"
	"github.com/zhangpeihaoks/firefly/pkg/middleware"
)

// Group is a set of routes that share a path prefix and middleware.
type Group struct {
	group *internal.RouterGroup
}

// Option configures a Server before it is created.
type Option func(*options)

// options collects the public configuration before it is mapped onto the
// internal server options. Fields are ordered for pointer alignment.
type options struct {
	middleware     []middleware.Middleware
	address        string
	timeout        time.Duration
	maxRequestBody int64
}

// Address sets the listening address, for example ":8080" or
// "127.0.0.1:8080". Defaults to ":8080".
func Address(addr string) Option {
	return func(o *options) {
		o.address = addr
	}
}

// Timeout sets the read, write and shutdown timeout. Defaults to 30s.
func Timeout(d time.Duration) Option {
	return func(o *options) {
		o.timeout = d
	}
}

// MaxRequestBodySize rejects requests whose Content-Length exceeds size
// bytes. Zero keeps the internal default.
func MaxRequestBodySize(size int64) Option {
	return func(o *options) {
		o.maxRequestBody = size
	}
}

// Middleware installs the global middleware chain. When called with nothing,
// the server keeps Firefly's default observability chain
// (RequestID/Recovery/Logging/Tracing/Metrics).
func Middleware(ms ...middleware.Middleware) Option {
	return func(o *options) {
		o.middleware = append(o.middleware, ms...)
	}
}

// Server is an HTTP server that implements the lifecycle the framework's
// application expects, so it can be passed to app.Server.
type Server struct {
	srv *internal.Server
}

// New creates a server from the given options. The returned server does not
// listen until Start is called.
func New(opts ...Option) *Server {
	o := &options{
		address: ":8080",
		timeout: 30 * time.Second,
	}
	for _, opt := range opts {
		opt(o)
	}

	internalOpts := []internal.ServerOption{
		internal.Address(o.address),
		internal.Timeout(o.timeout),
	}
	if len(o.middleware) > 0 {
		internalOpts = append(internalOpts, internal.Middleware(o.middleware...))
	}
	if o.maxRequestBody > 0 {
		internalOpts = append(internalOpts, internal.MaxRequestBodySize(o.maxRequestBody))
	}

	return &Server{srv: internal.NewServer(internalOpts...)}
}

// Route registers a handler for one method and path. Extra middleware runs
// only for this route, after the global chain.
func (s *Server) Route(method, path string, h middleware.Handler, ms ...middleware.Middleware) {
	s.srv.Route(method, path, h, ms...)
}

// Group creates a route group under prefix with its own middleware.
func (s *Server) Group(prefix string, ms ...middleware.Middleware) *Group {
	return &Group{group: s.srv.Group(prefix, ms...)}
}

// Route registers a handler inside the group.
func (g *Group) Route(method, path string, h middleware.Handler, ms ...middleware.Middleware) {
	g.group.Route(method, path, h, ms...)
}

// Group creates a nested group below this one.
func (g *Group) Group(prefix string, ms ...middleware.Middleware) *Group {
	return &Group{group: g.group.Group(prefix, ms...)}
}

// Start begins listening and serving. It returns once the listener is open;
// requests are served from a background goroutine.
func (s *Server) Start(ctx context.Context) error {
	return s.srv.Start(ctx)
}

// Stop stops accepting new connections and waits for in-flight requests
// until ctx is cancelled or the server timeout elapses.
func (s *Server) Stop(ctx context.Context) error {
	return s.srv.Stop(ctx)
}

// Endpoint returns the URL the server is listening on. The port is only
// meaningful after Start.
func (s *Server) Endpoint() (*url.URL, error) {
	return s.srv.Endpoint()
}
