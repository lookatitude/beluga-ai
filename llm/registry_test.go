package llm

import (
	"testing"

	"github.com/lookatitude/beluga-ai/config"
)

func TestRegisterAndNew(t *testing.T) {
	// Save and restore registry state.
	registryMu.Lock()
	orig := make(map[string]Factory, len(registry))
	for k, v := range registry {
		orig[k] = v
	}
	registryMu.Unlock()
	t.Cleanup(func() {
		registryMu.Lock()
		registry = orig
		registryMu.Unlock()
	})

	// Clear the registry for a clean test.
	registryMu.Lock()
	registry = make(map[string]Factory)
	registryMu.Unlock()

	tests := []struct {
		name        string
		provider    string
		factory     Factory
		lookupName  string
		wantErr     bool
		errContains string
	}{
		{
			name:     "register and create successfully",
			provider: "test-provider",
			factory: func(cfg config.ProviderConfig) (ChatModel, error) {
				return &stubModel{id: cfg.Model}, nil
			},
			lookupName: "test-provider",
			wantErr:    false,
		},
		{
			name:        "unknown provider returns error",
			provider:    "",
			factory:     nil,
			lookupName:  "nonexistent",
			wantErr:     true,
			errContains: "unknown provider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.factory != nil {
				Register(tt.provider, tt.factory)
			}

			model, err := New(tt.lookupName, config.ProviderConfig{Model: "test-model"})
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errContains != "" && !containsStr(err.Error(), tt.errContains) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if model.ModelID() != "test-model" {
				t.Errorf("ModelID() = %q, want %q", model.ModelID(), "test-model")
			}
		})
	}
}

func TestList(t *testing.T) {
	registryMu.Lock()
	orig := make(map[string]Factory, len(registry))
	for k, v := range registry {
		orig[k] = v
	}
	registryMu.Unlock()
	t.Cleanup(func() {
		registryMu.Lock()
		registry = orig
		registryMu.Unlock()
	})

	registryMu.Lock()
	registry = make(map[string]Factory)
	registryMu.Unlock()

	// Empty registry.
	if got := List(); len(got) != 0 {
		t.Fatalf("List() on empty registry = %v, want empty", got)
	}

	// Register providers and verify sorted order.
	dummyFactory := func(cfg config.ProviderConfig) (ChatModel, error) { return nil, nil }
	Register("zebra", dummyFactory)
	Register("alpha", dummyFactory)
	Register("middle", dummyFactory)

	got := List()
	want := []string{"alpha", "middle", "zebra"}
	if len(got) != len(want) {
		t.Fatalf("List() len = %d, want %d", len(got), len(want))
	}
	for i, name := range got {
		if name != want[i] {
			t.Errorf("List()[%d] = %q, want %q", i, name, want[i])
		}
	}
}

func TestRegisterOverwrite(t *testing.T) {
	registryMu.Lock()
	orig := make(map[string]Factory, len(registry))
	for k, v := range registry {
		orig[k] = v
	}
	registryMu.Unlock()
	t.Cleanup(func() {
		registryMu.Lock()
		registry = orig
		registryMu.Unlock()
	})

	registryMu.Lock()
	registry = make(map[string]Factory)
	registryMu.Unlock()

	Register("dup", func(cfg config.ProviderConfig) (ChatModel, error) {
		return &stubModel{id: "first"}, nil
	})
	Register("dup", func(cfg config.ProviderConfig) (ChatModel, error) {
		return &stubModel{id: "second"}, nil
	})

	model, err := New("dup", config.ProviderConfig{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if model.ModelID() != "second" {
		t.Errorf("expected overwritten factory, got ModelID() = %q", model.ModelID())
	}
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsSubstring(s, sub))
}

func containsSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
