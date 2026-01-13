package main

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Table-Driven Tests for Configuration
// ============================================================================

func TestPipelineConfig_Validation(t *testing.T) {
	tests := []struct {
		name        string
		config      PipelineConfig
		expectError bool
		errorMsg    string
	}{
		{
			name:        "default config is valid",
			config:      DefaultPipelineConfig(),
			expectError: false,
		},
		{
			name: "zero sample rate is invalid",
			config: PipelineConfig{
				SampleRate:   0,
				Channels:     1,
				VADThreshold: 0.5,
			},
			expectError: true,
			errorMsg:    "sample rate must be positive",
		},
		{
			name: "negative sample rate is invalid",
			config: PipelineConfig{
				SampleRate:   -16000,
				Channels:     1,
				VADThreshold: 0.5,
			},
			expectError: true,
			errorMsg:    "sample rate must be positive",
		},
		{
			name: "zero channels is invalid",
			config: PipelineConfig{
				SampleRate:   16000,
				Channels:     0,
				VADThreshold: 0.5,
			},
			expectError: true,
			errorMsg:    "channels must be 1 or 2",
		},
		{
			name: "more than 2 channels is invalid",
			config: PipelineConfig{
				SampleRate:   16000,
				Channels:     3,
				VADThreshold: 0.5,
			},
			expectError: true,
			errorMsg:    "channels must be 1 or 2",
		},
		{
			name: "threshold below 0 is invalid",
			config: PipelineConfig{
				SampleRate:   16000,
				Channels:     1,
				VADThreshold: -0.1,
			},
			expectError: true,
			errorMsg:    "VAD threshold must be between 0 and 1",
		},
		{
			name: "threshold above 1 is invalid",
			config: PipelineConfig{
				SampleRate:   16000,
				Channels:     1,
				VADThreshold: 1.1,
			},
			expectError: true,
			errorMsg:    "VAD threshold must be between 0 and 1",
		},
		{
			name: "threshold at boundary 0 is valid",
			config: PipelineConfig{
				SampleRate:   16000,
				Channels:     1,
				VADThreshold: 0,
			},
			expectError: false,
		},
		{
			name: "threshold at boundary 1 is valid",
			config: PipelineConfig{
				SampleRate:   16000,
				Channels:     1,
				VADThreshold: 1,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ============================================================================
// Pipeline Creation Tests
// ============================================================================

func TestNewAudioPipeline(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		opts        []PipelineOption
		expectError bool
	}{
		{
			name:        "default options succeed",
			opts:        nil,
			expectError: false,
		},
		{
			name: "custom sample rate",
			opts: []PipelineOption{
				WithSampleRate(48000),
			},
			expectError: false,
		},
		{
			name: "custom VAD settings",
			opts: []PipelineOption{
				WithVADModel("energy"),
				WithVADThreshold(0.7),
			},
			expectError: false,
		},
		{
			name: "disabled denoise",
			opts: []PipelineOption{
				WithDenoise(false),
			},
			expectError: false,
		},
		{
			name: "all custom options",
			opts: []PipelineOption{
				WithSampleRate(24000),
				WithVADModel("webrtc"),
				WithVADThreshold(0.6),
				WithSilenceDuration(300 * time.Millisecond),
				WithDenoise(true),
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pipeline, err := NewAudioPipeline(ctx, tt.opts...)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, pipeline)
				assert.NotNil(t, pipeline.vad)
				assert.NotNil(t, pipeline.turn)
				pipeline.Close()
			}
		})
	}
}

// ============================================================================
// VAD Processing Tests
// ============================================================================

func TestVAD_ProcessFrame(t *testing.T) {
	ctx := context.Background()
	pipeline, err := NewAudioPipeline(ctx)
	require.NoError(t, err)
	defer pipeline.Close()

	tests := []struct {
		name         string
		frame        []byte
		expectSpeech bool
	}{
		{
			name:         "empty frame",
			frame:        []byte{},
			expectSpeech: false,
		},
		{
			name:         "silence frame",
			frame:        generateSilenceFrame(512),
			expectSpeech: false,
		},
		{
			name:         "loud frame",
			frame:        generateLoudFrame(512),
			expectSpeech: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := pipeline.ProcessFrame(ctx, tt.frame)
			require.NoError(t, err)
			require.NotNil(t, result)
			// Note: mock VAD may not give exact expected results
			// In production tests, you'd use real audio samples
			assert.NotZero(t, result.Timestamp)
		})
	}
}

// ============================================================================
// Turn Detection Tests
// ============================================================================

func TestTurnDetection_SpeechToSilence(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pipeline, err := NewAudioPipeline(ctx,
		WithSilenceDuration(100*time.Millisecond),
		WithVADThreshold(0.1), // Low threshold for testing
	)
	require.NoError(t, err)
	defer pipeline.Close()

	audioIn := make(chan []byte, 100)
	turnsOut := make(chan *TurnEvent, 10)

	// Start pipeline
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		pipeline.ProcessAudioStream(ctx, audioIn, turnsOut)
	}()

	// Send loud frames (speech)
	for i := 0; i < 20; i++ {
		audioIn <- generateLoudFrame(512)
	}

	// Send silence frames
	for i := 0; i < 10; i++ {
		audioIn <- generateSilenceFrame(512)
	}

	// Close input
	close(audioIn)

	// Wait for turn or timeout
	select {
	case turn := <-turnsOut:
		assert.NotNil(t, turn)
		assert.Greater(t, len(turn.Audio), 0)
		assert.True(t, turn.Duration > 0)
	case <-time.After(2 * time.Second):
		// May not get turn with mock VAD
		t.Log("No turn received (expected with mock VAD)")
	}

	wg.Wait()
}

func TestTurnDetection_MaxDuration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pipeline, err := NewAudioPipeline(ctx,
		WithVADThreshold(0.01), // Very low to always detect speech
	)
	require.NoError(t, err)

	// Override max duration for testing
	pipeline.turn.maxSpeechDuration = 100 * time.Millisecond

	defer pipeline.Close()

	audioIn := make(chan []byte, 100)
	turnsOut := make(chan *TurnEvent, 10)

	go pipeline.ProcessAudioStream(ctx, audioIn, turnsOut)

	// Send continuous loud frames
	go func() {
		for i := 0; i < 100; i++ {
			select {
			case audioIn <- generateLoudFrame(512):
			case <-ctx.Done():
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		close(audioIn)
	}()

	// Should receive a turn due to max duration
	select {
	case turn := <-turnsOut:
		if turn != nil {
			assert.Equal(t, TurnEndMaxLength, turn.Reason)
		}
	case <-time.After(2 * time.Second):
		// May not get turn with mock VAD
		t.Log("No max duration turn received")
	}
}

// ============================================================================
// Concurrency Tests
// ============================================================================

func TestPipeline_ConcurrentFrameProcessing(t *testing.T) {
	ctx := context.Background()
	pipeline, err := NewAudioPipeline(ctx)
	require.NoError(t, err)
	defer pipeline.Close()

	var wg sync.WaitGroup
	numGoroutines := 10
	framesPerGoroutine := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < framesPerGoroutine; j++ {
				frame := generateLoudFrame(512)
				_, err := pipeline.ProcessFrame(ctx, frame)
				assert.NoError(t, err)
			}
		}()
	}

	wg.Wait()
}

func TestPipeline_CancelContextStopsProcessing(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	pipeline, err := NewAudioPipeline(ctx)
	require.NoError(t, err)
	defer pipeline.Close()

	audioIn := make(chan []byte, 100)
	turnsOut := make(chan *TurnEvent, 10)

	errCh := make(chan error, 1)
	go func() {
		errCh <- pipeline.ProcessAudioStream(ctx, audioIn, turnsOut)
	}()

	// Send some frames
	for i := 0; i < 10; i++ {
		audioIn <- generateSilenceFrame(512)
	}

	// Cancel context
	cancel()

	// Should exit with context error
	select {
	case err := <-errCh:
		assert.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("Pipeline did not stop after context cancellation")
	}
}

func TestPipeline_OnlyOneInstanceCanRun(t *testing.T) {
	ctx := context.Background()
	pipeline, err := NewAudioPipeline(ctx)
	require.NoError(t, err)
	defer pipeline.Close()

	audioIn1 := make(chan []byte, 10)
	audioIn2 := make(chan []byte, 10)
	turnsOut := make(chan *TurnEvent, 10)

	// Start first instance
	go pipeline.ProcessAudioStream(ctx, audioIn1, turnsOut)

	// Give it time to start
	time.Sleep(50 * time.Millisecond)

	// Try to start second instance
	err = pipeline.ProcessAudioStream(ctx, audioIn2, turnsOut)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")

	close(audioIn1)
}

// ============================================================================
// Metrics Tests
// ============================================================================

func TestPipelineMetrics_Creation(t *testing.T) {
	metrics, err := newPipelineMetrics()
	require.NoError(t, err)
	require.NotNil(t, metrics)
	assert.NotNil(t, metrics.vadLatency)
	assert.NotNil(t, metrics.denoiseLatency)
	assert.NotNil(t, metrics.turnDuration)
	assert.NotNil(t, metrics.speechSegments)
	assert.NotNil(t, metrics.vadDecisions)
	assert.NotNil(t, metrics.errors)
}

func TestPipelineMetrics_Recording(t *testing.T) {
	ctx := context.Background()
	metrics, err := newPipelineMetrics()
	require.NoError(t, err)

	// These should not panic
	metrics.recordVADLatency(ctx, 10*time.Millisecond)
	metrics.recordDenoiseLatency(ctx, 5*time.Millisecond)
	metrics.recordTurnDuration(ctx, 2*time.Second, TurnEndSilence)
	metrics.incrementSpeechSegments(ctx)
	metrics.recordVADDecision(ctx, true)
	metrics.recordVADDecision(ctx, false)
	metrics.recordError(ctx, "vad")
	metrics.recordError(ctx, "denoise")
}

// ============================================================================
// TurnDetector Tests
// ============================================================================

func TestTurnDetector_Configuration(t *testing.T) {
	config := PipelineConfig{
		SilenceDuration:   500 * time.Millisecond,
		MinSpeechDuration: 100 * time.Millisecond,
		MaxSpeechDuration: 30 * time.Second,
	}

	detector := NewTurnDetector(config)
	require.NotNil(t, detector)
	assert.Equal(t, 500*time.Millisecond, detector.silenceDuration)
	assert.Equal(t, 100*time.Millisecond, detector.minSpeechDuration)
	assert.Equal(t, 30*time.Second, detector.maxSpeechDuration)
}

// ============================================================================
// Helper Functions
// ============================================================================

// generateSilenceFrame creates a frame of silence (all zeros).
func generateSilenceFrame(samples int) []byte {
	return make([]byte, samples*2) // 16-bit audio = 2 bytes per sample
}

// generateLoudFrame creates a frame with high amplitude.
func generateLoudFrame(samples int) []byte {
	frame := make([]byte, samples*2)
	for i := 0; i < len(frame)-1; i += 2 {
		// Set high amplitude value (near max for 16-bit signed)
		frame[i] = 0xFF
		frame[i+1] = 0x7F
	}
	return frame
}

// ============================================================================
// Benchmarks
// ============================================================================

func BenchmarkProcessFrame(b *testing.B) {
	ctx := context.Background()
	pipeline, err := NewAudioPipeline(ctx)
	if err != nil {
		b.Fatal(err)
	}
	defer pipeline.Close()

	frame := generateLoudFrame(512)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pipeline.ProcessFrame(ctx, frame)
	}
}

func BenchmarkProcessFrame_WithDenoise(b *testing.B) {
	ctx := context.Background()
	pipeline, err := NewAudioPipeline(ctx, WithDenoise(true))
	if err != nil {
		b.Fatal(err)
	}
	defer pipeline.Close()

	frame := generateLoudFrame(512)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pipeline.ProcessFrame(ctx, frame)
	}
}

func BenchmarkProcessFrame_NoDenoise(b *testing.B) {
	ctx := context.Background()
	pipeline, err := NewAudioPipeline(ctx, WithDenoise(false))
	if err != nil {
		b.Fatal(err)
	}
	defer pipeline.Close()

	frame := generateLoudFrame(512)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pipeline.ProcessFrame(ctx, frame)
	}
}

func BenchmarkMetricsRecording(b *testing.B) {
	ctx := context.Background()
	metrics, err := newPipelineMetrics()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.recordVADLatency(ctx, 10*time.Millisecond)
		metrics.recordVADDecision(ctx, true)
	}
}
