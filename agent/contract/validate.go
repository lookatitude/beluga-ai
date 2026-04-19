package contract

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/schema"
)

// ValidateInput validates the given input value against the contract's input
// schema. A nil input schema is treated as a wildcard and always passes.
// Returns a *core.Error with code core.ErrInvalidInput on validation failure.
func ValidateInput(_ context.Context, c *schema.Contract, input any) error {
	if c == nil || c.InputSchema == nil {
		return nil
	}
	errs := validateValue(input, c.InputSchema, c.Strict, "input")
	if len(errs) > 0 {
		return core.NewError(
			"contract.validate_input",
			core.ErrInvalidInput,
			fmt.Sprintf("contract %q input validation failed: %s", c.Name, strings.Join(errs, "; ")),
			nil,
		)
	}
	return nil
}

// ValidateOutput validates the given output value against the contract's output
// schema. A nil output schema is treated as a wildcard and always passes.
// Returns a *core.Error with code core.ErrInvalidInput on validation failure.
func ValidateOutput(_ context.Context, c *schema.Contract, output any) error {
	if c == nil || c.OutputSchema == nil {
		return nil
	}
	errs := validateValue(output, c.OutputSchema, c.Strict, "output")
	if len(errs) > 0 {
		return core.NewError(
			"contract.validate_output",
			core.ErrInvalidInput,
			fmt.Sprintf("contract %q output validation failed: %s", c.Name, strings.Join(errs, "; ")),
			nil,
		)
	}
	return nil
}

// validateValue validates a value against a JSON Schema. Returns a list of
// human-readable error strings. An empty list means the value is valid.
func validateValue(value any, sch map[string]any, strict bool, path string) []string {
	// Normalize value: if it's a JSON string, try to unmarshal it.
	value = normalizeValue(value)

	var errs []string

	schemaType, _ := sch["type"].(string)

	switch schemaType {
	case "object":
		errs = append(errs, validateObject(value, sch, strict, path)...)
	case "array":
		errs = append(errs, validateArray(value, sch, strict, path)...)
	case "string":
		if _, ok := value.(string); !ok {
			errs = append(errs, fmt.Sprintf("%s: expected string, got %T", path, value))
		} else {
			errs = append(errs, validateEnum(value, sch, path)...)
		}
	case "number":
		if !isNumber(value) {
			errs = append(errs, fmt.Sprintf("%s: expected number, got %T", path, value))
		}
	case "integer":
		if !isInteger(value) {
			errs = append(errs, fmt.Sprintf("%s: expected integer, got %T", path, value))
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			errs = append(errs, fmt.Sprintf("%s: expected boolean, got %T", path, value))
		}
	case "":
		// No type constraint — wildcard, always passes.
	}

	return errs
}

// normalizeValue converts JSON strings and json.RawMessage to Go maps/slices.
func normalizeValue(value any) any {
	switch v := value.(type) {
	case string:
		trimmed := strings.TrimSpace(v)
		if len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[') {
			var parsed any
			if err := json.Unmarshal([]byte(v), &parsed); err == nil {
				return parsed
			}
		}
		return v
	case json.RawMessage:
		var parsed any
		if err := json.Unmarshal(v, &parsed); err == nil {
			return parsed
		}
		return v
	case []byte:
		var parsed any
		if err := json.Unmarshal(v, &parsed); err == nil {
			return parsed
		}
		return v
	default:
		return v
	}
}

// validateObject validates a value expected to be a JSON object.
func validateObject(value any, sch map[string]any, strict bool, path string) []string {
	obj, ok := toMap(value)
	if !ok {
		return []string{fmt.Sprintf("%s: expected object, got %T", path, value)}
	}

	var errs []string

	// Check required fields.
	if required, ok := sch["required"].([]any); ok {
		for _, r := range required {
			name, _ := r.(string)
			if name == "" {
				continue
			}
			if _, exists := obj[name]; !exists {
				errs = append(errs, fmt.Sprintf("%s: missing required field %q", path, name))
			}
		}
	}

	// Validate properties.
	properties, _ := sch["properties"].(map[string]any)
	for propName, propSchema := range properties {
		propSch, ok := propSchema.(map[string]any)
		if !ok {
			continue
		}
		val, exists := obj[propName]
		if !exists {
			continue // Not required and not present — skip.
		}
		errs = append(errs, validateValue(val, propSch, strict, path+"."+propName)...)
	}

	// Check additionalProperties when strict.
	apVal, apExists := sch["additionalProperties"]
	rejectAdditional := strict
	if apExists {
		if apBool, ok := apVal.(bool); ok {
			rejectAdditional = !apBool
		}
	}

	if rejectAdditional && properties != nil {
		for key := range obj {
			if _, defined := properties[key]; !defined {
				errs = append(errs, fmt.Sprintf("%s: unexpected property %q", path, key))
			}
		}
	}

	return errs
}

// validateArray validates a value expected to be a JSON array.
func validateArray(value any, sch map[string]any, strict bool, path string) []string {
	arr, ok := toSlice(value)
	if !ok {
		return []string{fmt.Sprintf("%s: expected array, got %T", path, value)}
	}

	var errs []string
	if itemsSchema, ok := sch["items"].(map[string]any); ok {
		for i, item := range arr {
			errs = append(errs, validateValue(item, itemsSchema, strict, fmt.Sprintf("%s[%d]", path, i))...)
		}
	}
	return errs
}

// validateEnum checks if the value matches one of the allowed enum values.
func validateEnum(value any, sch map[string]any, path string) []string {
	enumVals, ok := sch["enum"].([]any)
	if !ok || len(enumVals) == 0 {
		return nil
	}

	for _, allowed := range enumVals {
		if fmt.Sprintf("%v", value) == fmt.Sprintf("%v", allowed) {
			return nil
		}
	}
	return []string{fmt.Sprintf("%s: value %v not in enum %v", path, value, enumVals)}
}

// toMap attempts to convert a value to map[string]any.
func toMap(v any) (map[string]any, bool) {
	switch m := v.(type) {
	case map[string]any:
		return m, true
	default:
		// Try JSON round-trip for struct types.
		data, err := json.Marshal(v)
		if err != nil {
			return nil, false
		}
		var result map[string]any
		if err := json.Unmarshal(data, &result); err != nil {
			return nil, false
		}
		return result, true
	}
}

// toSlice attempts to convert a value to []any.
func toSlice(v any) ([]any, bool) {
	switch s := v.(type) {
	case []any:
		return s, true
	default:
		data, err := json.Marshal(v)
		if err != nil {
			return nil, false
		}
		var result []any
		if err := json.Unmarshal(data, &result); err != nil {
			return nil, false
		}
		return result, true
	}
}

// isNumber reports whether v is a numeric type.
func isNumber(v any) bool {
	switch v.(type) {
	case float64, float32, int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64, json.Number:
		return true
	default:
		return false
	}
}

// isInteger reports whether v is an integer type. JSON numbers that are
// whole numbers (e.g., float64(42.0)) are accepted as integers.
func isInteger(v any) bool {
	switch n := v.(type) {
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64:
		return true
	case float64:
		return n == float64(int64(n))
	case float32:
		return n == float32(int32(n))
	case json.Number:
		_, err := n.Int64()
		return err == nil
	default:
		return false
	}
}
