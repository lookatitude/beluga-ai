package guard

import (
	"context"
	"fmt"
	"testing"
)

// allowGuard is a test guard that always allows content.
type allowGuard struct {
	name string
}

func (g *allowGuard) Name() string { return g.name }
func (g *allowGuard) Validate(_ context.Context, _ GuardInput) (GuardResult, error) {
	return GuardResult{Allowed: true}, nil
}

// blockGuard is a test guard that always blocks content.
type blockGuard struct {
	name   string
	reason string
}

func (g *blockGuard) Name() string { return g.name }
func (g *blockGuard) Validate(_ context.Context, _ GuardInput) (GuardResult, error) {
	return GuardResult{Allowed: false, Reason: g.reason, GuardName: g.name}, nil
}

// modifyGuard replaces content.
type modifyGuard struct {
	name    string
	replace string
}

func (g *modifyGuard) Name() string { return g.name }
func (g *modifyGuard) Validate(_ context.Context, input GuardInput) (GuardResult, error) {
	return GuardResult{Allowed: true, Modified: g.replace}, nil
}

// errorGuard returns an error.
type errorGuard struct {
	name string
	err  error
}

func (g *errorGuard) Name() string { return g.name }
func (g *errorGuard) Validate(_ context.Context, _ GuardInput) (GuardResult, error) {
	return GuardResult{}, g.err
}

// recordingGuard records the content it receives.
type recordingGuard struct {
	name     string
	received string
}

func (g *recordingGuard) Name() string { return g.name }
func (g *recordingGuard) Validate(_ context.Context, input GuardInput) (GuardResult, error) {
	g.received = input.Content
	return GuardResult{Allowed: true}, nil
}

func TestNewPipeline_Empty(t *testing.T) {
	p := NewPipeline()
	if p == nil {
		t.Fatal("NewPipeline() returned nil")
	}
}

func TestPipeline_ValidateInput_AllAllow(t *testing.T) {
	p := NewPipeline(
		Input(&allowGuard{name: "g1"}, &allowGuard{name: "g2"}),
	)

	result, err := p.ValidateInput(context.Background(), "hello")
	if err != nil {
		t.Fatalf("ValidateInput() error = %v", err)
	}
	if !result.Allowed {
		t.Error("result.Allowed = false, want true")
	}
}

func TestPipeline_ValidateInput_FirstBlocks(t *testing.T) {
	p := NewPipeline(
		Input(
			&blockGuard{name: "blocker", reason: "unsafe content"},
			&allowGuard{name: "should_not_run"},
		),
	)

	result, err := p.ValidateInput(context.Background(), "bad content")
	if err != nil {
		t.Fatalf("ValidateInput() error = %v", err)
	}
	if result.Allowed {
		t.Error("result.Allowed = true, want false")
	}
	if result.GuardName != "blocker" {
		t.Errorf("GuardName = %q, want %q", result.GuardName, "blocker")
	}
	if result.Reason != "unsafe content" {
		t.Errorf("Reason = %q, want %q", result.Reason, "unsafe content")
	}
}

func TestPipeline_ValidateInput_SecondBlocks(t *testing.T) {
	p := NewPipeline(
		Input(
			&allowGuard{name: "g1"},
			&blockGuard{name: "g2", reason: "blocked by g2"},
		),
	)

	result, err := p.ValidateInput(context.Background(), "content")
	if err != nil {
		t.Fatalf("ValidateInput() error = %v", err)
	}
	if result.Allowed {
		t.Error("result.Allowed = true, want false")
	}
	if result.GuardName != "g2" {
		t.Errorf("GuardName = %q, want %q", result.GuardName, "g2")
	}
}

func TestPipeline_ValidateInput_ModifiesContent(t *testing.T) {
	recorder := &recordingGuard{name: "recorder"}
	p := NewPipeline(
		Input(
			&modifyGuard{name: "modifier", replace: "sanitized content"},
			recorder,
		),
	)

	result, err := p.ValidateInput(context.Background(), "original content")
	if err != nil {
		t.Fatalf("ValidateInput() error = %v", err)
	}
	if !result.Allowed {
		t.Error("result.Allowed = false, want true")
	}
	if result.Modified != "sanitized content" {
		t.Errorf("Modified = %q, want %q", result.Modified, "sanitized content")
	}
	// Second guard should see modified content.
	if recorder.received != "sanitized content" {
		t.Errorf("recorder received = %q, want %q", recorder.received, "sanitized content")
	}
}

func TestPipeline_ValidateInput_NoModification(t *testing.T) {
	p := NewPipeline(
		Input(&allowGuard{name: "g1"}),
	)

	result, err := p.ValidateInput(context.Background(), "unchanged")
	if err != nil {
		t.Fatalf("ValidateInput() error = %v", err)
	}
	if result.Modified != "" {
		t.Errorf("Modified = %q, want empty (no modification)", result.Modified)
	}
}

func TestPipeline_ValidateInput_GuardError(t *testing.T) {
	p := NewPipeline(
		Input(&errorGuard{name: "broken", err: fmt.Errorf("guard failed")}),
	)

	_, err := p.ValidateInput(context.Background(), "content")
	if err == nil {
		t.Fatal("ValidateInput() expected error, got nil")
	}
	if err.Error() != "guard failed" {
		t.Errorf("error = %q, want %q", err.Error(), "guard failed")
	}
}

func TestPipeline_ValidateInput_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	p := NewPipeline(
		Input(&allowGuard{name: "g1"}),
	)

	_, err := p.ValidateInput(ctx, "content")
	if err == nil {
		t.Fatal("ValidateInput() expected context error")
	}
}

func TestPipeline_ValidateInput_EmptyGuards(t *testing.T) {
	p := NewPipeline()

	result, err := p.ValidateInput(context.Background(), "content")
	if err != nil {
		t.Fatalf("ValidateInput() error = %v", err)
	}
	if !result.Allowed {
		t.Error("result.Allowed = false, want true (no guards)")
	}
}

func TestPipeline_ValidateOutput(t *testing.T) {
	p := NewPipeline(
		Output(&blockGuard{name: "output_blocker", reason: "toxic output"}),
	)

	result, err := p.ValidateOutput(context.Background(), "toxic response")
	if err != nil {
		t.Fatalf("ValidateOutput() error = %v", err)
	}
	if result.Allowed {
		t.Error("result.Allowed = true, want false")
	}
	if result.Reason != "toxic output" {
		t.Errorf("Reason = %q, want %q", result.Reason, "toxic output")
	}
}

func TestPipeline_ValidateOutput_Allowed(t *testing.T) {
	p := NewPipeline(
		Output(&allowGuard{name: "output_allow"}),
	)

	result, err := p.ValidateOutput(context.Background(), "safe response")
	if err != nil {
		t.Fatalf("ValidateOutput() error = %v", err)
	}
	if !result.Allowed {
		t.Error("result.Allowed = false, want true")
	}
}

func TestPipeline_ValidateTool(t *testing.T) {
	// A recording guard to verify tool_name metadata.
	var receivedMeta map[string]any
	metaGuard := &metaRecordingGuard{
		name:    "meta_recorder",
		capture: &receivedMeta,
	}

	p := NewPipeline(
		Tool(metaGuard),
	)

	result, err := p.ValidateTool(context.Background(), "web_search", `{"query":"test"}`)
	if err != nil {
		t.Fatalf("ValidateTool() error = %v", err)
	}
	if !result.Allowed {
		t.Error("result.Allowed = false, want true")
	}
	if receivedMeta["tool_name"] != "web_search" {
		t.Errorf("tool_name = %v, want %q", receivedMeta["tool_name"], "web_search")
	}
}

func TestPipeline_ValidateTool_Blocked(t *testing.T) {
	p := NewPipeline(
		Tool(&blockGuard{name: "tool_blocker", reason: "dangerous tool args"}),
	)

	result, err := p.ValidateTool(context.Background(), "shell", "rm -rf /")
	if err != nil {
		t.Fatalf("ValidateTool() error = %v", err)
	}
	if result.Allowed {
		t.Error("result.Allowed = true, want false")
	}
}

func TestPipeline_ThreeStages_Independent(t *testing.T) {
	p := NewPipeline(
		Input(&blockGuard{name: "input_block", reason: "input blocked"}),
		Output(&allowGuard{name: "output_allow"}),
		Tool(&allowGuard{name: "tool_allow"}),
	)

	// Input should be blocked.
	result, err := p.ValidateInput(context.Background(), "content")
	if err != nil {
		t.Fatalf("ValidateInput() error = %v", err)
	}
	if result.Allowed {
		t.Error("input should be blocked")
	}

	// Output should be allowed.
	result, err = p.ValidateOutput(context.Background(), "content")
	if err != nil {
		t.Fatalf("ValidateOutput() error = %v", err)
	}
	if !result.Allowed {
		t.Error("output should be allowed")
	}

	// Tool should be allowed.
	result, err = p.ValidateTool(context.Background(), "tool", "args")
	if err != nil {
		t.Fatalf("ValidateTool() error = %v", err)
	}
	if !result.Allowed {
		t.Error("tool should be allowed")
	}
}

// metaRecordingGuard records metadata from GuardInput.
type metaRecordingGuard struct {
	name    string
	capture *map[string]any
}

func (g *metaRecordingGuard) Name() string { return g.name }
func (g *metaRecordingGuard) Validate(_ context.Context, input GuardInput) (GuardResult, error) {
	*g.capture = input.Metadata
	return GuardResult{Allowed: true}, nil
}
