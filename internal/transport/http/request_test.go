package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/zhangpeihaoks/firefly/internal/errors"
)

func TestRequest_GetHeader(t *testing.T) {
	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Test-Header", "test-value")
	req.Header.Set("Authorization", "Bearer token123")

	// Create test context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Get header
	reqHelper := GetRequest()
	value := reqHelper.GetHeader(c, "X-Test-Header")
	if value != "test-value" {
		t.Errorf("expected header value 'test-value', got '%s'", value)
	}

	value = reqHelper.GetHeader(c, "Authorization")
	if value != "Bearer token123" {
		t.Errorf("expected header value 'Bearer token123', got '%s'", value)
	}

	// Test non-existent header
	value = reqHelper.GetHeader(c, "Non-Existent")
	if value != "" {
		t.Errorf("expected empty string for non-existent header, got '%s'", value)
	}
}

func TestRequest_GetHeaders(t *testing.T) {
	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Add("X-Test-Header", "value1")
	req.Header.Add("X-Test-Header", "value2")

	// Create test context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Get headers
	reqHelper := GetRequest()
	values := reqHelper.GetHeaders(c, "X-Test-Header")
	if len(values) != 2 {
		t.Errorf("expected 2 header values, got %d", len(values))
	}
	if values[0] != "value1" || values[1] != "value2" {
		t.Errorf("expected header values ['value1', 'value2'], got %v", values)
	}
}

func TestRequest_GetQuery(t *testing.T) {
	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/test?page=1&sort=name&filter=active", nil)

	// Create test context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Get query parameters
	reqHelper := GetRequest()

	page := reqHelper.GetQuery(c, "page")
	if page != "1" {
		t.Errorf("expected query parameter 'page' = '1', got '%s'", page)
	}

	sort := reqHelper.GetQuery(c, "sort")
	if sort != "name" {
		t.Errorf("expected query parameter 'sort' = 'name', got '%s'", sort)
	}

	filter := reqHelper.GetQuery(c, "filter")
	if filter != "active" {
		t.Errorf("expected query parameter 'filter' = 'active', got '%s'", filter)
	}

	// Test non-existent parameter
	nonExistent := reqHelper.GetQuery(c, "non-existent")
	if nonExistent != "" {
		t.Errorf("expected empty string for non-existent parameter, got '%s'", nonExistent)
	}
}

func TestRequest_GetQueryDefault(t *testing.T) {
	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/test?page=2", nil)

	// Create test context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Get query parameters with defaults
	reqHelper := GetRequest()

	// Existing parameter
	page := reqHelper.GetQueryDefault(c, "page", "1")
	if page != "2" {
		t.Errorf("expected '2' for existing parameter, got '%s'", page)
	}

	// Non-existent parameter with default
	limit := reqHelper.GetQueryDefault(c, "limit", "10")
	if limit != "10" {
		t.Errorf("expected default '10' for non-existent parameter, got '%s'", limit)
	}
}

func TestRequest_GetPathParam(t *testing.T) {
	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/users/123/posts/456", nil)

	// Create test context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Set up path parameters (simulating Gin's behavior)
	c.Params = []gin.Param{
		{Key: "userID", Value: "123"},
		{Key: "postID", Value: "456"},
	}

	// Get path parameters
	reqHelper := GetRequest()

	userID := reqHelper.GetPathParam(c, "userID")
	if userID != "123" {
		t.Errorf("expected path parameter 'userID' = '123', got '%s'", userID)
	}

	postID := reqHelper.GetPathParam(c, "postID")
	if postID != "456" {
		t.Errorf("expected path parameter 'postID' = '456', got '%s'", postID)
	}

	// Test non-existent parameter
	nonExistent := reqHelper.GetPathParam(c, "non-existent")
	if nonExistent != "" {
		t.Errorf("expected empty string for non-existent parameter, got '%s'", nonExistent)
	}
}

func TestRequest_GetClientIP(t *testing.T) {
	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:8080"

	// Create test context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Get client IP
	reqHelper := GetRequest()
	ip := reqHelper.GetClientIP(c)

	// Gin should extract the IP from RemoteAddr
	if ip != "192.168.1.1" {
		t.Errorf("expected client IP '192.168.1.1', got '%s'", ip)
	}
}

func TestRequest_GetMethod(t *testing.T) {
	// Create test request
	req := httptest.NewRequest(http.MethodPost, "/test", nil)

	// Create test context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Get method
	reqHelper := GetRequest()
	method := reqHelper.GetMethod(c)

	if method != http.MethodPost {
		t.Errorf("expected method 'POST', got '%s'", method)
	}
}

func TestRequest_GetContentType(t *testing.T) {
	// Create test request
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("Content-Type", "application/json")

	// Create test context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Get content type
	reqHelper := GetRequest()
	contentType := reqHelper.GetContentType(c)

	if contentType != "application/json" {
		t.Errorf("expected content type 'application/json', got '%s'", contentType)
	}
}

func TestRequest_ValidateRequiredQuery(t *testing.T) {
	// Create test request with some parameters
	req := httptest.NewRequest(http.MethodGet, "/test?page=1&sort=name", nil)

	// Create test context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Validate required parameters
	reqHelper := GetRequest()

	// Should succeed when all parameters are present
	err := reqHelper.ValidateRequiredQuery(c, "page", "sort")
	if err != nil {
		t.Errorf("unexpected error when all parameters are present: %v", err)
	}

	// Should fail when a parameter is missing
	err = reqHelper.ValidateRequiredQuery(c, "page", "sort", "filter")
	if err == nil {
		t.Error("expected error when parameter is missing")
	} else if fwErr := errors.FromError(err); fwErr.Code != int32(errors.CodeBadRequest) {
		t.Errorf("expected error code %d, got %d", errors.CodeBadRequest, fwErr.Code)
	}
}

func TestRequest_ValidateRequiredPath(t *testing.T) {
	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/users/123", nil)

	// Create test context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Set up path parameters
	c.Params = []gin.Param{
		{Key: "id", Value: "123"},
	}

	// Validate required parameters
	reqHelper := GetRequest()

	// Should succeed when parameter is present
	err := reqHelper.ValidateRequiredPath(c, "id")
	if err != nil {
		t.Errorf("unexpected error when parameter is present: %v", err)
	}

	// Should fail when parameter is missing
	err = reqHelper.ValidateRequiredPath(c, "id", "name")
	if err == nil {
		t.Error("expected error when parameter is missing")
	} else if fwErr := errors.FromError(err); fwErr.Code != int32(errors.CodeBadRequest) {
		t.Errorf("expected error code %d, got %d", errors.CodeBadRequest, fwErr.Code)
	}
}

func TestRequest_ValidateRequiredHeader(t *testing.T) {
	// Create test request with headers
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer token123")
	req.Header.Set("X-Request-ID", "req-123")

	// Create test context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Validate required headers
	reqHelper := GetRequest()

	// Should succeed when all headers are present
	err := reqHelper.ValidateRequiredHeader(c, "Authorization", "X-Request-ID")
	if err != nil {
		t.Errorf("unexpected error when all headers are present: %v", err)
	}

	// Should fail when a header is missing
	err = reqHelper.ValidateRequiredHeader(c, "Authorization", "X-Request-ID", "X-Custom-Header")
	if err == nil {
		t.Error("expected error when header is missing")
	} else if fwErr := errors.FromError(err); fwErr.Code != int32(errors.CodeBadRequest) {
		t.Errorf("expected error code %d, got %d", errors.CodeBadRequest, fwErr.Code)
	}
}

func TestWrapHandler(t *testing.T) {
	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	// Create test context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Test successful handler
	successHandler := func(c *gin.Context) (any, error) {
		return map[string]interface{}{"message": "success"}, nil
	}

	handler := WrapHandler(successHandler)
	handler(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Test error handler
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request = req

	errorHandler := func(c *gin.Context) (any, error) {
		return nil, errors.New(errors.CodeBadRequest, "BAD_REQUEST", "Invalid request")
	}

	handler2 := WrapHandler(errorHandler)
	handler2(c2)

	if w2.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w2.Code)
	}
}

func TestWrapHandlerWithStatus(t *testing.T) {
	// Create test request
	req := httptest.NewRequest(http.MethodPost, "/test", nil)

	// Create test context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Test created status
	createHandler := func(c *gin.Context) (any, error) {
		return map[string]interface{}{"id": 1, "name": "test"}, nil
	}

	handler := WrapHandlerWithStatus(createHandler, http.StatusCreated)
	handler(c)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}

	// Test no content status
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request = req

	deleteHandler := func(c *gin.Context) (any, error) {
		return nil, nil
	}

	handler2 := WrapHandlerWithStatus(deleteHandler, http.StatusNoContent)
	handler2(c2)

	if w2.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, w2.Code)
	}
}
