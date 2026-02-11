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

func TestHybridPipelineRunS2SWithSession(t *testing.T) {
	// S2S with session set should run the S2S processor successfully.
	session := NewSession("s2s-test")
	hp := NewHybridPipeline(
		WithS2S(passThroughProcessor),
		WithHybridSession(session),
	)

	err := hp.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if hp.CurrentMode() != ModeS2S {
		t.Errorf("CurrentMode() = %q, want %q", hp.CurrentMode(), ModeS2S)
	}
}

func TestHybridPipelineRunS2SNoSession(t *testing.T) {
	// S2S without session should return "requires a session" error.
	hp := NewHybridPipeline(
		WithS2S(passThroughProcessor),
		// No session
	)

	err := hp.Run(context.Background())
	if err == nil {
		t.Fatal("Run() should return error for S2S without session")
	}
	if err.Error() != "voice: S2S pipeline requires a session" {
		t.Errorf("Run() error = %q, want session required error", err)
	}
}

func TestHybridPipelineRunCascadeNotConfigured(t *testing.T) {
	// Force cascade mode but no cascade configured → error.
	hp := NewHybridPipeline(
		WithS2S(passThroughProcessor),
		// Force switch to cascade immediately.
		WithSwitchPolicy(SwitchPolicyFunc(func(_ context.Context, _ PipelineState) bool {
			return true
		})),
	)

	err := hp.Run(context.Background())
	if err == nil {
		t.Fatal("Run() should return error when cascade not configured")
	}
	if err.Error() != "voice: cascade pipeline not configured" {
		t.Errorf("Run() error = %q, want cascade not configured error", err)
	}
}

func TestHybridPipelineUnknownMode(t *testing.T) {
	// Force an unknown pipeline mode to hit the default case.
	hp := NewHybridPipeline(
		WithS2S(passThroughProcessor),
	)
	// Manually set an invalid mode.
	hp.state.CurrentMode = PipelineMode("invalid_mode")

	err := hp.Run(context.Background())
	if err == nil {
		t.Fatal("Run() should return error for unknown mode")
	}
	expected := `voice: unknown pipeline mode "invalid_mode"`
	if err.Error() != expected {
		t.Errorf("Run() error = %q, want %q", err, expected)
	}
}

func TestHybridPipelineFallbackToCascadeNotConfigured(t *testing.T) {
	// S2S is nil, default mode is S2S → falls to runCascade → cascade is nil → error.
	hp := NewHybridPipeline(
		// No S2S, no cascade.
		WithSwitchPolicy(nil), // Disable switch policy.
	)

	err := hp.Run(context.Background())
	if err == nil {
		t.Fatal("Run() should return error")
	}
	// With nil policy and nil S2S, it hits the first check and returns
	// "requires at least one of S2S or cascade".
}

func TestHybridPipelineCascadeModeDirectly(t *testing.T) {
	// Start in cascade mode explicitly by switching policy.
	transport := &mockTransport{
		frames: []Frame{NewTextFrame("cascade-direct")},
	}
	cascade := NewPipeline(
		WithTransport(transport),
		WithSTT(passThroughProcessor),
	)

	hp := NewHybridPipeline(
		WithS2S(passThroughProcessor),
		WithCascade(cascade),
	)
	// Force the mode to cascade.
	hp.state.CurrentMode = ModeCascade

	err := hp.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if hp.CurrentMode() != ModeCascade {
		t.Errorf("CurrentMode() = %q, want %q", hp.CurrentMode(), ModeCascade)
	}
	if len(transport.sent) != 1 {
		t.Fatalf("sent %d frames, want 1", len(transport.sent))
	}
}
