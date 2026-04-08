package contract

import (
	"context"
	"errors"
	"testing"

	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateInput(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		contract *schema.Contract
		input    any
		wantErr  bool
	}{
		{
			name:     "nil contract passes",
			contract: nil,
			input:    "hello",
			wantErr:  false,
		},
		{
			name:     "nil input schema passes",
			contract: &schema.Contract{Name: "test"},
			input:    "hello",
			wantErr:  false,
		},
		{
			name: "string type matches string",
			contract: &schema.Contract{
				Name:        "test",
				InputSchema: map[string]any{"type": "string"},
			},
			input:   "hello",
			wantErr: false,
		},
		{
			name: "string type rejects number",
			contract: &schema.Contract{
				Name:        "test",
				InputSchema: map[string]any{"type": "number"},
			},
			input:   "hello",
			wantErr: true,
		},
		{
			name: "object with required fields passes",
			contract: &schema.Contract{
				Name: "test",
				InputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name": map[string]any{"type": "string"},
						"age":  map[string]any{"type": "integer"},
					},
					"required": []any{"name"},
				},
			},
			input:   map[string]any{"name": "Alice", "age": 30},
			wantErr: false,
		},
		{
			name: "object missing required field fails",
			contract: &schema.Contract{
				Name: "test",
				InputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name": map[string]any{"type": "string"},
					},
					"required": []any{"name"},
				},
			},
			input:   map[string]any{"age": 30},
			wantErr: true,
		},
		{
			name: "enum passes valid value",
			contract: &schema.Contract{
				Name: "test",
				InputSchema: map[string]any{
					"type": "string",
					"enum": []any{"red", "green", "blue"},
				},
			},
			input:   "red",
			wantErr: false,
		},
		{
			name: "enum rejects invalid value",
			contract: &schema.Contract{
				Name: "test",
				InputSchema: map[string]any{
					"type": "string",
					"enum": []any{"red", "green", "blue"},
				},
			},
			input:   "yellow",
			wantErr: true,
		},
		{
			name: "strict rejects additional properties",
			contract: &schema.Contract{
				Name:   "test",
				Strict: true,
				InputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name": map[string]any{"type": "string"},
					},
				},
			},
			input:   map[string]any{"name": "Alice", "extra": "value"},
			wantErr: true,
		},
		{
			name: "non-strict allows additional properties",
			contract: &schema.Contract{
				Name: "test",
				InputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name": map[string]any{"type": "string"},
					},
				},
			},
			input:   map[string]any{"name": "Alice", "extra": "value"},
			wantErr: false,
		},
		{
			name: "JSON string input parsed as object",
			contract: &schema.Contract{
				Name: "test",
				InputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name": map[string]any{"type": "string"},
					},
					"required": []any{"name"},
				},
			},
			input:   `{"name": "Alice"}`,
			wantErr: false,
		},
		{
			name: "array validation passes",
			contract: &schema.Contract{
				Name: "test",
				InputSchema: map[string]any{
					"type":  "array",
					"items": map[string]any{"type": "string"},
				},
			},
			input:   []any{"a", "b", "c"},
			wantErr: false,
		},
		{
			name: "boolean type matches",
			contract: &schema.Contract{
				Name:        "test",
				InputSchema: map[string]any{"type": "boolean"},
			},
			input:   true,
			wantErr: false,
		},
		{
			name: "integer type matches int",
			contract: &schema.Contract{
				Name:        "test",
				InputSchema: map[string]any{"type": "integer"},
			},
			input:   42,
			wantErr: false,
		},
		{
			name: "integer type matches whole float64",
			contract: &schema.Contract{
				Name:        "test",
				InputSchema: map[string]any{"type": "integer"},
			},
			input:   float64(42),
			wantErr: false,
		},
		{
			name: "integer type rejects fractional float",
			contract: &schema.Contract{
				Name:        "test",
				InputSchema: map[string]any{"type": "integer"},
			},
			input:   42.5,
			wantErr: true,
		},
		{
			name: "number type matches int (widening)",
			contract: &schema.Contract{
				Name:        "test",
				InputSchema: map[string]any{"type": "number"},
			},
			input:   42,
			wantErr: false,
		},
		{
			name: "nested object validation",
			contract: &schema.Contract{
				Name: "test",
				InputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"address": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"city": map[string]any{"type": "string"},
							},
							"required": []any{"city"},
						},
					},
					"required": []any{"address"},
				},
			},
			input:   map[string]any{"address": map[string]any{"city": "NYC"}},
			wantErr: false,
		},
		{
			name: "nested object validation fails on missing required",
			contract: &schema.Contract{
				Name: "test",
				InputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"address": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"city": map[string]any{"type": "string"},
							},
							"required": []any{"city"},
						},
					},
					"required": []any{"address"},
				},
			},
			input:   map[string]any{"address": map[string]any{"zip": "10001"}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateInput(ctx, tt.contract, tt.input)
			if tt.wantErr {
				require.Error(t, err)
				var coreErr *core.Error
				assert.True(t, errors.As(err, &coreErr))
				assert.Equal(t, core.ErrInvalidInput, coreErr.Code)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateOutput(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		contract *schema.Contract
		output   any
		wantErr  bool
	}{
		{
			name:     "nil contract passes",
			contract: nil,
			output:   "result",
			wantErr:  false,
		},
		{
			name: "string output matches string schema",
			contract: &schema.Contract{
				Name:         "test",
				OutputSchema: map[string]any{"type": "string"},
			},
			output:  "result",
			wantErr: false,
		},
		{
			name: "number output fails string schema",
			contract: &schema.Contract{
				Name:         "test",
				OutputSchema: map[string]any{"type": "string"},
			},
			output:  42,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOutput(ctx, tt.contract, tt.output)
			if tt.wantErr {
				require.Error(t, err)
				var coreErr *core.Error
				assert.True(t, errors.As(err, &coreErr))
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
