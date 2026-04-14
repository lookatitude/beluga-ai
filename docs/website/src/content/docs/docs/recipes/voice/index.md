---
title: Voice Recipes
description: "Go recipes for voice AI: stream scaling, noise reduction, barge-in detection, latency optimization, and frame-based pipelines with Beluga AI."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, voice AI recipes, Go voice pipeline, FrameProcessor, speech processing, real-time audio, voice engineering"
sidebar:
  order: 0
---

Beluga AI's voice system is built around the **FrameProcessor** interface, a composable, frame-based pipeline architecture. Each processor handles a single concern (noise reduction, VAD, STT, LLM, TTS) and processors chain together via channels. This design lets you insert, remove, or reorder processing stages without rewriting the pipeline, and it mirrors how real-time audio systems work in production: small, focused units connected by typed data streams.

The recipes in this section cover common voice engineering challenges. Each recipe demonstrates a self-contained pattern that you can adapt for your use case.

## Scale Concurrent Voice Streams

**Problem:** You need to handle hundreds of simultaneous voice sessions without degrading latency or exhausting resources.

**Solution:** Use a session pool with bounded concurrency and per-session resource limits. Voice sessions are resource-intensive (each holds STT/TTS connections, audio buffers, and agent state), so unbounded creation leads to memory exhaustion and cascading latency spikes. A pool with a semaphore-based processing limit ensures graceful degradation under load rather than catastrophic failure.

```go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"

	"github.com/lookatitude/beluga-ai/voice"
)

// SessionPool manages concurrent voice sessions with resource limits.
type SessionPool struct {
	maxSessions int
	active      atomic.Int64
	sessions    sync.Map // id → *voice.VoiceSession

	sem chan struct{} // Bounds concurrent processing goroutines.
}

func NewSessionPool(maxSessions, maxConcurrentProcessing int) *SessionPool {
	return &SessionPool{
		maxSessions: maxSessions,
		sem:         make(chan struct{}, maxConcurrentProcessing),
	}
}

// Acquire creates a new session if capacity allows.
func (p *SessionPool) Acquire(sessionID string) (*voice.VoiceSession, error) {
	current := p.active.Load()
	if int(current) >= p.maxSessions {
		return nil, fmt.Errorf("session pool full: %d/%d", current, p.maxSessions)
	}

	session := voice.NewSession(sessionID)
	p.sessions.Store(sessionID, session)
	p.active.Add(1)

	slog.Info("session acquired",
		"session_id", sessionID,
		"active", p.active.Load(),
	)

	return session, nil
}

// Release removes a session and frees resources.
func (p *SessionPool) Release(sessionID string) {
	if _, loaded := p.sessions.LoadAndDelete(sessionID); loaded {
		p.active.Add(-1)
		slog.Info("session released",
			"session_id", sessionID,
			"active", p.active.Load(),
		)
	}
}

// ProcessFrame processes an audio frame with bounded concurrency.
func (p *SessionPool) ProcessFrame(ctx context.Context, sessionID string, frame voice.Frame) error {
	p.sem <- struct{}{}        // Acquire processing slot.
	defer func() { <-p.sem }() // Release processing slot.

	val, ok := p.sessions.Load(sessionID)
	if !ok {
		return fmt.Errorf("session %q not found", sessionID)
	}
	session := val.(*voice.VoiceSession)
	_ = session // Process the frame against this session.

	return nil
}

func main() {
	// Allow 500 concurrent sessions, 50 concurrent audio processing goroutines.
	pool := NewSessionPool(500, 50)

	session, err := pool.Acquire("session-123")
	if err != nil {
		slog.Error("acquire failed", "error", err)
		return
	}
	defer pool.Release(session.ID)

	fmt.Printf("Active sessions: %d\n", pool.active.Load())
}
```

---

## Handle Speech Interruptions (Barge-In)

**Problem:** The user starts speaking while the agent is still generating audio. You need to immediately stop TTS output and start processing the new input.

**Solution:** Use control frames to signal interruptions through the pipeline. In Beluga AI's frame-based architecture, control frames propagate through the same channel as audio and text frames but carry metadata that triggers special behavior in downstream processors. This approach keeps the interrupt signal in-band with the data stream, avoiding race conditions between separate signal and data paths.

```go
package main

import (
	"context"
	"log/slog"

	"github.com/lookatitude/beluga-ai/voice"
)

// InterruptHandler processes barge-in events in the voice pipeline.
type InterruptHandler struct{}

func (h *InterruptHandler) Process(ctx context.Context, in <-chan voice.Frame, out chan<- voice.Frame) error {
	defer close(out)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case frame, ok := <-in:
			if !ok {
				return nil
			}

			// Check for interrupt signal from VAD.
			if frame.Type == voice.FrameControl {
				signal, _ := frame.Metadata["signal"].(string)
				if signal == voice.SignalInterrupt {
					slog.Info("barge-in detected, flushing TTS queue")

					// Send interrupt downstream to stop TTS output.
					out <- voice.Frame{
						Type: voice.FrameControl,
						Metadata: map[string]any{
							"signal": voice.SignalInterrupt,
						},
					}
					continue
				}
			}

			// Pass through non-interrupt frames.
			out <- frame
		}
	}
}

func main() {
	// Chain the interrupt handler into the pipeline.
	pipeline := voice.Chain(
		&InterruptHandler{},
		// ... STT, LLM, TTS processors follow.
	)

	ctx := context.Background()
	in := make(chan voice.Frame)
	out := make(chan voice.Frame)

	go func() {
		err := pipeline.Process(ctx, in, out)
		if err != nil {
			slog.Error("pipeline error", "error", err)
		}
	}()

	// Simulate a barge-in.
	in <- voice.Frame{
		Type: voice.FrameControl,
		Metadata: map[string]any{
			"signal": voice.SignalInterrupt,
		},
	}

	close(in)
}
```

---

## Minimize Glass-to-Glass Latency

**Problem:** End-to-end voice latency exceeds 1 second, making the interaction feel unnatural. The target budget is under 800ms.

**Solution:** Optimize each pipeline stage and use preemptive generation. Glass-to-glass latency (the time from when the user stops speaking to when they hear the agent's response) is the single most important quality metric for voice agents. Research shows that latency above 800ms feels noticeably unnatural, while below 500ms feels conversational. The key is measuring each stage independently so you can identify and address bottlenecks.

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/lookatitude/beluga-ai/voice"
)

// LatencyTracker measures time through each pipeline stage.
type LatencyTracker struct {
	stageTimings map[string]time.Duration
}

func NewLatencyTracker() *LatencyTracker {
	return &LatencyTracker{
		stageTimings: make(map[string]time.Duration),
	}
}

// Wrap creates a FrameProcessor that tracks latency through a stage.
func (lt *LatencyTracker) Wrap(name string, inner voice.FrameProcessor) voice.FrameProcessor {
	return voice.FrameProcessorFunc(func(ctx context.Context, in <-chan voice.Frame, out chan<- voice.Frame) error {
		defer close(out)

		for frame := range in {
			start := time.Now()
			// Process through inner via a temp channel pair.
			tempIn := make(chan voice.Frame, 1)
			tempOut := make(chan voice.Frame, 1)
			tempIn <- frame
			close(tempIn)

			go inner.Process(ctx, tempIn, tempOut)

			for result := range tempOut {
				elapsed := time.Since(start)
				lt.stageTimings[name] = elapsed

				// Attach timing metadata.
				if result.Metadata == nil {
					result.Metadata = make(map[string]any)
				}
				result.Metadata["stage"] = name
				result.Metadata["latency_ms"] = elapsed.Milliseconds()

				out <- result
			}
		}
		return nil
	})
}

// Report prints the latency budget breakdown.
func (lt *LatencyTracker) Report() {
	var total time.Duration
	fmt.Println("Latency Budget:")
	fmt.Println("  Stage          | Target  | Actual")
	fmt.Println("  -------------- | ------- | ------")

	targets := map[string]time.Duration{
		"transport": 50 * time.Millisecond,
		"vad":       1 * time.Millisecond,
		"stt":       200 * time.Millisecond,
		"llm":       300 * time.Millisecond,
		"tts":       200 * time.Millisecond,
	}

	for stage, target := range targets {
		actual := lt.stageTimings[stage]
		total += actual
		status := "OK"
		if actual > target {
			status = "OVER"
		}
		fmt.Printf("  %-14s | %6dms | %6dms [%s]\n",
			stage, target.Milliseconds(), actual.Milliseconds(), status)
	}
	fmt.Printf("  %-14s | %6dms | %6dms\n", "TOTAL", 800, total.Milliseconds())
}

func main() {
	tracker := NewLatencyTracker()

	// Use tracker.Wrap() around each pipeline stage for visibility.
	// tracker.Wrap("stt", sttProcessor)
	// tracker.Wrap("llm", llmProcessor)
	// tracker.Wrap("tts", ttsProcessor)

	tracker.stageTimings["transport"] = 35 * time.Millisecond
	tracker.stageTimings["vad"] = 1 * time.Millisecond
	tracker.stageTimings["stt"] = 180 * time.Millisecond
	tracker.stageTimings["llm"] = 250 * time.Millisecond
	tracker.stageTimings["tts"] = 190 * time.Millisecond

	tracker.Report()
}
```

---

## Handle Long Utterances

**Problem:** Users sometimes speak for 30+ seconds, causing STT buffers to overflow or timeout.

**Solution:** Chunk long audio into segments at natural boundaries (pauses) and process incrementally. Most STT providers have internal buffer limits (typically 10-15 seconds), and sending a single 30-second audio blob results in timeouts or truncation. By segmenting at silence boundaries, each chunk stays within provider limits while preserving sentence integrity. The `SignalEndOfUtterance` control frame tells downstream processors that a complete thought has been delivered.

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/lookatitude/beluga-ai/voice"
)

// LongUtteranceHandler splits long speech into manageable segments.
type LongUtteranceHandler struct {
	maxSegmentDuration time.Duration
	silenceThreshold   time.Duration
}

func NewLongUtteranceHandler(maxSegment, silenceThreshold time.Duration) *LongUtteranceHandler {
	return &LongUtteranceHandler{
		maxSegmentDuration: maxSegment,
		silenceThreshold:   silenceThreshold,
	}
}

func (h *LongUtteranceHandler) Process(ctx context.Context, in <-chan voice.Frame, out chan<- voice.Frame) error {
	defer close(out)

	var buffer []voice.Frame
	var segmentStart time.Time
	var lastSpeech time.Time

	for frame := range in {
		now := time.Now()

		if frame.Type == voice.FrameAudio {
			if segmentStart.IsZero() {
				segmentStart = now
			}
			lastSpeech = now
			buffer = append(buffer, frame)

			// Check if we should flush the segment.
			segmentDuration := now.Sub(segmentStart)
			silenceDuration := now.Sub(lastSpeech)

			shouldFlush := segmentDuration >= h.maxSegmentDuration ||
				silenceDuration >= h.silenceThreshold

			if shouldFlush && len(buffer) > 0 {
				// Emit all buffered frames as a segment.
				for _, f := range buffer {
					out <- f
				}
				// Signal end of segment.
				out <- voice.Frame{
					Type: voice.FrameControl,
					Metadata: map[string]any{
						"signal":           voice.SignalEndOfUtterance,
						"segment_duration": segmentDuration.String(),
					},
				}
				buffer = buffer[:0]
				segmentStart = time.Time{}
			}
		} else {
			out <- frame
		}
	}

	// Flush remaining buffer.
	for _, f := range buffer {
		out <- f
	}

	return nil
}

func main() {
	handler := NewLongUtteranceHandler(
		10*time.Second,        // Max 10s per segment.
		500*time.Millisecond,  // Flush after 500ms silence.
	)
	fmt.Printf("Long utterance handler: max=%v, silence=%v\n",
		handler.maxSegmentDuration, handler.silenceThreshold)
}
```

---

## Preemptive Voice Generation

**Problem:** Waiting for the complete LLM response before starting TTS adds latency. You want TTS to start as soon as the first sentence is available.

**Solution:** Stream LLM output and feed complete sentences to TTS incrementally. This technique, sometimes called "sentence-level pipelining," overlaps LLM generation with TTS synthesis. Instead of waiting for the full response (which could take 2-3 seconds for a long answer), TTS begins synthesizing the first sentence while the LLM is still producing subsequent sentences. The result is that the user hears the start of the response much sooner, dramatically reducing perceived latency.

```go
package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/lookatitude/beluga-ai/voice"
)

// PreemptiveTTSFeeder sends complete sentences to TTS as they arrive from the LLM.
type PreemptiveTTSFeeder struct {
	sentenceEnders string
}

func NewPreemptiveTTSFeeder() *PreemptiveTTSFeeder {
	return &PreemptiveTTSFeeder{sentenceEnders: ".!?"}
}

func (f *PreemptiveTTSFeeder) Process(ctx context.Context, in <-chan voice.Frame, out chan<- voice.Frame) error {
	defer close(out)

	var buffer strings.Builder

	for frame := range in {
		if frame.Type != voice.FrameText {
			out <- frame
			continue
		}

		text, _ := frame.Data.(string)
		buffer.WriteString(text)

		// Check if the buffer contains a complete sentence.
		content := buffer.String()
		lastEnd := -1
		for i, ch := range content {
			if strings.ContainsRune(f.sentenceEnders, ch) {
				lastEnd = i
			}
		}

		if lastEnd >= 0 {
			// Send the complete sentence(s) to TTS.
			sentence := content[:lastEnd+1]
			out <- voice.Frame{
				Type: voice.FrameText,
				Data: strings.TrimSpace(sentence),
				Metadata: map[string]any{
					"preemptive": true,
				},
			}

			// Keep the remainder in the buffer.
			buffer.Reset()
			if lastEnd+1 < len(content) {
				buffer.WriteString(content[lastEnd+1:])
			}
		}
	}

	// Flush any remaining text.
	if buffer.Len() > 0 {
		out <- voice.Frame{
			Type: voice.FrameText,
			Data: strings.TrimSpace(buffer.String()),
			Metadata: map[string]any{
				"preemptive": true,
				"final":      true,
			},
		}
	}

	return nil
}

func main() {
	feeder := NewPreemptiveTTSFeeder()

	// Chain: LLM → PreemptiveTTSFeeder → TTS
	// TTS starts generating audio for the first sentence while the
	// LLM is still producing the rest of the response.
	_ = feeder
	fmt.Println("Preemptive TTS: sentences are spoken as soon as they complete")
}
```

---

## Manage STT Jitter Buffers

**Problem:** Audio packets arrive at irregular intervals due to network jitter, causing STT input to be choppy or contain gaps.

**Solution:** Buffer incoming audio and deliver it at a steady rate to the STT processor. Network jitter is an inherent property of real-time audio delivery over the internet: packets that were sent at regular 20ms intervals arrive with variable spacing (sometimes bunched together, sometimes with gaps). Without a jitter buffer, these irregularities pass directly to the STT processor, causing transcription errors and audio artifacts. The buffer introduces a small, controlled delay to smooth out arrival times.

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/lookatitude/beluga-ai/voice"
)

// JitterBuffer smooths irregular audio arrival for STT processing.
type JitterBuffer struct {
	bufferDuration time.Duration
	sampleRate     int
}

func NewJitterBuffer(bufferDuration time.Duration, sampleRate int) *JitterBuffer {
	return &JitterBuffer{
		bufferDuration: bufferDuration,
		sampleRate:     sampleRate,
	}
}

func (jb *JitterBuffer) Process(ctx context.Context, in <-chan voice.Frame, out chan<- voice.Frame) error {
	defer close(out)

	// Buffer incoming frames.
	buffer := make([]voice.Frame, 0, 100)
	ticker := time.NewTicker(20 * time.Millisecond) // 20ms output intervals.
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case frame, ok := <-in:
			if !ok {
				// Flush remaining buffer.
				for _, f := range buffer {
					out <- f
				}
				return nil
			}
			if frame.Type == voice.FrameAudio {
				buffer = append(buffer, frame)
			} else {
				out <- frame // Pass through non-audio frames immediately.
			}

		case <-ticker.C:
			// Deliver buffered frames at a steady rate.
			if len(buffer) > 0 {
				out <- buffer[0]
				buffer = buffer[1:]
			}
		}
	}
}

func main() {
	jitter := NewJitterBuffer(100*time.Millisecond, 16000)
	fmt.Printf("Jitter buffer: %v at %d Hz\n", jitter.bufferDuration, jitter.sampleRate)
}
```

---

## Overcome Background Noise

**Problem:** Background noise causes false-positive speech detection and degrades STT accuracy.

**Solution:** Add a noise gate FrameProcessor before VAD that filters low-energy audio. The noise gate is a signal processing technique borrowed from audio engineering: audio below a threshold energy level is silently dropped. By placing this processor before VAD in the pipeline chain, you prevent ambient noise from triggering voice activity detection and improve the signal-to-noise ratio of audio reaching the STT provider. The RMS (root mean square) calculation provides a reliable measure of audio energy that works across different microphone gains and environments.

```go
package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"

	"github.com/lookatitude/beluga-ai/voice"
)

// NoiseGate filters audio frames below an energy threshold.
type NoiseGate struct {
	threshold float64 // RMS threshold (0.0 to 1.0).
}

func NewNoiseGate(threshold float64) *NoiseGate {
	return &NoiseGate{threshold: threshold}
}

func (ng *NoiseGate) Process(ctx context.Context, in <-chan voice.Frame, out chan<- voice.Frame) error {
	defer close(out)

	for frame := range in {
		if frame.Type != voice.FrameAudio {
			out <- frame
			continue
		}

		// Calculate RMS energy of the audio frame.
		data, ok := frame.Data.([]byte)
		if !ok || len(data) < 2 {
			continue
		}

		rms := calculateRMS(data)
		if rms >= ng.threshold {
			out <- frame // Audio is above noise floor.
		}
		// Below threshold: frame is silently dropped (noise gate closed).
	}
	return nil
}

func calculateRMS(pcm16 []byte) float64 {
	samples := len(pcm16) / 2
	if samples == 0 {
		return 0
	}

	var sumSquares float64
	for i := 0; i < len(pcm16)-1; i += 2 {
		sample := int16(binary.LittleEndian.Uint16(pcm16[i:]))
		normalized := float64(sample) / 32768.0
		sumSquares += normalized * normalized
	}

	return math.Sqrt(sumSquares / float64(samples))
}

func main() {
	gate := NewNoiseGate(0.02) // Filter below 2% RMS energy.

	// Insert before VAD in the pipeline.
	pipeline := voice.Chain(
		gate,
		// vadProcessor,
		// sttProcessor,
	)

	_ = pipeline
	fmt.Println("Noise gate active: threshold=0.02 RMS")
}
```

---

## Multi-Speaker Dialogue Synthesis

**Problem:** Your application involves multiple AI characters speaking in a conversation. Each needs a distinct voice.

**Solution:** Route TTS requests to different voice configurations based on speaker identity. This router sits between the LLM output and TTS in the pipeline, inspecting frame metadata to determine which speaker is talking and dispatching to the appropriate TTS processor. A fallback processor handles cases where the speaker ID is unknown, ensuring the pipeline never silently drops frames.

```go
package main

import (
	"context"
	"fmt"

	"github.com/lookatitude/beluga-ai/voice"
)

// SpeakerRouter routes text frames to the correct TTS voice based on speaker metadata.
type SpeakerRouter struct {
	speakers map[string]voice.FrameProcessor // speaker_id → TTS processor
	fallback voice.FrameProcessor
}

func NewSpeakerRouter(fallback voice.FrameProcessor) *SpeakerRouter {
	return &SpeakerRouter{
		speakers: make(map[string]voice.FrameProcessor),
		fallback: fallback,
	}
}

func (r *SpeakerRouter) AddSpeaker(id string, tts voice.FrameProcessor) {
	r.speakers[id] = tts
}

func (r *SpeakerRouter) Process(ctx context.Context, in <-chan voice.Frame, out chan<- voice.Frame) error {
	defer close(out)

	for frame := range in {
		if frame.Type != voice.FrameText {
			out <- frame
			continue
		}

		// Look up the speaker from frame metadata.
		speakerID, _ := frame.Metadata["speaker"].(string)
		tts, ok := r.speakers[speakerID]
		if !ok {
			tts = r.fallback
		}

		// Process through the speaker-specific TTS.
		tempIn := make(chan voice.Frame, 1)
		tempOut := make(chan voice.Frame, 1)
		tempIn <- frame
		close(tempIn)

		go tts.Process(ctx, tempIn, tempOut)

		for result := range tempOut {
			result.Metadata["speaker"] = speakerID
			out <- result
		}
	}

	return nil
}

func main() {
	router := NewSpeakerRouter(nil)
	// router.AddSpeaker("narrator", narratorTTS)
	// router.AddSpeaker("character-a", characterATTS)
	// router.AddSpeaker("character-b", characterBTTS)

	fmt.Printf("Speaker router configured with %d voices\n", len(router.speakers))
}
```

---

## SSML Tuning for Natural Speech

**Problem:** TTS output sounds robotic. You need fine control over emphasis, pauses, and prosody.

**Solution:** Transform text frames into SSML-annotated text before sending to TTS. SSML (Speech Synthesis Markup Language) is a W3C standard that most TTS providers support for fine-grained speech control. By inserting an SSML annotator as a FrameProcessor between the LLM output and TTS, you can automatically add natural pauses after sentences, emphasize important words, and control speaking rate without modifying the LLM prompt or TTS configuration.

```go
package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/lookatitude/beluga-ai/voice"
)

// SSMLAnnotator transforms plain text into SSML for more natural TTS output.
type SSMLAnnotator struct {
	defaultRate  string // "slow", "medium", "fast"
	defaultPitch string // "low", "medium", "high"
}

func NewSSMLAnnotator(rate, pitch string) *SSMLAnnotator {
	return &SSMLAnnotator{defaultRate: rate, defaultPitch: pitch}
}

func (a *SSMLAnnotator) Process(ctx context.Context, in <-chan voice.Frame, out chan<- voice.Frame) error {
	defer close(out)

	for frame := range in {
		if frame.Type != voice.FrameText {
			out <- frame
			continue
		}

		text, _ := frame.Data.(string)
		ssml := a.annotate(text)

		frame.Data = ssml
		frame.Metadata["format"] = "ssml"
		out <- frame
	}
	return nil
}

func (a *SSMLAnnotator) annotate(text string) string {
	var b strings.Builder
	b.WriteString("<speak>")
	b.WriteString(fmt.Sprintf(`<prosody rate="%s" pitch="%s">`, a.defaultRate, a.defaultPitch))

	// Add pauses after sentences.
	text = strings.ReplaceAll(text, ". ", `.</prosody><break time="300ms"/><prosody rate="`+a.defaultRate+`" pitch="`+a.defaultPitch+`">`)

	// Add emphasis for text in *asterisks*.
	text = strings.ReplaceAll(text, "*", "<emphasis>")

	b.WriteString(text)
	b.WriteString("</prosody></speak>")
	return b.String()
}

func main() {
	annotator := NewSSMLAnnotator("medium", "medium")
	result := annotator.annotate("Hello. This is *important* information. Please listen carefully.")
	fmt.Println(result)
}
```

---

## VAD Sensitivity Profiles

**Problem:** Different environments need different VAD sensitivity. A quiet office needs high sensitivity; a noisy call center needs lower sensitivity to avoid false triggers.

**Solution:** Configure VAD with environment-specific profiles. Voice Activity Detection sensitivity is a fundamental trade-off: too sensitive and background noise triggers false positives, too aggressive and soft-spoken users get cut off. Rather than tuning a single set of magic numbers, defining named profiles lets you switch behavior based on detected environment, user preference, or deployment context. This approach also makes A/B testing straightforward since each profile is a discrete, testable configuration.

```go
package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/lookatitude/beluga-ai/voice"
)

// VADProfile configures VAD for a specific environment.
type VADProfile struct {
	Name             string
	EnergyThreshold  float64
	SpeechPadding    int // Frames of padding around speech.
	SilenceThreshold int // Consecutive silence frames to end speech.
}

// Predefined profiles for common environments.
var (
	ProfileQuietRoom = VADProfile{
		Name:             "quiet",
		EnergyThreshold:  0.01,
		SpeechPadding:    3,
		SilenceThreshold: 15,
	}
	ProfileOffice = VADProfile{
		Name:             "office",
		EnergyThreshold:  0.03,
		SpeechPadding:    5,
		SilenceThreshold: 20,
	}
	ProfileNoisyCallCenter = VADProfile{
		Name:             "noisy",
		EnergyThreshold:  0.08,
		SpeechPadding:    8,
		SilenceThreshold: 30,
	}
)

func createVAD(profile VADProfile) (voice.VAD, error) {
	return voice.NewVAD("energy", map[string]any{
		"threshold":         profile.EnergyThreshold,
		"speech_padding":    profile.SpeechPadding,
		"silence_threshold": profile.SilenceThreshold,
	})
}

func main() {
	ctx := context.Background()

	profile := ProfileOffice
	vad, err := createVAD(profile)
	if err != nil {
		slog.Error("VAD creation failed", "error", err)
		return
	}

	// Test with a sample audio frame.
	sample := make([]byte, 640) // 20ms at 16kHz mono PCM16.
	result, err := vad.DetectActivity(ctx, sample)
	if err != nil {
		slog.Error("VAD failed", "error", err)
		return
	}

	fmt.Printf("Profile: %s, IsSpeech: %v, Confidence: %.2f\n",
		profile.Name, result.IsSpeech, result.Confidence)
}
```

---

## Voice Pipeline with Hooks

**Problem:** You need to monitor and respond to pipeline events (speech start/end, transcripts, errors) without modifying processor code.

**Solution:** Use `voice.Hooks` and `voice.ComposeHooks` for event callbacks. Beluga AI follows the hooks pattern consistently across all packages: optional function fields where `nil` means "skip this hook." This lets you layer observability, error handling, and business logic onto the voice pipeline without touching processor implementations. `ComposeHooks` merges multiple hook sets into one, so monitoring and error handling can be defined independently and combined at configuration time.

```go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/lookatitude/beluga-ai/voice"
)

func main() {
	// Monitoring hooks for observability.
	monitoring := voice.Hooks{
		OnSpeechStart: func(ctx context.Context) {
			slog.Info("speech started", "time", time.Now())
		},
		OnSpeechEnd: func(ctx context.Context) {
			slog.Info("speech ended", "time", time.Now())
		},
		OnTranscript: func(ctx context.Context, text string) {
			slog.Info("transcript", "text", text)
		},
		OnResponse: func(ctx context.Context, text string) {
			slog.Info("agent response", "text", text)
		},
	}

	// Error handling hooks.
	errorHandling := voice.Hooks{
		OnError: func(ctx context.Context, err error) error {
			slog.Error("pipeline error", "error", err)
			// Return nil to suppress, return err to propagate.
			return err
		},
	}

	// Compose multiple hook sets.
	hooks := voice.ComposeHooks(monitoring, errorHandling)
	_ = hooks // Pass to pipeline: voice.WithPipelineHooks(hooks)

	fmt.Println("Voice hooks configured: monitoring + error handling")
}
```
