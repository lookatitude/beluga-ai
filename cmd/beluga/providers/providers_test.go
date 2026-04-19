package providers

import (
	"slices"
	"testing"

	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/rag/embedding"
	"github.com/lookatitude/beluga-ai/rag/vectorstore"
)

// TestBlankImportsRegister verifies that importing this package triggers
// init()-time registration for the curated provider set. The assertions
// below match the blank imports in providers.go.
//
// Scope note: memory/stores/inmemory is intentionally not covered here.
// That provider's init() does not currently call memory.Register (the
// package exposes constructors for memory.MessageStore / memory.GraphStore
// but not a memory.Memory factory). Wiring a memory.Memory adapter is a
// separate framework gap tracked outside DX-1 S1. The blank import remains
// in providers.go because the CGo-free audit contract (brief section 317)
// lists it verbatim as part of the curated set.
func TestBlankImportsRegister(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		list     func() []string
		expected []string
	}{
		{"llm", llm.List, []string{"anthropic", "ollama", "openai"}},
		{"embedding", embedding.List, []string{"ollama", "openai"}},
		{"vectorstore", vectorstore.List, []string{"inmemory"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.list()
			for _, want := range tc.expected {
				if !slices.Contains(got, want) {
					t.Errorf("%s.List(): want %q in registry, got %v", tc.name, want, got)
				}
			}
		})
	}
}
