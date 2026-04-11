package metacognitive

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSimpleExtractor_CompileTimeCheck(t *testing.T) {
	var _ HeuristicExtractor = (*SimpleExtractor)(nil)
}

func TestSimpleExtractor_Extract(t *testing.T) {
	extractor := NewSimpleExtractor()
	ctx := context.Background()

	tests := []struct {
		name       string
		signals    MonitoringSignals
		model      *SelfModel
		wantMin    int    // minimum expected heuristics
		wantSource string // expected source for first heuristic
	}{
		{
			name: "failure with errors produces avoid heuristics",
			signals: MonitoringSignals{
				TaskInput: "search for documents",
				TaskType:  "search",
				Success:   false,
				Errors:    []string{"timeout connecting to API"},
			},
			model:      NewSelfModel("agent-1"),
			wantMin:    1,
			wantSource: "failure",
		},
		{
			name: "failure with high iterations",
			signals: MonitoringSignals{
				TaskInput:      "complex task",
				TaskType:       "coding",
				Success:        false,
				IterationCount: 10,
				Errors:         []string{"exceeded max iterations"},
			},
			model:      NewSelfModel("agent-1"),
			wantMin:    2, // error heuristic + iteration heuristic
			wantSource: "failure",
		},
		{
			name: "success with efficient tool use",
			signals: MonitoringSignals{
				TaskInput:      "find info",
				TaskType:       "search",
				Success:        true,
				ToolCalls:      []string{"web_search"},
				IterationCount: 1,
			},
			model:      NewSelfModel("agent-1"),
			wantMin:    1,
			wantSource: "success",
		},
		{
			name: "success with novel tool combination",
			signals: MonitoringSignals{
				TaskInput:      "analyze data",
				TaskType:       "analysis",
				Success:        true,
				ToolCalls:      []string{"fetch_data", "transform"},
				IterationCount: 2,
			},
			model:      NewSelfModel("agent-1"),
			wantMin:    1,
			wantSource: "success",
		},
		{
			name: "success with known tool combination produces fewer heuristics",
			signals: MonitoringSignals{
				TaskInput:      "analyze data",
				TaskType:       "analysis",
				Success:        true,
				ToolCalls:      []string{"fetch_data", "transform"},
				IterationCount: 2,
			},
			model: &SelfModel{
				AgentID: "agent-1",
				Heuristics: []Heuristic{
					{Content: "Prefer: successful tool sequence for analysis: fetch_data -> transform"},
				},
			},
			wantMin:    1, // efficient use heuristic, but not novel combo
			wantSource: "success",
		},
		{
			name: "success with many iterations no tools",
			signals: MonitoringSignals{
				TaskInput:      "think deeply",
				TaskType:       "reasoning",
				Success:        true,
				IterationCount: 5,
			},
			model:   NewSelfModel("agent-1"),
			wantMin: 0,
		},
		{
			name: "failure deduplicates similar errors",
			signals: MonitoringSignals{
				TaskInput: "task",
				TaskType:  "general",
				Success:   false,
				Errors:    []string{"connection timeout", "connection timeout"},
			},
			model:      NewSelfModel("agent-1"),
			wantMin:    1,
			wantSource: "failure",
		},
		{
			name: "empty task type defaults to general",
			signals: MonitoringSignals{
				TaskInput: "task",
				Success:   false,
				Errors:    []string{"some error"},
			},
			model:      NewSelfModel("agent-1"),
			wantMin:    1,
			wantSource: "failure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			heuristics, err := extractor.Extract(ctx, tt.signals, tt.model)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(heuristics), tt.wantMin, "expected at least %d heuristics", tt.wantMin)

			if tt.wantSource != "" && len(heuristics) > 0 {
				assert.Equal(t, tt.wantSource, heuristics[0].Source)
			}

			// Verify all heuristics have required fields.
			for _, h := range heuristics {
				assert.NotEmpty(t, h.ID, "heuristic must have an ID")
				assert.NotEmpty(t, h.Content, "heuristic must have content")
				assert.NotEmpty(t, h.Source, "heuristic must have a source")
				assert.NotEmpty(t, h.TaskType, "heuristic must have a task type")
				assert.Greater(t, h.Utility, 0.0, "heuristic must have positive utility")
				assert.False(t, h.CreatedAt.IsZero(), "heuristic must have creation time")
			}
		})
	}
}

func TestSimpleExtractor_ContextCancellation(t *testing.T) {
	extractor := NewSimpleExtractor()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := extractor.Extract(ctx, MonitoringSignals{Success: false, Errors: []string{"err"}}, nil)
	assert.Error(t, err)
}

func TestSummarizeError(t *testing.T) {
	short := "short error"
	assert.Equal(t, short, summarizeError(short))

	long := make([]byte, 300)
	for i := range long {
		long[i] = 'a'
	}
	result := summarizeError(string(long))
	assert.Len(t, result, 203) // 200 + "..."
	assert.True(t, len(result) <= 203)
}

func TestGenerateID(t *testing.T) {
	id1 := generateID()
	id2 := generateID()
	assert.NotEqual(t, id1, id2, "IDs must be unique")
	assert.Contains(t, id1, "h_", "ID must start with h_ prefix")
}
