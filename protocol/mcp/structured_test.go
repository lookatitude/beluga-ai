package mcp

import (
	"strings"
	"testing"
)

func TestValidateToolOutput(t *testing.T) {
	tests := []struct {
		name    string
		output  any
		schema  map[string]any
		wantErr string
	}{
		{
			name:   "nil schema always passes",
			output: "anything",
			schema: nil,
		},
		{
			name:   "empty schema always passes",
			output: 42,
			schema: map[string]any{},
		},
		{
			name:   "valid string",
			output: "hello",
			schema: map[string]any{"type": "string"},
		},
		{
			name:    "expected string got number",
			output:  42,
			schema:  map[string]any{"type": "string"},
			wantErr: "expected string",
		},
		{
			name:   "valid number",
			output: 3.14,
			schema: map[string]any{"type": "number"},
		},
		{
			name:    "expected number got string",
			output:  "not a number",
			schema:  map[string]any{"type": "number"},
			wantErr: "expected number",
		},
		{
			name:   "valid integer",
			output: 42,
			schema: map[string]any{"type": "integer"},
		},
		{
			name:    "expected integer got float",
			output:  3.14,
			schema:  map[string]any{"type": "integer"},
			wantErr: "expected integer, got float",
		},
		{
			name:   "valid boolean",
			output: true,
			schema: map[string]any{"type": "boolean"},
		},
		{
			name:    "expected boolean got string",
			output:  "true",
			schema:  map[string]any{"type": "boolean"},
			wantErr: "expected boolean",
		},
		{
			name:   "valid object",
			output: map[string]any{"name": "test"},
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{"type": "string"},
				},
			},
		},
		{
			name:   "object missing required",
			output: map[string]any{},
			schema: map[string]any{
				"type":     "object",
				"required": []any{"name"},
				"properties": map[string]any{
					"name": map[string]any{"type": "string"},
				},
			},
			wantErr: "missing required property",
		},
		{
			name:   "object wrong property type",
			output: map[string]any{"name": 42},
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{"type": "string"},
				},
			},
			wantErr: "expected string",
		},
		{
			name:   "valid array",
			output: []any{"a", "b"},
			schema: map[string]any{
				"type":  "array",
				"items": map[string]any{"type": "string"},
			},
		},
		{
			name:   "array wrong item type",
			output: []any{"a", 42},
			schema: map[string]any{
				"type":  "array",
				"items": map[string]any{"type": "string"},
			},
			wantErr: "expected string",
		},
		{
			name:   "string enum valid",
			output: "red",
			schema: map[string]any{
				"type": "string",
				"enum": []any{"red", "green", "blue"},
			},
		},
		{
			name:   "string enum invalid",
			output: "yellow",
			schema: map[string]any{
				"type": "string",
				"enum": []any{"red", "green", "blue"},
			},
			wantErr: "not in enum",
		},
		{
			name:    "null value",
			output:  nil,
			schema:  map[string]any{"type": "string"},
			wantErr: "expected string, got null",
		},
		{
			name: "nested object validation",
			output: map[string]any{
				"user": map[string]any{
					"name": "Alice",
					"age":  30,
				},
			},
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"user": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"name": map[string]any{"type": "string"},
							"age":  map[string]any{"type": "integer"},
						},
						"required": []any{"name"},
					},
				},
			},
		},
		{
			name:   "schema without type passes",
			output: "anything",
			schema: map[string]any{"description": "no type constraint"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateToolOutput(tt.output, tt.schema)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatal("expected error")
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("error %q does not contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestStructuredToolInfo_ToToolInfo(t *testing.T) {
	s := StructuredToolInfo{
		Name:        "test-tool",
		Description: "A test tool",
		InputSchema: map[string]any{"type": "object"},
		OutputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"result": map[string]any{"type": "string"},
			},
		},
	}

	info := s.ToToolInfo()
	if info.Name != "test-tool" {
		t.Errorf("expected name 'test-tool', got %q", info.Name)
	}
	if info.Description != "A test tool" {
		t.Errorf("expected description 'A test tool', got %q", info.Description)
	}
	if info.InputSchema == nil {
		t.Error("expected non-nil InputSchema")
	}
}
