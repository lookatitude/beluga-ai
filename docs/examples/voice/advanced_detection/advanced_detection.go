// Package advanced_detection demonstrates Voice Activity Detection (VAD),
// turn detection, and noise cancellation for production voice agents.
//
// This example shows you how to build a complete audio processing pipeline
// that can detect when users are speaking, determine when they've finished
// their turn, and clean up noisy audio for better transcription accuracy.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// ============================================================================
// Configuration Types
// ============================================================================

// PipelineConfig configures the audio processing pipeline.
// We use functional options to make configuration flexible and clear.
type PipelineConfig struct {
	// Audio format settings
	SampleRate int // Hz (typically 16000 for speech)
	Channels   int // 1 for mono (recommended), 2 for stereo
	FrameSize  int // Samples per frame (512 = 32ms at 16kHz)

	// VAD settings
	VADModel     string  // "silero", "webrtc", "energy"
	VADThreshold float64 // 0.0 to 1.0

	// Turn detection settings
	SilenceDuration   time.Duration // Silence to end turn
	MinSpeechDuration time.Duration // Minimum speech to count as turn
	MaxSpeechDuration time.Duration // Maximum turn length

	// Noise cancellation
	EnableDenoise bool
	DenoiseModel  string // "rnnoise", "spectral"
}

// DefaultPipelineConfig returns sensible defaults for speech processing.
// These values work well for most conversational AI use cases.
func DefaultPipelineConfig() PipelineConfig {
	return PipelineConfig{
		SampleRate:        16000,
		Channels:          1,
		FrameSize:         512,
		VADModel:          "silero",
		VADThreshold:      0.5,
		SilenceDuration:   500 * time.Millisecond,
		MinSpeechDuration: 100 * time.Millisecond,
		MaxSpeechDuration: 30 * time.Second,
		EnableDenoise:     true,
		DenoiseModel:      "rnnoise",
	}
}

// PipelineOption is a functional option for configuring the pipeline.
type PipelineOption func(*PipelineConfig)

// WithSampleRate sets the audio sample rate.
func WithSampleRate(rate int) PipelineOption {
	return func(c *PipelineConfig) {
		c.SampleRate = rate
	}
}

// WithVADModel sets the VAD model to use.
func WithVADModel(model string) PipelineOption {
	return func(c *PipelineConfig) {
		c.VADModel = model
	}
}

// WithVADThreshold sets the speech detection threshold.
func WithVADThreshold(threshold float64) PipelineOption {
	return func(c *PipelineConfig) {
		c.VADThreshold = threshold
	}
}

// WithSilenceDuration sets how long silence must last to end a turn.
func WithSilenceDuration(d time.Duration) PipelineOption {
	return func(c *PipelineConfig) {
		c.SilenceDuration = d
	}
}

// WithDenoise enables or disables noise cancellation.
func WithDenoise(enabled bool) PipelineOption {
	return func(c *PipelineConfig) {
		c.EnableDenoise = enabled
	}
}

// ============================================================================
// Core Types
// ============================================================================

// TurnEvent represents a complete user speech turn.
// This is emitted when the turn detector determines the user has finished speaking.
type TurnEvent struct {
	// Audio contains the raw audio data for this turn
	Audio []byte

	// StartTime when speech began
	StartTime time.Time

	// EndTime when speech ended
	EndTime time.Time

	// Duration of the speech
	Duration time.Duration

	// Reason why the turn ended
	Reason TurnEndReason
}

// TurnEndReason indicates why a turn ended.
type TurnEndReason string

const (
	// TurnEndSilence indicates the user stopped speaking (normal case)
	TurnEndSilence TurnEndReason = "silence"

	// TurnEndMaxLength indicates the turn hit the maximum duration
	TurnEndMaxLength TurnEndReason = "max_length"

	// TurnEndInterrupt indicates the turn was interrupted
	TurnEndInterrupt TurnEndReason = "interrupt"

	// TurnEndStreamClosed indicates the audio stream closed
	TurnEndStreamClosed TurnEndReason = "stream_closed"
)

// VADResult contains the result of processing a single audio frame.
type VADResult struct {
	IsSpeech    bool
	Probability float64
	Timestamp   time.Time
}

// ============================================================================
// Audio Pipeline
// ============================================================================

// AudioPipeline processes audio through VAD, denoising, and turn detection.
// It's designed to be concurrent-safe and integrates with OTEL for observability.
type AudioPipeline struct {
	config PipelineConfig

	// Components
	vad      VADDetector
	denoiser Denoiser
	turn     *TurnDetector

	// State - protected by mutex
	mu           sync.Mutex
	isSpeaking   bool
	speechBuffer []byte
	speechStart  time.Time
	silenceStart time.Time

	// Metrics
	metrics *PipelineMetrics

	// State tracking
	running atomic.Bool
}

// VADDetector interface for voice activity detection.
// Implementing this interface allows for different VAD backends.
type VADDetector interface {
	// Detect analyzes audio and returns probability of speech (0.0 to 1.0)
	Detect(ctx context.Context, audio []byte) (float64, error)

	// Threshold returns the configured threshold for speech detection
	Threshold() float64

	// Close releases resources
	Close() error
}

// Denoiser interface for noise cancellation.
type Denoiser interface {
	// Process applies noise cancellation to audio
	Process(ctx context.Context, audio []byte) ([]byte, error)

	// Close releases resources
	Close() error
}

// TurnDetector tracks speech segments and determines turn boundaries.
type TurnDetector struct {
	silenceDuration   time.Duration
	minSpeechDuration time.Duration
	maxSpeechDuration time.Duration
}

// NewTurnDetector creates a turn detector with the given configuration.
func NewTurnDetector(config PipelineConfig) *TurnDetector {
	return &TurnDetector{
		silenceDuration:   config.SilenceDuration,
		minSpeechDuration: config.MinSpeechDuration,
		maxSpeechDuration: config.MaxSpeechDuration,
	}
}

// NewAudioPipeline creates a new audio processing pipeline.
// The pipeline is configured using functional options for flexibility.
func NewAudioPipeline(ctx context.Context, opts ...PipelineOption) (*AudioPipeline, error) {
	// Start with defaults
	config := DefaultPipelineConfig()

	// Apply options
	for _, opt := range opts {
		opt(&config)
	}

	// Validate configuration
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Create VAD detector
	vad, err := createVAD(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create VAD: %w", err)
	}

	// Create denoiser if enabled
	var denoiser Denoiser
	if config.EnableDenoise {
		denoiser, err = createDenoiser(ctx, config)
		if err != nil {
			vad.Close()
			return nil, fmt.Errorf("failed to create denoiser: %w", err)
		}
	}

	// Create metrics
	metrics, err := newPipelineMetrics()
	if err != nil {
		vad.Close()
		if denoiser != nil {
			denoiser.Close()
		}
		return nil, fmt.Errorf("failed to create metrics: %w", err)
	}

	return &AudioPipeline{
		config:   config,
		vad:      vad,
		denoiser: denoiser,
		turn:     NewTurnDetector(config),
		metrics:  metrics,
	}, nil
}

func validateConfig(config PipelineConfig) error {
	if config.SampleRate <= 0 {
		return errors.New("sample rate must be positive")
	}
	if config.Channels < 1 || config.Channels > 2 {
		return errors.New("channels must be 1 or 2")
	}
	if config.VADThreshold < 0 || config.VADThreshold > 1 {
		return errors.New("VAD threshold must be between 0 and 1")
	}
	return nil
}

// ProcessFrame runs the audio pipeline on a single frame.
// It applies denoising (if enabled) and VAD detection.
func (p *AudioPipeline) ProcessFrame(ctx context.Context, frame []byte) (*VADResult, error) {
	tracer := otel.Tracer("beluga.voice.pipeline")
	ctx, span := tracer.Start(ctx, "pipeline.ProcessFrame",
		trace.WithAttributes(
			attribute.Int("frame_size", len(frame)),
			attribute.Int("sample_rate", p.config.SampleRate),
		),
	)
	defer span.End()

	start := time.Now()

	// Step 1: Denoise if enabled
	var cleanFrame []byte
	if p.denoiser != nil {
		denoiseStart := time.Now()
		var err error
		cleanFrame, err = p.denoiser.Process(ctx, frame)
		if err != nil {
			span.RecordError(err)
			p.metrics.recordError(ctx, "denoise")
			return nil, fmt.Errorf("denoise failed: %w", err)
		}
		p.metrics.recordDenoiseLatency(ctx, time.Since(denoiseStart))
	} else {
		cleanFrame = frame
	}

	// Step 2: Run VAD
	vadStart := time.Now()
	probability, err := p.vad.Detect(ctx, cleanFrame)
	if err != nil {
		span.RecordError(err)
		p.metrics.recordError(ctx, "vad")
		return nil, fmt.Errorf("VAD failed: %w", err)
	}
	p.metrics.recordVADLatency(ctx, time.Since(vadStart))

	// Determine if this is speech
	isSpeech := probability >= p.vad.Threshold()

	// Record metrics
	p.metrics.recordVADDecision(ctx, isSpeech)

	span.SetAttributes(
		attribute.Bool("is_speech", isSpeech),
		attribute.Float64("probability", probability),
		attribute.Float64("total_latency_ms", float64(time.Since(start).Milliseconds())),
	)

	return &VADResult{
		IsSpeech:    isSpeech,
		Probability: probability,
		Timestamp:   time.Now(),
	}, nil
}

// ProcessAudioStream processes a continuous stream of audio frames.
// It handles turn detection and emits TurnEvents when users complete speaking.
//
// This method is designed to run in its own goroutine. It processes frames
// from audioIn and emits completed turns to turnsOut.
func (p *AudioPipeline) ProcessAudioStream(
	ctx context.Context,
	audioIn <-chan []byte,
	turnsOut chan<- *TurnEvent,
) error {
	tracer := otel.Tracer("beluga.voice.pipeline")
	ctx, span := tracer.Start(ctx, "pipeline.ProcessAudioStream")
	defer span.End()

	// Mark as running
	if !p.running.CompareAndSwap(false, true) {
		return errors.New("pipeline already running")
	}
	defer p.running.Store(false)

	// Initialize state
	p.silenceStart = time.Now()

	log.Println("Audio pipeline started")

	for {
		select {
		case <-ctx.Done():
			// Context cancelled - flush any remaining speech
			p.flushRemainingTurn(turnsOut, TurnEndInterrupt)
			return ctx.Err()

		case frame, ok := <-audioIn:
			if !ok {
				// Channel closed - flush remaining speech and exit
				p.flushRemainingTurn(turnsOut, TurnEndStreamClosed)
				log.Println("Audio pipeline stopped: input channel closed")
				return nil
			}

			// Process the frame
			result, err := p.ProcessFrame(ctx, frame)
			if err != nil {
				// Log error but continue processing
				log.Printf("Frame processing error: %v", err)
				continue
			}

			// Handle turn detection
			turn := p.handleTurnDetection(ctx, result, frame)
			if turn != nil {
				// Non-blocking send to avoid deadlock
				select {
				case turnsOut <- turn:
				default:
					log.Println("Warning: turns channel full, dropping turn")
				}
			}
		}
	}
}

// handleTurnDetection processes a VAD result and manages turn state.
// Returns a TurnEvent if a turn has completed, nil otherwise.
func (p *AudioPipeline) handleTurnDetection(
	ctx context.Context,
	result *VADResult,
	frame []byte,
) *TurnEvent {
	p.mu.Lock()
	defer p.mu.Unlock()

	tracer := otel.Tracer("beluga.voice.pipeline")

	if result.IsSpeech {
		if !p.isSpeaking {
			// Speech just started
			_, span := tracer.Start(ctx, "turn.SpeechStarted")
			span.SetAttributes(
				attribute.Float64("probability", result.Probability),
			)
			span.End()

			p.isSpeaking = true
			p.speechStart = result.Timestamp
			p.speechBuffer = nil
			p.metrics.incrementSpeechSegments(ctx)

			log.Printf("Speech started (probability: %.2f)", result.Probability)
		}

		// Accumulate speech audio
		p.speechBuffer = append(p.speechBuffer, frame...)
		p.silenceStart = time.Now()

		// Check max duration
		speechDuration := time.Since(p.speechStart)
		if speechDuration >= p.turn.maxSpeechDuration {
			log.Printf("Turn ended: max duration reached (%v)", speechDuration)
			return p.emitTurnLocked(TurnEndMaxLength)
		}

	} else {
		// Silence detected
		if p.isSpeaking {
			// Include some trailing silence for natural boundaries
			p.speechBuffer = append(p.speechBuffer, frame...)

			// Check if silence duration exceeds threshold
			silenceDuration := time.Since(p.silenceStart)
			speechDuration := time.Since(p.speechStart)

			if silenceDuration >= p.turn.silenceDuration &&
				speechDuration >= p.turn.minSpeechDuration {
				// Turn complete!
				log.Printf("Turn ended: silence detected (duration: %v)", speechDuration)
				return p.emitTurnLocked(TurnEndSilence)
			}
		}
	}

	return nil
}

// emitTurnLocked creates a TurnEvent and resets state.
// Caller must hold the mutex.
func (p *AudioPipeline) emitTurnLocked(reason TurnEndReason) *TurnEvent {
	endTime := time.Now()
	duration := endTime.Sub(p.speechStart)

	// Record metrics
	p.metrics.recordTurnDuration(context.Background(), duration, reason)

	// Create a copy of the buffer
	audioCopy := make([]byte, len(p.speechBuffer))
	copy(audioCopy, p.speechBuffer)

	// Reset state
	p.isSpeaking = false
	p.speechBuffer = nil

	return &TurnEvent{
		Audio:     audioCopy,
		StartTime: p.speechStart,
		EndTime:   endTime,
		Duration:  duration,
		Reason:    reason,
	}
}

// flushRemainingTurn emits any in-progress turn.
func (p *AudioPipeline) flushRemainingTurn(turnsOut chan<- *TurnEvent, reason TurnEndReason) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.isSpeaking && len(p.speechBuffer) > 0 {
		speechDuration := time.Since(p.speechStart)
		if speechDuration >= p.turn.minSpeechDuration {
			turn := p.emitTurnLocked(reason)
			select {
			case turnsOut <- turn:
			default:
			}
		}
	}
}

// Close releases all resources.
func (p *AudioPipeline) Close() error {
	var errs []error

	if p.vad != nil {
		if err := p.vad.Close(); err != nil {
			errs = append(errs, fmt.Errorf("VAD close: %w", err))
		}
	}

	if p.denoiser != nil {
		if err := p.denoiser.Close(); err != nil {
			errs = append(errs, fmt.Errorf("denoiser close: %w", err))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// ============================================================================
// OTEL Metrics
// ============================================================================

// PipelineMetrics provides OTEL instrumentation for the audio pipeline.
type PipelineMetrics struct {
	tracer trace.Tracer
	meter  metric.Meter

	vadLatency       metric.Float64Histogram
	denoiseLatency   metric.Float64Histogram
	turnDuration     metric.Float64Histogram
	speechSegments   metric.Int64Counter
	vadDecisions     metric.Int64Counter
	errors           metric.Int64Counter
}

func newPipelineMetrics() (*PipelineMetrics, error) {
	meter := otel.Meter("beluga.voice.pipeline")

	vadLatency, err := meter.Float64Histogram(
		"beluga.voice.vad_latency_seconds",
		metric.WithDescription("Latency of VAD processing per frame"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.001, 0.005, 0.01, 0.02, 0.05, 0.1),
	)
	if err != nil {
		return nil, err
	}

	denoiseLatency, err := meter.Float64Histogram(
		"beluga.voice.denoise_latency_seconds",
		metric.WithDescription("Latency of noise cancellation per frame"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.001, 0.005, 0.01, 0.02, 0.05, 0.1),
	)
	if err != nil {
		return nil, err
	}

	turnDuration, err := meter.Float64Histogram(
		"beluga.voice.turn_duration_seconds",
		metric.WithDescription("Duration of user speech turns"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.5, 1.0, 2.0, 5.0, 10.0, 20.0, 30.0),
	)
	if err != nil {
		return nil, err
	}

	speechSegments, err := meter.Int64Counter(
		"beluga.voice.speech_segments_total",
		metric.WithDescription("Total number of detected speech segments"),
	)
	if err != nil {
		return nil, err
	}

	vadDecisions, err := meter.Int64Counter(
		"beluga.voice.vad_decisions_total",
		metric.WithDescription("Total VAD decisions by result"),
	)
	if err != nil {
		return nil, err
	}

	errors, err := meter.Int64Counter(
		"beluga.voice.pipeline_errors_total",
		metric.WithDescription("Total pipeline errors by component"),
	)
	if err != nil {
		return nil, err
	}

	return &PipelineMetrics{
		tracer:         otel.Tracer("beluga.voice.pipeline"),
		meter:          meter,
		vadLatency:     vadLatency,
		denoiseLatency: denoiseLatency,
		turnDuration:   turnDuration,
		speechSegments: speechSegments,
		vadDecisions:   vadDecisions,
		errors:         errors,
	}, nil
}

func (m *PipelineMetrics) recordVADLatency(ctx context.Context, d time.Duration) {
	m.vadLatency.Record(ctx, d.Seconds())
}

func (m *PipelineMetrics) recordDenoiseLatency(ctx context.Context, d time.Duration) {
	m.denoiseLatency.Record(ctx, d.Seconds())
}

func (m *PipelineMetrics) recordTurnDuration(ctx context.Context, d time.Duration, reason TurnEndReason) {
	m.turnDuration.Record(ctx, d.Seconds(),
		metric.WithAttributes(
			attribute.String("end_reason", string(reason)),
		),
	)
}

func (m *PipelineMetrics) incrementSpeechSegments(ctx context.Context) {
	m.speechSegments.Add(ctx, 1)
}

func (m *PipelineMetrics) recordVADDecision(ctx context.Context, isSpeech bool) {
	result := "silence"
	if isSpeech {
		result = "speech"
	}
	m.vadDecisions.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("result", result),
		),
	)
}

func (m *PipelineMetrics) recordError(ctx context.Context, component string) {
	m.errors.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("component", component),
		),
	)
}

// ============================================================================
// Mock Implementations (for demonstration/testing)
// ============================================================================

// mockVAD is a simple VAD implementation for demonstration.
// In production, you would use a real VAD library like Silero or WebRTC VAD.
type mockVAD struct {
	threshold float64
}

func createVAD(ctx context.Context, config PipelineConfig) (VADDetector, error) {
	// In a real implementation, you would instantiate the appropriate VAD
	// based on config.VADModel (silero, webrtc, energy, etc.)
	return &mockVAD{threshold: config.VADThreshold}, nil
}

func (v *mockVAD) Detect(ctx context.Context, audio []byte) (float64, error) {
	// Simple energy-based VAD for demonstration
	// Real implementation would use a proper VAD model
	if len(audio) == 0 {
		return 0, nil
	}

	// Calculate RMS energy
	var sum float64
	for i := 0; i < len(audio)-1; i += 2 {
		sample := int16(audio[i]) | int16(audio[i+1])<<8
		sum += float64(sample * sample)
	}
	rms := sum / float64(len(audio)/2)

	// Normalize to 0-1 range (very simplified)
	probability := rms / 32768.0 / 32768.0 * 100
	if probability > 1.0 {
		probability = 1.0
	}

	return probability, nil
}

func (v *mockVAD) Threshold() float64 {
	return v.threshold
}

func (v *mockVAD) Close() error {
	return nil
}

// mockDenoiser is a passthrough denoiser for demonstration.
type mockDenoiser struct{}

func createDenoiser(ctx context.Context, config PipelineConfig) (Denoiser, error) {
	// In a real implementation, you would instantiate RNNoise, etc.
	return &mockDenoiser{}, nil
}

func (d *mockDenoiser) Process(ctx context.Context, audio []byte) ([]byte, error) {
	// Passthrough for demonstration
	return audio, nil
}

func (d *mockDenoiser) Close() error {
	return nil
}

// ============================================================================
// Example Usage
// ============================================================================

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create the audio pipeline with configuration
	pipeline, err := NewAudioPipeline(ctx,
		WithSampleRate(16000),
		WithVADModel("silero"),
		WithVADThreshold(0.5),
		WithSilenceDuration(500*time.Millisecond),
		WithDenoise(true),
	)
	if err != nil {
		log.Fatalf("Failed to create pipeline: %v", err)
	}
	defer pipeline.Close()

	// Create channels for audio streaming
	audioIn := make(chan []byte, 100)
	turnsOut := make(chan *TurnEvent, 10)

	// Start the pipeline in a goroutine
	go func() {
		if err := pipeline.ProcessAudioStream(ctx, audioIn, turnsOut); err != nil {
			if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
				log.Printf("Pipeline error: %v", err)
			}
		}
	}()

	// Process turns in another goroutine
	go func() {
		for turn := range turnsOut {
			log.Printf("Turn received: duration=%v, reason=%s, audio_size=%d bytes",
				turn.Duration, turn.Reason, len(turn.Audio))
			// In a real application, you would:
			// 1. Send audio to STT for transcription
			// 2. Process the transcript with your agent
			// 3. Generate a TTS response
		}
	}()

	// Simulate audio input (in real app, this comes from microphone/WebSocket)
	frameSize := 512 * 2 // 512 samples * 2 bytes per sample (16-bit audio)
	for i := 0; i < 100; i++ {
		// Generate a fake audio frame
		frame := make([]byte, frameSize)
		// In production, this would be real audio data
		audioIn <- frame
		time.Sleep(32 * time.Millisecond) // ~32ms per frame at 16kHz
	}

	close(audioIn)

	// Wait for processing to complete
	time.Sleep(time.Second)
	log.Println("Pipeline demo complete")
}
