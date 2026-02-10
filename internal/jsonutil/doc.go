// Package jsonutil provides JSON utilities for the Beluga AI framework,
// including JSON Schema generation from Go struct types via reflection.
//
// This is an internal package and is not part of the public API. It is used by
// the tool system and structured output packages to automatically generate
// JSON Schema definitions from Go types.
//
// # Schema Generation
//
// The [GenerateSchema] function produces a JSON Schema (as a map) from a Go
// value by inspecting its type with reflection. It conforms to JSON Schema
// Draft-07 and handles structs, slices, maps, pointers, and all primitive
// types recursively.
//
// Supported struct tags:
//
//   - json:"name"        — sets the property name (use "-" to skip)
//   - description:"..."  — sets the property description
//   - required:"true"    — marks the property as required
//   - enum:"a,b,c"       — constrains the value to the listed options
//   - default:"..."      — sets the default value
//   - minimum:"N"        — sets the minimum numeric value
//   - maximum:"N"        — sets the maximum numeric value
//
// Example:
//
//	type SearchInput struct {
//	    Query string `json:"query" description:"Search query" required:"true"`
//	    Limit int    `json:"limit" description:"Max results" default:"10" minimum:"1" maximum:"100"`
//	}
//	schema := jsonutil.GenerateSchema(SearchInput{})
//	// schema is a map[string]any representing the JSON Schema
package jsonutil
