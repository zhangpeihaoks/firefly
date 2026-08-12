// Package response provides unified response structures for the Firefly framework.
// All responses share a consistent {code, msg, data} envelope, and the
// constructors are generic so callers keep their concrete data type.
package response

// Response is the unified response structure.
// Data is always serialized (null when absent) so the envelope is stable.
type Response[T any] struct {
	// Code is the response code (typically HTTP status code)
	Code int `json:"code"`
	// Msg is the response message
	Msg string `json:"msg"`
	// Data is the response data
	Data T `json:"data"`
}

// PageResponse is the paginated response structure.
type PageResponse[T any] struct {
	// Code is the response code
	Code int `json:"code"`
	// Msg is the response message
	Msg string `json:"msg"`
	// Data is the response data
	Data T `json:"data"`
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

// Success creates a successful response with code 200 and msg "success".
func Success[T any](data T) *Response[T] {
	return &Response[T]{
		Code: 200,
		Msg:  "success",
		Data: data,
	}
}

// SuccessWithMessage creates a successful response with a custom message.
func SuccessWithMessage[T any](msg string, data T) *Response[T] {
	return &Response[T]{
		Code: 200,
		Msg:  msg,
		Data: data,
	}
}

// SuccessWithPage creates a successful paginated response.
func SuccessWithPage[T any](data T, page, pageSize int, total int64) *PageResponse[T] {
	totalPage := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPage++
	}
	return &PageResponse[T]{
		Code: 200,
		Msg:  "success",
		Data: data,
		Page: &PageInfo{
			Page:      page,
			PageSize:  pageSize,
			Total:     total,
			TotalPage: totalPage,
		},
	}
}

// Error creates an error response.
// Data is nil (serialized as null) to keep the {code, msg, data} envelope stable.
func Error(code int, msg string) *Response[any] {
	return &Response[any]{
		Code: code,
		Msg:  msg,
	}
}

// ErrorWithData creates an error response with additional data.
func ErrorWithData[T any](code int, msg string, data T) *Response[T] {
	return &Response[T]{
		Code: code,
		Msg:  msg,
		Data: data,
	}
}
