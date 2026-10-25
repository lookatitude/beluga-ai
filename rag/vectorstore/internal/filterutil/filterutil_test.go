package filterutil

import (
	"errors"
	"testing"

	"github.com/lookatitude/beluga-ai/v2/core"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		filter  map[string]any
		want    []string
		wantErr bool
	}{
		{"nil", nil, []string{}, false},
		{"sorted", map[string]any{"b": 1, "a": "x", "_c": true, "d": 1.5, "e": uint8(2), "f": float32(1)}, []string{"_c", "a", "b", "d", "e", "f"}, false},
		{"quote in key", map[string]any{`a"b`: "x"}, nil, true},
		{"dollar key", map[string]any{"$where": "x"}, nil, true},
		{"dotted key", map[string]any{"a.b": "x"}, nil, true},
		{"space key", map[string]any{"a b": "x"}, nil, true},
		{"empty key", map[string]any{"": "x"}, nil, true},
		{"leading digit", map[string]any{"1a": "x"}, nil, true},
		{"map value", map[string]any{"a": map[string]any{"$ne": "x"}}, nil, true},
		{"slice value", map[string]any{"a": []string{"x"}}, nil, true},
		{"nil value", map[string]any{"a": nil}, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Validate("op", tt.filter)
			if tt.wantErr {
				var ce *core.Error
				if !errors.As(err, &ce) || ce.Code != core.ErrInvalidInput {
					t.Fatalf("want ErrInvalidInput, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %v want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("got %v want %v", got, tt.want)
				}
			}
		})
	}
}
