package tts

import (
	"context"
	"errors"
	"iter"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/voice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTTS is a test implementation of the TTS interface.
type mockTTS struct {
	synthesizeFunc       func(context.Context, string, ...Option) ([]byte, error)
	synthesizeStreamFunc func(context.Context, iter.Seq2[string, error], ...Option) iter.Seq2[[]byte, error]
}

// Compile-time interface check.
var _ TTS = (*mockTTS)(nil)

func (m *mockTTS) Synthesize(ctx context.Context, text string, opts ...Option) ([]byte, error) {
	if m.synthesizeFunc != nil {
		return m.synthesizeFunc(ctx, text, opts...)
	}
	return []byte("audio:" + text), nil
}

func (m *mockTTS) SynthesizeStream(ctx context.Context, textStream iter.Seq2[string, error], opts ...Option) iter.Seq2[[]byte, error] {
	if m.synthesizeStreamFunc != nil {
		return m.synthesizeStreamFunc(ctx, textStream, opts...)
	}
	return func(yield func([]byte, error) bool) {
		for text, err := range textStream {
			if err != nil {
				yield(nil, err)
				return
			}
			if !yield([]byte("audio:"+text), nil) {
				return
			}
		}
	}
}

func TestRegistry_RegisterAndNew(t *testing.T) {
	// Register a mock provider.
	Register("mock-tts", func(cfg Config) (TTS, error) {
		return &mockTTS{}, nil
	})

	// Create a TTS engine using the registered provider.
	engine, err := New("mock-tts", Config{Voice: "rachel"})
	require.NoError(t, err)
	require.NotNil(t, engine)

	// Verify it works.
	audio, err := engine.Synthesize(context.Background(), "hello")
	require.NoError(t, err)
	assert.Equal(t, []byte("audio:hello"), audio)
}

func TestRegistry_UnknownProvider(t *testing.T) {
	_, err := New("nonexistent-tts-provider", Config{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown provider")
}

func TestRegistry_PanicOnEmptyName(t *testing.T) {
	assert.Panics(t, func() {
		Register("", func(cfg Config) (TTS, error) {
			return &mockTTS{}, nil
		})
	})
}

func TestRegistry_PanicOnNilFactory(t *testing.T) {
	assert.Panics(t, func() {
		Register("test-tts-nil-factory", nil)
	})
}

func TestRegistry_PanicOnDuplicate(t *testing.T) {
	Register("test-tts-dup-check", func(cfg Config) (TTS, error) {
		return &mockTTS{}, nil
	})
	assert.Panics(t, func() {
		Register("test-tts-dup-check", func(cfg Config) (TTS, error) {
			return &mockTTS{}, nil
		})
	})
}

func TestList(t *testing.T) {
	// Register a test provider.
	Register("test-tts-list", func(cfg Config) (TTS, error) {
		return &mockTTS{}, nil
	})

	names := List()
	require.NotEmpty(t, names)

	// Verify the list is sorted and contains our provider.
	assert.Contains(t, names, "test-tts-list")
	for i := 1; i < len(names); i++ {
		assert.LessOrEqual(t, names[i-1], names[i], "list should be sorted")
	}
}

func TestApplyOptions(t *testing.T) {
	cfg := ApplyOptions(
		WithVoice("rachel"),
		WithModel("eleven_turbo_v2"),
		WithSampleRate(24000),
		WithFormat(FormatOpus),
		WithSpeed(1.2),
		WithPitch(5.0),
	)

	assert.Equal(t, "rachel", cfg.Voice)
	assert.Equal(t, "eleven_turbo_v2", cfg.Model)
	assert.Equal(t, 24000, cfg.SampleRate)
	assert.Equal(t, FormatOpus, cfg.Format)
	assert.Equal(t, 1.2, cfg.Speed)
	assert.Equal(t, 5.0, cfg.Pitch)
}

func TestAudioFormats(t *testing.T) {
	tests := []struct {
		name   string
		format AudioFormat
	}{
		{"PCM", FormatPCM},
		{"Opus", FormatOpus},
		{"MP3", FormatMP3},
		{"WAV", FormatWAV},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := ApplyOptions(WithFormat(tt.format))
			assert.Equal(t, tt.format, cfg.Format)
		})
	}
}

func TestComposeHooks(t *testing.T) {
	var callOrder []string

	hooks1 := Hooks{
		BeforeSynthesize: func(ctx context.Context, text string) {
			callOrder = append(callOrder, "hooks1-before:"+text)
		},
		OnAudioChunk: func(ctx context.Context, chunk []byte) {
			callOrder = append(callOrder, "hooks1-chunk")
		},
	}

	hooks2 := Hooks{
		BeforeSynthesize: func(ctx context.Context, text string) {
			callOrder = append(callOrder, "hooks2-before:"+text)
		},
		OnAudioChunk: func(ctx context.Context, chunk []byte) {
			callOrder = append(callOrder, "hooks2-chunk")
		},
	}

	composed := ComposeHooks(hooks1, hooks2)

	// Call composed hooks.
	ctx := context.Background()
	composed.BeforeSynthesize(ctx, "test")
	composed.OnAudioChunk(ctx, []byte{0x01})

	expected := []string{
		"hooks1-before:test",
		"hooks2-before:test",
		"hooks1-chunk",
		"hooks2-chunk",
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

func TestMockTTS_Synthesize(t *testing.T) {
	mock := &mockTTS{
		synthesizeFunc: func(ctx context.Context, text string, opts ...Option) ([]byte, error) {
			cfg := ApplyOptions(opts...)
			if cfg.Voice == "spanish" {
				return []byte("es-audio:" + text), nil
			}
			return []byte("en-audio:" + text), nil
		},
	}

	// Test with default voice.
	audio, err := mock.Synthesize(context.Background(), "hello")
	require.NoError(t, err)
	assert.Equal(t, []byte("en-audio:hello"), audio)

	// Test with Spanish voice option.
	audio, err = mock.Synthesize(context.Background(), "hola", WithVoice("spanish"))
	require.NoError(t, err)
	assert.Equal(t, []byte("es-audio:hola"), audio)
}

func TestMockTTS_SynthesizeStream(t *testing.T) {
	mock := &mockTTS{
		synthesizeStreamFunc: func(ctx context.Context, textStream iter.Seq2[string, error], opts ...Option) iter.Seq2[[]byte, error] {
			return func(yield func([]byte, error) bool) {
				// Consume the text stream and emit audio chunks.
				for text, err := range textStream {
					if err != nil {
						yield(nil, err)
						return
					}
					if !yield([]byte("chunk:"+text), nil) {
						return
					}
				}
			}
		},
	}

	// Create a test text stream.
	textStream := func(yield func(string, error) bool) {
		yield("hello", nil)
		yield("world", nil)
	}

	var chunks [][]byte
	for chunk, err := range mock.SynthesizeStream(context.Background(), textStream) {
		require.NoError(t, err)
		chunks = append(chunks, chunk)
	}

	require.Len(t, chunks, 2)
	assert.Equal(t, []byte("chunk:hello"), chunks[0])
	assert.Equal(t, []byte("chunk:world"), chunks[1])
}

func TestMockTTS_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mock := &mockTTS{
		synthesizeStreamFunc: func(ctx context.Context, textStream iter.Seq2[string, error], opts ...Option) iter.Seq2[[]byte, error] {
			return func(yield func([]byte, error) bool) {
				for i := 0; i < 100; i++ {
					select {
					case <-ctx.Done():
						yield(nil, ctx.Err())
						return
					default:
						if !yield([]byte("chunk"), nil) {
							return
						}
					}
				}
			}
		},
	}

	textStream := func(yield func(string, error) bool) {
		for i := 0; i < 100; i++ {
			if !yield("text", nil) {
				return
			}
		}
	}

	count := 0
	for chunk, err := range mock.SynthesizeStream(ctx, textStream) {
		count++
		if count == 5 {
			cancel()
		}
		if err != nil {
			assert.ErrorIs(t, err, context.Canceled)
			break
		}
		_ = chunk
	}

	assert.LessOrEqual(t, count, 10, "stream should stop shortly after cancellation")
}

func TestMockTTS_EmptyText(t *testing.T) {
	mock := &mockTTS{}

	audio, err := mock.Synthesize(context.Background(), "")
	require.NoError(t, err)
	assert.Equal(t, []byte("audio:"), audio)
}

func TestMockTTS_StreamError(t *testing.T) {
	mock := &mockTTS{}

	// Create a text stream that produces an error.
	textStream := func(yield func(string, error) bool) {
		yield("hello", nil)
		yield("", assert.AnError)
	}

	var chunks [][]byte
	var lastErr error
	for chunk, err := range mock.SynthesizeStream(context.Background(), textStream) {
		if err != nil {
			lastErr = err
			break
		}
		chunks = append(chunks, chunk)
	}

	require.Len(t, chunks, 1)
	assert.Equal(t, []byte("audio:hello"), chunks[0])
	assert.Error(t, lastErr)
}

// --- AsFrameProcessor tests ---

func TestAsFrameProcessor_TextToAudio(t *testing.T) {
	mock := &mockTTS{
		synthesizeFunc: func(ctx context.Context, text string, opts ...Option) ([]byte, error) {
			return []byte("audio:" + text), nil
		},
	}

	proc := AsFrameProcessor(mock, 24000)

	in := make(chan voice.Frame, 5)
	out := make(chan voice.Frame, 5)

	in <- voice.NewTextFrame("hello world")
	close(in)

	err := proc.Process(context.Background(), in, out)
	require.NoError(t, err)

	var frames []voice.Frame
	for f := range out {
		frames = append(frames, f)
	}

	require.Len(t, frames, 1)
	assert.Equal(t, voice.FrameAudio, frames[0].Type)
	assert.Equal(t, []byte("audio:hello world"), frames[0].Data)
	assert.Equal(t, 24000, frames[0].Metadata["sample_rate"])
}

func TestAsFrameProcessor_NonTextPassThrough(t *testing.T) {
	mock := &mockTTS{}

	proc := AsFrameProcessor(mock, 24000)

	in := make(chan voice.Frame, 5)
	out := make(chan voice.Frame, 5)

	// Send an audio frame and a control frame — both should pass through.
	in <- voice.NewAudioFrame([]byte{0x01, 0x02}, 16000)
	in <- voice.NewControlFrame(voice.SignalStart)
	close(in)

	err := proc.Process(context.Background(), in, out)
	require.NoError(t, err)

	var frames []voice.Frame
	for f := range out {
		frames = append(frames, f)
	}

	require.Len(t, frames, 2)
	assert.Equal(t, voice.FrameAudio, frames[0].Type)
	assert.Equal(t, []byte{0x01, 0x02}, frames[0].Data)
	assert.Equal(t, voice.FrameControl, frames[1].Type)
	assert.Equal(t, voice.SignalStart, frames[1].Signal())
}

func TestAsFrameProcessor_EmptyAudio(t *testing.T) {
	mock := &mockTTS{
		synthesizeFunc: func(ctx context.Context, text string, opts ...Option) ([]byte, error) {
			return nil, nil // empty audio
		},
	}

	proc := AsFrameProcessor(mock, 24000)

	in := make(chan voice.Frame, 5)
	out := make(chan voice.Frame, 5)

	in <- voice.NewTextFrame("hello")
	close(in)

	err := proc.Process(context.Background(), in, out)
	require.NoError(t, err)

	// No output frames should be produced for empty audio.
	var frames []voice.Frame
	for f := range out {
		frames = append(frames, f)
	}

	assert.Empty(t, frames)
}

func TestAsFrameProcessor_SynthesizeError(t *testing.T) {
	mock := &mockTTS{
		synthesizeFunc: func(ctx context.Context, text string, opts ...Option) ([]byte, error) {
			return nil, errors.New("synthesis failed")
		},
	}

	proc := AsFrameProcessor(mock, 24000)

	in := make(chan voice.Frame, 5)
	out := make(chan voice.Frame, 5)

	in <- voice.NewTextFrame("hello")
	close(in)

	err := proc.Process(context.Background(), in, out)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tts: synthesize:")
	assert.Contains(t, err.Error(), "synthesis failed")
}

func TestAsFrameProcessor_ContextCancellation(t *testing.T) {
	mock := &mockTTS{}

	proc := AsFrameProcessor(mock, 24000)

	ctx, cancel := context.WithCancel(context.Background())
	in := make(chan voice.Frame) // unbuffered so select blocks
	out := make(chan voice.Frame, 5)

	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	err := proc.Process(ctx, in, out)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestAsFrameProcessor_InputClosedReturnsNil(t *testing.T) {
	mock := &mockTTS{}

	proc := AsFrameProcessor(mock, 24000)

	in := make(chan voice.Frame)
	out := make(chan voice.Frame, 5)

	// Close input immediately.
	close(in)

	err := proc.Process(context.Background(), in, out)
	require.NoError(t, err)
}

func TestAsFrameProcessor_MultipleTextFrames(t *testing.T) {
	mock := &mockTTS{
		synthesizeFunc: func(ctx context.Context, text string, opts ...Option) ([]byte, error) {
			return []byte("synth:" + text), nil
		},
	}

	proc := AsFrameProcessor(mock, 16000)

	in := make(chan voice.Frame, 5)
	out := make(chan voice.Frame, 10)

	in <- voice.NewTextFrame("hello")
	in <- voice.NewTextFrame("world")
	in <- voice.NewTextFrame("test")
	close(in)

	err := proc.Process(context.Background(), in, out)
	require.NoError(t, err)

	var frames []voice.Frame
	for f := range out {
		frames = append(frames, f)
	}

	require.Len(t, frames, 3)
	assert.Equal(t, []byte("synth:hello"), frames[0].Data)
	assert.Equal(t, []byte("synth:world"), frames[1].Data)
	assert.Equal(t, []byte("synth:test"), frames[2].Data)
	// Verify sample rate is passed through.
	assert.Equal(t, 16000, frames[0].Metadata["sample_rate"])
}

func TestAsFrameProcessor_WithOptions(t *testing.T) {
	var capturedOpts []Option
	mock := &mockTTS{
		synthesizeFunc: func(ctx context.Context, text string, opts ...Option) ([]byte, error) {
			capturedOpts = opts
			cfg := ApplyOptions(opts...)
			return []byte("voice:" + cfg.Voice), nil
		},
	}

	proc := AsFrameProcessor(mock, 24000, WithVoice("rachel"))

	in := make(chan voice.Frame, 5)
	out := make(chan voice.Frame, 5)

	in <- voice.NewTextFrame("hello")
	close(in)

	err := proc.Process(context.Background(), in, out)
	require.NoError(t, err)

	require.NotEmpty(t, capturedOpts)

	var frames []voice.Frame
	for f := range out {
		frames = append(frames, f)
	}

	require.Len(t, frames, 1)
	assert.Equal(t, []byte("voice:rachel"), frames[0].Data)
}

func TestAsFrameProcessor_EmptyByteSliceAudio(t *testing.T) {
	mock := &mockTTS{
		synthesizeFunc: func(ctx context.Context, text string, opts ...Option) ([]byte, error) {
			return []byte{}, nil // empty byte slice (not nil)
		},
	}

	proc := AsFrameProcessor(mock, 24000)

	in := make(chan voice.Frame, 5)
	out := make(chan voice.Frame, 5)

	in <- voice.NewTextFrame("hello")
	close(in)

	err := proc.Process(context.Background(), in, out)
	require.NoError(t, err)

	// Empty byte slice has len 0, so no output frame.
	var frames []voice.Frame
	for f := range out {
		frames = append(frames, f)
	}

	assert.Empty(t, frames)
}
