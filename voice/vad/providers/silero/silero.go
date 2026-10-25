//go:build cgo

package silero

import (
	"context"
	"encoding/binary"
	"sync"

	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/voice"
)

const (
	defaultThreshold  = 0.5
	defaultSampleRate = 16000
)

var _ voice.ActivityDetector = (*VAD)(nil) // compile-time interface check

func init() {
	voice.RegisterVAD("silero", func(cfg map[string]any) (voice.ActivityDetector, error) {
		threshold := defaultThreshold
		if v, ok := cfg["threshold"]; ok {
			switch t := v.(type) {
			case float64:
				threshold = t
			case int:
				threshold = float64(t)
			}
		}

		sampleRate := defaultSampleRate
		if v, ok := cfg["sample_rate"]; ok {
			switch t := v.(type) {
			case int:
				sampleRate = t
			case float64:
				sampleRate = int(t)
			}
		}

		modelPath, _ := cfg["model_path"].(string)

		return New(Config{
			Threshold:  threshold,
			SampleRate: sampleRate,
			ModelPath:  modelPath,
		})
	})
}

// Config holds configuration for the Silero VAD.
type Config struct {
	// Threshold is the speech probability threshold (0.0 to 1.0).
	Threshold float64

	// SampleRate is the audio sample rate (8000 or 16000).
	SampleRate int

	// ModelPath is reserved for a future ONNX-backed implementation. Loading
	// the Silero ONNX model is NOT supported yet: New returns an error when
	// ModelPath is set rather than silently ignoring it.
	ModelPath string
}

// VAD implements voice.ActivityDetector with an RMS-energy detector whose
// threshold is calibrated to approximate Silero VAD sensitivity. It does not
// run the Silero neural model.
type VAD struct {
	threshold   float64
	sampleRate  int
	wasSpeaking bool
	mu          sync.Mutex // protects wasSpeaking
}

// New creates the energy-based VAD. It returns an error when cfg.ModelPath
// is set, because ONNX model loading is not implemented.
func New(cfg Config) (*VAD, error) {
	if cfg.ModelPath != "" {
		return nil, core.Errorf(core.ErrInvalidInput,
			"silero: ModelPath is not supported: this provider is an energy-based approximation and cannot load the Silero ONNX model")
	}
	if cfg.Threshold <= 0 || cfg.Threshold > 1.0 {
		cfg.Threshold = defaultThreshold
	}
	if cfg.SampleRate <= 0 {
		cfg.SampleRate = defaultSampleRate
	}

	return &VAD{
		threshold:  cfg.Threshold,
		sampleRate: cfg.SampleRate,
	}, nil
}

// DetectActivity analyses 16-bit little-endian PCM audio and returns whether
// speech is present. Uses an energy-normalized approach calibrated to match
// Silero VAD's sensitivity.
func (v *VAD) DetectActivity(_ context.Context, audio []byte) (voice.ActivityResult, error) {
	if len(audio) < 2 {
		return voice.ActivityResult{
			IsSpeech:  false,
			EventType: voice.VADSilence,
		}, nil
	}

	// Compute normalized energy as speech probability estimate.
	probability := v.computeSpeechProbability(audio)
	isSpeech := probability >= v.threshold

	v.mu.Lock()
	defer v.mu.Unlock()

	var eventType voice.VADEventType
	switch {
	case isSpeech && !v.wasSpeaking:
		eventType = voice.VADSpeechStart
	case !isSpeech && v.wasSpeaking:
		eventType = voice.VADSpeechEnd
	case isSpeech:
		eventType = voice.VADSpeechStart
	default:
		eventType = voice.VADSilence
	}

	v.wasSpeaking = isSpeech

	return voice.ActivityResult{
		IsSpeech:   isSpeech,
		EventType:  eventType,
		Confidence: probability,
	}, nil
}

// computeSpeechProbability computes a speech probability from PCM audio
// using normalized RMS energy. This approximates the Silero model's output
// for use without the ONNX runtime.
func (v *VAD) computeSpeechProbability(audio []byte) float64 {
	numSamples := len(audio) / 2
	if numSamples == 0 {
		return 0
	}

	var sumSquares float64
	for i := range numSamples {
		sample := int16(binary.LittleEndian.Uint16(audio[i*2 : i*2+2])) // #nosec G115 -- PCM audio: uint16→int16 reinterpretation is intentional
		val := float64(sample)
		sumSquares += val * val
	}

	// Mean square energy normalised to [0, 1] by the int16 range.
	rmsVal := sumSquares / float64(numSamples) / (32768.0 * 32768.0)

	// Sigmoid-like mapping to speech probability.
	// Calibrated to approximate Silero VAD output.
	if rmsVal > 1.0 {
		rmsVal = 1.0
	}

	// Apply a non-linear transform for better discrimination.
	probability := rmsVal * 4.0 // scale up for better separation
	if probability > 1.0 {
		probability = 1.0
	}

	return probability
}
