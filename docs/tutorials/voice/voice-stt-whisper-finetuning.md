# Fine-tuning Whisper for Industry Terms

<!--
Persona: Pair Programmer Colleague
- Treat the reader as a competent peer
- Be helpful, direct, and slightly informal but professional
- Focus on getting results quickly
- Keep it concise and functional
- Provide runnable code examples
-->

## What you will build
In this tutorial, we'll improve the accuracy of Speech-to-Text (STT) for specialized terminology (medical, legal, technical). We'll explore "prompting" strategies for Whisper and "keywords" boosting for Deepgram to handle jargon that standard models often miss.

## Learning Objectives
- ✅ Understand why STT models fail on industry terms
- ✅ Use the `Prompt` parameter to guide Whisper
- ✅ Use `Keywords` to boost Deepgram accuracy
- ✅ Implement a vocabulary-aware STT wrapper

## Introduction
Welcome, colleague! Standard STT models often struggle with technical jargon—turning "gRPC" into "Gee RPS" or "Beluga AI" into "Blue guy eye." Let's look at how we can guide these models with context to ensure our transcripts are accurate for specialized industries.

## Prerequisites

- OpenAI or Deepgram API Key
- List of industry-specific terms (e.g., "Kubernetes", "gRPC", "Beluga")

## The Problem: Vocabulary Mismatch

Standard models often mishear technical terms:
- "gRPC" → "Gee RPS"
- "Beluga AI" → "Blue guy eye"

## Step 1: Using Whisper Prompts

OpenAI's Whisper allows a "prompt" that the model uses as context for the audio.
```go
package main

import (
    "github.com/lookatitude/beluga-ai/pkg/voice/stt"
    "github.com/lookatitude/beluga-ai/pkg/voice/stt/providers/openai"
)

func main() {
    config := &openai.Config{
        APIKey: os.Getenv("OPENAI_API_KEY"),
        // Guiding the model with a prompt containing industry terms
        Prompt: "The transcript involves Kubernetes, gRPC, and Beluga AI framework.",
    }
    
    provider, _ := openai.NewProvider(config)
    // ...
}
```

## Step 2: Boosting Deepgram Keywords

Deepgram supports a `Keywords` parameter to boost the probability of specific terms.
```text
import "github.com/lookatitude/beluga-ai/pkg/voice/stt/providers/deepgram"
go
func main() {
    config := &deepgram.Config{
        APIKey: os.Getenv("DEEPGRAM_API_KEY"),
        // Boosting specific terms. 
        // Value after ':' is the boost level (default 1.0)
        Keywords: []string{
            "Kubernetes:2.0",
            "gRPC:2.5",
            "Beluga:3.0",
        },
    }
    
    provider, _ := deepgram.NewProvider(config)
}
```

## Step 3: Vocabulary-Aware Wrapper

If you want to dynamicallly update terms based on user context:
```go
type IndustrySTT struct {
    provider stt.Provider
    industry string
}

func (s *IndustrySTT) Transcribe(ctx context.Context, audio []byte) (string, error) {
    // Logic to select best prompt/keywords based on industry
    return s.provider.Transcribe(ctx, audio)
}
```

## Step 4: Post-Processing Correction (Optional)

If the model still fails, you can use an LLM to "fix" the transcript based on known terms.
```go
func fixTranscript(raw string, knownTerms []string) string {
    // Pass raw transcript + terms to a small LLM (Haiku/GPT-3.5)
    // "Fix spelling of technical terms in: ..."
    return fixed
}
```

## Verification

1. Record yourself saying a technical sentence: "Let's deploy the gRPC service to Kubernetes using Beluga."
2. Run transcription without prompts/keywords.
3. Run transcription WITH prompts/keywords.
4. Compare the accuracy.

## Next Steps

- **[Real-time STT Streaming](./voice-stt-realtime-streaming.md)** - Low latency transcription.
- **[Cloning Voices with ElevenLabs](./voice-tts-elevenlabs-cloning.md)** - The other side of voice.
