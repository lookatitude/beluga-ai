package webrtc

import (
	"context"
	"encoding/binary"
	"math"
	"sync"

	"github.com/lookatitude/beluga-ai/voice"
)

const (
	defaultEnergyThreshold = 1000.0
	defaultZCRThreshold    = 0.1
)

var _ voice.ActivityDetector = (*VAD)(nil) // compile-time interface check

func init() {
	voice.RegisterVAD("webrtc", func(cfg map[string]any) (voice.ActivityDetector, error) {
		energyThreshold := defaultEnergyThreshold
		if v, ok := cfg["threshold"]; ok {
			switch t := v.(type) {
			case float64:
				energyThreshold = t
			case int:
				energyThreshold = float64(t)
			}
		}

		zcrThreshold := defaultZCRThreshold
		if v, ok := cfg["zcr_threshold"]; ok {
			switch t := v.(type) {
			case float64:
				zcrThreshold = t
			}
		}

		return New(energyThreshold, zcrThreshold), nil
	})
}

// VAD implements voice.ActivityDetector using energy and zero-crossing rate analysis.
type VAD struct {
	energyThreshold float64
	zcrThreshold    float64
	wasSpeaking     bool
	mu              sync.Mutex // protects wasSpeaking
}

// New creates a new WebRTC-style VAD with the given thresholds.
func New(energyThreshold, zcrThreshold float64) *VAD {
	if energyThreshold == 0 {
		energyThreshold = defaultEnergyThreshold
	}
	if zcrThreshold == 0 {
		zcrThreshold = defaultZCRThreshold
	}
	return &VAD{
		energyThreshold: energyThreshold,
		zcrThreshold:    zcrThreshold,
	}
}

// decodeSamples converts raw 16-bit little-endian PCM bytes to int16 samples.
func decodeSamples(audio []byte) []int16 {
	numSamples := len(audio) / 2
	samples := make([]int16, numSamples)
	for i := range numSamples {
		samples[i] = int16(binary.LittleEndian.Uint16(audio[i*2 : i*2+2]))
	}
	return samples
}

// computeRMS computes the root-mean-square energy of the samples.
func computeRMS(samples []int16) float64 {
	var sumSquares float64
	for _, s := range samples {
		sumSquares += float64(s) * float64(s)
	}
	return math.Sqrt(sumSquares / float64(len(samples)))
}

// computeZCR computes the zero-crossing rate of the samples.
func computeZCR(samples []int16) float64 {
	var crossings int
	for i := 1; i < len(samples); i++ {
		if (samples[i-1] >= 0 && samples[i] < 0) ||
			(samples[i-1] < 0 && samples[i] >= 0) {
			crossings++
		}
	}
	return float64(crossings) / float64(len(samples)-1)
}

// computeConfidence calculates the speech confidence score from RMS and ZCR analysis.
func (v *VAD) computeConfidence(rms float64, zcrBelow bool) float64 {
	confidence := rms / (v.energyThreshold * 2)
	if confidence > 1.0 {
		confidence = 1.0
	}
	if !zcrBelow {
		confidence *= 0.5 // reduce confidence for high ZCR (likely noise)
	}
	return confidence
}

// resolveEventType determines the VAD event type based on current and previous speech state.
func (v *VAD) resolveEventType(isSpeech bool) voice.VADEventType {
	switch {
	case isSpeech && !v.wasSpeaking:
		return voice.VADSpeechStart
	case !isSpeech && v.wasSpeaking:
		return voice.VADSpeechEnd
	case isSpeech:
		return voice.VADSpeechStart // ongoing speech
	default:
		return voice.VADSilence
	}
}

// DetectActivity analyses 16-bit little-endian PCM audio and returns whether
// speech is present based on energy and zero-crossing rate.
func (v *VAD) DetectActivity(_ context.Context, audio []byte) (voice.ActivityResult, error) {
	if len(audio) < 4 {
		return voice.ActivityResult{
			IsSpeech:  false,
			EventType: voice.VADSilence,
		}, nil
	}

	samples := decodeSamples(audio)
	rms := computeRMS(samples)
	zcr := computeZCR(samples)

	energyAbove := rms >= v.energyThreshold
	zcrBelow := zcr <= v.zcrThreshold || v.zcrThreshold == 0
	isSpeech := energyAbove && zcrBelow
	confidence := v.computeConfidence(rms, zcrBelow)

	v.mu.Lock()
	defer v.mu.Unlock()

	eventType := v.resolveEventType(isSpeech)
	v.wasSpeaking = isSpeech

	return voice.ActivityResult{
		IsSpeech:   isSpeech,
		EventType:  eventType,
		Confidence: confidence,
	}, nil
}
