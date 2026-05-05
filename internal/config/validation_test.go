// Package config provides configuration management for the Firefly framework.
package config

import (
	"testing"
	"time"
)

func TestValidate_Required(t *testing.T) {
	type TestStruct struct {
		Name string `validate:"required"`
	}

	tests := []struct {
		name    string
		input   TestStruct
		wantErr bool
	}{
		{
			name:    "empty string fails required",
			input:   TestStruct{Name: ""},
			wantErr: true,
		},
		{
			name:    "non-empty string passes required",
			input:   TestStruct{Name: "test"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidate_MinMax(t *testing.T) {
	type TestStruct struct {
		Port    int    `validate:"min=1,max=65535"`
		Name    string `validate:"min=3,max=50"`
		Numbers []int  `validate:"min=1,max=10"`
	}

	tests := []struct {
		name    string
		input   TestStruct
		wantErr bool
	}{
		{
			name:    "valid values",
			input:   TestStruct{Port: 8080, Name: "myapp", Numbers: []int{1, 2, 3}},
			wantErr: false,
		},
		{
			name:    "port below min",
			input:   TestStruct{Port: 0, Name: "myapp", Numbers: []int{1}},
			wantErr: true,
		},
		{
			name:    "port above max",
			input:   TestStruct{Port: 70000, Name: "myapp", Numbers: []int{1}},
			wantErr: true,
		},
		{
			name:    "name too short",
			input:   TestStruct{Port: 8080, Name: "ab", Numbers: []int{1}},
			wantErr: true,
		},
		{
			name:    "empty slice fails min",
			input:   TestStruct{Port: 8080, Name: "myapp", Numbers: []int{}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidate_OneOf(t *testing.T) {
	type TestStruct struct {
		Protocol string `validate:"oneof=http,https,grpc"`
	}

	tests := []struct {
		name    string
		input   TestStruct
		wantErr bool
	}{
		{
			name:    "valid value http",
			input:   TestStruct{Protocol: "http"},
			wantErr: false,
		},
		{
			name:    "valid value https",
			input:   TestStruct{Protocol: "https"},
			wantErr: false,
		},
		{
			name:    "valid value grpc",
			input:   TestStruct{Protocol: "grpc"},
			wantErr: false,
		},
		{
			name:    "invalid value",
			input:   TestStruct{Protocol: "ftp"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidate_Regex(t *testing.T) {
	type TestStruct struct {
		ID string `validate:"regex=^[A-Z]{3}[0-9]{3}$"`
	}

	tests := []struct {
		name    string
		input   TestStruct
		wantErr bool
	}{
		{
			name:    "valid id",
			input:   TestStruct{ID: "ABC123"},
			wantErr: false,
		},
		{
			name:    "invalid id",
			input:   TestStruct{ID: "abc123"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidate_Duration(t *testing.T) {
	type TestStruct struct {
		Timeout string `validate:"duration"`
	}

	tests := []struct {
		name    string
		input   TestStruct
		wantErr bool
	}{
		{
			name:    "valid duration seconds",
			input:   TestStruct{Timeout: "30s"},
			wantErr: false,
		},
		{
			name:    "valid duration minutes",
			input:   TestStruct{Timeout: "5m"},
			wantErr: false,
		},
		{
			name:    "valid duration hours",
			input:   TestStruct{Timeout: "1h30m"},
			wantErr: false,
		},
		{
			name:    "invalid duration",
			input:   TestStruct{Timeout: "invalid"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidate_NestedStruct(t *testing.T) {
	type DatabaseConfig struct {
		Driver string `validate:"required"`
		DSN    string `validate:"required"`
	}

	type ServerConfig struct {
		Name     string         `validate:"required"`
		Database DatabaseConfig `validate:"required"`
	}

	tests := []struct {
		name    string
		input   ServerConfig
		wantErr bool
	}{
		{
			name: "valid nested struct",
			input: ServerConfig{
				Name: "myapp",
				Database: DatabaseConfig{
					Driver: "mysql",
					DSN:    "localhost:3306",
				},
			},
			wantErr: false,
		},
		{
			name: "missing nested field",
			input: ServerConfig{
				Name: "myapp",
				Database: DatabaseConfig{
					Driver: "mysql",
					DSN:    "",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidate_PointerStruct(t *testing.T) {
	type DatabaseConfig struct {
		Driver string `validate:"required"`
	}

	type ServerConfig struct {
		Name     string          `validate:"required"`
		Database *DatabaseConfig `validate:"required"`
	}

	tests := []struct {
		name    string
		input   ServerConfig
		wantErr bool
	}{
		{
			name: "valid pointer struct",
			input: ServerConfig{
				Name: "myapp",
				Database: &DatabaseConfig{
					Driver: "mysql",
				},
			},
			wantErr: false,
		},
		{
			name: "nil pointer",
			input: ServerConfig{
				Name:     "myapp",
				Database: nil,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidationErrors(t *testing.T) {
	var errors ValidationErrors

	if errors.HasErrors() {
		t.Error("empty ValidationErrors should not have errors")
	}

	errors.Add("field1", "is required")
	errors.Add("field2", "must be positive")

	if !errors.HasErrors() {
		t.Error("ValidationErrors should have errors after Add")
	}

	if len(errors) != 2 {
		t.Errorf("ValidationErrors length = %d, want 2", len(errors))
	}

	errMsg := errors.Error()
	if errMsg == "" {
		t.Error("Error() returned empty string")
	}
}

func TestBootstrap_Validate(t *testing.T) {
	tests := []struct {
		name    string
		input   *Bootstrap
		wantErr bool
	}{
		{
			name:    "valid bootstrap",
			input:   DefaultBootstrap(),
			wantErr: false,
		},
		{
			name: "missing name",
			input: &Bootstrap{
				Name: "",
				HTTP: DefaultHTTPConfig(),
			},
			wantErr: true,
		},
		{
			name: "invalid log level",
			input: &Bootstrap{
				Name: "test",
				Log: &LogConfig{
					Level: "invalid",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if (err.HasErrors()) != tt.wantErr {
				t.Errorf("Validate() hasErrors = %v, wantErr %v", err.HasErrors(), tt.wantErr)
			}
		})
	}
}

func TestHTTPConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		input   *HTTPConfig
		wantErr bool
	}{
		{
			name:    "valid config",
			input:   DefaultHTTPConfig(),
			wantErr: false,
		},
		{
			name: "missing address",
			input: &HTTPConfig{
				Network: "tcp",
				Address: "",
			},
			wantErr: true,
		},
		{
			name: "TLS enabled without cert",
			input: &HTTPConfig{
				Address: ":8080",
				TLS: &TLSConfig{
					Enabled:  true,
					CertFile: "",
					KeyFile:  "",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if (err.HasErrors()) != tt.wantErr {
				t.Errorf("Validate() hasErrors = %v, wantErr %v", err.HasErrors(), tt.wantErr)
			}
		})
	}
}

func TestDatabaseConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		input   *DatabaseConfig
		wantErr bool
	}{
		{
			name: "valid config",
			input: &DatabaseConfig{
				Driver: "mysql",
				DSN:    "localhost:3306",
			},
			wantErr: false,
		},
		{
			name: "missing driver",
			input: &DatabaseConfig{
				DSN: "localhost:3306",
			},
			wantErr: true,
		},
		{
			name: "missing dsn",
			input: &DatabaseConfig{
				Driver: "mysql",
			},
			wantErr: true,
		},
		{
			name: "invalid driver",
			input: &DatabaseConfig{
				Driver: "oracle",
				DSN:    "localhost:1521",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if (err.HasErrors()) != tt.wantErr {
				t.Errorf("Validate() hasErrors = %v, wantErr %v", err.HasErrors(), tt.wantErr)
			}
		})
	}
}

func TestTracingConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		input   *TracingConfig
		wantErr bool
	}{
		{
			name:    "disabled tracing is valid",
			input:   DefaultTracingConfig(),
			wantErr: false,
		},
		{
			name: "enabled tracing without endpoint",
			input: &TracingConfig{
				Enabled:  true,
				Endpoint: "",
			},
			wantErr: true,
		},
		{
			name: "enabled tracing with endpoint",
			input: &TracingConfig{
				Enabled:      true,
				Endpoint:     "http://localhost:14268/api/traces",
				SamplerRatio: 1.0,
			},
			wantErr: false,
		},
		{
			name: "invalid sampler ratio",
			input: &TracingConfig{
				Enabled:      false,
				SamplerRatio: 2.0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if (err.HasErrors()) != tt.wantErr {
				t.Errorf("Validate() hasErrors = %v, wantErr %v", err.HasErrors(), tt.wantErr)
			}
		})
	}
}

func TestMustValidate(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustValidate should panic on invalid config")
		}
	}()

	// This should panic
	MustValidate(&struct {
		Name string `validate:"required"`
	}{Name: ""})
}

func TestValidate_Email(t *testing.T) {
	type TestStruct struct {
		Email string `validate:"email"`
	}

	tests := []struct {
		name    string
		input   TestStruct
		wantErr bool
	}{
		{
			name:    "valid email",
			input:   TestStruct{Email: "test@example.com"},
			wantErr: false,
		},
		{
			name:    "invalid email no @",
			input:   TestStruct{Email: "testexample.com"},
			wantErr: true,
		},
		{
			name:    "invalid email no domain",
			input:   TestStruct{Email: "test@"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidate_URL(t *testing.T) {
	type TestStruct struct {
		URL string `validate:"url"`
	}

	tests := []struct {
		name    string
		input   TestStruct
		wantErr bool
	}{
		{
			name:    "valid http url",
			input:   TestStruct{URL: "http://example.com"},
			wantErr: false,
		},
		{
			name:    "valid https url",
			input:   TestStruct{URL: "https://example.com/path"},
			wantErr: false,
		},
		{
			name:    "invalid url no protocol",
			input:   TestStruct{URL: "example.com"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidate_Slice(t *testing.T) {
	type TestStruct struct {
		Tags []string `validate:"min=1,max=5"`
	}

	tests := []struct {
		name    string
		input   TestStruct
		wantErr bool
	}{
		{
			name:    "valid slice",
			input:   TestStruct{Tags: []string{"api", "web"}},
			wantErr: false,
		},
		{
			name:    "empty slice fails min",
			input:   TestStruct{Tags: []string{}},
			wantErr: true,
		},
		{
			name:    "nil slice fails min",
			input:   TestStruct{Tags: nil},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidate_TimeDuration(t *testing.T) {
	type TestStruct struct {
		Timeout time.Duration `validate:"min=1"`
	}

	tests := []struct {
		name    string
		input   TestStruct
		wantErr bool
	}{
		{
			name:    "valid duration",
			input:   TestStruct{Timeout: 30 * time.Second},
			wantErr: false,
		},
		{
			name:    "zero duration fails min",
			input:   TestStruct{Timeout: 0},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestValidate_DescriptiveErrors verifies requirement 6.7:
// WHEN 配置加载失败时，THE Framework SHALL 返回描述性错误信息
func TestValidate_DescriptiveErrors(t *testing.T) {
	type TestConfig struct {
		Name     string `validate:"required"`
		Port     int    `validate:"required,min=1,max=65535"`
		Protocol string `validate:"oneof=http,https,grpc"`
		Email    string `validate:"email"`
		URL      string `validate:"url"`
	}

	t.Run("required field error is descriptive", func(t *testing.T) {
		err := Validate(TestConfig{Name: "", Port: 8080, Protocol: "http"})
		if err == nil {
			t.Fatal("expected error for missing required field")
		}
		errMsg := err.Error()
		// Error should contain field name and description
		if !contains(errMsg, "Name") || !contains(errMsg, "required") {
			t.Errorf("error message should be descriptive: %v", errMsg)
		}
	})

	t.Run("min validation error is descriptive", func(t *testing.T) {
		err := Validate(TestConfig{Name: "test", Port: 0, Protocol: "http"})
		if err == nil {
			t.Fatal("expected error for invalid port")
		}
		errMsg := err.Error()
		// Error should contain field name, value info, and constraint
		if !contains(errMsg, "Port") || !contains(errMsg, "at least") {
			t.Errorf("error message should be descriptive: %v", errMsg)
		}
	})

	t.Run("max validation error is descriptive", func(t *testing.T) {
		err := Validate(TestConfig{Name: "test", Port: 70000, Protocol: "http"})
		if err == nil {
			t.Fatal("expected error for port above max")
		}
		errMsg := err.Error()
		if !contains(errMsg, "Port") || !contains(errMsg, "at most") {
			t.Errorf("error message should be descriptive: %v", errMsg)
		}
	})

	t.Run("oneof validation error is descriptive", func(t *testing.T) {
		err := Validate(TestConfig{Name: "test", Port: 8080, Protocol: "ftp"})
		if err == nil {
			t.Fatal("expected error for invalid protocol")
		}
		errMsg := err.Error()
		if !contains(errMsg, "Protocol") || !contains(errMsg, "one of") {
			t.Errorf("error message should be descriptive: %v", errMsg)
		}
	})

	t.Run("email validation error is descriptive", func(t *testing.T) {
		err := Validate(TestConfig{Name: "test", Port: 8080, Protocol: "http", Email: "invalid"})
		if err == nil {
			t.Fatal("expected error for invalid email")
		}
		errMsg := err.Error()
		if !contains(errMsg, "Email") || !contains(errMsg, "email") {
			t.Errorf("error message should be descriptive: %v", errMsg)
		}
	})

	t.Run("url validation error is descriptive", func(t *testing.T) {
		err := Validate(TestConfig{Name: "test", Port: 8080, Protocol: "http", URL: "not-a-url"})
		if err == nil {
			t.Fatal("expected error for invalid URL")
		}
		errMsg := err.Error()
		if !contains(errMsg, "URL") || !contains(errMsg, "URL") {
			t.Errorf("error message should be descriptive: %v", errMsg)
		}
	})

	t.Run("multiple validation errors are all descriptive", func(t *testing.T) {
		err := Validate(TestConfig{Name: "", Port: 0, Protocol: "ftp", Email: "bad", URL: "bad"})
		if err == nil {
			t.Fatal("expected error for multiple invalid fields")
		}
		errMsg := err.Error()
		// Should contain all field names
		if !contains(errMsg, "Name") || !contains(errMsg, "Port") ||
			!contains(errMsg, "Protocol") || !contains(errMsg, "Email") || !contains(errMsg, "URL") {
			t.Errorf("error message should contain all invalid fields: %v", errMsg)
		}
	})
}

// TestValidateWithPrefix_DescriptiveErrors verifies descriptive errors with prefix
func TestValidateWithPrefix_DescriptiveErrors(t *testing.T) {
	type DatabaseConfig struct {
		Host string `validate:"required"`
		Port int    `validate:"required,min=1,max=65535"`
	}

	type ServerConfig struct {
		Database DatabaseConfig `validate:"required"`
	}

	t.Run("nested field error includes path", func(t *testing.T) {
		cfg := ServerConfig{
			Database: DatabaseConfig{Host: "", Port: 8080},
		}
		err := ValidateWithPrefix(cfg, "config")
		if err == nil {
			t.Fatal("expected error for missing nested required field")
		}
		errMsg := err.Error()
		// Error should include the full path to the field
		if !contains(errMsg, "config.Database.Host") {
			t.Errorf("error should include full path: %v", errMsg)
		}
	})
}

// contains is a helper that checks if s contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
