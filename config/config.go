package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
)

// Load reads a configuration file at path and unmarshals it into T.
// The file format is detected by extension: ".json" uses encoding/json.
// After unmarshaling, defaults are applied for zero-valued fields and
// the result is validated.
func Load[T any](path string) (T, error) {
	var cfg T

	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("config: read %s: %w", path, err)
	}

	ext := strings.ToLower(filepath.Ext(path))
	var providedKeys map[string]json.RawMessage
	switch ext {
	case ".json":
		if err := json.Unmarshal(data, &cfg); err != nil {
			return cfg, fmt.Errorf("config: unmarshal json: %w", err)
		}
		// Also decode into a map to track which keys were explicitly provided.
		_ = json.Unmarshal(data, &providedKeys)
	default:
		return cfg, fmt.Errorf("config: unsupported file extension %q (supported: .json)", ext)
	}

	// Validate required fields before applying defaults so that missing
	// required values are caught even when a default tag exists.
	if err := validateRequired(&cfg, providedKeys); err != nil {
		return cfg, err
	}

	applyDefaultsSelective(&cfg, providedKeys)

	if err := Validate(&cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// LoadFromEnv creates a configuration of type T populated entirely from
// environment variables. Each exported struct field maps to an env var
// named PREFIX_FIELDNAME (uppercase, underscores replace nested dots).
// Defaults from struct tags are applied for unset variables, and the
// result is validated.
func LoadFromEnv[T any](prefix string) (T, error) {
	var cfg T
	applyDefaults(&cfg)

	if err := mergeEnvFields(reflect.ValueOf(&cfg).Elem(), strings.ToUpper(prefix)); err != nil {
		return cfg, err
	}

	if err := Validate(&cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// MergeEnv overrides fields in cfg from environment variables with the given
// prefix. Only fields with a corresponding set environment variable are
// overridden; unset variables leave the field unchanged.
// cfg must be a pointer to a struct.
func MergeEnv(cfg any, prefix string) error {
	v := reflect.ValueOf(cfg)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("config: MergeEnv requires a pointer to a struct, got %T", cfg)
	}
	return mergeEnvFields(v.Elem(), strings.ToUpper(prefix))
}

// Validate checks cfg against struct tag constraints. Supported tags:
//   - required:"true" — field must not be zero-valued
//   - min:"N" — numeric fields must be >= N
//   - max:"N" — numeric fields must be <= N
//
// cfg must be a pointer to a struct or a struct.
func Validate(cfg any) error {
	v := reflect.ValueOf(cfg)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("config: Validate requires a struct or pointer to struct, got %s", v.Kind())
	}
	return validateStruct(v)
}

// applyDefaultsSelective sets zero-valued struct fields to their `default`
// tag values, but skips fields that were explicitly provided in the JSON
// (even if their value is the zero value).
func applyDefaultsSelective(cfg any, provided map[string]json.RawMessage) {
	v := reflect.ValueOf(cfg)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return
	}
	applyDefaultsSelectiveValue(v, provided)
}

func applyDefaultsSelectiveValue(v reflect.Value, provided map[string]json.RawMessage) {
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		sf := t.Field(i)

		if !sf.IsExported() {
			continue
		}

		// Determine JSON key for this field.
		jsonKey := sf.Name
		if tag := sf.Tag.Get("json"); tag != "" {
			parts := strings.Split(tag, ",")
			if parts[0] != "" && parts[0] != "-" {
				jsonKey = parts[0]
			}
		}

		if field.Kind() == reflect.Struct {
			// For nested structs, check if the key was provided and pass its sub-map.
			var nested map[string]json.RawMessage
			if raw, ok := provided[jsonKey]; ok {
				_ = json.Unmarshal(raw, &nested)
			}
			applyDefaultsSelectiveValue(field, nested)
			continue
		}

		// Skip if the field was explicitly provided in JSON.
		if _, ok := provided[jsonKey]; ok {
			continue
		}

		if !field.IsZero() {
			continue
		}

		def := sf.Tag.Get("default")
		if def == "" {
			continue
		}

		setFieldFromString(field, def)
	}
}

// applyDefaults sets zero-valued struct fields to their `default` tag values.
func applyDefaults(cfg any) {
	v := reflect.ValueOf(cfg)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return
	}
	applyDefaultsToValue(v)
}

func applyDefaultsToValue(v reflect.Value) {
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		sf := t.Field(i)

		if !sf.IsExported() {
			continue
		}

		// Recurse into embedded or nested structs.
		if field.Kind() == reflect.Struct {
			applyDefaultsToValue(field)
			continue
		}

		// Only apply default to zero-valued fields.
		if !field.IsZero() {
			continue
		}

		def := sf.Tag.Get("default")
		if def == "" {
			continue
		}

		setFieldFromString(field, def)
	}
}

// validateRequired checks only `required:"true"` tags on a struct, before
// defaults are applied. This ensures that a missing required field is
// reported even when it has a default tag. The provided map tracks which
// JSON keys were explicitly present in the input; a field is considered
// missing only when it is zero-valued AND was not explicitly provided.
func validateRequired(cfg any, provided map[string]json.RawMessage) error {
	v := reflect.ValueOf(cfg)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil
	}
	return validateRequiredStruct(v, provided)
}

func validateRequiredStruct(v reflect.Value, provided map[string]json.RawMessage) error {
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		sf := t.Field(i)

		if !sf.IsExported() {
			continue
		}

		// Determine JSON key for this field.
		jsonKey := sf.Name
		if jsonTag := sf.Tag.Get("json"); jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "" && parts[0] != "-" {
				jsonKey = parts[0]
			}
		}

		if field.Kind() == reflect.Struct {
			var nested map[string]json.RawMessage
			if raw, ok := provided[jsonKey]; ok {
				_ = json.Unmarshal(raw, &nested)
			}
			if err := validateRequiredStruct(field, nested); err != nil {
				return err
			}
			continue
		}

		// A field is required and missing if: it has required:"true", its
		// value is zero, AND it was not explicitly provided in the JSON.
		if sf.Tag.Get("required") == "true" && field.IsZero() {
			if _, ok := provided[jsonKey]; !ok {
				return &ValidationError{
					Field:   jsonKey,
					Message: "field is required",
				}
			}
		}
	}
	return nil
}

// validateStruct validates a struct value against its field tags.
func validateStruct(v reflect.Value) error {
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		sf := t.Field(i)

		if !sf.IsExported() {
			continue
		}

		// Recurse into nested structs.
		if field.Kind() == reflect.Struct {
			if err := validateStruct(field); err != nil {
				return err
			}
			continue
		}

		fieldName := sf.Name
		if jsonTag := sf.Tag.Get("json"); jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "" && parts[0] != "-" {
				fieldName = parts[0]
			}
		}

		// Check required.
		if sf.Tag.Get("required") == "true" && field.IsZero() {
			return &ValidationError{
				Field:   fieldName,
				Message: "field is required",
			}
		}

		// Check min/max for numeric types.
		if err := validateNumericBounds(field, sf, fieldName); err != nil {
			return err
		}
	}
	return nil
}

// validateNumericBounds checks min/max constraints on numeric fields.
func validateNumericBounds(field reflect.Value, sf reflect.StructField, fieldName string) error {
	minStr := sf.Tag.Get("min")
	maxStr := sf.Tag.Get("max")

	if minStr == "" && maxStr == "" {
		return nil
	}

	var val float64
	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		val = float64(field.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		val = float64(field.Uint())
	case reflect.Float32, reflect.Float64:
		val = field.Float()
	default:
		return nil // min/max only apply to numeric types
	}

	if minStr != "" {
		minVal, err := strconv.ParseFloat(minStr, 64)
		if err != nil {
			return fmt.Errorf("config: invalid min tag %q on field %s: %w", minStr, fieldName, err)
		}
		if val < minVal {
			return &ValidationError{
				Field:   fieldName,
				Message: fmt.Sprintf("value %v is less than minimum %v", val, minVal),
			}
		}
	}

	if maxStr != "" {
		maxVal, err := strconv.ParseFloat(maxStr, 64)
		if err != nil {
			return fmt.Errorf("config: invalid max tag %q on field %s: %w", maxStr, fieldName, err)
		}
		if val > maxVal {
			return &ValidationError{
				Field:   fieldName,
				Message: fmt.Sprintf("value %v is greater than maximum %v", val, maxVal),
			}
		}
	}

	return nil
}

// mergeEnvFields reads environment variables matching PREFIX_FIELDNAME and
// sets them on the struct fields.
func mergeEnvFields(v reflect.Value, prefix string) error {
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		sf := t.Field(i)

		if !sf.IsExported() {
			continue
		}

		envName := prefix + "_" + toEnvName(sf.Name)

		// Recurse into nested structs with extended prefix.
		if field.Kind() == reflect.Struct {
			if err := mergeEnvFields(field, envName); err != nil {
				return err
			}
			continue
		}

		envVal, ok := os.LookupEnv(envName)
		if !ok {
			continue
		}

		if !setFieldFromString(field, envVal) {
			return fmt.Errorf("config: cannot set field %s (type %s) from env var %s=%q",
				sf.Name, field.Type(), envName, envVal)
		}
	}
	return nil
}

// setFieldFromString sets a reflect.Value from a string representation.
// Returns true on success, false if the type is unsupported.
func setFieldFromString(field reflect.Value, s string) bool {
	if !field.CanSet() {
		return false
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(s)
	case reflect.Bool:
		b, err := strconv.ParseBool(s)
		if err != nil {
			return false
		}
		field.SetBool(b)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return false
		}
		field.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return false
		}
		field.SetUint(n)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return false
		}
		field.SetFloat(f)
	default:
		return false
	}
	return true
}

// toEnvName converts a Go field name (e.g. "BaseURL") to an environment
// variable suffix (e.g. "BASE_URL") by inserting underscores before uppercase
// letters and uppercasing the result. Consecutive uppercase letters (acronyms
// like URL, API, ID) are kept together.
func toEnvName(name string) string {
	runes := []rune(name)
	var b strings.Builder
	for i, r := range runes {
		if i > 0 && r >= 'A' && r <= 'Z' {
			prev := runes[i-1]
			// Insert underscore before an uppercase letter if:
			// - previous char is lowercase (camelCase boundary), OR
			// - previous char is uppercase AND next char is lowercase (end of acronym, e.g. "URL" -> keep, "URLs" -> "UR_Ls" isn't desired, but "URLParser" -> "URL_PARSER")
			if prev >= 'a' && prev <= 'z' {
				b.WriteByte('_')
			} else if prev >= 'A' && prev <= 'Z' && i+1 < len(runes) && runes[i+1] >= 'a' && runes[i+1] <= 'z' {
				b.WriteByte('_')
			}
		}
		b.WriteRune(r)
	}
	return strings.ToUpper(b.String())
}

// ValidationError represents a validation failure for a specific field.
type ValidationError struct {
	// Field is the name of the field that failed validation.
	Field string

	// Message describes the validation failure.
	Message string
}

// Error returns a human-readable description of the validation failure.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("config: validation failed for %q: %s", e.Field, e.Message)
}
