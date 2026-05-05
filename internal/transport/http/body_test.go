package http

import (
	"bytes"
	"context"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/zhangpeihaoks/firefly/internal/errors"
)

func TestBodyParser_ParseJSON(t *testing.T) {
	// Create test data
	type TestStruct struct {
		Name  string `json:"name"`
		Age   int    `json:"age"`
		Email string `json:"email"`
	}

	tests := []struct {
		name          string
		contentType   string
		body          string
		expectedError bool
		errorCode     int
	}{
		{
			name:        "valid JSON",
			contentType: "application/json",
			body:        `{"name":"John","age":30,"email":"john@example.com"}`,
		},
		{
			name:          "invalid JSON",
			contentType:   "application/json",
			body:          `{"name":"John","age":30,"email":"john@example.com"`,
			expectedError: true,
			errorCode:     errors.CodeBadRequest,
		},
		{
			name:          "wrong content type",
			contentType:   "text/plain",
			body:          `{"name":"John","age":30,"email":"john@example.com"}`,
			expectedError: true,
			errorCode:     errors.CodeBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test request
			req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", tt.contentType)

			// Create test context
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			// Parse JSON
			parser := GetBodyParser()
			var result TestStruct
			err := parser.ParseJSON(c, &result)

			// Check results
			if tt.expectedError {
				if err == nil {
					t.Error("expected error but got none")
				} else if fwErr := errors.FromError(err); fwErr.Code != int32(tt.errorCode) {
					t.Errorf("expected error code %d, got %d", tt.errorCode, fwErr.Code)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result.Name != "John" || result.Age != 30 || result.Email != "john@example.com" {
					t.Errorf("unexpected result: %+v", result)
				}
			}
		})
	}
}

func TestBodyParser_ParseXML(t *testing.T) {
	// Create test data
	type TestStruct struct {
		XMLName xml.Name `xml:"person"`
		Name    string   `xml:"name"`
		Age     int      `xml:"age"`
		Email   string   `xml:"email"`
	}

	tests := []struct {
		name          string
		contentType   string
		body          string
		expectedError bool
		errorCode     int
	}{
		{
			name:        "valid XML",
			contentType: "application/xml",
			body:        `<person><name>John</name><age>30</age><email>john@example.com</email></person>`,
		},
		{
			name:          "invalid XML",
			contentType:   "application/xml",
			body:          `<person><name>John</name><age>30</age><email>john@example.com</email>`,
			expectedError: true,
			errorCode:     errors.CodeBadRequest,
		},
		{
			name:          "wrong content type",
			contentType:   "text/plain",
			body:          `<person><name>John</name><age>30</age><email>john@example.com</email></person>`,
			expectedError: true,
			errorCode:     errors.CodeBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test request
			req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", tt.contentType)

			// Create test context
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			// Parse XML
			parser := GetBodyParser()
			var result TestStruct
			err := parser.ParseXML(c, &result)

			// Check results
			if tt.expectedError {
				if err == nil {
					t.Error("expected error but got none")
				} else if fwErr := errors.FromError(err); fwErr.Code != int32(tt.errorCode) {
					t.Errorf("expected error code %d, got %d", tt.errorCode, fwErr.Code)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result.Name != "John" || result.Age != 30 || result.Email != "john@example.com" {
					t.Errorf("unexpected result: %+v", result)
				}
			}
		})
	}
}

func TestBodyParser_ParseBody(t *testing.T) {
	// Create test data
	type TestStruct struct {
		Name  string `json:"name" xml:"name"`
		Age   int    `json:"age" xml:"age"`
		Email string `json:"email" xml:"email"`
	}

	tests := []struct {
		name          string
		contentType   string
		body          string
		expectedError bool
	}{
		{
			name:        "JSON body",
			contentType: "application/json",
			body:        `{"name":"John","age":30,"email":"john@example.com"}`,
		},
		{
			name:        "XML body",
			contentType: "application/xml",
			body:        `<TestStruct><name>John</name><age>30</age><email>john@example.com</email></TestStruct>`,
		},
		{
			name:          "unsupported content type",
			contentType:   "text/csv",
			body:          "name,age,email\nJohn,30,john@example.com",
			expectedError: true,
		},
		{
			name:        "default to JSON when no content type",
			contentType: "",
			body:        `{"name":"John","age":30,"email":"john@example.com"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test request
			req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(tt.body))
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			// Create test context
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			// Parse body
			parser := GetBodyParser()
			var result TestStruct
			err := parser.ParseBody(c, &result)

			// Check results
			if tt.expectedError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result.Name != "John" || result.Age != 30 || result.Email != "john@example.com" {
					t.Errorf("unexpected result: %+v", result)
				}
			}
		})
	}
}

func TestParseJSONFromContext(t *testing.T) {
	// Create test data
	type TestStruct struct {
		Name string `json:"name"`
	}

	// Create test request
	req := httptest.NewRequest(http.MethodPost, "/test",
		bytes.NewBufferString(`{"name":"Test"}`))
	req.Header.Set("Content-Type", "application/json")

	// Create test context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Create context with gin context
	ctx := context.WithValue(context.Background(), gin.ContextKey, c)

	// Parse JSON from context
	var result TestStruct
	err := ParseJSONFromContext(ctx, &result)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result.Name != "Test" {
		t.Errorf("expected name 'Test', got %s", result.Name)
	}
}

func TestParseBodyFromContext(t *testing.T) {
	// Create test data
	type TestStruct struct {
		Name string `json:"name"`
	}

	// Create test request
	req := httptest.NewRequest(http.MethodPost, "/test",
		bytes.NewBufferString(`{"name":"Test"}`))
	req.Header.Set("Content-Type", "application/json")

	// Create test context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Create context with gin context
	ctx := context.WithValue(context.Background(), gin.ContextKey, c)

	// Parse body from context
	var result TestStruct
	err := ParseBodyFromContext(ctx, &result)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result.Name != "Test" {
		t.Errorf("expected name 'Test', got %s", result.Name)
	}
}
