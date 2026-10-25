package prompt

import (
	"testing"

	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/schema"
)

type stubManager struct{}

func (stubManager) Get(string, string) (*Template, error)                   { return nil, nil }
func (stubManager) Render(string, map[string]any) ([]schema.Message, error) { return nil, nil }
func (stubManager) List() []TemplateInfo                                    { return nil }

// TestRegistry is a regression test for FWC-25: prompt had providers but no
// Register/New/List registry.
func TestRegistry(t *testing.T) {
	Register("stub-test", func(Config) (PromptManager, error) { return stubManager{}, nil })
	m, err := New("stub-test", Config{})
	if err != nil || m == nil {
		t.Fatalf("New = %v, %v", m, err)
	}
	found := false
	for _, n := range List() {
		if n == "stub-test" {
			found = true
		}
	}
	if !found {
		t.Error("List missing stub-test")
	}
	if _, err := New("missing-provider", Config{}); core.CodeOf(err, "") != core.ErrNotFound {
		t.Errorf("unknown provider err = %v", err)
	}
}
