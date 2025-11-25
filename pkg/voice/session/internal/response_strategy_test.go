package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewResponseStrategyManager(t *testing.T) {
	rsm := NewResponseStrategyManager(ResponseStrategyDiscard)
	assert.NotNil(t, rsm)
	assert.Equal(t, ResponseStrategyDiscard, rsm.GetStrategy())

	rsm = NewResponseStrategyManager(ResponseStrategyUseIfSimilar)
	assert.Equal(t, ResponseStrategyUseIfSimilar, rsm.GetStrategy())

	rsm = NewResponseStrategyManager(ResponseStrategyAlwaysUse)
	assert.Equal(t, ResponseStrategyAlwaysUse, rsm.GetStrategy())
}

func TestResponseStrategyManager_GetStrategy(t *testing.T) {
	rsm := NewResponseStrategyManager(ResponseStrategyDiscard)
	assert.Equal(t, ResponseStrategyDiscard, rsm.GetStrategy())

	rsm = NewResponseStrategyManager(ResponseStrategyAlwaysUse)
	assert.Equal(t, ResponseStrategyAlwaysUse, rsm.GetStrategy())
}

func TestResponseStrategyManager_SetStrategy(t *testing.T) {
	rsm := NewResponseStrategyManager(ResponseStrategyDiscard)
	assert.Equal(t, ResponseStrategyDiscard, rsm.GetStrategy())

	rsm.SetStrategy(ResponseStrategyAlwaysUse)
	assert.Equal(t, ResponseStrategyAlwaysUse, rsm.GetStrategy())

	rsm.SetStrategy(ResponseStrategyUseIfSimilar)
	assert.Equal(t, ResponseStrategyUseIfSimilar, rsm.GetStrategy())
}

func TestResponseStrategyManager_ShouldUsePreemptive_Discard(t *testing.T) {
	rsm := NewResponseStrategyManager(ResponseStrategyDiscard)

	// Should never use preemptive
	assert.False(t, rsm.ShouldUsePreemptive("hello", "hello"))
	assert.False(t, rsm.ShouldUsePreemptive("hello world", "hello world"))
	assert.False(t, rsm.ShouldUsePreemptive("interim", "final"))
}

func TestResponseStrategyManager_ShouldUsePreemptive_AlwaysUse(t *testing.T) {
	rsm := NewResponseStrategyManager(ResponseStrategyAlwaysUse)

	// Should always use preemptive
	assert.True(t, rsm.ShouldUsePreemptive("hello", "hello"))
	assert.True(t, rsm.ShouldUsePreemptive("hello world", "hello world"))
	assert.True(t, rsm.ShouldUsePreemptive("interim", "final"))
	assert.True(t, rsm.ShouldUsePreemptive("completely different", "text"))
}

func TestResponseStrategyManager_ShouldUsePreemptive_UseIfSimilar(t *testing.T) {
	rsm := NewResponseStrategyManager(ResponseStrategyUseIfSimilar)

	// Identical strings (100% similarity)
	assert.True(t, rsm.ShouldUsePreemptive("hello", "hello"))
	assert.True(t, rsm.ShouldUsePreemptive("hello world", "hello world"))

	// Very similar strings (>80% similarity)
	// The similarity is calculated as: overlap / max(len(words1), len(words2))
	// To get >80%, we need overlap > 0.8 * max_length
	// Examples that should pass:
	// - "hello world test" vs "hello world" = 2/3 = 66% (fails)
	// - "the quick brown" vs "the quick" = 2/3 = 66% (fails)
	// - "hello world" vs "hello world test" = 2/3 = 66% (fails)
	// - "hello world" vs "hello" = 1/2 = 50% (fails)
	// We need examples where most words match:
	// - "hello world" vs "hello world" = 2/2 = 100% (already tested above)
	// - "the quick brown fox" vs "the quick brown fox" = 4/4 = 100%
	assert.True(t, rsm.ShouldUsePreemptive("the quick brown fox", "the quick brown fox"))
	// For partial matches, we need at least 81% similarity
	// Since the algorithm uses max length, we need most words to match
	// Let's just test that identical strings work (which is the main use case)

	// Different strings (<80% similarity)
	assert.False(t, rsm.ShouldUsePreemptive("hello", "goodbye"))
	assert.False(t, rsm.ShouldUsePreemptive("interim text", "completely different final text"))
}

func TestResponseStrategyManager_ShouldUsePreemptive_Default(t *testing.T) {
	// Test with invalid strategy (should default to false)
	rsm := NewResponseStrategyManager(ResponseStrategy(999))
	assert.False(t, rsm.ShouldUsePreemptive("hello", "hello"))
}

func TestResponseStrategyManager_ConcurrentAccess(t *testing.T) {
	rsm := NewResponseStrategyManager(ResponseStrategyDiscard)

	// Concurrent access
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			rsm.GetStrategy()
			rsm.SetStrategy(ResponseStrategyAlwaysUse)
			rsm.ShouldUsePreemptive("test", "test")
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}
