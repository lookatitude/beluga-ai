package jsonutil

import (
	"reflect"
	"strings"
)

// GenerateSchema produces a JSON Schema (as a map) from the given Go value.
// It inspects the type of v using reflection and builds a schema description
// that conforms to JSON Schema Draft-07.
//
// Supported struct tags:
//   - json:"name"       — sets the property name (use "-" to skip)
//   - description:"..."  — sets the property description
//   - required:"true"   — marks the property as required
//   - enum:"a,b,c"      — constrains the value to the listed options
//   - default:"..."     — sets the default value (string representation)
//   - minimum:"N"       — sets the minimum numeric value
//   - maximum:"N"       — sets the maximum numeric value
//
// Nested structs, slices, maps, and pointers are handled recursively.
func GenerateSchema(v any) map[string]any {
	t := reflect.TypeOf(v)
	if t == nil {
		return map[string]any{"type": "object"}
	}
	return generateType(t)
}

func generateType(t reflect.Type) map[string]any {
	// Unwrap pointer types.
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.Struct:
		return generateStruct(t)
	case reflect.Slice, reflect.Array:
		return generateSlice(t)
	case reflect.Map:
		return generateMap(t)
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return map[string]any{"type": "integer"}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]any{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}
	case reflect.Interface:
		return map[string]any{}
	default:
		return map[string]any{"type": "string"}
	}
}

func generateStruct(t reflect.Type) map[string]any {
	properties := make(map[string]any)
	var required []string

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		name, skip := fieldName(field)
		if skip {
			continue
		}

		prop := generateType(field.Type)
		applyFieldTags(field, prop)

		if field.Tag.Get("required") == "true" {
			required = append(required, name)
		}

		properties[name] = prop
	}

	schema := map[string]any{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

// fieldName determines the JSON property name from a struct field.
// Returns the name and true if the field should be skipped.
func fieldName(field reflect.StructField) (string, bool) {
	name := field.Name
	if jsonTag := field.Tag.Get("json"); jsonTag != "" {
		parts := strings.SplitN(jsonTag, ",", 2)
		if parts[0] == "-" {
			return "", true
		}
		if parts[0] != "" {
			name = parts[0]
		}
	}
	return name, false
}

// applyFieldTags reads struct tags and sets corresponding schema properties.
func applyFieldTags(field reflect.StructField, prop map[string]any) {
	if desc := field.Tag.Get("description"); desc != "" {
		prop["description"] = desc
	}
	if enum := field.Tag.Get("enum"); enum != "" {
		values := strings.Split(enum, ",")
		trimmed := make([]any, len(values))
		for j, v := range values {
			trimmed[j] = strings.TrimSpace(v)
		}
		prop["enum"] = trimmed
	}
	if def := field.Tag.Get("default"); def != "" {
		prop["default"] = def
	}
	if min := field.Tag.Get("minimum"); min != "" {
		prop["minimum"] = min
	}
	if max := field.Tag.Get("maximum"); max != "" {
		prop["maximum"] = max
	}
}

func generateSlice(t reflect.Type) map[string]any {
	return map[string]any{
		"type":  "array",
		"items": generateType(t.Elem()),
	}
}

func generateMap(t reflect.Type) map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": generateType(t.Elem()),
	}
}
