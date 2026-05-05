// Package database provides database connection management for the Firefly framework.
// This file implements database-specific errors.
package database

import (
	"fmt"
)

// Error represents a database error.
type Error struct {
	// Code is the error code.
	Code string `json:"code"`

	// Message is the error message.
	Message string `json:"message"`

	// Driver is the database driver that produced the error.
	Driver string `json:"driver,omitempty"`

	// Query is the SQL query that caused the error (if applicable).
	Query string `json:"query,omitempty"`

	// Err is the underlying error.
	Err error `json:"-"`
}

// Error implements the error interface.
func (e *Error) Error() string {
	var parts []string
	parts = append(parts, fmt.Sprintf("database error: [%s]", e.Code))

	if e.Driver != "" {
		parts = append(parts, fmt.Sprintf("driver=%s", e.Driver))
	}
	if e.Query != "" {
		parts = append(parts, fmt.Sprintf("query=%s", e.Query))
	}

	parts = append(parts, e.Message)

	if e.Err != nil {
		parts = append(parts, fmt.Sprintf(": %v", e.Err))
	}

	return fmt.Sprintf("%s", parts)
}

// Unwrap returns the underlying error.
func (e *Error) Unwrap() error {
	return e.Err
}

// NewError creates a new database error.
func NewError(code, message string, err error) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// NewConfigError creates a configuration error.
func NewConfigError(message string) *Error {
	return &Error{
		Code:    "CONFIG_ERROR",
		Message: message,
	}
}

// NewConnectionError creates a connection error.
func NewConnectionError(driver, message string, err error) *Error {
	return &Error{
		Code:    "CONNECTION_ERROR",
		Message: message,
		Driver:  driver,
		Err:     err,
	}
}

// NewQueryError creates a query error.
func NewQueryError(driver, query, message string, err error) *Error {
	return &Error{
		Code:    "QUERY_ERROR",
		Message: message,
		Driver:  driver,
		Query:   query,
		Err:     err,
	}
}

// Error codes.
const (
	// ErrCodeConfig indicates a configuration error.
	ErrCodeConfig = "CONFIG_ERROR"

	// ErrCodeConnection indicates a connection error.
	ErrCodeConnection = "CONNECTION_ERROR"

	// ErrCodeQuery indicates a query error.
	ErrCodeQuery = "QUERY_ERROR"

	// ErrCodeTimeout indicates a timeout error.
	ErrCodeTimeout = "TIMEOUT_ERROR"

	// ErrCodePoolExhausted indicates the connection pool is exhausted.
	ErrCodePoolExhausted = "POOL_EXHAUSTED"

	// ErrCodeNotConnected indicates the database is not connected.
	ErrCodeNotConnected = "NOT_CONNECTED"

	// ErrCodeUnsupportedDriver indicates an unsupported driver.
	ErrCodeUnsupportedDriver = "UNSUPPORTED_DRIVER"
)

// IsConfigError returns true if the error is a configuration error.
func IsConfigError(err error) bool {
	if e, ok := err.(*Error); ok {
		return e.Code == ErrCodeConfig
	}
	return false
}

// IsConnectionError returns true if the error is a connection error.
func IsConnectionError(err error) bool {
	if e, ok := err.(*Error); ok {
		return e.Code == ErrCodeConnection
	}
	return false
}

// IsQueryError returns true if the error is a query error.
func IsQueryError(err error) bool {
	if e, ok := err.(*Error); ok {
		return e.Code == ErrCodeQuery
	}
	return false
}

// IsTimeoutError returns true if the error is a timeout error.
func IsTimeoutError(err error) bool {
	if e, ok := err.(*Error); ok {
		return e.Code == ErrCodeTimeout
	}
	return false
}

// IsPoolExhaustedError returns true if the error is a pool exhausted error.
func IsPoolExhaustedError(err error) bool {
	if e, ok := err.(*Error); ok {
		return e.Code == ErrCodePoolExhausted
	}
	return false
}

// IsNotConnectedError returns true if the error is a not connected error.
func IsNotConnectedError(err error) bool {
	if e, ok := err.(*Error); ok {
		return e.Code == ErrCodeNotConnected
	}
	return false
}

// IsUnsupportedDriverError returns true if the error is an unsupported driver error.
func IsUnsupportedDriverError(err error) bool {
	if e, ok := err.(*Error); ok {
		return e.Code == ErrCodeUnsupportedDriver
	}
	return false
}
