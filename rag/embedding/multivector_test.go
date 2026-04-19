package embedding

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/v2/config"
)

func TestMultiVectorRegistry(t *testing.T) {
	// Clean up after test to avoid polluting other tests.
	origRegistry := make(map[string]MultiVectorFactory)
	mvRegistryMu.Lock()
	for k, v := range mvRegistry {
		origRegistry[k] = v
	}
	mvRegistryMu.Unlock()
	t.Cleanup(func() {
		mvRegistryMu.Lock()
		mvRegistry = origRegistry
		mvRegistryMu.Unlock()
	})

	t.Run("register and list", func(t *testing.T) {
		RegisterMultiVector("test-mv", func(_ config.ProviderConfig) (MultiVectorEmbedder, error) {
			return nil, nil
		})
		names := ListMultiVector()
		found := false
		for _, n := range names {
			if n == "test-mv" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("registered provider 'test-mv' not found in List(): %v", names)
		}
	})

	t.Run("new unknown provider", func(t *testing.T) {
		_, err := NewMultiVector("nonexistent", config.ProviderConfig{})
		if err == nil {
			t.Fatal("expected error for unknown provider")
		}
	})

	t.Run("new known provider", func(t *testing.T) {
		RegisterMultiVector("test-mv-ok", func(_ config.ProviderConfig) (MultiVectorEmbedder, error) {
			return &stubMultiVectorEmbedder{}, nil
		})
		emb, err := NewMultiVector("test-mv-ok", config.ProviderConfig{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if emb == nil {
			t.Fatal("expected non-nil embedder")
		}
	})

	t.Run("list is sorted", func(t *testing.T) {
		RegisterMultiVector("zzz", func(_ config.ProviderConfig) (MultiVectorEmbedder, error) {
			return nil, nil
		})
		RegisterMultiVector("aaa", func(_ config.ProviderConfig) (MultiVectorEmbedder, error) {
			return nil, nil
		})
		names := ListMultiVector()
		for i := 1; i < len(names); i++ {
			if names[i] < names[i-1] {
				t.Errorf("list not sorted: %v", names)
				break
			}
		}
	})
}

// stubMultiVectorEmbedder is a minimal test stub.
type stubMultiVectorEmbedder struct{}

func (s *stubMultiVectorEmbedder) EmbedMulti(_ context.Context, texts []string) ([][][]float32, error) {
	result := make([][][]float32, len(texts))
	for i := range texts {
		result[i] = [][]float32{{1, 0, 0}}
	}
	return result, nil
}

func (s *stubMultiVectorEmbedder) TokenDimensions() int { return 3 }
