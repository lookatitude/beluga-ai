# SSML Tuning for Expressive Speech

<!--
Persona: Pair Programmer Colleague
- Treat the reader as a competent peer
- Be helpful, direct, and slightly informal but professional
- Focus on getting results quickly
- Keep it concise and functional
- Provide runnable code examples
-->

## What you will build
In this tutorial, we'll use Speech Synthesis Markup Language (SSML) to add emphasis, pauses, and emotional depth to your AI's voice. You'll learn how to use tags like `<break>`, `<emphasis>`, and `<prosody>` to create natural-sounding speech patterns.

## Learning Objectives
- ✅ Understand SSML tags (`<break>`, `<emphasis>`, `<prosody>`)
- ✅ Implement SSML with Google or Azure providers
- ✅ Create "Natural" sounding speech patterns
- ✅ Handle SSML validation

## Introduction
Welcome, colleague! Plain text-to-speech can often sound robotic and flat. By using SSML, we can "direct" the AI's performance—adding meaningful pauses, stressing important words, and adjusting the pitch to match the context of the conversation.

## Prerequisites

- Google Cloud or Azure Speech API Key
- Beluga AI `pkg/voice/tts` package

## What is SSML?

SSML is an XML-based language that gives you granular control over text-to-speech. Instead of just passing text, you pass structured XML.

## Step 1: Basic SSML Structure
```go
const ssml = `
<speak>
  Hello! <break time="500ms"/> I am very <emphasis level="strong">excited</emphasis> to meet you.
</speak>
`
```

## Step 2: Implementation with Google Cloud TTS

Google Cloud TTS has extensive SSML support.
```go
package main

import (
    "context"
    "github.com/lookatitude/beluga-ai/pkg/voice/tts/providers/google"
)

func main() {
    config := &google.Config{
        APIKey: os.Getenv("GOOGLE_API_KEY"),
        VoiceName: "en-US-Neural2-F",
    }
    
    provider, _ := google.NewProvider(config)
    
    // Most Beluga TTS providers detect SSML automatically if it starts with <speak>
    audio, _ := provider.Synthesize(context.Background(), ssml)
}
```

## Step 3: Advanced Tags

### 1. Prosody (Pitch, Rate, Volume)<prosody rate="slow" pitch="+2st">I'm speaking slowly and higher.</prosody>
```

### 2. Say-As (Dates, Phone Numbers)<say-as interpret-as="telephone">555-0199</say-as>
```

### 3. Phoneme (Pronunciation)<phoneme alphabet="ipa" ph="pɪˈkɑːn">pecan</phoneme>
```

## Step 4: Building an SSML Helper

Manual XML string concatenation is error-prone. Use a helper.






type SpeechBuilder struct \{
    content string
}

func (b *SpeechBuilder) AddText(t string) *SpeechBuilder \{
    b.content += t
    return b
}

func (b *SpeechBuilder) AddPause(ms int) *SpeechBuilder \{
    b.content += fmt.Sprintf("<break time=\"%dms\"/>", ms)
    return b
}

func (b *SpeechBuilder) Build() string \{
    return "<speak>" + b.content + "</speak>"
}
```

## Step 5: Handling Incompatible Providers

Not all providers support SSML (e.g., standard OpenAI TTS doesn't). Use a fallback or strip tags.
```go
func SafeSynthesize(p tts.Provider, input string) ([]byte, error) {
    if !p.SupportsSSML() && isSSML(input) {
        input = stripTags(input)
    }
    return p.Synthesize(ctx, input)
}
```

## Verification

1. Generate speech with `<break time="3s"/>`. Verify the silence.
2. Use `<emphasis level="strong">` on a specific word. Listen for the volume/pitch change.
3. Compare the same text with and without SSML.

## Next Steps

- **[Cloning Voices with ElevenLabs](./voice-tts-elevenlabs-cloning.md)** - High-fidelity voices.
- **[Voice Session Management](../session/README.md)** - Integrate expressive TTS into agents.
