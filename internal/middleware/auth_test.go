// Package middleware provides middleware abstractions for the Firefly framework.
package middleware

import (
	"context"
	"testing"

	"github.com/zhangpeihaoks/firefly/internal/errors"
	"github.com/zhangpeihaoks/firefly/internal/transport"
)

// authMockTransporter implements transport.Transporter for testing.
type authMockTransporter struct {
	kind        transport.Kind
	endpoint    string
	operation   string
	reqHeader   transport.Header
	replyHeader transport.Header
	pathParams  map[string]string
	queryParams map[string][]string
}

func (m *authMockTransporter) Kind() transport.Kind {
	return m.kind
}

func (m *authMockTransporter) Endpoint() string {
	return m.endpoint
}

func (m *authMockTransporter) Operation() string {
	return m.operation
}

func (m *authMockTransporter) RequestHeader() transport.Header {
	return m.reqHeader
}

func (m *authMockTransporter) ReplyHeader() transport.Header {
	return m.replyHeader
}

func (m *authMockTransporter) PathParams() map[string]string {
	if m.pathParams == nil {
		return make(map[string]string)
	}
	return m.pathParams
}

func (m *authMockTransporter) QueryParams() map[string][]string {
	if m.queryParams == nil {
		return make(map[string][]string)
	}
	return m.queryParams
}

// authMockHeader implements transport.Header for testing.
type authMockHeader struct {
	values map[string]string
}

func newAuthMockHeader() *authMockHeader {
	return &authMockHeader{
		values: make(map[string]string),
	}
}

func (m *authMockHeader) Get(key string) string {
	return m.values[key]
}

func (m *authMockHeader) Set(key string, value string) {
	m.values[key] = value
}

func (m *authMockHeader) Keys() []string {
	keys := make([]string, 0, len(m.values))
	for k := range m.values {
		keys = append(keys, k)
	}
	return keys
}

// mockAuthenticator implements Authenticator for testing.
type mockAuthenticator struct {
	user *UserInfo
	err  error
}

func (m *mockAuthenticator) Authenticate(ctx context.Context, tr transport.Transporter) (*UserInfo, error) {
	return m.user, m.err
}

func TestAuth(t *testing.T) {
	tests := []struct {
		name          string
		authenticator Authenticator
		skipPaths     []string
		transport     transport.Transporter
		wantErr       bool
		wantErrCode   int
		wantUserInCtx bool
	}{
		{
			name: "successful authentication",
			authenticator: &mockAuthenticator{
				user: &UserInfo{ID: "1", Name: "test", Roles: []string{"user"}},
			},
			transport: &authMockTransporter{
				kind:      transport.KindHTTP,
				operation: "/api/test",
			},
			wantErr:       false,
			wantUserInCtx: true,
		},
		{
			name: "authentication failure",
			authenticator: &mockAuthenticator{
				err: errors.New(errors.CodeUnauthorized, "UNAUTHORIZED", "invalid token"),
			},
			transport: &authMockTransporter{
				kind:      transport.KindHTTP,
				operation: "/api/test",
			},
			wantErr:       true,
			wantErrCode:   errors.CodeUnauthorized,
			wantUserInCtx: false,
		},
		{
			name: "skip path",
			authenticator: &mockAuthenticator{
				err: errors.New(errors.CodeUnauthorized, "UNAUTHORIZED", "invalid token"),
			},
			skipPaths: []string{"/health"},
			transport: &authMockTransporter{
				kind:      transport.KindHTTP,
				operation: "/health",
			},
			wantErr:       false,
			wantUserInCtx: false,
		},
		{
			name: "no authenticator configured",
			transport: &authMockTransporter{
				kind:      transport.KindHTTP,
				operation: "/api/test",
			},
			wantErr:       true,
			wantErrCode:   errors.CodeInternal,
			wantUserInCtx: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build options
			var opts []AuthOption
			if tt.authenticator != nil {
				opts = append(opts, WithAuthenticator(tt.authenticator))
			}
			if tt.skipPaths != nil {
				opts = append(opts, WithSkipPaths(tt.skipPaths))
			}

			// Create middleware
			middleware := Auth(opts...)

			// Create handler that checks context
			handlerCalled := false
			handler := middleware(func(ctx context.Context, req any) (any, error) {
				handlerCalled = true
				user := UserFromContext(ctx)
				if tt.wantUserInCtx && user == nil {
					t.Error("expected user in context but got nil")
				}
				if !tt.wantUserInCtx && user != nil {
					t.Errorf("expected no user in context but got %v", user)
				}
				return "ok", nil
			})

			// Create context with transport
			ctx := context.Background()
			if tt.transport != nil {
				ctx = transport.NewContext(ctx, tt.transport)
			}

			// Execute handler
			_, err := handler(ctx, nil)

			// Check error
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				} else {
					if e, ok := err.(*errors.Error); ok {
						if int(e.Code) != tt.wantErrCode {
							t.Errorf("expected error code %d but got %d", tt.wantErrCode, e.Code)
						}
					} else {
						t.Errorf("expected *errors.Error but got %T", err)
					}
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got %v", err)
				}
				if !handlerCalled {
					t.Error("handler was not called")
				}
			}
		})
	}
}

func TestRequireRoles(t *testing.T) {
	tests := []struct {
		name        string
		user        *UserInfo
		require     []string
		wantErr     bool
		wantErrCode int
	}{
		{
			name:    "user has required role",
			user:    &UserInfo{ID: "1", Name: "test", Roles: []string{"admin"}},
			require: []string{"admin"},
			wantErr: false,
		},
		{
			name:    "user has one of required roles",
			user:    &UserInfo{ID: "1", Name: "test", Roles: []string{"moderator"}},
			require: []string{"admin", "moderator"},
			wantErr: false,
		},
		{
			name:        "user lacks required role",
			user:        &UserInfo{ID: "1", Name: "test", Roles: []string{"user"}},
			require:     []string{"admin"},
			wantErr:     true,
			wantErrCode: errors.CodeForbidden,
		},
		{
			name:        "no user in context",
			user:        nil,
			require:     []string{"admin"},
			wantErr:     true,
			wantErrCode: errors.CodeUnauthorized,
		},
		{
			name:        "user has no roles",
			user:        &UserInfo{ID: "1", Name: "test", Roles: []string{}},
			require:     []string{"admin"},
			wantErr:     true,
			wantErrCode: errors.CodeForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create middleware
			middleware := RequireRoles(tt.require...)

			// Create handler
			handlerCalled := false
			handler := middleware(func(ctx context.Context, req any) (any, error) {
				handlerCalled = true
				return "ok", nil
			})

			// Create context with user
			ctx := context.Background()
			if tt.user != nil {
				ctx = NewContextWithUser(ctx, tt.user)
			}

			// Execute handler
			_, err := handler(ctx, nil)

			// Check error
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				} else {
					if e, ok := err.(*errors.Error); ok {
						if int(e.Code) != tt.wantErrCode {
							t.Errorf("expected error code %d but got %d", tt.wantErrCode, e.Code)
						}
					} else {
						t.Errorf("expected *errors.Error but got %T", err)
					}
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got %v", err)
				}
				if !handlerCalled {
					t.Error("handler was not called")
				}
			}
		})
	}
}

func TestRequireRolesWithChecker(t *testing.T) {
	tests := []struct {
		name        string
		user        *UserInfo
		require     []string
		checker     RoleChecker
		wantErr     bool
		wantErrCode int
	}{
		{
			name:    "default checker - user has role",
			user:    &UserInfo{ID: "1", Name: "test", Roles: []string{"admin"}},
			require: []string{"admin"},
			checker: NewDefaultRoleChecker(),
			wantErr: false,
		},
		{
			name:        "default checker - user lacks role",
			user:        &UserInfo{ID: "1", Name: "test", Roles: []string{"user"}},
			require:     []string{"admin"},
			checker:     NewDefaultRoleChecker(),
			wantErr:     true,
			wantErrCode: errors.CodeForbidden,
		},
		{
			name:    "custom checker - always allow",
			user:    &UserInfo{ID: "1", Name: "test", Roles: []string{}},
			require: []string{"admin"},
			checker: &alwaysAllowChecker{},
			wantErr: false,
		},
		{
			name:        "custom checker - always deny",
			user:        &UserInfo{ID: "1", Name: "test", Roles: []string{"admin"}},
			require:     []string{"admin"},
			checker:     &alwaysDenyChecker{},
			wantErr:     true,
			wantErrCode: errors.CodeForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create middleware
			middleware := RequireRolesWithChecker(tt.checker, tt.require...)

			// Create handler
			handlerCalled := false
			handler := middleware(func(ctx context.Context, req any) (any, error) {
				handlerCalled = true
				return "ok", nil
			})

			// Create context with user
			ctx := context.Background()
			if tt.user != nil {
				ctx = NewContextWithUser(ctx, tt.user)
			}

			// Execute handler
			_, err := handler(ctx, nil)

			// Check error
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				} else {
					if e, ok := err.(*errors.Error); ok {
						if int(e.Code) != tt.wantErrCode {
							t.Errorf("expected error code %d but got %d", tt.wantErrCode, e.Code)
						}
					} else {
						t.Errorf("expected *errors.Error but got %T", err)
					}
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got %v", err)
				}
				if !handlerCalled {
					t.Error("handler was not called")
				}
			}
		})
	}
}

func TestUserFromContext(t *testing.T) {
	t.Run("user in context", func(t *testing.T) {
		user := &UserInfo{ID: "1", Name: "test", Roles: []string{"user"}}
		ctx := NewContextWithUser(context.Background(), user)
		got := UserFromContext(ctx)
		if got == nil {
			t.Error("expected user but got nil")
		}
		if got.ID != user.ID {
			t.Errorf("expected user ID %s but got %s", user.ID, got.ID)
		}
	})

	t.Run("no user in context", func(t *testing.T) {
		ctx := context.Background()
		got := UserFromContext(ctx)
		if got != nil {
			t.Errorf("expected nil but got %v", got)
		}
	})
}

func TestDefaultRoleChecker(t *testing.T) {
	checker := NewDefaultRoleChecker()

	tests := []struct {
		name          string
		user          *UserInfo
		requiredRoles []string
		want          bool
	}{
		{
			name:          "has role",
			user:          &UserInfo{Roles: []string{"admin"}},
			requiredRoles: []string{"admin"},
			want:          true,
		},
		{
			name:          "has one of roles",
			user:          &UserInfo{Roles: []string{"user", "moderator"}},
			requiredRoles: []string{"admin", "moderator"},
			want:          true,
		},
		{
			name:          "lacks all roles",
			user:          &UserInfo{Roles: []string{"user"}},
			requiredRoles: []string{"admin", "superadmin"},
			want:          false,
		},
		{
			name:          "no roles",
			user:          &UserInfo{Roles: []string{}},
			requiredRoles: []string{"admin"},
			want:          false,
		},
		{
			name:          "empty required roles",
			user:          &UserInfo{Roles: []string{"admin"}},
			requiredRoles: []string{},
			want:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checker.HasRole(tt.user, tt.requiredRoles)
			if got != tt.want {
				t.Errorf("HasRole() = %v, want %v", got, tt.want)
			}
		})
	}
}

// alwaysAllowChecker is a test RoleChecker that always allows access.
type alwaysAllowChecker struct{}

func (c *alwaysAllowChecker) HasRole(user *UserInfo, requiredRoles []string) bool {
	return true
}

// alwaysDenyChecker is a test RoleChecker that always denies access.
type alwaysDenyChecker struct{}

func (c *alwaysDenyChecker) HasRole(user *UserInfo, requiredRoles []string) bool {
	return false
}
