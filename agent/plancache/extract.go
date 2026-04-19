package plancache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/lookatitude/beluga-ai/v2/agent"
)

// ExtractTemplate converts a slice of agent actions into a Template. Tool
// argument values are discarded to allow reuse across similar inputs. The
// template ID is derived deterministically from the action sequence.
func ExtractTemplate(agentID, input string, actions []agent.Action, keywordExtractor func(string) []string) *Template {
	if keywordExtractor == nil {
		keywordExtractor = ExtractKeywords
	}

	tmplActions := make([]TemplateAction, len(actions))
	for i, a := range actions {
		ta := TemplateAction{
			Type: a.Type,
		}
		if a.ToolCall != nil {
			ta.ToolName = a.ToolCall.Name
			ta.Arguments = a.ToolCall.Arguments
		}
		if a.Type == agent.ActionRespond || a.Type == agent.ActionFinish {
			ta.Description = truncateDescription(a.Message, 100)
		}
		tmplActions[i] = ta
	}

	return &Template{
		ID:       templateID(agentID, tmplActions),
		Input:    input,
		Keywords: keywordExtractor(input),
		Actions:  tmplActions,
		AgentID:  agentID,
	}
}

// templateID generates a deterministic ID from the agent ID and action
// sequence. The same sequence of action types and tool names always produces
// the same ID.
func templateID(agentID string, actions []TemplateAction) string {
	h := sha256.New()
	h.Write([]byte(agentID))
	h.Write([]byte("|"))
	for _, a := range actions {
		fmt.Fprintf(h, "%s:%s|", a.Type, a.ToolName)
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// truncateDescription truncates a string to maxLen characters, appending
// "..." if truncated.
func truncateDescription(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
