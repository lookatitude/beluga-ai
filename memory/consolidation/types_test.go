package consolidation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultWeights(t *testing.T) {
	w := DefaultWeights()
	assert.Equal(t, 0.4, w.Recency)
	assert.Equal(t, 0.3, w.Importance)
	assert.Equal(t, 0.2, w.Relevance)
	assert.Equal(t, 0.1, w.EmotionalSalience)

	sum := w.Recency + w.Importance + w.Relevance + w.EmotionalSalience
	assert.InDelta(t, 1.0, sum, 1e-9, "weights should sum to 1.0")
}

func TestComposeHooks(t *testing.T) {
	var prunedCalls, compressedCalls, cycleCalls int

	h1 := Hooks{
		OnPruned:        func(_ []Record) { prunedCalls++ },
		OnCompressed:    func(_, _ []Record) { compressedCalls++ },
		OnCycleComplete: func(_ CycleMetrics) { cycleCalls++ },
	}
	h2 := Hooks{
		OnPruned:        func(_ []Record) { prunedCalls++ },
		OnCycleComplete: func(_ CycleMetrics) { cycleCalls++ },
	}
	h3 := Hooks{} // all nil

	composed := ComposeHooks(h1, h2, h3)

	composed.OnPruned(nil)
	assert.Equal(t, 2, prunedCalls)

	composed.OnCompressed(nil, nil)
	assert.Equal(t, 1, compressedCalls)

	composed.OnCycleComplete(CycleMetrics{})
	assert.Equal(t, 2, cycleCalls)
}
