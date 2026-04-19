package contract

import (
	"testing"

	"github.com/lookatitude/beluga-ai/v2/schema"
	"github.com/stretchr/testify/assert"
)

func TestCheckCompatibility(t *testing.T) {
	tests := []struct {
		name     string
		from     *schema.Contract
		to       *schema.Contract
		wantErrs int
	}{
		{
			name:     "nil from passes",
			from:     nil,
			to:       &schema.Contract{Name: "target"},
			wantErrs: 0,
		},
		{
			name:     "nil to passes",
			from:     &schema.Contract{Name: "source"},
			to:       nil,
			wantErrs: 0,
		},
		{
			name:     "nil schemas (wildcard) passes",
			from:     &schema.Contract{Name: "source"},
			to:       &schema.Contract{Name: "target"},
			wantErrs: 0,
		},
		{
			name: "matching string types passes",
			from: &schema.Contract{
				Name:         "source",
				OutputSchema: map[string]any{"type": "string"},
			},
			to: &schema.Contract{
				Name:        "target",
				InputSchema: map[string]any{"type": "string"},
			},
			wantErrs: 0,
		},
		{
			name: "mismatched types fails",
			from: &schema.Contract{
				Name:         "source",
				OutputSchema: map[string]any{"type": "string"},
			},
			to: &schema.Contract{
				Name:        "target",
				InputSchema: map[string]any{"type": "number"},
			},
			wantErrs: 1,
		},
		{
			name: "int to number widening passes",
			from: &schema.Contract{
				Name:         "source",
				OutputSchema: map[string]any{"type": "integer"},
			},
			to: &schema.Contract{
				Name:        "target",
				InputSchema: map[string]any{"type": "number"},
			},
			wantErrs: 0,
		},
		{
			name: "number to int narrowing fails",
			from: &schema.Contract{
				Name:         "source",
				OutputSchema: map[string]any{"type": "number"},
			},
			to: &schema.Contract{
				Name:        "target",
				InputSchema: map[string]any{"type": "integer"},
			},
			wantErrs: 1,
		},
		{
			name: "required field present passes",
			from: &schema.Contract{
				Name: "source",
				OutputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name": map[string]any{"type": "string"},
						"age":  map[string]any{"type": "integer"},
					},
				},
			},
			to: &schema.Contract{
				Name: "target",
				InputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name": map[string]any{"type": "string"},
					},
					"required": []any{"name"},
				},
			},
			wantErrs: 0,
		},
		{
			name: "required field missing fails",
			from: &schema.Contract{
				Name: "source",
				OutputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"age": map[string]any{"type": "integer"},
					},
				},
			},
			to: &schema.Contract{
				Name: "target",
				InputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name": map[string]any{"type": "string"},
					},
					"required": []any{"name"},
				},
			},
			wantErrs: 1,
		},
		{
			name: "field type mismatch fails",
			from: &schema.Contract{
				Name: "source",
				OutputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"age": map[string]any{"type": "string"},
					},
				},
			},
			to: &schema.Contract{
				Name: "target",
				InputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"age": map[string]any{"type": "integer"},
					},
					"required": []any{"age"},
				},
			},
			wantErrs: 1,
		},
		{
			name: "source wildcard output passes",
			from: &schema.Contract{
				Name:         "source",
				OutputSchema: nil,
			},
			to: &schema.Contract{
				Name: "target",
				InputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name": map[string]any{"type": "string"},
					},
					"required": []any{"name"},
				},
			},
			wantErrs: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := CheckCompatibility(tt.from, tt.to)
			assert.Len(t, errs, tt.wantErrs)
		})
	}
}

func TestCompatibilityError_Error(t *testing.T) {
	e := CompatibilityError{
		SourceAgent: "agent-a",
		TargetAgent: "agent-b",
		Field:       "name",
		Reason:      "missing from source output",
	}
	got := e.Error()
	assert.Contains(t, got, "agent-a")
	assert.Contains(t, got, "agent-b")
	assert.Contains(t, got, "name")
	assert.Contains(t, got, "missing from source output")
}
