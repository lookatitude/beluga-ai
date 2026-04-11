package agentfile

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestJSONSerializer_Serialize(t *testing.T) {
	af := NewAgentFile("test-agent", "Assistant", "openai", "gpt-4o")
	s := &JSONSerializer{}

	data, err := s.Serialize(context.Background(), af)
	if err != nil {
		t.Fatalf("Serialize: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty output")
	}
}

func TestJSONSerializer_NilInput(t *testing.T) {
	s := &JSONSerializer{}
	_, err := s.Serialize(context.Background(), nil)
	if err == nil {
		t.Error("expected error for nil input")
	}
}

func TestJSONDeserializer_Deserialize(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr bool
		wantID  string
	}{
		{
			name: "valid agent file",
			json: `{
				"version": "1.0",
				"agent": {
					"id": "test",
					"persona": {"role": "Helper"},
					"model": {"provider": "openai", "model": "gpt-4o"}
				}
			}`,
			wantID: "test",
		},
		{
			name:    "missing version",
			json:    `{"agent": {"id": "test", "persona": {"role": "R"}, "model": {"provider": "x", "model": "y"}}}`,
			wantErr: true,
		},
		{
			name:    "missing agent id",
			json:    `{"version": "1.0", "agent": {"persona": {"role": "R"}, "model": {"provider": "x", "model": "y"}}}`,
			wantErr: true,
		},
		{
			name:    "invalid json",
			json:    `{invalid`,
			wantErr: true,
		},
	}

	d := NewDeserializer()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			af, err := d.Deserialize(context.Background(), []byte(tt.json))
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Deserialize: %v", err)
			}
			if af.Agent.ID != tt.wantID {
				t.Errorf("ID = %q, want %q", af.Agent.ID, tt.wantID)
			}
		})
	}
}

func TestJSONDeserializer_MaxSize(t *testing.T) {
	d := NewDeserializer(WithMaxSize(10))
	_, err := d.Deserialize(context.Background(), make([]byte, 100))
	if err == nil {
		t.Error("expected error for oversized input")
	}
}

func TestRoundTrip(t *testing.T) {
	af := NewAgentFile("round-trip", "Tester", "anthropic", "claude-3")
	af.Agent.Description = "Test agent"
	af.Agent.Tools = []ToolDef{{Name: "search", Config: map[string]any{"api_key_env": "SEARCH_KEY"}}} //nolint:gosec // G101: false positive - env var name, not a credential

	s := &JSONSerializer{}
	data, err := s.Serialize(context.Background(), af)
	if err != nil {
		t.Fatalf("Serialize: %v", err)
	}

	d := NewDeserializer()
	af2, err := d.Deserialize(context.Background(), data)
	if err != nil {
		t.Fatalf("Deserialize: %v", err)
	}

	if af2.Agent.ID != "round-trip" {
		t.Errorf("ID = %q, want round-trip", af2.Agent.ID)
	}
	if af2.Agent.Description != "Test agent" {
		t.Errorf("Description = %q, want Test agent", af2.Agent.Description)
	}
	if len(af2.Agent.Tools) != 1 {
		t.Errorf("Tools = %d, want 1", len(af2.Agent.Tools))
	}
}

func TestDefaultMigrator(t *testing.T) {
	migrator := NewMigrator()
	ctx := context.Background()

	af := &AgentFile{
		Version: "0.1",
		Agent: AgentDef{
			ID:      "test",
			Persona: PersonaDef{Role: "Helper"},
			Model:   ModelDef{Provider: "openai", Model: "gpt-4o"},
		},
	}

	migrated, err := migrator.Migrate(ctx, af, "1.0")
	if err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if migrated.Version != "1.0" {
		t.Errorf("Version = %q, want 1.0", migrated.Version)
	}

	// Same version should be no-op.
	same, err := migrator.Migrate(ctx, af, "0.1")
	if err != nil {
		t.Fatalf("Migrate same: %v", err)
	}
	if same.Version != "0.1" {
		t.Errorf("Version = %q, want 0.1", same.Version)
	}

	versions := migrator.SupportedVersions()
	if len(versions) == 0 {
		t.Error("expected supported versions")
	}
}

func TestDefaultMigrator_NilInput(t *testing.T) {
	migrator := NewMigrator()
	_, err := migrator.Migrate(context.Background(), nil, "1.0")
	if err == nil {
		t.Error("expected error for nil input")
	}
}

func TestDefaultMigrator_UnsupportedVersion(t *testing.T) {
	migrator := NewMigrator()
	af := NewAgentFile("test", "R", "x", "y")
	_, err := migrator.Migrate(context.Background(), af, "99.9")
	if err == nil {
		t.Error("expected error for unsupported target version")
	}
}

func TestDefaultMigrator_DeepCopy(t *testing.T) {
	migrator := NewMigrator()
	af := &AgentFile{
		Version: "0.1",
		Agent: AgentDef{
			ID:       "test",
			Persona:  PersonaDef{Role: "Helper"},
			Model:    ModelDef{Provider: "openai", Model: "gpt-4o"},
			Settings: map[string]any{"k": "v"},
			Tools:    []ToolDef{{Name: "t1", Config: map[string]any{"a": 1}}},
		},
	}

	migrated, err := migrator.Migrate(context.Background(), af, "1.0")
	if err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	// Mutating the migrated copy must not affect the original.
	migrated.Agent.Settings["k"] = "other"
	migrated.Agent.Tools[0].Config["a"] = 999
	migrated.Agent.Tools = append(migrated.Agent.Tools, ToolDef{Name: "t2"})

	if af.Agent.Settings["k"] != "v" {
		t.Errorf("original Settings mutated: %v", af.Agent.Settings)
	}
	if af.Agent.Tools[0].Config["a"] != 1 {
		t.Errorf("original Tool config mutated: %v", af.Agent.Tools[0].Config)
	}
	if len(af.Agent.Tools) != 1 {
		t.Errorf("original Tools slice mutated: len=%d", len(af.Agent.Tools))
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.af")
	ctx := context.Background()

	af := NewAgentFile("save-test", "Helper", "openai", "gpt-4o")
	if err := Save(ctx, path, af); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(ctx, path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Agent.ID != "save-test" {
		t.Errorf("ID = %q, want save-test", loaded.Agent.ID)
	}
}

func TestSave_PathTraversal(t *testing.T) {
	af := NewAgentFile("test", "R", "x", "y")
	err := Save(context.Background(), "/tmp/../etc/test.af", af)
	if err == nil {
		t.Error("expected error for path traversal")
	}
}

func TestLoad_PathTraversal(t *testing.T) {
	_, err := Load(context.Background(), "/tmp/../etc/test.af")
	if err == nil {
		t.Error("expected error for path traversal")
	}
}

func TestLoad_NotFound(t *testing.T) {
	_, err := Load(context.Background(), "/nonexistent/path/file.af")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoad_ExceedsMaxSize(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "big.af")

	// Write a file larger than 1 MB to ensure the stat-based size guard fires
	// before ReadFile allocates memory.
	big := make([]byte, (1<<20)+1)
	if err := os.WriteFile(path, big, 0600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	_, err := Load(context.Background(), path)
	if err == nil {
		t.Error("expected error for oversized file")
	}
}

func TestNewAgentFile(t *testing.T) {
	af := NewAgentFile("my-agent", "Coder", "openai", "gpt-4o")
	if af.Version != CurrentVersion {
		t.Errorf("Version = %q, want %q", af.Version, CurrentVersion)
	}
	if af.Agent.ID != "my-agent" {
		t.Errorf("ID = %q, want my-agent", af.Agent.ID)
	}
	if af.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
}
