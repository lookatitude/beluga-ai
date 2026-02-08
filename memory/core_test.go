package memory

import (
	"context"
	"strings"
	"testing"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCore(t *testing.T) {
	tests := []struct {
		name             string
		cfg              CoreConfig
		wantPersonaLimit int
		wantHumanLimit   int
		wantSelfEditable bool
	}{
		{
			name:             "default config",
			cfg:              CoreConfig{},
			wantPersonaLimit: DefaultPersonaLimit,
			wantHumanLimit:   DefaultHumanLimit,
			wantSelfEditable: false,
		},
		{
			name: "custom limits",
			cfg: CoreConfig{
				PersonaLimit: 5000,
				HumanLimit:   3000,
			},
			wantPersonaLimit: 5000,
			wantHumanLimit:   3000,
			wantSelfEditable: false,
		},
		{
			name: "self editable",
			cfg: CoreConfig{
				SelfEditable: true,
			},
			wantPersonaLimit: DefaultPersonaLimit,
			wantHumanLimit:   DefaultHumanLimit,
			wantSelfEditable: true,
		},
		{
			name: "zero limits use defaults",
			cfg: CoreConfig{
				PersonaLimit: 0,
				HumanLimit:   0,
			},
			wantPersonaLimit: DefaultPersonaLimit,
			wantHumanLimit:   DefaultHumanLimit,
			wantSelfEditable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			core := NewCore(tt.cfg)
			assert.NotNil(t, core)
			assert.Equal(t, tt.wantPersonaLimit, core.personaLimit)
			assert.Equal(t, tt.wantHumanLimit, core.humanLimit)
			assert.Equal(t, tt.wantSelfEditable, core.selfEditable)
		})
	}
}

func TestCorePersona(t *testing.T) {
	core := NewCore(CoreConfig{PersonaLimit: 50})

	// Initially empty.
	assert.Empty(t, core.GetPersona())

	// Set valid persona.
	err := core.SetPersona("I am a helpful assistant.")
	require.NoError(t, err)
	assert.Equal(t, "I am a helpful assistant.", core.GetPersona())

	// Update persona.
	err = core.SetPersona("I am an expert coder.")
	require.NoError(t, err)
	assert.Equal(t, "I am an expert coder.", core.GetPersona())

	// Exceed limit.
	longPersona := strings.Repeat("a", 51)
	err = core.SetPersona(longPersona)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "persona exceeds limit")
	// Persona should remain unchanged after error.
	assert.Equal(t, "I am an expert coder.", core.GetPersona())
}

func TestCoreHuman(t *testing.T) {
	core := NewCore(CoreConfig{HumanLimit: 50})

	// Initially empty.
	assert.Empty(t, core.GetHuman())

	// Set valid human.
	err := core.SetHuman("User is a software developer.")
	require.NoError(t, err)
	assert.Equal(t, "User is a software developer.", core.GetHuman())

	// Update human.
	err = core.SetHuman("User prefers concise answers.")
	require.NoError(t, err)
	assert.Equal(t, "User prefers concise answers.", core.GetHuman())

	// Exceed limit.
	longHuman := strings.Repeat("b", 51)
	err = core.SetHuman(longHuman)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "human exceeds limit")
	// Human should remain unchanged after error.
	assert.Equal(t, "User prefers concise answers.", core.GetHuman())
}

func TestCoreIsSelfEditable(t *testing.T) {
	t.Run("not self editable by default", func(t *testing.T) {
		core := NewCore(CoreConfig{})
		assert.False(t, core.IsSelfEditable())
	})

	t.Run("self editable when configured", func(t *testing.T) {
		core := NewCore(CoreConfig{SelfEditable: true})
		assert.True(t, core.IsSelfEditable())
	})
}

func TestCoreToMessages(t *testing.T) {
	tests := []struct {
		name      string
		persona   string
		human     string
		wantCount int
	}{
		{
			name:      "empty blocks",
			persona:   "",
			human:     "",
			wantCount: 0,
		},
		{
			name:      "persona only",
			persona:   "I am a helpful assistant.",
			human:     "",
			wantCount: 1,
		},
		{
			name:      "human only",
			persona:   "",
			human:     "User is a developer.",
			wantCount: 1,
		},
		{
			name:      "both blocks",
			persona:   "I am a helpful assistant.",
			human:     "User is a developer.",
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			core := NewCore(CoreConfig{})
			if tt.persona != "" {
				require.NoError(t, core.SetPersona(tt.persona))
			}
			if tt.human != "" {
				require.NoError(t, core.SetHuman(tt.human))
			}

			msgs := core.ToMessages()
			assert.Len(t, msgs, tt.wantCount)

			for _, msg := range msgs {
				assert.Equal(t, schema.RoleSystem, msg.GetRole())
				text := msg.(*schema.SystemMessage).Text()
				assert.True(t,
					strings.Contains(text, "[Persona]") || strings.Contains(text, "[Human]"),
					"message should contain block label")
			}
		})
	}
}

func TestCoreSave(t *testing.T) {
	// Save is a no-op for core memory.
	core := NewCore(CoreConfig{})
	ctx := context.Background()

	input := schema.NewHumanMessage("hello")
	output := schema.NewAIMessage("hi")

	err := core.Save(ctx, input, output)
	assert.NoError(t, err)

	// Verify no state change.
	assert.Empty(t, core.GetPersona())
	assert.Empty(t, core.GetHuman())
}

func TestCoreLoad(t *testing.T) {
	core := NewCore(CoreConfig{})
	ctx := context.Background()

	require.NoError(t, core.SetPersona("I am a helpful assistant."))
	require.NoError(t, core.SetHuman("User is a developer."))

	// Load returns ToMessages regardless of query.
	msgs, err := core.Load(ctx, "any query")
	require.NoError(t, err)
	assert.Len(t, msgs, 2)

	msgs, err = core.Load(ctx, "")
	require.NoError(t, err)
	assert.Len(t, msgs, 2)
}

func TestCoreSearch(t *testing.T) {
	// Search always returns nil for core memory.
	core := NewCore(CoreConfig{})
	ctx := context.Background()

	require.NoError(t, core.SetPersona("I am a helpful assistant."))

	docs, err := core.Search(ctx, "assistant", 5)
	require.NoError(t, err)
	assert.Nil(t, docs)
}

func TestCoreClear(t *testing.T) {
	core := NewCore(CoreConfig{})
	ctx := context.Background()

	require.NoError(t, core.SetPersona("I am a helpful assistant."))
	require.NoError(t, core.SetHuman("User is a developer."))

	err := core.Clear(ctx)
	require.NoError(t, err)

	assert.Empty(t, core.GetPersona())
	assert.Empty(t, core.GetHuman())
}

func TestCoreRegistryEntry(t *testing.T) {
	// Verify "core" is registered via init().
	mem, err := New("core", config.ProviderConfig{
		Provider: "core",
		Options: map[string]any{
			"persona_limit":  float64(3000),
			"human_limit":    float64(2500),
			"self_editable": true,
		},
	})
	require.NoError(t, err)

	core, ok := mem.(*Core)
	require.True(t, ok)
	assert.Equal(t, 3000, core.personaLimit)
	assert.Equal(t, 2500, core.humanLimit)
	assert.True(t, core.selfEditable)
}

func TestCoreConcurrency(t *testing.T) {
	// Test that concurrent reads and writes are safe.
	core := NewCore(CoreConfig{})
	ctx := context.Background()

	done := make(chan bool)

	// Concurrent writes.
	for i := 0; i < 10; i++ {
		go func(n int) {
			_ = core.SetPersona(strings.Repeat("a", n+1))
			_ = core.SetHuman(strings.Repeat("b", n+1))
			done <- true
		}(i)
	}

	// Concurrent reads.
	for i := 0; i < 10; i++ {
		go func() {
			_ = core.GetPersona()
			_ = core.GetHuman()
			_, _ = core.Load(ctx, "")
			done <- true
		}()
	}

	// Wait for all goroutines.
	for i := 0; i < 20; i++ {
		<-done
	}

	// Verify no panics and core is in a valid state.
	assert.NotNil(t, core.GetPersona())
	assert.NotNil(t, core.GetHuman())
}
