// Package middleware provides middleware abstractions for the Firefly framework.
// This file defines the default observability middleware chain used when a
// server is created without explicit middleware.
package middleware

// DefaultObservabilityChain is the zero-configuration production middleware
// chain: correlation ID, panic recovery, request logging, distributed tracing
// and Prometheus metrics. It is installed by default on HTTP and gRPC servers
// unless the caller supplies explicit middleware.
//
//	RequestID → Recovery → Logging → Tracing → Metrics
func DefaultObservabilityChain() []Middleware {
	return []Middleware{
		RequestID(),
		Recovery(),
		Logging(),
		Tracing(),
		Metrics(),
	}
}
