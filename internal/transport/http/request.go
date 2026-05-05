// Package http provides HTTP server implementation for the Firefly framework.
package http

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/zhangpeihaoks/firefly/internal/errors"
)

// Request provides helper functions for HTTP request handling.
type Request struct{}

// GetRequest returns a Request instance.
func GetRequest() *Request {
	return &Request{}
}

// GetHeader retrieves a request header value.
func (r *Request) GetHeader(c *gin.Context, key string) string {
	return c.Request.Header.Get(key)
}

// GetHeaders retrieves all values for a request header.
func (r *Request) GetHeaders(c *gin.Context, key string) []string {
	return c.Request.Header.Values(key)
}

// GetQuery retrieves a query parameter value.
func (r *Request) GetQuery(c *gin.Context, key string) string {
	return c.Query(key)
}

// GetQueryDefault retrieves a query parameter value with default.
func (r *Request) GetQueryDefault(c *gin.Context, key, defaultValue string) string {
	value := c.Query(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// GetQueryInt retrieves a query parameter as integer.
func (r *Request) GetQueryInt(c *gin.Context, key string) (int, error) {
	value, ok := GetQueryParamInt(c.Request.Context(), key)
	if !ok {
		return 0, errors.New(errors.CodeBadRequest, "MISSING_PARAMETER", "missing query parameter: "+key)
	}
	return value, nil
}

// GetQueryInt64 retrieves a query parameter as int64.
func (r *Request) GetQueryInt64(c *gin.Context, key string) (int64, error) {
	value, ok := GetQueryParamInt64(c.Request.Context(), key)
	if !ok {
		return 0, errors.New(errors.CodeBadRequest, "MISSING_PARAMETER", "missing query parameter: "+key)
	}
	return value, nil
}

// GetQueryBool retrieves a query parameter as boolean.
func (r *Request) GetQueryBool(c *gin.Context, key string) (bool, error) {
	value := c.Query(key)
	if value == "" {
		return false, nil
	}
	return value == "true" || value == "1" || value == "yes", nil
}

// GetQueryArray retrieves all values for a query parameter.
func (r *Request) GetQueryArray(c *gin.Context, key string) []string {
	return c.QueryArray(key)
}

// GetPathParam retrieves a path parameter value.
func (r *Request) GetPathParam(c *gin.Context, key string) string {
	return c.Param(key)
}

// GetPathParamInt retrieves a path parameter as integer.
func (r *Request) GetPathParamInt(c *gin.Context, key string) (int, error) {
	value := c.Param(key)
	if value == "" {
		return 0, errors.New(errors.CodeBadRequest, "MISSING_PARAMETER", "missing path parameter: "+key)
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return 0, errors.Newf(errors.CodeBadRequest, "INVALID_PARAMETER", "invalid integer for parameter %s: %v", key, err)
	}
	return intValue, nil
}

// GetPathParamInt64 retrieves a path parameter as int64.
func (r *Request) GetPathParamInt64(c *gin.Context, key string) (int64, error) {
	value := c.Param(key)
	if value == "" {
		return 0, errors.New(errors.CodeBadRequest, "MISSING_PARAMETER", "missing path parameter: "+key)
	}
	intValue, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, errors.Newf(errors.CodeBadRequest, "INVALID_PARAMETER", "invalid int64 for parameter %s: %v", key, err)
	}
	return intValue, nil
}

// GetClientIP retrieves the client IP address.
func (r *Request) GetClientIP(c *gin.Context) string {
	return c.ClientIP()
}

// GetMethod retrieves the HTTP method.
func (r *Request) GetMethod(c *gin.Context) string {
	return c.Request.Method
}

// GetURL retrieves the request URL.
func (r *Request) GetURL(c *gin.Context) string {
	return c.Request.URL.String()
}

// GetContentType retrieves the Content-Type header.
func (r *Request) GetContentType(c *gin.Context) string {
	return c.ContentType()
}

// GetUserAgent retrieves the User-Agent header.
func (r *Request) GetUserAgent(c *gin.Context) string {
	return c.Request.UserAgent()
}

// GetReferer retrieves the Referer header.
func (r *Request) GetReferer(c *gin.Context) string {
	return c.Request.Referer()
}

// GetBody reads and returns the request body.
func (r *Request) GetBody(c *gin.Context) ([]byte, error) {
	return c.GetRawData()
}

// ValidateRequiredQuery validates that required query parameters are present.
func (r *Request) ValidateRequiredQuery(c *gin.Context, keys ...string) error {
	for _, key := range keys {
		if c.Query(key) == "" {
			return errors.New(errors.CodeBadRequest, "MISSING_PARAMETER",
				"missing required query parameter: "+key)
		}
	}
	return nil
}

// ValidateRequiredPath validates that required path parameters are present.
func (r *Request) ValidateRequiredPath(c *gin.Context, keys ...string) error {
	for _, key := range keys {
		if c.Param(key) == "" {
			return errors.New(errors.CodeBadRequest, "MISSING_PARAMETER",
				"missing required path parameter: "+key)
		}
	}
	return nil
}

// ValidateRequiredHeader validates that required headers are present.
func (r *Request) ValidateRequiredHeader(c *gin.Context, keys ...string) error {
	for _, key := range keys {
		if c.Request.Header.Get(key) == "" {
			return errors.New(errors.CodeBadRequest, "MISSING_HEADER",
				"missing required header: "+key)
		}
	}
	return nil
}

// RequestFromContext extracts request helper from context.
func RequestFromContext(ctx context.Context) *Request {
	return &Request{}
}

// HandlerFunc is a convenience type for HTTP handler functions.
type HandlerFunc func(c *gin.Context) (any, error)

// WrapHandler wraps a handler function with standard error handling.
func WrapHandler(handler HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Execute handler
		data, err := handler(c)

		// Handle response
		resp := GetResponse()
		if err != nil {
			resp.HandleError(c, err)
			return
		}

		// Send success response
		resp.Success(c, data)
	}
}

// WrapHandlerWithStatus wraps a handler function with custom status handling.
func WrapHandlerWithStatus(handler HandlerFunc, successStatus int) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Execute handler
		data, err := handler(c)

		// Handle response
		resp := GetResponse()
		if err != nil {
			resp.HandleError(c, err)
			return
		}

		// Send response with custom status
		if successStatus == http.StatusCreated {
			resp.Created(c, data)
		} else if successStatus == http.StatusNoContent {
			resp.NoContent(c)
		} else {
			resp.Success(c, data)
		}
	}
}

// BindAndValidate binds request body and validates it.
func BindAndValidate(c *gin.Context, v any) error {
	parser := GetBodyParser()
	return parser.BindBody(c, v)
}

// BindJSONAndValidate binds JSON request body and validates it.
func BindJSONAndValidate(c *gin.Context, v any) error {
	parser := GetBodyParser()
	return parser.BindJSON(c, v)
}

// BindFormAndValidate binds form request body and validates it.
func BindFormAndValidate(c *gin.Context, v any) error {
	parser := GetBodyParser()
	return parser.BindForm(c, v)
}

// BindXMLAndValidate binds XML request body and validates it.
func BindXMLAndValidate(c *gin.Context, v any) error {
	parser := GetBodyParser()
	return parser.BindXML(c, v)
}
