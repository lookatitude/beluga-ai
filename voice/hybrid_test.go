package voice

import (
	"context"
	"testing"
)

func TestDefaultSwitchPolicy(t *testing.T) {
	tests := []struct {
		name       string
		toolCalls  int
		mode       PipelineMode
		threshold  int
		wantSwitch bool
	}{
		{"below threshold", 1, ModeS2S, 3, false},
		{"at threshold", 3, ModeS2S, 3, true},
		{"above threshold", 5, ModeS2S, 3, true},
		{"cascade mode", 5, ModeCascade, 3, false},
		{"zero threshold defaults to 3", 3, ModeS2S, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy := &DefaultSwitchPolicy{ToolCallThreshold: tt.threshold}
			state := PipelineState{
				ToolCallCount: tt.toolCalls,
				CurrentMode:   tt.mode,
			}
			got := policy.ShouldSwitch(context.Background(), state)
			if got != tt.wantSwitch {
				t.Errorf("ShouldSwitch() = %v, want %v", got, tt.wantSwitch)
			}
		})
	}
}

func TestOnToolOverload(t *testing.T) {
	state := PipelineState{
		ToolCallCount: 5,
		CurrentMode:   ModeS2S,
	}
	if !OnToolOverload.ShouldSwitch(context.Background(), state) {
		t.Error("OnToolOverload.ShouldSwitch() = false, want true")
	}
}

func TestSwitchPolicyFunc(t *testing.T) {
	f := SwitchPolicyFunc(func(_ context.Context, state PipelineState) bool {
		return state.TurnCount > 10
	})

	if f.ShouldSwitch(context.Background(), PipelineState{TurnCount: 5}) {
		t.Error("ShouldSwitch() = true, want false")
	}
	if !f.ShouldSwitch(context.Background(), PipelineState{TurnCount: 15}) {
		t.Error("ShouldSwitch() = false, want true")
	}
}

func TestNewHybridPipeline(t *testing.T) {
	hp := NewHybridPipeline()
	if hp == nil {
		t.Fatal("NewHybridPipeline() returned nil")
	}
	if hp.CurrentMode() != ModeS2S {
		t.Errorf("CurrentMode() = %q, want %q", hp.CurrentMode(), ModeS2S)
	}
}

func TestHybridPipelineNoComponents(t *testing.T) {
	hp := NewHybridPipeline()
	err := hp.Run(context.Background())
	if err == nil {
		t.Error("expected error for no components")
	}
}

func TestHybridPipelineUpdateState(t *testing.T) {
	hp := NewHybridPipeline()
	hp.UpdateState(5, 10)

	if hp.state.ToolCallCount != 5 {
		t.Errorf("ToolCallCount = %d, want 5", hp.state.ToolCallCount)
	}
	if hp.state.TurnCount != 10 {
		t.Errorf("TurnCount = %d, want 10", hp.state.TurnCount)
	}
}

func TestHybridPipelineSwitchToCascade(t *testing.T) {
	transport := &mockTransport{
		frames: []Frame{NewTextFrame("hello")},
	}
	cascade := NewPipeline(
		WithTransport(transport),
		WithSTT(passThroughProcessor),
	)

	hp := NewHybridPipeline(
		WithCascade(cascade),
		WithSwitchPolicy(&DefaultSwitchPolicy{ToolCallThreshold: 2}),
	)

	// Update state to trigger switch.
	hp.UpdateState(5, 3)

	err := hp.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if hp.CurrentMode() != ModeCascade {
		t.Errorf("CurrentMode() = %q, want %q", hp.CurrentMode(), ModeCascade)
	}
}

func TestHybridPipelineS2SNotImplemented(t *testing.T) {
	// S2S mode with an actual S2S processor returns "not yet implemented".
	hp := NewHybridPipeline(
		WithS2S(passThroughProcessor),
	)

	err := hp.Run(context.Background())
	if err == nil {
		t.Error("expected error for S2S not yet implemented")
	}
}

func TestHybridPipelineFallbackToCascade(t *testing.T) {
	// When S2S is nil and mode is S2S, should fall back to cascade.
	transport := &mockTransport{
		frames: []Frame{NewTextFrame("fallback")},
	}
	cascade := NewPipeline(
		WithTransport(transport),
		WithSTT(passThroughProcessor),
	)

	hp := NewHybridPipeline(
		WithCascade(cascade),
		// No S2S configured, but default mode is S2S.
	)

	err := hp.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if hp.CurrentMode() != ModeCascade {
		t.Errorf("CurrentMode() = %q, want %q", hp.CurrentMode(), ModeCascade)
	}
}

func TestHybridPipelineOptions(t *testing.T) {
	session := NewSession("test")
	hp := NewHybridPipeline(
		WithHybridSession(session),
	)
	if hp.config.Session != session {
		t.Error("Session not set")
	}
}
