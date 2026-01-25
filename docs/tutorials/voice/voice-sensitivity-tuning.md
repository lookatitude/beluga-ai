# Tuning VAD and Turn Detection

In this tutorial, you'll learn how to optimize Voice Activity Detection (VAD) and Turn Detection to make your voice agents feel responsive and natural.

## Learning Objectives

- ✅ Configure Silero VAD sensitivity
- ✅ Set silence thresholds for Turn Detection
- ✅ Implement "Interrupt-to-Stop" patterns
- ✅ Balance speed vs. false-triggers

## Prerequisites

- [Real-time STT Streaming](./voice-stt-realtime-streaming.md)
- Basic understanding of audio frames

## The Challenge: Human Flow

If detection is too sensitive: The AI interrupts you when you breathe.
If detection is too slow: The AI waits 3 seconds after you stop before answering (feels awkward).

## Step 1: Configuring VAD (Voice Activity Detection)

VAD decides if a chunk of audio is "Human Speech" or "Background Noise".
```go
package main

import (
    "github.com/lookatitude/beluga-ai/pkg/voice/vad"
    "github.com/lookatitude/beluga-ai/pkg/voice/vad/providers/silero"
)

func main() {
    config := &silero.Config{
        Threshold: 0.5, // 0.0 to 1.0. Increase to be more strict.
        MinSpeechDuration: 250, // ms. Ignore short pops/clicks.
    }
    
    detector, _ := silero.NewDetector(config)
}
```

## Step 2: Turn Detection (End of Sentence)

Decides when the user is *done* talking.

```
import "github.com/lookatitude/beluga-ai/pkg/voice/turndetection"

go
```go
config := &turndetection.Config{
    // Wait for 800ms of silence before replying
    SilenceDuration: 800, 
    // If the user has talked for 10s, reply anyway (Long-utterance)
    MaxUtteranceDuration: 10000,
}
```

## Step 3: Handling Interruptions

If the agent is speaking and VAD detects human speech, the agent should stop immediately.
// Pseudo-code for interruption logic
session.OnUserSpeechStart(func() \{
```text
    if agent.IsSpeaking() \{
        agent.StopSpeaking()
        agent.ClearQueuedAudio()
    }
})


## Step 4: Environment Sensitivity

In noisy environments (cafes), you need higher thresholds.
if backgroundNoiseLevel > -40dB \{
    vadConfig.Threshold = 0.8
}
```

## Verification

1. Run your agent.
2. Hum or cough. The AI should NOT react.
3. Say "Hello". The AI should react within ~1 second.
4. Interrupt the AI mid-sentence. It should stop immediately.

## Next Steps

- **[Native S2S with Amazon Nova](./voice-s2s-amazon-nova.md)** - Models with built-in VAD.
- **[Voice Backends (LiveKit)](./voice-backends-livekit-vapi.md)** - Server-side VAD.
