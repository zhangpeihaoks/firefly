package errors

import (
	"google.golang.org/grpc/codes"
)

// Common HTTP status codes
const (
	CodeOK                 = 200
	CodeBadRequest         = 400
	CodeUnauthorized       = 401
	CodeForbidden          = 403
	CodeNotFound           = 404
	CodeConflict           = 409
	CodeInternal           = 500
	CodeServiceUnavailable = 503
)

// Standard error instances
var (
	// ErrBadRequest represents a 400 Bad Request error
	ErrBadRequest = New(CodeBadRequest, "BAD_REQUEST", "请求参数错误")
	// ErrUnauthorized represents a 401 Unauthorized error
	ErrUnauthorized = New(CodeUnauthorized, "UNAUTHORIZED", "未授权访问")
	// ErrForbidden represents a 403 Forbidden error
	ErrForbidden = New(CodeForbidden, "FORBIDDEN", "禁止访问")
	// ErrNotFound represents a 404 Not Found error
	ErrNotFound = New(CodeNotFound, "NOT_FOUND", "资源不存在")
	// ErrInternal represents a 500 Internal Server Error
	ErrInternal = New(CodeInternal, "INTERNAL_ERROR", "内部服务错误")
	// ErrServiceUnavailable represents a 503 Service Unavailable error
	ErrServiceUnavailable = New(CodeServiceUnavailable, "SERVICE_UNAVAILABLE", "服务不可用")
)

// HTTPToGRPCCode converts an HTTP status code to a gRPC status code.
func HTTPToGRPCCode(code int) codes.Code {
	switch code {
	case CodeOK:
		return codes.OK
	case CodeBadRequest:
		return codes.InvalidArgument
	case CodeUnauthorized:
		return codes.Unauthenticated
	case CodeForbidden:
		return codes.PermissionDenied
	case CodeNotFound:
		return codes.NotFound
	case CodeConflict:
		return codes.AlreadyExists
	case CodeInternal:
		return codes.Internal
	case CodeServiceUnavailable:
		return codes.Unavailable
	default:
		return codes.Unknown
	}
}

// GRPCToHTTPCode converts a gRPC status code to an HTTP status code.
func GRPCToHTTPCode(code codes.Code) int {
	switch code {
	case codes.OK:
		return CodeOK
	case codes.InvalidArgument:
		return CodeBadRequest
	case codes.Unauthenticated:
		return CodeUnauthorized
	case codes.PermissionDenied:
		return CodeForbidden
	case codes.NotFound:
		return CodeNotFound
	case codes.AlreadyExists:
		return CodeConflict
	case codes.Internal:
		return CodeInternal
	case codes.Unavailable:
		return CodeServiceUnavailable
	default:
		return CodeInternal
	}
}

// ToHTTPStatus converts an error code to an HTTP status code.
// This is a convenience function that ensures the code is a valid HTTP status.
func ToHTTPStatus(code int) int {
	// Map common error codes to HTTP status codes
	switch code {
	case CodeOK:
		return CodeOK
	case CodeBadRequest:
		return CodeBadRequest
	case CodeUnauthorized:
		return CodeUnauthorized
	case CodeForbidden:
		return CodeForbidden
	case CodeNotFound:
		return CodeNotFound
	case CodeConflict:
		return CodeConflict
	case CodeInternal:
		return CodeInternal
	case CodeServiceUnavailable:
		return CodeServiceUnavailable
	default:
		// Ensure it's a valid HTTP status code
		if code >= 100 && code < 600 {
			return code
		}
		// Default to internal server error for invalid codes
		return CodeInternal
	}
}
