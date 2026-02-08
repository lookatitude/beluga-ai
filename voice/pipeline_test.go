package voice

import (
	"context"
	"errors"
	"testing"
)

// mockTransport implements Transport for testing.
type mockTransport struct {
	frames []Frame
	sent   []Frame
}

func (m *mockTransport) Recv(_ context.Context) (<-chan Frame, error) {
	ch := make(chan Frame, len(m.frames))
	for _, f := range m.frames {
		ch <- f
	}
	close(ch)
	return ch, nil
}

func (m *mockTransport) Send(_ context.Context, frame Frame) error {
	m.sent = append(m.sent, frame)
	return nil
}

func (m *mockTransport) Close() error { return nil }

// passThroughProcessor forwards all frames unchanged.
var passThroughProcessor = FrameProcessorFunc(func(_ context.Context, in <-chan Frame, out chan<- Frame) error {
	defer close(out)
	for f := range in {
		out <- f
	}
	return nil
})

func TestNewPipeline(t *testing.T) {
	p := NewPipeline()
	if p == nil {
		t.Fatal("NewPipeline() returned nil")
	}
	if p.config.ChannelBufferSize != 64 {
		t.Errorf("ChannelBufferSize = %d, want 64", p.config.ChannelBufferSize)
	}
}

func TestPipelineOptions(t *testing.T) {
	transport := &mockTransport{}
	vad := NewEnergyVAD(EnergyVADConfig{Threshold: 500})
	session := NewSession("test")

	p := NewPipeline(
		WithTransport(transport),
		WithVAD(vad),
		WithSTT(passThroughProcessor),
		WithLLM(passThroughProcessor),
		WithTTS(passThroughProcessor),
		WithSession(session),
		WithChannelBufferSize(128),
	)

	if p.config.Transport != transport {
		t.Error("Transport not set")
	}
	if p.config.VAD == nil {
		t.Error("VAD not set")
	}
	if p.config.STT == nil {
		t.Error("STT not set")
	}
	if p.config.LLM == nil {
		t.Error("LLM not set")
	}
	if p.config.TTS == nil {
		t.Error("TTS not set")
	}
	if p.config.Session != session {
		t.Error("Session not set")
	}
	if p.config.ChannelBufferSize != 128 {
		t.Errorf("ChannelBufferSize = %d, want 128", p.config.ChannelBufferSize)
	}
}

func TestPipelineRunNoTransport(t *testing.T) {
	p := NewPipeline(WithSTT(passThroughProcessor))
	err := p.Run(context.Background())
	if err == nil {
		t.Error("expected error for missing transport")
	}
}

func TestPipelineRunNoProcessors(t *testing.T) {
	transport := &mockTransport{}
	p := NewPipeline(WithTransport(transport))
	err := p.Run(context.Background())
	if err == nil {
		t.Error("expected error for no processors")
	}
}

func TestPipelineRunPassThrough(t *testing.T) {
	transport := &mockTransport{
		frames: []Frame{
			NewTextFrame("hello"),
			NewTextFrame("world"),
		},
	}

	p := NewPipeline(
		WithTransport(transport),
		WithSTT(passThroughProcessor),
	)

	err := p.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if len(transport.sent) != 2 {
		t.Fatalf("sent %d frames, want 2", len(transport.sent))
	}
	if transport.sent[0].Text() != "hello" {
		t.Errorf("sent[0].Text() = %q, want %q", transport.sent[0].Text(), "hello")
	}
	if transport.sent[1].Text() != "world" {
		t.Errorf("sent[1].Text() = %q, want %q", transport.sent[1].Text(), "world")
	}
}

func TestPipelineWithVAD(t *testing.T) {
	// Create loud and quiet audio frames.
	loudAudio := generateSinePCM(480, 5000, 440, 16000)
	quietAudio := generatePCM(480, 10)

	transport := &mockTransport{
		frames: []Frame{
			NewAudioFrame(loudAudio, 16000),
			NewAudioFrame(quietAudio, 16000),
		},
	}

	var speechStarted, speechEnded bool
	hooks := Hooks{
		OnSpeechStart: func(_ context.Context) { speechStarted = true },
		OnSpeechEnd:   func(_ context.Context) { speechEnded = true },
	}

	p := NewPipeline(
		WithTransport(transport),
		WithVAD(NewEnergyVAD(EnergyVADConfig{Threshold: 500})),
		WithSTT(passThroughProcessor),
		WithHooks(hooks),
	)

	err := p.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if !speechStarted {
		t.Error("OnSpeechStart was not called")
	}
	if !speechEnded {
		t.Error("OnSpeechEnd was not called")
	}
}

func TestComposeHooks(t *testing.T) {
	var calls []string

	h1 := Hooks{
		OnSpeechStart: func(_ context.Context) { calls = append(calls, "h1_start") },
		OnTranscript:  func(_ context.Context, text string) { calls = append(calls, "h1_transcript:"+text) },
	}
	h2 := Hooks{
		OnSpeechStart: func(_ context.Context) { calls = append(calls, "h2_start") },
		OnResponse:    func(_ context.Context, text string) { calls = append(calls, "h2_response:"+text) },
	}

	composed := ComposeHooks(h1, h2)
	ctx := context.Background()

	composed.OnSpeechStart(ctx)
	composed.OnTranscript(ctx, "hello")
	composed.OnResponse(ctx, "world")

	expected := []string{"h1_start", "h2_start", "h1_transcript:hello", "h2_response:world"}
	if len(calls) != len(expected) {
		t.Fatalf("calls = %v, want %v", calls, expected)
	}
	for i, c := range calls {
		if c != expected[i] {
			t.Errorf("calls[%d] = %q, want %q", i, c, expected[i])
		}
	}
}

func TestComposeHooksOnError(t *testing.T) {
	testErr := errors.New("test error")

	h1 := Hooks{
		OnError: func(_ context.Context, err error) error { return err },
	}
	h2 := Hooks{
		OnError: func(_ context.Context, _ error) error { return nil },
	}

	// h1 passes error through, so composed should return the error.
	composed := ComposeHooks(h1, h2)
	if err := composed.OnError(context.Background(), testErr); err != testErr {
		t.Errorf("OnError = %v, want %v", err, testErr)
	}
}

func TestComposeHooksNilFields(t *testing.T) {
	// Composing hooks with nil fields should not panic.
	h1 := Hooks{} // all nil
	h2 := Hooks{OnSpeechEnd: func(_ context.Context) {}}

	composed := ComposeHooks(h1, h2)
	// Should not panic.
	composed.OnSpeechStart(context.Background())
	composed.OnSpeechEnd(context.Background())
	composed.OnTranscript(context.Background(), "test")
	composed.OnResponse(context.Background(), "test")
	err := composed.OnError(context.Background(), errors.New("test"))
	if err == nil {
		t.Error("OnError should pass through error by default")
	}
}
