// Package errors exposes Firefly's unified error type so services outside
// this module can return HTTP/gRPC aware errors.
//
// The HTTP transport maps *errors.Error to its Code field, so returning
// errors.New(http.StatusServiceUnavailable, "NOT_READY", "config missing")
// from a handler produces a 503 response with that reason and message, while
// any other error is masked as 500.
package errors

import (
	ferrors "github.com/zhangpeihaoks/firefly/internal/errors"
)

// Error is Firefly's unified error. It carries the status code, a stable
// reason identifier, a human-readable message, optional metadata, and an
// optional wrapped cause that errors.Is / errors.As can traverse.
type Error = ferrors.Error

// Common HTTP status codes used by services.
const (
	// CodeBadRequest is HTTP 400.
	CodeBadRequest = ferrors.CodeBadRequest
	// CodeUnauthorized is HTTP 401.
	CodeUnauthorized = ferrors.CodeUnauthorized
	// CodeForbidden is HTTP 403.
	CodeForbidden = ferrors.CodeForbidden
	// CodeNotFound is HTTP 404.
	CodeNotFound = ferrors.CodeNotFound
	// CodeConflict is HTTP 409.
	CodeConflict = ferrors.CodeConflict
	// CodeInternal is HTTP 500.
	CodeInternal = ferrors.CodeInternal
	// CodeServiceUnavailable is HTTP 503.
	CodeServiceUnavailable = ferrors.CodeServiceUnavailable
	// CodeGatewayTimeout is HTTP 504.
	CodeGatewayTimeout = ferrors.CodeGatewayTimeout
)

// New builds an error with the given status code, reason and message.
func New(code int, reason, message string) *Error {
	return ferrors.New(code, reason, message)
}

// Newf builds an error whose message is formatted with fmt.Sprintf.
func Newf(code int, reason, format string, a ...any) *Error {
	return ferrors.Newf(code, reason, format, a...)
}

// Wrap attaches a status code, reason and message to an existing error while
// keeping the original reachable through errors.Unwrap.
func Wrap(err error, code int, reason, message string) *Error {
	return ferrors.Wrap(err, code, reason, message)
}
