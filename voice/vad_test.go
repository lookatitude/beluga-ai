package voice

import (
	"context"
	"encoding/binary"
	"math"
	"testing"
)

// generatePCM creates 16-bit little-endian PCM audio at the given amplitude.
func generatePCM(numSamples int, amplitude int16) []byte {
	buf := make([]byte, numSamples*2)
	for i := 0; i < numSamples; i++ {
		binary.LittleEndian.PutUint16(buf[i*2:], uint16(amplitude))
	}
	return buf
}

// generateSinePCM creates a sine wave PCM signal.
func generateSinePCM(numSamples int, amplitude float64, freq float64, sampleRate float64) []byte {
	buf := make([]byte, numSamples*2)
	for i := 0; i < numSamples; i++ {
		sample := int16(amplitude * math.Sin(2*math.Pi*freq*float64(i)/sampleRate))
		binary.LittleEndian.PutUint16(buf[i*2:], uint16(sample))
	}
	return buf
}

func TestEnergyVADDetectSpeech(t *testing.T) {
	vad := NewEnergyVAD(EnergyVADConfig{Threshold: 500})
	ctx := context.Background()

	// Loud audio (high amplitude sine) should be detected as speech.
	loudAudio := generateSinePCM(480, 5000, 440, 16000)
	result, err := vad.DetectActivity(ctx, loudAudio)
	if err != nil {
		t.Fatalf("DetectActivity() error = %v", err)
	}
	if !result.IsSpeech {
		t.Error("expected IsSpeech=true for loud audio")
	}
	if result.EventType != VADSpeechStart {
		t.Errorf("EventType = %q, want %q", result.EventType, VADSpeechStart)
	}
}

func TestEnergyVADDetectSilence(t *testing.T) {
	vad := NewEnergyVAD(EnergyVADConfig{Threshold: 500})
	ctx := context.Background()

	// Very quiet audio should be silence.
	quietAudio := generatePCM(480, 10)
	result, err := vad.DetectActivity(ctx, quietAudio)
	if err != nil {
		t.Fatalf("DetectActivity() error = %v", err)
	}
	if result.IsSpeech {
		t.Error("expected IsSpeech=false for quiet audio")
	}
	if result.EventType != VADSilence {
		t.Errorf("EventType = %q, want %q", result.EventType, VADSilence)
	}
}

func TestEnergyVADStateTransitions(t *testing.T) {
	vad := NewEnergyVAD(EnergyVADConfig{Threshold: 500})
	ctx := context.Background()

	// Start with silence.
	quiet := generatePCM(480, 10)
	r1, _ := vad.DetectActivity(ctx, quiet)
	if r1.EventType != VADSilence {
		t.Errorf("initial: EventType = %q, want %q", r1.EventType, VADSilence)
	}

	// Transition to speech.
	loud := generateSinePCM(480, 5000, 440, 16000)
	r2, _ := vad.DetectActivity(ctx, loud)
	if r2.EventType != VADSpeechStart {
		t.Errorf("speech start: EventType = %q, want %q", r2.EventType, VADSpeechStart)
	}

	// Transition back to silence.
	r3, _ := vad.DetectActivity(ctx, quiet)
	if r3.EventType != VADSpeechEnd {
		t.Errorf("speech end: EventType = %q, want %q", r3.EventType, VADSpeechEnd)
	}
}

func TestEnergyVADEmptyAudio(t *testing.T) {
	vad := NewEnergyVAD(EnergyVADConfig{Threshold: 500})
	ctx := context.Background()

	// Empty audio should be silence.
	result, err := vad.DetectActivity(ctx, nil)
	if err != nil {
		t.Fatalf("DetectActivity() error = %v", err)
	}
	if result.IsSpeech {
		t.Error("expected IsSpeech=false for empty audio")
	}

	// Single byte (too short for a sample).
	result2, err := vad.DetectActivity(ctx, []byte{0x01})
	if err != nil {
		t.Fatalf("DetectActivity() error = %v", err)
	}
	if result2.IsSpeech {
		t.Error("expected IsSpeech=false for single byte")
	}
}

func TestEnergyVADDefaultThreshold(t *testing.T) {
	vad := NewEnergyVAD(EnergyVADConfig{}) // zero threshold defaults to 1000
	if vad.Threshold != 1000 {
		t.Errorf("Threshold = %f, want 1000", vad.Threshold)
	}
}

func TestEnergyVADConfidence(t *testing.T) {
	vad := NewEnergyVAD(EnergyVADConfig{Threshold: 500})
	ctx := context.Background()

	// Very loud audio should have confidence clamped to 1.0.
	loud := generateSinePCM(480, 20000, 440, 16000)
	result, _ := vad.DetectActivity(ctx, loud)
	if result.Confidence > 1.0 || result.Confidence < 0.0 {
		t.Errorf("Confidence = %f, want [0, 1]", result.Confidence)
	}
}

func TestVADRegistry(t *testing.T) {
	// "energy" should be registered via init().
	names := ListVAD()
	found := false
	for _, n := range names {
		if n == "energy" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'energy' in ListVAD()")
	}

	// Create via registry.
	vad, err := NewVAD("energy", map[string]any{"threshold": 2000.0})
	if err != nil {
		t.Fatalf("NewVAD('energy') error = %v", err)
	}
	if vad == nil {
		t.Fatal("NewVAD('energy') returned nil")
	}
}

func TestVADRegistryUnknown(t *testing.T) {
	_, err := NewVAD("nonexistent", nil)
	if err == nil {
		t.Error("expected error for unknown VAD")
	}
}

func TestComputeRMS(t *testing.T) {
	// All zeros should give RMS of 0.
	zeros := generatePCM(100, 0)
	if rms := computeRMS(zeros); rms != 0 {
		t.Errorf("computeRMS(zeros) = %f, want 0", rms)
	}

	// Empty should give 0.
	if rms := computeRMS(nil); rms != 0 {
		t.Errorf("computeRMS(nil) = %f, want 0", rms)
	}
}
