// Package errors provides unified error handling for the Firefly framework.
// It defines a standard Error structure with code, reason, message, and metadata.
package errors

import (
	"fmt"

	"google.golang.org/grpc/status"
)

// Error is the unified error structure for the framework.
type Error struct {
	// Code is the error code (typically HTTP status code)
	Code int32 `json:"code"`
	// Reason is the error reason identifier
	Reason string `json:"reason"`
	// Message is the human-readable error message
	Message string `json:"message"`
	// Metadata contains additional error context
	Metadata map[string]string `json:"metadata,omitempty"`
}

// New creates a new Error with the given code, reason, and message.
func New(code int, reason, message string) *Error {
	return &Error{
		Code:    int32(code),
		Reason:  reason,
		Message: message,
	}
}

// Newf creates a new Error with formatted message.
func Newf(code int, reason, format string, a ...any) *Error {
	return &Error{
		Code:    int32(code),
		Reason:  reason,
		Message: fmt.Sprintf(format, a...),
	}
}

// Error implements the error interface.
func (e *Error) Error() string {
	return fmt.Sprintf("error: code = %d reason = %s message = %s", e.Code, e.Reason, e.Message)
}

// GRPCStatus returns the gRPC status for this error.
func (e *Error) GRPCStatus() *status.Status {
	// Convert HTTP code to gRPC code
	grpcCode := HTTPToGRPCCode(int(e.Code))
	return status.New(grpcCode, e.Message)
}

// WithMetadata adds metadata to the error.
func (e *Error) WithMetadata(md map[string]string) *Error {
	if e.Metadata == nil {
		e.Metadata = make(map[string]string)
	}
	for k, v := range md {
		e.Metadata[k] = v
	}
	return e
}

// Is implements the errors.Is interface for error comparison.
func (e *Error) Is(err error) bool {
	if target, ok := err.(*Error); ok {
		return e.Code == target.Code && e.Reason == target.Reason
	}
	return false
}

// FromError extracts an Error from the given error.
// Returns nil if the error is not an Error.
func FromError(err error) *Error {
	if err == nil {
		return nil
	}
	if e, ok := err.(*Error); ok {
		return e
	}
	// Try to extract from gRPC status
	if s, ok := status.FromError(err); ok {
		return &Error{
			Code:    int32(GRPCToHTTPCode(s.Code())),
			Reason:  s.Code().String(),
			Message: s.Message(),
		}
	}
	// Return a generic internal error
	return &Error{
		Code:    CodeInternal,
		Reason:  "INTERNAL_ERROR",
		Message: err.Error(),
	}
}

// Code returns the error code from the given error.
// Returns CodeInternal if the error is not an Error.
func Code(err error) int {
	if err == nil {
		return CodeOK
	}
	if e, ok := err.(*Error); ok {
		return int(e.Code)
	}
	return CodeInternal
}

// Reason returns the error reason from the given error.
// Returns "INTERNAL_ERROR" if the error is not an Error.
func Reason(err error) string {
	if err == nil {
		return ""
	}
	if e, ok := err.(*Error); ok {
		return e.Reason
	}
	return "INTERNAL_ERROR"
}
