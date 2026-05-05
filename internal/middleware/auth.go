// Package middleware provides middleware abstractions for the Firefly framework.
package middleware

import (
	"context"
	"log/slog"

	"github.com/zhangpeihaoks/firefly/internal/errors"
	"github.com/zhangpeihaoks/firefly/internal/transport"
)

// UserInfo represents authenticated user information.
type UserInfo struct {
	// ID is the unique identifier of the user
	ID string
	// Name is the username
	Name string
	// Roles is the list of roles assigned to the user
	Roles []string
	// Metadata contains additional user information
	Metadata map[string]any
}

// contextKey is the key type for storing UserInfo in context.
type authContextKey struct{}

// userInfoKey is the context key for UserInfo values.
var userInfoKey = authContextKey{}

// NewContextWithUser returns a new context with the given UserInfo attached.
func NewContextWithUser(ctx context.Context, user *UserInfo) context.Context {
	return context.WithValue(ctx, userInfoKey, user)
}

// UserFromContext returns the UserInfo stored in the context.
// Returns nil if no UserInfo is found.
func UserFromContext(ctx context.Context) *UserInfo {
	if user, ok := ctx.Value(userInfoKey).(*UserInfo); ok {
		return user
	}
	return nil
}

// Authenticator is the interface for authentication logic.
// Implementations can provide different authentication strategies:
// - JWT token validation
// - API key validation
// - Session-based authentication
// - OAuth2 token validation
type Authenticator interface {
	// Authenticate authenticates the request and returns user information.
	// Returns an error if authentication fails.
	// The error should be *errors.Error for proper error handling.
	Authenticate(ctx context.Context, tr transport.Transporter) (*UserInfo, error)
}

// RoleChecker is the interface for role-based access control.
// Implementations can provide different role checking strategies:
// - Simple role list matching
// - Role hierarchy
// - Permission-based access control
type RoleChecker interface {
	// HasRole checks if the user has the required role.
	HasRole(user *UserInfo, requiredRoles []string) bool
}

// authOptions holds the configuration for Auth middleware.
type authOptions struct {
	logger        *slog.Logger
	authenticator Authenticator
	roleChecker   RoleChecker
	skipPaths     map[string]bool
}

// AuthOption is a configuration option for Auth middleware.
type AuthOption func(*authOptions)

// WithAuthLogger sets a custom logger for the Auth middleware.
// If not set, the default slog logger is used.
func WithAuthLogger(logger *slog.Logger) AuthOption {
	return func(o *authOptions) {
		o.logger = logger
	}
}

// WithAuthenticator sets the authenticator for the Auth middleware.
// The authenticator is responsible for validating credentials and extracting user info.
func WithAuthenticator(auth Authenticator) AuthOption {
	return func(o *authOptions) {
		o.authenticator = auth
	}
}

// WithRoleChecker sets the role checker for the Auth middleware.
// The role checker is responsible for verifying user roles.
func WithRoleChecker(checker RoleChecker) AuthOption {
	return func(o *authOptions) {
		o.roleChecker = checker
	}
}

// WithSkipPaths sets paths that should skip authentication.
// These paths will be accessible without authentication.
func WithSkipPaths(paths []string) AuthOption {
	return func(o *authOptions) {
		o.skipPaths = make(map[string]bool)
		for _, path := range paths {
			o.skipPaths[path] = true
		}
	}
}

// Auth returns a middleware that performs authentication.
// It:
//  1. Extracts credentials from the request (via Authenticator)
//  2. Validates the credentials and extracts user info
//  3. Stores user info in context for downstream handlers
//  4. Returns 401 Unauthorized if authentication fails
//
// Example:
//
//	// With custom authenticator
//	middleware.Auth(
//	    middleware.WithAuthenticator(myJWTAuthenticator),
//	)
//
//	// With skip paths
//	middleware.Auth(
//	    middleware.WithAuthenticator(myAuthenticator),
//	    middleware.WithSkipPaths([]string{"/health", "/metrics"}),
//	)
func Auth(opts ...AuthOption) Middleware {
	// Apply default options
	options := &authOptions{
		logger: slog.Default(),
	}

	// Apply custom options
	for _, opt := range opts {
		opt(options)
	}

	return func(next Handler) Handler {
		return func(ctx context.Context, req any) (any, error) {
			// Get transport info from context
			tr := transport.FromContext(ctx)

			// Check if path should skip authentication
			if tr != nil && options.skipPaths != nil {
				if options.skipPaths[tr.Operation()] {
					return next(ctx, req)
				}
			}

			// Check if authenticator is configured
			if options.authenticator == nil {
				options.logger.Warn("auth middleware: no authenticator configured")
				return nil, errors.New(errors.CodeInternal, "AUTH_CONFIG_ERROR", "认证配置错误")
			}

			// Perform authentication
			user, err := options.authenticator.Authenticate(ctx, tr)
			if err != nil {
				options.logger.Debug("authentication failed",
					"error", err.Error(),
					"operation", func() string {
						if tr != nil {
							return tr.Operation()
						}
						return ""
					}(),
				)
				return nil, err
			}

			// Store user info in context
			ctx = NewContextWithUser(ctx, user)

			// Call the next handler
			return next(ctx, req)
		}
	}
}

// RequireRoles returns a middleware that checks if the user has the required roles.
// This middleware should be used after the Auth middleware.
// It returns 403 Forbidden if the user doesn't have the required roles.
//
// Example:
//
//	// Require admin role
//	middleware.RequireRoles("admin")
//
//	// Require either admin or moderator role
//	middleware.RequireRoles("admin", "moderator")
func RequireRoles(roles ...string) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, req any) (any, error) {
			// Get user from context
			user := UserFromContext(ctx)
			if user == nil {
				return nil, errors.New(errors.CodeUnauthorized, "UNAUTHORIZED", "未授权访问")
			}

			// Check if user has any of the required roles
			hasRole := false
			for _, userRole := range user.Roles {
				for _, requiredRole := range roles {
					if userRole == requiredRole {
						hasRole = true
						break
					}
				}
				if hasRole {
					break
				}
			}

			if !hasRole {
				return nil, errors.New(errors.CodeForbidden, "FORBIDDEN", "权限不足")
			}

			// Call the next handler
			return next(ctx, req)
		}
	}
}

// RequireRolesWithChecker returns a middleware that checks roles using a custom RoleChecker.
// This allows for more complex role checking logic like role hierarchies or permissions.
//
// Example:
//
//	// With custom role checker
//	middleware.RequireRolesWithChecker(myRoleChecker, "admin")
func RequireRolesWithChecker(checker RoleChecker, roles ...string) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, req any) (any, error) {
			// Get user from context
			user := UserFromContext(ctx)
			if user == nil {
				return nil, errors.New(errors.CodeUnauthorized, "UNAUTHORIZED", "未授权访问")
			}

			// Check roles using the custom checker
			if !checker.HasRole(user, roles) {
				return nil, errors.New(errors.CodeForbidden, "FORBIDDEN", "权限不足")
			}

			// Call the next handler
			return next(ctx, req)
		}
	}
}

// DefaultRoleChecker is a simple implementation of RoleChecker.
// It checks if the user has any of the required roles.
type DefaultRoleChecker struct{}

// HasRole checks if the user has any of the required roles.
func (c *DefaultRoleChecker) HasRole(user *UserInfo, requiredRoles []string) bool {
	for _, userRole := range user.Roles {
		for _, requiredRole := range requiredRoles {
			if userRole == requiredRole {
				return true
			}
		}
	}
	return false
}

// NewDefaultRoleChecker creates a new DefaultRoleChecker.
func NewDefaultRoleChecker() *DefaultRoleChecker {
	return &DefaultRoleChecker{}
}
