// Package http provides HTTP server implementation for the Firefly framework.
package http

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zhangpeihaoks/firefly/internal/errors"
	"github.com/zhangpeihaoks/firefly/pkg/response"
)

// Response is a helper for sending standardized HTTP responses.
type Response struct{}

// JSON sends a JSON response with the given status code and data.
func (r *Response) JSON(c *gin.Context, status int, data any) {
	c.JSON(status, data)
}

// XML sends an XML response with the given status code and data.
func (r *Response) XML(c *gin.Context, status int, data any) {
	c.XML(status, data)
}

// HTML sends an HTML response with the given status code and data.
func (r *Response) HTML(c *gin.Context, status int, name string, data any) {
	c.HTML(status, name, data)
}

// File sends a file response.
func (r *Response) File(c *gin.Context, filepath string) {
	c.File(filepath)
}

// FileAttachment sends a file as an attachment.
func (r *Response) FileAttachment(c *gin.Context, filepath, filename string) {
	c.FileAttachment(filepath, filename)
}

// Success sends a success JSON response.
func (r *Response) Success(c *gin.Context, data any) {
	r.JSON(c, http.StatusOK, response.Success(data))
}

// SuccessXML sends a success XML response.
func (r *Response) SuccessXML(c *gin.Context, data any) {
	r.XML(c, http.StatusOK, response.Success(data))
}

// SuccessHTML sends a success HTML response.
func (r *Response) SuccessHTML(c *gin.Context, templateName string, data any) {
	r.HTML(c, http.StatusOK, templateName, data)
}

// SuccessWithMessage sends a success JSON response with custom message.
func (r *Response) SuccessWithMessage(c *gin.Context, message string, data any) {
	r.JSON(c, http.StatusOK, response.SuccessWithMessage(message, data))
}

// SuccessWithPage sends a paginated success JSON response.
func (r *Response) SuccessWithPage(c *gin.Context, data any, page, pageSize int, total int64) {
	r.JSON(c, http.StatusOK, response.SuccessWithPage(data, page, pageSize, total))
}

// Error sends an error JSON response.
func (r *Response) Error(c *gin.Context, status int, message string) {
	r.JSON(c, status, response.Error(status, message))
}

// ErrorXML sends an error XML response.
func (r *Response) ErrorXML(c *gin.Context, status int, message string) {
	r.XML(c, status, response.Error(status, message))
}

// ErrorHTML sends an error HTML response.
func (r *Response) ErrorHTML(c *gin.Context, status int, templateName string, data any) {
	r.HTML(c, status, templateName, data)
}

// ErrorWithData sends an error JSON response with additional data.
func (r *Response) ErrorWithData(c *gin.Context, status int, message string, data any) {
	r.JSON(c, status, response.ErrorWithData(status, message, data))
}

// Created sends a 201 Created response.
func (r *Response) Created(c *gin.Context, data any) {
	r.JSON(c, http.StatusCreated, response.SuccessWithMessage("created", data))
}

// NoContent sends a 204 No Content response.
func (r *Response) NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
	c.Writer.WriteHeaderNow()
}

// BadRequest sends a 400 Bad Request response.
func (r *Response) BadRequest(c *gin.Context, message string) {
	r.Error(c, http.StatusBadRequest, message)
}

// Unauthorized sends a 401 Unauthorized response.
func (r *Response) Unauthorized(c *gin.Context, message string) {
	r.Error(c, http.StatusUnauthorized, message)
}

// Forbidden sends a 403 Forbidden response.
func (r *Response) Forbidden(c *gin.Context, message string) {
	r.Error(c, http.StatusForbidden, message)
}

// NotFound sends a 404 Not Found response.
func (r *Response) NotFound(c *gin.Context, message string) {
	r.Error(c, http.StatusNotFound, message)
}

// InternalServerError sends a 500 Internal Server Error response.
func (r *Response) InternalServerError(c *gin.Context, message string) {
	r.Error(c, http.StatusInternalServerError, message)
}

// HandleError handles framework errors and sends appropriate HTTP response.
func (r *Response) HandleError(c *gin.Context, err error) {
	if err == nil {
		return
	}

	// Convert to framework error
	fwErr := errors.FromError(err)

	// Map error code to HTTP status
	status := errors.ToHTTPStatus(int(fwErr.Code))

	// Determine response format based on Accept header
	accept := c.GetHeader("Accept")

	switch {
	case accept == "application/xml" || accept == "text/xml":
		r.XML(c, status, response.Error(int(fwErr.Code), fwErr.Message))
	case accept == "text/html":
		// For HTML errors, we need to provide template data
		errorData := map[string]interface{}{
			"code":    fwErr.Code,
			"message": fwErr.Message,
			"reason":  fwErr.Reason,
		}
		r.HTML(c, status, "error.html", errorData)
	default:
		// Default to JSON
		r.JSON(c, status, response.Error(int(fwErr.Code), fwErr.Message))
	}
}

// HandleSuccess handles successful responses with optional data transformation.
func (r *Response) HandleSuccess(c *gin.Context, data any, err error) {
	if err != nil {
		r.HandleError(c, err)
		return
	}

	r.Success(c, data)
}

// JSONResponse sends a JSON response using the unified response structure.
func JSONResponse(c *gin.Context, status int, resp *response.Response) {
	c.JSON(status, resp)
}

// PageResponse sends a paginated JSON response.
func PageResponse(c *gin.Context, status int, resp *response.PageResponse) {
	c.JSON(status, resp)
}

// ResponseFromContext extracts response helper from context.
func ResponseFromContext(ctx context.Context) *Response {
	return &Response{}
}

// GetResponse returns a Response instance.
func GetResponse() *Response {
	return &Response{}
}
