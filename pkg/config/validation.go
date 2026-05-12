// Package config provides configuration management for the Firefly framework.
// This file implements configuration validation logic.
package config

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"
)

// ValidationError represents a configuration validation error.
type ValidationError struct {
	Field   string // The field that failed validation
	Message string // The validation error message
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("config validation error: %s: %s", e.Field, e.Message)
	}
	return fmt.Sprintf("config validation error: %s", e.Message)
}

// ValidationErrors is a collection of validation errors.
type ValidationErrors []*ValidationError

// Error implements the error interface.
func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return "no validation errors"
	}
	var msgs []string
	for _, err := range e {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// Add adds a new validation error.
func (e *ValidationErrors) Add(field, message string) {
	*e = append(*e, &ValidationError{Field: field, Message: message})
}

// HasErrors returns true if there are any validation errors.
func (e ValidationErrors) HasErrors() bool {
	return len(e) > 0
}

// Validator is the interface for configuration validators.
type Validator interface {
	// Validate validates the configuration and returns any errors.
	Validate() ValidationErrors
}

// Validate validates a configuration structure using struct tags.
// Supported tags:
//   - required: field must not be empty
//   - min: minimum value for numeric types or minimum length for strings
//   - max: maximum value for numeric types or maximum length for strings
//   - oneof: value must be one of the specified values (comma-separated)
//   - regex: string must match the specified regex pattern
//   - duration: string must be a valid duration (e.g., "30s", "5m")
//
// Example:
//
//	type ServerConfig struct {
//	    Host     string `validate:"required"`
//	    Port     int    `validate:"required,min=1,max=65535"`
//	    Protocol string `validate:"required,oneof=http,https"`
//	}
func Validate(v any) error {
	return ValidateWithPrefix(v, "")
}

// ValidateWithPrefix validates a configuration structure with a field name prefix.
func ValidateWithPrefix(v any, prefix string) error {
	var errors ValidationErrors
	validateValue(reflect.ValueOf(v), prefix, &errors)
	if errors.HasErrors() {
		return errors
	}
	return nil
}

// validateValue validates a reflect.Value.
func validateValue(v reflect.Value, prefix string, errors *ValidationErrors) {
	// Handle nil values
	if !v.IsValid() {
		return
	}

	// Dereference pointers
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return
		}
		v = v.Elem()
	}

	// Check if the type implements Validator interface
	if v.CanInterface() {
		if validator, ok := v.Interface().(Validator); ok {
			for _, err := range validator.Validate() {
				if prefix != "" && err.Field != "" {
					err.Field = prefix + "." + err.Field
				}
				*errors = append(*errors, err)
			}
		}
	}

	switch v.Kind() {
	case reflect.Struct:
		validateStruct(v, prefix, errors)
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			elemPrefix := fmt.Sprintf("%s[%d]", prefix, i)
			validateValue(v.Index(i), elemPrefix, errors)
		}
	case reflect.Map:
		for _, key := range v.MapKeys() {
			elemPrefix := fmt.Sprintf("%s[%v]", prefix, key.Interface())
			validateValue(v.MapIndex(key), elemPrefix, errors)
		}
	}
}

// validateStruct validates a struct using struct tags.
func validateStruct(v reflect.Value, prefix string, errors *ValidationErrors) {
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Build field name with prefix
		fieldName := field.Name
		if prefix != "" {
			fieldName = prefix + "." + fieldName
		}

		// Get validate tag
		tag := field.Tag.Get("validate")
		if tag != "" {
			validateField(field.Name, fieldName, fieldValue, tag, errors)
		}

		// Recursively validate nested structs
		validateValue(fieldValue, fieldName, errors)
	}
}

// validateField validates a single field based on its validate tag.
func validateField(name, fullName string, v reflect.Value, tag string, errors *ValidationErrors) {
	rules := parseValidationRules(tag)

	for _, rule := range rules {
		rule = strings.TrimSpace(rule)
		if rule == "" {
			continue
		}

		// Parse rule name and value
		var ruleName, ruleValue string
		if idx := strings.Index(rule, "="); idx != -1 {
			ruleName = rule[:idx]
			ruleValue = rule[idx+1:]
		} else {
			ruleName = rule
		}

		// Apply validation rule
		switch ruleName {
		case "required":
			validateRequired(name, fullName, v, errors)
		case "min":
			validateMin(name, fullName, v, ruleValue, errors)
		case "max":
			validateMax(name, fullName, v, ruleValue, errors)
		case "oneof":
			validateOneOf(name, fullName, v, ruleValue, errors)
		case "regex":
			validateRegex(name, fullName, v, ruleValue, errors)
		case "duration":
			validateDuration(name, fullName, v, errors)
		case "url":
			validateURL(name, fullName, v, errors)
		case "email":
			validateEmail(name, fullName, v, errors)
		}
	}
}

// parseValidationRules parses validation rules from a tag string.
// It handles rules like "oneof=http,https,grpc" correctly by not splitting on commas inside rule values.
func parseValidationRules(tag string) []string {
	var rules []string
	var current strings.Builder
	i := 0

	for i < len(tag) {
		r := rune(tag[i])

		if r == ',' {
			// Check if the current rule is oneof, min, max, or regex (rules that have values with commas)
			currentStr := current.String()
			if strings.HasPrefix(currentStr, "oneof=") {
				// For oneof, we need to include everything after the = until we see a new rule
				current.WriteRune(r)
			} else {
				// This is a rule separator
				if current.Len() > 0 {
					rules = append(rules, current.String())
					current.Reset()
				}
			}
		} else {
			current.WriteRune(r)
		}
		i++
	}

	if current.Len() > 0 {
		rules = append(rules, current.String())
	}

	return rules
}

// isValidRuleName checks if a string is a valid validation rule name.
func isValidRuleName(s string) bool {
	validRules := map[string]bool{
		"required": true,
		"min":      true,
		"max":      true,
		"oneof":    true,
		"regex":    true,
		"duration": true,
		"url":      true,
		"email":    true,
	}
	trimmed := strings.TrimSpace(s)
	return validRules[trimmed]
}

// validateRequired checks if a field is not empty.
func validateRequired(name, fullName string, v reflect.Value, errors *ValidationErrors) {
	if isEmpty(v) {
		errors.Add(fullName, "is required")
	}
}

// isEmpty checks if a value is empty.
func isEmpty(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		return v.String() == ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	case reflect.Slice, reflect.Array, reflect.Map:
		return v.Len() == 0
	default:
		return false
	}
}

// validateMin validates minimum value/length.
func validateMin(name, fullName string, v reflect.Value, ruleValue string, errors *ValidationErrors) {
	var min int
	if _, err := fmt.Sscanf(ruleValue, "%d", &min); err != nil {
		return
	}

	switch v.Kind() {
	case reflect.String:
		if len(v.String()) < min {
			errors.Add(fullName, fmt.Sprintf("length must be at least %d", min))
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if v.Int() < int64(min) {
			errors.Add(fullName, fmt.Sprintf("must be at least %d", min))
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if v.Uint() < uint64(min) {
			errors.Add(fullName, fmt.Sprintf("must be at least %d", min))
		}
	case reflect.Slice, reflect.Array:
		if v.Len() < min {
			errors.Add(fullName, fmt.Sprintf("must contain at least %d elements", min))
		}
	}
}

// validateMax validates maximum value/length.
func validateMax(name, fullName string, v reflect.Value, ruleValue string, errors *ValidationErrors) {
	var max int
	if _, err := fmt.Sscanf(ruleValue, "%d", &max); err != nil {
		return
	}

	switch v.Kind() {
	case reflect.String:
		if len(v.String()) > max {
			errors.Add(fullName, fmt.Sprintf("length must be at most %d", max))
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if v.Int() > int64(max) {
			errors.Add(fullName, fmt.Sprintf("must be at most %d", max))
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if v.Uint() > uint64(max) {
			errors.Add(fullName, fmt.Sprintf("must be at most %d", max))
		}
	case reflect.Slice, reflect.Array:
		if v.Len() > max {
			errors.Add(fullName, fmt.Sprintf("must contain at most %d elements", max))
		}
	}
}

// validateOneOf validates that a value is one of the specified values.
func validateOneOf(name, fullName string, v reflect.Value, ruleValue string, errors *ValidationErrors) {
	allowed := strings.Split(ruleValue, ",")
	value := fmt.Sprintf("%v", v.Interface())

	for _, a := range allowed {
		if value == strings.TrimSpace(a) {
			return
		}
	}

	errors.Add(fullName, fmt.Sprintf("must be one of: %s", ruleValue))
}

// validateRegex validates that a string matches a regex pattern.
func validateRegex(name, fullName string, v reflect.Value, pattern string, errors *ValidationErrors) {
	if v.Kind() != reflect.String {
		return
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		errors.Add(fullName, fmt.Sprintf("invalid regex pattern: %s", pattern))
		return
	}

	if !re.MatchString(v.String()) {
		errors.Add(fullName, fmt.Sprintf("must match pattern: %s", pattern))
	}
}

// validateDuration validates that a string is a valid duration.
func validateDuration(name, fullName string, v reflect.Value, errors *ValidationErrors) {
	if v.Kind() == reflect.String {
		if _, err := time.ParseDuration(v.String()); err != nil {
			errors.Add(fullName, "must be a valid duration (e.g., '30s', '5m', '1h')")
		}
	} else if v.Type() == reflect.TypeOf(time.Duration(0)) {
		// time.Duration is already valid as int64
		return
	}
}

// validateURL validates that a string is a valid URL.
func validateURL(name, fullName string, v reflect.Value, errors *ValidationErrors) {
	if v.Kind() != reflect.String {
		return
	}

	urlPattern := `^https?://[^\s/$.?#].[^\s]*$`
	re := regexp.MustCompile(urlPattern)
	if !re.MatchString(v.String()) {
		errors.Add(fullName, "must be a valid URL")
	}
}

// validateEmail validates that a string is a valid email.
func validateEmail(name, fullName string, v reflect.Value, errors *ValidationErrors) {
	if v.Kind() != reflect.String {
		return
	}

	emailPattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	re := regexp.MustCompile(emailPattern)
	if !re.MatchString(v.String()) {
		errors.Add(fullName, "must be a valid email address")
	}
}

// MustValidate validates a configuration and panics if validation fails.
// This is useful for configuration that must be valid at startup.
func MustValidate(v any) {
	if err := Validate(v); err != nil {
		panic(fmt.Sprintf("configuration validation failed: %v", err))
	}
}
