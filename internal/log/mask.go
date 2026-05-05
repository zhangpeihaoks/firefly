// Package log provides structured logging for the Firefly framework.
package log

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
)

// MaskConfig is the configuration for log masking.
type MaskConfig struct {
	// Fields is a list of field names that should be masked.
	// Field names are case-insensitive.
	Fields []string
	// Patterns is a list of regex patterns to match and mask sensitive data.
	Patterns []*regexp.Regexp
	// Replacement is the string to replace sensitive data with.
	// Default is "******"
	Replacement string
	// EnableDefaultFields enables masking of default sensitive fields.
	// Default fields: password, token, secret, key, api_key, apikey, auth, credential
	EnableDefaultFields bool
}

// defaultSensitiveFields contains default field names that are considered sensitive.
var defaultSensitiveFields = []string{
	"password", "passwd", "pwd",
	"token", "access_token", "refresh_token",
	"secret", "secret_key", "client_secret",
	"key", "api_key", "apikey", "api-key",
	"auth", "authorization",
	"credential", "credentials",
	"private_key", "private-key",
	"session_id", "sessionid",
	"ssn", "social_security",
	"credit_card", "card_number", "cvv",
}

// defaultMaskPatterns contains default regex patterns for detecting sensitive data.
var defaultMaskPatterns = []*regexp.Regexp{
	// Email addresses (partially mask)
	regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`),
	// Credit card numbers (mask all but last 4 digits)
	regexp.MustCompile(`\b\d{4}[-\s]?\d{4}[-\s]?\d{4}[-\s]?\d{4}\b`),
	// Phone numbers
	regexp.MustCompile(`\b\d{3}[-.]?\d{3}[-.]?\d{4}\b`),
	// JWT tokens
	regexp.MustCompile(`eyJ[a-zA-Z0-9_-]*\.eyJ[a-zA-Z0-9_-]*\.[a-zA-Z0-9_-]*`),
	// AWS keys
	regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`),
	// Generic API keys (common patterns)
	regexp.MustCompile(`(?i)(api[_-]?key|apikey)[=:]\s*['"]?[a-zA-Z0-9_-]{8,}['"]?`),
	// Generic tokens
	regexp.MustCompile(`(?i)(bearer|token)[=:]\s*['"]?[a-zA-Z0-9_-]{10,}['"]?`),
}

// DefaultMaskConfig returns a MaskConfig with default settings.
func DefaultMaskConfig() *MaskConfig {
	return &MaskConfig{
		Fields:              defaultSensitiveFields,
		Patterns:            defaultMaskPatterns,
		Replacement:         "******",
		EnableDefaultFields: true,
	}
}

// Mask creates a log masker with the given configuration.
func Mask(opts ...MaskOption) *Masker {
	cfg := DefaultMaskConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	// Build field lookup map for O(1) access
	fieldMap := make(map[string]bool)
	for _, f := range cfg.Fields {
		fieldMap[strings.ToLower(f)] = true
	}

	return &Masker{
		fieldMap:    fieldMap,
		patterns:    cfg.Patterns,
		replacement: cfg.Replacement,
	}
}

// MaskOption is a function that configures the mask config.
type MaskOption func(*MaskConfig)

// WithMaskFields sets the list of field names to mask.
func WithMaskFields(fields []string) MaskOption {
	return func(c *MaskConfig) {
		c.Fields = fields
	}
}

// WithMaskPatterns sets the list of regex patterns to mask.
func WithMaskPatterns(patterns []*regexp.Regexp) MaskOption {
	return func(c *MaskConfig) {
		c.Patterns = patterns
	}
}

// WithMaskReplacement sets the replacement string for masked values.
func WithMaskReplacement(replacement string) MaskOption {
	return func(c *MaskConfig) {
		c.Replacement = replacement
	}
}

// WithoutDefaultFields disables default sensitive field detection.
func WithoutDefaultFields() MaskOption {
	return func(c *MaskConfig) {
		c.EnableDefaultFields = false
	}
}

// Masker is the log masker that masks sensitive information.
type Masker struct {
	fieldMap    map[string]bool
	patterns    []*regexp.Regexp
	replacement string
}

// IsSensitiveField checks if a field name is considered sensitive.
func (m *Masker) IsSensitiveField(fieldName string) bool {
	return m.fieldMap[strings.ToLower(fieldName)]
}

// MaskValue masks a value if it matches any of the configured patterns.
func (m *Masker) MaskValue(value string) string {
	if value == "" {
		return value
	}

	result := value
	for _, pattern := range m.patterns {
		result = pattern.ReplaceAllString(result, m.replacement)
	}

	return result
}

// MaskMap recursively masks sensitive fields in a map.
// It handles nested maps and slices.
func (m *Masker) MaskMap(data map[string]any) map[string]any {
	if data == nil {
		return nil
	}

	result := make(map[string]any, len(data))
	for k, v := range data {
		if m.IsSensitiveField(k) {
			result[k] = m.replacement
		} else {
			result[k] = m.maskValue(v)
		}
	}

	return result
}

// maskValue recursively masks sensitive values.
func (m *Masker) maskValue(v any) any {
	switch val := v.(type) {
	case string:
		return m.MaskValue(val)
	case map[string]any:
		return m.MaskMap(val)
	case []any:
		result := make([]any, len(val))
		for i, item := range val {
			result[i] = m.maskValue(item)
		}
		return result
	case map[string]string:
		result := make(map[string]string, len(val))
		for k, v := range val {
			if m.IsSensitiveField(k) {
				result[k] = m.replacement
			} else {
				result[k] = m.MaskValue(v)
			}
		}
		return result
	default:
		return val
	}
}

// MaskJSON masks sensitive information in JSON strings.
func (m *Masker) MaskJSON(data string) (string, error) {
	if data == "" {
		return data, nil
	}

	// Try to parse as JSON
	var raw any
	if err := json.Unmarshal([]byte(data), &raw); err != nil {
		// Not valid JSON, apply pattern-based masking
		return m.MaskValue(data), nil
	}

	// Mask the parsed data
	masked := m.maskValue(raw)

	// Marshal back to JSON
	maskedJSON, err := json.Marshal(masked)
	if err != nil {
		return "", fmt.Errorf("failed to marshal masked data: %w", err)
	}

	return string(maskedJSON), nil
}

// MaskAttrs masks sensitive attributes in slog.Attr slice.
func (m *Masker) MaskAttrs(attrs []slog.Attr) []slog.Attr {
	if attrs == nil {
		return nil
	}

	result := make([]slog.Attr, len(attrs))
	for i, attr := range attrs {
		if attr.Key == "" {
			continue
		}

		if m.IsSensitiveField(attr.Key) {
			result[i] = slog.String(attr.Key, m.replacement)
		} else {
			result[i] = m.maskAttr(attr)
		}
	}

	return result
}

// maskAttr recursively masks sensitive slog.Attr values.
func (m *Masker) maskAttr(attr slog.Attr) slog.Attr {
	switch val := attr.Value.Any().(type) {
	case string:
		return slog.String(attr.Key, m.MaskValue(val))
	case map[string]any:
		return slog.Any(attr.Key, m.MaskMap(val))
	case []any:
		masked := make([]any, len(val))
		for i, item := range val {
			masked[i] = m.maskValue(item)
		}
		return slog.Any(attr.Key, masked)
	case map[string]string:
		result := make(map[string]string, len(val))
		for k, v := range val {
			if m.IsSensitiveField(k) {
				result[k] = m.replacement
			} else {
				result[k] = m.MaskValue(v)
			}
		}
		return slog.Any(attr.Key, result)
	default:
		return attr
	}
}

// GlobalMask returns the global masker instance.
// If no global masker is set, it returns a default masker.
var globalMasker *Masker

// SetGlobalMask sets the global masker instance.
func SetGlobalMask(m *Masker) {
	globalMasker = m
}

// GetGlobalMask returns the global masker instance.
func GetGlobalMask() *Masker {
	if globalMasker == nil {
		return DefaultMaskConfig().CreateMasker()
	}
	return globalMasker
}

// CreateMasker creates a masker from the config.
func (c *MaskConfig) CreateMasker() *Masker {
	fieldMap := make(map[string]bool)
	for _, f := range c.Fields {
		fieldMap[strings.ToLower(f)] = true
	}

	patterns := c.Patterns
	if c.EnableDefaultFields && len(patterns) == 0 {
		patterns = defaultMaskPatterns
	}

	replacement := c.Replacement
	if replacement == "" {
		replacement = "******"
	}

	return &Masker{
		fieldMap:    fieldMap,
		patterns:    patterns,
		replacement: replacement,
	}
}

// MaskString is a convenience function that masks sensitive data in a string.
// It applies pattern-based masking to the input string.
func MaskString(data string) string {
	return GetGlobalMask().MaskValue(data)
}

// MaskMap is a convenience function that masks sensitive fields in a map.
func MaskMap(data map[string]any) map[string]any {
	return GetGlobalMask().MaskMap(data)
}

// MaskJSON is a convenience function that masks sensitive data in JSON.
func MaskJSON(data string) (string, error) {
	return GetGlobalMask().MaskJSON(data)
}
