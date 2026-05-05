// Package http provides HTTP server implementation for the Firefly framework.
package http

import (
	"github.com/zhangpeihaoks/firefly/internal/transport"
)

// transporter is the HTTP implementation of transport.Transporter.
type transporter struct {
	kind          transport.Kind
	endpoint      string
	operation     string
	requestHeader transport.Header
	replyHeader   transport.Header
	pathParams    map[string]string
	queryParams   map[string][]string
}

// Kind returns the transport type (http).
func (t *transporter) Kind() transport.Kind {
	return t.kind
}

// Endpoint returns the service endpoint as a string.
func (t *transporter) Endpoint() string {
	return t.endpoint
}

// Operation returns the operation name (HTTP path).
func (t *transporter) Operation() string {
	return t.operation
}

// RequestHeader returns the request header interface.
func (t *transporter) RequestHeader() transport.Header {
	return t.requestHeader
}

// ReplyHeader returns the response header interface.
func (t *transporter) ReplyHeader() transport.Header {
	return t.replyHeader
}

// PathParams returns the path parameters from the request.
func (t *transporter) PathParams() map[string]string {
	return t.pathParams
}

// QueryParams returns the query parameters from the request.
func (t *transporter) QueryParams() map[string][]string {
	return t.queryParams
}

// header is the HTTP implementation of transport.Header.
type header struct {
	headers map[string]string
}

// Get returns the value for the given key.
func (h *header) Get(key string) string {
	return h.headers[key]
}

// Set sets the value for the given key.
func (h *header) Set(key string, value string) {
	h.headers[key] = value
}

// Keys returns all header keys.
func (h *header) Keys() []string {
	keys := make([]string, 0, len(h.headers))
	for k := range h.headers {
		keys = append(keys, k)
	}
	return keys
}

// newHeader creates a new header instance.
func newHeader() *header {
	return &header{
		headers: make(map[string]string),
	}
}

// newTransporter creates a new HTTP transporter.
func newTransporter(endpoint, operation string) *transporter {
	return &transporter{
		kind:          transport.KindHTTP,
		endpoint:      endpoint,
		operation:     operation,
		requestHeader: newHeader(),
		replyHeader:   newHeader(),
		pathParams:    make(map[string]string),
		queryParams:   make(map[string][]string),
	}
}
