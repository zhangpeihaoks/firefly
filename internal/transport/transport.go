// Package transport provides transport layer abstractions for the Firefly framework.
// It defines core interfaces for servers, endpoints, and transport contexts.
package transport

import (
	"context"
	"net/url"
)

// Kind represents the transport type (HTTP, gRPC, etc.)
type Kind string

const (
	// KindHTTP represents HTTP transport
	KindHTTP Kind = "http"
	// KindGRPC represents gRPC transport
	KindGRPC Kind = "grpc"
)

// Server is the transport layer server interface.
// It defines the contract for starting and stopping servers.
type Server interface {
	// Start starts the server with the given context.
	Start(ctx context.Context) error
	// Stop stops the server with the given context.
	Stop(ctx context.Context) error
}

// Endpointer is the interface for registering service endpoints.
// It returns the service endpoint URL for service discovery and registration.
type Endpointer interface {
	// Endpoint returns the service endpoint URL.
	Endpoint() (*url.URL, error)
}

// Transporter is the transport context interface.
// It provides access to transport-layer metadata during request processing.
type Transporter interface {
	// Kind returns the transport type (http/grpc).
	Kind() Kind
	// Endpoint returns the service endpoint as a string.
	Endpoint() string
	// Operation returns the operation name (e.g., HTTP path or gRPC method).
	Operation() string
	// RequestHeader returns the request header interface.
	RequestHeader() Header
	// ReplyHeader returns the response header interface.
	ReplyHeader() Header
	// PathParams returns the path parameters from the request.
	PathParams() map[string]string
	// QueryParams returns the query parameters from the request.
	QueryParams() map[string][]string
}

// Header is the interface for header operations.
// It provides a unified way to access and modify headers across different transports.
type Header interface {
	// Get returns the value for the given key.
	Get(key string) string
	// Set sets the value for the given key.
	Set(key string, value string)
	// Keys returns all header keys.
	Keys() []string
}
