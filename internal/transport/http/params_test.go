package http

import (
	"context"
	"testing"

	"github.com/zhangpeihaoks/firefly/internal/transport"
)

func TestGetPathParam(t *testing.T) {
	// Create a mock transporter with path parameters
	transporter := &transporter{
		pathParams: map[string]string{
			"id":   "123",
			"name": "test",
		},
	}

	// Create context with transporter
	ctx := NewContext(context.Background(), transporter)

	// Test existing parameter
	value, ok := GetPathParam(ctx, "id")
	if !ok {
		t.Error("expected to find path parameter 'id'")
	}
	if value != "123" {
		t.Errorf("expected value '123', got %s", value)
	}

	// Test non-existing parameter
	_, ok = GetPathParam(ctx, "nonexistent")
	if ok {
		t.Error("expected not to find non-existent parameter")
	}

	// Test integer conversion
	intValue, ok := GetPathParamInt(ctx, "id")
	if !ok {
		t.Error("expected to parse integer parameter")
	}
	if intValue != 123 {
		t.Errorf("expected integer 123, got %d", intValue)
	}

	// Test invalid integer conversion
	_, ok = GetPathParamInt(ctx, "name")
	if ok {
		t.Error("expected invalid integer conversion to fail")
	}

	// Test int64 conversion
	int64Value, ok := GetPathParamInt64(ctx, "id")
	if !ok {
		t.Error("expected to parse int64 parameter")
	}
	if int64Value != 123 {
		t.Errorf("expected int64 123, got %d", int64Value)
	}
}

func TestGetQueryParam(t *testing.T) {
	// Create a mock transporter with query parameters
	transporter := &transporter{
		queryParams: map[string][]string{
			"page":  {"1"},
			"sort":  {"name", "date"},
			"empty": {},
		},
	}

	// Create context with transporter
	ctx := NewContext(context.Background(), transporter)

	// Test existing single-value parameter
	value, ok := GetQueryParam(ctx, "page")
	if !ok {
		t.Error("expected to find query parameter 'page'")
	}
	if value != "1" {
		t.Errorf("expected value '1', got %s", value)
	}

	// Test existing multi-value parameter (returns first)
	value, ok = GetQueryParam(ctx, "sort")
	if !ok {
		t.Error("expected to find query parameter 'sort'")
	}
	if value != "name" {
		t.Errorf("expected first value 'name', got %s", value)
	}

	// Test non-existing parameter
	_, ok = GetQueryParam(ctx, "nonexistent")
	if ok {
		t.Error("expected not to find non-existent parameter")
	}

	// Test empty parameter
	_, ok = GetQueryParam(ctx, "empty")
	if ok {
		t.Error("expected empty parameter to not be found")
	}

	// Test integer conversion
	intValue, ok := GetQueryParamInt(ctx, "page")
	if !ok {
		t.Error("expected to parse integer parameter")
	}
	if intValue != 1 {
		t.Errorf("expected integer 1, got %d", intValue)
	}

	// Test int64 conversion
	int64Value, ok := GetQueryParamInt64(ctx, "page")
	if !ok {
		t.Error("expected to parse int64 parameter")
	}
	if int64Value != 1 {
		t.Errorf("expected int64 1, got %d", int64Value)
	}

	// Test getting all values
	values, ok := GetQueryParamAll(ctx, "sort")
	if !ok {
		t.Error("expected to get all values for 'sort'")
	}
	if len(values) != 2 {
		t.Errorf("expected 2 values, got %d", len(values))
	}
	if values[0] != "name" || values[1] != "date" {
		t.Errorf("expected values ['name', 'date'], got %v", values)
	}

	// Test integer array conversion
	intValues, ok := GetQueryParamInts(ctx, "sort")
	if ok {
		t.Error("expected invalid integer conversion to fail")
	}
	if intValues != nil {
		t.Error("expected nil for failed conversion")
	}
}

func TestMustGetPathParam(t *testing.T) {
	// Create a mock transporter with path parameters
	transporter := &transporter{
		pathParams: map[string]string{
			"id": "456",
		},
	}

	// Create context with transporter
	ctx := NewContext(context.Background(), transporter)

	// Test successful retrieval
	defer func() {
		if r := recover(); r != nil {
			t.Error("unexpected panic:", r)
		}
	}()
	value := MustGetPathParam(ctx, "id")
	if value != "456" {
		t.Errorf("expected value '456', got %s", value)
	}

	// Test panic for non-existent parameter
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for non-existent parameter")
			}
		}()
		MustGetPathParam(ctx, "nonexistent")
	}()
}

func TestPathParamExists(t *testing.T) {
	// Create a mock transporter with path parameters
	transporter := &transporter{
		pathParams: map[string]string{
			"id": "789",
		},
	}

	// Create context with transporter
	ctx := NewContext(context.Background(), transporter)

	// Test existing parameter
	if !PathParamExists(ctx, "id") {
		t.Error("expected parameter 'id' to exist")
	}

	// Test non-existing parameter
	if PathParamExists(ctx, "nonexistent") {
		t.Error("expected parameter 'nonexistent' to not exist")
	}
}

func TestQueryParamExists(t *testing.T) {
	// Create a mock transporter with query parameters
	transporter := &transporter{
		queryParams: map[string][]string{
			"filter": {"active"},
		},
	}

	// Create context with transporter
	ctx := NewContext(context.Background(), transporter)

	// Test existing parameter
	if !QueryParamExists(ctx, "filter") {
		t.Error("expected parameter 'filter' to exist")
	}

	// Test non-existing parameter
	if QueryParamExists(ctx, "nonexistent") {
		t.Error("expected parameter 'nonexistent' to not exist")
	}
}

func TestTransporterInterface(t *testing.T) {
	// Verify that transporter implements the updated Transporter interface
	var _ transport.Transporter = &transporter{}
}
