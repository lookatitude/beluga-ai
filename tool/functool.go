package tool

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/lookatitude/beluga-ai/internal/jsonutil"
)

// FuncTool wraps a typed Go function as a Tool. It uses generics to provide
// type-safe input handling and automatically generates a JSON Schema from the
// input struct's tags using internal/jsonutil.GenerateSchema.
//
// I must be a struct type with json, description, required, and other tags
// recognized by jsonutil.GenerateSchema.
type FuncTool[I any] struct {
	name        string
	description string
	fn          func(ctx context.Context, input I) (*Result, error)
	schema      map[string]any
}

// NewFuncTool creates a new FuncTool that wraps fn as a Tool. The JSON Schema
// for the input type I is generated once at construction time.
//
// Example:
//
//	type SearchInput struct {
//	    Query string `json:"query" description:"Search query" required:"true"`
//	    Limit int    `json:"limit" description:"Max results" default:"10"`
//	}
//
//	search := tool.NewFuncTool("search", "Search the web",
//	    func(ctx context.Context, input SearchInput) (*tool.Result, error) {
//	        return tool.TextResult("results for: " + input.Query), nil
//	    },
//	)
func NewFuncTool[I any](name, description string, fn func(ctx context.Context, input I) (*Result, error)) *FuncTool[I] {
	var zero I
	return &FuncTool[I]{
		name:        name,
		description: description,
		fn:          fn,
		schema:      jsonutil.GenerateSchema(zero),
	}
}

// Name returns the tool's name.
func (f *FuncTool[I]) Name() string { return f.name }

// Description returns the tool's description.
func (f *FuncTool[I]) Description() string { return f.description }

// InputSchema returns the auto-generated JSON Schema for the input type I.
func (f *FuncTool[I]) InputSchema() map[string]any { return f.schema }

// Execute deserializes the input map into the typed struct I and calls the
// wrapped function. The input map is marshaled to JSON and then unmarshaled
// into the target type to leverage Go's json tag-based mapping.
func (f *FuncTool[I]) Execute(ctx context.Context, input map[string]any) (*Result, error) {
	// Marshal map to JSON, then unmarshal into the typed struct.
	data, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("tool %s: failed to marshal input: %w", f.name, err)
	}

	var typed I
	if err := json.Unmarshal(data, &typed); err != nil {
		return nil, fmt.Errorf("tool %s: failed to unmarshal input: %w", f.name, err)
	}

	return f.fn(ctx, typed)
}
