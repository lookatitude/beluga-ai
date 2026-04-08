package consolidation

import (
	"context"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorker_StartStop(t *testing.T) {
	store := NewInMemoryConsolidationStore()
	w := NewWorker(store, WithInterval(time.Second))

	ctx := context.Background()
	err := w.Start(ctx)
	require.NoError(t, err)

	health := w.Health()
	assert.Equal(t, core.HealthHealthy, health.Status)

	err = w.Stop(ctx)
	require.NoError(t, err)

	health = w.Health()
	assert.Equal(t, core.HealthUnhealthy, health.Status)
	assert.Equal(t, "stopped", health.Message)
}

func TestWorker_DoubleStart(t *testing.T) {
	store := NewInMemoryConsolidationStore()
	w := NewWorker(store, WithInterval(time.Second))

	ctx := context.Background()
	require.NoError(t, w.Start(ctx))
	defer func() { _ = w.Stop(ctx) }()

	err := w.Start(ctx)
	assert.Error(t, err)
}

func TestWorker_StopWithoutStart(t *testing.T) {
	store := NewInMemoryConsolidationStore()
	w := NewWorker(store)

	err := w.Stop(context.Background())
	assert.NoError(t, err)
}

func TestWorker_RunCycle_Prune(t *testing.T) {
	now := time.Now()
	store := newTestStore(
		Record{ID: "keep", CreatedAt: now, Utility: UtilityScore{Importance: 1.0, Relevance: 1.0, EmotionalSalience: 1.0}},
		Record{ID: "prune", CreatedAt: now.Add(-365 * 24 * time.Hour), Utility: UtilityScore{}},
	)

	var prunedIDs []string
	hooks := Hooks{
		OnPruned: func(records []Record) {
			for _, r := range records {
				prunedIDs = append(prunedIDs, r.ID)
			}
		},
	}

	w := NewWorker(store, WithHooks(hooks))
	metrics := w.runCycle(context.Background())

	assert.Equal(t, 2, metrics.RecordsEvaluated)
	assert.GreaterOrEqual(t, metrics.RecordsPruned, 1)
	assert.Contains(t, prunedIDs, "prune")
	assert.NotContains(t, prunedIDs, "keep")
	assert.Empty(t, metrics.Errors)

	// Verify pruned record is gone from store.
	assert.Equal(t, 1, store.Len())
	_, ok := store.Get("keep")
	assert.True(t, ok)
	_, ok = store.Get("prune")
	assert.False(t, ok)
}

func TestWorker_RunCycle_Compress(t *testing.T) {
	now := time.Now()
	store := newTestStore(
		Record{ID: "compress-me", CreatedAt: now, Content: "long content", Utility: UtilityScore{
			Importance: 0.1, Relevance: 0.1, EmotionalSalience: 0.1,
		}},
	)

	policy := &ThresholdPolicy{
		Threshold:         0.5,
		CompressThreshold: 0.2,
		Weights:           DefaultWeights(),
	}

	compressor := NewSummaryCompressor(&mockChatModel{response: "compressed"})

	var compressedCount int
	hooks := Hooks{
		OnCompressed: func(_, _ []Record) {
			compressedCount++
		},
	}

	w := NewWorker(store,
		WithPolicy(policy),
		WithCompressor(compressor),
		WithHooks(hooks),
	)

	metrics := w.runCycle(context.Background())
	assert.Equal(t, 1, metrics.RecordsEvaluated)
	assert.Equal(t, 1, metrics.RecordsCompressed)
	assert.Equal(t, 1, compressedCount)
	assert.Empty(t, metrics.Errors)

	// Verify content was updated in store.
	r, ok := store.Get("compress-me")
	require.True(t, ok)
	assert.Equal(t, "compressed", r.Content)
}

func TestWorker_RunCycle_CompressFallbackToPrune(t *testing.T) {
	now := time.Now()
	store := newTestStore(
		Record{ID: "no-compressor", CreatedAt: now, Utility: UtilityScore{
			Importance: 0.1, Relevance: 0.1, EmotionalSalience: 0.1,
		}},
	)

	policy := &ThresholdPolicy{
		Threshold:         0.5,
		CompressThreshold: 0.2,
		Weights:           DefaultWeights(),
	}

	// No compressor set, so compress actions should fall back to prune.
	w := NewWorker(store, WithPolicy(policy))
	metrics := w.runCycle(context.Background())

	assert.Equal(t, 1, metrics.RecordsPruned)
	assert.Equal(t, 0, metrics.RecordsCompressed)
	assert.Equal(t, 0, store.Len())
}

func TestWorker_RunCycle_EmptyStore(t *testing.T) {
	store := NewInMemoryConsolidationStore()
	w := NewWorker(store)

	metrics := w.runCycle(context.Background())
	assert.Equal(t, 0, metrics.RecordsEvaluated)
	assert.Empty(t, metrics.Errors)
}

func TestWorker_RunCycle_ContextCancelled(t *testing.T) {
	now := time.Now()
	store := newTestStore(
		Record{ID: "a", CreatedAt: now},
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	w := NewWorker(store)
	metrics := w.runCycle(ctx)

	// Should get an error from the cancelled context.
	assert.NotEmpty(t, metrics.Errors)
}

func TestWorker_ContextCancelledStopsLoop(t *testing.T) {
	store := NewInMemoryConsolidationStore()
	w := NewWorker(store, WithInterval(time.Second))

	ctx, cancel := context.WithCancel(context.Background())
	require.NoError(t, w.Start(ctx))
	cancel()

	// Give the goroutine time to exit.
	time.Sleep(100 * time.Millisecond)

	// Stop should work cleanly even after context cancellation.
	err := w.Stop(context.Background())
	assert.NoError(t, err)
}

func TestWorker_Options(t *testing.T) {
	store := NewInMemoryConsolidationStore()

	t.Run("WithInterval clamps minimum", func(t *testing.T) {
		w := NewWorker(store, WithInterval(time.Millisecond))
		assert.Equal(t, time.Second, w.opts.interval)
	})

	t.Run("WithMaxRecordsPerCycle", func(t *testing.T) {
		w := NewWorker(store, WithMaxRecordsPerCycle(42))
		assert.Equal(t, 42, w.opts.maxRecordsPerCycle)
	})

	t.Run("WithMaxRecordsPerCycle ignores zero", func(t *testing.T) {
		w := NewWorker(store, WithMaxRecordsPerCycle(0))
		assert.Equal(t, defaultMaxRecords, w.opts.maxRecordsPerCycle)
	})
}

func TestWorker_LifecycleInterface(t *testing.T) {
	var lc core.Lifecycle = NewWorker(NewInMemoryConsolidationStore())
	_ = lc
}

func TestWorker_HooksIntegration(t *testing.T) {
	now := time.Now()
	store := newTestStore(
		Record{ID: "old", CreatedAt: now.Add(-365 * 24 * time.Hour), Utility: UtilityScore{}},
	)

	var cycleMetrics CycleMetrics
	hooks := Hooks{
		OnCycleComplete: func(m CycleMetrics) {
			cycleMetrics = m
		},
	}

	w := NewWorker(store, WithHooks(hooks))

	// runCycle calls OnCycleComplete via the hook in the worker loop,
	// but runCycle itself does not call it. We call it manually here
	// to test the hook mechanism.
	metrics := w.runCycle(context.Background())

	// OnCycleComplete is called from the loop, not runCycle directly,
	// so we invoke it ourselves.
	if w.opts.hooks.OnCycleComplete != nil {
		w.opts.hooks.OnCycleComplete(metrics)
	}

	assert.Equal(t, 1, cycleMetrics.RecordsEvaluated)
	assert.GreaterOrEqual(t, cycleMetrics.RecordsPruned, 1)
}
