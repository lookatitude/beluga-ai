package memory

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	bm "github.com/lookatitude/beluga-ai/v2/memory"
	"github.com/lookatitude/beluga-ai/v2/memory/stores/inmemory"
	"github.com/lookatitude/beluga-ai/v2/schema"
)

// docMemory is a Memory whose Search returns fixed documents.
type docMemory struct {
	mockMemory
	docs []schema.Document
}

func (m *docMemory) Search(_ context.Context, _ string, _ int) ([]schema.Document, error) {
	return m.docs, nil
}

// FI-08: input turns must survive a Save/Load round trip.
func TestSignedMemory_RoundTripKeepsInputAndOutput(t *testing.T) {
	s, err := NewSignedMemoryMiddleware([]byte("k"))
	require.NoError(t, err)
	m := s.Wrap(bm.NewRecall(inmemory.NewMessageStore()))
	ctx := context.Background()

	require.NoError(t, m.Save(ctx, schema.NewHumanMessage("hello"), schema.NewAIMessage("hi there")))
	msgs, err := m.Load(ctx, "")
	require.NoError(t, err)
	require.Len(t, msgs, 2)
	assert.Equal(t, schema.RoleHuman, msgs[0].GetRole())
	assert.Equal(t, schema.RoleAI, msgs[1].GetRole())
}

// FI-08: the MAC must bind the role and tool calls.
func TestSignedMemory_RejectsRoleAndToolCallTampering(t *testing.T) {
	s, err := NewSignedMemoryMiddleware([]byte("k"))
	require.NoError(t, err)

	tool, err := s.SignMessage(schema.NewToolMessage("call-1", "result text"))
	require.NoError(t, err)
	ai, err := s.SignMessage(&schema.AIMessage{
		Parts:     []schema.ContentPart{schema.TextPart{Text: "calling"}},
		ToolCalls: []schema.ToolCall{{ID: "c1", Name: "search", Arguments: `{"q":"x"}`}},
	})
	require.NoError(t, err)

	relabelled := &schema.SystemMessage{Parts: tool.GetContent(), Metadata: tool.GetMetadata()}
	aiTampered := &schema.AIMessage{
		Parts:     ai.GetContent(),
		ToolCalls: []schema.ToolCall{{ID: "c1", Name: "delete_all", Arguments: `{}`}},
		Metadata:  ai.GetMetadata(),
	}
	toolIDTampered := &schema.ToolMessage{ToolCallID: "call-2", Parts: tool.GetContent(), Metadata: tool.GetMetadata()}

	var reasons []string
	sm, err := NewSignedMemoryMiddleware([]byte("k"), WithSignedHooks(Hooks{
		OnSignatureInvalid: func(_ context.Context, r string) { reasons = append(reasons, r) },
	}))
	require.NoError(t, err)
	msgs, err := sm.Wrap(&mockMemory{loaded: []schema.Message{tool, ai, relabelled, aiTampered, toolIDTampered}}).Load(context.Background(), "")
	require.NoError(t, err)
	assert.Len(t, msgs, 2)
	assert.Equal(t, []string{"signature mismatch", "signature mismatch", "signature mismatch"}, reasons)
}

func TestSignedMemory_LegacySignatures(t *testing.T) {
	legacy := &schema.AIMessage{
		Parts: []schema.ContentPart{schema.TextPart{Text: "old"}},
	}
	s, err := NewSignedMemoryMiddleware([]byte("k"))
	require.NoError(t, err)
	legacy.Metadata = map[string]any{MetaKeySignature: s.Sign("old")}
	unknown := &schema.AIMessage{
		Parts:    []schema.ContentPart{schema.TextPart{Text: "old"}},
		Metadata: map[string]any{MetaKeySignature: s.Sign("old"), MetaKeySignatureVersion: "99"},
	}
	badLegacy := &schema.AIMessage{
		Parts:    []schema.ContentPart{schema.TextPart{Text: "new"}},
		Metadata: map[string]any{MetaKeySignature: s.Sign("old")},
	}
	loaded := []schema.Message{legacy, unknown, badLegacy, nil}

	var reasons []string
	strict, err := NewSignedMemoryMiddleware([]byte("k"), WithSignedHooks(Hooks{
		OnSignatureInvalid: func(_ context.Context, r string) { reasons = append(reasons, r) },
	}))
	require.NoError(t, err)
	msgs, err := strict.Wrap(&mockMemory{loaded: loaded}).Load(context.Background(), "")
	require.NoError(t, err)
	assert.Empty(t, msgs)
	assert.Equal(t, []string{"legacy signature rejected", "unknown signature version", "legacy signature rejected", "missing signature"}, reasons)

	lenient, err := NewSignedMemoryMiddleware([]byte("k"), WithLegacySignatures())
	require.NoError(t, err)
	msgs, err = lenient.Wrap(&mockMemory{loaded: loaded}).Load(context.Background(), "")
	require.NoError(t, err)
	assert.Len(t, msgs, 1)
}

// FI-08: Search must verify documents.
func TestSignedMemory_SearchVerifiesDocuments(t *testing.T) {
	s, err := NewSignedMemoryMiddleware([]byte("k"))
	require.NoError(t, err)
	good := s.SignDocument(schema.Document{ID: "d1", Content: "fact", Metadata: map[string]any{"role": "ai"}})
	tampered := good
	tampered.Content = "forged fact"
	relabelled := s.SignDocument(schema.Document{ID: "d2", Content: "x", Metadata: map[string]any{"role": "tool"}})
	relabelled.Metadata["role"] = "system"
	unsigned := schema.Document{ID: "d3", Content: "plain"}

	var reasons []string
	sm, err := NewSignedMemoryMiddleware([]byte("k"), WithSignedHooks(Hooks{
		OnSignatureInvalid: func(_ context.Context, r string) { reasons = append(reasons, r) },
	}))
	require.NoError(t, err)
	inner := &docMemory{docs: []schema.Document{good, tampered, relabelled, unsigned}}
	docs, err := sm.Wrap(inner).Search(context.Background(), "q", 5)
	require.NoError(t, err)
	require.Len(t, docs, 1)
	assert.Equal(t, "d1", docs[0].ID)
	assert.Equal(t, []string{"signature mismatch", "signature mismatch", "missing signature"}, reasons)

	open, err := NewSignedMemoryMiddleware([]byte("k"), WithUnverifiedSearch())
	require.NoError(t, err)
	docs, err = open.MessageMiddleware()(inner).Search(context.Background(), "q", 5)
	require.NoError(t, err)
	assert.Len(t, docs, 4)
}

func TestSignedMemory_SaveSignsNilInputSafely(t *testing.T) {
	s, err := NewSignedMemoryMiddleware([]byte("k"))
	require.NoError(t, err)
	mock := &mockMemory{}
	require.NoError(t, s.Wrap(mock).Save(context.Background(), nil, schema.NewAIMessage("x")))
	require.Len(t, mock.saved, 1)
	assert.Nil(t, mock.saved[0].input)
}
