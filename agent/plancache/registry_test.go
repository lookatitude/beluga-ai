package plancache

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistryKeywordRegistered(t *testing.T) {
	matchers := ListMatchers()
	assert.Contains(t, matchers, "keyword")
}

func TestNewMatcher_Keyword(t *testing.T) {
	m, err := NewMatcher("keyword")
	require.NoError(t, err)
	assert.NotNil(t, m)
	_, ok := m.(*KeywordMatcher)
	assert.True(t, ok)
}

func TestNewMatcher_NotRegistered(t *testing.T) {
	_, err := NewMatcher("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "matcher not registered")
}

func TestListMatchers_Sorted(t *testing.T) {
	matchers := ListMatchers()
	for i := 1; i < len(matchers); i++ {
		assert.LessOrEqual(t, matchers[i-1], matchers[i])
	}
}

func TestRegisterMatcher_Custom(t *testing.T) {
	RegisterMatcher("test-custom", func() (Matcher, error) {
		return &KeywordMatcher{}, nil
	})
	defer func() {
		matcherMu.Lock()
		delete(matcherRegistry, "test-custom")
		matcherMu.Unlock()
	}()

	m, err := NewMatcher("test-custom")
	require.NoError(t, err)
	assert.NotNil(t, m)
	assert.Contains(t, ListMatchers(), "test-custom")
}
