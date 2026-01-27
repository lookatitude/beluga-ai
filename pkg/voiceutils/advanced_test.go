package voiceutils

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
)

// =============================================================================
// BufferPool Tests
// =============================================================================

func TestBufferPool_NewBufferPool(t *testing.T) {
	pool := NewBufferPool()
	require.NotNil(t, pool)
	require.NotNil(t, pool.pools)
}

func TestBufferPool_Get(t *testing.T) {
	tests := []struct {
		name string
		size int
	}{
		{"small buffer", 512},
		{"medium buffer", 4096},
		{"large buffer", 16384},
		{"non-standard size", 1000},
	}

	pool := NewBufferPool()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := pool.Get(tt.size)
			require.NotNil(t, buf)
			assert.Equal(t, 0, len(buf))
			assert.GreaterOrEqual(t, cap(buf), tt.size)
		})
	}
}

func TestBufferPool_GetExact(t *testing.T) {
	pool := NewBufferPool()

	tests := []struct {
		name string
		size int
	}{
		{"exact 512", 512},
		{"exact 1024", 1024},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := pool.GetExact(tt.size)
			require.NotNil(t, buf)
			assert.Equal(t, tt.size, len(buf))
		})
	}
}

func TestBufferPool_Put(t *testing.T) {
	pool := NewBufferPool()

	// Get a buffer
	buf := pool.Get(4096)
	require.NotNil(t, buf)

	// Modify it
	buf = append(buf, 1, 2, 3, 4)

	// Put it back
	pool.Put(buf)

	// Get another buffer - should be the same one (reset)
	buf2 := pool.Get(4096)
	assert.Equal(t, 0, len(buf2))
}

func TestBufferPool_Put_Nil(t *testing.T) {
	pool := NewBufferPool()

	// Should not panic on nil
	pool.Put(nil)

	// Should not panic on empty slice
	pool.Put([]byte{})
}

func TestBufferPool_Concurrent(t *testing.T) {
	pool := NewBufferPool()
	var wg sync.WaitGroup
	iterations := 100

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				buf := pool.Get(4096)
				buf = append(buf, byte(j))
				pool.Put(buf)
			}
		}()
	}

	wg.Wait()
}

func TestBufferPool_Global(t *testing.T) {
	pool1 := GetGlobalBufferPool()
	pool2 := GetGlobalBufferPool()

	assert.Same(t, pool1, pool2, "Global buffer pool should be singleton")
}

// =============================================================================
// Config Tests
// =============================================================================

func TestConfig_SetDefaults(t *testing.T) {
	cfg := &Config{}
	cfg.SetDefaults()

	require.NotNil(t, cfg.Audio)
	assert.Equal(t, 16000, cfg.Audio.SampleRate)
	assert.Equal(t, 1, cfg.Audio.Channels)
	assert.Equal(t, 16, cfg.Audio.BitDepth)
	assert.Equal(t, "pcm", cfg.Audio.Encoding)

	require.NotNil(t, cfg.BufferPool)
	assert.True(t, cfg.BufferPool.Enabled)

	require.NotNil(t, cfg.Retry)
	assert.Equal(t, 3, cfg.Retry.MaxRetries)

	require.NotNil(t, cfg.RateLimit)
	assert.True(t, cfg.RateLimit.Enabled)

	require.NotNil(t, cfg.CircuitBreaker)
	assert.True(t, cfg.CircuitBreaker.Enabled)
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name:    "valid default config",
			cfg:     DefaultConfig(),
			wantErr: false,
		},
		{
			name: "invalid sample rate",
			cfg: func() *Config {
				c := DefaultConfig()
				c.Audio.SampleRate = 1000 // Too low
				return c
			}(),
			wantErr: true,
		},
		{
			name: "invalid channels",
			cfg: func() *Config {
				c := DefaultConfig()
				c.Audio.Channels = 5 // Too high
				return c
			}(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	require.NotNil(t, cfg)
	require.NotNil(t, cfg.Audio)
	require.NotNil(t, cfg.BufferPool)
	require.NotNil(t, cfg.Retry)
	require.NotNil(t, cfg.RateLimit)
	require.NotNil(t, cfg.CircuitBreaker)
}

// =============================================================================
// Error Tests
// =============================================================================

func TestVoiceError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *VoiceError
		contains string
	}{
		{
			name: "with message",
			err: &VoiceError{
				Op:      "test_op",
				Code:    ErrorCodeInvalidInput,
				Message: "test message",
			},
			contains: "test message",
		},
		{
			name: "with underlying error",
			err: &VoiceError{
				Op:   "test_op",
				Code: ErrorCodeTimeout,
				Err:  ErrTimeout,
			},
			contains: "timed out",
		},
		{
			name: "empty error",
			err: &VoiceError{
				Op:   "test_op",
				Code: ErrorCodeInternalError,
			},
			contains: "unknown error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errStr := tt.err.Error()
			assert.Contains(t, errStr, tt.contains)
			assert.Contains(t, errStr, "voiceutils")
			assert.Contains(t, errStr, "test_op")
		})
	}
}

func TestVoiceError_Unwrap(t *testing.T) {
	inner := ErrTimeout
	err := NewTimeoutError("test", "timeout occurred", inner)

	unwrapped := err.Unwrap()
	assert.Equal(t, inner, unwrapped)
}

func TestVoiceError_AddContext(t *testing.T) {
	err := NewVoiceError("test", ErrorCodeInvalidInput, "test", nil)
	err.AddContext("key1", "value1")
	err.AddContext("key2", 42)

	assert.Equal(t, "value1", err.Context["key1"])
	assert.Equal(t, 42, err.Context["key2"])
}

func TestNewVoiceError_Variants(t *testing.T) {
	tests := []struct {
		name      string
		createErr func() *VoiceError
		wantCode  ErrorCode
	}{
		{
			name:      "invalid input",
			createErr: func() *VoiceError { return NewInvalidInputError("op", "msg", nil) },
			wantCode:  ErrorCodeInvalidInput,
		},
		{
			name:      "invalid format",
			createErr: func() *VoiceError { return NewInvalidFormatError("op", "msg", nil) },
			wantCode:  ErrorCodeInvalidFormat,
		},
		{
			name:      "unsupported codec",
			createErr: func() *VoiceError { return NewUnsupportedCodecError("op", "opus") },
			wantCode:  ErrorCodeUnsupportedCodec,
		},
		{
			name:      "connection error",
			createErr: func() *VoiceError { return NewConnectionError("op", "msg", nil) },
			wantCode:  ErrorCodeConnectionFailed,
		},
		{
			name:      "timeout",
			createErr: func() *VoiceError { return NewTimeoutError("op", "msg", nil) },
			wantCode:  ErrorCodeTimeout,
		},
		{
			name:      "stream closed",
			createErr: func() *VoiceError { return NewStreamClosedError("op", "msg") },
			wantCode:  ErrorCodeStreamClosed,
		},
		{
			name:      "buffer overflow",
			createErr: func() *VoiceError { return NewBufferOverflowError("op", 100, 200) },
			wantCode:  ErrorCodeBufferOverflow,
		},
		{
			name:      "rate limit",
			createErr: func() *VoiceError { return NewRateLimitError("op", "5s") },
			wantCode:  ErrorCodeRateLimited,
		},
		{
			name:      "circuit open",
			createErr: func() *VoiceError { return NewCircuitOpenError("op") },
			wantCode:  ErrorCodeCircuitOpen,
		},
		{
			name:      "provider error",
			createErr: func() *VoiceError { return NewProviderError("op", "deepgram", "msg", nil) },
			wantCode:  ErrorCodeProviderError,
		},
		{
			name:      "internal error",
			createErr: func() *VoiceError { return NewInternalError("op", "msg", nil) },
			wantCode:  ErrorCodeInternalError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.createErr()
			assert.Equal(t, tt.wantCode, err.Code)
		})
	}
}

func TestIsVoiceError(t *testing.T) {
	voiceErr := NewInvalidInputError("op", "msg", nil)
	assert.True(t, IsVoiceError(voiceErr))

	regularErr := ErrTimeout
	assert.False(t, IsVoiceError(regularErr))
}

func TestAsVoiceError(t *testing.T) {
	voiceErr := NewInvalidInputError("op", "msg", nil)
	result, ok := AsVoiceError(voiceErr)
	assert.True(t, ok)
	assert.Equal(t, voiceErr, result)

	regularErr := ErrTimeout
	result, ok = AsVoiceError(regularErr)
	assert.False(t, ok)
	assert.Nil(t, result)
}

func TestGetErrorCode(t *testing.T) {
	voiceErr := NewInvalidInputError("op", "msg", nil)
	code, ok := GetErrorCode(voiceErr)
	assert.True(t, ok)
	assert.Equal(t, ErrorCodeInvalidInput, code)

	regularErr := ErrTimeout
	code, ok = GetErrorCode(regularErr)
	assert.False(t, ok)
	assert.Equal(t, ErrorCode(""), code)
}

func TestIsErrorCode(t *testing.T) {
	voiceErr := NewInvalidInputError("op", "msg", nil)
	assert.True(t, IsErrorCode(voiceErr, ErrorCodeInvalidInput))
	assert.False(t, IsErrorCode(voiceErr, ErrorCodeTimeout))

	regularErr := ErrTimeout
	assert.False(t, IsErrorCode(regularErr, ErrorCodeTimeout))
}

// =============================================================================
// Metrics Tests
// =============================================================================

func TestMetrics_NewMetrics(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	tracer := trace.NewNoopTracerProvider().Tracer("test")

	m, err := NewMetrics(meter, tracer)
	require.NoError(t, err)
	require.NotNil(t, m)
}

func TestMetrics_RecordAudioProcessed(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	tracer := trace.NewNoopTracerProvider().Tracer("test")

	m, err := NewMetrics(meter, tracer)
	require.NoError(t, err)

	// Should not panic
	m.RecordAudioProcessed(context.Background(), "transcribe", 100*time.Millisecond, 1000)
}

func TestMetrics_RecordAudioConversion(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	tracer := trace.NewNoopTracerProvider().Tracer("test")

	m, err := NewMetrics(meter, tracer)
	require.NoError(t, err)

	// Should not panic
	m.RecordAudioConversion(context.Background(), "pcm", "opus", 10*time.Millisecond)
}

func TestMetrics_RecordBuffer(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	tracer := trace.NewNoopTracerProvider().Tracer("test")

	m, err := NewMetrics(meter, tracer)
	require.NoError(t, err)

	ctx := context.Background()

	// Should not panic
	m.RecordBufferAcquired(ctx, 4096, true)
	m.RecordBufferAcquired(ctx, 1024, false)
	m.RecordBufferReleased(ctx)
}

func TestMetrics_RecordResilience(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	tracer := trace.NewNoopTracerProvider().Tracer("test")

	m, err := NewMetrics(meter, tracer)
	require.NoError(t, err)

	ctx := context.Background()

	// Should not panic
	m.RecordRetryAttempt(ctx, "transcribe", 1)
	m.RecordCircuitStateChange(ctx, "stt", "closed", "open")
	m.RecordRateLimitHit(ctx, "api")
	m.RecordError(ctx, "transcribe", "timeout")
}

func TestMetrics_NilSafe(t *testing.T) {
	var m *Metrics

	ctx := context.Background()

	// All methods should be nil-safe
	m.RecordAudioProcessed(ctx, "op", time.Second, 100)
	m.RecordAudioConversion(ctx, "a", "b", time.Second)
	m.RecordBufferAcquired(ctx, 1024, true)
	m.RecordBufferReleased(ctx)
	m.RecordRetryAttempt(ctx, "op", 1)
	m.RecordCircuitStateChange(ctx, "n", "a", "b")
	m.RecordRateLimitHit(ctx, "l")
	m.RecordError(ctx, "op", "err")
}

func TestNoOpMetrics(t *testing.T) {
	m := NoOpMetrics()
	require.NotNil(t, m)
	require.NotNil(t, m.tracer)
}

// =============================================================================
// Test Utilities Tests
// =============================================================================

func TestMockHTTPRoundTripper(t *testing.T) {
	tests := []struct {
		name       string
		setup      func() *MockHTTPRoundTripper
		wantStatus int
		wantErr    bool
	}{
		{
			name: "default response",
			setup: func() *MockHTTPRoundTripper {
				return &MockHTTPRoundTripper{}
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "custom response",
			setup: func() *MockHTTPRoundTripper {
				return &MockHTTPRoundTripper{
					Response: NewSuccessResponse(http.StatusCreated, "created"),
				}
			},
			wantStatus: http.StatusCreated,
		},
		{
			name: "error response",
			setup: func() *MockHTTPRoundTripper {
				return &MockHTTPRoundTripper{
					Error: ErrConnectionFailed,
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rt := tt.setup()
			req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
			resp, err := rt.RoundTrip(req)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantStatus, resp.StatusCode)
			}
		})
	}
}

func TestNewJSONResponse(t *testing.T) {
	resp := NewJSONResponse(http.StatusOK, `{"key": "value"}`)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
}

func TestGenerateTestAudioData(t *testing.T) {
	samples := 1000
	sampleRate := 16000

	data := GenerateTestAudioData(samples, sampleRate)
	assert.Equal(t, samples*2, len(data)) // 16-bit = 2 bytes per sample
}

func TestGenerateSilentAudioData(t *testing.T) {
	samples := 1000

	data := GenerateSilentAudioData(samples)
	assert.Equal(t, samples*2, len(data))

	// All bytes should be zero
	for _, b := range data {
		assert.Equal(t, byte(0), b)
	}
}

func TestMockSTTProvider(t *testing.T) {
	provider := &MockSTTProvider{
		TranscribeFunc: func(audio []byte) (string, error) {
			return "custom transcription", nil
		},
	}

	result, err := provider.Transcribe([]byte{1, 2, 3})
	require.NoError(t, err)
	assert.Equal(t, "custom transcription", result)
	assert.Equal(t, 1, provider.TranscribeCallCount)
}

func TestMockTTSProvider(t *testing.T) {
	provider := &MockTTSProvider{}

	audio, err := provider.Synthesize("hello")
	require.NoError(t, err)
	assert.NotEmpty(t, audio)
	assert.Equal(t, 1, provider.SynthesizeCallCount)
}

func TestMockVADProvider(t *testing.T) {
	provider := &MockVADProvider{
		DetectFunc: func(audio []byte) (bool, error) {
			return false, nil
		},
	}

	detected, err := provider.Detect([]byte{1, 2, 3})
	require.NoError(t, err)
	assert.False(t, detected)
	assert.Equal(t, 1, provider.DetectCallCount)
}

func TestMockStreamingSession(t *testing.T) {
	session := &MockStreamingSessionImpl{}

	// Test SendAudio
	err := session.SendAudio([]byte{1, 2, 3})
	assert.NoError(t, err)

	// Test ReceiveTranscript
	ch := session.ReceiveTranscript()
	assert.NotNil(t, ch)

	// Test SendMockTranscript
	go session.SendMockTranscript("hello")
	select {
	case text := <-ch:
		assert.Equal(t, "hello", text)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for transcript")
	}

	// Test Close
	err = session.Close()
	assert.NoError(t, err)
}

// =============================================================================
// Options Tests
// =============================================================================

func TestOptions(t *testing.T) {
	cfg := defaultOptionConfig()

	WithTimeout(5 * time.Second)(cfg)
	assert.Equal(t, 5*time.Second, cfg.timeout)

	WithMaxRetries(5)(cfg)
	assert.Equal(t, 5, cfg.maxRetries)

	WithBufferSize(8192)(cfg)
	assert.Equal(t, 8192, cfg.bufferSize)

	WithSampleRate(48000)(cfg)
	assert.Equal(t, 48000, cfg.sampleRate)

	WithChannels(2)(cfg)
	assert.Equal(t, 2, cfg.channels)

	WithBitDepth(24)(cfg)
	assert.Equal(t, 24, cfg.bitDepth)

	WithEncoding("opus")(cfg)
	assert.Equal(t, "opus", cfg.encoding)

	WithMetrics(false)(cfg)
	assert.False(t, cfg.enableMetrics)
}

func TestDefaultOptionConfig(t *testing.T) {
	cfg := defaultOptionConfig()

	assert.Equal(t, 30*time.Second, cfg.timeout)
	assert.Equal(t, 3, cfg.maxRetries)
	assert.Equal(t, 4096, cfg.bufferSize)
	assert.Equal(t, 16000, cfg.sampleRate)
	assert.Equal(t, 1, cfg.channels)
	assert.Equal(t, 16, cfg.bitDepth)
	assert.Equal(t, "pcm", cfg.encoding)
	assert.True(t, cfg.enableMetrics)
}
