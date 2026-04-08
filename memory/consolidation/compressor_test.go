package consolidation

import (
	"context"
	"errors"
	"iter"
	"testing"

	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockChatModel is a minimal llm.ChatModel for testing the compressor.
type mockChatModel struct {
	response string
	err      error
}

var _ llm.ChatModel = (*mockChatModel)(nil)

func (m *mockChatModel) Generate(_ context.Context, _ []schema.Message, _ ...llm.GenerateOption) (*schema.AIMessage, error) {
	if m.err != nil {
		return nil, m.err
	}
	return schema.NewAIMessage(m.response), nil
}

func (m *mockChatModel) Stream(_ context.Context, _ []schema.Message, _ ...llm.GenerateOption) iter.Seq2[schema.StreamChunk, error] {
	return func(_ func(schema.StreamChunk, error) bool) {}
}

func (m *mockChatModel) BindTools(_ []schema.ToolDefinition) llm.ChatModel {
	return m
}

func (m *mockChatModel) ModelID() string { return "mock" }

func TestSummaryCompressor_Compress(t *testing.T) {
	tests := []struct {
		name     string
		model    *mockChatModel
		records  []Record
		wantErr  bool
		wantText string
	}{
		{
			name:  "successful compression",
			model: &mockChatModel{response: "summary of content"},
			records: []Record{
				{ID: "1", Content: "a very long memory about things"},
			},
			wantText: "summary of content",
		},
		{
			name:  "empty LLM response falls back to original",
			model: &mockChatModel{response: ""},
			records: []Record{
				{ID: "1", Content: "original content"},
			},
			wantText: "original content",
		},
		{
			name:    "LLM error propagates",
			model:   &mockChatModel{err: errors.New("llm failure")},
			records: []Record{{ID: "1", Content: "content"}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewSummaryCompressor(tt.model)
			got, err := c.Compress(context.Background(), tt.records)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, got, len(tt.records))
			assert.Equal(t, tt.wantText, got[0].Content)
			assert.Equal(t, tt.records[0].ID, got[0].ID)
		})
	}
}

func TestSummaryCompressor_CompressedMetadata(t *testing.T) {
	c := NewSummaryCompressor(&mockChatModel{response: "short"})
	records := []Record{{ID: "1", Content: "long"}}

	got, err := c.Compress(context.Background(), records)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, true, got[0].Metadata["compressed"])
}

func TestSummaryCompressor_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	c := NewSummaryCompressor(&mockChatModel{response: "x"})
	_, err := c.Compress(ctx, []Record{{ID: "1", Content: "text"}})
	assert.ErrorIs(t, err, context.Canceled)
}
