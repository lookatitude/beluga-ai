---
title: "Multi-Speaker Dialogue Synthesis"
description: "Recipe for generating multi-speaker conversations in Go with distinct voices, natural turn transitions, and pacing using Beluga AI TTS pipeline."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, multi-speaker TTS, Go dialogue synthesis, voice conversation, speaker voices, turn transitions, audiobook recipe"
---

## Problem

You need to generate natural-sounding conversations with multiple speakers, where each speaker has a distinct voice and the dialogue flows naturally between speakers with appropriate pauses. Single-voice TTS fails to convey the conversational structure of multi-party dialogue, making it difficult for listeners to distinguish who is speaking or follow the conversation flow. This becomes problematic for applications like audiobook narration, training simulations, or conversational agents that simulate realistic human interactions. Listeners rely on voice differences and pacing cues to parse dialogue, and without these cues, multi-speaker content sounds confusing or monotonous.

## Solution

Implement multi-speaker TTS that manages multiple voice models, assigns speakers to dialogue turns, handles turn-taking transitions, and synthesizes speech with appropriate pauses and intonation. The key insight is to separate speaker identity from content: each turn specifies both what is said and who says it. This design mirrors theatrical scripts, where character names precede dialogue. By maintaining a registry of speaker-to-voice mappings, you can consistently apply the same voice to a speaker across turns, even when turns are generated dynamically or out of order.

Pauses between turns are critical for natural flow. In real conversations, speakers leave brief silences between turns for comprehension and turn-taking coordination. Without these pauses, multi-speaker TTS sounds like a single person talking to themselves. The solution here inserts configurable silence (represented as zero-sample audio) between turns, tuned based on dialogue pacing: longer pauses for thoughtful exchanges, shorter pauses for rapid back-and-forth.

## Code Example

```go
package main

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("beluga.voice.tts.multi_speaker")

// DialogueTurn represents a single turn in dialogue.
type DialogueTurn struct {
	Speaker    string
	Text       string
	VoiceID    string
	PauseAfter time.Duration
}

// MultiSpeakerTTS synthesizes multi-speaker dialogue.
type MultiSpeakerTTS struct {
	ttsProvider interface{}
	voices      map[string]string // speaker name -> voice ID
}

func NewMultiSpeakerTTS(ttsProvider interface{}) *MultiSpeakerTTS {
	return &MultiSpeakerTTS{
		ttsProvider: ttsProvider,
		voices:      make(map[string]string),
	}
}

// RegisterVoice assigns a voice ID to a speaker name.
func (mstts *MultiSpeakerTTS) RegisterVoice(speaker string, voiceID string) {
	mstts.voices[speaker] = voiceID
}

// SynthesizeDialogue synthesizes a complete dialogue.
func (mstts *MultiSpeakerTTS) SynthesizeDialogue(ctx context.Context, turns []DialogueTurn) ([][]byte, error) {
	ctx, span := tracer.Start(ctx, "multi_speaker_tts.synthesize")
	defer span.End()

	span.SetAttributes(attribute.Int("turn_count", len(turns)))

	audioSegments := make([][]byte, 0, len(turns))

	for i, turn := range turns {
		voiceID := mstts.voices[turn.Speaker]
		if voiceID == "" {
			voiceID = turn.VoiceID
		}

		span.SetAttributes(
			attribute.String("speaker", turn.Speaker),
			attribute.String("voice_id", voiceID),
			attribute.Int("turn_index", i),
		)

		// In production: audio, err := mstts.ttsProvider.Synthesize(ctx, turn.Text, voiceID)
		audio := []byte(turn.Text)
		audioSegments = append(audioSegments, audio)

		// Add silence for pause between turns
		if turn.PauseAfter > 0 {
			silenceSamples := int(turn.PauseAfter.Seconds() * 16000 * 2) // 16kHz, 16-bit
			silence := make([]byte, silenceSamples)
			audioSegments = append(audioSegments, silence)
		}
	}

	span.SetStatus(trace.StatusOK, "dialogue synthesized")
	return audioSegments, nil
}

func main() {
	ctx := context.Background()

	mstts := NewMultiSpeakerTTS(nil)

	mstts.RegisterVoice("alice", "voice-alice")
	mstts.RegisterVoice("bob", "voice-bob")

	turns := []DialogueTurn{
		{Speaker: "alice", Text: "Hello, how are you?", PauseAfter: 500 * time.Millisecond},
		{Speaker: "bob", Text: "I'm doing well, thanks!", PauseAfter: 500 * time.Millisecond},
		{Speaker: "alice", Text: "That's great to hear!"},
	}

	segments, err := mstts.SynthesizeDialogue(ctx, turns)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Synthesized %d audio segments\n", len(segments))
}
```

## Explanation

1. **Voice mapping** -- Speaker names are mapped to voice IDs, allowing consistent voice assignment throughout the dialogue regardless of turn order. This registry pattern decouples speaker identity from TTS provider details. If you switch TTS providers or update voice IDs, you only need to update the registry, not every turn definition. The fallback to `turn.VoiceID` allows one-off speaker voices without registration, useful for minor characters or guest speakers.

2. **Turn-based synthesis** -- Each turn is synthesized separately with its assigned voice, creating distinct voices for each speaker. This per-turn synthesis approach integrates easily with streaming dialogue generation, where turns arrive incrementally from an LLM. You do not need to buffer the entire conversation before starting synthesis. Each turn is independent, allowing parallel synthesis if needed (though order must be preserved for playback).

3. **Natural pauses** -- Configurable pauses between turns create natural conversation flow. Silence is generated at the target sample rate. The calculation here (16kHz, 16-bit samples) must match your TTS output format. Silence duration is per-turn configurable because different dialogue contexts require different pacing: a dramatic pause before a reveal might be 2 seconds, while rapid banter might use 200ms pauses. The silence is represented as zero-valued samples, which is sonically neutral and avoids clicks or pops during playback.

**Key insight:** Use distinct voices for each speaker and add appropriate pauses between turns. This creates natural-sounding dialogue that is easy to follow. The registry pattern for voice mapping is critical for scalability: as dialogue length increases, managing voice consistency manually becomes error-prone. The combination of consistent voice assignment and deliberate pausing mirrors how humans structure conversations, making synthesized dialogue feel authentic.

## Variations

### Emotional Tones

Add an `Emotion` field to `DialogueTurn` (e.g. "happy", "sad", "excited") and pass it to the TTS provider for more expressive speech.

### Real-Time Synthesis

Stream dialogue turns as they become available using a channel-based API for live applications.

## Related Recipes

- **[SSML Emphasis and Pause Tuning](./ssml-tuning)** -- Fine-tune speech with SSML markup
- **[Glass-to-Glass Latency](./glass-to-glass-latency)** -- Reduce end-to-end latency
- **[Voice Backends Configuration](./voice-backends)** -- Backend provider setup
