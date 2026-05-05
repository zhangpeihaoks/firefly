// Package middleware provides middleware abstractions for the Firefly framework.
package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/zhangpeihaoks/firefly/internal/transport"
)

// CORSConfig holds the configuration for CORS middleware.
type CORSConfig struct {
	// AllowOrigins is the list of allowed origins.
	// Use "*" to allow all origins.
	// Origins can contain wildcards, e.g., "*.example.com"
	AllowOrigins []string

	// AllowMethods is the list of allowed HTTP methods.
	// Default: GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS
	AllowMethods []string

	// AllowHeaders is the list of allowed request headers.
	// Use "*" to allow all headers.
	AllowHeaders []string

	// ExposeHeaders is the list of headers that can be exposed to the client.
	ExposeHeaders []string

	// AllowCredentials indicates whether credentials can be included in requests.
	// When true, AllowOrigins cannot contain "*".
	AllowCredentials bool

	// MaxAge indicates how long (in seconds) the results of a preflight request
	// can be cached.
	MaxAge int

	// OptionsPassthrough indicates whether to pass OPTIONS requests to the next handler.
	// When false (default), OPTIONS requests are handled directly by the middleware.
	OptionsPassthrough bool
}

// corsOptions holds the configuration for CORS middleware.
type corsOptions struct {
	logger *slog.Logger
	config CORSConfig
}

// CORSOption is a configuration option for CORS middleware.
type CORSOption func(*corsOptions)

// WithCORSLogger sets a custom logger for the CORS middleware.
// If not set, the default slog logger is used.
func WithCORSLogger(logger *slog.Logger) CORSOption {
	return func(o *corsOptions) {
		o.logger = logger
	}
}

// WithCORSConfig sets the CORS configuration.
func WithCORSConfig(config CORSConfig) CORSOption {
	return func(o *corsOptions) {
		o.config = config
	}
}

// WithAllowOrigins sets the allowed origins.
// Use "*" to allow all origins, or provide a list of specific origins.
// Origins can contain wildcards, e.g., "*.example.com"
func WithAllowOrigins(origins ...string) CORSOption {
	return func(o *corsOptions) {
		o.config.AllowOrigins = origins
	}
}

// WithAllowMethods sets the allowed HTTP methods.
func WithAllowMethods(methods ...string) CORSOption {
	return func(o *corsOptions) {
		o.config.AllowMethods = methods
	}
}

// WithAllowHeaders sets the allowed request headers.
func WithAllowHeaders(headers ...string) CORSOption {
	return func(o *corsOptions) {
		o.config.AllowHeaders = headers
	}
}

// WithExposeHeaders sets the headers that can be exposed to the client.
func WithExposeHeaders(headers ...string) CORSOption {
	return func(o *corsOptions) {
		o.config.ExposeHeaders = headers
	}
}

// WithAllowCredentials sets whether credentials can be included in requests.
// When true, AllowOrigins cannot contain "*".
func WithAllowCredentials(allow bool) CORSOption {
	return func(o *corsOptions) {
		o.config.AllowCredentials = allow
	}
}

// WithMaxAge sets how long (in seconds) the results of a preflight request can be cached.
func WithMaxAge(seconds int) CORSOption {
	return func(o *corsOptions) {
		o.config.MaxAge = seconds
	}
}

// WithOptionsPassthrough sets whether to pass OPTIONS requests to the next handler.
func WithOptionsPassthrough(passthrough bool) CORSOption {
	return func(o *corsOptions) {
		o.config.OptionsPassthrough = passthrough
	}
}

// DefaultCORSConfig returns the default CORS configuration.
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodDelete,
			http.MethodPatch,
			http.MethodHead,
			http.MethodOptions,
		},
		AllowHeaders:     []string{"Content-Type", "Authorization", "Accept"},
		ExposeHeaders:    []string{},
		AllowCredentials: false,
		MaxAge:           86400, // 24 hours
	}
}

// CORS returns a middleware that handles Cross-Origin Resource Sharing (CORS).
// It implements the CORS specification as defined in:
// https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS
//
// The middleware:
//  1. Handles preflight OPTIONS requests by returning appropriate CORS headers
//  2. Sets CORS headers on responses for actual requests
//  3. Validates origin against the allowed origins list
//  4. Supports wildcard origins and pattern matching
//
// Example (allow all origins):
//
//	middleware.CORS()
//
// Example (specific origins):
//
//	middleware.CORS(
//	    middleware.WithAllowOrigins("https://example.com", "https://api.example.com"),
//	)
//
// Example (with credentials):
//
//	middleware.CORS(
//	    middleware.WithAllowOrigins("https://example.com"),
//	    middleware.WithAllowCredentials(true),
//	)
//
// Example (full configuration):
//
//	middleware.CORS(
//	    middleware.WithCORSConfig(middleware.CORSConfig{
//	        AllowOrigins:     []string{"https://example.com"},
//	        AllowMethods:     []string{"GET", "POST"},
//	        AllowHeaders:     []string{"Content-Type", "Authorization"},
//	        ExposeHeaders:    []string{"X-Custom-Header"},
//	        AllowCredentials: true,
//	        MaxAge:           3600,
//	    }),
//	)
func CORS(opts ...CORSOption) Middleware {
	// Apply default options
	options := &corsOptions{
		logger: slog.Default(),
		config: DefaultCORSConfig(),
	}

	// Apply custom options
	for _, opt := range opts {
		opt(options)
	}

	// Validate configuration
	if options.config.AllowCredentials && len(options.config.AllowOrigins) == 1 && options.config.AllowOrigins[0] == "*" {
		options.logger.Warn("cors middleware: AllowCredentials=true with AllowOrigins=* is not recommended. Some browsers may reject such requests.")
	}

	return func(next Handler) Handler {
		return func(ctx context.Context, req any) (any, error) {
			// Get transport info from context
			tr := transport.FromContext(ctx)

			// Get request origin
			var origin string
			if tr != nil && tr.RequestHeader() != nil {
				origin = tr.RequestHeader().Get("Origin")
			}

			// For requests without Origin header (same-origin requests),
			// no CORS headers are needed
			if origin == "" {
				return next(ctx, req)
			}

			// Check if origin is allowed
			allowedOrigin := options.getAllowedOrigin(origin)
			if allowedOrigin == "" {
				// Origin not allowed - proceed without CORS headers
				// The browser will block the response
				options.logger.Debug("cors: origin not allowed",
					"origin", origin,
					"operation", func() string {
						if tr != nil {
							return tr.Operation()
						}
						return ""
					}(),
				)
				return next(ctx, req)
			}

			// Check if this is a preflight request
			if tr != nil && options.isPreflight(tr) {
				return options.handlePreflight(ctx, tr, allowedOrigin)
			}

			// Set CORS headers on the response
			if tr != nil && tr.ReplyHeader() != nil {
				options.setCORSHeaders(tr.ReplyHeader(), allowedOrigin)
			}

			// Call the next handler
			return next(ctx, req)
		}
	}
}

// getAllowedOrigin checks if the origin is allowed and returns the allowed origin value.
// Returns empty string if the origin is not allowed.
func (o *corsOptions) getAllowedOrigin(origin string) string {
	for _, allowed := range o.config.AllowOrigins {
		if allowed == "*" {
			// Allow all origins
			if o.config.AllowCredentials {
				// When credentials are allowed, we must return the actual origin
				// instead of "*"
				return origin
			}
			return "*"
		}

		// Check for exact match
		if allowed == origin {
			return origin
		}

		// Check for wildcard match (e.g., "*.example.com")
		if strings.HasPrefix(allowed, "*.") {
			// Extract domain from wildcard pattern
			domain := allowed[2:] // Remove "*."
			if strings.HasSuffix(origin, domain) {
				// Check that it's either the exact domain or a subdomain
				originHost := extractHost(origin)
				if originHost == domain[1:] || strings.HasSuffix(originHost, domain) {
					return origin
				}
			}
		}
	}

	return ""
}

// isPreflight checks if the request is a CORS preflight request.
func (o *corsOptions) isPreflight(tr transport.Transporter) bool {
	// Preflight requests are always OPTIONS
	if tr.Kind() != transport.KindHTTP {
		return false
	}

	// Check for Origin header (already verified in caller)
	// Check for Access-Control-Request-Method header
	requestMethod := tr.RequestHeader().Get("Access-Control-Request-Method")
	return requestMethod != ""
}

// handlePreflight handles OPTIONS preflight requests.
func (o *corsOptions) handlePreflight(ctx context.Context, tr transport.Transporter, allowedOrigin string) (any, error) {
	// Get requested method and headers
	requestMethod := tr.RequestHeader().Get("Access-Control-Request-Method")
	requestHeaders := tr.RequestHeader().Get("Access-Control-Request-Headers")

	// Validate requested method
	if !o.isMethodAllowed(requestMethod) {
		o.logger.Debug("cors: method not allowed", "method", requestMethod)
		// Return 200 but don't set Access-Control-Allow-Methods
		// Browser will block the actual request
	}

	// Validate requested headers
	if requestHeaders != "" && !o.areHeadersAllowed(requestHeaders) {
		o.logger.Debug("cors: headers not allowed", "headers", requestHeaders)
	}

	// Set CORS preflight response headers
	header := tr.ReplyHeader()
	header.Set("Access-Control-Allow-Origin", allowedOrigin)

	// Set allowed methods
	if len(o.config.AllowMethods) > 0 {
		header.Set("Access-Control-Allow-Methods", strings.Join(o.config.AllowMethods, ", "))
	}

	// Set allowed headers
	if len(o.config.AllowHeaders) > 0 {
		// If AllowHeaders contains "*", reflect the requested headers
		if len(o.config.AllowHeaders) == 1 && o.config.AllowHeaders[0] == "*" {
			header.Set("Access-Control-Allow-Headers", requestHeaders)
		} else {
			header.Set("Access-Control-Allow-Headers", strings.Join(o.config.AllowHeaders, ", "))
		}
	}

	// Set credentials
	if o.config.AllowCredentials {
		header.Set("Access-Control-Allow-Credentials", "true")
	}

	// Set max age
	if o.config.MaxAge > 0 {
		header.Set("Access-Control-Max-Age", strconv.Itoa(o.config.MaxAge))
	}

	// For preflight requests, return empty response with 200 OK
	// If OptionsPassthrough is true, call the next handler
	if o.config.OptionsPassthrough {
		return nil, nil // Let the next handler process
	}

	// Return nil response and nil error for successful preflight
	return nil, nil
}

// setCORSHeaders sets CORS headers on the response for actual requests.
func (o *corsOptions) setCORSHeaders(header transport.Header, allowedOrigin string) {
	header.Set("Access-Control-Allow-Origin", allowedOrigin)

	// Set credentials
	if o.config.AllowCredentials {
		header.Set("Access-Control-Allow-Credentials", "true")
	}

	// Set exposed headers
	if len(o.config.ExposeHeaders) > 0 {
		header.Set("Access-Control-Expose-Headers", strings.Join(o.config.ExposeHeaders, ", "))
	}
}

// isMethodAllowed checks if the request method is allowed.
func (o *corsOptions) isMethodAllowed(method string) bool {
	if method == "" {
		return false
	}

	for _, allowed := range o.config.AllowMethods {
		if strings.EqualFold(allowed, method) {
			return true
		}
	}

	return false
}

// areHeadersAllowed checks if all requested headers are allowed.
func (o *corsOptions) areHeadersAllowed(headers string) bool {
	if headers == "" {
		return true
	}

	// If AllowHeaders contains "*", all headers are allowed
	for _, allowed := range o.config.AllowHeaders {
		if allowed == "*" {
			return true
		}
	}

	// Check each requested header
	requestedHeaders := strings.Split(headers, ",")
	for _, h := range requestedHeaders {
		h = strings.TrimSpace(h)
		if h == "" {
			continue
		}

		found := false
		for _, allowed := range o.config.AllowHeaders {
			if strings.EqualFold(allowed, h) {
				found = true
				break
			}
		}

		if !found {
			return false
		}
	}

	return true
}

// extractHost extracts the host (origin without scheme) from a URL string.
// For "https://example.com:8080", it returns "example.com:8080".
func extractHost(origin string) string {
	// Remove scheme
	origin = strings.TrimPrefix(origin, "https://")
	origin = strings.TrimPrefix(origin, "http://")

	// Remove path if any
	if idx := strings.Index(origin, "/"); idx != -1 {
		origin = origin[:idx]
	}

	return origin
}
