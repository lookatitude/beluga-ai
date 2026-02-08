package memory

import (
	"context"
	"sort"
	"testing"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time interface check.
var _ Memory = (*testMemory)(nil)

// testMemory is a minimal Memory implementation for registry tests.
type testMemory struct{}

func (m *testMemory) Save(ctx context.Context, input, output schema.Message) error {
	return nil
}

func (m *testMemory) Load(ctx context.Context, query string) ([]schema.Message, error) {
	return nil, nil
}

func (m *testMemory) Search(ctx context.Context, query string, k int) ([]schema.Document, error) {
	return nil, nil
}

func (m *testMemory) Clear(ctx context.Context) error {
	return nil
}

func TestRegister(t *testing.T) {
	// Register a test provider.
	Register("test-memory", func(cfg config.ProviderConfig) (Memory, error) {
		return &testMemory{}, nil
	})

	// Verify it's in the registry.
	providers := List()
	assert.Contains(t, providers, "test-memory")
}

func TestNew(t *testing.T) {
	tests := []struct {
		name      string
		provider  string
		cfg       config.ProviderConfig
		wantError bool
	}{
		{
			name:     "core provider",
			provider: "core",
			cfg:      config.ProviderConfig{Provider: "core"},
		},
		{
			name:     "recall provider",
			provider: "recall",
			cfg:      config.ProviderConfig{Provider: "recall"},
		},
		{
			name:     "composite provider",
			provider: "composite",
			cfg:      config.ProviderConfig{Provider: "composite"},
		},
		{
			name:      "unknown provider",
			provider:  "unknown-memory",
			cfg:       config.ProviderConfig{Provider: "unknown-memory"},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mem, err := New(tt.provider, tt.cfg)
			if tt.wantError {
				require.Error(t, err)
				assert.Nil(t, mem)
				assert.Contains(t, err.Error(), "unknown provider")
			} else {
				require.NoError(t, err)
				assert.NotNil(t, mem)
			}
		})
	}
}

func TestList(t *testing.T) {
	providers := List()

	// Verify core providers are registered.
	assert.Contains(t, providers, "core")
	assert.Contains(t, providers, "recall")
	assert.Contains(t, providers, "composite")
	assert.Contains(t, providers, "archival")

	// Verify the list is sorted.
	sortedProviders := make([]string, len(providers))
	copy(sortedProviders, providers)
	sort.Strings(sortedProviders)
	assert.Equal(t, sortedProviders, providers)
}

func TestMemoryInterface(t *testing.T) {
	// Verify all built-in providers implement Memory.
	ctx := context.Background()

	t.Run("core implements Memory", func(t *testing.T) {
		mem, err := New("core", config.ProviderConfig{Provider: "core"})
		require.NoError(t, err)

		input := schema.NewHumanMessage("hello")
		output := schema.NewAIMessage("hi")

		err = mem.Save(ctx, input, output)
		assert.NoError(t, err)

		msgs, err := mem.Load(ctx, "")
		assert.NoError(t, err)
		// Core returns nil when persona/human are empty.
		assert.Empty(t, msgs)

		docs, err := mem.Search(ctx, "hello", 5)
		assert.NoError(t, err)
		assert.Nil(t, docs)

		err = mem.Clear(ctx)
		assert.NoError(t, err)
	})

	t.Run("recall implements Memory", func(t *testing.T) {
		mem, err := New("recall", config.ProviderConfig{Provider: "recall"})
		require.NoError(t, err)

		input := schema.NewHumanMessage("hello")
		output := schema.NewAIMessage("hi")

		err = mem.Save(ctx, input, output)
		assert.NoError(t, err)

		msgs, err := mem.Load(ctx, "")
		assert.NoError(t, err)
		assert.Len(t, msgs, 2)

		docs, err := mem.Search(ctx, "hello", 5)
		assert.NoError(t, err)
		assert.Nil(t, docs)

		err = mem.Clear(ctx)
		assert.NoError(t, err)

		msgs, err = mem.Load(ctx, "")
		assert.NoError(t, err)
		assert.Empty(t, msgs)
	})

	t.Run("composite implements Memory", func(t *testing.T) {
		mem, err := New("composite", config.ProviderConfig{Provider: "composite"})
		require.NoError(t, err)

		input := schema.NewHumanMessage("hello")
		output := schema.NewAIMessage("hi")

		err = mem.Save(ctx, input, output)
		assert.NoError(t, err)

		msgs, err := mem.Load(ctx, "")
		assert.NoError(t, err)
		assert.NotNil(t, msgs)

		docs, err := mem.Search(ctx, "hello", 5)
		assert.NoError(t, err)
		assert.Nil(t, docs)

		err = mem.Clear(ctx)
		assert.NoError(t, err)
	})
}
