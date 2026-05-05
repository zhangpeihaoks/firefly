// Package errors_test provides unit tests for the errors package.
// Feature: backend-server-framework, Property 9: 错误创建和提取
// Feature: backend-server-framework, Property 10: 状态码双向转换
package errors_test

import (
	stderrors "errors"
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"testing/quick"

	fferr "github.com/zhangpeihaoks/firefly/internal/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// =============================================================================
// Unit Tests - Error Creation (Property 9)
// =============================================================================

// TestErrorCreation_New tests New() function creates Error correctly
// Validates: Requirement 4.4
func TestErrorCreation_New(t *testing.T) {
	tests := []struct {
		name    string
		code    int
		reason  string
		message string
	}{
		{"basic error", 400, "BAD_REQUEST", "Invalid request parameter"},
		{"not found", 404, "NOT_FOUND", "Resource not found"},
		{"internal error", 500, "INTERNAL_ERROR", "Database connection failed"},
		{"custom code", 418, "TEAPOT", "I am a teapot"},
		{"empty message", 400, "BAD_REQUEST", ""},
		{"unicode message", 400, "BAD_REQUEST", "错误信息"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := fferr.New(tt.code, tt.reason, tt.message)

			if err == nil {
				t.Fatal("New() should return non-nil error")
			}
			if err.Code != int32(tt.code) {
				t.Errorf("expected Code %d, got %d", tt.code, err.Code)
			}
			if err.Reason != tt.reason {
				t.Errorf("expected Reason %s, got %s", tt.reason, err.Reason)
			}
			if err.Message != tt.message {
				t.Errorf("expected Message %s, got %s", tt.message, err.Message)
			}
		})
	}
}

// TestErrorCreation_Newf tests Newf() function with formatted messages
// Validates: Requirement 4.4
func TestErrorCreation_Newf(t *testing.T) {
	tests := []struct {
		name     string
		code     int
		reason   string
		format   string
		args     []any
		expected string
	}{
		{"single arg", 400, "BAD_REQUEST", "Invalid value: %s", []any{"test"}, "Invalid value: test"},
		{"multiple args", 500, "INTERNAL_ERROR", "Failed %s at step %d", []any{"connection", 3}, "Failed connection at step 3"},
		{"no args", 400, "BAD_REQUEST", "Static message", []any{}, "Static message"},
		{"special chars", 400, "BAD_REQUEST", "Value: %s", []any{"%100"}, "Value: %100"},
		{"empty args", 400, "BAD_REQUEST", "Error: %s %s", []any{"", ""}, "Error:  "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := fferr.Newf(tt.code, tt.reason, tt.format, tt.args...)

			if err == nil {
				t.Fatal("Newf() should return non-nil error")
			}
			if err.Code != int32(tt.code) {
				t.Errorf("expected Code %d, got %d", tt.code, err.Code)
			}
			if err.Reason != tt.reason {
				t.Errorf("expected Reason %s, got %s", tt.reason, err.Reason)
			}
			if err.Message != tt.expected {
				t.Errorf("expected Message %q, got %q", tt.expected, err.Message)
			}
		})
	}
}

// TestErrorCreation_FieldAccess tests all Error fields are accessible
// Validates: Requirements 4.1, 4.4
func TestErrorCreation_FieldAccess(t *testing.T) {
	err := fferr.New(400, "BAD_REQUEST", "test message")
	err.Metadata = map[string]string{"key": "value"}

	// Test all field access
	if got := err.Code; got != 400 {
		t.Errorf("Code field access failed: expected 400, got %d", got)
	}
	if got := err.Reason; got != "BAD_REQUEST" {
		t.Errorf("Reason field access failed: expected BAD_REQUEST, got %s", got)
	}
	if got := err.Message; got != "test message" {
		t.Errorf("Message field access failed: expected test message, got %s", got)
	}
	if got := err.Metadata["key"]; got != "value" {
		t.Errorf("Metadata field access failed: expected value, got %s", got)
	}
}

// TestErrorCreation_NilMetadata tests Error with nil Metadata field
// Validates: Requirement 4.1
func TestErrorCreation_NilMetadata(t *testing.T) {
	err := fferr.New(400, "BAD_REQUEST", "test")

	// Metadata should be nil by default
	if err.Metadata != nil {
		t.Error("Metadata should be nil by default")
	}
}

// =============================================================================
// Unit Tests - Error Extraction (Property 9)
// =============================================================================

// TestErrorExtraction_FromError tests FromError() extracts Error from fferr.Error
// Validates: Requirement 4.5
func TestErrorExtraction_FromError(t *testing.T) {
	tests := []struct {
		name         string
		inputErr     error
		expectNil    bool
		expectedCode int
		expectedMsg  string
	}{
		{
			name:         "nil error returns nil",
			inputErr:     nil,
			expectNil:    true,
			expectedCode: 0,
		},
		{
			name:         "Error type extracts correctly",
			inputErr:     fferr.New(404, "NOT_FOUND", "resource not found"),
			expectNil:    false,
			expectedCode: 404,
			expectedMsg:  "resource not found",
		},
		{
			name:         "standard error converts to internal",
			inputErr:     stderrors.New("standard error"),
			expectNil:    false,
			expectedCode: fferr.CodeInternal,
			expectedMsg:  "standard error",
		},
		{
			name:         "wrapped error message preserved",
			inputErr:     fmt.Errorf("wrapped: %w", stderrors.New("inner")),
			expectNil:    false,
			expectedCode: fferr.CodeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fferr.FromError(tt.inputErr)

			if tt.expectNil {
				if result != nil {
					t.Error("expected nil result, got non-nil")
				}
				return
			}

			if result == nil {
				t.Fatal("expected non-nil result")
			}

			if tt.expectedCode != 0 && int(result.Code) != tt.expectedCode {
				t.Errorf("expected Code %d, got %d", tt.expectedCode, result.Code)
			}

			if tt.expectedMsg != "" && !strings.Contains(result.Message, tt.expectedMsg) {
				t.Errorf("expected Message to contain %q, got %q", tt.expectedMsg, result.Message)
			}
		})
	}
}

// TestErrorExtraction_FromErrorGRPC tests FromError() extracts from gRPC status
// Validates: Requirement 4.5
func TestErrorExtraction_FromErrorGRPC(t *testing.T) {
	tests := []struct {
		name           string
		grpcCode       codes.Code
		message        string
		expectedCode   int
		expectedReason string
	}{
		{
			name:           "NotFound maps to HTTP 404",
			grpcCode:       codes.NotFound,
			message:        "item not found",
			expectedCode:   fferr.CodeNotFound,
			expectedReason: "NotFound",
		},
		{
			name:           "InvalidArgument maps to HTTP 400",
			grpcCode:       codes.InvalidArgument,
			message:        "invalid param",
			expectedCode:   fferr.CodeBadRequest,
			expectedReason: "InvalidArgument",
		},
		{
			name:           "Unauthenticated maps to HTTP 401",
			grpcCode:       codes.Unauthenticated,
			message:        "unauthorized",
			expectedCode:   fferr.CodeUnauthorized,
			expectedReason: "Unauthenticated",
		},
		{
			name:           "PermissionDenied maps to HTTP 403",
			grpcCode:       codes.PermissionDenied,
			message:        "forbidden",
			expectedCode:   fferr.CodeForbidden,
			expectedReason: "PermissionDenied",
		},
		{
			name:           "AlreadyExists maps to HTTP 409",
			grpcCode:       codes.AlreadyExists,
			message:        "conflict",
			expectedCode:   fferr.CodeConflict,
			expectedReason: "AlreadyExists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			grpcStatus := status.New(tt.grpcCode, tt.message)
			extracted := fferr.FromError(grpcStatus.Err())

			if extracted == nil {
				t.Fatal("FromError should extract from gRPC status")
			}

			if extracted.Code != int32(tt.expectedCode) {
				t.Errorf("expected Code %d, got %d", tt.expectedCode, extracted.Code)
			}

			if extracted.Reason != tt.expectedReason {
				t.Errorf("expected Reason %s, got %s", tt.expectedReason, extracted.Reason)
			}

			if extracted.Message != tt.message {
				t.Errorf("expected Message %s, got %s", tt.message, extracted.Message)
			}
		})
	}
}

// TestErrorExtraction_CodeFunction tests Code() function returns correct code
// Validates: Requirement 4.6
func TestErrorExtraction_CodeFunction(t *testing.T) {
	tests := []struct {
		name         string
		inputErr     error
		expectedCode int
	}{
		{
			name:         "nil returns CodeOK",
			inputErr:     nil,
			expectedCode: fferr.CodeOK,
		},
		{
			name:         "Error returns correct code",
			inputErr:     fferr.New(404, "NOT_FOUND", "test"),
			expectedCode: 404,
		},
		{
			name:         "standard error returns CodeInternal",
			inputErr:     stderrors.New("test"),
			expectedCode: fferr.CodeInternal,
		},
		{
			name:         "predefined ErrBadRequest",
			inputErr:     fferr.ErrBadRequest,
			expectedCode: fferr.CodeBadRequest,
		},
		{
			name:         "predefined ErrNotFound",
			inputErr:     fferr.ErrNotFound,
			expectedCode: fferr.CodeNotFound,
		},
		{
			name:         "predefined ErrInternal",
			inputErr:     fferr.ErrInternal,
			expectedCode: fferr.CodeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := fferr.Code(tt.inputErr)
			if code != tt.expectedCode {
				t.Errorf("expected Code %d, got %d", tt.expectedCode, code)
			}
		})
	}
}

// TestErrorExtraction_ReasonFunction tests Reason() function returns correct reason
// Validates: Requirement 4.6
func TestErrorExtraction_ReasonFunction(t *testing.T) {
	tests := []struct {
		name           string
		inputErr       error
		expectedReason string
	}{
		{
			name:           "nil returns empty string",
			inputErr:       nil,
			expectedReason: "",
		},
		{
			name:           "Error returns correct reason",
			inputErr:       fferr.New(404, "NOT_FOUND", "test"),
			expectedReason: "NOT_FOUND",
		},
		{
			name:           "standard error returns INTERNAL_ERROR",
			inputErr:       stderrors.New("test"),
			expectedReason: "INTERNAL_ERROR",
		},
		{
			name:           "predefined ErrBadRequest",
			inputErr:       fferr.ErrBadRequest,
			expectedReason: "BAD_REQUEST",
		},
		{
			name:           "predefined ErrNotFound",
			inputErr:       fferr.ErrNotFound,
			expectedReason: "NOT_FOUND",
		},
		{
			name:           "predefined ErrInternal",
			inputErr:       fferr.ErrInternal,
			expectedReason: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reason := fferr.Reason(tt.inputErr)
			if reason != tt.expectedReason {
				t.Errorf("expected Reason %q, got %q", tt.expectedReason, reason)
			}
		})
	}
}

// TestErrorExtraction_StandardErrorTypes tests FromError with various standard error types
// Validates: Requirement 4.5
func TestErrorExtraction_StandardErrorTypes(t *testing.T) {
	// Test with wrapped errors
	wrappedErr := fmt.Errorf("outer error: %w", fferr.New(400, "TEST", "inner"))
	extracted := fferr.FromError(wrappedErr)
	if extracted == nil {
		t.Error("FromError should extract from wrapped error")
	}

	// Test with errorf
	errfErr := fmt.Errorf("formatted error: %d %s", 42, "test")
	extracted = fferr.FromError(errfErr)
	if extracted == nil {
		t.Error("FromError should handle errorf")
	}
	if extracted.Code != fferr.CodeInternal {
		t.Errorf("expected CodeInternal for errorf, got %d", extracted.Code)
	}
}

// =============================================================================
// Unit Tests - Error Comparison (Property 9)
// =============================================================================

// TestErrorComparison_Is tests Is() method implements error comparison
// Validates: Requirements 4.2
func TestErrorComparison_Is(t *testing.T) {
	err1 := fferr.New(404, "NOT_FOUND", "resource not found")
	err2 := fferr.New(404, "NOT_FOUND", "different message")
	err3 := fferr.New(404, "DIFFERENT", "resource not found")
	err4 := fferr.New(500, "NOT_FOUND", "resource not found")
	err5 := fferr.New(404, "NOT_FOUND", "resource not found")

	// Same code and reason should match
	if !err1.Is(err2) {
		t.Error("errors with same Code and Reason should match")
	}

	// Different reason should not match
	if err1.Is(err3) {
		t.Error("errors with different Reason should not match")
	}

	// Different code should not match
	if err1.Is(err4) {
		t.Error("errors with different Code should not match")
	}

	// Same values should match (reflexive)
	if !err1.Is(err5) {
		t.Error("errors with identical Code and Reason should match")
	}
}

// TestErrorComparison_IsWithStandardErrors tests Is() with standard errors
// Validates: Requirement 4.2
func TestErrorComparison_IsWithStandardErrors(t *testing.T) {
	fferrErr := fferr.ErrNotFound
	stdErr := stderrors.New("test")

	// Error should not match standard error
	if fferrErr.Is(stdErr) {
		t.Error("Error should not match standard error")
	}

	// Test errors.Is compatibility
	if !stderrors.Is(fferrErr, fferr.ErrNotFound) {
		t.Error("errors.Is should return true for same Error instance")
	}

	if stderrors.Is(fferrErr, fferr.ErrBadRequest) {
		t.Error("errors.Is should return false for different Error")
	}
}

// TestErrorComparison_ErrorInterface tests Error() method output format
// Validates: Requirement 4.2
func TestErrorComparison_ErrorInterface(t *testing.T) {
	err := fferr.New(400, "BAD_REQUEST", "invalid parameter")
	errStr := err.Error()

	// Error() should not be empty
	if errStr == "" {
		t.Error("Error() should not return empty string")
	}

	// Error() should contain all fields
	if !strings.Contains(errStr, "code = 400") {
		t.Errorf("Error() should contain code, got %s", errStr)
	}
	if !strings.Contains(errStr, "reason = BAD_REQUEST") {
		t.Errorf("Error() should contain reason, got %s", errStr)
	}
	if !strings.Contains(errStr, "message = invalid parameter") {
		t.Errorf("Error() should contain message, got %s", errStr)
	}
}

// TestErrorComparison_PredefinedErrors tests predefined error constants
// Validates: Requirements 4.4, 4.5
func TestErrorComparison_PredefinedErrors(t *testing.T) {
	// Test predefined errors have correct values
	if fferr.ErrBadRequest.Code != fferr.CodeBadRequest {
		t.Errorf("ErrBadRequest.Code = %d, want %d", fferr.ErrBadRequest.Code, fferr.CodeBadRequest)
	}
	if fferr.ErrBadRequest.Reason != "BAD_REQUEST" {
		t.Errorf("ErrBadRequest.Reason = %s, want BAD_REQUEST", fferr.ErrBadRequest.Reason)
	}

	if fferr.ErrUnauthorized.Code != fferr.CodeUnauthorized {
		t.Errorf("ErrUnauthorized.Code = %d, want %d", fferr.ErrUnauthorized.Code, fferr.CodeUnauthorized)
	}

	if fferr.ErrForbidden.Code != fferr.CodeForbidden {
		t.Errorf("ErrForbidden.Code = %d, want %d", fferr.ErrForbidden.Code, fferr.CodeForbidden)
	}

	if fferr.ErrNotFound.Code != fferr.CodeNotFound {
		t.Errorf("ErrNotFound.Code = %d, want %d", fferr.ErrNotFound.Code, fferr.CodeNotFound)
	}

	if fferr.ErrInternal.Code != fferr.CodeInternal {
		t.Errorf("ErrInternal.Code = %d, want %d", fferr.ErrInternal.Code, fferr.CodeInternal)
	}

	if fferr.ErrServiceUnavailable.Code != fferr.CodeServiceUnavailable {
		t.Errorf("ErrServiceUnavailable.Code = %d, want %d", fferr.ErrServiceUnavailable.Code, fferr.CodeServiceUnavailable)
	}
}

// TestErrorStructure tests the Error structure fields
// Validates: Requirements 4.1, 4.2
func TestErrorStructure(t *testing.T) {
	err := fferr.New(400, "BAD_REQUEST", "Invalid parameter")

	if err.Code != 400 {
		t.Errorf("expected Code 400, got %d", err.Code)
	}
	if err.Reason != "BAD_REQUEST" {
		t.Errorf("expected Reason BAD_REQUEST, got %s", err.Reason)
	}
	if err.Message != "Invalid parameter" {
		t.Errorf("expected Message 'Invalid parameter', got %s", err.Message)
	}
}

// TestErrorInterface tests that Error implements the error interface
// Validates: Requirements 4.2
func TestErrorInterface(t *testing.T) {
	err := fferr.New(404, "NOT_FOUND", "Resource not found")

	// Verify it implements the error interface
	var _ error = err

	// Verify Error() method output
	errStr := err.Error()
	if errStr == "" {
		t.Error("Error() should not return empty string")
	}
}

// TestNewf tests the Newf function with formatted message
// Validates: Requirements 4.4
func TestNewf(t *testing.T) {
	err := fferr.Newf(500, "INTERNAL_ERROR", "Failed to process %s: %d", "item", 42)

	if err.Code != 500 {
		t.Errorf("expected Code 500, got %d", err.Code)
	}
	if err.Reason != "INTERNAL_ERROR" {
		t.Errorf("expected Reason INTERNAL_ERROR, got %s", err.Reason)
	}
	if err.Message != "Failed to process item: 42" {
		t.Errorf("expected formatted message, got %s", err.Message)
	}
}

// TestGRPCStatus tests the GRPCStatus method
// Validates: Requirements 4.3
func TestGRPCStatus(t *testing.T) {
	tests := []struct {
		name         string
		httpCode     int
		expectedGRPC codes.Code
	}{
		{"OK", 200, codes.OK},
		{"BadRequest", 400, codes.InvalidArgument},
		{"Unauthorized", 401, codes.Unauthenticated},
		{"Forbidden", 403, codes.PermissionDenied},
		{"NotFound", 404, codes.NotFound},
		{"Conflict", 409, codes.AlreadyExists},
		{"Internal", 500, codes.Internal},
		{"ServiceUnavailable", 503, codes.Unavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := fferr.New(tt.httpCode, "TEST_REASON", "test message")
			s := err.GRPCStatus()

			if s.Code() != tt.expectedGRPC {
				t.Errorf("expected gRPC code %v, got %v", tt.expectedGRPC, s.Code())
			}
		})
	}
}

// TestWithMetadata tests the WithMetadata method
// Validates: Requirements 4.8
func TestWithMetadata(t *testing.T) {
	err := fferr.New(400, "BAD_REQUEST", "Invalid parameter")
	result := err.WithMetadata(map[string]string{
		"field": "email",
		"value": "invalid",
	})

	if result == nil {
		t.Fatal("WithMetadata should return non-nil error")
	}
	if result.Metadata == nil {
		t.Fatal("Metadata should not be nil")
	}
	if result.Metadata["field"] != "email" {
		t.Errorf("expected field=email, got %s", result.Metadata["field"])
	}
	if result.Metadata["value"] != "invalid" {
		t.Errorf("expected value=invalid, got %s", result.Metadata["value"])
	}
}

// TestWithMetadataNil tests WithMetadata with nil initial metadata
// Validates: Requirements 4.8
func TestWithMetadataNil(t *testing.T) {
	err := &fferr.Error{
		Code:    400,
		Reason:  "BAD_REQUEST",
		Message: "test",
	}

	if err.Metadata != nil {
		t.Fatal("initial Metadata should be nil for this test")
	}

	result := err.WithMetadata(map[string]string{"key": "value"})
	if result.Metadata == nil {
		t.Fatal("Metadata should be initialized")
	}
	if result.Metadata["key"] != "value" {
		t.Errorf("expected key=value, got %s", result.Metadata["key"])
	}
}

// TestWithMetadataMerge tests that WithMetadata merges with existing metadata
// Validates: Requirements 4.8
func TestWithMetadataMerge(t *testing.T) {
	err := fferr.New(400, "BAD_REQUEST", "test")
	err.WithMetadata(map[string]string{"existing": "value"})

	result := err.WithMetadata(map[string]string{"new": "data"})

	if result.Metadata["existing"] != "value" {
		t.Errorf("existing metadata should be preserved")
	}
	if result.Metadata["new"] != "data" {
		t.Errorf("new metadata should be added")
	}
}

// TestIs tests the Is method for error comparison
// Validates: Requirements 4.2
func TestIs(t *testing.T) {
	err1 := fferr.New(404, "NOT_FOUND", "Resource not found")
	err2 := fferr.New(404, "NOT_FOUND", "Different message")
	err3 := fferr.New(404, "DIFFERENT", "Resource not found")
	err4 := fferr.New(500, "NOT_FOUND", "Resource not found")

	if !err1.Is(err2) {
		t.Error("errors with same Code and Reason should be equal")
	}
	if err1.Is(err3) {
		t.Error("errors with different Reason should not be equal")
	}
	if err1.Is(err4) {
		t.Error("errors with different Code should not be equal")
	}
}

// TestFromError tests the FromError function
// Validates: Requirements 4.5
func TestFromError(t *testing.T) {
	// Test with nil error
	if result := fferr.FromError(nil); result != nil {
		t.Error("FromError(nil) should return nil")
	}

	// Test with Error type
	original := fferr.New(404, "NOT_FOUND", "Resource not found")
	extracted := fferr.FromError(original)
	if extracted == nil {
		t.Fatal("FromError should return non-nil for Error type")
	}
	if extracted.Code != original.Code {
		t.Errorf("expected Code %d, got %d", original.Code, extracted.Code)
	}

	// Test with standard error
	stdErr := stderrors.New("standard error")
	extracted = fferr.FromError(stdErr)
	if extracted == nil {
		t.Fatal("FromError should return non-nil for standard error")
	}
	if extracted.Code != fferr.CodeInternal {
		t.Errorf("expected Code %d for standard error, got %d", fferr.CodeInternal, extracted.Code)
	}
}

// TestFromErrorGRPCStatus tests FromError with gRPC status
// Validates: Requirements 4.5
func TestFromErrorGRPCStatus(t *testing.T) {
	// Create a gRPC status error
	grpcStatus := status.New(codes.NotFound, "resource not found")
	grpcErr := grpcStatus.Err()

	extracted := fferr.FromError(grpcErr)
	if extracted == nil {
		t.Fatal("FromError should extract from gRPC status")
	}
	if extracted.Code != fferr.CodeNotFound {
		t.Errorf("expected HTTP Code %d, got %d", fferr.CodeNotFound, extracted.Code)
	}
}

// TestCodeFunction tests the Code function
// Validates: Requirements 4.6
func TestCodeFunction(t *testing.T) {
	// Test with nil error
	if code := fferr.Code(nil); code != fferr.CodeOK {
		t.Errorf("Code(nil) should return %d, got %d", fferr.CodeOK, code)
	}

	// Test with Error type
	err := fferr.New(404, "NOT_FOUND", "test")
	if code := fferr.Code(err); code != 404 {
		t.Errorf("expected Code 404, got %d", code)
	}

	// Test with standard error
	stdErr := stderrors.New("standard error")
	if code := fferr.Code(stdErr); code != fferr.CodeInternal {
		t.Errorf("expected Code %d for standard error, got %d", fferr.CodeInternal, code)
	}
}

// TestReasonFunction tests the Reason function
// Validates: Requirements 4.6
func TestReasonFunction(t *testing.T) {
	// Test with nil error
	if reason := fferr.Reason(nil); reason != "" {
		t.Errorf("Reason(nil) should return empty string, got %s", reason)
	}

	// Test with Error type
	err := fferr.New(404, "NOT_FOUND", "test")
	if reason := fferr.Reason(err); reason != "NOT_FOUND" {
		t.Errorf("expected Reason NOT_FOUND, got %s", reason)
	}

	// Test with standard error
	stdErr := stderrors.New("standard error")
	if reason := fferr.Reason(stdErr); reason != "INTERNAL_ERROR" {
		t.Errorf("expected Reason INTERNAL_ERROR for standard error, got %s", reason)
	}
}

// TestHTTPToGRPCCode tests HTTP to gRPC code conversion
// Validates: Requirements 4.7
func TestHTTPToGRPCCode(t *testing.T) {
	tests := []struct {
		httpCode     int
		expectedGRPC codes.Code
	}{
		{200, codes.OK},
		{400, codes.InvalidArgument},
		{401, codes.Unauthenticated},
		{403, codes.PermissionDenied},
		{404, codes.NotFound},
		{409, codes.AlreadyExists},
		{500, codes.Internal},
		{503, codes.Unavailable},
		{999, codes.Unknown}, // Unknown code
	}

	for _, tt := range tests {
		result := fferr.HTTPToGRPCCode(tt.httpCode)
		if result != tt.expectedGRPC {
			t.Errorf("HTTPToGRPCCode(%d) = %v, want %v", tt.httpCode, result, tt.expectedGRPC)
		}
	}
}

// TestGRPCToHTTPCode tests gRPC to HTTP code conversion
// Validates: Requirements 4.7
func TestGRPCToHTTPCode(t *testing.T) {
	tests := []struct {
		grpcCode     codes.Code
		expectedHTTP int
	}{
		{codes.OK, 200},
		{codes.InvalidArgument, 400},
		{codes.Unauthenticated, 401},
		{codes.PermissionDenied, 403},
		{codes.NotFound, 404},
		{codes.AlreadyExists, 409},
		{codes.Internal, 500},
		{codes.Unavailable, 503},
		{codes.Unknown, 500}, // Unknown code defaults to 500
	}

	for _, tt := range tests {
		result := fferr.GRPCToHTTPCode(tt.grpcCode)
		if result != tt.expectedHTTP {
			t.Errorf("GRPCToHTTPCode(%v) = %d, want %d", tt.grpcCode, result, tt.expectedHTTP)
		}
	}
}

// TestPredefinedErrors tests predefined error constants
// Validates: Requirements 4.7
func TestPredefinedErrors(t *testing.T) {
	if fferr.CodeOK != 200 {
		t.Errorf("CodeOK should be 200, got %d", fferr.CodeOK)
	}
	if fferr.CodeBadRequest != 400 {
		t.Errorf("CodeBadRequest should be 400, got %d", fferr.CodeBadRequest)
	}
	if fferr.CodeUnauthorized != 401 {
		t.Errorf("CodeUnauthorized should be 401, got %d", fferr.CodeUnauthorized)
	}
	if fferr.CodeForbidden != 403 {
		t.Errorf("CodeForbidden should be 403, got %d", fferr.CodeForbidden)
	}
	if fferr.CodeNotFound != 404 {
		t.Errorf("CodeNotFound should be 404, got %d", fferr.CodeNotFound)
	}
	if fferr.CodeConflict != 409 {
		t.Errorf("CodeConflict should be 409, got %d", fferr.CodeConflict)
	}
	if fferr.CodeInternal != 500 {
		t.Errorf("CodeInternal should be 500, got %d", fferr.CodeInternal)
	}
	if fferr.CodeServiceUnavailable != 503 {
		t.Errorf("CodeServiceUnavailable should be 503, got %d", fferr.CodeServiceUnavailable)
	}
}

// TestStandardErrorInstances tests standard error instances
// Validates: Requirements 4.7
func TestStandardErrorInstances(t *testing.T) {
	if fferr.ErrBadRequest.Code != fferr.CodeBadRequest {
		t.Errorf("ErrBadRequest should have Code %d", fferr.CodeBadRequest)
	}
	if fferr.ErrUnauthorized.Code != fferr.CodeUnauthorized {
		t.Errorf("ErrUnauthorized should have Code %d", fferr.CodeUnauthorized)
	}
	if fferr.ErrForbidden.Code != fferr.CodeForbidden {
		t.Errorf("ErrForbidden should have Code %d", fferr.CodeForbidden)
	}
	if fferr.ErrNotFound.Code != fferr.CodeNotFound {
		t.Errorf("ErrNotFound should have Code %d", fferr.CodeNotFound)
	}
	if fferr.ErrInternal.Code != fferr.CodeInternal {
		t.Errorf("ErrInternal should have Code %d", fferr.CodeInternal)
	}
	if fferr.ErrServiceUnavailable.Code != fferr.CodeServiceUnavailable {
		t.Errorf("ErrServiceUnavailable should have Code %d", fferr.CodeServiceUnavailable)
	}
}

// =============================================================================
// Unit Tests - Property 10: 状态码双向转换 (Requirements 4.7)
// =============================================================================

// Feature: backend-server-framework, Property 10: 状态码双向转换

// TestHTTPToGRPCCodeWithEdgeCases tests HTTP to gRPC code conversion with edge cases
// Validates: Requirement 4.7
func TestHTTPToGRPCCodeWithEdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		httpCode     int
		expectedGRPC codes.Code
	}{
		// Standard HTTP status codes
		{"OK", 200, codes.OK},
		{"BadRequest", 400, codes.InvalidArgument},
		{"Unauthorized", 401, codes.Unauthenticated},
		{"Forbidden", 403, codes.PermissionDenied},
		{"NotFound", 404, codes.NotFound},
		{"Conflict", 409, codes.AlreadyExists},
		{"Internal", 500, codes.Internal},
		{"ServiceUnavailable", 503, codes.Unavailable},
		// Edge cases - unknown codes
		{"UnknownCode_999", 999, codes.Unknown},
		{"UnknownCode_418", 418, codes.Unknown},
		{"UnknownCode_599", 599, codes.Unknown},
		// Boundary values
		{"Code_100", 100, codes.Unknown},
		{"Code_199", 199, codes.Unknown},
		{"Code_600", 600, codes.Unknown},
		// Additional valid codes
		{"GatewayTimeout", 504, codes.Unknown},
		{"BadGateway", 502, codes.Unknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fferr.HTTPToGRPCCode(tt.httpCode)
			if result != tt.expectedGRPC {
				t.Errorf("HTTPToGRPCCode(%d) = %v, want %v", tt.httpCode, result, tt.expectedGRPC)
			}
		})
	}
}

// TestGRPCToHTTPCodeWithEdgeCases tests gRPC to HTTP code conversion with edge cases
// Validates: Requirement 4.7
func TestGRPCToHTTPCodeWithEdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		grpcCode     codes.Code
		expectedHTTP int
	}{
		// Standard gRPC codes
		{"OK", codes.OK, 200},
		{"InvalidArgument", codes.InvalidArgument, 400},
		{"Unauthenticated", codes.Unauthenticated, 401},
		{"PermissionDenied", codes.PermissionDenied, 403},
		{"NotFound", codes.NotFound, 404},
		{"AlreadyExists", codes.AlreadyExists, 409},
		{"Internal", codes.Internal, 500},
		{"Unavailable", codes.Unavailable, 503},
		// Edge cases - unknown codes default to Internal (500)
		{"Unknown", codes.Unknown, 500},
		{"Canceled", codes.Canceled, 500},
		{"DeadlineExceeded", codes.DeadlineExceeded, 500},
		{"ResourceExhausted", codes.ResourceExhausted, 500},
		{"Aborted", codes.Aborted, 500},
		{"OutOfRange", codes.OutOfRange, 500},
		{"Unimplemented", codes.Unimplemented, 500},
		{"Internal_grpc", codes.Internal, 500},
		{"DataLoss", codes.DataLoss, 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fferr.GRPCToHTTPCode(tt.grpcCode)
			if result != tt.expectedHTTP {
				t.Errorf("GRPCToHTTPCode(%v) = %d, want %d", tt.grpcCode, result, tt.expectedHTTP)
			}
		})
	}
}

// TestToHTTPStatus tests the ToHTTPStatus helper function
// Validates: Requirement 4.7
func TestToHTTPStatus(t *testing.T) {
	tests := []struct {
		name         string
		inputCode    int
		expectedHTTP int
	}{
		// Known codes pass through
		{"OK", 200, 200},
		{"BadRequest", 400, 400},
		{"Unauthorized", 401, 401},
		{"Forbidden", 403, 403},
		{"NotFound", 404, 404},
		{"Conflict", 409, 409},
		{"Internal", 500, 500},
		{"ServiceUnavailable", 503, 503},
		// Valid HTTP codes outside known range pass through
		{"Created", 201, 201},
		{"Accepted", 202, 202},
		{"NoContent", 204, 204},
		{"MovedPermanently", 301, 301},
		{"Found", 302, 302},
		{"SeeOther", 303, 303},
		{"TemporaryRedirect", 307, 307},
		{"PermanentRedirect", 308, 308},
		{"BadGateway", 502, 502},
		{"GatewayTimeout", 504, 504},
		// Invalid codes default to Internal (500)
		{"Invalid_99", 99, fferr.CodeInternal},
		{"Invalid_600", 600, fferr.CodeInternal},
		{"Invalid_700", 700, fferr.CodeInternal},
		{"Invalid_Negative", -1, fferr.CodeInternal},
		{"Invalid_1000", 1000, fferr.CodeInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fferr.ToHTTPStatus(tt.inputCode)
			if result != tt.expectedHTTP {
				t.Errorf("ToHTTPStatus(%d) = %d, want %d", tt.inputCode, result, tt.expectedHTTP)
			}
		})
	}
}

// TestBidirectionalConversion tests that HTTP->gRPC->HTTP roundtrip preserves meaning
// Validates: Requirement 4.7
func TestBidirectionalConversion(t *testing.T) {
	// Test round-trip for mapped codes
	httpCodes := []int{
		fferr.CodeOK,
		fferr.CodeBadRequest,
		fferr.CodeUnauthorized,
		fferr.CodeForbidden,
		fferr.CodeNotFound,
		fferr.CodeConflict,
		fferr.CodeInternal,
		fferr.CodeServiceUnavailable,
	}

	for _, httpCode := range httpCodes {
		// HTTP -> gRPC -> HTTP
		grpcCode := fferr.HTTPToGRPCCode(httpCode)
		resultHTTP := fferr.GRPCToHTTPCode(grpcCode)

		// For known codes, round-trip should preserve the code
		if resultHTTP != httpCode {
			t.Errorf("Round-trip failed for HTTP %d: got %d", httpCode, resultHTTP)
		}
	}
}

// TestGRPCBidirectionalConversion tests that gRPC->HTTP->gRPC roundtrip preserves meaning
// Validates: Requirement 4.7
func TestGRPCBidirectionalConversion(t *testing.T) {
	// Test round-trip for mapped codes
	grpcCodes := []codes.Code{
		codes.OK,
		codes.InvalidArgument,
		codes.Unauthenticated,
		codes.PermissionDenied,
		codes.NotFound,
		codes.AlreadyExists,
		codes.Internal,
		codes.Unavailable,
	}

	for _, grpcCode := range grpcCodes {
		// gRPC -> HTTP -> gRPC
		httpCode := fferr.GRPCToHTTPCode(grpcCode)
		resultGRPC := fferr.HTTPToGRPCCode(httpCode)

		// For known codes, round-trip should preserve the code
		if resultGRPC != grpcCode {
			t.Errorf("Round-trip failed for gRPC %v: got %v", grpcCode, resultGRPC)
		}
	}
}

// =============================================================================
// Property-Based Tests (Property 7 & Property 8)
// =============================================================================

// Feature: backend-server-framework, Property 7: 错误结构正确性
// Property 7: Error Structure Correctness
// Validates: Requirements 4.1, 4.2
//
// For any Error instance:
// - All fields (Code, Reason, Message, Metadata) should be correctly accessible
// - Error() method output should contain all field values
func TestProperty7ErrorStructureCorrectness(t *testing.T) {
	type errorTestCase struct {
		code         int
		reason       string
		message      string
		metadataKeys []string
	}

	if err := quick.Check(
		func(tc errorTestCase) bool {
			// Constrain code to reasonable HTTP status code range (100-699)
			httpCode := 100 + (tc.code % 600)
			reason := filterPrintableASCII(tc.reason)
			message := filterPrintableASCII(tc.message)

			// Generate valid metadata keys (max 5 keys)
			metadata := make(map[string]string)
			numKeys := len(tc.metadataKeys)
			if numKeys > 5 {
				numKeys = 5
			}
			for i := 0; i < numKeys; i++ {
				key := fmt.Sprintf("key%d", i)
				metadata[key] = fmt.Sprintf("value%d", i)
			}

			testErr := fferr.New(httpCode, reason, message).WithMetadata(metadata)

			// Field access correctness (Requirement 4.1)
			if testErr.Code != int32(httpCode) {
				t.Logf("Code mismatch: expected %d, got %d", httpCode, testErr.Code)
				return false
			}
			if testErr.Reason != reason {
				t.Logf("Reason mismatch: expected %s, got %s", reason, testErr.Reason)
				return false
			}
			if testErr.Message != message {
				t.Logf("Message mismatch: expected %s, got %s", message, testErr.Message)
				return false
			}

			// Metadata field access (Requirement 4.1)
			if testErr.Metadata == nil {
				t.Log("Metadata should not be nil after WithMetadata")
				return false
			}
			for i := 0; i < numKeys; i++ {
				key := fmt.Sprintf("key%d", i)
				expectedValue := fmt.Sprintf("value%d", i)
				if testErr.Metadata[key] != expectedValue {
					t.Logf("Metadata mismatch for key %s: expected %s, got %s", key, expectedValue, testErr.Metadata[key])
					return false
				}
			}

			// Error() method should contain all fields (Requirement 4.2)
			errStr := testErr.Error()
			if errStr == "" {
				t.Log("Error() returned empty string")
				return false
			}

			// Error() should contain the code
			codeStr := fmt.Sprintf("code = %d", httpCode)
			if !strings.Contains(errStr, codeStr) {
				t.Logf("Error() should contain code: expected to contain '%s', got '%s'", codeStr, errStr)
				return false
			}

			// Error() should contain the reason
			if !strings.Contains(errStr, reason) {
				t.Logf("Error() should contain reason: expected to contain '%s', got '%s'", reason, errStr)
				return false
			}

			// Error() should contain the message
			if !strings.Contains(errStr, message) {
				t.Logf("Error() should contain message: expected to contain '%s', got '%s'", message, errStr)
				return false
			}

			return true
		},
		&quick.Config{Values: func(v []reflect.Value, r *rand.Rand) {
			v[0] = reflect.ValueOf(errorTestCase{
				code:         r.Intn(600),       // 0-599 -> HTTP codes 100-699
				reason:       randString(r, 20), // Random string up to 20 chars
				message:      randString(r, 50), // Random string up to 50 chars
				metadataKeys: []string{"a", "b", "c", "d", "e"},
			})
		}, MaxCount: 100}, // Minimum 100 iterations
	); err != nil {
		t.Errorf("Property 7 failed: %v", err)
	}
}

// randString generates a random printable ASCII string
func randString(r *rand.Rand, maxLen int) string {
	length := r.Intn(maxLen) + 1
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		result[i] = byte(32 + r.Intn(95)) // printable ASCII: 32-126
	}
	return string(result)
}

// filterPrintableASCII converts a string to printable ASCII characters only
func filterPrintableASCII(s string) string {
	result := make([]byte, 0, len(s))
	for _, b := range []byte(s) {
		if b >= 32 && b <= 126 { // printable ASCII range
			result = append(result, b)
		}
	}
	if len(result) == 0 {
		return "test" // fallback for empty strings
	}
	return string(result)
}

// Feature: backend-server-framework, Property 8: 错误 gRPC 转换
// Property 8: Error gRPC Conversion
// Validates: Requirement 4.3
//
// For any Error instance:
// - GRPCStatus() method should return a valid gRPC Status
// - The gRPC code should correctly correspond to the HTTP error code
func TestProperty8ErrorGRPCCConversion(t *testing.T) {
	f := func(code int) bool {
		// Ensure we test reasonable HTTP codes (including edge cases)
		httpCode := code % 1000
		err := fferr.New(httpCode, "TEST_REASON", "test message")

		// GRPCStatus() should return non-nil
		grpcStatus := err.GRPCStatus()
		if grpcStatus == nil {
			t.Log("GRPCStatus() returned nil")
			return false
		}

		// The gRPC code should match what HTTPToGRPCCode returns
		expectedGRPCode := fferr.HTTPToGRPCCode(httpCode)
		actualGRPCode := grpcStatus.Code()

		if actualGRPCode != expectedGRPCode {
			t.Logf("gRPC code mismatch: HTTP %d -> expected gRPC %v, got %v",
				httpCode, expectedGRPCode, actualGRPCode)
			return false
		}

		// The message should be preserved in gRPC status
		if grpcStatus.Message() != "test message" {
			t.Logf("Message not preserved: expected 'test message', got '%s'", grpcStatus.Message())
			return false
		}

		return true
	}

	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil { // Minimum 100 iterations
		t.Errorf("Property 8 failed: %v", err)
	}
}

// TestErrorIsWithStandardErrors tests errors.Is compatibility
// Validates: Requirements 4.2
func TestErrorIsWithStandardErrors(t *testing.T) {
	err := fferr.ErrNotFound

	// Test with errors.Is
	if !stderrors.Is(err, fferr.ErrNotFound) {
		t.Error("errors.Is should return true for same error instance")
	}

	// Test with different error
	if stderrors.Is(err, fferr.ErrBadRequest) {
		t.Error("errors.Is should return false for different error")
	}
}

// =============================================================================
// Property-Based Tests - Property 9: 错误创建和提取
// =============================================================================

// Feature: backend-server-framework, Property 9: 错误创建和提取
// Property 9: Error Creation and Extraction
// Validates: Requirements 4.4, 4.5, 4.6
//
// For any error input:
// - New() should create Error with correct fields
// - FromError() should correctly extract Error from various error types
// - Code() and Reason() should return correct values
func TestProperty9ErrorCreationAndExtraction(t *testing.T) {
	type errorTestCase struct {
		code        int
		reason      string
		message     string
		isGRPC      bool
		grpcCode    int
		standardErr bool
	}

	if err := quick.Check(
		func(tc errorTestCase) bool {
			// Constrain inputs to valid ranges
			code := tc.code%1000 + 100 // HTTP codes 100-1099
			reason := filterPrintableASCII(tc.reason)
			if reason == "" {
				reason = "TEST_ERROR"
			}
			message := filterPrintableASCII(tc.message)
			if message == "" {
				message = "test message"
			}

			// Test 1: New() creates Error correctly (Requirement 4.4)
			newErr := fferr.New(code, reason, message)
			if newErr == nil {
				t.Log("New() returned nil")
				return false
			}
			if newErr.Code != int32(code) {
				t.Logf("New() Code mismatch: expected %d, got %d", code, newErr.Code)
				return false
			}
			if newErr.Reason != reason {
				t.Logf("New() Reason mismatch: expected %s, got %s", reason, newErr.Reason)
				return false
			}
			if newErr.Message != message {
				t.Logf("New() Message mismatch: expected %s, got %s", message, newErr.Message)
				return false
			}

			// Test 2: FromError extracts from Error type (Requirement 4.5)
			extracted := fferr.FromError(newErr)
			if extracted == nil {
				t.Log("FromError(Error) returned nil")
				return false
			}
			if extracted.Code != newErr.Code {
				t.Logf("FromError Code mismatch: expected %d, got %d", newErr.Code, extracted.Code)
				return false
			}
			if extracted.Reason != newErr.Reason {
				t.Logf("FromError Reason mismatch: expected %s, got %s", newErr.Reason, extracted.Reason)
				return false
			}

			// Test 3: Code() function returns correct value (Requirement 4.6)
			codeFromFunc := fferr.Code(newErr)
			if codeFromFunc != code {
				t.Logf("Code() mismatch: expected %d, got %d", code, codeFromFunc)
				return false
			}

			// Test 4: Reason() function returns correct value (Requirement 4.6)
			reasonFromFunc := fferr.Reason(newErr)
			if reasonFromFunc != reason {
				t.Logf("Reason() mismatch: expected %s, got %s", reason, reasonFromFunc)
				return false
			}

			// Test 5: FromError with standard error
			stdErr := stderrors.New("standard error message")
			extractedStd := fferr.FromError(stdErr)
			if extractedStd == nil {
				t.Log("FromError(standard error) returned nil")
				return false
			}
			// Standard errors should be converted to internal
			if extractedStd.Code != fferr.CodeInternal {
				t.Logf("Standard error should convert to CodeInternal, got %d", extractedStd.Code)
				return false
			}
			if !strings.Contains(extractedStd.Message, "standard error message") {
				t.Logf("Standard error message not preserved: got %s", extractedStd.Message)
				return false
			}

			// Test 6: FromError with nil
			if fferr.FromError(nil) != nil {
				t.Log("FromError(nil) should return nil")
				return false
			}

			// Test 7: Code with nil returns CodeOK
			if fferr.Code(nil) != fferr.CodeOK {
				t.Logf("Code(nil) should return CodeOK, got %d", fferr.Code(nil))
				return false
			}

			// Test 8: Reason with nil returns empty string
			if fferr.Reason(nil) != "" {
				t.Logf("Reason(nil) should return empty string, got %s", fferr.Reason(nil))
				return false
			}

			return true
		},
		&quick.Config{Values: func(v []reflect.Value, r *rand.Rand) {
			v[0] = reflect.ValueOf(errorTestCase{
				code:        r.Intn(600),       // 0-599 -> HTTP codes 100-699
				reason:      randString(r, 20), // Random string up to 20 chars
				message:     randString(r, 50), // Random string up to 50 chars
				isGRPC:      r.Intn(2) == 1,
				grpcCode:    r.Intn(20),
				standardErr: r.Intn(2) == 1,
			})
		}, MaxCount: 100}, // Minimum 100 iterations
	); err != nil {
		t.Errorf("Property 9 failed: %v", err)
	}
}
