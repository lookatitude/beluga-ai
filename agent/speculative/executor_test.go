package speculative

import (
	"context"
	"errors"
	"iter"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mock types ---

// mockAgent implements agent.Agent for testing.
type mockAgent struct {
	id        string
	invokeFn  func(ctx context.Context, input string) (string, error)
	streamFn  func(ctx context.Context, input string) iter.Seq2[agent.Event, error]
	delay     time.Duration
	callCount atomic.Int64
}

func (m *mockAgent) ID() string              { return m.id }
func (m *mockAgent) Persona() agent.Persona  { return agent.Persona{Role: "test"} }
func (m *mockAgent) Tools() []tool.Tool      { return nil }
func (m *mockAgent) Children() []agent.Agent { return nil }

func (m *mockAgent) Invoke(ctx context.Context, input string, _ ...agent.Option) (string, error) {
	m.callCount.Add(1)
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	if m.invokeFn != nil {
		return m.invokeFn(ctx, input)
	}
	return "ground truth", nil
}

func (m *mockAgent) Stream(ctx context.Context, input string, _ ...agent.Option) iter.Seq2[agent.Event, error] {
	if m.streamFn != nil {
		return m.streamFn(ctx, input)
	}
	return func(yield func(agent.Event, error) bool) {
		if m.delay > 0 {
			select {
			case <-time.After(m.delay):
			case <-ctx.Done():
				yield(agent.Event{}, ctx.Err())
				return
			}
		}
		yield(agent.Event{Type: agent.EventText, Text: "streamed", AgentID: m.id}, nil)
		yield(agent.Event{Type: agent.EventDone, AgentID: m.id}, nil)
	}
}

// mockPredictor implements Predictor for testing.
type mockPredictor struct {
	prediction string
	confidence float64
	err        error
	delay      time.Duration
	callCount  atomic.Int64
}

func (p *mockPredictor) Predict(ctx context.Context, _ string) (string, float64, error) {
	p.callCount.Add(1)
	if p.delay > 0 {
		select {
		case <-time.After(p.delay):
		case <-ctx.Done():
			return "", 0, ctx.Err()
		}
	}
	return p.prediction, p.confidence, p.err
}

// mockValidator implements Validator for testing.
type mockValidator struct {
	valid bool
	err   error
}

func (v *mockValidator) Validate(_ context.Context, _, _ string) (bool, error) {
	return v.valid, v.err
}

// --- Tests ---

func TestSpeculativeExecutor_ID(t *testing.T) {
	gt := &mockAgent{id: "my-agent"}
	e := NewSpeculativeExecutor(gt)
	assert.Equal(t, "speculative-my-agent", e.ID())
}

func TestSpeculativeExecutor_Persona(t *testing.T) {
	gt := &mockAgent{id: "test"}
	e := NewSpeculativeExecutor(gt)
	assert.Equal(t, "test", e.Persona().Role)
}

func TestSpeculativeExecutor_Tools(t *testing.T) {
	gt := &mockAgent{id: "test"}
	e := NewSpeculativeExecutor(gt)
	assert.Nil(t, e.Tools())
}

func TestSpeculativeExecutor_Children(t *testing.T) {
	gt := &mockAgent{id: "test"}
	e := NewSpeculativeExecutor(gt)
	children := e.Children()
	require.Len(t, children, 1)
	assert.Equal(t, "test", children[0].ID())
}

func TestSpeculativeExecutor_Invoke(t *testing.T) {
	tests := []struct {
		name           string
		predictor      *mockPredictor
		validator      *mockValidator
		gtResult       string
		gtErr          error
		threshold      float64
		wantResult     string
		wantErr        bool
		wantHit        bool
		wantPredCalled bool
	}{
		{
			name:       "no predictor falls through to ground truth",
			predictor:  nil,
			gtResult:   "ground truth result",
			wantResult: "ground truth result",
		},
		{
			name:           "prediction hit: exact match",
			predictor:      &mockPredictor{prediction: "the answer", confidence: 0.9},
			validator:      &mockValidator{valid: true},
			gtResult:       "the answer",
			threshold:      0.7,
			wantResult:     "the answer",
			wantHit:        true,
			wantPredCalled: true,
		},
		{
			name:           "prediction miss: different result",
			predictor:      &mockPredictor{prediction: "wrong answer", confidence: 0.9},
			validator:      &mockValidator{valid: false},
			gtResult:       "correct answer",
			threshold:      0.7,
			wantResult:     "correct answer",
			wantHit:        false,
			wantPredCalled: true,
		},
		{
			name:           "low confidence: falls through to ground truth",
			predictor:      &mockPredictor{prediction: "maybe", confidence: 0.3},
			validator:      &mockValidator{valid: true},
			gtResult:       "definitely",
			threshold:      0.7,
			wantResult:     "definitely",
			wantHit:        false,
			wantPredCalled: true,
		},
		{
			name:           "predictor error: falls through to ground truth",
			predictor:      &mockPredictor{err: errors.New("predictor failed")},
			validator:      &mockValidator{valid: true},
			gtResult:       "ground truth",
			wantResult:     "ground truth",
			wantHit:        false,
			wantPredCalled: true,
		},
		{
			name:       "ground truth error propagated",
			predictor:  &mockPredictor{prediction: "pred", confidence: 0.9},
			validator:  &mockValidator{valid: false},
			gtErr:      errors.New("gt failed"),
			threshold:  0.7,
			wantResult: "",
			wantErr:    true,
		},
		{
			name:           "validator error: falls through to ground truth",
			predictor:      &mockPredictor{prediction: "pred", confidence: 0.9},
			validator:      &mockValidator{valid: false, err: errors.New("validation error")},
			gtResult:       "ground truth",
			threshold:      0.7,
			wantResult:     "ground truth",
			wantPredCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gt := &mockAgent{
				id: "gt",
				invokeFn: func(_ context.Context, _ string) (string, error) {
					return tt.gtResult, tt.gtErr
				},
			}

			var opts []ExecutorOption
			if tt.predictor != nil {
				opts = append(opts, WithPredictor(tt.predictor))
			}
			if tt.validator != nil {
				opts = append(opts, WithValidator(tt.validator))
			}
			if tt.threshold > 0 {
				opts = append(opts, WithConfidenceThreshold(tt.threshold))
			}

			e := NewSpeculativeExecutor(gt, opts...)
			result, err := e.Invoke(context.Background(), "test input")

			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantResult, result)

			if tt.wantPredCalled && tt.predictor != nil {
				assert.Equal(t, int64(1), tt.predictor.callCount.Load(), "predictor should be called once")
			}

			if tt.wantHit {
				assert.Equal(t, int64(1), e.Metrics().Hits())
			}
		})
	}
}

func TestSpeculativeExecutor_ParallelExecution(t *testing.T) {
	// Verify that predictor and ground truth run concurrently.
	var (
		predStarted atomic.Bool
		gtStarted   atomic.Bool
		bothRunning atomic.Bool
	)

	pred := &mockPredictor{
		prediction: "fast",
		confidence: 0.9,
	}
	// Override Predict to track concurrency.
	origPred := pred
	concurrentPredictor := &trackingPredictor{
		inner: origPred,
		onPredict: func() {
			predStarted.Store(true)
			// Spin briefly to let gt start.
			for i := 0; i < 100; i++ {
				if gtStarted.Load() {
					bothRunning.Store(true)
					break
				}
				time.Sleep(time.Millisecond)
			}
		},
	}

	gt := &mockAgent{
		id: "gt",
		invokeFn: func(_ context.Context, _ string) (string, error) {
			gtStarted.Store(true)
			// Spin briefly to let pred start.
			for i := 0; i < 100; i++ {
				if predStarted.Load() {
					bothRunning.Store(true)
					break
				}
				time.Sleep(time.Millisecond)
			}
			time.Sleep(10 * time.Millisecond) // simulate work
			return "fast", nil
		},
	}

	e := NewSpeculativeExecutor(gt,
		WithPredictor(concurrentPredictor),
		WithValidator(&mockValidator{valid: true}),
		WithConfidenceThreshold(0.5),
	)

	result, err := e.Invoke(context.Background(), "test")
	require.NoError(t, err)
	assert.Equal(t, "fast", result)
	assert.True(t, bothRunning.Load(), "predictor and ground truth should run concurrently")
}

// trackingPredictor wraps a Predictor to observe execution.
type trackingPredictor struct {
	inner     Predictor
	onPredict func()
}

func (p *trackingPredictor) Predict(ctx context.Context, input string) (string, float64, error) {
	if p.onPredict != nil {
		p.onPredict()
	}
	return p.inner.Predict(ctx, input)
}

func TestSpeculativeExecutor_Stream_GroundTruth(t *testing.T) {
	gt := &mockAgent{id: "gt"}
	e := NewSpeculativeExecutor(gt) // no predictor

	var events []agent.Event
	for event, err := range e.Stream(context.Background(), "test") {
		require.NoError(t, err)
		events = append(events, event)
	}

	require.Len(t, events, 2)
	assert.Equal(t, agent.EventText, events[0].Type)
	assert.Equal(t, "streamed", events[0].Text)
	assert.Equal(t, agent.EventDone, events[1].Type)
}

func TestSpeculativeExecutor_Stream_HighConfidence(t *testing.T) {
	gt := &mockAgent{id: "gt"}
	fast := &mockAgent{
		id: "fast",
		streamFn: func(_ context.Context, _ string) iter.Seq2[agent.Event, error] {
			return func(yield func(agent.Event, error) bool) {
				yield(agent.Event{Type: agent.EventText, Text: "fast-stream", AgentID: "fast"}, nil)
				yield(agent.Event{Type: agent.EventDone, AgentID: "fast"}, nil)
			}
		},
	}

	e := NewSpeculativeExecutor(gt,
		WithPredictor(&mockPredictor{prediction: "pred", confidence: 0.9}),
		WithFastAgent(fast),
		WithConfidenceThreshold(0.7),
	)

	var events []agent.Event
	for event, err := range e.Stream(context.Background(), "test") {
		require.NoError(t, err)
		events = append(events, event)
	}

	require.Len(t, events, 2)
	assert.Equal(t, "fast-stream", events[0].Text)
}

func TestSpeculativeExecutor_Stream_LowConfidence(t *testing.T) {
	gt := &mockAgent{id: "gt"}
	fast := &mockAgent{id: "fast"}

	e := NewSpeculativeExecutor(gt,
		WithPredictor(&mockPredictor{prediction: "pred", confidence: 0.3}),
		WithFastAgent(fast),
		WithConfidenceThreshold(0.7),
	)

	var events []agent.Event
	for event, err := range e.Stream(context.Background(), "test") {
		require.NoError(t, err)
		events = append(events, event)
	}

	// Should use ground truth (gt), not fast agent.
	require.Len(t, events, 2)
	assert.Equal(t, "streamed", events[0].Text)
	assert.Equal(t, "gt", events[0].AgentID)
}

func TestSpeculativeExecutor_Stream_PredictorError(t *testing.T) {
	gt := &mockAgent{id: "gt"}
	fast := &mockAgent{id: "fast"}

	var cancelHookCalled atomic.Bool
	e := NewSpeculativeExecutor(gt,
		WithPredictor(&mockPredictor{err: errors.New("fail")}),
		WithFastAgent(fast),
		WithHooks(Hooks{
			OnCancel: func(_ context.Context, _ error) {
				cancelHookCalled.Store(true)
			},
		}),
	)

	var events []agent.Event
	for event, err := range e.Stream(context.Background(), "test") {
		require.NoError(t, err)
		events = append(events, event)
	}

	// Should fall through to ground truth.
	require.Len(t, events, 2)
	assert.Equal(t, "streamed", events[0].Text)
	assert.True(t, cancelHookCalled.Load())
}

func TestSpeculativeExecutor_Stream_ContextCancellation(t *testing.T) {
	gt := &mockAgent{
		id: "gt",
		streamFn: func(ctx context.Context, _ string) iter.Seq2[agent.Event, error] {
			return func(yield func(agent.Event, error) bool) {
				for i := 0; i < 10; i++ {
					select {
					case <-ctx.Done():
						yield(agent.Event{}, ctx.Err())
						return
					case <-time.After(10 * time.Millisecond):
					}
					if !yield(agent.Event{Type: agent.EventText, Text: "chunk", AgentID: "gt"}, nil) {
						return
					}
				}
			}
		},
	}

	e := NewSpeculativeExecutor(gt)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var count int
	for _, err := range e.Stream(ctx, "test") {
		count++
		if count == 2 {
			cancel()
		}
		if err != nil {
			assert.ErrorIs(t, err, context.Canceled)
			break
		}
	}
	assert.LessOrEqual(t, count, 4)
}

func TestSpeculativeExecutor_Invoke_ContextCancellation(t *testing.T) {
	gt := &mockAgent{
		id:    "gt",
		delay: 5 * time.Second,
	}
	pred := &mockPredictor{
		prediction: "fast",
		confidence: 0.9,
		delay:      5 * time.Second,
	}

	e := NewSpeculativeExecutor(gt, WithPredictor(pred))

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := e.Invoke(ctx, "test")
	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestSpeculativeExecutor_Hooks(t *testing.T) {
	var (
		predictionCalled    atomic.Bool
		validationCalled    atomic.Bool
		mispredictionCalled atomic.Bool
	)

	t.Run("prediction hit fires OnValidation", func(t *testing.T) {
		predictionCalled.Store(false)
		validationCalled.Store(false)

		gt := &mockAgent{
			id: "gt",
			invokeFn: func(_ context.Context, _ string) (string, error) {
				return "answer", nil
			},
		}

		e := NewSpeculativeExecutor(gt,
			WithPredictor(&mockPredictor{prediction: "answer", confidence: 0.9}),
			WithValidator(&mockValidator{valid: true}),
			WithConfidenceThreshold(0.7),
			WithHooks(Hooks{
				OnPrediction: func(_ context.Context, _ string, _ float64) {
					predictionCalled.Store(true)
				},
				OnValidation: func(_ context.Context, _ Result) {
					validationCalled.Store(true)
				},
			}),
		)

		result, err := e.Invoke(context.Background(), "test")
		require.NoError(t, err)
		assert.Equal(t, "answer", result)
		assert.True(t, predictionCalled.Load())
		assert.True(t, validationCalled.Load())
	})

	t.Run("prediction miss fires OnMisprediction", func(t *testing.T) {
		mispredictionCalled.Store(false)

		gt := &mockAgent{
			id: "gt",
			invokeFn: func(_ context.Context, _ string) (string, error) {
				return "correct", nil
			},
		}

		e := NewSpeculativeExecutor(gt,
			WithPredictor(&mockPredictor{prediction: "wrong", confidence: 0.9}),
			WithValidator(&mockValidator{valid: false}),
			WithConfidenceThreshold(0.7),
			WithHooks(Hooks{
				OnMisprediction: func(_ context.Context, r Result) {
					mispredictionCalled.Store(true)
					assert.Equal(t, "wrong", r.Prediction)
					assert.Equal(t, "correct", r.GroundTruth)
				},
			}),
		)

		result, err := e.Invoke(context.Background(), "test")
		require.NoError(t, err)
		assert.Equal(t, "correct", result)
		assert.True(t, mispredictionCalled.Load())
	})
}

func TestSpeculativeExecutor_Metrics(t *testing.T) {
	metrics := NewMetrics()

	gt := &mockAgent{
		id: "gt",
		invokeFn: func(_ context.Context, _ string) (string, error) {
			return "answer", nil
		},
	}

	e := NewSpeculativeExecutor(gt,
		WithPredictor(&mockPredictor{prediction: "answer", confidence: 0.9}),
		WithValidator(&mockValidator{valid: true}),
		WithConfidenceThreshold(0.7),
		WithMetrics(metrics),
	)

	// Run multiple invocations.
	for i := 0; i < 3; i++ {
		_, err := e.Invoke(context.Background(), "test")
		require.NoError(t, err)
	}

	assert.Equal(t, int64(3), metrics.Predictions())
	assert.Equal(t, int64(3), metrics.Hits())
	assert.Equal(t, int64(0), metrics.Misses())
	assert.Equal(t, 1.0, metrics.HitRate())
}

func TestSpeculativeExecutor_Metrics_ConcurrentAccess(t *testing.T) {
	metrics := NewMetrics()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			metrics.RecordHit(time.Millisecond)
		}()
		go func() {
			defer wg.Done()
			metrics.RecordMiss(10)
		}()
	}

	wg.Wait()
	assert.Equal(t, int64(200), metrics.Predictions())
	assert.Equal(t, int64(100), metrics.Hits())
	assert.Equal(t, int64(100), metrics.Misses())
	assert.Equal(t, int64(1000), metrics.WastedTokens())

	snap := metrics.Snapshot()
	assert.Equal(t, int64(200), snap.Predictions)
	assert.Equal(t, 0.5, snap.HitRate())
}

// --- Validator Tests ---

func TestExactValidator(t *testing.T) {
	tests := []struct {
		name       string
		prediction string
		truth      string
		want       bool
	}{
		{name: "exact match", prediction: "hello", truth: "hello", want: true},
		{name: "case insensitive", prediction: "Hello", truth: "hello", want: true},
		{name: "whitespace trimmed", prediction: "  hello  ", truth: "hello", want: true},
		{name: "different", prediction: "hello", truth: "world", want: false},
		{name: "empty both", prediction: "", truth: "", want: true},
		{name: "empty vs non-empty", prediction: "", truth: "hello", want: false},
	}

	v := NewExactValidator()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, err := v.Validate(context.Background(), tt.prediction, tt.truth)
			require.NoError(t, err)
			assert.Equal(t, tt.want, valid)
		})
	}
}

func TestSemanticValidator(t *testing.T) {
	tests := []struct {
		name       string
		prediction string
		truth      string
		threshold  float64
		want       bool
	}{
		{name: "identical text", prediction: "the quick brown fox", truth: "the quick brown fox", threshold: 0.9, want: true},
		{name: "similar text", prediction: "the quick brown fox jumps", truth: "the quick brown fox leaps", threshold: 0.5, want: true},
		{name: "completely different", prediction: "hello world", truth: "foo bar baz", threshold: 0.9, want: false},
		{name: "both empty", prediction: "", truth: "", threshold: 0.9, want: true},
		{name: "one empty", prediction: "hello", truth: "", threshold: 0.1, want: false},
		{name: "zero threshold", prediction: "a", truth: "b", threshold: 0.0, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewSemanticValidator(tt.threshold)
			valid, err := v.Validate(context.Background(), tt.prediction, tt.truth)
			require.NoError(t, err)
			assert.Equal(t, tt.want, valid)
		})
	}
}

// --- Predictor Tests ---

func TestEstimateConfidence(t *testing.T) {
	tests := []struct {
		name string
		text string
		want float64
	}{
		{name: "empty", text: "", want: 0.0},
		{name: "very short", text: "yes", want: 0.95},
		{name: "short", text: "the answer is four", want: 0.95},
		{name: "medium", text: strings.Repeat("word ", 15), want: 0.8},
		{name: "long", text: strings.Repeat("word ", 40), want: 0.6},
		{name: "very long", text: strings.Repeat("word ", 80), want: 0.45},
		{name: "extremely long", text: strings.Repeat("word ", 200), want: 0.3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := estimateConfidence(tt.text)
			assert.Equal(t, tt.want, got)
		})
	}
}

// --- Cosine Similarity Tests ---

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want float64
	}{
		{name: "identical", a: "hello world", b: "hello world", want: 1.0},
		{name: "both empty", a: "", b: "", want: 1.0},
		{name: "one empty", a: "hello", b: "", want: 0.0},
		{name: "completely different", a: "foo bar", b: "baz qux", want: 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cosineSimilarity(tt.a, tt.b)
			assert.InDelta(t, tt.want, got, 0.01)
		})
	}
}

// --- ComposeHooks Tests ---

func TestComposeHooks(t *testing.T) {
	var calls []string
	var mu sync.Mutex

	record := func(s string) {
		mu.Lock()
		defer mu.Unlock()
		calls = append(calls, s)
	}

	h1 := Hooks{
		OnPrediction: func(_ context.Context, _ string, _ float64) { record("h1-pred") },
		OnValidation: func(_ context.Context, _ Result) { record("h1-valid") },
	}
	h2 := Hooks{
		OnPrediction:    func(_ context.Context, _ string, _ float64) { record("h2-pred") },
		OnMisprediction: func(_ context.Context, _ Result) { record("h2-mispred") },
	}

	composed := ComposeHooks(h1, h2)

	composed.OnPrediction(context.Background(), "test", 0.9)
	composed.OnValidation(context.Background(), Result{})
	composed.OnMisprediction(context.Background(), Result{})
	composed.OnCancel(context.Background(), nil) // should not panic with no hooks

	assert.Equal(t, []string{"h1-pred", "h2-pred", "h1-valid", "h2-mispred"}, calls)
}

// --- MetricsSnapshot Tests ---

func TestMetricsSnapshot_HitRate(t *testing.T) {
	tests := []struct {
		name string
		snap MetricsSnapshot
		want float64
	}{
		{name: "no predictions", snap: MetricsSnapshot{}, want: 0},
		{name: "all hits", snap: MetricsSnapshot{Predictions: 10, Hits: 10}, want: 1.0},
		{name: "half hits", snap: MetricsSnapshot{Predictions: 10, Hits: 5}, want: 0.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.snap.HitRate())
		})
	}
}

// --- Compile-time checks ---

var _ agent.Agent = (*SpeculativeExecutor)(nil)
var _ Predictor = (*LightModelPredictor)(nil)
var _ Predictor = (*mockPredictor)(nil)
var _ Validator = (*ExactValidator)(nil)
var _ Validator = (*SemanticValidator)(nil)
var _ Validator = (*mockValidator)(nil)
