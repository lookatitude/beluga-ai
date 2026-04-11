package mcp

import (
	"encoding/json"
	"fmt"

	"github.com/lookatitude/beluga-ai/core"
)

// StructuredToolInfo extends ToolInfo with an optional output schema for
// structured output validation. This corresponds to the November 2025 MCP
// spec addition for tool output schemas.
type StructuredToolInfo struct {
	// Name is the tool's unique identifier.
	Name string `json:"name"`

	// Description is a human-readable description of the tool.
	Description string `json:"description,omitempty"`

	// InputSchema is a JSON Schema describing the tool's input parameters.
	InputSchema map[string]any `json:"inputSchema"`

	// OutputSchema is an optional JSON Schema describing the tool's output
	// structure. When present, tool outputs should conform to this schema.
	OutputSchema map[string]any `json:"outputSchema,omitempty"`
}

// ToToolInfo converts a StructuredToolInfo to a basic ToolInfo, dropping the
// output schema.
func (s StructuredToolInfo) ToToolInfo() ToolInfo {
	return ToolInfo{
		Name:        s.Name,
		Description: s.Description,
		InputSchema: s.InputSchema,
	}
}

// ValidateToolOutput validates a tool's output against the provided output
// schema. The schema is a JSON Schema (as a map). This performs structural
// validation covering type checking, required properties, and enum constraints.
//
// If schema is nil or empty, validation always succeeds.
func ValidateToolOutput(output any, schema map[string]any) error {
	if len(schema) == 0 {
		return nil
	}

	// Marshal output to JSON and back to get a normalized map representation.
	data, err := json.Marshal(output)
	if err != nil {
		return core.Errorf(core.ErrInvalidInput, "mcp/structured: marshal output: %w", err)
	}

	var normalized any
	if err := json.Unmarshal(data, &normalized); err != nil {
		return core.Errorf(core.ErrInvalidInput, "mcp/structured: unmarshal output: %w", err)
	}

	return validateValue(normalized, schema, "")
}

// validateValue performs recursive JSON Schema validation on a value.
func validateValue(value any, schema map[string]any, path string) error {
	if len(schema) == 0 {
		return nil
	}

	schemaType, _ := schema["type"].(string)
	if schemaType == "" {
		return nil
	}

	if err := validateType(value, schemaType, path); err != nil {
		return err
	}

	switch schemaType {
	case "object":
		return validateObject(value, schema, path)
	case "array":
		return validateArray(value, schema, path)
	case "string":
		return validateEnum(value, schema, path)
	}

	return nil
}

// validateType checks that the value matches the expected JSON Schema type.
func validateType(value any, schemaType string, path string) error {
	if value == nil {
		return core.Errorf(core.ErrInvalidInput, "mcp/structured: %s: expected %s, got null", pathOrRoot(path), schemaType)
	}

	switch schemaType {
	case "object":
		if _, ok := value.(map[string]any); !ok {
			return core.Errorf(core.ErrInvalidInput, "mcp/structured: %s: expected object, got %T", pathOrRoot(path), value)
		}
	case "array":
		if _, ok := value.([]any); !ok {
			return core.Errorf(core.ErrInvalidInput, "mcp/structured: %s: expected array, got %T", pathOrRoot(path), value)
		}
	case "string":
		if _, ok := value.(string); !ok {
			return core.Errorf(core.ErrInvalidInput, "mcp/structured: %s: expected string, got %T", pathOrRoot(path), value)
		}
	case "number":
		if _, ok := value.(float64); !ok {
			return core.Errorf(core.ErrInvalidInput, "mcp/structured: %s: expected number, got %T", pathOrRoot(path), value)
		}
	case "integer":
		v, ok := value.(float64)
		if !ok {
			return core.Errorf(core.ErrInvalidInput, "mcp/structured: %s: expected integer, got %T", pathOrRoot(path), value)
		}
		if v != float64(int64(v)) {
			return core.Errorf(core.ErrInvalidInput, "mcp/structured: %s: expected integer, got float", pathOrRoot(path))
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return core.Errorf(core.ErrInvalidInput, "mcp/structured: %s: expected boolean, got %T", pathOrRoot(path), value)
		}
	}

	return nil
}

// validateObject validates required properties and nested property schemas.
func validateObject(value any, schema map[string]any, path string) error {
	obj, ok := value.(map[string]any)
	if !ok {
		return nil
	}

	// Check required properties.
	if required, ok := schema["required"].([]any); ok {
		for _, r := range required {
			name, _ := r.(string)
			if name == "" {
				continue
			}
			if _, exists := obj[name]; !exists {
				return core.Errorf(core.ErrInvalidInput, "mcp/structured: %s: missing required property %q", pathOrRoot(path), name)
			}
		}
	}

	// Validate nested properties.
	if properties, ok := schema["properties"].(map[string]any); ok {
		for key, propSchema := range properties {
			propValue, exists := obj[key]
			if !exists {
				continue
			}
			ps, ok := propSchema.(map[string]any)
			if !ok {
				continue
			}
			childPath := key
			if path != "" {
				childPath = path + "." + key
			}
			if err := validateValue(propValue, ps, childPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateArray validates array items against the items schema.
func validateArray(value any, schema map[string]any, path string) error {
	arr, ok := value.([]any)
	if !ok {
		return nil
	}

	itemSchema, ok := schema["items"].(map[string]any)
	if !ok {
		return nil
	}

	for i, item := range arr {
		childPath := fmt.Sprintf("%s[%d]", pathOrRoot(path), i)
		if err := validateValue(item, itemSchema, childPath); err != nil {
			return err
		}
	}

	return nil
}

// validateEnum checks string values against an enum constraint.
func validateEnum(value any, schema map[string]any, path string) error {
	enumValues, ok := schema["enum"].([]any)
	if !ok {
		return nil
	}

	str, ok := value.(string)
	if !ok {
		return nil
	}

	for _, e := range enumValues {
		if s, ok := e.(string); ok && s == str {
			return nil
		}
	}

	return core.Errorf(core.ErrInvalidInput, "mcp/structured: %s: value %q not in enum", pathOrRoot(path), str)
}

// pathOrRoot returns "root" if path is empty, otherwise returns path.
func pathOrRoot(path string) string {
	if path == "" {
		return "root"
	}
	return path
}
