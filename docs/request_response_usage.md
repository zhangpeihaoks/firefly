# Request and Response Handling in Firefly Framework

This document provides usage examples for the request and response handling functionality implemented in task 9.7.

## Overview

The Firefly framework now provides comprehensive request and response handling capabilities:

1. **BodyParser**: Parse request bodies in JSON, Form, and XML formats
2. **Response**: Standardized response helpers with error handling
3. **Request**: Helper functions for accessing request data
4. **WrapHandler**: Clean handler patterns with automatic error handling

## BodyParser Usage

### Parsing JSON Request Bodies

```go
import "github.com/zhangpeihaoks/firefly/internal/transport/http"

// In a handler
func createUserHandler(c *gin.Context) {
    type CreateUserRequest struct {
        Name  string `json:"name"`
        Email string `json:"email"`
    }
    
    var req CreateUserRequest
    parser := http.GetBodyParser()
    
    // Parse JSON body
    if err := parser.ParseJSON(c, &req); err != nil {
        // Handle error
        resp := http.GetResponse()
        resp.HandleError(c, err)
        return
    }
    
    // Use parsed data
    // ...
}
```

### Parsing XML Request Bodies

```go
func updateUserHandler(c *gin.Context) {
    type UpdateUserRequest struct {
        XMLName xml.Name `xml:"user"`
        Name    string   `xml:"name"`
        Email   string   `xml:"email"`
    }
    
    var req UpdateUserRequest
    parser := http.GetBodyParser()
    
    if err := parser.ParseXML(c, &req); err != nil {
        resp := http.GetResponse()
        resp.HandleError(c, err)
        return
    }
    
    // Use parsed data
    // ...
}
```

### Automatic Content-Type Detection

```go
func handleRequest(c *gin.Context) {
    type RequestData struct {
        Name  string `json:"name" xml:"name"`
        Email string `json:"email" xml:"email"`
    }
    
    var data RequestData
    parser := http.GetBodyParser()
    
    // Automatically detects content type (JSON, XML, Form)
    if err := parser.ParseBody(c, &data); err != nil {
        resp := http.GetResponse()
        resp.HandleError(c, err)
        return
    }
    
    // Use parsed data
    // ...
}
```

## Response Helpers

### Standard Success Responses

```go
import "github.com/zhangpeihaoks/firefly/internal/transport/http"

func getUserHandler(c *gin.Context) {
    user := getUserFromDB() // Your business logic
    
    resp := http.GetResponse()
    resp.Success(c, user)
}

func createUserHandler(c *gin.Context) {
    user := createUserInDB() // Your business logic
    
    resp := http.GetResponse()
    resp.Created(c, user) // Returns 201 Created
}

func deleteUserHandler(c *gin.Context) {
    deleteUserFromDB() // Your business logic
    
    resp := http.GetResponse()
    resp.NoContent(c) // Returns 204 No Content
}
```

### Paginated Responses

```go
func listUsersHandler(c *gin.Context) {
    page, _ := strconv.Atoi(c.Query("page"))
    pageSize, _ := strconv.Atoi(c.Query("pageSize"))
    
    users, total := getUsersFromDB(page, pageSize) // Your business logic
    
    resp := http.GetResponse()
    resp.SuccessWithPage(c, users, page, pageSize, total)
}
```

### Error Handling

```go
func getUserHandler(c *gin.Context) {
    userID := c.Param("id")
    user, err := getUserFromDB(userID)
    
    resp := http.GetResponse()
    
    if err != nil {
        // Handle framework errors automatically
        resp.HandleError(c, err)
        return
    }
    
    if user == nil {
        // Custom error response
        resp.NotFound(c, "User not found")
        return
    }
    
    resp.Success(c, user)
}

// Or use HandleSuccess for combined handling
func getUserHandler2(c *gin.Context) {
    userID := c.Param("id")
    user, err := getUserFromDB(userID)
    
    resp := http.GetResponse()
    resp.HandleSuccess(c, user, err) // Handles both success and error
}
```

## Request Helpers

### Accessing Request Data

```go
import "github.com/zhangpeihaoks/firefly/internal/transport/http"

func getUserHandler(c *gin.Context) {
    req := http.GetRequest()
    
    // Get path parameter
    userID := req.GetPathParam(c, "id")
    
    // Get query parameters
    page := req.GetQuery(c, "page")
    sort := req.GetQuery(c, "sort")
    
    // Get query parameters with type conversion
    pageInt, err := req.GetQueryInt(c, "page")
    if err != nil {
        // Handle error
    }
    
    // Get headers
    authHeader := req.GetHeader(c, "Authorization")
    contentType := req.GetContentType(c)
    
    // Get client IP
    clientIP := req.GetClientIP(c)
    
    // Use the data...
}
```

### Validation

```go
func createUserHandler(c *gin.Context) {
    req := http.GetRequest()
    
    // Validate required query parameters
    if err := req.ValidateRequiredQuery(c, "api_key"); err != nil {
        resp := http.GetResponse()
        resp.HandleError(c, err)
        return
    }
    
    // Validate required path parameters
    if err := req.ValidateRequiredPath(c, "id"); err != nil {
        resp := http.GetResponse()
        resp.HandleError(c, err)
        return
    }
    
    // Validate required headers
    if err := req.ValidateRequiredHeader(c, "X-Request-ID"); err != nil {
        resp := http.GetResponse()
        resp.HandleError(c, err)
        return
    }
}
```

## WrapHandler Pattern

### Basic Usage

```go
import "github.com/zhangpeihaoks/firefly/internal/transport/http"

// Define handler that returns (data, error)
var getUserHandler = http.WrapHandler(func(c *gin.Context) (any, error) {
    req := http.GetRequest()
    userID, err := req.GetPathParamInt(c, "id")
    if err != nil {
        return nil, err
    }
    
    user, err := getUserFromDB(userID)
    if err != nil {
        return nil, err
    }
    
    if user == nil {
        return nil, errors.New(errors.CodeNotFound, "USER_NOT_FOUND", "User not found")
    }
    
    return user, nil
})

// In router setup
router.GET("/users/:id", getUserHandler)
```

### With Custom Status Codes

```go
// Create handler returns 201 Created
var createUserHandler = http.WrapHandlerWithStatus(func(c *gin.Context) (any, error) {
    type CreateRequest struct {
        Name  string `json:"name"`
        Email string `json:"email"`
    }
    
    var req CreateRequest
    if err := http.BindJSONAndValidate(c, &req); err != nil {
        return nil, err
    }
    
    user := createUserInDB(req.Name, req.Email)
    return user, nil
}, http.StatusCreated) // Custom status code

// Delete handler returns 204 No Content
var deleteUserHandler = http.WrapHandlerWithStatus(func(c *gin.Context) (any, error) {
    req := http.GetRequest()
    userID, err := req.GetPathParamInt(c, "id")
    if err != nil {
        return nil, err
    }
    
    deleteUserFromDB(userID)
    return nil, nil // nil data for NoContent
}, http.StatusNoContent)
```

### Binding and Validation

```go
var updateUserHandler = http.WrapHandler(func(c *gin.Context) (any, error) {
    type UpdateRequest struct {
        Name  string `json:"name" validate:"required,min=3"`
        Email string `json:"email" validate:"required,email"`
    }
    
    var req UpdateRequest
    
    // Bind and validate JSON body
    if err := http.BindJSONAndValidate(c, &req); err != nil {
        return nil, err
    }
    
    reqHelper := http.GetRequest()
    userID, err := reqHelper.GetPathParamInt(c, "id")
    if err != nil {
        return nil, err
    }
    
    user := updateUserInDB(userID, req.Name, req.Email)
    return user, nil
})
```

## Complete Example

Here's a complete example showing all features together:

```go
package main

import (
	"github.com/gin-gonic/gin"
	"github.com/zhangpeihaoks/firefly/internal/errors"
	httptransport "github.com/zhangpeihaoks/firefly/internal/transport/http"
)

func main() {
	router := gin.Default()
	
	// User routes with WrapHandler
	router.GET("/users/:id", httptransport.WrapHandler(getUserHandler))
	router.POST("/users", httptransport.WrapHandlerWithStatus(createUserHandler, httptransport.StatusCreated))
	router.PUT("/users/:id", httptransport.WrapHandler(updateUserHandler))
	router.DELETE("/users/:id", httptransport.WrapHandlerWithStatus(deleteUserHandler, httptransport.StatusNoContent))
	router.GET("/users", httptransport.WrapHandler(listUsersHandler))
	
	router.Run(":8080")
}

// Handler implementations
func getUserHandler(c *gin.Context) (any, error) {
	req := httptransport.GetRequest()
	userID, err := req.GetPathParamInt(c, "id")
	if err != nil {
		return nil, err
	}
	
	// Validate required query parameters
	if err := req.ValidateRequiredQuery(c, "fields"); err != nil {
		return nil, err
	}
	
	user := getUserFromDB(userID)
	if user == nil {
		return nil, errors.New(errors.CodeNotFound, "USER_NOT_FOUND", "User not found")
	}
	
	return user, nil
}

func createUserHandler(c *gin.Context) (any, error) {
	type CreateRequest struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	
	var req CreateRequest
	
	// Parse JSON body
	parser := httptransport.GetBodyParser()
	if err := parser.ParseJSON(c, &req); err != nil {
		return nil, err
	}
	
	// Validate request
	if req.Name == "" || req.Email == "" {
		return nil, errors.New(errors.CodeBadRequest, "INVALID_REQUEST", "name and email are required")
	}
	
	user := createUserInDB(req.Name, req.Email)
	return user, nil
}

func listUsersHandler(c *gin.Context) (any, error) {
	req := httptransport.GetRequest()
	
	page, _ := req.GetQueryInt(c, "page")
	if page == 0 {
		page = 1
	}
	
	pageSize, _ := req.GetQueryInt(c, "pageSize")
	if pageSize == 0 {
		pageSize = 10
	}
	
	users, total := getUsersFromDB(page, pageSize)
	
	// Return paginated response
	resp := httptransport.GetResponse()
	return resp.SuccessWithPage(users, page, pageSize, total), nil
}

// Mock database functions
func getUserFromDB(id int) map[string]interface{} {
	return map[string]interface{}{
		"id":    id,
		"name":  "Test User",
		"email": "test@example.com",
	}
}

func createUserInDB(name, email string) map[string]interface{} {
	return map[string]interface{}{
		"id":    1,
		"name":  name,
		"email": email,
	}
}

func getUsersFromDB(page, pageSize int) ([]map[string]interface{}, int64) {
	users := []map[string]interface{}{
		{"id": 1, "name": "User 1", "email": "user1@example.com"},
		{"id": 2, "name": "User 2", "email": "user2@example.com"},
	}
	return users, 100
}

func updateUserInDB(id int, name, email string) map[string]interface{} {
	return map[string]interface{}{
		"id":    id,
		"name":  name,
		"email": email,
	}
}

func deleteUserFromDB(id int) {
	// Delete logic
}
```

## Property Tests

The implementation includes property-based tests that validate:

1. **Property 20: Request parameter获取** - Request parameters (Header, Query, Path) are correctly retrieved
2. **Property 21: Request body解析** - Request bodies (JSON, Form, XML) are correctly parsed
3. **Property 24: Unified response structure** - Responses follow the unified structure (Code, Message, Data)

These properties are tested using the Go testing framework with comprehensive test coverage.

## Integration with Existing Code

The new request/response functionality integrates seamlessly with the existing Firefly framework:

1. **Error Handling**: Uses the framework's unified error system
2. **Middleware**: Works with existing middleware (Recovery, Logging, Tracing, etc.)
3. **Routing**: Integrates with the existing Gin-based routing system
4. **Serialization**: Compatible with JSON and Protobuf serialization modes

## Benefits

1. **Consistency**: Standardized request/response patterns across the codebase
2. **Type Safety**: Strongly typed parameter access and response building
3. **Error Handling**: Unified error handling with automatic HTTP status mapping
4. **Testability**: Easy to test with mock contexts and request/response objects
5. **Productivity**: Reduces boilerplate code for common request/response patterns