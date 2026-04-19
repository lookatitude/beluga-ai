package trajectory

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/v2/agent"
	"github.com/lookatitude/beluga-ai/v2/schema"
	"github.com/lookatitude/beluga-ai/v2/tool"
)

// RecorderOption configures a Recorder.
type RecorderOption func(*Recorder)

// WithAgentID sets the agent ID on recorded trajectories.
func WithAgentID(id string) RecorderOption {
	return func(r *Recorder) {
		r.agentID = id
	}
}

// WithTrajectoryID sets the trajectory ID on recorded trajectories.
func WithTrajectoryID(id string) RecorderOption {
	return func(r *Recorder) {
		r.trajectoryID = id
	}
}

// Recorder captures agent execution steps into a Trajectory via agent.Hooks.
// It is thread-safe and composable with user hooks via agent.ComposeHooks.
type Recorder struct {
	mu           sync.Mutex
	agentID      string
	trajectoryID string
	input        string
	output       string
	steps        []Step
	startTime    time.Time
	endTime      time.Time
	lastStepTime time.Time
}

// NewRecorder creates a new Recorder with the given options.
func NewRecorder(opts ...RecorderOption) *Recorder {
	r := &Recorder{}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Hooks returns agent.Hooks that capture execution steps into the trajectory.
// The returned hooks can be composed with user hooks via agent.ComposeHooks.
func (r *Recorder) Hooks() agent.Hooks {
	return agent.Hooks{
		OnStart: func(_ context.Context, input string) error {
			r.mu.Lock()
			defer r.mu.Unlock()
			r.input = input
			r.startTime = time.Now()
			r.lastStepTime = r.startTime
			return nil
		},
		OnEnd: func(_ context.Context, result string, _ error) {
			r.mu.Lock()
			defer r.mu.Unlock()
			r.output = result
			now := time.Now()
			r.endTime = now
			r.addStepLocked(StepFinish, StepAction{Message: result}, StepResult{Output: result}, now)
		},
		AfterPlan: func(_ context.Context, actions []agent.Action) error {
			r.mu.Lock()
			defer r.mu.Unlock()
			now := time.Now()
			msg := formatActions(actions)
			r.addStepLocked(StepPlan, StepAction{Message: msg}, StepResult{}, now)
			return nil
		},
		OnToolCall: func(_ context.Context, call agent.ToolCallInfo) error {
			r.mu.Lock()
			defer r.mu.Unlock()
			now := time.Now()
			r.addStepLocked(StepToolCall,
				StepAction{ToolName: call.Name, ToolArgs: call.Arguments},
				StepResult{},
				now,
			)
			return nil
		},
		OnToolResult: func(_ context.Context, call agent.ToolCallInfo, result *tool.Result) error {
			r.mu.Lock()
			defer r.mu.Unlock()
			// Update the last tool_call step with the result.
			for i := len(r.steps) - 1; i >= 0; i-- {
				if r.steps[i].Type == StepToolCall && r.steps[i].Action.ToolName == call.Name && r.steps[i].Result.Output == "" {
					if result != nil {
						r.steps[i].Result.Output = contentPartsToString(result.Content)
						if result.IsError {
							r.steps[i].Result.Error = contentPartsToString(result.Content)
						}
					}
					r.steps[i].Latency = time.Since(r.steps[i].Timestamp)
					break
				}
			}
			return nil
		},
		OnHandoff: func(_ context.Context, from, to string) error {
			r.mu.Lock()
			defer r.mu.Unlock()
			now := time.Now()
			r.addStepLocked(StepHandoff,
				StepAction{Target: to, Message: "handoff from " + from},
				StepResult{},
				now,
			)
			return nil
		},
	}
}

// Trajectory returns a copy of the recorded trajectory. Returns nil if no
// execution has been recorded (OnStart was never called).
func (r *Recorder) Trajectory() *Trajectory {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.startTime.IsZero() {
		return nil
	}

	steps := make([]Step, len(r.steps))
	copy(steps, r.steps)

	// TotalLatency is measured from OnStart to OnEnd. If OnEnd has not yet
	// fired (still running), fall back to time.Since(startTime) so callers
	// still see a monotonically increasing latency while the agent runs.
	var totalLatency time.Duration
	if !r.endTime.IsZero() {
		totalLatency = r.endTime.Sub(r.startTime)
	} else {
		totalLatency = time.Since(r.startTime)
	}

	return &Trajectory{
		ID:           r.trajectoryID,
		AgentID:      r.agentID,
		Input:        r.input,
		Output:       r.output,
		Steps:        steps,
		TotalLatency: totalLatency,
		Timestamp:    r.startTime,
	}
}

// Reset clears all recorded data so the recorder can be reused.
func (r *Recorder) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.input = ""
	r.output = ""
	r.steps = nil
	r.startTime = time.Time{}
	r.endTime = time.Time{}
	r.lastStepTime = time.Time{}
}

// addStepLocked adds a step while holding the mutex.
func (r *Recorder) addStepLocked(typ StepType, action StepAction, result StepResult, now time.Time) {
	latency := now.Sub(r.lastStepTime)
	r.steps = append(r.steps, Step{
		Index:     len(r.steps),
		Type:      typ,
		Action:    action,
		Result:    result,
		Latency:   latency,
		Timestamp: now,
	})
	r.lastStepTime = now
}

// contentPartsToString extracts text from content parts.
func contentPartsToString(parts []schema.ContentPart) string {
	var b strings.Builder
	for _, p := range parts {
		if tp, ok := p.(schema.TextPart); ok {
			b.WriteString(tp.Text)
		}
	}
	return b.String()
}

// formatActions summarizes planner actions into a readable string.
func formatActions(actions []agent.Action) string {
	if len(actions) == 0 {
		return "no actions planned"
	}
	var result string
	for i, a := range actions {
		if i > 0 {
			result += "; "
		}
		switch a.Type {
		case agent.ActionTool:
			if a.ToolCall != nil {
				result += "tool:" + a.ToolCall.Name
			} else {
				result += "tool:(unknown)"
			}
		case agent.ActionRespond:
			result += "respond"
		case agent.ActionFinish:
			result += "finish"
		case agent.ActionHandoff:
			result += "handoff"
		default:
			result += string(a.Type)
		}
	}
	return result
}
