// Package webrtc provides a pure Go WebRTC-style VAD (Voice Activity Detection)
// provider for the Beluga AI voice pipeline. It uses energy and zero-crossing
// rate analysis on 16-bit PCM audio to detect speech.
//
// Usage:
//
//	import _ "github.com/lookatitude/beluga-ai/voice/vad/providers/webrtc"
//
//	vad, err := voice.NewVAD("webrtc", map[string]any{"threshold": 1500.0})
//	result, err := vad.DetectActivity(ctx, audioPCM)
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

var _ voice.VAD = (*VAD)(nil) // compile-time interface check

func init() {
	voice.RegisterVAD("webrtc", func(cfg map[string]any) (voice.VAD, error) {
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

// VAD implements voice.VAD using energy and zero-crossing rate analysis.
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

// DetectActivity analyses 16-bit little-endian PCM audio and returns whether
// speech is present based on energy and zero-crossing rate.
func (v *VAD) DetectActivity(_ context.Context, audio []byte) (voice.ActivityResult, error) {
	if len(audio) < 4 {
		return voice.ActivityResult{
			IsSpeech:  false,
			EventType: voice.VADSilence,
		}, nil
	}

	numSamples := len(audio) / 2
	samples := make([]int16, numSamples)
	for i := range numSamples {
		samples[i] = int16(binary.LittleEndian.Uint16(audio[i*2 : i*2+2]))
	}

	// Compute RMS energy.
	var sumSquares float64
	for _, s := range samples {
		sumSquares += float64(s) * float64(s)
	}
	rms := math.Sqrt(sumSquares / float64(numSamples))

	// Compute zero-crossing rate (ZCR).
	var crossings int
	for i := 1; i < numSamples; i++ {
		if (samples[i-1] >= 0 && samples[i] < 0) ||
			(samples[i-1] < 0 && samples[i] >= 0) {
			crossings++
		}
	}
	zcr := float64(crossings) / float64(numSamples-1)

	// Speech is detected when energy is above threshold and ZCR suggests
	// voiced content (not just noise which tends to have high ZCR).
	energyAbove := rms >= v.energyThreshold
	zcrBelow := zcr <= v.zcrThreshold || v.zcrThreshold == 0
	isSpeech := energyAbove && zcrBelow

	// Compute confidence.
	confidence := rms / (v.energyThreshold * 2)
	if confidence > 1.0 {
		confidence = 1.0
	}
	if !zcrBelow {
		confidence *= 0.5 // reduce confidence for high ZCR (likely noise)
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	var eventType voice.VADEventType
	switch {
	case isSpeech && !v.wasSpeaking:
		eventType = voice.VADSpeechStart
	case !isSpeech && v.wasSpeaking:
		eventType = voice.VADSpeechEnd
	case isSpeech:
		eventType = voice.VADSpeechStart // ongoing speech
	default:
		eventType = voice.VADSilence
	}

	v.wasSpeaking = isSpeech

	return voice.ActivityResult{
		IsSpeech:   isSpeech,
		EventType:  eventType,
		Confidence: confidence,
	}, nil
}
