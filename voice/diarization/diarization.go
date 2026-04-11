package diarization

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

// speakerSilence is the sentinel SpeakerID used internally by the
// EnergyDiarizer to mark low-energy (silence) windows. It is never
// emitted as part of a returned SpeakerSegment — callers only see
// real speaker IDs.
const speakerSilence = "silence"

// Diarizer identifies different speakers within audio data.
type Diarizer interface {
	// Diarize processes audio data and returns speaker segments.
	Diarize(ctx context.Context, audio []byte, opts ...DiarizeOption) ([]SpeakerSegment, error)
}

// SpeakerTracker maintains speaker identity across multiple diarization calls,
// allowing consistent speaker IDs across a session.
type SpeakerTracker interface {
	// Track assigns consistent speaker IDs to segments based on prior observations.
	Track(ctx context.Context, segments []SpeakerSegment) ([]SpeakerSegment, error)
	// Reset clears all tracked speaker state.
	Reset(ctx context.Context) error
}

// SpeakerSegment represents a time-bounded segment attributed to a speaker.
type SpeakerSegment struct {
	// SpeakerID identifies the speaker.
	SpeakerID string `json:"speaker_id"`
	// Start is the start time of the segment relative to audio start.
	Start time.Duration `json:"start"`
	// End is the end time of the segment relative to audio start.
	End time.Duration `json:"end"`
	// Confidence is the diarization confidence score in [0.0, 1.0].
	Confidence float64 `json:"confidence"`
	// Text is the optional transcript for this segment.
	Text string `json:"text,omitempty"`
	// Metadata holds extra attributes.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// Duration returns the length of the segment.
func (s SpeakerSegment) Duration() time.Duration {
	return s.End - s.Start
}

// DiarizeOption configures a diarization call.
type DiarizeOption func(*diarizeOptions)

type diarizeOptions struct {
	maxSpeakers int
	minSegment  time.Duration
	sampleRate  int
}

// WithMaxSpeakers sets the maximum number of speakers to detect.
func WithMaxSpeakers(n int) DiarizeOption {
	return func(o *diarizeOptions) { o.maxSpeakers = n }
}

// WithMinSegmentDuration sets the minimum segment duration.
func WithMinSegmentDuration(d time.Duration) DiarizeOption {
	return func(o *diarizeOptions) { o.minSegment = d }
}

// WithSampleRate sets the audio sample rate.
func WithSampleRate(rate int) DiarizeOption {
	return func(o *diarizeOptions) { o.sampleRate = rate }
}

// Factory creates a Diarizer from a Config.
type Factory func(cfg Config) (Diarizer, error)

// Config holds configuration for creating a Diarizer.
type Config struct {
	// MaxSpeakers is the default maximum number of speakers.
	MaxSpeakers int
	// MinSegmentDuration is the default minimum segment duration.
	MinSegmentDuration time.Duration
	// SampleRate is the default audio sample rate.
	SampleRate int
}

var (
	registryMu sync.RWMutex
	registry   = make(map[string]Factory)
)

// Register adds a diarizer factory to the global registry.
func Register(name string, f Factory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[name] = f
}

// New creates a Diarizer by name from the registry.
func New(name string, cfg Config) (Diarizer, error) {
	registryMu.RLock()
	f, ok := registry[name]
	registryMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("diarization: unknown diarizer %q (registered: %v)", name, List())
	}
	return f(cfg)
}

// List returns the names of all registered diarizers, sorted alphabetically.
func List() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// EnergyDiarizer is a simple energy-based diarizer that detects speaker
// changes based on audio energy levels. Suitable for testing and as a
// baseline; production use requires a provider-backed implementation.
type EnergyDiarizer struct {
	maxSpeakers int
	minSegment  time.Duration
	sampleRate  int
}

var _ Diarizer = (*EnergyDiarizer)(nil)

// NewEnergyDiarizer creates an energy-based diarizer.
func NewEnergyDiarizer(cfg Config) *EnergyDiarizer {
	if cfg.MaxSpeakers <= 0 {
		cfg.MaxSpeakers = 2
	}
	if cfg.MinSegmentDuration <= 0 {
		cfg.MinSegmentDuration = 500 * time.Millisecond
	}
	if cfg.SampleRate <= 0 {
		cfg.SampleRate = 16000
	}
	return &EnergyDiarizer{
		maxSpeakers: cfg.MaxSpeakers,
		minSegment:  cfg.MinSegmentDuration,
		sampleRate:  cfg.SampleRate,
	}
}

// bytesPerSample is the PCM sample width used by the energy diarizer
// (16-bit little-endian).
const bytesPerSample = 2

// resolveOptions applies the variadic options on top of the diarizer's
// defaults and normalises any zero/negative values so the window-offset
// math below cannot divide by zero.
func (d *EnergyDiarizer) resolveOptions(opts []DiarizeOption) *diarizeOptions {
	o := &diarizeOptions{
		maxSpeakers: d.maxSpeakers,
		minSegment:  d.minSegment,
		sampleRate:  d.sampleRate,
	}
	for _, opt := range opts {
		opt(o)
	}
	if o.sampleRate <= 0 {
		o.sampleRate = 16000
	}
	if o.maxSpeakers <= 0 {
		o.maxSpeakers = 2
	}
	return o
}

// windowBytesFor returns the byte size of a 100ms window for the given
// sample rate, with a safe fallback.
func windowBytesFor(sampleRate int) int {
	wb := (sampleRate / 10) * bytesPerSample
	if wb <= 0 {
		return 3200 // fallback: 100ms @ 16kHz, 16-bit PCM
	}
	return wb
}

// offsetDuration converts a byte offset in the PCM stream to a duration
// relative to the start of the audio.
func offsetDuration(byteOffset, sampleRate int) time.Duration {
	return time.Duration(byteOffset/bytesPerSample) * time.Second / time.Duration(sampleRate)
}

// emitSegment appends seg to segments when it corresponds to a real
// speaker (not silence) and meets the minimum duration. It returns the
// possibly-updated segment slice.
func emitSegment(segments []SpeakerSegment, seg SpeakerSegment, minSegment time.Duration) []SpeakerSegment {
	if seg.SpeakerID == "" || seg.SpeakerID == speakerSilence {
		return segments
	}
	if seg.Duration() < minSegment {
		return segments
	}
	return append(segments, seg)
}

// Diarize segments audio based on energy levels.
func (d *EnergyDiarizer) Diarize(ctx context.Context, audio []byte, opts ...DiarizeOption) ([]SpeakerSegment, error) {
	if len(audio) == 0 {
		return nil, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	o := d.resolveOptions(opts)
	windowBytes := windowBytesFor(o.sampleRate)

	var (
		segments       []SpeakerSegment
		currentSpeaker string
		segStart       time.Duration
	)

	for i := 0; i < len(audio); i += windowBytes {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		end := i + windowBytes
		if end > len(audio) {
			end = len(audio)
		}

		speaker := assignSpeaker(computeEnergy(audio[i:end]), o.maxSpeakers)
		windowStart := offsetDuration(i, o.sampleRate)

		if speaker == currentSpeaker {
			continue
		}
		segments = emitSegment(segments, SpeakerSegment{
			SpeakerID:  currentSpeaker,
			Start:      segStart,
			End:        windowStart,
			Confidence: 0.7,
		}, o.minSegment)
		currentSpeaker = speaker
		segStart = windowStart
	}

	// Close final segment.
	segments = emitSegment(segments, SpeakerSegment{
		SpeakerID:  currentSpeaker,
		Start:      segStart,
		End:        offsetDuration(len(audio), o.sampleRate),
		Confidence: 0.7,
	}, o.minSegment)

	return segments, nil
}

func computeEnergy(window []byte) float64 {
	var sum float64
	for i := 0; i+1 < len(window); i += 2 {
		sample := int16(window[i]) | int16(window[i+1])<<8
		sum += float64(sample) * float64(sample)
	}
	n := len(window) / 2
	if n == 0 {
		return 0
	}
	return sum / float64(n)
}

func assignSpeaker(energy float64, maxSpeakers int) string {
	if maxSpeakers <= 0 {
		maxSpeakers = 2
	}
	// Simple threshold-based assignment. Low-energy windows are tagged
	// with the silence sentinel so the caller can filter them out rather
	// than emit them as real speaker segments.
	if energy < 100 {
		return speakerSilence
	}
	// Use energy level to assign speakers.
	speakerIdx := int(energy/10000) % maxSpeakers
	return fmt.Sprintf("speaker-%d", speakerIdx)
}

// InMemorySpeakerTracker tracks speakers across calls using in-memory state.
type InMemorySpeakerTracker struct {
	mu      sync.Mutex
	mapping map[string]string // provider speaker ID -> stable ID
	counter int
}

var _ SpeakerTracker = (*InMemorySpeakerTracker)(nil)

// NewSpeakerTracker creates an in-memory speaker tracker.
func NewSpeakerTracker() *InMemorySpeakerTracker {
	return &InMemorySpeakerTracker{
		mapping: make(map[string]string),
	}
}

// Track assigns consistent IDs to speaker segments.
func (t *InMemorySpeakerTracker) Track(_ context.Context, segments []SpeakerSegment) ([]SpeakerSegment, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	result := make([]SpeakerSegment, len(segments))
	copy(result, segments)

	for i, seg := range result {
		stableID, ok := t.mapping[seg.SpeakerID]
		if !ok {
			t.counter++
			stableID = fmt.Sprintf("speaker-%d", t.counter)
			t.mapping[seg.SpeakerID] = stableID
		}
		result[i].SpeakerID = stableID
	}

	return result, nil
}

// Reset clears all tracked state.
func (t *InMemorySpeakerTracker) Reset(_ context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.mapping = make(map[string]string)
	t.counter = 0
	return nil
}

func init() {
	Register("energy", func(cfg Config) (Diarizer, error) {
		return NewEnergyDiarizer(cfg), nil
	})
}
