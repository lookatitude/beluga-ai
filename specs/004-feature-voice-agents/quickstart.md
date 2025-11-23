# Quickstart Guide: Voice Agents

**Feature**: Voice Agents Framework  
**Date**: 2025-01-27  
**Status**: Implementation Guide

## Overview

This guide provides step-by-step instructions for creating and using a voice agent with the Beluga AI Voice Framework.

---

## Prerequisites

1. Go 1.21 or later
2. Beluga AI framework packages installed
3. API keys for chosen providers (Deepgram, OpenAI, etc.)
4. WebRTC transport configured (if using WebRTC)

---

## Step 1: Install Dependencies

```bash
go get github.com/lookatitude/beluga-ai/pkg/voice
go get github.com/lookatitude/beluga-ai/pkg/agents
go get github.com/lookatitude/beluga-ai/pkg/llms
go get github.com/lookatitude/beluga-ai/pkg/memory
```

---

## Step 2: Configure Providers

### 2.1 Create STT Provider

```go
import (
    "github.com/lookatitude/beluga-ai/pkg/voice/stt"
    "github.com/lookatitude/beluga-ai/pkg/config"
)

// Load configuration
cfg, err := config.LoadConfig()
if err != nil {
    log.Fatal(err)
}

// Create STT provider
sttProvider, err := stt.NewProvider(ctx, "deepgram", stt.Config{
    APIKey:   cfg.GetString("voice.stt.deepgram.api_key"),
    Model:    "nova-3",
    Language: "en",
})
if err != nil {
    log.Fatal(err)
}
```

### 2.2 Create TTS Provider

```go
import "github.com/lookatitude/beluga-ai/pkg/voice/tts"

// Create TTS provider
ttsProvider, err := tts.NewProvider(ctx, "openai", tts.Config{
    APIKey: cfg.GetString("voice.tts.openai.api_key"),
    Model:  "tts-1",
    Voice:  "nova",
})
if err != nil {
    log.Fatal(err)
}
```

### 2.3 Create VAD Provider

```go
import "github.com/lookatitude/beluga-ai/pkg/voice/vad"

// Create VAD provider
vadProvider, err := vad.NewProvider(ctx, "silero", vad.Config{
    ModelPath: "./models/silero_vad.onnx",
})
if err != nil {
    log.Fatal(err)
}
```

---

## Step 3: Create Beluga AI Agent

```go
import (
    "github.com/lookatitude/beluga-ai/pkg/agents"
    "github.com/lookatitude/beluga-ai/pkg/llms"
)

// Create LLM
llm, err := llms.NewProvider(ctx, "openai", llms.Config{
    APIKey:    cfg.GetString("llm.openai.api_key"),
    ModelName: "gpt-4o-mini",
})
if err != nil {
    log.Fatal(err)
}

// Create agent
agent, err := agents.NewReActAgent(ctx, "voice-agent", llm, nil, 
    "You are a helpful voice assistant.")
if err != nil {
    log.Fatal(err)
}
```

---

## Step 4: Create Memory Provider

```go
import "github.com/lookatitude/beluga-ai/pkg/memory"

// Create memory provider
memoryProvider, err := memory.NewProvider(ctx, "inmemory", memory.Config{})
if err != nil {
    log.Fatal(err)
}
```

---

## Step 5: Create Transport

```go
import "github.com/lookatitude/beluga-ai/pkg/voice/transport/webrtc"

// Create WebRTC transport
transport, err := webrtc.NewTransport(ctx, webrtc.Config{
    RoomName:      "voice-session-1",
    ParticipantID: "agent-1",
    ServerURL:     "wss://your-livekit-server.com",
    APIKey:        cfg.GetString("livekit.api_key"),
    APISecret:     cfg.GetString("livekit.api_secret"),
})
if err != nil {
    log.Fatal(err)
}
```

---

## Step 6: Create Voice Session

```go
import (
    "github.com/lookatitude/beluga-ai/pkg/voice/session"
    "github.com/lookatitude/beluga-ai/pkg/voice/turndetection"
)

// Create turn detector
turnDetector, err := turndetection.NewDetector(ctx, "heuristic", turndetection.Config{
    MinSilenceDuration: 500 * time.Millisecond,
})
if err != nil {
    log.Fatal(err)
}

// Create voice session
voiceSession, err := session.NewVoiceSession(ctx, session.VoiceSessionConfig{
    Agent:        agent,
    STT:          sttProvider,
    TTS:          ttsProvider,
    VAD:          vadProvider,
    TurnDetector: turnDetector,
    Transport:    transport,
    Memory:       memoryProvider,
    Options: &session.VoiceOptions{
        AllowInterruptions:    true,
        PreemptiveGeneration:  true,
        StreamingTTS:          true,
        StreamingAgent:        true,
        UserAwayTimeout:       30 * time.Second,
    },
})
if err != nil {
    log.Fatal(err)
}
```

---

## Step 7: Start Voice Session

```go
// Set up state change callback
voiceSession.OnStateChanged(func(state session.SessionState) {
    log.Printf("Session state changed: %s", state)
})

// Start the session
ctx := context.Background()
if err := voiceSession.Start(ctx); err != nil {
    log.Fatal(err)
}

log.Println("Voice session started and listening...")
```

---

## Step 8: Handle Audio Input

The transport will automatically handle audio input via callbacks. The voice session processes audio through the pipeline:

1. Audio received → VAD detects speech
2. Speech → STT converts to text
3. Text → Turn detector identifies turn end
4. Turn end → Agent processes transcript
5. Agent response → TTS converts to speech
6. Speech → Transport plays to user

---

## Step 9: Programmatic Interaction (Optional)

You can also interact programmatically:

```go
// Say something to the user
handle, err := voiceSession.Say(ctx, "Hello! How can I help you?")
if err != nil {
    log.Printf("Error saying text: %v", err)
} else {
    // Wait for playback to complete
    handle.WaitForPlayout(ctx)
}

// Process audio manually (if not using transport callbacks)
audioData := []byte{...} // Your audio data
if err := voiceSession.ProcessAudio(ctx, audioData); err != nil {
    log.Printf("Error processing audio: %v", err)
}
```

---

## Step 10: Cleanup

```go
// Stop the session gracefully
if err := voiceSession.Stop(ctx); err != nil {
    log.Printf("Error stopping session: %v", err)
}
```

---

## Complete Example

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/lookatitude/beluga-ai/pkg/agents"
    "github.com/lookatitude/beluga-ai/pkg/config"
    "github.com/lookatitude/beluga-ai/pkg/llms"
    "github.com/lookatitude/beluga-ai/pkg/memory"
    "github.com/lookatitude/beluga-ai/pkg/voice/session"
    "github.com/lookatitude/beluga-ai/pkg/voice/stt"
    "github.com/lookatitude/beluga-ai/pkg/voice/tts"
    "github.com/lookatitude/beluga-ai/pkg/voice/transport/webrtc"
    "github.com/lookatitude/beluga-ai/pkg/voice/turndetection"
    "github.com/lookatitude/beluga-ai/pkg/voice/vad"
)

func main() {
    ctx := context.Background()

    // Load config
    cfg, err := config.LoadConfig()
    if err != nil {
        log.Fatal(err)
    }

    // Create providers
    sttProvider, _ := stt.NewProvider(ctx, "deepgram", stt.Config{
        APIKey: cfg.GetString("voice.stt.deepgram.api_key"),
        Model:  "nova-3",
    })

    ttsProvider, _ := tts.NewProvider(ctx, "openai", tts.Config{
        APIKey: cfg.GetString("voice.tts.openai.api_key"),
        Model:  "tts-1",
        Voice:  "nova",
    })

    vadProvider, _ := vad.NewProvider(ctx, "silero", vad.Config{})

    llm, _ := llms.NewProvider(ctx, "openai", llms.Config{
        APIKey:    cfg.GetString("llm.openai.api_key"),
        ModelName: "gpt-4o-mini",
    })

    agent, _ := agents.NewReActAgent(ctx, "voice-agent", llm, nil,
        "You are a helpful voice assistant.")

    memoryProvider, _ := memory.NewProvider(ctx, "inmemory", memory.Config{})

    transport, _ := webrtc.NewTransport(ctx, webrtc.Config{
        RoomName:  "voice-session-1",
        ServerURL: "wss://your-livekit-server.com",
        APIKey:    cfg.GetString("livekit.api_key"),
        APISecret: cfg.GetString("livekit.api_secret"),
    })

    turnDetector, _ := turndetection.NewDetector(ctx, "heuristic", turndetection.Config{})

    // Create voice session
    voiceSession, err := session.NewVoiceSession(ctx, session.VoiceSessionConfig{
        Agent:        agent,
        STT:          sttProvider,
        TTS:          ttsProvider,
        VAD:          vadProvider,
        TurnDetector: turnDetector,
        Transport:    transport,
        Memory:       memoryProvider,
        Options: &session.VoiceOptions{
            AllowInterruptions:   true,
            PreemptiveGeneration: true,
        },
    })
    if err != nil {
        log.Fatal(err)
    }

    // Start session
    if err := voiceSession.Start(ctx); err != nil {
        log.Fatal(err)
    }

    log.Println("Voice session started. Press Ctrl+C to stop.")

    // Keep running
    select {}
}
```

---

## Testing the Setup

### Validation Steps

1. **Check Session State**:
   ```go
   state := voiceSession.GetState()
   if state != session.SessionStateListening {
       log.Printf("Expected Listening state, got: %s", state)
   }
   ```

2. **Test Say()**:
   ```go
   handle, err := voiceSession.Say(ctx, "Test message")
   if err != nil {
       log.Fatal(err)
   }
   handle.WaitForPlayout(ctx)
   ```

3. **Test Audio Processing**:
   ```go
   // Simulate audio input
   audioData := make([]byte, 1024)
   if err := voiceSession.ProcessAudio(ctx, audioData); err != nil {
       log.Printf("Error: %v", err)
   }
   ```

---

## Next Steps

1. **Customize Options**: Adjust `VoiceOptions` for your use case
2. **Add Tools**: Extend agent with custom tools
3. **Configure Fallbacks**: Set up fallback providers for reliability
4. **Monitor Performance**: Use observability metrics to track latency
5. **Scale**: Deploy multiple instances for concurrent sessions

---

## Troubleshooting

### Common Issues

1. **Provider Initialization Fails**:
   - Check API keys in configuration
   - Verify network connectivity
   - Check provider-specific requirements

2. **Transport Connection Fails**:
   - Verify WebRTC server URL and credentials
   - Check firewall/network settings
   - Ensure DTLS/TLS certificates are valid

3. **High Latency**:
   - Check network latency to providers
   - Consider using closer provider regions
   - Enable preemptive generation
   - Use streaming where possible

4. **Audio Quality Issues**:
   - Verify audio format matches provider requirements
   - Check sample rate and bit depth
   - Enable noise cancellation if needed

---

## Additional Resources

- [Voice Package Documentation](../../../pkg/voice/README.md)
- [Agent Integration Guide](../../../docs/guides/voice-agents.md)
- [Provider Configuration](../../../docs/guides/voice-providers.md)
- [Performance Tuning](../../../docs/guides/voice-performance.md)

