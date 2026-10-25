package nova

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lookatitude/beluga-ai/v2/schema"
	"github.com/lookatitude/beluga-ai/v2/voice/s2s"
)

// fakeStream is an in-memory eventStream.
type fakeStream struct {
	mu      sync.Mutex
	sent    []map[string]map[string]any
	in      chan []byte
	recvErr error
	sendErr error
	closed  bool
	once    sync.Once
}

func newFakeStream() *fakeStream { return &fakeStream{in: make(chan []byte, 32)} }

func (f *fakeStream) Send(_ context.Context, payload []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.sendErr != nil {
		return f.sendErr
	}
	var env struct {
		Event map[string]map[string]any `json:"event"`
	}
	if err := json.Unmarshal(payload, &env); err != nil {
		return err
	}
	f.sent = append(f.sent, env.Event)
	return nil
}

func (f *fakeStream) Recv() ([]byte, error) {
	b, ok := <-f.in
	if !ok {
		f.mu.Lock()
		defer f.mu.Unlock()
		if f.recvErr != nil {
			return nil, f.recvErr
		}
		return nil, io.EOF
	}
	return b, nil
}

func (f *fakeStream) Close() error {
	f.once.Do(func() {
		f.mu.Lock()
		f.closed = true
		f.mu.Unlock()
		close(f.in)
	})
	return nil
}

// names returns the event names sent so far, in order.
func (f *fakeStream) names() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []string
	for _, ev := range f.sent {
		for k := range ev {
			out = append(out, k)
		}
	}
	return out
}

func (f *fakeStream) event(i int) (string, map[string]any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for k, v := range f.sent[i] {
		return k, v
	}
	return "", nil
}

func (f *fakeStream) push(t *testing.T, name string, body map[string]any) {
	t.Helper()
	b, err := json.Marshal(map[string]any{"event": map[string]any{name: body}})
	require.NoError(t, err)
	f.in <- b
}

func startFake(t *testing.T, cfg s2s.Config, opts ...s2s.Option) (*fakeStream, s2s.Session) {
	t.Helper()
	fs := newFakeStream()
	var gotModel string
	e := newEngine("us-east-1", cfg, func(_ context.Context, model string) (eventStream, error) {
		gotModel = model
		return fs, nil
	})
	sess, err := e.Start(context.Background(), opts...)
	require.NoError(t, err)
	if cfg.Model != "" {
		assert.Equal(t, cfg.Model, gotModel)
	}
	t.Cleanup(func() { _ = sess.Close() })
	return fs, sess
}

func collect(t *testing.T, sess s2s.Session, n int) []s2s.SessionEvent {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var out []s2s.SessionEvent
	for ev, err := range sess.Recv(ctx) {
		require.NoError(t, err)
		out = append(out, ev)
		if len(out) == n {
			break
		}
	}
	return out
}

func TestNew(t *testing.T) {
	e, err := New(s2s.Config{Extra: map[string]any{
		"region": "eu-west-1", "access_key_id": "AKID", "secret_access_key": "secret",
		"base_url": "https://bedrock-runtime.eu-west-1.amazonaws.com",
	}})
	require.NoError(t, err)
	assert.Equal(t, "eu-west-1", e.region)
	assert.Equal(t, defaultModel, e.cfg.Model)

	e, err = New(s2s.Config{Model: "amazon.nova-2-sonic-v1:0"})
	require.NoError(t, err)
	assert.Equal(t, defaultRegion, e.region)
	assert.Equal(t, "amazon.nova-2-sonic-v1:0", e.cfg.Model)
}

// FI-12: the legacy unsigned WebSocket endpoint must not be accepted.
func TestNew_RejectsWebSocketBaseURL(t *testing.T) {
	_, err := New(s2s.Config{Extra: map[string]any{"base_url": "wss://bedrock-runtime.us-east-1.amazonaws.com/model/x/converse-stream"}})
	require.Error(t, err)
}

func TestStart_SendsPreamble(t *testing.T) {
	fs, _ := startFake(t, s2s.Config{
		Model:        defaultModel,
		Voice:        "tiffany",
		Instructions: "be brief",
		SampleRate:   8000,
		Tools: []schema.ToolDefinition{{
			Name: "lookup", Description: "find", InputSchema: map[string]any{"type": "object"},
		}},
	})
	assert.Equal(t, []string{"sessionStart", "promptStart", "contentStart", "textInput", "contentEnd", "contentStart"}, fs.names())

	_, prompt := fs.event(1)
	audioOut := prompt["audioOutputConfiguration"].(map[string]any)
	assert.Equal(t, "tiffany", audioOut["voiceId"])
	tools := prompt["toolConfiguration"].(map[string]any)["tools"].([]any)
	spec := tools[0].(map[string]any)["toolSpec"].(map[string]any)
	assert.Equal(t, "lookup", spec["name"])
	assert.Equal(t, `{"type":"object"}`, spec["inputSchema"].(map[string]any)["json"])

	_, sys := fs.event(2)
	assert.Equal(t, "SYSTEM", sys["role"])
	_, txt := fs.event(3)
	assert.Equal(t, "be brief", txt["content"])

	_, audio := fs.event(5)
	assert.Equal(t, "AUDIO", audio["type"])
	assert.Equal(t, true, audio["interactive"])
	assert.EqualValues(t, 8000, audio["audioInputConfiguration"].(map[string]any)["sampleRateHertz"])
}

func TestStart_DefaultsWithoutInstructionsOrTools(t *testing.T) {
	fs, _ := startFake(t, s2s.Config{}, s2s.WithVoice("amy"))
	assert.Equal(t, []string{"sessionStart", "promptStart", "contentStart"}, fs.names())
	_, prompt := fs.event(1)
	assert.Nil(t, prompt["toolConfiguration"])
	assert.Equal(t, "amy", prompt["audioOutputConfiguration"].(map[string]any)["voiceId"])
	_, audio := fs.event(2)
	assert.EqualValues(t, defaultInputSampleRate, audio["audioInputConfiguration"].(map[string]any)["sampleRateHertz"])
}

func TestStart_OpenError(t *testing.T) {
	e := newEngine("us-east-1", s2s.Config{}, func(context.Context, string) (eventStream, error) {
		return nil, errors.New("no creds")
	})
	_, err := e.Start(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "open bidirectional stream")
}

func TestStart_PreambleError(t *testing.T) {
	fs := newFakeStream()
	fs.sendErr = errors.New("broken")
	e := newEngine("us-east-1", s2s.Config{}, func(context.Context, string) (eventStream, error) { return fs, nil })
	_, err := e.Start(context.Background())
	require.Error(t, err)
	assert.True(t, fs.closed)
}

func TestSendMethods(t *testing.T) {
	fs, sess := startFake(t, s2s.Config{})
	ctx := context.Background()
	base := len(fs.names())

	require.NoError(t, sess.SendAudio(ctx, []byte{1, 2, 3}))
	name, body := fs.event(base)
	assert.Equal(t, "audioInput", name)
	assert.Equal(t, base64.StdEncoding.EncodeToString([]byte{1, 2, 3}), body["content"])

	require.NoError(t, sess.SendText(ctx, "hello"))
	assert.Equal(t, []string{"contentStart", "textInput", "contentEnd"}, fs.names()[base+1:base+4])
	_, start := fs.event(base + 1)
	assert.Equal(t, "USER", start["role"])

	require.NoError(t, sess.SendToolResult(ctx, schema.ToolResult{
		CallID:  "tu-1",
		Content: []schema.ContentPart{schema.TextPart{Text: "42"}},
	}))
	_, tstart := fs.event(base + 4)
	assert.Equal(t, "TOOL", tstart["type"])
	assert.Equal(t, "tu-1", tstart["toolResultInputConfiguration"].(map[string]any)["toolUseId"])
	_, tres := fs.event(base + 5)
	assert.Equal(t, "42", tres["content"])

	require.NoError(t, sess.Interrupt(ctx))
}

func TestClose_EndsSessionAndRejectsSends(t *testing.T) {
	fs, sess := startFake(t, s2s.Config{})
	require.NoError(t, sess.Close())
	require.NoError(t, sess.Close())
	names := fs.names()
	assert.Equal(t, []string{"contentEnd", "promptEnd", "sessionEnd"}, names[len(names)-3:])
	assert.True(t, fs.closed)
	assert.Error(t, sess.SendAudio(context.Background(), []byte{1}))
	assert.Error(t, sess.Interrupt(context.Background()))
}

func TestRecv_TranslatesOutputEvents(t *testing.T) {
	fs, sess := startFake(t, s2s.Config{})
	audio := []byte{9, 8, 7}

	fs.push(t, "contentStart", map[string]any{"contentId": "c1", "role": "USER", "type": "TEXT"})
	fs.push(t, "textOutput", map[string]any{"contentId": "c1", "role": "USER", "content": "what time"})
	fs.push(t, "contentStart", map[string]any{"contentId": "c2", "role": "ASSISTANT", "type": "TEXT",
		"additionalModelFields": `{"generationStage":"SPECULATIVE"}`})
	fs.push(t, "textOutput", map[string]any{"contentId": "c2", "role": "ASSISTANT", "content": "draft"})
	fs.push(t, "contentEnd", map[string]any{"contentId": "c2", "stopReason": "PARTIAL_TURN"})
	fs.push(t, "contentStart", map[string]any{"contentId": "c3", "role": "ASSISTANT", "type": "TEXT",
		"additionalModelFields": `{"generationStage":"FINAL"}`})
	fs.push(t, "textOutput", map[string]any{"contentId": "c3", "role": "ASSISTANT", "content": "noon"})
	fs.push(t, "audioOutput", map[string]any{"content": base64.StdEncoding.EncodeToString(audio)})
	fs.push(t, "audioOutput", map[string]any{"content": "!!not base64"})
	fs.in <- []byte("not json")
	fs.push(t, "toolUse", map[string]any{"toolUseId": "t1", "toolName": "clock", "content": `{"tz":"UTC"}`})
	fs.push(t, "contentEnd", map[string]any{"contentId": "c3", "stopReason": "END_TURN"})

	events := collect(t, sess, 5)
	require.Len(t, events, 5)
	assert.Equal(t, s2s.SessionEvent{Type: s2s.EventTranscript, Text: "what time"}, events[0])
	assert.Equal(t, s2s.SessionEvent{Type: s2s.EventTextOutput, Text: "noon"}, events[1])
	assert.Equal(t, s2s.EventAudioOutput, events[2].Type)
	assert.Equal(t, audio, events[2].Audio)
	assert.Equal(t, s2s.EventToolCall, events[3].Type)
	assert.Equal(t, &schema.ToolCall{ID: "t1", Name: "clock", Arguments: `{"tz":"UTC"}`}, events[3].ToolCall)
	assert.Equal(t, s2s.EventTurnEnd, events[4].Type)
}

func TestRecv_StreamErrorSurfaces(t *testing.T) {
	fs, sess := startFake(t, s2s.Config{})
	fs.mu.Lock()
	fs.recvErr = errors.New("throttled")
	fs.mu.Unlock()
	close(fs.in)
	fs.once.Do(func() {})

	events := collect(t, sess, 1)
	require.Len(t, events, 1)
	assert.Equal(t, s2s.EventError, events[0].Type)
	assert.ErrorContains(t, events[0].Error, "throttled")
}

func TestRecv_ContextCancel(t *testing.T) {
	_, sess := startFake(t, s2s.Config{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for range sess.Recv(ctx) {
		t.Fatal("expected no events after cancel")
	}
}

func TestRegistry(t *testing.T) {
	assert.Contains(t, s2s.List(), "nova")
	engine, err := s2s.New("nova", s2s.Config{})
	require.NoError(t, err)
	assert.NotNil(t, engine)
}
