// Package http provides HTTP server implementation for the Firefly framework.
package http

import (
	"crypto/tls"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zhangpeihaoks/firefly/internal/middleware"
)

// Network sets the network type for the server (e.g., "tcp", "tcp4", "tcp6").
func Network(network string) ServerOption {
	return func(s *Server) {
		s.network = network
	}
}

// Address sets the listening address for the server (e.g., ":8080", "127.0.0.1:8080").
func Address(addr string) ServerOption {
	return func(s *Server) {
		s.address = addr
	}
}

// Timeout sets the read and write timeout for the server.
func Timeout(timeout time.Duration) ServerOption {
	return func(s *Server) {
		s.timeout = timeout
	}
}

// ReadTimeout sets the read timeout for the server.
func ReadTimeout(timeout time.Duration) ServerOption {
	return func(s *Server) {
		s.ReadTimeout = timeout
	}
}

// WriteTimeout sets the write timeout for the server.
func WriteTimeout(timeout time.Duration) ServerOption {
	return func(s *Server) {
		s.WriteTimeout = timeout
	}
}

// IdleTimeout sets the idle timeout for the server.
func IdleTimeout(timeout time.Duration) ServerOption {
	return func(s *Server) {
		s.IdleTimeout = timeout
	}
}

// Middleware sets the global middleware for the server.
func Middleware(m ...middleware.Middleware) ServerOption {
	return func(s *Server) {
		s.ms = append(s.ms, m...)
	}
}

// Logger sets the logger for the server.
func Logger(logger *slog.Logger) ServerOption {
	return func(s *Server) {
		s.log = logger
	}
}

// TLSConfig sets the TLS configuration for the server.
func TLSConfig(cfg *tls.Config) ServerOption {
	return func(s *Server) {
		s.tlsConf = cfg
	}
}

// TLSConfigFromFile loads TLS configuration from files and sets it on the server.
// It reads certificate, key, and optional CA files from the filesystem.
func TLSConfigFromFile(certFile, keyFile string, caFile ...string) ServerOption {
	return func(s *Server) {
		var tlsCfg *TLSCfg
		if len(caFile) > 0 && caFile[0] != "" {
			tlsCfg = NewTLSCfg(
				WithCertFile(certFile),
				WithKeyFile(keyFile),
				WithCAFile(caFile[0]),
			)
		} else {
			tlsCfg = NewTLSCfg(
				WithCertFile(certFile),
				WithKeyFile(keyFile),
			)
		}

		cfg, err := LoadTLSCfg(tlsCfg)
		if err != nil {
			// Log the error but don't fail - TLS will be disabled
			s.log.Error("failed to load TLS config", "error", err)
			return
		}
		s.tlsConf = cfg
	}
}

// MaxRequestSize sets the maximum request size in bytes.
// This limit helps prevent denial-of-service attacks by rejecting oversized requests.
// Default is 10MB (10 * 1024 * 1024 bytes).
func MaxRequestSize(size int64) ServerOption {
	return func(s *Server) {
		s.MaxHeaderBytes = int(size)
	}
}

// MaxRequestBodySize enables and sets the maximum request body size in bytes.
// This is a middleware-level check that validates the Content-Length header
// before the full body is read. Default is 10MB (10 * 1024 * 1024 bytes).
// Set size to 0 to disable the limit.
func MaxRequestBodySize(size int64) ServerOption {
	return func(s *Server) {
		s.maxRequestSize = size
		if size > 0 {
			s.requestSizeLimit = true
		}
	}
}

// RequestTimeout sets the request timeout for the server.
// This includes the time to read the request header and body.
// Default is 30 seconds.
// Note: This only sets the internal timeout field which is applied when creating the http.Server.
// Use ReadTimeout/WriteTimeout options to set timeouts on an already-created server.
func RequestTimeout(timeout time.Duration) ServerOption {
	return func(s *Server) {
		s.timeout = timeout
	}
}

// GinMode sets the Gin mode (debug, release, test).
func GinMode(mode string) ServerOption {
	return func(s *Server) {
		gin.SetMode(mode)
	}
}
