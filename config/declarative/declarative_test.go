package declarative

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestJSONParser_Parse(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr bool
		wantID  string
	}{
		{
			name: "valid spec",
			json: `{
				"id": "test-agent",
				"persona": {"role": "Assistant"},
				"model": {"provider": "openai", "model": "gpt-4o"}
			}`,
			wantID: "test-agent",
		},
		{
			name: "with tools and options",
			json: `{
				"id": "tool-agent",
				"persona": {"role": "Helper", "goal": "Help users"},
				"model": {"provider": "anthropic", "model": "claude-3", "temperature": 0.7, "max_tokens": 1000},
				"tools": ["search", "calculator"],
				"max_iterations": 5
			}`,
			wantID: "tool-agent",
		},
		{
			name:    "missing id",
			json:    `{"persona": {"role": "Test"}, "model": {"provider": "x", "model": "y"}}`,
			wantErr: true,
		},
		{
			name:    "missing role",
			json:    `{"id": "a", "persona": {}, "model": {"provider": "x", "model": "y"}}`,
			wantErr: true,
		},
		{
			name:    "missing provider",
			json:    `{"id": "a", "persona": {"role": "R"}, "model": {"model": "y"}}`,
			wantErr: true,
		},
		{
			name:    "missing model name",
			json:    `{"id": "a", "persona": {"role": "R"}, "model": {"provider": "x"}}`,
			wantErr: true,
		},
		{
			name:    "invalid json",
			json:    `{invalid`,
			wantErr: true,
		},
		{
			name:    "negative max_iterations",
			json:    `{"id": "a", "persona": {"role": "R"}, "model": {"provider": "x", "model": "y"}, "max_iterations": -1}`,
			wantErr: true,
		},
		{
			name:    "temperature too high",
			json:    `{"id": "a", "persona": {"role": "R"}, "model": {"provider": "x", "model": "y", "temperature": 3.0}}`,
			wantErr: true,
		},
	}

	parser := NewJSONParser()
	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec, err := parser.Parse(ctx, []byte(tt.json))
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if spec.ID != tt.wantID {
				t.Errorf("ID = %q, want %q", spec.ID, tt.wantID)
			}
		})
	}
}

func TestJSONParser_MaxSize(t *testing.T) {
	parser := NewJSONParser(WithMaxSize(10))
	_, err := parser.Parse(context.Background(), make([]byte, 100))
	if err == nil {
		t.Error("expected error for oversized input")
	}
}

func TestDefaultBuilder_Build(t *testing.T) {
	builder := NewBuilder()
	ctx := context.Background()

	spec := &AgentSpec{
		ID:      "test",
		Persona: PersonaSpec{Role: "Assistant"},
		Model:   ModelSpec{Provider: "openai", Model: "gpt-4o"},
		Tools:   []string{"search"},
	}

	build, err := builder.Build(ctx, spec)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if build.ProviderName != "openai" {
		t.Errorf("ProviderName = %q, want openai", build.ProviderName)
	}
	if build.ModelName != "gpt-4o" {
		t.Errorf("ModelName = %q, want gpt-4o", build.ModelName)
	}
	if len(build.ToolNames) != 1 || build.ToolNames[0] != "search" {
		t.Errorf("ToolNames = %v, want [search]", build.ToolNames)
	}
}

func TestDefaultBuilder_NilSpec(t *testing.T) {
	builder := NewBuilder()
	_, err := builder.Build(context.Background(), nil)
	if err == nil {
		t.Error("expected error for nil spec")
	}
}

func TestLoadSpec(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "agent.json")
	data := []byte(`{"id":"file-agent","persona":{"role":"Test"},"model":{"provider":"openai","model":"gpt-4o"}}`)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}

	spec, err := LoadSpec(context.Background(), path)
	if err != nil {
		t.Fatalf("LoadSpec: %v", err)
	}
	if spec.ID != "file-agent" {
		t.Errorf("ID = %q, want file-agent", spec.ID)
	}
}

func TestLoadSpec_PathTraversal(t *testing.T) {
	cases := []string{
		"/tmp/../etc/passwd",
		"../secret.json",
		"a/../../etc/passwd",
	}
	for _, p := range cases {
		t.Run(p, func(t *testing.T) {
			_, err := LoadSpec(context.Background(), p)
			if err == nil {
				t.Fatalf("expected error for path traversal input %q", p)
			}
			if !strings.Contains(err.Error(), "path traversal") {
				t.Errorf("expected path traversal error, got %v", err)
			}
		})
	}
}

func TestLoadSpec_MaxSize(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "big.json")
	// Write a file larger than the 1 MB default limit.
	big := make([]byte, (1<<20)+16)
	for i := range big {
		big[i] = 'a'
	}
	if err := os.WriteFile(path, big, 0600); err != nil {
		t.Fatal(err)
	}
	_, err := LoadSpec(context.Background(), path)
	if err == nil {
		t.Fatal("expected error for oversized file")
	}
	if !strings.Contains(err.Error(), "exceeds maximum size") {
		t.Errorf("expected size-limit error, got %v", err)
	}
}

func TestModelSpec_TemperaturePointer(t *testing.T) {
	// Explicit zero must round-trip and remain distinguishable from unset.
	zero := 0.0
	spec := &AgentSpec{
		ID:      "t",
		Persona: PersonaSpec{Role: "R"},
		Model:   ModelSpec{Provider: "p", Model: "m", Temperature: &zero},
	}
	data, err := MarshalSpec(spec)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := NewJSONParser().Parse(context.Background(), data)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Model.Temperature == nil {
		t.Fatal("expected Temperature to round-trip as non-nil")
	}
	if *parsed.Model.Temperature != 0 {
		t.Errorf("Temperature = %v, want 0", *parsed.Model.Temperature)
	}

	// Unset temperature must remain nil.
	spec2 := &AgentSpec{
		ID:      "t",
		Persona: PersonaSpec{Role: "R"},
		Model:   ModelSpec{Provider: "p", Model: "m"},
	}
	data2, err := MarshalSpec(spec2)
	if err != nil {
		t.Fatal(err)
	}
	parsed2, err := NewJSONParser().Parse(context.Background(), data2)
	if err != nil {
		t.Fatal(err)
	}
	if parsed2.Model.Temperature != nil {
		t.Errorf("Temperature = %v, want nil", *parsed2.Model.Temperature)
	}
}

func TestMarshalSpec(t *testing.T) {
	spec := &AgentSpec{
		ID:      "test",
		Persona: PersonaSpec{Role: "Assistant"},
		Model:   ModelSpec{Provider: "openai", Model: "gpt-4o"},
	}

	data, err := MarshalSpec(spec)
	if err != nil {
		t.Fatalf("MarshalSpec: %v", err)
	}

	// Round-trip.
	parser := NewJSONParser()
	spec2, err := parser.Parse(context.Background(), data)
	if err != nil {
		t.Fatalf("Parse round-trip: %v", err)
	}
	if spec2.ID != spec.ID {
		t.Errorf("round-trip ID = %q, want %q", spec2.ID, spec.ID)
	}
}
