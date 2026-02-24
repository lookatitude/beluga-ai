---
title: SSML Tuning for Expressive Speech
description: "Control TTS pauses, emphasis, pitch, and pronunciation with SSML in Go — build expressive speech output with programmatic SSML builders in Beluga AI."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, Go, tutorial, SSML, TTS, expressive speech, pronunciation, text-to-speech"
---

Speech Synthesis Markup Language (SSML) provides granular control over text-to-speech output, enabling pauses, emphasis, pitch adjustment, and pronunciation overrides. Without SSML, TTS engines apply default prosody that may not match the conversational context -- a notification should sound different from a thoughtful explanation, and a brand name needs consistent pronunciation. This tutorial demonstrates how to use SSML with Beluga's TTS providers and build helpers for constructing SSML documents programmatically.

## What You Will Build

An expressive TTS pipeline that uses SSML tags to control speech timing, emphasis, and pronunciation, with a programmatic builder for safe SSML construction.

## Prerequisites

- Google Cloud or Azure Speech API key (both have extensive SSML support)
- Basic familiarity with XML syntax

## What is SSML?

SSML is an XML-based markup language that wraps plain text with directives for the speech synthesizer. Instead of passing raw text and relying on the TTS engine to infer appropriate delivery, you pass structured XML that explicitly controls timing, emphasis, and pronunciation. This is important for voice agents because the default prosody optimized for reading articles aloud is often wrong for conversational interactions.

```xml
<speak>
  Hello! <break time="500ms"/>
  I am very <emphasis level="strong">excited</emphasis> to meet you.
</speak>
```

## Step 1: Basic SSML with Google Cloud TTS

Google Cloud TTS has comprehensive SSML support. Most Beluga TTS providers auto-detect SSML when the input begins with `<speak>`, so you do not need to set a separate flag -- the provider inspects the input string and routes it to the appropriate synthesis endpoint automatically.

```go
package main

import (
	"context"
	"log"
	"os"

	"github.com/lookatitude/beluga-ai/voice/tts"
	_ "github.com/lookatitude/beluga-ai/voice/tts/providers/google"
)

const greeting = `<speak>
  Hello! <break time="500ms"/>
  I am very <emphasis level="strong">excited</emphasis> to meet you.
  <break time="300ms"/>
  How can I help you today?
</speak>`

func main() {
	ctx := context.Background()

	provider, err := tts.NewProvider(ctx, "google", tts.DefaultConfig(),
		tts.WithAPIKey(os.Getenv("GOOGLE_API_KEY")),
		tts.WithVoice("en-US-Neural2-F"),
	)
	if err != nil {
		log.Fatalf("create TTS provider: %v", err)
	}

	// Providers detect SSML automatically when input starts with <speak>
	audio, err := provider.GenerateSpeech(ctx, greeting)
	if err != nil {
		log.Fatalf("generate speech: %v", err)
	}

	if err := os.WriteFile("greeting.mp3", audio, 0644); err != nil {
		log.Fatalf("write file: %v", err)
	}
}
```

## Step 2: Common SSML Tags

Each SSML tag addresses a specific aspect of speech delivery. Understanding when to use each tag helps you craft output that sounds natural rather than robotic.

### Break (Pauses)

Breaks create silence between phrases. Use them to separate logical sections, give the listener time to process information, or create a conversational rhythm. The `time` attribute provides exact control, while `strength` uses the engine's natural pause model.

```xml
<break time="500ms"/>      <!-- Pause for 500 milliseconds -->
<break time="1s"/>         <!-- Pause for 1 second -->
<break strength="strong"/> <!-- Natural paragraph-level pause -->
```

### Emphasis

Emphasis changes the volume, pitch, and speed of a word to draw attention to it. This is useful for highlighting key terms, correcting misunderstandings, or expressing urgency.

```xml
<emphasis level="strong">important</emphasis>  <!-- Louder, slower -->
<emphasis level="moderate">notable</emphasis>   <!-- Slightly stressed -->
<emphasis level="reduced">minor</emphasis>      <!-- Softer, faster -->
```

### Prosody (Pitch, Rate, Volume)

Prosody controls the overall delivery style of a phrase. Adjusting rate and pitch together can convey emotions: slow rate with low pitch suggests seriousness, while fast rate with higher pitch suggests excitement.

```xml
<prosody rate="slow" pitch="+2st">
  I'm speaking slowly and at a higher pitch.
</prosody>
<prosody volume="loud" rate="fast">
  This is urgent!
</prosody>
```

### Say-As (Interpretation)

Say-as tells the engine how to interpret ambiguous content. Without it, "12/25/2025" might be read as a fraction or a phone number. The `interpret-as` attribute disambiguates the intended reading.

```xml
<say-as interpret-as="telephone">555-0199</say-as>
<say-as interpret-as="date" format="mdy">12/25/2025</say-as>
<say-as interpret-as="cardinal">42</say-as>
```

### Phoneme (Pronunciation Override)

Phoneme overrides the default pronunciation for words the engine mispronounces. This is essential for brand names, technical terms, and proper nouns that do not follow standard pronunciation rules.

```xml
<phoneme alphabet="ipa" ph="p&#618;&#712;k&#593;&#720;n">pecan</phoneme>
```

## Step 3: Programmatic SSML Builder

Manual XML string construction is error-prone -- unclosed tags, unescaped characters, and malformed attributes can cause synthesis failures at runtime. A builder constructs valid SSML safely by handling escaping and structure internally, so you focus on content rather than XML syntax.

```go
package ssml

import (
	"fmt"
	"strings"
)

// Builder constructs SSML documents programmatically.
type Builder struct {
	parts []string
}

// NewBuilder returns a new SSML builder.
func NewBuilder() *Builder {
	return &Builder{}
}

// Text adds plain text content.
func (b *Builder) Text(text string) *Builder {
	b.parts = append(b.parts, text)
	return b
}

// Pause adds a break with the given duration (e.g., "500ms", "1s").
func (b *Builder) Pause(duration string) *Builder {
	b.parts = append(b.parts, fmt.Sprintf(`<break time="%s"/>`, duration))
	return b
}

// Emphasis wraps text with the given emphasis level (strong, moderate, reduced).
func (b *Builder) Emphasis(text, level string) *Builder {
	b.parts = append(b.parts, fmt.Sprintf(`<emphasis level="%s">%s</emphasis>`, level, text))
	return b
}

// Prosody wraps text with pitch, rate, and volume adjustments.
func (b *Builder) Prosody(text, rate, pitch string) *Builder {
	b.parts = append(b.parts, fmt.Sprintf(`<prosody rate="%s" pitch="%s">%s</prosody>`, rate, pitch, text))
	return b
}

// Build returns the complete SSML document.
func (b *Builder) Build() string {
	return "<speak>" + strings.Join(b.parts, "") + "</speak>"
}
```

Usage:

```go
	speech := ssml.NewBuilder().
		Text("Welcome to Beluga AI.").
		Pause("500ms").
		Text("Let me help you with ").
		Emphasis("deployment", "strong").
		Text(" today.").
		Build()

	audio, err := provider.GenerateSpeech(ctx, speech)
	if err != nil {
		log.Fatalf("generate speech: %v", err)
	}
```

## Step 4: Handling Providers Without SSML Support

Not all TTS providers support SSML. OpenAI's standard TTS API, for example, does not process SSML tags. Rather than maintaining separate code paths for SSML and non-SSML providers, use a fallback that attempts SSML synthesis first and strips tags on failure. This approach lets you write SSML throughout your application and degrade gracefully when a provider lacks support.

```go
import (
	"regexp"
	"strings"
)

var tagPattern = regexp.MustCompile(`<[^>]+>`)

// SafeSynthesize generates speech, stripping SSML tags if the provider
// does not support them.
func SafeSynthesize(ctx context.Context, provider tts.TTSProvider, input string) ([]byte, error) {
	if isSSML(input) {
		// Attempt SSML synthesis first; fall back to plain text on error
		audio, err := provider.GenerateSpeech(ctx, input)
		if err == nil {
			return audio, nil
		}
		input = stripTags(input)
	}
	return provider.GenerateSpeech(ctx, input)
}

func isSSML(input string) bool {
	return strings.HasPrefix(strings.TrimSpace(input), "<speak>")
}

func stripTags(input string) string {
	return strings.TrimSpace(tagPattern.ReplaceAllString(input, " "))
}
```

## Verification

1. Generate speech with `<break time="3s"/>` and verify three seconds of silence in the output.
2. Use `<emphasis level="strong">` on a word and listen for the volume and pitch change.
3. Compare the same text rendered with and without SSML.
4. Test the `SafeSynthesize` fallback with a provider that does not support SSML.

## Next Steps

- [Cloning Voices with ElevenLabs](/docs/tutorials/voice/elevenlabs-cloning) -- High-fidelity voice synthesis
- [Voice Session Interruptions](/docs/tutorials/voice/session-interruptions) -- Integrate expressive TTS into voice agents
- [Sensitivity Tuning](/docs/tutorials/voice/sensitivity-tuning) -- Balance detection thresholds for natural conversation
