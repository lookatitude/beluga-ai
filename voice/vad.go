package voice

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"sort"
	"sync"
)

// VADEventType identifies the type of voice activity event.
type VADEventType string

const (
	// VADSpeechStart indicates the beginning of speech.
	VADSpeechStart VADEventType = "speech_start"

	// VADSpeechEnd indicates the end of speech.
	VADSpeechEnd VADEventType = "speech_end"

	// VADSilence indicates silence (no speech detected).
	VADSilence VADEventType = "silence"
)

// ActivityResult holds the result of a voice activity detection check.
type ActivityResult struct {
	// IsSpeech is true if speech was detected in the audio.
	IsSpeech bool

	// EventType describes the VAD event (start, end, or silence).
	EventType VADEventType

	// Confidence is the confidence level of the detection (0.0 to 1.0).
	Confidence float64
}

// VAD detects voice activity in audio data.
type VAD interface {
	// DetectActivity analyses an audio chunk and returns whether speech is
	// present along with an event type indicating state transitions.
	DetectActivity(ctx context.Context, audio []byte) (ActivityResult, error)
}

// VADFactory creates a VAD from configuration.
type VADFactory func(cfg map[string]any) (VAD, error)

var (
	vadRegistryMu sync.RWMutex
	vadRegistry   = make(map[string]VADFactory)
)

// RegisterVAD adds a named VAD factory to the global registry. It is intended
// to be called from provider init() functions. RegisterVAD panics if name is
// empty or already registered.
func RegisterVAD(name string, f VADFactory) {
	if name == "" {
		panic("voice: RegisterVAD called with empty name")
	}
	if f == nil {
		panic("voice: RegisterVAD called with nil factory for " + name)
	}

	vadRegistryMu.Lock()
	defer vadRegistryMu.Unlock()

	if _, dup := vadRegistry[name]; dup {
		panic("voice: RegisterVAD called twice for " + name)
	}
	vadRegistry[name] = f
}

// NewVAD creates a VAD by looking up the named factory in the registry.
func NewVAD(name string, cfg map[string]any) (VAD, error) {
	vadRegistryMu.RLock()
	f, ok := vadRegistry[name]
	vadRegistryMu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("voice: unknown VAD %q", name)
	}
	return f(cfg)
}

// ListVAD returns the sorted names of all registered VAD factories.
func ListVAD() []string {
	vadRegistryMu.RLock()
	defer vadRegistryMu.RUnlock()

	names := make([]string, 0, len(vadRegistry))
	for name := range vadRegistry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// EnergyVAD is a simple energy-threshold voice activity detector.
// It computes the RMS energy of 16-bit PCM audio and compares it to a
// configurable threshold to determine speech presence.
type EnergyVAD struct {
	// Threshold is the RMS energy level above which speech is detected.
	// Typical values range from 500 to 3000 for 16-bit PCM audio.
	Threshold float64

	// wasSpeaking tracks the previous state for edge detection.
	wasSpeaking bool
}

// EnergyVADConfig holds configuration for creating an EnergyVAD.
type EnergyVADConfig struct {
	// Threshold is the RMS energy threshold for speech detection.
	// Defaults to 1000 if zero.
	Threshold float64
}

// NewEnergyVAD creates a new EnergyVAD with the given configuration.
func NewEnergyVAD(cfg EnergyVADConfig) *EnergyVAD {
	threshold := cfg.Threshold
	if threshold == 0 {
		threshold = 1000
	}
	return &EnergyVAD{
		Threshold: threshold,
	}
}

// DetectActivity computes the RMS energy of the audio data and determines
// whether speech is present. The audio data must be 16-bit little-endian PCM.
func (v *EnergyVAD) DetectActivity(_ context.Context, audio []byte) (ActivityResult, error) {
	if len(audio) < 2 {
		return ActivityResult{
			IsSpeech:  false,
			EventType: VADSilence,
		}, nil
	}

	rms := computeRMS(audio)
	isSpeech := rms >= v.Threshold

	// Compute confidence as a ratio clamped to [0, 1].
	confidence := rms / (v.Threshold * 2)
	if confidence > 1.0 {
		confidence = 1.0
	}

	var eventType VADEventType
	switch {
	case isSpeech && !v.wasSpeaking:
		eventType = VADSpeechStart
	case !isSpeech && v.wasSpeaking:
		eventType = VADSpeechEnd
	case isSpeech:
		eventType = VADSpeechStart // ongoing speech
	default:
		eventType = VADSilence
	}

	v.wasSpeaking = isSpeech

	return ActivityResult{
		IsSpeech:   isSpeech,
		EventType:  eventType,
		Confidence: confidence,
	}, nil
}

// computeRMS calculates the root mean square of 16-bit little-endian PCM audio.
func computeRMS(audio []byte) float64 {
	numSamples := len(audio) / 2
	if numSamples == 0 {
		return 0
	}

	var sumSquares float64
	for i := 0; i < numSamples; i++ {
		sample := int16(binary.LittleEndian.Uint16(audio[i*2 : i*2+2]))
		sumSquares += float64(sample) * float64(sample)
	}

	return math.Sqrt(sumSquares / float64(numSamples))
}

func init() {
	RegisterVAD("energy", func(cfg map[string]any) (VAD, error) {
		threshold := 1000.0
		if v, ok := cfg["threshold"]; ok {
			switch t := v.(type) {
			case float64:
				threshold = t
			case int:
				threshold = float64(t)
			}
		}
		return NewEnergyVAD(EnergyVADConfig{Threshold: threshold}), nil
	})
}
