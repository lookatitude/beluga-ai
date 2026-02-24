---
title: Voice AI Applications in Go
description: "Build voice-enabled applications with STT, TTS, S2S, and composable frame-based pipelines. Production-ready Go voice system."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "voice AI application, STT TTS pipeline, speech-to-speech, frame processor, voice assistant Go, Beluga AI, voice use case"
---

Voice interfaces enable natural, hands-free interaction with AI systems. Unlike text-based interfaces that require a screen and keyboard, voice reaches users in situations where their hands and eyes are occupied — driving, cooking, operating equipment, or navigating a physical environment. The technical challenge is that voice processing involves multiple interdependent stages (audio capture, speech detection, transcription, response generation, synthesis) that must coordinate with sub-second latency to feel conversational.

Beluga AI provides a frame-based voice pipeline that composes STT (speech-to-text), TTS (text-to-speech), S2S (speech-to-speech), VAD (voice activity detection), and transport layers into flexible processing chains. The frame-based design was chosen over monolithic pipeline architectures because it allows each stage to be developed, tested, and swapped independently. A hotel concierge and a meeting transcription system share the same VAD and STT components but differ in downstream processing — the frame model makes this composition natural without framework-level abstraction leaks.

## Voice Pipeline Architecture

Beluga AI's voice system is built around the `FrameProcessor` interface. Each component (STT, TTS, VAD, turn detector) is a frame processor that reads frames from an input channel and writes processed frames to an output channel. Frame processors compose into pipelines using `voice.Chain()`.

This composable design follows the Unix pipe philosophy: each processor does one thing well, and complex behavior emerges from composition rather than configuration. You can insert a noise filter before VAD, add logging between stages, or swap a Deepgram STT for Whisper without touching any other component in the chain.

```
┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐
│  Audio   │───▶│   VAD   │───▶│   STT   │───▶│  Agent  │───▶│   TTS   │
│  Input   │    │ (Silero)│    │(Deepgram│    │ (LLM +  │    │(ElevenL │
│ (WebSocket)│  │         │    │ Whisper)│    │  Tools) │    │  OpenAI)│
└─────────┘    └─────────┘    └─────────┘    └─────────┘    └─────────┘
     Frames ──────▶ Frames ──────▶ Frames ──────▶ Frames ──────▶ Frames
```

## Frame Processing Basics

All voice components implement the `FrameProcessor` interface:

```go
import "github.com/lookatitude/beluga-ai/voice"

// FrameProcessor processes a stream of frames
type FrameProcessor interface {
    Process(ctx context.Context, in <-chan voice.Frame, out chan<- voice.Frame) error
}

// Compose processors into a pipeline
pipeline := voice.Chain(
    vadProcessor,
    sttProcessor,
    agentProcessor,
    ttsProcessor,
)
```

Frames carry typed data — audio, text, control signals, or images:

```go
// Audio frame from microphone or WebSocket
audioFrame := voice.NewAudioFrame(pcmData, 16000)

// Text frame from STT or agent
textFrame := voice.NewTextFrame("Hello, how can I help?")

// Control frame for signaling
endOfTurn := voice.NewControlFrame("end_of_turn")
```

## Use Case 1: Real-Time AI Concierge

A hotel concierge that handles guest inquiries, makes reservations, and provides information through natural voice conversations. Uses S2S (speech-to-speech) for the lowest possible latency — S2S processes audio end-to-end without separate STT/TTS stages, reducing the number of network round-trips and eliminating text as an intermediate representation.

```go
package main

import (
    "context"
    "fmt"

    "github.com/lookatitude/beluga-ai/schema"
    "github.com/lookatitude/beluga-ai/tool"
    "github.com/lookatitude/beluga-ai/voice"
    "github.com/lookatitude/beluga-ai/voice/s2s"

    _ "github.com/lookatitude/beluga-ai/voice/s2s/providers/openai"
)

func createConcierge(ctx context.Context) error {
    // Define concierge tools
    bookingTool := tool.NewFuncTool[BookingInput](
        "make_reservation",
        "Make a restaurant, spa, or activity reservation for the guest",
        func(ctx context.Context, input BookingInput) (*tool.Result, error) {
            confirmation, err := bookingSystem.Reserve(ctx, input)
            if err != nil {
                return tool.ErrorResult(err), nil
            }
            return tool.TextResult(fmt.Sprintf("Reservation confirmed: %s", confirmation)), nil
        },
    )

    infoTool := tool.NewFuncTool[InfoInput](
        "hotel_info",
        "Look up hotel information (hours, amenities, directions)",
        func(ctx context.Context, input InfoInput) (*tool.Result, error) {
            info, err := hotelDB.Lookup(ctx, input.Topic)
            if err != nil {
                return tool.ErrorResult(err), nil
            }
            return tool.TextResult(info), nil
        },
    )

    // Create S2S engine with tools
    engine, err := s2s.New("openai", nil)
    if err != nil {
        return fmt.Errorf("create s2s engine: %w", err)
    }

    // Start a streaming session
    session, err := engine.Start(ctx,
        s2s.WithVoice("nova"),
        s2s.WithInstructions("You are a luxury hotel concierge. Be warm, professional, "+
            "and helpful. Use the guest's name when known."),
        s2s.WithTools([]schema.ToolDefinition{
            tool.ToDefinition(bookingTool),
            tool.ToDefinition(infoTool),
        }),
    )
    if err != nil {
        return fmt.Errorf("start session: %w", err)
    }
    defer session.Close()

    // Process events from the session
    for event := range session.Recv() {
        switch event.Type {
        case s2s.EventAudioOutput:
            // Send audio to guest's device
            transport.SendAudio(event.Audio)
        case s2s.EventToolCall:
            // Execute tool and return result
            result := executeTool(ctx, event.ToolCall, bookingTool, infoTool)
            session.SendToolResult(ctx, result)
        case s2s.EventTranscript:
            // Log transcript for quality assurance
            log.Printf("Guest: %s", event.Text)
        }
    }

    return nil
}

type BookingInput struct {
    Type     string `json:"type" jsonschema:"enum=restaurant,spa,activity"`
    Date     string `json:"date" jsonschema:"description=Reservation date (YYYY-MM-DD)"`
    Time     string `json:"time" jsonschema:"description=Reservation time (HH:MM)"`
    Guests   int    `json:"guests"`
    Name     string `json:"name" jsonschema:"description=Guest name"`
}

type InfoInput struct {
    Topic string `json:"topic" jsonschema:"description=Topic to look up (pool hours, restaurant menu, etc.)"`
}
```

## Use Case 2: Meeting Transcription

A live meeting minutes generator that transcribes audio in real time, identifies speakers, and generates structured summaries. This use case chooses STT over S2S because the output is text (transcript and minutes), not audio — there is no need for speech synthesis on the output side.

```go
import (
    "github.com/lookatitude/beluga-ai/voice/stt"

    _ "github.com/lookatitude/beluga-ai/voice/stt/providers/deepgram"
)

func transcribeMeeting(ctx context.Context, audioStream iter.Seq2[[]byte, error]) error {
    engine, err := stt.New("deepgram", nil)
    if err != nil {
        return fmt.Errorf("create stt engine: %w", err)
    }

    // Stream transcription with speaker diarization
    transcripts := engine.TranscribeStream(ctx, audioStream,
        stt.WithLanguage("en"),
        stt.WithPunctuation(true),
        stt.WithDiarization(true),
    )

    var fullTranscript strings.Builder
    for event, err := range transcripts {
        if err != nil {
            return fmt.Errorf("transcription error: %w", err)
        }

        if event.IsFinal {
            fullTranscript.WriteString(event.Text + "\n")
            // Real-time display
            fmt.Printf("[%s] %s\n", event.Timestamp, event.Text)
        }
    }

    // Generate meeting minutes from transcript
    minutes, err := generateMinutes(ctx, fullTranscript.String())
    if err != nil {
        return fmt.Errorf("generate minutes: %w", err)
    }

    fmt.Println(minutes)
    return nil
}

func generateMinutes(ctx context.Context, transcript string) (string, error) {
    model, err := llm.New("openai", nil)
    if err != nil {
        return "", err
    }

    msgs := []schema.Message{
        &schema.SystemMessage{Parts: []schema.ContentPart{
            schema.TextPart{Text: "Generate structured meeting minutes from this transcript. " +
                "Include: attendees, key discussion points, decisions made, and action items."},
        }},
        &schema.HumanMessage{Parts: []schema.ContentPart{
            schema.TextPart{Text: transcript},
        }},
    }

    resp, err := model.Generate(ctx, msgs)
    if err != nil {
        return "", err
    }

    return resp.Parts[0].(schema.TextPart).Text, nil
}
```

## Use Case 3: Voice Forms

Collect structured data through natural voice conversations. The form orchestrator manages state across turns, validates answers, and supports corrections. This use case separates STT and TTS (rather than using S2S) because the form logic needs to inspect and validate the transcribed text between speech input and speech output — a step that requires text as an intermediate representation.

```go
import (
    "github.com/lookatitude/beluga-ai/voice/stt"
    "github.com/lookatitude/beluga-ai/voice/tts"

    _ "github.com/lookatitude/beluga-ai/voice/stt/providers/deepgram"
    _ "github.com/lookatitude/beluga-ai/voice/tts/providers/elevenlabs"
)

type FormField struct {
    Name       string
    Prompt     string
    Validate   func(string) error
    Required   bool
}

type VoiceForm struct {
    fields  []FormField
    current int
    answers map[string]string
    stt     stt.STT
    tts     tts.TTS
}

func (f *VoiceForm) Run(ctx context.Context, audioIn iter.Seq2[[]byte, error]) (map[string]string, error) {
    // Ask the first question
    question := f.fields[f.current].Prompt
    audio, err := f.tts.Synthesize(ctx, question,
        tts.WithVoice("aria"),
        tts.WithSpeed(1.0),
    )
    if err != nil {
        return nil, fmt.Errorf("synthesize: %w", err)
    }
    sendAudio(audio)

    // Process answers
    transcripts := f.stt.TranscribeStream(ctx, audioIn,
        stt.WithLanguage("en"),
        stt.WithPunctuation(true),
    )

    for event, err := range transcripts {
        if err != nil {
            return nil, fmt.Errorf("transcribe: %w", err)
        }

        if !event.IsFinal {
            continue
        }

        field := f.fields[f.current]

        // Validate the answer
        if err := field.Validate(event.Text); err != nil {
            reprompt := fmt.Sprintf("I didn't quite get that. %s", field.Prompt)
            audio, _ := f.tts.Synthesize(ctx, reprompt, tts.WithVoice("aria"))
            sendAudio(audio)
            continue
        }

        // Save and advance
        f.answers[field.Name] = event.Text
        f.current++

        if f.current >= len(f.fields) {
            // Form complete
            confirm := "Thank you. I have all the information I need."
            audio, _ := f.tts.Synthesize(ctx, confirm, tts.WithVoice("aria"))
            sendAudio(audio)
            return f.answers, nil
        }

        // Ask next question
        next := f.fields[f.current].Prompt
        audio, _ := f.tts.Synthesize(ctx, next, tts.WithVoice("aria"))
        sendAudio(audio)
    }

    return f.answers, nil
}
```

## Use Case 4: Interactive Audiobooks

Dynamic narration with character voices and branching storylines. This is a TTS-only use case — the story text is pre-authored and the system synthesizes it with character-appropriate voices. STT is not needed because user choices come from the UI, not from speech.

```go
import (
    "github.com/lookatitude/beluga-ai/voice/tts"
    _ "github.com/lookatitude/beluga-ai/voice/tts/providers/elevenlabs"
)

type Character struct {
    Name  string
    Voice string // TTS voice ID
    Pitch float64
}

func narrateScene(ctx context.Context, engine tts.TTS, scene Scene) error {
    for _, line := range scene.Lines {
        char := scene.Characters[line.Speaker]

        audio, err := engine.Synthesize(ctx, line.Text,
            tts.WithVoice(char.Voice),
            tts.WithPitch(char.Pitch),
            tts.WithSpeed(0.95), // Slightly slower for narration
        )
        if err != nil {
            return fmt.Errorf("synthesize line: %w", err)
        }

        sendAudio(audio)
    }
    return nil
}
```

## Building a Full Voice Pipeline

Compose multiple frame processors into a complete voice pipeline. The `voice.Chain()` function connects processors in sequence — each processor's output channel becomes the next processor's input channel. This chain-of-responsibility pattern means adding or removing stages is a one-line change.

```go
func buildVoicePipeline(ctx context.Context) (voice.FrameProcessor, error) {
    // VAD detects speech vs silence
    vad := voice.NewSileroVAD(voice.VADConfig{
        Threshold: 0.5,
        MinSpeechDuration: 250 * time.Millisecond,
    })

    // STT converts speech to text
    sttProc := stt.AsFrameProcessor(sttEngine,
        stt.WithLanguage("en"),
        stt.WithPunctuation(true),
    )

    // TTS converts text to speech
    ttsProc := tts.AsFrameProcessor(ttsEngine, 16000,
        tts.WithVoice("nova"),
    )

    // Compose into pipeline
    pipeline := voice.Chain(vad, sttProc, agentProcessor, ttsProc)

    return pipeline, nil
}
```

## Production Considerations

### Latency

Voice applications are latency-sensitive. Target end-to-end latency under 500ms for real-time conversations:

- Use S2S providers (OpenAI Realtime) for the lowest latency
- Pre-buffer audio frames to reduce jitter
- Deploy close to your users (edge compute or regional deployment)
- Use WebSocket transport for persistent, low-overhead connections

### Observability

Track voice-specific metrics:

```go
span.SetAttributes(
    attribute.Float64("voice.stt_latency_ms", sttLatency),
    attribute.Float64("voice.tts_latency_ms", ttsLatency),
    attribute.Float64("voice.e2e_latency_ms", endToEndLatency),
    attribute.String("voice.stt_provider", "deepgram"),
    attribute.String("voice.tts_provider", "elevenlabs"),
)
```

### Resilience

- Use Beluga AI's circuit breaker for STT/TTS provider failover
- Buffer audio during brief network interruptions
- Implement graceful degradation: fall back to text-only mode if voice fails
- Monitor provider health and switch providers dynamically

### Scaling

- Voice sessions are stateful — use sticky sessions or session affinity at the load balancer
- Scale STT/TTS independently based on demand
- Use connection pooling for WebSocket transports
- For meeting transcription, process audio in parallel tracks (one per speaker)

## Related Resources

- [Voice AI Pipeline](/guides/voice-ai/) for detailed voice configuration
- [Conversational AI Assistant](/use-cases/conversational-ai/) for adding memory to voice agents
- [Production Agent Platform](/use-cases/production-platform/) for deployment patterns
