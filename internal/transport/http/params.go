// Package http provides HTTP server implementation for the Firefly framework.
package http

import (
	"context"
	"strconv"
)

// GetPathParam retrieves a path parameter from the context.
// Returns the value and true if the parameter exists, empty string and false otherwise.
func GetPathParam(ctx context.Context, key string) (string, bool) {
	t, ok := FromContext(ctx)
	if !ok {
		return "", false
	}
	value := t.PathParams()[key]
	return value, value != ""
}

// GetPathParamInt retrieves a path parameter as an integer from the context.
// Returns the value and true if the parameter exists and can be parsed, 0 and false otherwise.
func GetPathParamInt(ctx context.Context, key string) (int, bool) {
	value, ok := GetPathParam(ctx, key)
	if !ok {
		return 0, false
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return 0, false
	}
	return intValue, true
}

// GetPathParamInt64 retrieves a path parameter as an int64 from the context.
// Returns the value and true if the parameter exists and can be parsed, 0 and false otherwise.
func GetPathParamInt64(ctx context.Context, key string) (int64, bool) {
	value, ok := GetPathParam(ctx, key)
	if !ok {
		return 0, false
	}
	intValue, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, false
	}
	return intValue, true
}

// GetQueryParam retrieves a query parameter from the context.
// Returns the first value and true if the parameter exists, empty string and false otherwise.
func GetQueryParam(ctx context.Context, key string) (string, bool) {
	t, ok := FromContext(ctx)
	if !ok {
		return "", false
	}
	values := t.QueryParams()[key]
	if len(values) == 0 {
		return "", false
	}
	return values[0], true
}

// GetQueryParamInt retrieves a query parameter as an integer from the context.
// Returns the value and true if the parameter exists and can be parsed, 0 and false otherwise.
func GetQueryParamInt(ctx context.Context, key string) (int, bool) {
	value, ok := GetQueryParam(ctx, key)
	if !ok {
		return 0, false
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return 0, false
	}
	return intValue, true
}

// GetQueryParamInt64 retrieves a query parameter as an int64 from the context.
// Returns the value and true if the parameter exists and can be parsed, 0 and false otherwise.
func GetQueryParamInt64(ctx context.Context, key string) (int64, bool) {
	value, ok := GetQueryParam(ctx, key)
	if !ok {
		return 0, false
	}
	intValue, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, false
	}
	return intValue, true
}

// GetQueryParamAll retrieves all values of a query parameter from the context.
// Returns the values slice and true if the parameter exists, nil and false otherwise.
func GetQueryParamAll(ctx context.Context, key string) ([]string, bool) {
	t, ok := FromContext(ctx)
	if !ok {
		return nil, false
	}
	values := t.QueryParams()[key]
	if len(values) == 0 {
		return nil, false
	}
	return values, true
}

// GetQueryParamInts retrieves all values of a query parameter as integers from the context.
// Returns the values slice and true if all values can be parsed, nil and false otherwise.
func GetQueryParamInts(ctx context.Context, key string) ([]int, bool) {
	values, ok := GetQueryParamAll(ctx, key)
	if !ok {
		return nil, false
	}
	intValues := make([]int, len(values))
	for i, value := range values {
		intValue, err := strconv.Atoi(value)
		if err != nil {
			return nil, false
		}
		intValues[i] = intValue
	}
	return intValues, true
}

// MustGetPathParam retrieves a path parameter from the context, panicking if not found.
func MustGetPathParam(ctx context.Context, key string) string {
	value, ok := GetPathParam(ctx, key)
	if !ok {
		panic("path parameter not found: " + key)
	}
	return value
}

// MustGetPathParamInt retrieves a path parameter as an integer from the context, panicking if not found or invalid.
func MustGetPathParamInt(ctx context.Context, key string) int {
	value, ok := GetPathParamInt(ctx, key)
	if !ok {
		panic("path parameter not found or invalid: " + key)
	}
	return value
}

// MustGetPathParamInt64 retrieves a path parameter as an int64 from the context, panicking if not found or invalid.
func MustGetPathParamInt64(ctx context.Context, key string) int64 {
	value, ok := GetPathParamInt64(ctx, key)
	if !ok {
		panic("path parameter not found or invalid: " + key)
	}
	return value
}

// PathParamExists checks if a path parameter exists in the context.
func PathParamExists(ctx context.Context, key string) bool {
	_, ok := GetPathParam(ctx, key)
	return ok
}

// QueryParamExists checks if a query parameter exists in the context.
func QueryParamExists(ctx context.Context, key string) bool {
	_, ok := GetQueryParam(ctx, key)
	return ok
}
