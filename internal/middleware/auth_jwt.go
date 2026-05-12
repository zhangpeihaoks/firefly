// Package middleware provides middleware abstractions for the Firefly framework.
package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/zhangpeihaoks/firefly/internal/errors"
	"github.com/zhangpeihaoks/firefly/internal/transport"
)

// jwtClaims represents the standard JWT claims used by the framework.
type jwtClaims struct {
	// Sub is the subject (user identifier)
	Sub string `json:"sub"`
	// Name is the username or display name
	Name string `json:"name"`
	// Exp is the expiration time (Unix timestamp)
	Exp int64 `json:"exp"`
	// Iat is the issued-at time (Unix timestamp)
	Iat int64 `json:"iat"`
}

// jwtAuthOptions holds the configuration for JWTAuth middleware.
type jwtAuthOptions struct {
	logger    *slog.Logger
	headerKey string
	skipPaths map[string]bool
}

// JWTAuthOption is a configuration option for JWTAuth middleware.
type JWTAuthOption func(*jwtAuthOptions)

// WithJWTAuthLogger sets a custom logger for the JWTAuth middleware.
func WithJWTAuthLogger(logger *slog.Logger) JWTAuthOption {
	return func(o *jwtAuthOptions) {
		o.logger = logger
	}
}

// WithJWTHeaderKey sets a custom header name for the JWT token.
// Default is "Authorization".
func WithJWTHeaderKey(key string) JWTAuthOption {
	return func(o *jwtAuthOptions) {
		o.headerKey = key
	}
}

// WithJWTSkipPaths sets paths that should skip JWT authentication.
func WithJWTSkipPaths(paths []string) JWTAuthOption {
	return func(o *jwtAuthOptions) {
		o.skipPaths = make(map[string]bool)
		for _, p := range paths {
			o.skipPaths[p] = true
		}
	}
}

// JWTAuth returns a middleware that validates HS256 JWT tokens from the
// Authorization header (Bearer scheme). On success it extracts sub and name
// claims into UserInfo and injects them into the context. On failure it
// returns a 401 Unauthorized error.
//
// The secret must be the raw HMAC key (not base64-encoded).
//
// Example:
//
//	middleware.JWTAuth([]byte("my-secret-key"))
//	middleware.JWTAuth([]byte("secret"), middleware.WithJWTSkipPaths([]string{"/health"}))
func JWTAuth(secret []byte, opts ...JWTAuthOption) Middleware {
	options := &jwtAuthOptions{
		logger:    slog.Default(),
		headerKey: "Authorization",
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

			// Extract token from header
			var token string
			if tr != nil {
				authHeader := tr.RequestHeader().Get(options.headerKey)
				token = extractBearerToken(authHeader)
			}

			if token == "" {
				options.logger.Debug("jwt auth: missing or empty token")
				return nil, errors.New(errors.CodeUnauthorized, "JWT_MISSING_TOKEN", "缺少认证令牌")
			}

			// Parse and validate the JWT
			claims, err := parseJWT(token, secret)
			if err != nil {
				options.logger.Debug("jwt auth: invalid token", "error", err)
				return nil, errors.New(errors.CodeUnauthorized, "JWT_INVALID_TOKEN", "无效的认证令牌")
			}

			// Check expiration
			if claims.Exp > 0 && time.Now().Unix() > claims.Exp {
				options.logger.Debug("jwt auth: token expired", "exp", claims.Exp)
				return nil, errors.New(errors.CodeUnauthorized, "JWT_TOKEN_EXPIRED", "认证令牌已过期")
			}

			// Build UserInfo from claims
			user := &UserInfo{
				ID:   claims.Sub,
				Name: claims.Name,
			}

			// Inject user into context
			ctx = NewContextWithUser(ctx, user)

			return next(ctx, req)
		}
	}
}

// extractBearerToken extracts the token from a "Bearer <token>" header value.
func extractBearerToken(header string) string {
	if header == "" {
		return ""
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(header[len(prefix):])
}

// parseJWT parses and validates a JWT token using HS256.
// It returns the claims if the signature is valid, or an error otherwise.
func parseJWT(token string, secret []byte) (*jwtClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New(errors.CodeUnauthorized, "JWT_MALFORMED", "令牌格式错误")
	}

	// Verify signature: HMAC-SHA256(header.payload, secret)
	signingInput := parts[0] + "." + parts[1]
	expectedSig := hmacSign(signingInput, secret)
	actualSig, err := base64URLDecode(parts[2])
	if err != nil {
		return nil, errors.New(errors.CodeUnauthorized, "JWT_MALFORMED_SIG", "令牌签名格式错误")
	}

	if !hmac.Equal(expectedSig, actualSig) {
		return nil, errors.New(errors.CodeUnauthorized, "JWT_INVALID_SIG", "令牌签名验证失败")
	}

	// Decode payload
	payload, err := base64URLDecode(parts[1])
	if err != nil {
		return nil, errors.New(errors.CodeUnauthorized, "JWT_MALFORMED_PAYLOAD", "令牌载荷格式错误")
	}

	var claims jwtClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, errors.New(errors.CodeUnauthorized, "JWT_MALFORMED_CLAIMS", "令牌声明格式错误")
	}

	return &claims, nil
}

// hmacSign computes HMAC-SHA256 of the input using the given secret.
func hmacSign(input string, secret []byte) []byte {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(input))
	return mac.Sum(nil)
}

// base64URLDecode decodes a base64url-encoded string (with optional padding).
func base64URLDecode(s string) ([]byte, error) {
	// Add padding if necessary
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}
	return base64.URLEncoding.DecodeString(s)
}
