package degradation

import (
	"context"
	"fmt"
	"iter"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Level tests
// ---------------------------------------------------------------------------

func TestAutonomyLevel_String(t *testing.T) {
	tests := []struct {
		level AutonomyLevel
		want  string
	}{
		{Full, "full"},
		{Restricted, "restricted"},
		{ReadOnly, "read_only"},
		{Sequestered, "sequestered"},
		{AutonomyLevel(99), "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.level.String())
		})
	}
}

func TestLevelCapabilities(t *testing.T) {
	tests := []struct {
		level AutonomyLevel
		want  Capabilities
	}{
		{
			level: Full,
			want: Capabilities{
				CanExecuteTools:  true,
				ToolsAllowlisted: false,
				CanWrite:         true,
				CanRespond:       true,
			},
		},
		{
			level: Restricted,
			want: Capabilities{
				CanExecuteTools:  true,
				ToolsAllowlisted: true,
				CanWrite:         true,
				CanRespond:       true,
			},
		},
		{
			level: ReadOnly,
			want: Capabilities{
				CanExecuteTools:  false,
				ToolsAllowlisted: false,
				CanWrite:         false,
				CanRespond:       true,
			},
		},
		{
			level: Sequestered,
			want: Capabilities{
				CanExecuteTools:  false,
				ToolsAllowlisted: false,
				CanWrite:         false,
				CanRespond:       false,
			},
		},
		{
			level: AutonomyLevel(99),
			want: Capabilities{
				CanExecuteTools:  false,
				ToolsAllowlisted: false,
				CanWrite:         false,
				CanRespond:       false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.level.String(), func(t *testing.T) {
			assert.Equal(t, tt.want, LevelCapabilities(tt.level))
		})
	}
}

// ---------------------------------------------------------------------------
// Monitor tests
// ---------------------------------------------------------------------------

func TestSecurityMonitor_RecordEvent_ClampsSeverity(t *testing.T) {
	now := time.Now()
	m := NewSecurityMonitor(withNowFunc(func() time.Time { return now }))

	ctx := context.Background()
	m.RecordEvent(ctx, SecurityEvent{Type: EventGuardBlocked, Severity: -0.5, Timestamp: now})
	m.RecordEvent(ctx, SecurityEvent{Type: EventGuardBlocked, Severity: 2.0, Timestamp: now})

	// Both events should be clamped; max possible is 1.0.
	sev := m.CurrentSeverity()
	assert.LessOrEqual(t, sev, 1.0)
	assert.GreaterOrEqual(t, sev, 0.0)
}

func TestSecurityMonitor_CurrentSeverity_Empty(t *testing.T) {
	m := NewSecurityMonitor()
	assert.Equal(t, 0.0, m.CurrentSeverity())
}

func TestSecurityMonitor_CurrentSeverity_Decay(t *testing.T) {
	window := 10 * time.Second
	now := time.Now()
	m := NewSecurityMonitor(
		WithWindowSize(window),
		withNowFunc(func() time.Time { return now }),
	)

	// Event at the start of the window (full decay).
	m.RecordEvent(context.Background(), SecurityEvent{
		Type:      EventGuardBlocked,
		Severity:  0.5,
		Timestamp: now.Add(-9 * time.Second), // 9s ago, weight ~0.1
	})

	sev := m.CurrentSeverity()
	// With linear decay, 9s into a 10s window gives weight 0.1.
	assert.InDelta(t, 0.05, sev, 0.01)
}

func TestSecurityMonitor_CurrentSeverity_RecentEvent(t *testing.T) {
	now := time.Now()
	m := NewSecurityMonitor(
		WithWindowSize(10*time.Second),
		withNowFunc(func() time.Time { return now }),
	)

	// Very recent event should have weight close to 1.0.
	m.RecordEvent(context.Background(), SecurityEvent{
		Type:      EventInjectionDetected,
		Severity:  0.8,
		Timestamp: now,
	})

	sev := m.CurrentSeverity()
	assert.InDelta(t, 0.8, sev, 0.01)
}

func TestSecurityMonitor_EventsExpire(t *testing.T) {
	window := 1 * time.Second
	currentTime := time.Now()
	m := NewSecurityMonitor(
		WithWindowSize(window),
		withNowFunc(func() time.Time { return currentTime }),
	)

	m.RecordEvent(context.Background(), SecurityEvent{
		Type:      EventGuardBlocked,
		Severity:  0.9,
		Timestamp: currentTime,
	})
	assert.Equal(t, 1, m.EventCount())

	// Advance time past the window.
	currentTime = currentTime.Add(2 * time.Second)
	assert.Equal(t, 0, m.EventCount())
	assert.Equal(t, 0.0, m.CurrentSeverity())
}

func TestSecurityMonitor_Reset(t *testing.T) {
	m := NewSecurityMonitor()
	m.RecordEvent(context.Background(), SecurityEvent{
		Type: EventToolAbuse, Severity: 0.5, Timestamp: time.Now(),
	})
	assert.Equal(t, 1, m.EventCount())

	m.Reset()
	assert.Equal(t, 0, m.EventCount())
	assert.Equal(t, 0.0, m.CurrentSeverity())
}

func TestSecurityMonitor_DefaultTimestamp(t *testing.T) {
	now := time.Now()
	m := NewSecurityMonitor(withNowFunc(func() time.Time { return now }))
	m.RecordEvent(context.Background(), SecurityEvent{
		Type:     EventCustom,
		Severity: 0.3,
		// Timestamp left zero; should be set to now.
	})
	assert.Equal(t, 1, m.EventCount())
	sev := m.CurrentSeverity()
	assert.InDelta(t, 0.3, sev, 0.01)
}

func TestSecurityMonitor_ConcurrentAccess(t *testing.T) {
	m := NewSecurityMonitor(WithWindowSize(5 * time.Second))
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.RecordEvent(ctx, SecurityEvent{
				Type:     EventGuardBlocked,
				Severity: 0.1,
			})
			_ = m.CurrentSeverity()
			_ = m.EventCount()
		}()
	}
	wg.Wait()

	assert.Equal(t, 100, m.EventCount())
}

// ---------------------------------------------------------------------------
// Policy tests
// ---------------------------------------------------------------------------

func TestThresholdPolicy_Evaluate(t *testing.T) {
	p := NewThresholdPolicy()

	tests := []struct {
		severity float64
		want     AutonomyLevel
	}{
		{0.0, Full},
		{0.1, Full},
		{0.29, Full},
		{0.3, Restricted},
		{0.5, Restricted},
		{0.59, Restricted},
		{0.6, ReadOnly},
		{0.7, ReadOnly},
		{0.84, ReadOnly},
		{0.85, Sequestered},
		{1.0, Sequestered},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("severity_%.2f", tt.severity), func(t *testing.T) {
			assert.Equal(t, tt.want, p.Evaluate(tt.severity))
		})
	}
}

func TestThresholdPolicy_CustomThresholds(t *testing.T) {
	p := NewThresholdPolicy(WithLevelThresholds(LevelThresholds{
		Restricted:  0.1,
		ReadOnly:    0.5,
		Sequestered: 0.9,
	}))

	assert.Equal(t, Full, p.Evaluate(0.05))
	assert.Equal(t, Restricted, p.Evaluate(0.1))
	assert.Equal(t, ReadOnly, p.Evaluate(0.5))
	assert.Equal(t, Sequestered, p.Evaluate(0.9))
}

// ---------------------------------------------------------------------------
// Hooks tests
// ---------------------------------------------------------------------------

func TestComposeHooks_NilFields(t *testing.T) {
	composed := ComposeHooks(Hooks{}, Hooks{})
	assert.Nil(t, composed.OnLevelChanged)
	assert.Nil(t, composed.OnAnomalyDetected)
	assert.Nil(t, composed.OnRecovery)
}

func TestComposeHooks_CallsAll(t *testing.T) {
	var calls []string

	h1 := Hooks{
		OnLevelChanged: func(_ context.Context, _, _ AutonomyLevel) {
			calls = append(calls, "h1-level")
		},
		OnAnomalyDetected: func(_ context.Context, _ SecurityEvent) {
			calls = append(calls, "h1-anomaly")
		},
	}
	h2 := Hooks{
		OnLevelChanged: func(_ context.Context, _, _ AutonomyLevel) {
			calls = append(calls, "h2-level")
		},
		OnRecovery: func(_ context.Context, _, _ AutonomyLevel) {
			calls = append(calls, "h2-recovery")
		},
	}

	composed := ComposeHooks(h1, h2)

	composed.OnLevelChanged(context.Background(), Full, Restricted)
	assert.Equal(t, []string{"h1-level", "h2-level"}, calls)

	calls = nil
	composed.OnAnomalyDetected(context.Background(), SecurityEvent{})
	assert.Equal(t, []string{"h1-anomaly"}, calls)

	calls = nil
	composed.OnRecovery(context.Background(), Restricted, Full)
	assert.Equal(t, []string{"h2-recovery"}, calls)
}

// ---------------------------------------------------------------------------
// Degrader tests
// ---------------------------------------------------------------------------

// mockAgent is a minimal agent.Agent implementation for testing.
type mockAgent struct {
	id        string
	tools     []tool.Tool
	invokeRes string
	invokeErr error
	streamFn  func(ctx context.Context, input string) iter.Seq2[agent.Event, error]
}

func (m *mockAgent) ID() string              { return m.id }
func (m *mockAgent) Persona() agent.Persona  { return agent.Persona{} }
func (m *mockAgent) Tools() []tool.Tool      { return m.tools }
func (m *mockAgent) Children() []agent.Agent { return nil }
func (m *mockAgent) Invoke(ctx context.Context, input string, _ ...agent.Option) (string, error) {
	return m.invokeRes, m.invokeErr
}
func (m *mockAgent) Stream(ctx context.Context, input string, _ ...agent.Option) iter.Seq2[agent.Event, error] {
	if m.streamFn != nil {
		return m.streamFn(ctx, input)
	}
	return func(yield func(agent.Event, error) bool) {
		yield(agent.Event{Type: agent.EventText, Text: "hello"}, nil)
	}
}

var _ agent.Agent = (*mockAgent)(nil)

// mockTool is a minimal tool.Tool for testing.
type mockTool struct {
	name string
}

func (t *mockTool) Name() string                { return t.name }
func (t *mockTool) Description() string         { return "" }
func (t *mockTool) InputSchema() map[string]any { return nil }
func (t *mockTool) Execute(_ context.Context, _ map[string]any) (*tool.Result, error) {
	return tool.TextResult("ok"), nil
}

var _ tool.Tool = (*mockTool)(nil)

func TestDegrader_FullLevel_AllToolsAvailable(t *testing.T) {
	monitor := NewSecurityMonitor()
	policy := NewThresholdPolicy()
	degrader := NewRuntimeDegrader(monitor, policy,
		WithToolAllowlist("search"),
	)

	inner := &mockAgent{
		id:        "test-agent",
		tools:     []tool.Tool{&mockTool{name: "search"}, &mockTool{name: "write"}},
		invokeRes: "response",
	}

	wrapped := agent.ApplyMiddleware(inner, degrader.Middleware())
	tools := wrapped.Tools()
	assert.Len(t, tools, 2)

	result, err := wrapped.Invoke(context.Background(), "hello")
	require.NoError(t, err)
	assert.Equal(t, "response", result)
}

func TestDegrader_RestrictedLevel_FilteredTools(t *testing.T) {
	now := time.Now()
	monitor := NewSecurityMonitor(withNowFunc(func() time.Time { return now }))
	policy := NewThresholdPolicy()
	degrader := NewRuntimeDegrader(monitor, policy,
		WithToolAllowlist("search"),
	)

	// Push severity into Restricted range (0.3-0.6).
	monitor.RecordEvent(context.Background(), SecurityEvent{
		Type:      EventGuardBlocked,
		Severity:  0.4,
		Timestamp: now,
	})

	inner := &mockAgent{
		id:    "test-agent",
		tools: []tool.Tool{&mockTool{name: "search"}, &mockTool{name: "write"}},
	}

	wrapped := agent.ApplyMiddleware(inner, degrader.Middleware())
	tools := wrapped.Tools()
	require.Len(t, tools, 1)
	assert.Equal(t, "search", tools[0].Name())
}

func TestDegrader_ReadOnlyLevel_NoTools(t *testing.T) {
	now := time.Now()
	monitor := NewSecurityMonitor(withNowFunc(func() time.Time { return now }))
	policy := NewThresholdPolicy()
	degrader := NewRuntimeDegrader(monitor, policy)

	// Push severity into ReadOnly range (0.6-0.85).
	monitor.RecordEvent(context.Background(), SecurityEvent{
		Type:      EventInjectionDetected,
		Severity:  0.7,
		Timestamp: now,
	})

	inner := &mockAgent{
		id:        "test-agent",
		tools:     []tool.Tool{&mockTool{name: "search"}},
		invokeRes: "still responds",
	}

	wrapped := agent.ApplyMiddleware(inner, degrader.Middleware())
	assert.Empty(t, wrapped.Tools())

	// Can still invoke (respond).
	result, err := wrapped.Invoke(context.Background(), "hello")
	require.NoError(t, err)
	assert.Equal(t, "still responds", result)
}

func TestDegrader_SequesteredLevel_BlocksInvoke(t *testing.T) {
	now := time.Now()
	monitor := NewSecurityMonitor(withNowFunc(func() time.Time { return now }))
	policy := NewThresholdPolicy()
	degrader := NewRuntimeDegrader(monitor, policy)

	// Push severity into Sequestered range (>= 0.85).
	monitor.RecordEvent(context.Background(), SecurityEvent{
		Type:      EventInjectionDetected,
		Severity:  0.95,
		Timestamp: now,
	})

	inner := &mockAgent{id: "test-agent", invokeRes: "should not see this"}
	wrapped := agent.ApplyMiddleware(inner, degrader.Middleware())

	_, err := wrapped.Invoke(context.Background(), "hello")
	require.Error(t, err)

	var coreErr *core.Error
	require.ErrorAs(t, err, &coreErr)
	assert.Equal(t, core.ErrGuardBlocked, coreErr.Code)
}

func TestDegrader_SequesteredLevel_BlocksStream(t *testing.T) {
	now := time.Now()
	monitor := NewSecurityMonitor(withNowFunc(func() time.Time { return now }))
	policy := NewThresholdPolicy()
	degrader := NewRuntimeDegrader(monitor, policy)

	monitor.RecordEvent(context.Background(), SecurityEvent{
		Type:      EventToolAbuse,
		Severity:  0.95,
		Timestamp: now,
	})

	inner := &mockAgent{id: "test-agent"}
	wrapped := agent.ApplyMiddleware(inner, degrader.Middleware())

	var gotErr error
	for _, err := range wrapped.Stream(context.Background(), "hello") {
		if err != nil {
			gotErr = err
			break
		}
	}
	require.Error(t, gotErr)

	var coreErr *core.Error
	require.ErrorAs(t, gotErr, &coreErr)
	assert.Equal(t, core.ErrGuardBlocked, coreErr.Code)
}

func TestDegrader_Recovery(t *testing.T) {
	currentTime := time.Now()
	monitor := NewSecurityMonitor(
		WithWindowSize(1*time.Second),
		withNowFunc(func() time.Time { return currentTime }),
	)
	policy := NewThresholdPolicy()

	var transitions []string
	degrader := NewRuntimeDegrader(monitor, policy,
		WithHooks(Hooks{
			OnLevelChanged: func(_ context.Context, prev, next AutonomyLevel) {
				transitions = append(transitions, fmt.Sprintf("%s->%s", prev, next))
			},
			OnRecovery: func(_ context.Context, prev, next AutonomyLevel) {
				transitions = append(transitions, fmt.Sprintf("recovery:%s->%s", prev, next))
			},
		}),
	)

	ctx := context.Background()

	// Push to Sequestered.
	monitor.RecordEvent(ctx, SecurityEvent{
		Type:      EventInjectionDetected,
		Severity:  0.95,
		Timestamp: currentTime,
	})
	level := degrader.Evaluate(ctx)
	assert.Equal(t, Sequestered, level)

	// Advance time past the window so events expire.
	currentTime = currentTime.Add(2 * time.Second)
	level = degrader.Evaluate(ctx)
	assert.Equal(t, Full, level)

	// Verify hooks fired.
	require.Len(t, transitions, 3)
	assert.Equal(t, "full->sequestered", transitions[0])
	assert.Equal(t, "sequestered->full", transitions[1])
	assert.Equal(t, "recovery:sequestered->full", transitions[2])
}

func TestDegrader_LevelTransition_HooksFireCorrectly(t *testing.T) {
	now := time.Now()
	monitor := NewSecurityMonitor(withNowFunc(func() time.Time { return now }))
	policy := NewThresholdPolicy()

	var levelChanges []AutonomyLevel
	degrader := NewRuntimeDegrader(monitor, policy,
		WithHooks(Hooks{
			OnLevelChanged: func(_ context.Context, _, next AutonomyLevel) {
				levelChanges = append(levelChanges, next)
			},
		}),
	)

	ctx := context.Background()

	// Start at Full, push to Restricted.
	monitor.RecordEvent(ctx, SecurityEvent{
		Type: EventGuardBlocked, Severity: 0.35, Timestamp: now,
	})
	degrader.Evaluate(ctx)
	require.Len(t, levelChanges, 1)
	assert.Equal(t, Restricted, levelChanges[0])

	// Evaluate again at same severity - no additional hook call.
	degrader.Evaluate(ctx)
	assert.Len(t, levelChanges, 1)
}

func TestDegrader_ConcurrentEvaluate(t *testing.T) {
	monitor := NewSecurityMonitor()
	policy := NewThresholdPolicy()
	degrader := NewRuntimeDegrader(monitor, policy)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = degrader.Evaluate(context.Background())
			_ = degrader.CurrentLevel()
		}()
	}
	wg.Wait()
}

func TestDegrader_IDAndPersonaDelegation(t *testing.T) {
	monitor := NewSecurityMonitor()
	policy := NewThresholdPolicy()
	degrader := NewRuntimeDegrader(monitor, policy)

	inner := &mockAgent{id: "my-agent"}
	wrapped := agent.ApplyMiddleware(inner, degrader.Middleware())

	assert.Equal(t, "my-agent", wrapped.ID())
	assert.Equal(t, agent.Persona{}, wrapped.Persona())
	assert.Nil(t, wrapped.Children())
}
