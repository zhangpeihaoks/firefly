// Package middleware provides middleware abstractions for the Firefly framework.
package middleware

import (
	"context"
	"log/slog"

	"github.com/zhangpeihaoks/firefly/internal/errors"
	"github.com/zhangpeihaoks/firefly/internal/transport"
)

// APIKeyExtractor extracts an API key from the request context.
// It receives the transport context and returns the key string (empty if not found).
type APIKeyExtractor func(ctx context.Context, tr transport.Transporter) string

// APIKeyValidator validates an API key and returns whether it is valid.
// It receives the request context and the key string.
// Returns true if valid, false otherwise, and an error if validation itself fails.
type APIKeyValidator func(ctx context.Context, key string) (bool, error)

// apiKeyAuthOptions holds the configuration for APIKeyAuth middleware.
type apiKeyAuthOptions struct {
	logger       *slog.Logger
	skipPaths    map[string]bool
	userProvider func(ctx context.Context, key string) (*UserInfo, error)
}

// APIKeyAuthOption is a configuration option for APIKeyAuth middleware.
type APIKeyAuthOption func(*apiKeyAuthOptions)

// WithAPIKeyAuthLogger sets a custom logger for the APIKeyAuth middleware.
func WithAPIKeyAuthLogger(logger *slog.Logger) APIKeyAuthOption {
	return func(o *apiKeyAuthOptions) {
		o.logger = logger
	}
}

// WithAPIKeySkipPaths sets paths that should skip API key authentication.
func WithAPIKeySkipPaths(paths []string) APIKeyAuthOption {
	return func(o *apiKeyAuthOptions) {
		o.skipPaths = make(map[string]bool)
		for _, p := range paths {
			o.skipPaths[p] = true
		}
	}
}

// WithAPIKeyUserProvider sets a function that resolves an API key to UserInfo.
// When set, the resolved user is injected into the context after successful
// validation, enabling downstream handlers to use UserFromContext().
//
// Example:
//
//	middleware.WithAPIKeyUserProvider(func(ctx context.Context, key string) (*middleware.UserInfo, error) {
//	    user := lookupUserByAPIKey(ctx, key)
//	    return &middleware.UserInfo{ID: user.ID, Name: user.Name}, nil
//	})
func WithAPIKeyUserProvider(provider func(ctx context.Context, key string) (*UserInfo, error)) APIKeyAuthOption {
	return func(o *apiKeyAuthOptions) {
		o.userProvider = provider
	}
}

// APIKeyAuth returns a middleware that validates API keys using a custom
// extractor and validator. The extractor is responsible for locating the key
// in the request (header, query param, etc.), and the validator checks
// whether the key is valid.
//
// Example:
//
//	middleware.APIKeyAuth(
//	    middleware.APIKeyHeaderAuth("X-API-Key"),
//	    func(ctx context.Context, key string) (bool, error) {
//	        return key == "my-secret-key", nil
//	    },
//	)
func APIKeyAuth(extractor APIKeyExtractor, validator APIKeyValidator, opts ...APIKeyAuthOption) Middleware {
	options := &apiKeyAuthOptions{
		logger: slog.Default(),
	}
	for _, opt := range opts {
		opt(options)
	}

	return func(next Handler) Handler {
		return func(ctx context.Context, req any) (any, error) {
			tr := transport.FromContext(ctx)

			// Check skip paths
			if tr != nil && options.skipPaths != nil {
				if options.skipPaths[tr.Operation()] {
					return next(ctx, req)
				}
			}

			// Extract key
			key := extractor(ctx, tr)
			if key == "" {
				options.logger.Debug("api key auth: missing api key")
				return nil, errors.New(errors.CodeUnauthorized, "APIKEY_MISSING", "缺少 API Key")
			}

			// Validate key
			valid, err := validator(ctx, key)
			if err != nil {
				options.logger.Error("api key auth: validation error", "error", err)
				return nil, errors.New(errors.CodeInternal, "APIKEY_VALIDATION_ERROR", "API Key 验证失败")
			}
			if !valid {
				options.logger.Debug("api key auth: invalid api key")
				return nil, errors.New(errors.CodeUnauthorized, "APIKEY_INVALID", "无效的 API Key")
			}

			// Resolve and inject user identity if provider is configured
			if options.userProvider != nil {
				user, err := options.userProvider(ctx, key)
				if err != nil {
					options.logger.Error("api key auth: user provider error", "error", err)
					return nil, errors.New(errors.CodeInternal, "APIKEY_USER_ERROR", "无法解析用户信息")
				}
				if user != nil {
					ctx = NewContextWithUser(ctx, user)
				}
			}

			return next(ctx, req)
		}
	}
}

// APIKeyHeaderAuth returns an extractor that retrieves the API key from the
// specified HTTP header.
//
// Example:
//
//	middleware.APIKeyHeaderAuth("X-API-Key")
func APIKeyHeaderAuth(headerName string) APIKeyExtractor {
	return func(_ context.Context, tr transport.Transporter) string {
		if tr == nil {
			return ""
		}
		return tr.RequestHeader().Get(headerName)
	}
}

// APIKeyQueryAuth returns an extractor that retrieves the API key from the
// specified URL query parameter.
//
// Example:
//
//	middleware.APIKeyQueryAuth("api_key")
func APIKeyQueryAuth(paramName string) APIKeyExtractor {
	return func(_ context.Context, tr transport.Transporter) string {
		if tr == nil {
			return ""
		}
		values, exists := tr.QueryParams()[paramName]
		if !exists || len(values) == 0 {
			return ""
		}
		return values[0]
	}
}
