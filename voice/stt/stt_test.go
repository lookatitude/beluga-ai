package stt

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

// mockSTT is a test implementation of the STT interface.
type mockSTT struct {
	transcribeFunc       func(context.Context, []byte, ...Option) (string, error)
	transcribeStreamFunc func(context.Context, iter.Seq2[[]byte, error], ...Option) iter.Seq2[TranscriptEvent, error]
}

// Compile-time interface check.
var _ STT = (*mockSTT)(nil)

func (m *mockSTT) Transcribe(ctx context.Context, audio []byte, opts ...Option) (string, error) {
	if m.transcribeFunc != nil {
		return m.transcribeFunc(ctx, audio, opts...)
	}
	return "test transcription", nil
}

func (m *mockSTT) TranscribeStream(ctx context.Context, audioStream iter.Seq2[[]byte, error], opts ...Option) iter.Seq2[TranscriptEvent, error] {
	if m.transcribeStreamFunc != nil {
		return m.transcribeStreamFunc(ctx, audioStream, opts...)
	}
	return func(yield func(TranscriptEvent, error) bool) {
		yield(TranscriptEvent{Text: "test", IsFinal: true, Confidence: 0.95}, nil)
	}
}

func TestRegistry_RegisterAndNew(t *testing.T) {
	// Register a mock provider.
	Register("mock-stt", func(cfg Config) (STT, error) {
		return &mockSTT{}, nil
	})

	// Create an STT engine using the registered provider.
	engine, err := New("mock-stt", Config{Language: "en"})
	require.NoError(t, err)
	require.NotNil(t, engine)

	// Verify it works.
	text, err := engine.Transcribe(context.Background(), []byte("audio"))
	require.NoError(t, err)
	assert.Equal(t, "test transcription", text)
}

func TestRegistry_UnknownProvider(t *testing.T) {
	_, err := New("nonexistent-stt-provider", Config{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown provider")
}

func TestRegistry_PanicOnEmptyName(t *testing.T) {
	assert.Panics(t, func() {
		Register("", func(cfg Config) (STT, error) {
			return &mockSTT{}, nil
		})
	})
}

func TestRegistry_PanicOnNilFactory(t *testing.T) {
	assert.Panics(t, func() {
		Register("test-stt-nil-factory", nil)
	})
}

func TestRegistry_PanicOnDuplicate(t *testing.T) {
	Register("test-stt-dup-check", func(cfg Config) (STT, error) {
		return &mockSTT{}, nil
	})
	assert.Panics(t, func() {
		Register("test-stt-dup-check", func(cfg Config) (STT, error) {
			return &mockSTT{}, nil
		})
	})
}

func TestList(t *testing.T) {
	// Register a test provider.
	Register("test-stt-list", func(cfg Config) (STT, error) {
		return &mockSTT{}, nil
	})

	names := List()
	require.NotEmpty(t, names)

	// Verify the list is sorted and contains our provider.
	assert.Contains(t, names, "test-stt-list")
	for i := 1; i < len(names); i++ {
		assert.LessOrEqual(t, names[i-1], names[i], "list should be sorted")
	}
}

func TestApplyOptions(t *testing.T) {
	cfg := ApplyOptions(
		WithLanguage("es"),
		WithModel("whisper-large"),
		WithPunctuation(true),
		WithDiarization(false),
		WithSampleRate(16000),
		WithEncoding("linear16"),
	)

	assert.Equal(t, "es", cfg.Language)
	assert.Equal(t, "whisper-large", cfg.Model)
	assert.True(t, cfg.Punctuation)
	assert.False(t, cfg.Diarization)
	assert.Equal(t, 16000, cfg.SampleRate)
	assert.Equal(t, "linear16", cfg.Encoding)
}

func TestComposeHooks(t *testing.T) {
	var callOrder []string

	hooks1 := Hooks{
		OnTranscript: func(ctx context.Context, event TranscriptEvent) {
			callOrder = append(callOrder, "hooks1-transcript")
		},
		OnUtterance: func(ctx context.Context, text string) {
			callOrder = append(callOrder, "hooks1-utterance")
		},
	}

	hooks2 := Hooks{
		OnTranscript: func(ctx context.Context, event TranscriptEvent) {
			callOrder = append(callOrder, "hooks2-transcript")
		},
		OnUtterance: func(ctx context.Context, text string) {
			callOrder = append(callOrder, "hooks2-utterance")
		},
	}

	composed := ComposeHooks(hooks1, hooks2)

	// Call composed hooks.
	ctx := context.Background()
	composed.OnTranscript(ctx, TranscriptEvent{Text: "test"})
	composed.OnUtterance(ctx, "test")

	expected := []string{
		"hooks1-transcript",
		"hooks2-transcript",
		"hooks1-utterance",
		"hooks2-utterance",
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

func TestTranscriptEvent(t *testing.T) {
	event := TranscriptEvent{
		Text:       "Hello, world!",
		IsFinal:    true,
		Confidence: 0.95,
		Timestamp:  2 * time.Second,
		Language:   "en",
		Words: []Word{
			{Text: "Hello", Start: 0, End: 500 * time.Millisecond, Confidence: 0.98},
			{Text: "world", Start: 600 * time.Millisecond, End: time.Second, Confidence: 0.92},
		},
	}

	assert.Equal(t, "Hello, world!", event.Text)
	assert.True(t, event.IsFinal)
	assert.Equal(t, 0.95, event.Confidence)
	assert.Equal(t, 2*time.Second, event.Timestamp)
	assert.Equal(t, "en", event.Language)
	assert.Len(t, event.Words, 2)
}

func TestMockSTT_Transcribe(t *testing.T) {
	mock := &mockSTT{
		transcribeFunc: func(ctx context.Context, audio []byte, opts ...Option) (string, error) {
			cfg := ApplyOptions(opts...)
			if cfg.Language == "es" {
				return "Hola mundo", nil
			}
			return "Hello world", nil
		},
	}

	// Test with default language.
	text, err := mock.Transcribe(context.Background(), []byte("audio"))
	require.NoError(t, err)
	assert.Equal(t, "Hello world", text)

	// Test with Spanish language option.
	text, err = mock.Transcribe(context.Background(), []byte("audio"), WithLanguage("es"))
	require.NoError(t, err)
	assert.Equal(t, "Hola mundo", text)
}

func TestMockSTT_TranscribeStream(t *testing.T) {
	mock := &mockSTT{
		transcribeStreamFunc: func(ctx context.Context, audioStream iter.Seq2[[]byte, error], opts ...Option) iter.Seq2[TranscriptEvent, error] {
			return func(yield func(TranscriptEvent, error) bool) {
				// Consume the audio stream.
				for audio, err := range audioStream {
					if err != nil {
						yield(TranscriptEvent{}, err)
						return
					}
					// Emit a transcript event for each audio chunk.
					if !yield(TranscriptEvent{
						Text:       string(audio),
						IsFinal:    false,
						Confidence: 0.8,
					}, nil) {
						return
					}
				}
				// Emit final event.
				yield(TranscriptEvent{
					Text:       "final",
					IsFinal:    true,
					Confidence: 0.95,
				}, nil)
			}
		},
	}

	// Create a test audio stream.
	audioStream := func(yield func([]byte, error) bool) {
		yield([]byte("chunk1"), nil)
		yield([]byte("chunk2"), nil)
	}

	var events []TranscriptEvent
	for event, err := range mock.TranscribeStream(context.Background(), audioStream) {
		require.NoError(t, err)
		events = append(events, event)
	}

	require.Len(t, events, 3)
	assert.Equal(t, "chunk1", events[0].Text)
	assert.False(t, events[0].IsFinal)
	assert.Equal(t, "chunk2", events[1].Text)
	assert.False(t, events[1].IsFinal)
	assert.Equal(t, "final", events[2].Text)
	assert.True(t, events[2].IsFinal)
}

func TestMockSTT_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mock := &mockSTT{
		transcribeStreamFunc: func(ctx context.Context, audioStream iter.Seq2[[]byte, error], opts ...Option) iter.Seq2[TranscriptEvent, error] {
			return func(yield func(TranscriptEvent, error) bool) {
				for i := 0; i < 100; i++ {
					select {
					case <-ctx.Done():
						yield(TranscriptEvent{}, ctx.Err())
						return
					default:
						if !yield(TranscriptEvent{Text: "chunk", IsFinal: false}, nil) {
							return
						}
					}
				}
			}
		},
	}

	audioStream := func(yield func([]byte, error) bool) {
		for i := 0; i < 100; i++ {
			if !yield([]byte("audio"), nil) {
				return
			}
		}
	}

	count := 0
	for _, err := range mock.TranscribeStream(ctx, audioStream) {
		count++
		if count == 5 {
			cancel()
		}
		if err != nil {
			assert.ErrorIs(t, err, context.Canceled)
			break
		}
	}

	assert.LessOrEqual(t, count, 10, "stream should stop shortly after cancellation")
}

// --- AsFrameProcessor tests ---

func TestAsFrameProcessor_AudioToText(t *testing.T) {
	mock := &mockSTT{
		transcribeFunc: func(ctx context.Context, audio []byte, opts ...Option) (string, error) {
			return "hello world", nil
		},
	}

	proc := AsFrameProcessor(mock)

	in := make(chan voice.Frame, 5)
	out := make(chan voice.Frame, 5)

	in <- voice.NewAudioFrame([]byte{0x01, 0x02}, 16000)
	close(in)

	err := proc.Process(context.Background(), in, out)
	require.NoError(t, err)

	var frames []voice.Frame
	for f := range out {
		frames = append(frames, f)
	}

	require.Len(t, frames, 1)
	assert.Equal(t, voice.FrameText, frames[0].Type)
	assert.Equal(t, "hello world", frames[0].Text())
}

func TestAsFrameProcessor_NonAudioPassThrough(t *testing.T) {
	mock := &mockSTT{}

	proc := AsFrameProcessor(mock)

	in := make(chan voice.Frame, 5)
	out := make(chan voice.Frame, 5)

	// Send a text frame and a control frame — both should pass through.
	in <- voice.NewTextFrame("some text")
	in <- voice.NewControlFrame(voice.SignalStart)
	close(in)

	err := proc.Process(context.Background(), in, out)
	require.NoError(t, err)

	var frames []voice.Frame
	for f := range out {
		frames = append(frames, f)
	}

	require.Len(t, frames, 2)
	assert.Equal(t, voice.FrameText, frames[0].Type)
	assert.Equal(t, "some text", frames[0].Text())
	assert.Equal(t, voice.FrameControl, frames[1].Type)
	assert.Equal(t, voice.SignalStart, frames[1].Signal())
}

func TestAsFrameProcessor_EmptyTranscription(t *testing.T) {
	mock := &mockSTT{
		transcribeFunc: func(ctx context.Context, audio []byte, opts ...Option) (string, error) {
			return "", nil // empty transcription
		},
	}

	proc := AsFrameProcessor(mock)

	in := make(chan voice.Frame, 5)
	out := make(chan voice.Frame, 5)

	in <- voice.NewAudioFrame([]byte{0x01}, 16000)
	close(in)

	err := proc.Process(context.Background(), in, out)
	require.NoError(t, err)

	// No output frames should be produced for empty transcription.
	var frames []voice.Frame
	for f := range out {
		frames = append(frames, f)
	}

	assert.Empty(t, frames)
}

func TestAsFrameProcessor_TranscribeError(t *testing.T) {
	mock := &mockSTT{
		transcribeFunc: func(ctx context.Context, audio []byte, opts ...Option) (string, error) {
			return "", errors.New("transcription failed")
		},
	}

	proc := AsFrameProcessor(mock)

	in := make(chan voice.Frame, 5)
	out := make(chan voice.Frame, 5)

	in <- voice.NewAudioFrame([]byte{0x01}, 16000)
	close(in)

	err := proc.Process(context.Background(), in, out)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stt: transcribe:")
	assert.Contains(t, err.Error(), "transcription failed")
}

func TestAsFrameProcessor_ContextCancellation(t *testing.T) {
	mock := &mockSTT{}

	proc := AsFrameProcessor(mock)

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
	mock := &mockSTT{}

	proc := AsFrameProcessor(mock)

	in := make(chan voice.Frame)
	out := make(chan voice.Frame, 5)

	// Close input immediately.
	close(in)

	err := proc.Process(context.Background(), in, out)
	require.NoError(t, err)
}

func TestAsFrameProcessor_MultipleAudioFrames(t *testing.T) {
	callCount := 0
	mock := &mockSTT{
		transcribeFunc: func(ctx context.Context, audio []byte, opts ...Option) (string, error) {
			callCount++
			return "text" + string(rune('0'+callCount)), nil
		},
	}

	proc := AsFrameProcessor(mock)

	in := make(chan voice.Frame, 5)
	out := make(chan voice.Frame, 10)

	in <- voice.NewAudioFrame([]byte{0x01}, 16000)
	in <- voice.NewAudioFrame([]byte{0x02}, 16000)
	in <- voice.NewAudioFrame([]byte{0x03}, 16000)
	close(in)

	err := proc.Process(context.Background(), in, out)
	require.NoError(t, err)

	var frames []voice.Frame
	for f := range out {
		frames = append(frames, f)
	}

	require.Len(t, frames, 3)
	assert.Equal(t, "text1", frames[0].Text())
	assert.Equal(t, "text2", frames[1].Text())
	assert.Equal(t, "text3", frames[2].Text())
}

func TestAsFrameProcessor_WithOptions(t *testing.T) {
	var capturedOpts []Option
	mock := &mockSTT{
		transcribeFunc: func(ctx context.Context, audio []byte, opts ...Option) (string, error) {
			capturedOpts = opts
			cfg := ApplyOptions(opts...)
			return "lang:" + cfg.Language, nil
		},
	}

	proc := AsFrameProcessor(mock, WithLanguage("fr"))

	in := make(chan voice.Frame, 5)
	out := make(chan voice.Frame, 5)

	in <- voice.NewAudioFrame([]byte{0x01}, 16000)
	close(in)

	err := proc.Process(context.Background(), in, out)
	require.NoError(t, err)

	require.NotEmpty(t, capturedOpts)

	var frames []voice.Frame
	for f := range out {
		frames = append(frames, f)
	}

	require.Len(t, frames, 1)
	assert.Equal(t, "lang:fr", frames[0].Text())
}
