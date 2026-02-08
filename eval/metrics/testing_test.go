package metrics_test

import (
	"context"
	"iter"

	"github.com/lookatitude/beluga-ai/internal/testutil/mockllm"
	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
)

// mockChatModel wraps MockChatModel to satisfy the ChatModel interface.
// It adapts mockllm.MockChatModel's signatures to match llm.ChatModel.
type mockChatModel struct {
	*mockllm.MockChatModel
}

func (m *mockChatModel) Generate(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
	// Mock ignores options, just call underlying Generate with empty opts
	return m.MockChatModel.Generate(ctx, msgs)
}

func (m *mockChatModel) Stream(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) iter.Seq2[schema.StreamChunk, error] {
	// Mock ignores options, just call underlying Stream with empty opts
	return m.MockChatModel.Stream(ctx, msgs)
}

func (m *mockChatModel) BindTools(tools []schema.ToolDefinition) llm.ChatModel {
	return &mockChatModel{m.MockChatModel.BindTools(tools)}
}

func newMockChatModel(opts ...mockllm.Option) *mockChatModel {
	return &mockChatModel{mockllm.New(opts...)}
}
