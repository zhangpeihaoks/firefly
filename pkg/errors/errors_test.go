package errors

import (
	"errors"
	"fmt"
	"testing"
)

// TestNew verifies the status code, reason and message survive construction.
func TestNew(t *testing.T) {
	err := New(CodeServiceUnavailable, "NOT_READY", "webhook url missing")
	if err.Code != CodeServiceUnavailable {
		t.Fatalf("Code = %d, want %d", err.Code, CodeServiceUnavailable)
	}
	if err.Reason != "NOT_READY" {
		t.Fatalf("Reason = %q, want %q", err.Reason, "NOT_READY")
	}
	if err.Message != "webhook url missing" {
		t.Fatalf("Message = %q, want %q", err.Message, "webhook url missing")
	}
}

// TestWrapKeepsCause verifies the wrapped error stays reachable through the
// standard library's errors.Is.
func TestWrapKeepsCause(t *testing.T) {
	root := fmt.Errorf("dial failed")
	err := Wrap(root, CodeInternal, "UPSTREAM", "cannot reach feishu")

	if !errors.Is(err, root) {
		t.Fatal("wrapped error does not unwrap to the root cause")
	}

	var target *Error
	if !errors.As(err, &target) {
		t.Fatal("wrapped error is not an *Error")
	}
	if target.Code != CodeInternal {
		t.Fatalf("Code = %d, want %d", target.Code, CodeInternal)
	}
}

// TestNewf verifies the formatted-message constructor.
func TestNewf(t *testing.T) {
	err := Newf(CodeBadRequest, "INVALID_PAYLOAD", "field %q is required", "alerts")
	if err.Message != `field "alerts" is required` {
		t.Fatalf("Message = %q, want %q", err.Message, `field "alerts" is required`)
	}
}
