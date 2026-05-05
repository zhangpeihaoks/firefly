// Package http provides HTTP server implementation for the Firefly framework.
// It implements the transport.Server interface using Gin as the HTTP router.
package http

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zhangpeihaoks/firefly/internal/log"
	"github.com/zhangpeihaoks/firefly/internal/middleware"
)

// Server is the HTTP server implementation.
// It implements transport.Server and transport.Endpointer interfaces.
type Server struct {
	*http.Server
	lis              net.Listener
	once             sync.Once
	endpoint         *url.URL
	err              error
	network          string
	address          string
	timeout          time.Duration
	maxRequestSize   int64
	ms               []middleware.Middleware
	router           *gin.Engine
	log              *slog.Logger
	tlsConf          *tls.Config
	requestSizeLimit bool
}

// ServerOption is a function that configures the HTTP server.
type ServerOption func(*Server)

// NewServer creates a new HTTP server with the given options.
func NewServer(opts ...ServerOption) *Server {
	// Create Gin router with default settings
	router := gin.New()

	// Create server with default values
	s := &Server{
		network:        "tcp",
		address:        ":8080",
		timeout:        30 * time.Second,
		maxRequestSize: 10 * 1024 * 1024, // Default 10MB
		router:         router,
		log:            log.L(),
	}

	// Apply options
	for _, opt := range opts {
		opt(s)
	}

	// Create HTTP server
	s.Server = &http.Server{
		Addr:           s.address,
		Handler:        s.router,
		ReadTimeout:    s.timeout,
		WriteTimeout:   s.timeout,
		IdleTimeout:    60 * time.Second,
		TLSConfig:      s.tlsConf,
		MaxHeaderBytes: int(s.maxRequestSize),
	}

	// Add request size limit middleware if configured
	if s.requestSizeLimit && s.maxRequestSize > 0 {
		s.router.Use(func(c *gin.Context) {
			if c.Request.ContentLength > s.maxRequestSize {
				c.AbortWithStatusJSON(413, gin.H{
					"code":    413,
					"message": "Request body too large",
				})
				return
			}
			c.Next()
		})
	}

	return s
}

// Start starts the HTTP server.
// It implements transport.Server interface.
func (s *Server) Start(ctx context.Context) error {
	s.once.Do(func() {
		// Create listener
		lis, err := net.Listen(s.network, s.address)
		if err != nil {
			s.err = err
			return
		}
		s.lis = lis

		// Build endpoint URL
		s.endpoint = &url.URL{
			Scheme: "http",
			Host:   lis.Addr().String(),
		}

		// Log server start
		s.log.Info("HTTP server starting",
			"address", s.address,
			"network", s.network,
		)

		// Start server in goroutine
		go func() {
			var err error
			if s.tlsConf != nil {
				err = s.ServeTLS(s.lis, "", "")
			} else {
				err = s.Serve(s.lis)
			}
			if err != nil && err != http.ErrServerClosed {
				s.log.Error("HTTP server error", "error", err)
			}
		}()
	})

	return s.err
}

// Stop stops the HTTP server gracefully.
// It implements transport.Server interface.
func (s *Server) Stop(ctx context.Context) error {
	s.log.Info("HTTP server stopping")

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	// Shutdown server
	if err := s.Shutdown(shutdownCtx); err != nil {
		s.log.Error("HTTP server shutdown error", "error", err)
		return err
	}

	s.log.Info("HTTP server stopped")
	return nil
}

// Endpoint returns the server endpoint URL.
// It implements transport.Endpointer interface.
func (s *Server) Endpoint() (*url.URL, error) {
	if s.endpoint == nil {
		return nil, fmt.Errorf("server not started")
	}
	return s.endpoint, nil
}

// Route registers a route with the given method, path, and handler.
func (s *Server) Route(method, path string, handler middleware.Handler, ms ...middleware.Middleware) {
	// Convert middleware.Handler to Gin handler
	ginHandler := s.createGinHandler(handler, ms...)

	// Register route with Gin
	switch method {
	case http.MethodGet:
		s.router.GET(path, ginHandler)
	case http.MethodPost:
		s.router.POST(path, ginHandler)
	case http.MethodPut:
		s.router.PUT(path, ginHandler)
	case http.MethodDelete:
		s.router.DELETE(path, ginHandler)
	case http.MethodPatch:
		s.router.PATCH(path, ginHandler)
	case http.MethodHead:
		s.router.HEAD(path, ginHandler)
	case http.MethodOptions:
		s.router.OPTIONS(path, ginHandler)
	default:
		s.router.Handle(method, path, ginHandler)
	}
}

// Static serves static files from the given directory.
func (s *Server) Static(relativePath, root string) {
	s.router.Static(relativePath, root)
}

// StaticFS serves static files from the given http.FileSystem.
func (s *Server) StaticFS(relativePath string, fs http.FileSystem) {
	s.router.StaticFS(relativePath, fs)
}

// StaticFile serves a single static file.
func (s *Server) StaticFile(relativePath, filepath string) {
	s.router.StaticFile(relativePath, filepath)
}

// StaticFileFS serves a single static file from the given http.FileSystem.
func (s *Server) StaticFileFS(relativePath, filepath string, fs http.FileSystem) {
	s.router.StaticFileFS(relativePath, filepath, fs)
}

// Group creates a new router group with the given prefix and middleware.
func (s *Server) Group(prefix string, ms ...middleware.Middleware) *RouterGroup {
	ginGroup := s.router.Group(prefix)
	return &RouterGroup{
		ginGroup: ginGroup,
		server:   s,
		ms:       ms,
	}
}

// Use adds global middleware to the server.
func (s *Server) Use(ms ...middleware.Middleware) {
	s.ms = append(s.ms, ms...)
}

// createGinHandler converts a middleware.Handler to a Gin handler function.
func (s *Server) createGinHandler(handler middleware.Handler, ms ...middleware.Middleware) gin.HandlerFunc {
	// Create middleware chain
	chain := middleware.Chain(append(s.ms, ms...)...)

	// Wrap handler with middleware chain
	wrappedHandler := chain(handler)

	return func(c *gin.Context) {
		// Create HTTP transporter
		endpoint := ""
		if s.endpoint != nil {
			endpoint = s.endpoint.String()
		}
		transporter := newTransporter(endpoint, c.FullPath())

		// Copy request headers to transporter
		for k, v := range c.Request.Header {
			if len(v) > 0 {
				transporter.RequestHeader().Set(k, v[0])
			}
		}

		// Extract path parameters from Gin context
		for _, param := range c.Params {
			transporter.pathParams[param.Key] = param.Value
		}

		// Extract query parameters from Gin context
		queryParams := c.Request.URL.Query()
		for key, values := range queryParams {
			transporter.queryParams[key] = values
		}

		// Create context with transporter
		ctx := NewContext(c.Request.Context(), transporter)

		// Call wrapped handler
		resp, err := wrappedHandler(ctx, c.Request)
		if err != nil {
			// TODO: Handle error response properly
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		// Set response headers from transporter
		for _, k := range transporter.ReplyHeader().Keys() {
			c.Header(k, transporter.ReplyHeader().Get(k))
		}

		// Handle response based on type
		c.JSON(200, resp)
	}
}

// RouterGroup represents a group of routes with common prefix and middleware.
type RouterGroup struct {
	ginGroup *gin.RouterGroup
	server   *Server
	ms       []middleware.Middleware
}

// Route registers a route in the group.
func (g *RouterGroup) Route(method, path string, handler middleware.Handler, ms ...middleware.Middleware) {
	ginHandler := g.server.createGinHandler(handler, append(g.ms, ms...)...)

	switch method {
	case http.MethodGet:
		g.ginGroup.GET(path, ginHandler)
	case http.MethodPost:
		g.ginGroup.POST(path, ginHandler)
	case http.MethodPut:
		g.ginGroup.PUT(path, ginHandler)
	case http.MethodDelete:
		g.ginGroup.DELETE(path, ginHandler)
	case http.MethodPatch:
		g.ginGroup.PATCH(path, ginHandler)
	case http.MethodHead:
		g.ginGroup.HEAD(path, ginHandler)
	case http.MethodOptions:
		g.ginGroup.OPTIONS(path, ginHandler)
	default:
		g.ginGroup.Handle(method, path, ginHandler)
	}
}

// Group creates a subgroup with additional prefix and middleware.
func (g *RouterGroup) Group(prefix string, ms ...middleware.Middleware) *RouterGroup {
	ginGroup := g.ginGroup.Group(prefix)
	return &RouterGroup{
		ginGroup: ginGroup,
		server:   g.server,
		ms:       append(g.ms, ms...),
	}
}

// Use adds middleware to the group.
func (g *RouterGroup) Use(ms ...middleware.Middleware) {
	g.ms = append(g.ms, ms...)
}

// RegisterMetrics registers the Prometheus metrics endpoint.
// The endpoint path is configurable via the MetricsConfig.Path field.
func (s *Server) RegisterMetrics(path string, handler http.Handler) {
	s.router.GET(path, func(c *gin.Context) {
		handler.ServeHTTP(c.Writer, c.Request)
	})
}
