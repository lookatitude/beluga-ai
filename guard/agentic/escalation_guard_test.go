package agentic

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/v2/guard"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrivilegeEscalationGuard_Name(t *testing.T) {
	g := NewPrivilegeEscalationGuard()
	assert.Equal(t, "privilege_escalation_guard", g.Name())
}

func TestPrivilegeEscalationGuard_Validate(t *testing.T) {
	tests := []struct {
		name    string
		opts    []EscalationOption
		input   guard.GuardInput
		allowed bool
		reason  string
	}{
		{
			name:    "no handoff metadata allows",
			input:   guard.GuardInput{Content: "hello", Metadata: map[string]any{}},
			allowed: true,
		},
		{
			name: "simple handoff within depth allows",
			opts: []EscalationOption{WithMaxHandoffDepth(3)},
			input: guard.GuardInput{
				Content: "",
				Metadata: map[string]any{
					"source_agent":  "agent_a",
					"target_agent":  "agent_b",
					"handoff_chain": []string{"agent_a"},
				},
			},
			allowed: true,
		},
		{
			name: "handoff exceeding depth blocks",
			opts: []EscalationOption{WithMaxHandoffDepth(2)},
			input: guard.GuardInput{
				Content: "",
				Metadata: map[string]any{
					"source_agent":  "agent_b",
					"target_agent":  "agent_c",
					"handoff_chain": []string{"agent_a", "agent_b"},
				},
			},
			allowed: false,
			reason:  "handoff chain depth 3 exceeds maximum of 2",
		},
		{
			name: "handoff to blocked agent",
			opts: []EscalationOption{WithBlockedAgents("evil_agent")},
			input: guard.GuardInput{
				Content: "",
				Metadata: map[string]any{
					"source_agent": "agent_a",
					"target_agent": "evil_agent",
				},
			},
			allowed: false,
			reason:  `handoff to blocked agent "evil_agent" is not allowed`,
		},
		{
			name: "permission subset violation blocks",
			opts: []EscalationOption{
				WithAgentPermissions("agent_a", "read:files"),
				WithAgentPermissions("agent_b", "read:files", "write:files"),
			},
			input: guard.GuardInput{
				Content: "",
				Metadata: map[string]any{
					"source_agent": "agent_a",
					"target_agent": "agent_b",
				},
			},
			allowed: false,
			reason:  "has permissions not held by source",
		},
		{
			name: "permission subset satisfied allows",
			opts: []EscalationOption{
				WithAgentPermissions("agent_a", "read:files", "write:files"),
				WithAgentPermissions("agent_b", "read:files"),
			},
			input: guard.GuardInput{
				Content: "",
				Metadata: map[string]any{
					"source_agent": "agent_a",
					"target_agent": "agent_b",
				},
			},
			allowed: true,
		},
		{
			name: "permission subset disabled allows escalation",
			opts: []EscalationOption{
				WithPermissionSubsetEnforcement(false),
				WithAgentPermissions("agent_a", "read:files"),
				WithAgentPermissions("agent_b", "read:files", "write:files", "execute:code"),
			},
			input: guard.GuardInput{
				Content: "",
				Metadata: map[string]any{
					"source_agent": "agent_a",
					"target_agent": "agent_b",
				},
			},
			allowed: true,
		},
		{
			name: "unknown agent permissions allows by default",
			opts: []EscalationOption{
				WithAgentPermissions("agent_a", "read:files"),
			},
			input: guard.GuardInput{
				Content: "",
				Metadata: map[string]any{
					"source_agent": "agent_a",
					"target_agent": "agent_unknown",
				},
			},
			allowed: true,
		},
		{
			name: "handoff chain as []any works",
			opts: []EscalationOption{WithMaxHandoffDepth(2)},
			input: guard.GuardInput{
				Content: "",
				Metadata: map[string]any{
					"source_agent":  "b",
					"target_agent":  "c",
					"handoff_chain": []any{"a", "b"},
				},
			},
			allowed: false,
			reason:  "handoff chain depth 3 exceeds maximum of 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewPrivilegeEscalationGuard(tt.opts...)
			result, err := g.Validate(context.Background(), tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.allowed, result.Allowed)
			if tt.reason != "" {
				assert.Contains(t, result.Reason, tt.reason)
			}
		})
	}
}

func TestPrivilegeEscalationGuard_ContextCancellation(t *testing.T) {
	g := NewPrivilegeEscalationGuard()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := g.Validate(ctx, guard.GuardInput{
		Metadata: map[string]any{
			"source_agent": "a",
			"target_agent": "b",
		},
	})
	assert.ErrorIs(t, err, context.Canceled)
}

func TestPrivilegeEscalationGuard_CompileTimeCheck(t *testing.T) {
	var _ guard.Guard = (*PrivilegeEscalationGuard)(nil)
}
