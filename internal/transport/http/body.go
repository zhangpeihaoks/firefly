// Package http provides HTTP server implementation for the Firefly framework.
package http

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"io"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/zhangpeihaoks/firefly/internal/errors"
)

// BodyParser provides methods for parsing HTTP request bodies in different formats.
type BodyParser struct{}

// ParseJSON parses a JSON request body into the provided struct.
// Returns an error if the body cannot be parsed or if the content type is not JSON.
func (p *BodyParser) ParseJSON(c *gin.Context, v any) error {
	// Check content type
	contentType := c.Request.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		return errors.New(errors.CodeBadRequest, "INVALID_CONTENT_TYPE", "expected application/json")
	}

	// Read and parse body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return errors.Newf(errors.CodeBadRequest, "READ_BODY_ERROR", "failed to read request body: %v", err)
	}

	if err := json.Unmarshal(body, v); err != nil {
		return errors.Newf(errors.CodeBadRequest, "INVALID_JSON", "invalid JSON format: %v", err)
	}

	return nil
}

// ParseForm parses a form request body (application/x-www-form-urlencoded or multipart/form-data).
// Returns an error if the form cannot be parsed.
func (p *BodyParser) ParseForm(c *gin.Context, v any) error {
	// Check content type
	contentType := c.Request.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/x-www-form-urlencoded") &&
		!strings.Contains(contentType, "multipart/form-data") {
		return errors.New(errors.CodeBadRequest, "INVALID_CONTENT_TYPE", "expected application/x-www-form-urlencoded or multipart/form-data")
	}

	// Parse form
	if err := c.Request.ParseForm(); err != nil {
		return errors.Newf(errors.CodeBadRequest, "PARSE_FORM_ERROR", "failed to parse form: %v", err)
	}

	// For now, we'll handle simple form parsing
	// In a more complete implementation, we would bind form values to struct
	return nil
}

// ParseXML parses an XML request body into the provided struct.
// Returns an error if the body cannot be parsed or if the content type is not XML.
func (p *BodyParser) ParseXML(c *gin.Context, v any) error {
	// Check content type
	contentType := c.Request.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/xml") &&
		!strings.Contains(contentType, "text/xml") {
		return errors.New(errors.CodeBadRequest, "INVALID_CONTENT_TYPE", "expected application/xml or text/xml")
	}

	// Read and parse body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return errors.Newf(errors.CodeBadRequest, "READ_BODY_ERROR", "failed to read request body: %v", err)
	}

	if err := xml.Unmarshal(body, v); err != nil {
		return errors.Newf(errors.CodeBadRequest, "INVALID_XML", "invalid XML format: %v", err)
	}

	return nil
}

// ParseBody automatically parses the request body based on Content-Type header.
// Supports JSON, Form, and XML formats.
func (p *BodyParser) ParseBody(c *gin.Context, v any) error {
	contentType := c.Request.Header.Get("Content-Type")

	switch {
	case strings.Contains(contentType, "application/json"):
		return p.ParseJSON(c, v)
	case strings.Contains(contentType, "application/x-www-form-urlencoded"),
		strings.Contains(contentType, "multipart/form-data"):
		return p.ParseForm(c, v)
	case strings.Contains(contentType, "application/xml"),
		strings.Contains(contentType, "text/xml"):
		return p.ParseXML(c, v)
	default:
		// Default to JSON if no content type specified
		if contentType == "" {
			// Try to parse as JSON without checking content type
			body, err := io.ReadAll(c.Request.Body)
			if err != nil {
				return errors.Newf(errors.CodeBadRequest, "READ_BODY_ERROR", "failed to read request body: %v", err)
			}

			if err := json.Unmarshal(body, v); err != nil {
				return errors.Newf(errors.CodeBadRequest, "INVALID_JSON", "invalid JSON format: %v", err)
			}
			return nil
		}
		return errors.New(errors.CodeBadRequest, "UNSUPPORTED_CONTENT_TYPE",
			"unsupported content type: "+contentType)
	}
}

// BindJSON binds JSON request body to struct with validation.
func (p *BodyParser) BindJSON(c *gin.Context, v any) error {
	if err := p.ParseJSON(c, v); err != nil {
		return err
	}
	// TODO: Add validation using go-playground/validator
	return nil
}

// BindForm binds form request body to struct with validation.
func (p *BodyParser) BindForm(c *gin.Context, v any) error {
	if err := p.ParseForm(c, v); err != nil {
		return err
	}
	// TODO: Add form binding and validation
	return nil
}

// BindXML binds XML request body to struct with validation.
func (p *BodyParser) BindXML(c *gin.Context, v any) error {
	if err := p.ParseXML(c, v); err != nil {
		return err
	}
	// TODO: Add validation
	return nil
}

// BindBody automatically binds request body based on Content-Type.
func (p *BodyParser) BindBody(c *gin.Context, v any) error {
	return p.ParseBody(c, v)
}

// GetBodyParser returns a BodyParser instance.
func GetBodyParser() *BodyParser {
	return &BodyParser{}
}

// ParseJSONFromContext parses JSON request body from context.
func ParseJSONFromContext(ctx context.Context, v any) error {
	c, ok := ctx.Value(gin.ContextKey).(*gin.Context)
	if !ok {
		return errors.New(errors.CodeInternal, "CONTEXT_ERROR", "gin context not found")
	}

	parser := GetBodyParser()
	return parser.ParseJSON(c, v)
}

// ParseFormFromContext parses form request body from context.
func ParseFormFromContext(ctx context.Context, v any) error {
	c, ok := ctx.Value(gin.ContextKey).(*gin.Context)
	if !ok {
		return errors.New(errors.CodeInternal, "CONTEXT_ERROR", "gin context not found")
	}

	parser := GetBodyParser()
	return parser.ParseForm(c, v)
}

// ParseXMLFromContext parses XML request body from context.
func ParseXMLFromContext(ctx context.Context, v any) error {
	c, ok := ctx.Value(gin.ContextKey).(*gin.Context)
	if !ok {
		return errors.New(errors.CodeInternal, "CONTEXT_ERROR", "gin context not found")
	}

	parser := GetBodyParser()
	return parser.ParseXML(c, v)
}

// ParseBodyFromContext automatically parses request body from context based on Content-Type.
func ParseBodyFromContext(ctx context.Context, v any) error {
	c, ok := ctx.Value(gin.ContextKey).(*gin.Context)
	if !ok {
		return errors.New(errors.CodeInternal, "CONTEXT_ERROR", "gin context not found")
	}

	parser := GetBodyParser()
	return parser.ParseBody(c, v)
}
