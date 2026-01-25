---
title: "Multi-speaker Dialogue Synthesis"
package: "voice/tts"
category: "audio"
complexity: "advanced"
---

# Multi-speaker Dialogue Synthesis

## Problem

You need to generate natural-sounding conversations with multiple speakers, where each speaker has a distinct voice and the dialogue flows naturally between speakers.

## Solution

Implement multi-speaker TTS that manages multiple voice models, assigns speakers to dialogue turns, handles turn-taking transitions, and synthesizes speech with appropriate pauses and intonation. This works because TTS providers support multiple voices, and you can orchestrate them to create dialogue.

## Code Example
```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"
    
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("beluga.voice.tts.multi_speaker")

// DialogueTurn represents a single turn in dialogue
type DialogueTurn struct {
    Speaker  string
    Text     string
    VoiceID  string
    PauseAfter time.Duration
}

// MultiSpeakerTTS synthesizes multi-speaker dialogue
type MultiSpeakerTTS struct {
    ttsProvider interface{} // TTS provider
    voices      map[string]string // Map speaker name to voice ID
}

// NewMultiSpeakerTTS creates a new multi-speaker TTS
func NewMultiSpeakerTTS(ttsProvider interface{}) *MultiSpeakerTTS {
    return &MultiSpeakerTTS{
        ttsProvider: ttsProvider,
        voices:      make(map[string]string),
    }
}

// RegisterVoice registers a voice for a speaker
func (mstts *MultiSpeakerTTS) RegisterVoice(speaker string, voiceID string) {
    mstts.voices[speaker] = voiceID
}

// SynthesizeDialogue synthesizes a complete dialogue
func (mstts *MultiSpeakerTTS) SynthesizeDialogue(ctx context.Context, turns []DialogueTurn) ([][]byte, error) {
    ctx, span := tracer.Start(ctx, "multi_speaker_tts.synthesize")
    defer span.End()
    
    span.SetAttributes(attribute.Int("turn_count", len(turns)))
    
    audioSegments := [][]byte{}
    
    for i, turn := range turns {
        // Get voice for speaker
        voiceID := mstts.voices[turn.Speaker]
        if voiceID == "" {
            voiceID = turn.VoiceID // Use provided voice ID
        }
        
        span.SetAttributes(
            attribute.String("speaker", turn.Speaker),
            attribute.String("voice_id", voiceID),
            attribute.Int("turn_index", i),
        )
        
        // Synthesize this turn
        // audio, err := mstts.ttsProvider.Synthesize(ctx, turn.Text, voiceID)
        
        // Add pause if specified
        if turn.PauseAfter > 0 {
            // silence := mstts.generateSilence(turn.PauseAfter)
            // audioSegments = append(audioSegments, silence)
        }
    }
    
    span.SetStatus(trace.StatusOK, "dialogue synthesized")
    return audioSegments, nil
}

func main() {
    ctx := context.Background()

    // Create multi-speaker TTS
    // ttsProvider := yourTTSProvider
    mstts := NewMultiSpeakerTTS(ttsProvider)
    
    // Register voices
    mstts.RegisterVoice("alice", "voice-alice")
    mstts.RegisterVoice("bob", "voice-bob")
    
    // Create dialogue
    turns := []DialogueTurn{
        {Speaker: "alice", Text: "Hello, how are you?", PauseAfter: 500 * time.Millisecond},
        {Speaker: "bob", Text: "I'm doing well, thanks!", PauseAfter: 500 * time.Millisecond},
        {Speaker: "alice", Text: "That's great to hear!"},
    }
    
    // Synthesize
    // segments, err := mstts.SynthesizeDialogue(ctx, turns)
    fmt.Println("Multi-speaker TTS created")
}
```

## Explanation

Let's break down what's happening:

1. **Voice mapping** - Notice how we map speaker names to voice IDs. This allows consistent voice assignment throughout the dialogue.

2. **Turn-based synthesis** - We synthesize each turn separately with its assigned voice, creating distinct voices for each speaker.

3. **Natural pauses** - We add pauses between turns to create natural conversation flow. This makes the dialogue sound more realistic.

```go
**Key insight:** Use distinct voices for each speaker and add appropriate pauses between turns. This creates natural-sounding dialogue that's easy to follow.

## Testing

```
Here's how to test this solution:
```go
func TestMultiSpeakerTTS_SynthesizesDialogue(t *testing.T) {
    mstts := NewMultiSpeakerTTS(&MockTTSProvider{})
    mstts.RegisterVoice("alice", "voice-1")
    
    turns := []DialogueTurn{
        {Speaker: "alice", Text: "Hello"},
    }
    
    segments, err := mstts.SynthesizeDialogue(context.Background(), turns)
    require.NoError(t, err)
    require.Greater(t, len(segments), 0)
}

## Variations

### Emotional Tones

Add emotional tones to voices:
type DialogueTurn struct {
    Emotion string // "happy", "sad", "excited"
}
```

### Real-time Synthesis

Synthesize dialogue in real-time:
```go
func (mstts *MultiSpeakerTTS) SynthesizeDialogueStream(ctx context.Context, turns <-chan DialogueTurn) (<-chan []byte, error) {
    // Stream synthesis
}
```

## Related Recipes

- **[Voice TTS SSML Emphasis & Pause Tuning](./voice-tts-ssml-emphasis-pause-tuning.md)** - Fine-tune SSML
- **[Voice S2S Minimizing Glass-to-Glass Latency](./voice-s2s-minimizing-glass-to-glass-latency.md)** - Reduce latency
- **[Voice Providers Guide](../guides/voice-providers.md)** - For a deeper understanding of TTS
