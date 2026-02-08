package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScopedKey(t *testing.T) {
	tests := []struct {
		scope Scope
		key   string
		want  string
	}{
		{ScopeAgent, "counter", "agent:counter"},
		{ScopeSession, "user_id", "session:user_id"},
		{ScopeGlobal, "config", "global:config"},
		{ScopeAgent, "", "agent:"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := ScopedKey(tt.scope, tt.key)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRegisterAndNew(t *testing.T) {
	// Clean up registry for test isolation.
	mu.Lock()
	orig := make(map[string]Factory, len(registry))
	for k, v := range registry {
		orig[k] = v
	}
	registry = make(map[string]Factory)
	mu.Unlock()
	defer func() {
		mu.Lock()
		registry = orig
		mu.Unlock()
	}()

	// Register a mock factory.
	called := false
	Register("test", func(cfg Config) (Store, error) {
		called = true
		return nil, nil
	})

	// New should find the factory.
	_, err := New("test", Config{})
	require.NoError(t, err)
	assert.True(t, called)

	// New with unknown name should error.
	_, err = New("unknown", Config{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `store "unknown" not registered`)
}

func TestList(t *testing.T) {
	// Clean up registry for test isolation.
	mu.Lock()
	orig := make(map[string]Factory, len(registry))
	for k, v := range registry {
		orig[k] = v
	}
	registry = make(map[string]Factory)
	mu.Unlock()
	defer func() {
		mu.Lock()
		registry = orig
		mu.Unlock()
	}()

	Register("bravo", func(cfg Config) (Store, error) { return nil, nil })
	Register("alpha", func(cfg Config) (Store, error) { return nil, nil })
	Register("charlie", func(cfg Config) (Store, error) { return nil, nil })

	names := List()
	assert.Equal(t, []string{"alpha", "bravo", "charlie"}, names)
}

func TestListEmpty(t *testing.T) {
	mu.Lock()
	orig := make(map[string]Factory, len(registry))
	for k, v := range registry {
		orig[k] = v
	}
	registry = make(map[string]Factory)
	mu.Unlock()
	defer func() {
		mu.Lock()
		registry = orig
		mu.Unlock()
	}()

	names := List()
	assert.Empty(t, names)
}

func TestScopeConstants(t *testing.T) {
	assert.Equal(t, Scope("agent"), ScopeAgent)
	assert.Equal(t, Scope("session"), ScopeSession)
	assert.Equal(t, Scope("global"), ScopeGlobal)
}

func TestChangeOpConstants(t *testing.T) {
	assert.Equal(t, ChangeOp("set"), OpSet)
	assert.Equal(t, ChangeOp("delete"), OpDelete)
}
