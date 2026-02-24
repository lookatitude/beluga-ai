---
title: Fine-Tuning Whisper for Industry Terms
description: "Improve STT accuracy for specialized vocabulary in Go — use Whisper prompt guidance and Deepgram keyword boosting to correctly transcribe domain-specific terms."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, Go, tutorial, Whisper, STT, vocabulary, Deepgram, keyword boosting, transcription"
---

Standard speech-to-text models frequently misrecognize domain-specific terminology because they are trained on general speech corpora that underrepresent technical vocabulary. This tutorial demonstrates two approaches to improving accuracy: using Whisper's prompt parameter to provide contextual vocabulary, and using Deepgram's keyword boosting to increase the probability of specific terms. Both approaches work without model retraining and can be applied at configuration time.

## What You Will Build

A vocabulary-aware STT configuration that correctly transcribes technical and industry-specific terms such as "Kubernetes," "gRPC," and "Beluga AI" that standard models often misrecognize.

## Prerequisites

- OpenAI API key (for Whisper) or Deepgram API key
- A list of industry-specific terms relevant to your domain

## The Problem: Vocabulary Mismatch

Standard STT models are trained on general speech. When they encounter domain-specific terms, they approximate the closest-sounding common words. This produces transcription errors that propagate through the rest of your pipeline -- if the STT layer transcribes "Cooper Netties" instead of "Kubernetes," no amount of downstream processing can recover the original intent:

| Spoken Term    | Typical Misrecognition |
|---------------|----------------------|
| gRPC          | "Gee RPS"            |
| Beluga AI     | "Blue guy eye"       |
| Kubernetes    | "Cooper Netties"     |
| pgvector      | "PG Vector"          |

## Step 1: Whisper Prompt-Based Guidance

OpenAI's Whisper API accepts a `Prompt` parameter that provides the model with vocabulary context. Unlike a system instruction, the prompt is treated as prior transcript text that biases the model toward expected terms. By including the exact spelling of your domain terms in the prompt, you shift the model's probability distribution to favor those terms when it encounters similar-sounding audio.

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

To pass the prompt for vocabulary guidance, use provider-specific configuration. The `ProviderSpecific` map allows passing options that are unique to a particular provider without polluting the shared `stt.Config` interface:

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

Deepgram's API supports a `Keywords` parameter that explicitly boosts the probability of specific terms during recognition. This approach is more precise than prompt-based guidance because each keyword receives an individual boost level, allowing you to weight terms by their importance or by how frequently they are misrecognized. Each keyword can include a numeric boost value after a colon.

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

The boost value after the colon controls intensity. Values above 1.0 increase recognition probability; the default is 1.0. Higher boost values make the model more likely to select that term, but excessively high values can cause false positives where the model forces the keyword even when the speaker said something different.

## Step 3: Vocabulary-Aware Wrapper

For applications that serve multiple domains, create a wrapper that selects the appropriate vocabulary context based on the active conversation. This pattern keeps the domain-specific configuration separate from the STT pipeline logic, making it straightforward to add new industries without modifying the core transcription flow.

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

When the model still misrecognizes terms despite prompting, apply a post-processing step that uses an LLM to correct known vocabulary in the transcript. This is a fallback strategy for cases where acoustic similarity is too high for the STT model to disambiguate. The LLM has access to the vocabulary list and can use semantic context to correct errors that the STT model cannot resolve from audio alone.

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

This approach adds latency from the LLM call and should only be used when high accuracy is required and the vocabulary is not adequately handled by prompt-based guidance. For real-time voice applications, consider running the correction asynchronously and using the uncorrected transcript for initial response generation.

## Verification

1. Record a sentence containing technical terms: "Deploy the gRPC service to Kubernetes using Beluga."
2. Transcribe without vocabulary guidance and note the errors.
3. Transcribe with the Whisper prompt or Deepgram keywords and compare accuracy.
4. For multi-domain use, verify that switching industries produces the correct vocabulary context.

## Next Steps

- [Real-time STT Streaming](/docs/tutorials/voice/stt-realtime-streaming) -- Low-latency streaming transcription
- [Cloning Voices with ElevenLabs](/docs/tutorials/voice/elevenlabs-cloning) -- High-fidelity text-to-speech
- [Sensitivity Tuning](/docs/tutorials/voice/sensitivity-tuning) -- Balance VAD sensitivity for your environment
