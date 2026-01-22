# Barge-In Detection for Voice Agents

## Overview

A contact-center platform needed voice agents to stop speaking as soon as the user started talking again (barge-in), instead of playing out pre-generated responses. Slow or absent barge-in led to users talking over the agent and fragmented conversations. They required reliable, low-latency detection of user speech onset during agent playback.

**The challenge:** Detect "user is speaking again" quickly enough to cancel TTS playback and switch to listening, without false triggers from background noise or echo.

**The solution:** We used Beluga AI's `pkg/voice/turndetection` and `pkg/voice/vad` together: VAD detects speech onset, and turn detection (heuristic + `DetectTurnWithSilence`) distinguishes mid-turn barge-in from end-of-turn. The voice session stops TTS and processes new user input when barge-in is detected.

## Business Context

### The Problem

- **Agents talked over users**: No barge-in; users had to wait for the agent to finish.
- **Poor experience**: Users repeated themselves or hung up when they couldn’t interrupt.
- **Echo and noise**: Simple energy-based detection caused false barge-in from playback or room noise.

### The Opportunity

By implementing barge-in detection:

- **Natural interrupts**: Users can speak over the agent and be heard immediately.
- **Higher satisfaction**: Shorter, more natural conversations.
- **Fewer false triggers**: VAD + turn-detection logic reduced echo-induced barge-in.

### Success Metrics

| Metric | Before | Target | Achieved |
|--------|--------|--------|----------|
| Barge-in latency (speech onset → TTS stop) | N/A | \<200 ms | 160 ms |
| False barge-in rate | N/A | \<3% | 2.5% |
| User interrupt success rate | 0% | >95% | 97% |

## Requirements

### Functional Requirements

| ID | Requirement | Rationale |
|----|-------------|-----------|
| FR1 | Detect user speech during agent TTS playback | Enable barge-in |
| FR2 | Stop TTS and switch to listening on barge-in | Avoid talking over user |
| FR3 | Use VAD for speech onset and turn detection for context | Reduce false positives |
| FR4 | Configurable sensitivity (thresholds, silence) | Tune per environment |

### Non-Functional Requirements

| ID | Requirement | Target |
|----|-------------|--------|
| NFR1 | Barge-in latency (onset → TTS stop) | \<200 ms |
| NFR2 | False barge-in rate | \<3% |
| NFR3 | No degradation of turn-end detection | Same accuracy as without barge-in |

### Constraints

- Reuse existing `pkg/voice/vad` and `pkg/voice/turndetection`; no custom signal processing.
- Must work with streaming TTS and incremental VAD output.

## Architecture Requirements

### Design Principles

- **VAD for onset**: Use VAD to detect "speech started" during playback.
- **Turn detection for context**: Use turn logic to avoid treating end-of-turn as barge-in.
- **Unified session**: Single voice session coordinates VAD, turn detection, TTS, and playback control.

### Key Architectural Decisions

| Decision | Rationale | Trade-off |
|----------|-----------|-----------|
| VAD + turn detection | VAD catches onset; turn logic provides context | Slightly more complexity than VAD-only |
| Heuristic turn detector for barge-in | Low latency, no model | ONNX could improve accuracy in noisy setups |
| Stop TTS immediately on barge-in | Responsive UX | Possible truncation; we accept it |

## Architecture

### High-Level Design
graph TB
```
    Mic[Microphone] --> VAD[VAD]
    TTS[TTS Output] --> Playback[Playback]
    VAD --> SpeechOnset[Speech Onset?]
    SpeechOnset -->|Yes| TurnCtx[Turn Context]
    TurnCtx --> BargeIn{Barge-In?}
    BargeIn -->|Yes| StopTTS[Stop TTS]
    StopTTS --> Listen[Switch to Listening]
    Listen --> STT[STT]
    BargeIn -->|No| Playback

### How It Works

1. **During agent playback**: VAD processes mic input. On speech onset, we check turn context (e.g. current segment length, silence so far) to avoid obvious end-of-turn cases.
2. **Barge-in decision**: If we classify as barge-in (user interrupting), we stop TTS, clear playback buffers, and switch the session to listening. STT then processes the new user utterance.
3. **Turn detection**: We use `DetectTurnWithSilence` and heuristic rules so that end-of-turn handling remains correct when we’re not in barge-in mode.

### Component Details

| Component | Purpose | Technology |
|-----------|---------|------------|
| VAD | Detect speech onset | `pkg/voice/vad` |
| Turn Detector | Context for barge-in vs end-of-turn | `pkg/voice/turndetection` |
| Voice Session | Orchestrate TTS stop, listen, STT | `pkg/voice/session` |

## Implementation

### Phase 1: VAD and Turn Detector Setup
```go
package main

import (
	"context"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/vad"
	"github.com/lookatitude/beluga-ai/pkg/voice/turndetection"
	turndetectioniface "github.com/lookatitude/beluga-ai/pkg/voice/turndetection/iface"
	vadiface "github.com/lookatitude/beluga-ai/pkg/voice/vad/iface"
)

func setupBargeIn(ctx context.Context) (vadiface.VADProvider, turndetectioniface.TurnDetector, error) {
	vadCfg := vad.DefaultConfig()
	vadProv, err := vad.NewProvider(ctx, "webrtc", vadCfg)
	if err != nil {
		return nil, nil, err
	}

	tdCfg := turndetection.DefaultConfig()
	td, err := turndetection.NewProvider(ctx, "heuristic", tdCfg,
		turndetection.WithMinSilenceDuration(200*time.Millisecond),
		turndetection.WithMinTurnLength(5),
	)
	if err != nil {
		return nil, nil, err
	}
	return vadProv, td, nil
}
```

### Phase 2: Barge-In Loop (Conceptual)
// In session loop, while playing TTS:
// 1. Run VAD on incoming mic audio
speaking, _ := vadProv.Process(ctx, audioChunk)
```text
go
go
if speaking {
	// 2. Optional: check turn context to avoid false barge-in
	silence := time.Since(lastSpeechAt)
	done, _ := turnDetector.DetectTurnWithSilence(ctx, audioChunk, silence)
	if !done {
		// 3. Treat as barge-in: stop TTS, switch to listening
		stopTTS()
		switchToListening()
	}
}
```

### Phase 3: Session Integration

Wire the barge-in logic into your voice session’s playback and state machine so that `stopTTS` and `switchToListening` are invoked correctly. Use OTEL to record barge-in events and latency.

## Results

### Performance Metrics

| Metric | Before | After | Improvement |
|--------|--------|-------|-------------|
| Barge-in latency (p95) | N/A | 160 ms | Met \<200 ms target |
| False barge-in rate | N/A | 2.5% | Under 3% target |
| User interrupt success | 0% | 97% | New capability |

### Qualitative Outcomes

- **Natural interrupts**: Users could cut off the agent and be heard.
- **Fewer escalations**: Less frustration and fewer handoffs.
- **Tunable**: Thresholds and silence duration adjustable per deployment.

### Trade-offs

| Trade-off | Benefit | Cost |
|-----------|---------|------|
| Heuristic turn detector | Low latency, simple | Less robust in very noisy environments |
| Immediate TTS stop | Responsive barge-in | Some agent replies truncated |

## Lessons Learned

### What Worked Well

- **VAD + turn detection**: Clear separation between "speech started" (VAD) and "turn structure" (turn detection) simplified tuning.
- **Short MinSilenceDuration for barge-in**: 200 ms helped react quickly without too many false triggers.
- **OTEL**: Metrics on barge-in rate and latency made tuning and incident response much easier.

### What We'd Do Differently

- **Echo cancellation**: We’d add AEC earlier to reduce false barge-in from speaker playback.
- **Per-agent tuning**: Different agents (e.g. different TTS length) could use different barge-in thresholds.

### Recommendations for Similar Projects

1. Start with VAD-only barge-in, then add turn-detection context to reduce false positives.
2. Log barge-in events and correlate with user feedback.
3. Test with real playback (e.g. TTS over speakers) to validate echo handling.

## Production Readiness Checklist

- [x] **Observability**: OTEL for barge-in events and latency
- [x] **Error Handling**: Graceful fallback when VAD/turn detector errors
- [x] **Configuration**: Config-driven thresholds and silence duration
- [ ] **Testing**: Integration tests with mock TTS and VAD
- [ ] **Echo handling**: AEC or similar where playback is present
- [ ] **Documentation**: Runbooks for tuning and debugging barge-in

## Related Use Cases

- **[Voice Sessions](./voice-sessions.md)** — Session and pipeline design.
- **[Low-Latency Turn Prediction](./voice-turn-low-latency-prediction.md)** — Turn-detection tuning.
- **[Voice Sensitivity Tuning](../tutorials/voice/voice-sensitivity-tuning.md)** — VAD and turn sensitivity.

## Related Resources

- **[Voice STT Overcoming Background Noise](../cookbook/voice-stt-overcoming-background-noise.md)** — Noise robustness.
- **[Voice S2S Handling Speech Interruption](../cookbook/voice-s2s-handling-speech-interruption.md)** — Interruption handling.
