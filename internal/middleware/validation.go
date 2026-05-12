// Package middleware provides middleware abstractions for the Firefly framework.
package middleware

import (
	"context"
	"encoding/json"

	"github.com/zhangpeihaoks/firefly/internal/errors"
)

// Body returns a middleware that ensures the request body is deserialized into
// the specified type T. It performs a JSON round-trip to convert the raw request
// (typically map[string]any) into the target type, providing compile-time type
// safety for downstream handlers.
//
// On deserialization failure it returns a 400 Bad Request error.
//
// Example:
//
//	type CreateUserReq struct {
//	    Name  string `json:"name"`
//	    Email string `json:"email"`
//	}
//
//	handler := middleware.Body[CreateUserReq]()(
//	    func(ctx context.Context, req any) (any, error) {
//	        body := req.(CreateUserReq) // safe cast after Body middleware
//	        // ...
//	    },
//	)
func Body[T any]() Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, req any) (any, error) {
			// Fast path: if req is already the target type, pass through.
			if typed, ok := req.(T); ok {
				return next(ctx, typed)
			}

			// JSON round-trip: marshal the raw request then unmarshal into T.
			// This handles the common case where req is map[string]any from
			// the transport layer's JSON decoding.
			data, err := json.Marshal(req)
			if err != nil {
				return nil, errors.Newf(errors.CodeBadRequest, "BODY_MARSHAL_ERROR", "请求体序列化失败: %v", err)
			}

			var body T
			if err := json.Unmarshal(data, &body); err != nil {
				return nil, errors.Newf(errors.CodeBadRequest, "BODY_INVALID_FORMAT", "请求体格式错误: %v", err)
			}

			return next(ctx, body)
		}
	}
}
