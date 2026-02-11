package voice

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/lookatitude/beluga-ai/schema"
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

// --- Additional mock types for coverage ---

// errorTransport returns an error from Recv.
type errorTransport struct {
	recvErr error
}

func (e *errorTransport) Recv(_ context.Context) (<-chan Frame, error) {
	return nil, e.recvErr
}

func (e *errorTransport) Send(_ context.Context, _ Frame) error { return nil }
func (e *errorTransport) Close() error                          { return nil }

// sendErrorTransport succeeds on Recv but returns an error from Send.
type sendErrorTransport struct {
	frames  []Frame
	sendErr error
}

func (s *sendErrorTransport) Recv(_ context.Context) (<-chan Frame, error) {
	ch := make(chan Frame, len(s.frames))
	for _, f := range s.frames {
		ch <- f
	}
	close(ch)
	return ch, nil
}

func (s *sendErrorTransport) Send(_ context.Context, _ Frame) error {
	return s.sendErr
}

func (s *sendErrorTransport) Close() error { return nil }

// errorVAD always returns an error from DetectActivity.
type errorVAD struct {
	err error
}

func (v *errorVAD) DetectActivity(_ context.Context, _ []byte) (ActivityResult, error) {
	return ActivityResult{}, v.err
}

// --- Events tests ---

func TestPipelineEventsError(t *testing.T) {
	// Pipeline with no transport → Run() fails → Events yields the error.
	p := NewPipeline(WithSTT(passThroughProcessor))

	var gotErr error
	for _, err := range p.Events(context.Background()) {
		if err != nil {
			gotErr = err
			break
		}
	}
	if gotErr == nil {
		t.Error("Events() should yield an error when pipeline fails")
	}
}

func TestPipelineEventsSuccess(t *testing.T) {
	// Pipeline succeeds → Events yields nothing (no error).
	transport := &mockTransport{
		frames: []Frame{NewTextFrame("hello")},
	}
	p := NewPipeline(
		WithTransport(transport),
		WithSTT(passThroughProcessor),
	)

	var count int
	for _, err := range p.Events(context.Background()) {
		_ = err
		count++
	}
	if count != 0 {
		t.Errorf("Events() yielded %d items, want 0 for successful pipeline", count)
	}
}

// --- Transport error path tests ---

func TestPipelineRunTransportRecvError(t *testing.T) {
	recvErr := fmt.Errorf("connection refused")
	p := NewPipeline(
		WithTransport(&errorTransport{recvErr: recvErr}),
		WithSTT(passThroughProcessor),
	)

	err := p.Run(context.Background())
	if err == nil {
		t.Fatal("Run() should return error when Recv fails")
	}
	if !errors.Is(err, recvErr) {
		// Wrapped error check
		if err.Error() != "voice: transport recv: connection refused" {
			t.Errorf("Run() error = %q, want wrapped recv error", err)
		}
	}
}

func TestPipelineRunTransportSendError(t *testing.T) {
	sendErr := fmt.Errorf("write broken pipe")
	p := NewPipeline(
		WithTransport(&sendErrorTransport{
			frames:  []Frame{NewTextFrame("hello")},
			sendErr: sendErr,
		}),
		WithSTT(passThroughProcessor),
	)

	err := p.Run(context.Background())
	if err == nil {
		t.Fatal("Run() should return error when Send fails")
	}
}

// --- VAD processor additional tests ---

func TestPipelineVADNonAudioPassThrough(t *testing.T) {
	// Non-audio frames should pass through VAD unchanged.
	transport := &mockTransport{
		frames: []Frame{
			NewTextFrame("text-frame"),
			NewControlFrame(SignalStart),
		},
	}

	p := NewPipeline(
		WithTransport(transport),
		WithVAD(NewEnergyVAD(EnergyVADConfig{Threshold: 500})),
		WithSTT(passThroughProcessor),
	)

	err := p.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if len(transport.sent) != 2 {
		t.Fatalf("sent %d frames, want 2", len(transport.sent))
	}
	if transport.sent[0].Type != FrameText {
		t.Errorf("sent[0].Type = %q, want %q", transport.sent[0].Type, FrameText)
	}
	if transport.sent[0].Text() != "text-frame" {
		t.Errorf("sent[0].Text() = %q, want %q", transport.sent[0].Text(), "text-frame")
	}
	if transport.sent[1].Type != FrameControl {
		t.Errorf("sent[1].Type = %q, want %q", transport.sent[1].Type, FrameControl)
	}
}

func TestPipelineVADErrorOnErrorHookSuppresses(t *testing.T) {
	// VAD returns error, OnError hook returns nil → error is suppressed, continue.
	vadErr := fmt.Errorf("vad processing failed")
	transport := &mockTransport{
		frames: []Frame{
			NewAudioFrame(generatePCM(480, 100), 16000),
			NewTextFrame("after-error"), // non-audio, should pass through
		},
	}

	var hookCalled bool
	p := NewPipeline(
		WithTransport(transport),
		WithVAD(&errorVAD{err: vadErr}),
		WithSTT(passThroughProcessor),
		WithHooks(Hooks{
			OnError: func(_ context.Context, err error) error {
				hookCalled = true
				return nil // suppress error
			},
		}),
	)

	err := p.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !hookCalled {
		t.Error("OnError hook was not called")
	}
	// The text frame should have passed through.
	if len(transport.sent) != 1 {
		t.Fatalf("sent %d frames, want 1 (text only)", len(transport.sent))
	}
	if transport.sent[0].Text() != "after-error" {
		t.Errorf("sent[0].Text() = %q, want %q", transport.sent[0].Text(), "after-error")
	}
}

func TestPipelineVADErrorOnErrorHookPropagates(t *testing.T) {
	// VAD returns error, OnError hook returns non-nil → propagates as fatal error.
	vadErr := fmt.Errorf("vad failure")
	hookErr := fmt.Errorf("fatal: vad failure")
	transport := &mockTransport{
		frames: []Frame{
			NewAudioFrame(generatePCM(480, 100), 16000),
		},
	}

	p := NewPipeline(
		WithTransport(transport),
		WithVAD(&errorVAD{err: vadErr}),
		WithSTT(passThroughProcessor),
		WithHooks(Hooks{
			OnError: func(_ context.Context, _ error) error {
				return hookErr
			},
		}),
	)

	err := p.Run(context.Background())
	if err == nil {
		t.Fatal("Run() should return error when OnError hook propagates")
	}
	if !errors.Is(err, hookErr) {
		t.Errorf("Run() error = %v, want %v", err, hookErr)
	}
}

func TestPipelineVADErrorNoHook(t *testing.T) {
	// VAD returns error but no OnError hook → error is silently skipped, continue.
	vadErr := fmt.Errorf("vad processing failed")
	transport := &mockTransport{
		frames: []Frame{
			NewAudioFrame(generatePCM(480, 100), 16000),
			NewTextFrame("pass-through"),
		},
	}

	p := NewPipeline(
		WithTransport(transport),
		WithVAD(&errorVAD{err: vadErr}),
		WithSTT(passThroughProcessor),
		// No hooks set
	)

	err := p.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	// Text frame should still pass through.
	if len(transport.sent) != 1 {
		t.Fatalf("sent %d frames, want 1", len(transport.sent))
	}
}

func TestPipelineVADContextCancel(t *testing.T) {
	// Cancel context during VAD processing.
	ctx, cancel := context.WithCancel(context.Background())

	// Create a transport that blocks until context is cancelled.
	blockingRecv := make(chan Frame)
	blockTransport := &mockTransport{}
	// Override Recv to return a blocking channel.
	bt := &blockingTransport{recv: blockingRecv}

	p := NewPipeline(
		WithTransport(bt),
		WithVAD(NewEnergyVAD(EnergyVADConfig{Threshold: 500})),
		WithSTT(passThroughProcessor),
	)
	_ = blockTransport // unused

	done := make(chan error, 1)
	go func() {
		done <- p.Run(ctx)
	}()

	cancel()
	err := <-done
	if err == nil {
		t.Error("Run() should return error on context cancel")
	}
}

// blockingTransport provides a transport that uses a provided channel.
type blockingTransport struct {
	recv chan Frame
	sent []Frame
}

func (b *blockingTransport) Recv(_ context.Context) (<-chan Frame, error) {
	return b.recv, nil
}

func (b *blockingTransport) Send(_ context.Context, frame Frame) error {
	b.sent = append(b.sent, frame)
	return nil
}

func (b *blockingTransport) Close() error { return nil }

func TestPipelineVADSpeechFiltersSilence(t *testing.T) {
	// Verify that audio frames with no speech are NOT forwarded.
	quietAudio := generatePCM(480, 10) // Below threshold
	transport := &mockTransport{
		frames: []Frame{
			NewAudioFrame(quietAudio, 16000),
		},
	}

	p := NewPipeline(
		WithTransport(transport),
		WithVAD(NewEnergyVAD(EnergyVADConfig{Threshold: 500})),
		WithSTT(passThroughProcessor),
	)

	err := p.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Quiet audio should be filtered; nothing sent through transport.
	if len(transport.sent) != 0 {
		t.Errorf("sent %d frames, want 0 (silence filtered)", len(transport.sent))
	}
}

func TestPipelineMultipleProcessors(t *testing.T) {
	// Test with STT + LLM + TTS all as pass-through.
	transport := &mockTransport{
		frames: []Frame{NewTextFrame("input")},
	}

	p := NewPipeline(
		WithTransport(transport),
		WithSTT(passThroughProcessor),
		WithLLM(passThroughProcessor),
		WithTTS(passThroughProcessor),
	)

	err := p.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(transport.sent) != 1 {
		t.Fatalf("sent %d frames, want 1", len(transport.sent))
	}
	if transport.sent[0].Text() != "input" {
		t.Errorf("sent[0].Text() = %q, want %q", transport.sent[0].Text(), "input")
	}
}

// Silence the unused import warning for schema.
var _ schema.AgentEvent
