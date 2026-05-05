// Package response provides unified response structures for the Firefly framework.
package response

// Response is the unified response structure.
type Response struct {
	// Code is the response code
	Code int `json:"code"`
	// Message is the response message
	Message string `json:"message"`
	// Data is the response data
	Data any `json:"data,omitempty"`
}

// PageResponse is the paginated response structure.
type PageResponse struct {
	// Code is the response code
	Code int `json:"code"`
	// Message is the response message
	Message string `json:"message"`
	// Data is the response data
	Data any `json:"data,omitempty"`
	// Page contains pagination information
	Page *PageInfo `json:"page,omitempty"`
}

// PageInfo contains pagination information.
type PageInfo struct {
	// Page is the current page number
	Page int `json:"page"`
	// PageSize is the number of items per page
	PageSize int `json:"page_size"`
	// Total is the total number of items
	Total int64 `json:"total"`
	// TotalPage is the total number of pages
	TotalPage int `json:"total_page"`
}

// Success creates a successful response.
func Success(data any) *Response {
	return &Response{
		Code:    200,
		Message: "success",
		Data:    data,
	}
}

// SuccessWithMessage creates a successful response with a custom message.
func SuccessWithMessage(message string, data any) *Response {
	return &Response{
		Code:    200,
		Message: message,
		Data:    data,
	}
}

// SuccessWithPage creates a successful paginated response.
func SuccessWithPage(data any, page, pageSize int, total int64) *PageResponse {
	totalPage := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPage++
	}
	return &PageResponse{
		Code:    200,
		Message: "success",
		Data:    data,
		Page: &PageInfo{
			Page:      page,
			PageSize:  pageSize,
			Total:     total,
			TotalPage: totalPage,
		},
	}
}

// Error creates an error response.
func Error(code int, message string) *Response {
	return &Response{
		Code:    code,
		Message: message,
	}
}

// ErrorWithData creates an error response with data.
func ErrorWithData(code int, message string, data any) *Response {
	return &Response{
		Code:    code,
		Message: message,
		Data:    data,
	}
}
