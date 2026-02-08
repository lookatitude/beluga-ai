package s2s

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockSession is a test implementation of the Session interface.
type mockSession struct {
	recvChan   chan SessionEvent
	audioSent  [][]byte
	textSent   []string
	resultSent []schema.ToolResult
	interrupted bool
	closed     bool
}

var _ Session = (*mockSession)(nil)

func newMockSession() *mockSession {
	return &mockSession{
		recvChan: make(chan SessionEvent, 10),
	}
}

func (m *mockSession) SendAudio(ctx context.Context, audio []byte) error {
	m.audioSent = append(m.audioSent, audio)
	return nil
}

func (m *mockSession) SendText(ctx context.Context, text string) error {
	m.textSent = append(m.textSent, text)
	return nil
}

func (m *mockSession) SendToolResult(ctx context.Context, result schema.ToolResult) error {
	m.resultSent = append(m.resultSent, result)
	return nil
}

func (m *mockSession) Recv() <-chan SessionEvent {
	return m.recvChan
}

func (m *mockSession) Interrupt(ctx context.Context) error {
	m.interrupted = true
	return nil
}

func (m *mockSession) Close() error {
	if !m.closed {
		close(m.recvChan)
		m.closed = true
	}
	return nil
}

// mockS2S is a test implementation of the S2S interface.
type mockS2S struct {
	startFunc func(context.Context, ...Option) (Session, error)
	session   *mockSession
}

var _ S2S = (*mockS2S)(nil)

func (m *mockS2S) Start(ctx context.Context, opts ...Option) (Session, error) {
	if m.startFunc != nil {
		return m.startFunc(ctx, opts...)
	}
	if m.session == nil {
		m.session = newMockSession()
	}
	return m.session, nil
}

func TestRegistry_RegisterAndNew(t *testing.T) {
	// Register a mock provider.
	Register("mock-s2s", func(cfg Config) (S2S, error) {
		return &mockS2S{}, nil
	})

	// Create an S2S engine using the registered provider.
	engine, err := New("mock-s2s", Config{Voice: "alloy"})
	require.NoError(t, err)
	require.NotNil(t, engine)

	// Start a session.
	session, err := engine.Start(context.Background())
	require.NoError(t, err)
	require.NotNil(t, session)
	defer session.Close()
}

func TestRegistry_UnknownProvider(t *testing.T) {
	_, err := New("nonexistent-s2s-provider", Config{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown provider")
}

func TestList(t *testing.T) {
	// Register a test provider.
	Register("test-s2s-list", func(cfg Config) (S2S, error) {
		return &mockS2S{}, nil
	})

	names := List()
	require.NotEmpty(t, names)

	// Verify the list is sorted and contains our provider.
	assert.Contains(t, names, "test-s2s-list")
	for i := 1; i < len(names); i++ {
		assert.LessOrEqual(t, names[i-1], names[i], "list should be sorted")
	}
}

func TestApplyOptions(t *testing.T) {
	tools := []schema.ToolDefinition{
		{Name: "search", Description: "Search the web"},
	}

	cfg := ApplyOptions(
		WithVoice("alloy"),
		WithModel("gpt-4o-realtime"),
		WithInstructions("You are a helpful assistant"),
		WithTools(tools),
		WithSampleRate(24000),
	)

	assert.Equal(t, "alloy", cfg.Voice)
	assert.Equal(t, "gpt-4o-realtime", cfg.Model)
	assert.Equal(t, "You are a helpful assistant", cfg.Instructions)
	assert.Len(t, cfg.Tools, 1)
	assert.Equal(t, "search", cfg.Tools[0].Name)
	assert.Equal(t, 24000, cfg.SampleRate)
}

func TestSessionEventTypes(t *testing.T) {
	tests := []struct {
		name      string
		eventType SessionEventType
		expected  string
	}{
		{"AudioOutput", EventAudioOutput, "audio_output"},
		{"TextOutput", EventTextOutput, "text_output"},
		{"Transcript", EventTranscript, "transcript"},
		{"ToolCall", EventToolCall, "tool_call"},
		{"TurnEnd", EventTurnEnd, "turn_end"},
		{"Error", EventError, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.eventType))
		})
	}
}

func TestMockSession_SendAudio(t *testing.T) {
	session := newMockSession()
	defer session.Close()

	audio1 := []byte{0x01, 0x02}
	audio2 := []byte{0x03, 0x04}

	err := session.SendAudio(context.Background(), audio1)
	require.NoError(t, err)

	err = session.SendAudio(context.Background(), audio2)
	require.NoError(t, err)

	assert.Len(t, session.audioSent, 2)
	assert.Equal(t, audio1, session.audioSent[0])
	assert.Equal(t, audio2, session.audioSent[1])
}

func TestMockSession_SendText(t *testing.T) {
	session := newMockSession()
	defer session.Close()

	err := session.SendText(context.Background(), "hello")
	require.NoError(t, err)

	err = session.SendText(context.Background(), "world")
	require.NoError(t, err)

	assert.Len(t, session.textSent, 2)
	assert.Equal(t, "hello", session.textSent[0])
	assert.Equal(t, "world", session.textSent[1])
}

func TestMockSession_SendToolResult(t *testing.T) {
	session := newMockSession()
	defer session.Close()

	result := schema.ToolResult{
		CallID:  "call-123",
		Content: []schema.ContentPart{schema.TextPart{Text: "result"}},
		IsError: false,
	}

	err := session.SendToolResult(context.Background(), result)
	require.NoError(t, err)

	assert.Len(t, session.resultSent, 1)
	assert.Equal(t, "call-123", session.resultSent[0].CallID)
}

func TestMockSession_Recv(t *testing.T) {
	session := newMockSession()

	// Send some events.
	session.recvChan <- SessionEvent{Type: EventTextOutput, Text: "hello"}
	session.recvChan <- SessionEvent{Type: EventAudioOutput, Audio: []byte{0x01}}
	close(session.recvChan)

	var events []SessionEvent
	for event := range session.Recv() {
		events = append(events, event)
	}

	require.Len(t, events, 2)
	assert.Equal(t, EventTextOutput, events[0].Type)
	assert.Equal(t, "hello", events[0].Text)
	assert.Equal(t, EventAudioOutput, events[1].Type)
	assert.Equal(t, []byte{0x01}, events[1].Audio)
}

func TestMockSession_Interrupt(t *testing.T) {
	session := newMockSession()
	defer session.Close()

	assert.False(t, session.interrupted)

	err := session.Interrupt(context.Background())
	require.NoError(t, err)
	assert.True(t, session.interrupted)
}

func TestMockSession_Close(t *testing.T) {
	session := newMockSession()

	assert.False(t, session.closed)

	err := session.Close()
	require.NoError(t, err)
	assert.True(t, session.closed)

	// Closing again should be idempotent.
	err = session.Close()
	require.NoError(t, err)
}

func TestComposeHooks(t *testing.T) {
	var callOrder []string

	hooks1 := Hooks{
		OnTurn: func(ctx context.Context, userText, agentText string) {
			callOrder = append(callOrder, "hooks1-turn:"+userText+":"+agentText)
		},
		OnInterrupt: func(ctx context.Context) {
			callOrder = append(callOrder, "hooks1-interrupt")
		},
	}

	hooks2 := Hooks{
		OnTurn: func(ctx context.Context, userText, agentText string) {
			callOrder = append(callOrder, "hooks2-turn:"+userText+":"+agentText)
		},
		OnToolCall: func(ctx context.Context, call schema.ToolCall) {
			callOrder = append(callOrder, "hooks2-tool:"+call.Name)
		},
	}

	composed := ComposeHooks(hooks1, hooks2)

	// Call composed hooks.
	ctx := context.Background()
	composed.OnTurn(ctx, "user", "agent")
	composed.OnInterrupt(ctx)
	composed.OnToolCall(ctx, schema.ToolCall{Name: "search"})

	expected := []string{
		"hooks1-turn:user:agent",
		"hooks2-turn:user:agent",
		"hooks1-interrupt",
		"hooks2-tool:search",
	}
	assert.Equal(t, expected, callOrder)
}

func TestComposeHooks_ErrorHandling(t *testing.T) {
	var called []string

	hooks1 := Hooks{
		OnError: func(ctx context.Context, err error) error {
			called = append(called, "hooks1")
			// First hook suppresses the error by returning nil.
			return nil
		},
	}

	hooks2 := Hooks{
		OnError: func(ctx context.Context, err error) error {
			called = append(called, "hooks2")
			// Second hook also suppresses the error.
			return nil
		},
	}

	composed := ComposeHooks(hooks1, hooks2)
	err := composed.OnError(context.Background(), assert.AnError)

	// Both hooks should be called.
	assert.Equal(t, []string{"hooks1", "hooks2"}, called)
	// The original error is returned at the end if no hook returns an error.
	assert.Error(t, err)
}

func TestMockS2S_StartWithOptions(t *testing.T) {
	engine := &mockS2S{
		startFunc: func(ctx context.Context, opts ...Option) (Session, error) {
			cfg := ApplyOptions(opts...)
			assert.Equal(t, "nova", cfg.Voice)
			assert.Equal(t, "test-model", cfg.Model)
			return newMockSession(), nil
		},
	}

	session, err := engine.Start(
		context.Background(),
		WithVoice("nova"),
		WithModel("test-model"),
	)
	require.NoError(t, err)
	require.NotNil(t, session)
	defer session.Close()
}

func TestSessionEvent_ToolCall(t *testing.T) {
	event := SessionEvent{
		Type: EventToolCall,
		ToolCall: &schema.ToolCall{
			ID:        "call-123",
			Name:      "search",
			Arguments: `{"query":"test"}`,
		},
	}

	assert.Equal(t, EventToolCall, event.Type)
	require.NotNil(t, event.ToolCall)
	assert.Equal(t, "call-123", event.ToolCall.ID)
	assert.Equal(t, "search", event.ToolCall.Name)
}

func TestSessionEvent_Error(t *testing.T) {
	testErr := assert.AnError
	event := SessionEvent{
		Type:  EventError,
		Error: testErr,
	}

	assert.Equal(t, EventError, event.Type)
	assert.Equal(t, testErr, event.Error)
}
