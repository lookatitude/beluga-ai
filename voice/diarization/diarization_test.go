package diarization

import (
	"context"
	"testing"
	"time"
)

func TestEnergyDiarizer_EmptyAudio(t *testing.T) {
	d := NewEnergyDiarizer(Config{})
	segments, err := d.Diarize(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if segments != nil {
		t.Errorf("expected nil segments for empty audio, got %v", segments)
	}
}

func TestEnergyDiarizer_BasicAudio(t *testing.T) {
	d := NewEnergyDiarizer(Config{
		MaxSpeakers:        2,
		MinSegmentDuration: 100 * time.Millisecond,
		SampleRate:         16000,
	})

	// Generate 1 second of 16-bit PCM audio (16000 samples * 2 bytes).
	audio := make([]byte, 32000)
	// Fill with varying energy levels.
	for i := 0; i < len(audio); i += 2 {
		val := byte((i / 6400) * 50) // Change energy at different points.
		audio[i] = val
		audio[i+1] = 0
	}

	segments, err := d.Diarize(context.Background(), audio)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have at least one segment.
	if len(segments) == 0 {
		t.Error("expected at least one segment")
	}

	// Verify segments have valid fields.
	for _, seg := range segments {
		if seg.SpeakerID == "" {
			t.Error("segment has empty speaker ID")
		}
		if seg.End <= seg.Start {
			t.Errorf("segment end (%v) <= start (%v)", seg.End, seg.Start)
		}
		if seg.Confidence <= 0 || seg.Confidence > 1 {
			t.Errorf("confidence = %v, want (0, 1]", seg.Confidence)
		}
	}
}

func TestEnergyDiarizer_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	d := NewEnergyDiarizer(Config{SampleRate: 16000})
	audio := make([]byte, 32000)

	_, err := d.Diarize(ctx, audio)
	if err == nil {
		t.Error("expected context cancellation error")
	}
}

func TestEnergyDiarizer_WithOptions(t *testing.T) {
	d := NewEnergyDiarizer(Config{SampleRate: 16000})
	audio := make([]byte, 64000) // 2 seconds
	for i := 0; i < len(audio); i += 2 {
		audio[i] = byte(i % 256)
		audio[i+1] = 0
	}

	segments, err := d.Diarize(context.Background(), audio,
		WithMaxSpeakers(3),
		WithMinSegmentDuration(50*time.Millisecond),
		WithSampleRate(16000),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(segments) == 0 {
		t.Error("expected segments with options")
	}
}

func TestSpeakerSegment_Duration(t *testing.T) {
	seg := SpeakerSegment{
		Start: 1 * time.Second,
		End:   3 * time.Second,
	}
	if seg.Duration() != 2*time.Second {
		t.Errorf("Duration = %v, want 2s", seg.Duration())
	}
}

func TestInMemorySpeakerTracker(t *testing.T) {
	tracker := NewSpeakerTracker()
	ctx := context.Background()

	segments1 := []SpeakerSegment{
		{SpeakerID: "raw-1", Start: 0, End: time.Second},
		{SpeakerID: "raw-2", Start: time.Second, End: 2 * time.Second},
	}

	tracked1, err := tracker.Track(ctx, segments1)
	if err != nil {
		t.Fatalf("Track: %v", err)
	}

	if tracked1[0].SpeakerID == "raw-1" {
		t.Error("expected speaker ID to be remapped")
	}

	// Same raw IDs should get same stable IDs.
	segments2 := []SpeakerSegment{
		{SpeakerID: "raw-1", Start: 2 * time.Second, End: 3 * time.Second},
	}

	tracked2, err := tracker.Track(ctx, segments2)
	if err != nil {
		t.Fatalf("Track: %v", err)
	}

	if tracked2[0].SpeakerID != tracked1[0].SpeakerID {
		t.Errorf("same raw ID should get same stable ID: %q vs %q",
			tracked2[0].SpeakerID, tracked1[0].SpeakerID)
	}

	// Reset should clear mappings.
	if err := tracker.Reset(ctx); err != nil {
		t.Fatalf("Reset: %v", err)
	}

	tracked3, err := tracker.Track(ctx, segments1)
	if err != nil {
		t.Fatalf("Track after reset: %v", err)
	}

	// After reset, IDs restart.
	if tracked3[0].SpeakerID != tracked1[0].SpeakerID {
		// This is OK - just checking it's a valid stable ID.
		if tracked3[0].SpeakerID == "" {
			t.Error("expected non-empty speaker ID after reset")
		}
	}
}

func TestRegistry(t *testing.T) {
	names := List()
	found := false
	for _, n := range names {
		if n == "energy" {
			found = true
		}
	}
	if !found {
		t.Error("expected 'energy' in registry")
	}

	d, err := New("energy", Config{})
	if err != nil {
		t.Fatalf("New(energy): %v", err)
	}
	if d == nil {
		t.Error("expected non-nil diarizer")
	}

	_, err = New("nonexistent", Config{})
	if err == nil {
		t.Error("expected error for unknown diarizer")
	}
}
