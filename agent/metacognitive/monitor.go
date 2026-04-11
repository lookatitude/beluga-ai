package metacognitive

import (
	"context"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/tool"
)

// Monitor collects execution signals from agent hooks. It attaches to an
// agent's AfterAct, OnEnd, OnError, and OnToolCall hooks to build a
// MonitoringSignals snapshot for each turn.
type Monitor struct {
	mu      sync.Mutex
	signals MonitoringSignals
	start   time.Time
}

// NewMonitor creates a new Monitor ready to collect signals.
func NewMonitor() *Monitor {
	return &Monitor{
		start: time.Now(),
	}
}

// Hooks returns agent.Hooks that the monitor uses to collect signals.
// Attach these to an agent via agent.WithHooks.
func (m *Monitor) Hooks() agent.Hooks {
	return agent.Hooks{
		OnStart: func(_ context.Context, input string) error {
			m.mu.Lock()
			defer m.mu.Unlock()
			m.start = time.Now()
			m.signals.TaskInput = input
			m.signals.Success = true // optimistic; cleared on error
			return nil
		},
		AfterAct: func(_ context.Context, _ agent.Action, obs agent.Observation) error {
			m.mu.Lock()
			defer m.mu.Unlock()
			m.signals.IterationCount++
			if obs.Error != nil {
				m.signals.Errors = append(m.signals.Errors, obs.Error.Error())
			}
			return nil
		},
		OnEnd: func(_ context.Context, result string, err error) {
			m.mu.Lock()
			defer m.mu.Unlock()
			m.signals.Outcome = result
			m.signals.TotalLatency = time.Since(m.start)
			if err != nil {
				m.signals.Success = false
				m.signals.Errors = append(m.signals.Errors, err.Error())
			}
		},
		OnError: func(_ context.Context, err error) error {
			m.mu.Lock()
			defer m.mu.Unlock()
			m.signals.Success = false
			if err != nil {
				m.signals.Errors = append(m.signals.Errors, err.Error())
			}
			return err
		},
		OnToolCall: func(_ context.Context, call agent.ToolCallInfo) error {
			m.mu.Lock()
			defer m.mu.Unlock()
			m.signals.ToolCalls = append(m.signals.ToolCalls, call.Name)
			return nil
		},
		OnToolResult: func(_ context.Context, call agent.ToolCallInfo, result *tool.Result) error {
			m.mu.Lock()
			defer m.mu.Unlock()
			if result != nil && result.IsError {
				m.signals.Errors = append(m.signals.Errors, "tool "+call.Name+" failed")
			}
			return nil
		},
	}
}

// Signals returns a copy of the currently accumulated monitoring signals.
func (m *Monitor) Signals() MonitoringSignals {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := m.signals
	// Copy slices to prevent aliasing.
	if len(m.signals.ToolCalls) > 0 {
		s.ToolCalls = make([]string, len(m.signals.ToolCalls))
		copy(s.ToolCalls, m.signals.ToolCalls)
	}
	if len(m.signals.Errors) > 0 {
		s.Errors = make([]string, len(m.signals.Errors))
		copy(s.Errors, m.signals.Errors)
	}
	return s
}

// Reset clears all accumulated signals and restarts the timer.
func (m *Monitor) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.signals = MonitoringSignals{}
	m.start = time.Now()
}
