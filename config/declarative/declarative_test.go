package declarative

import (
	"context"
	"os"
	"path/filepath"
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
	_, err := LoadSpec(context.Background(), "/tmp/../etc/passwd")
	if err == nil {
		t.Error("expected error for path traversal")
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
