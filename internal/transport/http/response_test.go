package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/zhangpeihaoks/firefly/internal/errors"
	"github.com/zhangpeihaoks/firefly/pkg/response"
)

func TestResponse_Success(t *testing.T) {
	// Create test context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Test data
	testData := map[string]interface{}{
		"id":   1,
		"name": "Test",
	}

	// Send success response
	resp := GetResponse()
	resp.Success(c, testData)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Parse response body
	// var result map[string]interface{}
	// c.Writer.WriteHeaderNow()

	// In a real test, we would parse the JSON response
	// For now, just verify the status code
}

func TestResponse_Error(t *testing.T) {
	// Create test context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Send error response
	resp := GetResponse()
	resp.Error(c, http.StatusBadRequest, "Bad request")

	// Check response
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestResponse_HandleError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedStatus int
	}{
		{
			name:           "bad request error",
			err:            errors.New(errors.CodeBadRequest, "BAD_REQUEST", "Invalid input"),
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "not found error",
			err:            errors.New(errors.CodeNotFound, "NOT_FOUND", "Resource not found"),
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "internal error",
			err:            errors.New(errors.CodeInternal, "INTERNAL_ERROR", "Internal server error"),
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "generic error",
			err:            errors.New(418, "TEAPOT", "I'm a teapot"),
			expectedStatus: 418, // Custom status code
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test context with a proper HTTP request
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			// Create a request with Accept header to avoid nil pointer dereference
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Accept", "application/json")
			c.Request = req

			// Handle error
			resp := GetResponse()
			resp.HandleError(c, tt.err)

			// Check response
			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestResponse_HandleSuccess(t *testing.T) {
	// Create test context with a proper HTTP request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Accept", "application/json")
	c.Request = req

	// Test data
	testData := map[string]interface{}{
		"id":   1,
		"name": "Test",
	}

	// Handle success with data
	resp := GetResponse()
	resp.HandleSuccess(c, testData, nil)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Test with error
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.Header.Set("Accept", "application/json")
	c2.Request = req2

	resp.HandleSuccess(c2, nil, errors.New(errors.CodeBadRequest, "BAD_REQUEST", "Error"))

	if w2.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w2.Code)
	}
}

func TestResponse_Created(t *testing.T) {
	// Create test context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Test data
	testData := map[string]interface{}{
		"id":   1,
		"name": "Test",
	}

	// Send created response
	resp := GetResponse()
	resp.Created(c, testData)

	// Check response
	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}
}

func TestResponse_NoContent(t *testing.T) {
	// Create test context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Send no content response
	resp := GetResponse()
	resp.NoContent(c)

	// Check response
	if w.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, w.Code)
	}
}

func TestResponse_ErrorHelpers(t *testing.T) {
	tests := []struct {
		name           string
		callFunc       func(*Response, *gin.Context, string)
		expectedStatus int
	}{
		{
			name: "BadRequest",
			callFunc: func(r *Response, c *gin.Context, msg string) {
				r.BadRequest(c, msg)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Unauthorized",
			callFunc: func(r *Response, c *gin.Context, msg string) {
				r.Unauthorized(c, msg)
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Forbidden",
			callFunc: func(r *Response, c *gin.Context, msg string) {
				r.Forbidden(c, msg)
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "NotFound",
			callFunc: func(r *Response, c *gin.Context, msg string) {
				r.NotFound(c, msg)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "InternalServerError",
			callFunc: func(r *Response, c *gin.Context, msg string) {
				r.InternalServerError(c, msg)
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test context
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// Call error helper
			resp := GetResponse()
			tt.callFunc(resp, c, "test message")

			// Check response
			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestJSONResponse(t *testing.T) {
	// Create test context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Create response
	resp := response.Success(map[string]interface{}{"test": "data"})

	// Send JSON response
	JSONResponse(c, http.StatusOK, resp)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestPageResponse(t *testing.T) {
	// Create test context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Create page response
	resp := response.SuccessWithPage([]interface{}{"item1", "item2"}, 1, 10, 100)

	// Send page response
	PageResponse(c, http.StatusOK, resp)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestResponse_SuccessWithMessage(t *testing.T) {
	// Create test context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Test data
	testData := map[string]interface{}{
		"id":   1,
		"name": "Test",
	}

	// Send success with message
	resp := GetResponse()
	resp.SuccessWithMessage(c, "Custom message", testData)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestResponse_SuccessWithPage(t *testing.T) {
	// Create test context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Test data
	testData := []interface{}{
		map[string]interface{}{"id": 1, "name": "Item 1"},
		map[string]interface{}{"id": 2, "name": "Item 2"},
	}

	// Send success with page
	resp := GetResponse()
	resp.SuccessWithPage(c, testData, 1, 10, 100)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestResponse_ErrorWithData(t *testing.T) {
	// Create test context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Test data
	testData := map[string]interface{}{
		"field":  "email",
		"reason": "invalid format",
	}

	// Send error with data
	resp := GetResponse()
	resp.ErrorWithData(c, http.StatusBadRequest, "Validation failed", testData)

	// Check response
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}
