// Package gofunc provides a tool implementation that wraps an existing Go function.
package gofunc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	"github.com/lookatitude/beluga-ai/pkg/core"
)

// GoFunctionTool wraps an arbitrary Go function, allowing it to be used as a Tool
// within the Beluga-ai framework, callable by agents.
// It uses reflection to call the underlying function.
//
// Note: This implementation makes specific assumptions about the function signature
// (context.Context, map[string]any) -> (string, error). A more robust version
// might inspect the function signature more deeply, handle different input/output types,
// automatically generate the InputSchema from struct tags, or use code generation.
type GoFunctionTool struct {
	tools.BaseTool
	Def      tools.ToolDefinition // Store definition directly
	function reflect.Value        // Holds the reflected value of the wrapped Go function.
}

// NewGoFunctionTool creates a new GoFunctionTool.
//
// Parameters:
//   - name: The unique name for the tool.
//   - description: A description of what the tool does, for the LLM to understand.
//   - inputSchemaJSON: A JSON string defining the expected input arguments schema (e.g., JSON Schema).
//   - fn: The Go function to wrap. It is expected to have a signature compatible with
//     `func(ctx context.Context, args map[string]any) (string, error)`.
//     Variations might be possible with more advanced reflection, but this is the assumed default.
//
// Returns:
//   - A pointer to the created GoFunctionTool or an error if initialization fails
//     (e.g., fn is not a function, schema is invalid JSON).
func NewGoFunctionTool(name, description, inputSchemaJSON string, fn any) (*GoFunctionTool, error) {
	fnVal := reflect.ValueOf(fn)
	if fnVal.Kind() != reflect.Func {
		return nil, fmt.Errorf("provided fn is not a function, but %T", fn)
	}

	fnType := reflect.TypeOf(fn)
	if fnType.NumIn() != 2 {
		return nil, errors.New("function must have exactly 2 inputs: context.Context and map[string]any")
	}
	if fnType.NumOut() != 2 {
		return nil, errors.New("function must have exactly 2 outputs: any and error")
	}

	if fnType.In(0) != reflect.TypeOf((*context.Context)(nil)).Elem() {
		return nil, errors.New("first input must be context.Context")
	}
	if fnType.In(1) != reflect.TypeOf(map[string]any(nil)) {
		return nil, errors.New("second input must be map[string]any")
	}

	if !fnType.Out(1).Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		return nil, errors.New("second output must be error")
	}

	var inputSchema map[string]any // Changed to map[string]any
	if inputSchemaJSON != "" {
		if err := json.Unmarshal([]byte(inputSchemaJSON), &inputSchema); err != nil {
			return nil, fmt.Errorf("invalid input schema JSON for tool \"%s\": %w", name, err)
		}
	}

	return &GoFunctionTool{
		Def: tools.ToolDefinition{
			Name:        name,
			Description: description,
			InputSchema: inputSchema,
		},
		function: fnVal,
	}, nil
}

// Definition returns the tool's definition.
func (gft *GoFunctionTool) Definition() tools.ToolDefinition {
	return gft.Def
}

// Description returns the tool's description.
func (gft *GoFunctionTool) Description() string {
	return gft.Definition().Description
}

// Name returns the tool's name.
func (gft *GoFunctionTool) Name() string {
	return gft.Definition().Name
}

// Execute implements the tools.Tool interface.
// It calls the wrapped Go function using reflection with the provided arguments.
// Corrected input type to any.
func (gft *GoFunctionTool) Execute(ctx context.Context, input any) (any, error) {
	// Expect input to be map[string]any for function call
	args, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("input to GoFunctionTool \"%s\" must be map[string]any, got %T", gft.Def.Name, input)
	}

	// Prepare arguments for the reflection call based on the assumed signature.
	in := []reflect.Value{
		reflect.ValueOf(ctx),
		reflect.ValueOf(args),
	}

	// Call the wrapped function.
	out := gft.function.Call(in)

	// Process the return values, expecting (string, error).
	if len(out) != 2 {
		return "", fmt.Errorf("wrapped function for tool \"%s\" did not return 2 values as expected", gft.Def.Name)
	}

	// Extract the result (assuming string for now, matching previous logic).
	var result any
	if !out[0].IsNil() {
		result = out[0].Interface()
		// Keep previous string check for compatibility if needed, but Execute returns any
		/*
		   resultStr, ok := out[0].Interface().(string)
		   if !ok {
		       return "", fmt.Errorf("wrapped function for tool \"%s\" first return value is %T, not string", gft.Def.Name, out[0].Interface())
		   }
		   result = resultStr
		*/
	}

	// Extract the error result.
	var err error
	if !out[1].IsNil() {
		var ok bool
		err, ok = out[1].Interface().(error)
		if !ok {
			return "", fmt.Errorf("wrapped function for tool \"%s\" second return value is %T, not error", gft.Def.Name, out[1].Interface())
		}
	}

	return result, err
}

// Invoke implements the core.Runnable interface for GoFunctionTool.
// It directly calls the Execute method.
func (gft *GoFunctionTool) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	// Execute now takes any, so no type assertion needed here
	return gft.Execute(ctx, input)
}

// Batch implements the core.Runnable interface for GoFunctionTool.
// It calls Execute sequentially for each input.
// TODO: Consider if the underlying function supports batching for potential optimization.
// Batch implements the tools.Tool interface.
func (gft *GoFunctionTool) Batch(ctx context.Context, inputs []any) ([]any, error) {
	results := make([]any, len(inputs))
	for i, input := range inputs {
		result, err := gft.Execute(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("error processing batch item %d: %w", i, err)
		}
		results[i] = result
	}
	return results, nil
}

// Run implements the core.Runnable Batch method with options.
func (gft *GoFunctionTool) Run(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
	return gft.Batch(ctx, inputs) // Options are ignored for now
}

// Stream implements the core.Runnable interface for GoFunctionTool.
// Since function calls are typically not streaming, it executes the function
// and returns the result immediately on a channel.
func (gft *GoFunctionTool) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
	resultChan := make(chan any, 1)
	go func() {
		defer close(resultChan)
		output, err := gft.Invoke(ctx, input, options...)
		if err != nil {
			resultChan <- err // Send error on the channel
		} else {
			resultChan <- output // Send result on the channel
		}
	}()
	return resultChan, nil // No immediate error, result/error comes via channel.
}

// Compile-time checks to ensure implementation satisfies interfaces.
// Make sure interfaces are correctly implemented.
var _ tools.Tool = (*GoFunctionTool)(nil)

// Define a custom interface that matches what we've implemented.
type batcherWithOptions interface {
	Run(ctx context.Context, inputs []any, options ...core.Option) ([]any, error)
	Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error)
	Invoke(ctx context.Context, input any, options ...core.Option) (any, error)
}

var _ batcherWithOptions = (*GoFunctionTool)(nil)
