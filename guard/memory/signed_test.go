package memory

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockMemory is a simple in-memory implementation for testing.
type mockMemory struct {
	saved   []savedPair
	loaded  []schema.Message
	cleared bool
}

type savedPair struct {
	input, output schema.Message
}

func (m *mockMemory) Save(_ context.Context, input, output schema.Message) error {
	m.saved = append(m.saved, savedPair{input: input, output: output})
	return nil
}

func (m *mockMemory) Load(_ context.Context, _ string) ([]schema.Message, error) {
	return m.loaded, nil
}

func (m *mockMemory) Search(_ context.Context, _ string, _ int) ([]schema.Document, error) {
	return nil, nil
}

func (m *mockMemory) Clear(_ context.Context) error {
	m.cleared = true
	return nil
}

func TestSignedMemoryMiddleware_NewRequiresKey(t *testing.T) {
	_, err := NewSignedMemoryMiddleware(nil)
	require.Error(t, err)

	_, err = NewSignedMemoryMiddleware([]byte{})
	require.Error(t, err)

	s, err := NewSignedMemoryMiddleware([]byte("secret"))
	require.NoError(t, err)
	require.NotNil(t, s)
}

func TestSignedMemoryMiddleware_SignAndVerify(t *testing.T) {
	s, err := NewSignedMemoryMiddleware([]byte("test-key"))
	require.NoError(t, err)

	sig := s.Sign("hello world")
	assert.NotEmpty(t, sig)
	assert.True(t, s.Verify("hello world", sig))
	assert.False(t, s.Verify("tampered", sig))
	assert.False(t, s.Verify("hello world", "invalid"))
}

func TestSignedMemory_SaveAddsSignature(t *testing.T) {
	mock := &mockMemory{}
	s, err := NewSignedMemoryMiddleware([]byte("key123"))
	require.NoError(t, err)

	signed := s.Wrap(mock)
	input := schema.NewHumanMessage("hello")
	output := &schema.AIMessage{
		Parts:    []schema.ContentPart{schema.TextPart{Text: "world"}},
		Metadata: map[string]any{"existing": "value"},
	}

	err = signed.Save(context.Background(), input, output)
	require.NoError(t, err)
	require.Len(t, mock.saved, 1)

	// Check that signature was added to metadata.
	savedOutput := mock.saved[0].output
	meta := savedOutput.GetMetadata()
	sig, ok := meta[MetaKeySignature].(string)
	assert.True(t, ok)
	assert.NotEmpty(t, sig)

	// Verify the signature is valid.
	assert.True(t, s.Verify("world", sig))

	// Verify existing metadata preserved.
	assert.Equal(t, "value", meta["existing"])
}

func TestSignedMemory_LoadVerifiesSignature(t *testing.T) {
	s, err := NewSignedMemoryMiddleware([]byte("key123"))
	require.NoError(t, err)

	sig := s.Sign("valid response")

	tests := []struct {
		name       string
		messages   []schema.Message
		wantCount  int
		hookCalled bool
	}{
		{
			name: "valid signature passes",
			messages: []schema.Message{
				&schema.AIMessage{
					Parts:    []schema.ContentPart{schema.TextPart{Text: "valid response"}},
					Metadata: map[string]any{MetaKeySignature: sig},
				},
			},
			wantCount: 1,
		},
		{
			name: "missing signature filtered",
			messages: []schema.Message{
				&schema.AIMessage{
					Parts:    []schema.ContentPart{schema.TextPart{Text: "no sig"}},
					Metadata: map[string]any{},
				},
			},
			wantCount:  0,
			hookCalled: true,
		},
		{
			name: "tampered content filtered",
			messages: []schema.Message{
				&schema.AIMessage{
					Parts:    []schema.ContentPart{schema.TextPart{Text: "tampered content"}},
					Metadata: map[string]any{MetaKeySignature: sig},
				},
			},
			wantCount:  0,
			hookCalled: true,
		},
		{
			name: "mixed valid and invalid",
			messages: []schema.Message{
				&schema.AIMessage{
					Parts:    []schema.ContentPart{schema.TextPart{Text: "valid response"}},
					Metadata: map[string]any{MetaKeySignature: sig},
				},
				&schema.AIMessage{
					Parts:    []schema.ContentPart{schema.TextPart{Text: "tampered"}},
					Metadata: map[string]any{MetaKeySignature: sig},
				},
			},
			wantCount:  1,
			hookCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var hookCalled bool
			opts := []SignedOption{}
			if tt.hookCalled {
				opts = append(opts, WithSignedHooks(Hooks{
					OnSignatureInvalid: func(_ context.Context, _ string) {
						hookCalled = true
					},
				}))
			}

			sm, err := NewSignedMemoryMiddleware([]byte("key123"), opts...)
			require.NoError(t, err)

			mock := &mockMemory{loaded: tt.messages}
			signed := sm.Wrap(mock)

			msgs, err := signed.Load(context.Background(), "query")
			require.NoError(t, err)
			assert.Len(t, msgs, tt.wantCount)

			if tt.hookCalled {
				assert.True(t, hookCalled)
			}
		})
	}
}

func TestSignedMemory_MessageMiddleware(t *testing.T) {
	s, err := NewSignedMemoryMiddleware([]byte("key"))
	require.NoError(t, err)

	mw := s.MessageMiddleware()
	require.NotNil(t, mw)

	mock := &mockMemory{}
	wrapped := mw(mock)
	require.NotNil(t, wrapped)
}

func TestSignedMemory_DelegatesClearAndSearch(t *testing.T) {
	mock := &mockMemory{}
	s, err := NewSignedMemoryMiddleware([]byte("key"))
	require.NoError(t, err)

	signed := s.Wrap(mock)

	err = signed.Clear(context.Background())
	require.NoError(t, err)
	assert.True(t, mock.cleared)

	docs, err := signed.Search(context.Background(), "query", 5)
	require.NoError(t, err)
	assert.Nil(t, docs)
}

func TestWithSignature_AllMessageTypes(t *testing.T) {
	sig := "test-sig"

	tests := []struct {
		name string
		msg  schema.Message
	}{
		{"human", &schema.HumanMessage{Parts: []schema.ContentPart{schema.TextPart{Text: "hi"}}}},
		{"ai", &schema.AIMessage{Parts: []schema.ContentPart{schema.TextPart{Text: "hi"}}, ModelID: "gpt-4"}},
		{"system", &schema.SystemMessage{Parts: []schema.ContentPart{schema.TextPart{Text: "hi"}}}},
		{"tool", &schema.ToolMessage{Parts: []schema.ContentPart{schema.TextPart{Text: "hi"}}, ToolCallID: "tc1"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := withSignature(tt.msg, sig)
			require.NoError(t, err)
			meta := result.GetMetadata()
			assert.Equal(t, sig, meta[MetaKeySignature])
		})
	}
}

func TestComputeAndVerifyHMAC(t *testing.T) {
	key := []byte("secret-key")

	sig := computeHMAC(key, "test data")
	assert.NotEmpty(t, sig)
	assert.True(t, verifyHMAC(key, "test data", sig))
	assert.False(t, verifyHMAC(key, "different data", sig))
	assert.False(t, verifyHMAC([]byte("wrong-key"), "test data", sig))
}
