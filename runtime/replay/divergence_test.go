package replay

import (
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/schema"
	"github.com/stretchr/testify/assert"
)

func TestDivergenceDetector_Detect(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name      string
		original  []schema.AgentEvent
		replayed  []schema.AgentEvent
		wantCount int
		wantTypes []DivergenceType
	}{
		{
			name:      "identical sequences",
			original:  []schema.AgentEvent{{Type: "start", AgentID: "a1", Timestamp: now}},
			replayed:  []schema.AgentEvent{{Type: "start", AgentID: "a1", Timestamp: now}},
			wantCount: 0,
		},
		{
			name:      "both empty",
			original:  nil,
			replayed:  nil,
			wantCount: 0,
		},
		{
			name: "type mismatch",
			original: []schema.AgentEvent{
				{Type: "start", AgentID: "a1", Timestamp: now},
			},
			replayed: []schema.AgentEvent{
				{Type: "stop", AgentID: "a1", Timestamp: now},
			},
			wantCount: 1,
			wantTypes: []DivergenceType{DivergenceTypeMismatch},
		},
		{
			name: "agent mismatch",
			original: []schema.AgentEvent{
				{Type: "start", AgentID: "a1", Timestamp: now},
			},
			replayed: []schema.AgentEvent{
				{Type: "start", AgentID: "a2", Timestamp: now},
			},
			wantCount: 1,
			wantTypes: []DivergenceType{DivergenceAgentMismatch},
		},
		{
			name: "extra in replay",
			original: []schema.AgentEvent{
				{Type: "start", AgentID: "a1", Timestamp: now},
			},
			replayed: []schema.AgentEvent{
				{Type: "start", AgentID: "a1", Timestamp: now},
				{Type: "extra", AgentID: "a1", Timestamp: now},
			},
			wantCount: 1,
			wantTypes: []DivergenceType{DivergenceExtra},
		},
		{
			name: "missing from replay",
			original: []schema.AgentEvent{
				{Type: "start", AgentID: "a1", Timestamp: now},
				{Type: "end", AgentID: "a1", Timestamp: now},
			},
			replayed: []schema.AgentEvent{
				{Type: "start", AgentID: "a1", Timestamp: now},
			},
			wantCount: 1,
			wantTypes: []DivergenceType{DivergenceMissing},
		},
		{
			name: "multiple divergences",
			original: []schema.AgentEvent{
				{Type: "start", AgentID: "a1", Timestamp: now},
				{Type: "tool_call", AgentID: "a1", Timestamp: now},
				{Type: "end", AgentID: "a1", Timestamp: now},
			},
			replayed: []schema.AgentEvent{
				{Type: "start", AgentID: "a2", Timestamp: now},
				{Type: "thought", AgentID: "a1", Timestamp: now},
			},
			wantCount: 3,
			wantTypes: []DivergenceType{DivergenceAgentMismatch, DivergenceTypeMismatch, DivergenceMissing},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDivergenceDetector()
			divergences := d.Detect(tt.original, tt.replayed)

			assert.Len(t, divergences, tt.wantCount)
			if tt.wantTypes != nil {
				for i, dt := range tt.wantTypes {
					assert.Equal(t, dt, divergences[i].Type, "divergence %d type", i)
				}
			}
		})
	}
}

func TestIntToStr(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{0, "0"},
		{1, "1"},
		{42, "42"},
		{-5, "-5"},
		{100, "100"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, intToStr(tt.input))
	}
}
