package s2s

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/schema"
	"github.com/lookatitude/beluga-ai/voice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockSession is a test implementation of the Session interface.
type mockSession struct {
	recvChan    chan SessionEvent
	audioSent   [][]byte
	textSent    []string
	resultSent  []schema.ToolResult
	interrupted bool
	closed      bool
	sendAudioFn func(ctx context.Context, audio []byte) error
	sendTextFn  func(ctx context.Context, text string) error
	interruptFn func(ctx context.Context) error
}

var _ Session = (*mockSession)(nil)

func newMockSession() *mockSession {
	return &mockSession{
		recvChan: make(chan SessionEvent, 10),
	}
}

func (m *mockSession) SendAudio(ctx context.Context, audio []byte) error {
	if m.sendAudioFn != nil {
		return m.sendAudioFn(ctx, audio)
	}
	m.audioSent = append(m.audioSent, audio)
	return nil
}

func (m *mockSession) SendText(ctx context.Context, text string) error {
	if m.sendTextFn != nil {
		return m.sendTextFn(ctx, text)
	}
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
	if m.interruptFn != nil {
		return m.interruptFn(ctx)
	}
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

func TestRegistry_PanicOnEmptyName(t *testing.T) {
	assert.Panics(t, func() {
		Register("", func(cfg Config) (S2S, error) {
			return &mockS2S{}, nil
		})
	})
}

func TestRegistry_PanicOnNilFactory(t *testing.T) {
	assert.Panics(t, func() {
		Register("test-s2s-nil-factory", nil)
	})
}

func TestRegistry_PanicOnDuplicate(t *testing.T) {
	Register("test-s2s-dup-check", func(cfg Config) (S2S, error) {
		return &mockS2S{}, nil
	})
	assert.Panics(t, func() {
		Register("test-s2s-dup-check", func(cfg Config) (S2S, error) {
			return &mockS2S{}, nil
		})
	})
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

func TestComposeHooks_OnError_ShortCircuit(t *testing.T) {
	var called []string
	interceptedErr := errors.New("intercepted")

	hooks1 := Hooks{
		OnError: func(ctx context.Context, err error) error {
			called = append(called, "hooks1")
			return interceptedErr // non-nil: short-circuits
		},
	}

	hooks2 := Hooks{
		OnError: func(ctx context.Context, err error) error {
			called = append(called, "hooks2")
			return nil
		},
	}

	composed := ComposeHooks(hooks1, hooks2)
	err := composed.OnError(context.Background(), assert.AnError)

	// Only hooks1 should be called; hooks2 is short-circuited.
	assert.Equal(t, []string{"hooks1"}, called)
	assert.Equal(t, interceptedErr, err)
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

// --- AsFrameProcessor tests ---

func TestAsFrameProcessor_AudioForwarding(t *testing.T) {
	session := newMockSession()
	// Pre-close recv so the output goroutine finishes immediately.
	close(session.recvChan)
	session.closed = true

	engine := &mockS2S{session: session}
	proc := AsFrameProcessor(engine)

	in := make(chan voice.Frame, 5)
	out := make(chan voice.Frame, 5)

	in <- voice.NewAudioFrame([]byte{0x01, 0x02}, 16000)
	in <- voice.NewAudioFrame([]byte{0x03, 0x04}, 16000)
	close(in)

	err := proc.Process(context.Background(), in, out)
	require.NoError(t, err)

	assert.Len(t, session.audioSent, 2)
	assert.Equal(t, []byte{0x01, 0x02}, session.audioSent[0])
	assert.Equal(t, []byte{0x03, 0x04}, session.audioSent[1])
}

func TestAsFrameProcessor_TextForwarding(t *testing.T) {
	session := newMockSession()
	close(session.recvChan)
	session.closed = true

	engine := &mockS2S{session: session}
	proc := AsFrameProcessor(engine)

	in := make(chan voice.Frame, 5)
	out := make(chan voice.Frame, 5)

	in <- voice.NewTextFrame("hello from user")
	close(in)

	err := proc.Process(context.Background(), in, out)
	require.NoError(t, err)

	assert.Len(t, session.textSent, 1)
	assert.Equal(t, "hello from user", session.textSent[0])
}

func TestAsFrameProcessor_InterruptForwarding(t *testing.T) {
	session := newMockSession()
	close(session.recvChan)
	session.closed = true

	engine := &mockS2S{session: session}
	proc := AsFrameProcessor(engine)

	in := make(chan voice.Frame, 5)
	out := make(chan voice.Frame, 5)

	in <- voice.NewControlFrame(voice.SignalInterrupt)
	close(in)

	err := proc.Process(context.Background(), in, out)
	require.NoError(t, err)

	assert.True(t, session.interrupted)
}

func TestAsFrameProcessor_NonInterruptControlIgnored(t *testing.T) {
	session := newMockSession()
	close(session.recvChan)
	session.closed = true

	engine := &mockS2S{session: session}
	proc := AsFrameProcessor(engine)

	in := make(chan voice.Frame, 5)
	out := make(chan voice.Frame, 5)

	// Send a non-interrupt control frame; should not trigger interrupt.
	in <- voice.NewControlFrame(voice.SignalStart)
	close(in)

	err := proc.Process(context.Background(), in, out)
	require.NoError(t, err)

	assert.False(t, session.interrupted)
}

func TestAsFrameProcessor_AudioOutputEvent(t *testing.T) {
	session := newMockSession()
	session.recvChan <- SessionEvent{Type: EventAudioOutput, Audio: []byte{0xAA, 0xBB}}
	close(session.recvChan)
	session.closed = true

	engine := &mockS2S{session: session}
	proc := AsFrameProcessor(engine)

	in := make(chan voice.Frame)
	out := make(chan voice.Frame, 10)

	close(in)

	err := proc.Process(context.Background(), in, out)
	require.NoError(t, err)

	var frames []voice.Frame
	for f := range out {
		frames = append(frames, f)
	}

	require.Len(t, frames, 1)
	assert.Equal(t, voice.FrameAudio, frames[0].Type)
	assert.Equal(t, []byte{0xAA, 0xBB}, frames[0].Data)
	assert.Equal(t, 24000, frames[0].Metadata["sample_rate"])
}

func TestAsFrameProcessor_TextOutputEvent(t *testing.T) {
	session := newMockSession()
	session.recvChan <- SessionEvent{Type: EventTextOutput, Text: "hello from model"}
	close(session.recvChan)
	session.closed = true

	engine := &mockS2S{session: session}
	proc := AsFrameProcessor(engine)

	in := make(chan voice.Frame)
	out := make(chan voice.Frame, 10)

	close(in)

	err := proc.Process(context.Background(), in, out)
	require.NoError(t, err)

	var frames []voice.Frame
	for f := range out {
		frames = append(frames, f)
	}

	require.Len(t, frames, 1)
	assert.Equal(t, voice.FrameText, frames[0].Type)
	assert.Equal(t, "hello from model", frames[0].Text())
}

func TestAsFrameProcessor_TurnEndEvent(t *testing.T) {
	session := newMockSession()
	session.recvChan <- SessionEvent{Type: EventTurnEnd}
	close(session.recvChan)
	session.closed = true

	engine := &mockS2S{session: session}
	proc := AsFrameProcessor(engine)

	in := make(chan voice.Frame)
	out := make(chan voice.Frame, 10)

	close(in)

	err := proc.Process(context.Background(), in, out)
	require.NoError(t, err)

	var frames []voice.Frame
	for f := range out {
		frames = append(frames, f)
	}

	require.Len(t, frames, 1)
	assert.Equal(t, voice.FrameControl, frames[0].Type)
	assert.Equal(t, voice.SignalEndOfUtterance, frames[0].Signal())
}

func TestAsFrameProcessor_ErrorEvent(t *testing.T) {
	session := newMockSession()
	testErr := errors.New("session error")
	session.recvChan <- SessionEvent{Type: EventError, Error: testErr}

	engine := &mockS2S{session: session}
	proc := AsFrameProcessor(engine)

	in := make(chan voice.Frame)
	out := make(chan voice.Frame, 10)

	close(in)

	err := proc.Process(context.Background(), in, out)
	assert.ErrorIs(t, err, testErr)
}

func TestAsFrameProcessor_ErrorEventNilError(t *testing.T) {
	// EventError with nil Error field should not cause failure.
	session := newMockSession()
	session.recvChan <- SessionEvent{Type: EventError, Error: nil}
	close(session.recvChan)
	session.closed = true

	engine := &mockS2S{session: session}
	proc := AsFrameProcessor(engine)

	in := make(chan voice.Frame)
	out := make(chan voice.Frame, 10)

	close(in)

	err := proc.Process(context.Background(), in, out)
	require.NoError(t, err)
}

func TestAsFrameProcessor_StartError(t *testing.T) {
	engine := &mockS2S{
		startFunc: func(ctx context.Context, opts ...Option) (Session, error) {
			return nil, errors.New("connection failed")
		},
	}
	proc := AsFrameProcessor(engine)

	in := make(chan voice.Frame)
	out := make(chan voice.Frame, 10)

	err := proc.Process(context.Background(), in, out)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "s2s: start session:")
	assert.Contains(t, err.Error(), "connection failed")
}

func TestAsFrameProcessor_ContextCancellation(t *testing.T) {
	session := newMockSession()

	engine := &mockS2S{session: session}
	proc := AsFrameProcessor(engine)

	ctx, cancel := context.WithCancel(context.Background())
	in := make(chan voice.Frame) // unbuffered so main loop blocks on select
	out := make(chan voice.Frame, 10)

	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	err := proc.Process(ctx, in, out)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestAsFrameProcessor_SendAudioError(t *testing.T) {
	session := newMockSession()
	close(session.recvChan)
	session.closed = true
	session.sendAudioFn = func(ctx context.Context, audio []byte) error {
		return errors.New("send audio failed")
	}

	engine := &mockS2S{session: session}
	proc := AsFrameProcessor(engine)

	in := make(chan voice.Frame, 5)
	out := make(chan voice.Frame, 5)

	in <- voice.NewAudioFrame([]byte{0x01}, 16000)
	close(in)

	err := proc.Process(context.Background(), in, out)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "s2s: send audio:")
}

func TestAsFrameProcessor_SendTextError(t *testing.T) {
	session := newMockSession()
	close(session.recvChan)
	session.closed = true
	session.sendTextFn = func(ctx context.Context, text string) error {
		return errors.New("send text failed")
	}

	engine := &mockS2S{session: session}
	proc := AsFrameProcessor(engine)

	in := make(chan voice.Frame, 5)
	out := make(chan voice.Frame, 5)

	in <- voice.NewTextFrame("hello")
	close(in)

	err := proc.Process(context.Background(), in, out)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "s2s: send text:")
}

func TestAsFrameProcessor_InterruptError(t *testing.T) {
	session := newMockSession()
	close(session.recvChan)
	session.closed = true
	session.interruptFn = func(ctx context.Context) error {
		return errors.New("interrupt failed")
	}

	engine := &mockS2S{session: session}
	proc := AsFrameProcessor(engine)

	in := make(chan voice.Frame, 5)
	out := make(chan voice.Frame, 5)

	in <- voice.NewControlFrame(voice.SignalInterrupt)
	close(in)

	err := proc.Process(context.Background(), in, out)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "s2s: interrupt:")
}

func TestAsFrameProcessor_MultipleOutputEvents(t *testing.T) {
	session := newMockSession()
	session.recvChan <- SessionEvent{Type: EventAudioOutput, Audio: []byte{0x01}}
	session.recvChan <- SessionEvent{Type: EventTextOutput, Text: "transcript"}
	session.recvChan <- SessionEvent{Type: EventTurnEnd}
	close(session.recvChan)
	session.closed = true

	engine := &mockS2S{session: session}
	proc := AsFrameProcessor(engine)

	in := make(chan voice.Frame)
	out := make(chan voice.Frame, 10)

	close(in)

	err := proc.Process(context.Background(), in, out)
	require.NoError(t, err)

	var frames []voice.Frame
	for f := range out {
		frames = append(frames, f)
	}

	require.Len(t, frames, 3)
	assert.Equal(t, voice.FrameAudio, frames[0].Type)
	assert.Equal(t, voice.FrameText, frames[1].Type)
	assert.Equal(t, voice.FrameControl, frames[2].Type)
}
