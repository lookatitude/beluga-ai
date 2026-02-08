package schema

import (
	"testing"
	"time"
)

func TestStreamChunk_Fields(t *testing.T) {
	tests := []struct {
		name           string
		chunk          StreamChunk
		wantDelta      string
		wantToolCalls  int
		wantFinish     string
		wantUsage      bool
		wantModelID    string
	}{
		{
			name: "text_delta",
			chunk: StreamChunk{
				Delta:   "Hello",
				ModelID: "gpt-4o",
			},
			wantDelta:     "Hello",
			wantToolCalls: 0,
			wantFinish:    "",
			wantUsage:     false,
			wantModelID:   "gpt-4o",
		},
		{
			name: "final_chunk_with_usage",
			chunk: StreamChunk{
				Delta:        "",
				FinishReason: "stop",
				Usage:        &Usage{InputTokens: 10, OutputTokens: 20, TotalTokens: 30},
				ModelID:      "claude-3-opus",
			},
			wantDelta:     "",
			wantToolCalls: 0,
			wantFinish:    "stop",
			wantUsage:     true,
			wantModelID:   "claude-3-opus",
		},
		{
			name: "tool_call_chunk",
			chunk: StreamChunk{
				ToolCalls: []ToolCall{
					{ID: "tc1", Name: "search", Arguments: `{"q":"test"}`},
				},
				FinishReason: "tool_calls",
			},
			wantDelta:     "",
			wantToolCalls: 1,
			wantFinish:    "tool_calls",
			wantUsage:     false,
			wantModelID:   "",
		},
		{
			name: "multiple_tool_calls",
			chunk: StreamChunk{
				ToolCalls: []ToolCall{
					{ID: "tc1", Name: "search", Arguments: `{"q":"a"}`},
					{ID: "tc2", Name: "calculate", Arguments: `{"x":1}`},
				},
				FinishReason: "tool_calls",
			},
			wantDelta:     "",
			wantToolCalls: 2,
			wantFinish:    "tool_calls",
			wantUsage:     false,
			wantModelID:   "",
		},
		{
			name: "length_finish",
			chunk: StreamChunk{
				Delta:        "truncated...",
				FinishReason: "length",
				Usage:        &Usage{InputTokens: 100, OutputTokens: 4096, TotalTokens: 4196},
			},
			wantDelta:     "truncated...",
			wantToolCalls: 0,
			wantFinish:    "length",
			wantUsage:     true,
			wantModelID:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.chunk.Delta != tt.wantDelta {
				t.Errorf("Delta = %q, want %q", tt.chunk.Delta, tt.wantDelta)
			}
			if len(tt.chunk.ToolCalls) != tt.wantToolCalls {
				t.Errorf("len(ToolCalls) = %d, want %d", len(tt.chunk.ToolCalls), tt.wantToolCalls)
			}
			if tt.chunk.FinishReason != tt.wantFinish {
				t.Errorf("FinishReason = %q, want %q", tt.chunk.FinishReason, tt.wantFinish)
			}
			hasUsage := tt.chunk.Usage != nil
			if hasUsage != tt.wantUsage {
				t.Errorf("has Usage = %v, want %v", hasUsage, tt.wantUsage)
			}
			if tt.chunk.ModelID != tt.wantModelID {
				t.Errorf("ModelID = %q, want %q", tt.chunk.ModelID, tt.wantModelID)
			}
		})
	}
}

func TestStreamChunk_ZeroValue(t *testing.T) {
	var chunk StreamChunk
	if chunk.Delta != "" {
		t.Errorf("zero Delta = %q, want empty", chunk.Delta)
	}
	if chunk.ToolCalls != nil {
		t.Errorf("zero ToolCalls = %v, want nil", chunk.ToolCalls)
	}
	if chunk.FinishReason != "" {
		t.Errorf("zero FinishReason = %q, want empty", chunk.FinishReason)
	}
	if chunk.Usage != nil {
		t.Errorf("zero Usage = %v, want nil", chunk.Usage)
	}
	if chunk.ModelID != "" {
		t.Errorf("zero ModelID = %q, want empty", chunk.ModelID)
	}
}

func TestStreamChunk_UsageAccess(t *testing.T) {
	chunk := StreamChunk{
		Usage: &Usage{
			InputTokens:  100,
			OutputTokens: 50,
			TotalTokens:  150,
			CachedTokens: 20,
		},
	}

	if chunk.Usage.InputTokens != 100 {
		t.Errorf("Usage.InputTokens = %d, want 100", chunk.Usage.InputTokens)
	}
	if chunk.Usage.OutputTokens != 50 {
		t.Errorf("Usage.OutputTokens = %d, want 50", chunk.Usage.OutputTokens)
	}
	if chunk.Usage.TotalTokens != 150 {
		t.Errorf("Usage.TotalTokens = %d, want 150", chunk.Usage.TotalTokens)
	}
	if chunk.Usage.CachedTokens != 20 {
		t.Errorf("Usage.CachedTokens = %d, want 20", chunk.Usage.CachedTokens)
	}
}

func TestStreamChunk_ToolCallDetails(t *testing.T) {
	chunk := StreamChunk{
		ToolCalls: []ToolCall{
			{ID: "call-abc", Name: "web_search", Arguments: `{"query":"go generics"}`},
		},
	}

	if len(chunk.ToolCalls) != 1 {
		t.Fatalf("len(ToolCalls) = %d, want 1", len(chunk.ToolCalls))
	}
	tc := chunk.ToolCalls[0]
	if tc.ID != "call-abc" {
		t.Errorf("ToolCalls[0].ID = %q, want %q", tc.ID, "call-abc")
	}
	if tc.Name != "web_search" {
		t.Errorf("ToolCalls[0].Name = %q, want %q", tc.Name, "web_search")
	}
	if tc.Arguments != `{"query":"go generics"}` {
		t.Errorf("ToolCalls[0].Arguments = %q, want %q", tc.Arguments, `{"query":"go generics"}`)
	}
}

func TestAgentEvent_Fields(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name         string
		event        AgentEvent
		wantType     string
		wantAgentID  string
		wantPayload  bool
	}{
		{
			name: "agent_start",
			event: AgentEvent{
				Type:      "agent_start",
				AgentID:   "agent-1",
				Payload:   map[string]any{"input": "hello"},
				Timestamp: now,
			},
			wantType:    "agent_start",
			wantAgentID: "agent-1",
			wantPayload: true,
		},
		{
			name: "tool_call",
			event: AgentEvent{
				Type:      "tool_call",
				AgentID:   "agent-2",
				Payload:   ToolCall{ID: "tc1", Name: "search"},
				Timestamp: now,
			},
			wantType:    "tool_call",
			wantAgentID: "agent-2",
			wantPayload: true,
		},
		{
			name: "thought",
			event: AgentEvent{
				Type:      "thought",
				AgentID:   "agent-3",
				Payload:   "I need to search for this information",
				Timestamp: now,
			},
			wantType:    "thought",
			wantAgentID: "agent-3",
			wantPayload: true,
		},
		{
			name: "handoff",
			event: AgentEvent{
				Type:      "handoff",
				AgentID:   "agent-1",
				Payload:   nil,
				Timestamp: now,
			},
			wantType:    "handoff",
			wantAgentID: "agent-1",
			wantPayload: false,
		},
		{
			name: "no_payload",
			event: AgentEvent{
				Type:      "agent_end",
				AgentID:   "agent-1",
				Timestamp: now,
			},
			wantType:    "agent_end",
			wantAgentID: "agent-1",
			wantPayload: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.event.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", tt.event.Type, tt.wantType)
			}
			if tt.event.AgentID != tt.wantAgentID {
				t.Errorf("AgentID = %q, want %q", tt.event.AgentID, tt.wantAgentID)
			}
			hasPayload := tt.event.Payload != nil
			if hasPayload != tt.wantPayload {
				t.Errorf("has Payload = %v, want %v", hasPayload, tt.wantPayload)
			}
			if tt.event.Timestamp.IsZero() {
				t.Error("Timestamp is zero, want non-zero")
			}
		})
	}
}

func TestAgentEvent_ZeroValue(t *testing.T) {
	var event AgentEvent
	if event.Type != "" {
		t.Errorf("zero Type = %q, want empty", event.Type)
	}
	if event.AgentID != "" {
		t.Errorf("zero AgentID = %q, want empty", event.AgentID)
	}
	if event.Payload != nil {
		t.Errorf("zero Payload = %v, want nil", event.Payload)
	}
	if !event.Timestamp.IsZero() {
		t.Errorf("zero Timestamp = %v, want zero", event.Timestamp)
	}
}

func TestAgentEvent_Timestamp(t *testing.T) {
	before := time.Now()
	event := AgentEvent{
		Type:      "test",
		AgentID:   "agent-1",
		Timestamp: time.Now(),
	}
	after := time.Now()

	if event.Timestamp.Before(before) {
		t.Error("Timestamp is before creation time")
	}
	if event.Timestamp.After(after) {
		t.Error("Timestamp is after check time")
	}
}

func TestAgentEvent_PayloadTypes(t *testing.T) {
	t.Run("string_payload", func(t *testing.T) {
		event := AgentEvent{Type: "thought", Payload: "thinking..."}
		if s, ok := event.Payload.(string); !ok || s != "thinking..." {
			t.Errorf("Payload = %v, want string %q", event.Payload, "thinking...")
		}
	})

	t.Run("map_payload", func(t *testing.T) {
		payload := map[string]any{"key": "value", "count": 42}
		event := AgentEvent{Type: "custom", Payload: payload}
		m, ok := event.Payload.(map[string]any)
		if !ok {
			t.Fatal("Payload is not map[string]any")
		}
		if m["key"] != "value" {
			t.Errorf("Payload[\"key\"] = %v, want %q", m["key"], "value")
		}
	})

	t.Run("toolcall_payload", func(t *testing.T) {
		tc := ToolCall{ID: "tc1", Name: "search", Arguments: `{"q":"test"}`}
		event := AgentEvent{Type: "tool_call", Payload: tc}
		got, ok := event.Payload.(ToolCall)
		if !ok {
			t.Fatal("Payload is not ToolCall")
		}
		if got.Name != "search" {
			t.Errorf("Payload.Name = %q, want %q", got.Name, "search")
		}
	})
}
