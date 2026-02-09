---
title: Fine-tuning Whisper for Industry Terms
description: Improve STT accuracy for specialized vocabulary using Whisper prompts and Deepgram keyword boosting.
---

Standard speech-to-text models frequently misrecognize domain-specific terminology. This tutorial demonstrates two approaches to improving accuracy: using Whisper's prompt parameter to provide contextual vocabulary, and using Deepgram's keyword boosting to increase the probability of specific terms.

## What You Will Build

A vocabulary-aware STT configuration that correctly transcribes technical and industry-specific terms such as "Kubernetes," "gRPC," and "Beluga AI" that standard models often misrecognize.

## Prerequisites

- OpenAI API key (for Whisper) or Deepgram API key
- A list of industry-specific terms relevant to your domain

## The Problem: Vocabulary Mismatch

Standard STT models are trained on general speech. They often produce incorrect transcriptions for domain-specific terms:

| Spoken Term    | Typical Misrecognition |
|---------------|----------------------|
| gRPC          | "Gee RPS"            |
| Beluga AI     | "Blue guy eye"       |
| Kubernetes    | "Cooper Netties"     |
| pgvector      | "PG Vector"          |

## Step 1: Whisper Prompt-Based Guidance

OpenAI's Whisper API accepts a `Prompt` parameter that provides the model with vocabulary context. The prompt is not a system instruction; it is treated as prior transcript text that biases the model toward expected terms.

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/lookatitude/beluga-ai/voice/stt"
	_ "github.com/lookatitude/beluga-ai/voice/stt/providers/openai"
)

func main() {
	ctx := context.Background()

	provider, err := stt.NewProvider(ctx, "openai", stt.DefaultConfig(),
		stt.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
		stt.WithModel("whisper-1"),
		stt.WithLanguage("en"),
	)
	if err != nil {
		log.Fatalf("create provider: %v", err)
	}

	// Transcribe with vocabulary context
	// The provider-specific "prompt" parameter biases recognition
	transcript, err := provider.Transcribe(ctx, audioData)
	if err != nil {
		log.Fatalf("transcribe: %v", err)
	}

	fmt.Println("Transcript:", transcript)
}
```

To pass the prompt for vocabulary guidance, use provider-specific configuration:

```go
	provider, err := stt.NewProvider(ctx, "openai", &stt.Config{
		Provider: "openai",
		APIKey:   os.Getenv("OPENAI_API_KEY"),
		Model:    "whisper-1",
		Language: "en",
		ProviderSpecific: map[string]any{
			"prompt": "The transcript involves Kubernetes, gRPC, pgvector, and Beluga AI framework.",
		},
	})
```

The prompt should contain the exact spelling of terms you expect. The model uses this text as context when decoding audio, increasing the likelihood of matching those terms.

## Step 2: Deepgram Keyword Boosting

Deepgram's API supports a `Keywords` parameter that explicitly boosts the probability of specific terms during recognition. Each keyword can include a numeric boost level.

```go
package main

import (
	"context"
	"log"
	"os"

	"github.com/lookatitude/beluga-ai/voice/stt"
	_ "github.com/lookatitude/beluga-ai/voice/stt/providers/deepgram"
)

func main() {
	ctx := context.Background()

	provider, err := stt.NewProvider(ctx, "deepgram", &stt.Config{
		Provider: "deepgram",
		APIKey:   os.Getenv("DEEPGRAM_API_KEY"),
		Model:    "nova-2",
		Language: "en-US",
		ProviderSpecific: map[string]any{
			"keywords": []string{
				"Kubernetes:2.0",
				"gRPC:2.5",
				"Beluga:3.0",
				"pgvector:2.0",
			},
		},
	})
	if err != nil {
		log.Fatalf("create provider: %v", err)
	}

	_ = provider
}
```

The boost value after the colon controls intensity. Values above 1.0 increase recognition probability; the default is 1.0.

## Step 3: Vocabulary-Aware Wrapper

For applications that serve multiple domains, create a wrapper that selects the appropriate vocabulary context based on the active conversation.

```go
// IndustrySTT wraps an STT provider with domain-specific vocabulary context.
type IndustrySTT struct {
	baseConfig *stt.Config
	vocabularies map[string][]string // industry -> terms
}

// NewIndustrySTT creates a vocabulary-aware STT wrapper.
func NewIndustrySTT(baseConfig *stt.Config, vocabularies map[string][]string) *IndustrySTT {
	return &IndustrySTT{
		baseConfig:   baseConfig,
		vocabularies: vocabularies,
	}
}

// ProviderFor returns a configured STT provider for the given industry.
func (s *IndustrySTT) ProviderFor(ctx context.Context, industry string) (stt.STTProvider, error) {
	terms, ok := s.vocabularies[industry]
	if !ok {
		terms = []string{}
	}

	cfg := *s.baseConfig
	cfg.ProviderSpecific = map[string]any{
		"prompt": "The transcript involves: " + strings.Join(terms, ", "),
	}

	return stt.NewProvider(ctx, cfg.Provider, &cfg)
}
```

## Step 4: Post-Processing Correction

When the model still misrecognizes terms despite prompting, apply a post-processing step that uses an LLM to correct known vocabulary in the transcript.

```go
// CorrectTranscript uses an LLM to fix misrecognized domain terms.
func CorrectTranscript(ctx context.Context, model llm.ChatModel, raw string, terms []string) (string, error) {
	prompt := fmt.Sprintf(
		"Fix the spelling of technical terms in this transcript. "+
			"Known terms: %s\n\nTranscript: %s\n\nCorrected transcript:",
		strings.Join(terms, ", "),
		raw,
	)

	result, err := model.Generate(ctx, []schema.Message{
		schema.NewUserMessage(schema.TextContent(prompt)),
	})
	if err != nil {
		return raw, fmt.Errorf("correction failed: %w", err)
	}

	return result.Content, nil
}
```

This approach adds latency and should only be used when high accuracy is required and the vocabulary is not adequately handled by prompt-based guidance.

## Verification

1. Record a sentence containing technical terms: "Deploy the gRPC service to Kubernetes using Beluga."
2. Transcribe without vocabulary guidance and note the errors.
3. Transcribe with the Whisper prompt or Deepgram keywords and compare accuracy.
4. For multi-domain use, verify that switching industries produces the correct vocabulary context.

## Next Steps

- [Real-time STT Streaming](/tutorials/voice/stt-realtime-streaming) -- Low-latency streaming transcription
- [Cloning Voices with ElevenLabs](/tutorials/voice/elevenlabs-cloning) -- High-fidelity text-to-speech
- [Sensitivity Tuning](/tutorials/voice/sensitivity-tuning) -- Balance VAD sensitivity for your environment
