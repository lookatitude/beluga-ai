package agentic

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/guard"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgenticPipeline_EmptyAllows(t *testing.T) {
	p := NewAgenticPipeline()
	result, err := p.Validate(context.Background(), guard.GuardInput{
		Content:  "anything",
		Metadata: map[string]any{},
	})
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Empty(t, result.Assessments)
}

func TestAgenticPipeline_AllGuardsPass(t *testing.T) {
	p := NewAgenticPipeline(
		WithToolMisuseGuard(WithAllowedTools("search")),
		WithEscalationGuard(),
		WithCascadeGuard(),
	)

	result, err := p.Validate(context.Background(), guard.GuardInput{
		Content: `{"query": "hello"}`,
		Metadata: map[string]any{
			"tool_name": "search",
		},
	})
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Len(t, result.Assessments, 3)

	for _, a := range result.Assessments {
		assert.False(t, a.Blocked)
	}
}

func TestAgenticPipeline_FirstBlockStops(t *testing.T) {
	p := NewAgenticPipeline(
		WithToolMisuseGuard(WithAllowedTools("search")),
		WithExfiltrationGuard(),
		WithCascadeGuard(),
	)

	// Tool "exec" is not allowed -- should block at first guard.
	result, err := p.Validate(context.Background(), guard.GuardInput{
		Content: `{"cmd": "rm -rf /"}`,
		Metadata: map[string]any{
			"tool_name": "exec",
		},
	})
	require.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Equal(t, "tool_misuse_guard", result.GuardName)
	// Only one assessment since pipeline stops at first block.
	assert.Len(t, result.Assessments, 1)
	assert.Equal(t, RiskToolMisuse, result.Assessments[0].Risk)
	assert.True(t, result.Assessments[0].Blocked)
	assert.Equal(t, "high", result.Assessments[0].Severity)
}

func TestAgenticPipeline_AssessmentRisks(t *testing.T) {
	p := NewAgenticPipeline(
		WithToolMisuseGuard(),
		WithEscalationGuard(),
		WithExfiltrationGuard(WithoutDefaultPII(), WithScanURLEncoding(false)),
		WithCascadeGuard(),
	)

	result, err := p.Validate(context.Background(), guard.GuardInput{
		Content:  "clean content",
		Metadata: map[string]any{},
	})
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Len(t, result.Assessments, 4)

	expectedRisks := []AgenticRisk{
		RiskToolMisuse,
		RiskPrivilegeEscalation,
		RiskDataExfiltration,
		RiskCascadingFailure,
	}
	for i, a := range result.Assessments {
		assert.Equal(t, expectedRisks[i], a.Risk)
		assert.False(t, a.Blocked)
	}
}

func TestAgenticPipeline_WithCustomGuard(t *testing.T) {
	custom := &mockGuard{
		name:    "custom_guard",
		allowed: false,
		reason:  "custom block",
	}

	p := NewAgenticPipeline(
		WithGuard(custom),
	)

	result, err := p.Validate(context.Background(), guard.GuardInput{
		Content:  "test",
		Metadata: map[string]any{},
	})
	require.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Equal(t, "custom block", result.Reason)
}

func TestAgenticPipeline_Guards(t *testing.T) {
	p := NewAgenticPipeline(
		WithToolMisuseGuard(),
		WithCascadeGuard(),
	)
	guards := p.Guards()
	assert.Len(t, guards, 2)
	assert.Equal(t, "tool_misuse_guard", guards[0].Name())
	assert.Equal(t, "cascade_guard", guards[1].Name())
}

func TestAgenticPipeline_ContextCancellation(t *testing.T) {
	p := NewAgenticPipeline(
		WithToolMisuseGuard(),
		WithCascadeGuard(),
	)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := p.Validate(ctx, guard.GuardInput{
		Content:  "test",
		Metadata: map[string]any{},
	})
	assert.ErrorIs(t, err, context.Canceled)
}

// mockGuard is a test helper that returns configurable results.
type mockGuard struct {
	name    string
	allowed bool
	reason  string
}

func (m *mockGuard) Name() string { return m.name }

func (m *mockGuard) Validate(_ context.Context, _ guard.GuardInput) (guard.GuardResult, error) {
	return guard.GuardResult{
		Allowed:   m.allowed,
		Reason:    m.reason,
		GuardName: m.name,
	}, nil
}
