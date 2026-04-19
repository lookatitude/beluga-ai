package plancache

import (
	"testing"

	"github.com/lookatitude/beluga-ai/v2/agent"
	"github.com/lookatitude/beluga-ai/v2/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractTemplate(t *testing.T) {
	tests := []struct {
		name        string
		agentID     string
		input       string
		actions     []agent.Action
		wantActions int
		wantTool    string
	}{
		{
			name:    "single tool action",
			agentID: "agent1",
			input:   "search database",
			actions: []agent.Action{
				{
					Type:     agent.ActionTool,
					ToolCall: &schema.ToolCall{Name: "search", Arguments: `{"query":"test"}`},
				},
			},
			wantActions: 1,
			wantTool:    "search",
		},
		{
			name:    "multiple actions",
			agentID: "agent1",
			input:   "search and respond",
			actions: []agent.Action{
				{Type: agent.ActionTool, ToolCall: &schema.ToolCall{Name: "search"}},
				{Type: agent.ActionTool, ToolCall: &schema.ToolCall{Name: "format"}},
				{Type: agent.ActionFinish, Message: "done"},
			},
			wantActions: 3,
			wantTool:    "search",
		},
		{
			name:    "respond action preserves description",
			agentID: "agent1",
			input:   "greet user",
			actions: []agent.Action{
				{Type: agent.ActionRespond, Message: "Hello!"},
			},
			wantActions: 1,
		},
		{
			name:        "empty actions",
			agentID:     "agent1",
			input:       "test",
			actions:     nil,
			wantActions: 0,
		},
		{
			name:    "handoff action",
			agentID: "agent1",
			input:   "delegate task",
			actions: []agent.Action{
				{Type: agent.ActionHandoff, Message: "transferring", Metadata: map[string]any{"target": "agent2"}},
			},
			wantActions: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl := ExtractTemplate(tt.agentID, tt.input, tt.actions, nil)
			require.NotNil(t, tmpl)
			assert.Equal(t, tt.agentID, tmpl.AgentID)
			assert.Equal(t, tt.input, tmpl.Input)
			assert.Len(t, tmpl.Actions, tt.wantActions)
			assert.NotEmpty(t, tmpl.ID)

			if tt.wantTool != "" && len(tmpl.Actions) > 0 {
				assert.Equal(t, tt.wantTool, tmpl.Actions[0].ToolName)
				// Arguments should NOT be preserved.
				assert.Empty(t, tmpl.Actions[0].Description)
			}
		})
	}
}

func TestExtractTemplate_DeterministicID(t *testing.T) {
	actions := []agent.Action{
		{Type: agent.ActionTool, ToolCall: &schema.ToolCall{Name: "search"}},
		{Type: agent.ActionFinish, Message: "done"},
	}

	tmpl1 := ExtractTemplate("a1", "input1", actions, nil)
	tmpl2 := ExtractTemplate("a1", "input2", actions, nil)

	// Same agent + same action sequence = same ID.
	assert.Equal(t, tmpl1.ID, tmpl2.ID)

	// Different agent = different ID.
	tmpl3 := ExtractTemplate("a2", "input1", actions, nil)
	assert.NotEqual(t, tmpl1.ID, tmpl3.ID)
}

func TestExtractTemplate_CustomExtractor(t *testing.T) {
	custom := func(input string) []string {
		return []string{"custom", "keywords"}
	}

	tmpl := ExtractTemplate("a1", "test", []agent.Action{
		{Type: agent.ActionFinish, Message: "ok"},
	}, custom)

	assert.Equal(t, []string{"custom", "keywords"}, tmpl.Keywords)
}

func TestTruncateDescription(t *testing.T) {
	assert.Equal(t, "short", truncateDescription("short", 100))
	long := "this is a very long description that should be truncated because it exceeds the maximum allowed length for descriptions"
	result := truncateDescription(long, 50)
	assert.Len(t, result, 50)
	assert.True(t, len(result) <= 50)
	assert.Contains(t, result, "...")
}
