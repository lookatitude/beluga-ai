---
title: Multi-Turn Voice Forms
description: "Collect structured data through natural voice conversations with real-time validation. Reduce form abandonment from 38% to under 10%."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "voice forms, voice data collection, phone intake, voice validation, healthcare voice, Beluga AI, Go, conversational forms"
---

Healthcare providers, insurance companies, and government agencies need to collect structured intake data over the phone. The data itself is well-defined (name, date of birth, policy number, symptoms), but the collection process is difficult: callers give fragmented answers, change their mind, provide information out of order, and go off-topic. Traditional touch-tone forms force callers into rigid sequences that do not accommodate these natural behaviors, resulting in 38% abandonment rates on forms longer than 5 fields.

A voice-based form system using Beluga AI's voice pipeline manages form state across turns, validates answers in real time, and supports corrections and resumption. The system uses separate STT and TTS components (not S2S) because the form orchestrator needs to inspect the transcribed text for validation before generating the next prompt — this validation step requires text as an intermediate representation.

## Solution Architecture

```mermaid
graph TB
    User[User] --> Mic[Microphone]
    Mic --> Pipeline[Voice Pipeline]
    Pipeline --> STT[STT]
    Pipeline --> TTS[TTS]
    Pipeline --> VAD[VAD]
    STT --> Orchestrator[Form Orchestrator]
    Orchestrator --> State[Form State]
    State --> Valid[Validate Answer]
    Valid --> Next[Next Question or Reprompt]
    Next --> TTS
    TTS --> Speaker[Speaker]
    Speaker --> User
```

The voice pipeline handles audio processing (STT, TTS, VAD, turn detection). The form orchestrator manages form state separately from the session layer, advancing through questions, validating answers, and persisting progress for resumption.

Separating form state from session state is a key architectural decision. Sessions are transient (tied to a WebSocket connection), while form state must survive disconnections so callers can resume where they left off. By decoupling them, the form orchestrator can be tested independently of the voice pipeline, and form state can be persisted to any storage backend.

## Implementation

### Voice Form System

The form is defined as a sequence of `FormField` structs, each with a prompt, validation function, and required flag. The `Run` method drives a one-question-per-turn loop: synthesize the prompt via TTS, listen for a response via streaming STT, validate the answer, and either advance or reprompt. This sequential, field-by-field approach was chosen over free-form collection because it produces higher completion rates — callers know exactly what information is needed at each step.

```go
package main

import (
    "context"
    "fmt"
    "iter"

    "github.com/lookatitude/beluga-ai/voice/stt"
    "github.com/lookatitude/beluga-ai/voice/tts"

    _ "github.com/lookatitude/beluga-ai/voice/stt/providers/deepgram"
    _ "github.com/lookatitude/beluga-ai/voice/tts/providers/elevenlabs"
)

// FormField defines a single question in the form.
type FormField struct {
    Name     string
    Prompt   string
    Validate func(string) error
    Required bool
}

// VoiceForm collects structured data through voice conversation.
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
            audio, err := f.tts.Synthesize(ctx, reprompt, tts.WithVoice("aria"))
            if err != nil {
                return nil, fmt.Errorf("synthesize reprompt: %w", err)
            }
            sendAudio(audio)
            continue
        }

        // Save and advance
        f.answers[field.Name] = event.Text
        f.current++

        if f.current >= len(f.fields) {
            confirm := "Thank you. I have all the information I need."
            audio, err := f.tts.Synthesize(ctx, confirm, tts.WithVoice("aria"))
            if err != nil {
                return nil, fmt.Errorf("synthesize confirm: %w", err)
            }
            sendAudio(audio)
            return f.answers, nil
        }

        // Ask next question
        next := f.fields[f.current].Prompt
        audio, err := f.tts.Synthesize(ctx, next, tts.WithVoice("aria"))
        if err != nil {
            return nil, fmt.Errorf("synthesize next: %w", err)
        }
        sendAudio(audio)
    }

    return f.answers, nil
}
```

### Form State Persistence

Store form state (current question index, collected answers, session ID) for resumption after dropped calls:

```go
type FormState struct {
    SessionID string
    Current   int
    Answers   map[string]string
    CreatedAt time.Time
    UpdatedAt time.Time
}

// Persist by session ID; restore on reconnect
func (f *VoiceForm) SaveState(sessionID string) FormState {
    return FormState{
        SessionID: sessionID,
        Current:   f.current,
        Answers:   f.answers,
    }
}
```

## Deployment Considerations

- **Turn detection**: Use VAD and turn detection to avoid treating partial utterances as final answers
- **Interruptions**: Enable interruption support so users can correct themselves mid-prompt
- **State persistence**: Persist form state by session ID from day one for resumption after dropped calls
- **Navigation**: Add "repeat", "go back", and "skip" intents for user control
- **Confirmation turns**: For critical fields, add optional "Did you say X?" confirmation turns
- **PII handling**: Encrypt form state at rest when collecting sensitive data

## Results

| Metric | Before | After | Improvement |
|--------|--------|-------|-------------|
| Form completion rate | 62% | 88% | +42% |
| User correction rate | N/A | 12% | Under 15% target |

### Lessons Learned

- **One question per turn**: Clear structure prevents confusion, though longer forms need more turns
- **Turn detection matters**: Avoiding partial-utterance answers was critical for form accuracy
- **Separate form state from session**: Makes testing, persistence, and resumption straightforward

## Related Resources

- [Voice Sessions Overview](/docs/use-cases/voice-sessions-overview/) for session and transport patterns
- [Voice AI Applications](/docs/use-cases/voice-applications/) for voice pipeline architecture
- [Conversational AI Assistant](/docs/use-cases/conversational-ai/) for broader conversational patterns
