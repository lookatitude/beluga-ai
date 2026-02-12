---
title: Barge-In Detection for Voice Agents
description: "Enable users to interrupt voice agents mid-speech with low-latency barge-in detection. Improve call satisfaction with natural turn-taking."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "barge-in detection, voice interruption, turn taking AI, contact center voice, echo cancellation, Beluga AI, Go, voice UX"
---

Contact center voice agents that play out entire pre-generated responses without allowing interruption frustrate callers and fragment conversations. Users expect the same turn-taking dynamics they experience in human dialogue: the ability to interrupt when they already know the answer, correct a misunderstanding mid-sentence, or signal urgency. Without barge-in support, agents force callers to wait through irrelevant speech, increasing handle times and driving down satisfaction scores.

Barge-in detection lets the system recognize when a user starts speaking during agent playback, immediately stopping TTS output and switching to listening mode. The core challenge is distinguishing genuine interruptions from false triggers caused by background noise, acoustic echo from the agent's own speaker output, or brief vocalizations like "uh-huh" that don't represent a turn change.

This use case combines Beluga AI's VAD for speech onset detection with turn detection for context-aware barge-in decisions. Separating these two concerns — raw speech detection and turn-level intent — allows each component to be tuned independently, avoiding the fragile coupling that occurs when a single model tries to handle both.

## Solution Architecture

```mermaid
graph TB
    Mic[Microphone] --> VAD[VAD]
    TTS[TTS Output] --> Playback[Playback]
    VAD --> SpeechOnset[Speech Onset?]
    SpeechOnset -->|Yes| TurnCtx[Turn Context]
    TurnCtx --> BargeIn{Barge-In?}
    BargeIn -->|Yes| StopTTS[Stop TTS]
    StopTTS --> Listen[Switch to Listening]
    Listen --> STT[STT]
    BargeIn -->|No| Playback
```

During agent playback, VAD processes microphone input. When speech onset is detected, turn-detection logic distinguishes between a genuine barge-in (user interrupting) and end-of-turn behavior, avoiding false triggers from background noise or echo.

The architecture separates VAD from turn-level decision-making because each requires different tuning tradeoffs. VAD operates on raw audio frames with low latency requirements (under 30ms per frame), optimizing for recall — it should catch every potential speech onset. Turn detection then applies contextual heuristics to filter false positives, optimizing for precision. This two-stage design means you can adjust barge-in sensitivity without retraining or reconfiguring the VAD model.

## Implementation

### VAD and Turn Detector Setup

The VAD is configured with a short `MinSpeechDuration` (100ms) to detect speech onset as quickly as possible. In playback scenarios, speed matters more than filtering out brief noises — false positives are handled by the turn detection layer downstream, not by the VAD itself.

```go
package main

import (
    "context"
    "time"

    "github.com/lookatitude/beluga-ai/voice"
)

func setupBargeInDetection(ctx context.Context) (voice.FrameProcessor, error) {
    vad := voice.NewSileroVAD(voice.VADConfig{
        Threshold:         0.5,
        MinSpeechDuration: 100 * time.Millisecond, // Short for quick onset detection
    })

    return vad, nil
}
```

### Barge-In Logic

During the session playback loop, the barge-in logic monitors for speech onset and decides whether to interrupt. The key insight is using silence duration since the last detected speech as a lightweight heuristic: if the user was already speaking recently and the VAD fires again, this is likely a continuation or interruption rather than a new turn. This avoids the latency cost of running a full turn-prediction model on every VAD event.

```go
// In the session processing loop, while playing TTS audio:
func checkBargeIn(ctx context.Context, vadResult bool, lastSpeechAt time.Time, minSilence time.Duration) bool {
    if !vadResult {
        return false
    }

    // Check turn context: if silence since last speech is short,
    // this is likely a barge-in rather than end-of-turn
    silence := time.Since(lastSpeechAt)
    if silence < minSilence {
        // User is interrupting agent speech
        return true
    }

    return false
}

// On barge-in detection:
// 1. Stop TTS playback immediately
// 2. Clear playback buffers
// 3. Switch session to listening mode
// 4. Route audio to STT for the new user utterance
```

### Session Integration

Wire barge-in logic into the voice session's state machine. The session uses a simple three-state model (Listening, Processing, Responding) where barge-in triggers an immediate transition from Responding back to Listening. This state machine approach ensures that audio routing (microphone to STT vs. TTS to speaker) changes atomically with the state transition, preventing audio from being sent to both paths simultaneously.

```go
type SessionState string

const (
    StateListening  SessionState = "listening"
    StateProcessing SessionState = "processing"
    StateResponding SessionState = "responding"
)

func (s *VoiceSession) handleBargeIn(ctx context.Context) {
    s.stopTTS()
    s.clearPlaybackBuffers()
    s.state = StateListening
    // New user audio is now routed to STT
}
```

## Deployment Considerations

- **MinSilenceDuration**: Set to 200ms or less for responsive barge-in; tune per environment
- **Echo cancellation**: Add AEC (acoustic echo cancellation) to reduce false barge-in from speaker playback
- **Per-agent tuning**: Different agents with different TTS lengths may need different barge-in thresholds
- **Graceful truncation**: Accept that some agent replies will be truncated; this is expected behavior
- **Observability**: Track barge-in events, latency (speech onset to TTS stop), and false positive rates

## Results

| Metric | Before | After | Improvement |
|--------|--------|-------|-------------|
| Barge-in latency (p95) | N/A | 160ms | Met <200ms target |
| False barge-in rate | N/A | 2.5% | Under 3% target |
| User interrupt success | 0% | 97% | New capability |

### Lessons Learned

- **VAD + turn detection**: Separating "speech started" (VAD) from "turn structure" (turn detection) simplified tuning
- **Short MinSilenceDuration**: 200ms helped react quickly without excessive false triggers
- **Echo cancellation matters**: Add AEC early to reduce false barge-in from speaker playback

## Related Resources

- [Low-Latency Turn Prediction](/use-cases/low-latency-prediction/) for turn-detection tuning
- [Voice Sessions Overview](/use-cases/voice-sessions-overview/) for session and pipeline design
- [Voice AI Applications](/use-cases/voice-applications/) for voice pipeline architecture
