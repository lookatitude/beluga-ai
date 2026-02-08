package tool

import (
	"context"
	"testing"
)

type calcInput struct {
	Expression string `json:"expression" description:"Math expression" required:"true"`
	Precision  int    `json:"precision" description:"Decimal precision" default:"2"`
}

func TestNewFuncTool_NameAndDescription(t *testing.T) {
	ft := NewFuncTool("calc", "Calculate math",
		func(ctx context.Context, input calcInput) (*Result, error) {
			return TextResult("42"), nil
		},
	)

	if ft.Name() != "calc" {
		t.Errorf("Name() = %q, want %q", ft.Name(), "calc")
	}
	if ft.Description() != "Calculate math" {
		t.Errorf("Description() = %q, want %q", ft.Description(), "Calculate math")
	}
}

func TestNewFuncTool_SchemaGeneration(t *testing.T) {
	ft := NewFuncTool("calc", "Calculate math",
		func(ctx context.Context, input calcInput) (*Result, error) {
			return TextResult("42"), nil
		},
	)

	s := ft.InputSchema()
	if s == nil {
		t.Fatal("InputSchema() returned nil")
	}
	if s["type"] != "object" {
		t.Errorf("schema type = %v, want %q", s["type"], "object")
	}

	props, ok := s["properties"].(map[string]any)
	if !ok {
		t.Fatalf("properties is not map[string]any: %T", s["properties"])
	}

	// Check expression field exists.
	exprProp, ok := props["expression"]
	if !ok {
		t.Fatal("schema missing 'expression' property")
	}
	exprMap, ok := exprProp.(map[string]any)
	if !ok {
		t.Fatalf("expression property is not map[string]any: %T", exprProp)
	}
	if exprMap["type"] != "string" {
		t.Errorf("expression type = %v, want %q", exprMap["type"], "string")
	}
	if exprMap["description"] != "Math expression" {
		t.Errorf("expression description = %v, want %q", exprMap["description"], "Math expression")
	}

	// Check required fields.
	req, ok := s["required"]
	if !ok {
		t.Fatal("schema missing 'required' field")
	}
	reqSlice, ok := req.([]string)
	if !ok {
		t.Fatalf("required is not []string: %T", req)
	}
	found := false
	for _, r := range reqSlice {
		if r == "expression" {
			found = true
		}
	}
	if !found {
		t.Error("'expression' not found in required fields")
	}
}

func TestFuncTool_Execute_Success(t *testing.T) {
	ft := NewFuncTool("calc", "Calculate",
		func(ctx context.Context, input calcInput) (*Result, error) {
			return TextResult("result: " + input.Expression), nil
		},
	)

	result, err := ft.Execute(context.Background(), map[string]any{
		"expression": "2+2",
		"precision":  3,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.IsError {
		t.Error("result should not be error")
	}
}

func TestFuncTool_Execute_InvalidInput(t *testing.T) {
	ft := NewFuncTool("calc", "Calculate",
		func(ctx context.Context, input calcInput) (*Result, error) {
			return TextResult("ok"), nil
		},
	)

	// Pass invalid type for precision field.
	_, err := ft.Execute(context.Background(), map[string]any{
		"expression": "2+2",
		"precision":  "not-a-number", // should fail json unmarshal into int
	})
	if err == nil {
		t.Fatal("expected error for invalid input type")
	}
}

func TestFuncTool_Execute_ContextPropagation(t *testing.T) {
	type ctxKey struct{}
	ft := NewFuncTool("ctx-test", "Test context",
		func(ctx context.Context, input struct{}) (*Result, error) {
			val, ok := ctx.Value(ctxKey{}).(string)
			if !ok || val != "test-value" {
				t.Error("context value not propagated")
			}
			return TextResult("ok"), nil
		},
	)

	ctx := context.WithValue(context.Background(), ctxKey{}, "test-value")
	_, err := ft.Execute(ctx, map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

type emptyInput struct{}

func TestFuncTool_Execute_EmptyInput(t *testing.T) {
	ft := NewFuncTool("empty", "No-input tool",
		func(ctx context.Context, input emptyInput) (*Result, error) {
			return TextResult("done"), nil
		},
	)

	result, err := ft.Execute(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}
