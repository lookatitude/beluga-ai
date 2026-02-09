---
title: "VAD Sensitivity Profiles"
description: "Define named VAD sensitivity profiles for different environments from quiet offices to noisy factories."
---

## Problem

You need different VAD sensitivity settings per environment (quiet office vs noisy factory vs vehicle) or per use case (wake word vs barge-in vs segmentation). Hard-coding a single config leads to either false triggers or missed speech when switching contexts.

## Solution

Define **sensitivity profiles** (e.g. `sensitive`, `balanced`, `robust`) as named configs. Each profile sets `Threshold`, `MinSpeechDuration`, `MaxSilenceDuration`, and optionally `EnablePreprocessing`. Create the VAD provider with the chosen profile at runtime.

## Code Example

```go
package main

import (
	"context"
	"log"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/lookatitude/beluga-ai/voice/vad"
	vadiface "github.com/lookatitude/beluga-ai/voice/vad/iface"
	_ "github.com/lookatitude/beluga-ai/voice/vad/providers/silero"
)

var tracer = otel.Tracer("beluga.voice.vad.profiles")

const (
	ProfileSensitive = "sensitive"
	ProfileBalanced  = "balanced"
	ProfileRobust    = "robust"
)

func main() {
	ctx := context.Background()
	ctx, span := tracer.Start(ctx, "vad_profile_example")
	defer span.End()

	profile := ProfileBalanced
	span.SetAttributes(attribute.String("vad.profile", profile))

	provider, err := NewVADProviderForProfile(ctx, "silero", profile)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(trace.StatusError, err.Error())
		log.Fatalf("vad: %v", err)
	}

	audio := make([]byte, 1024)
	speech, err := provider.Process(ctx, audio)
	if err != nil {
		span.RecordError(err)
		log.Fatalf("process: %v", err)
	}
	span.SetAttributes(attribute.Bool("vad.speech", speech))
	_ = speech
}

// NewVADProviderForProfile creates a VAD provider with the given sensitivity profile.
func NewVADProviderForProfile(ctx context.Context, providerName, profile string) (vadiface.VADProvider, error) {
	cfg := vad.DefaultConfig()
	var opts []vad.ConfigOption
	switch profile {
	case ProfileSensitive:
		opts = []vad.ConfigOption{
			vad.WithThreshold(0.4),
			vad.WithMinSpeechDuration(150 * time.Millisecond),
			vad.WithMaxSilenceDuration(400 * time.Millisecond),
			vad.WithEnablePreprocessing(true),
		}
	case ProfileBalanced:
		opts = []vad.ConfigOption{
			vad.WithThreshold(0.5),
			vad.WithMinSpeechDuration(200 * time.Millisecond),
			vad.WithMaxSilenceDuration(500 * time.Millisecond),
			vad.WithEnablePreprocessing(true),
		}
	case ProfileRobust:
		opts = []vad.ConfigOption{
			vad.WithThreshold(0.6),
			vad.WithMinSpeechDuration(250 * time.Millisecond),
			vad.WithMaxSilenceDuration(600 * time.Millisecond),
			vad.WithEnablePreprocessing(true),
		}
	}

	return vad.NewProvider(ctx, providerName, cfg, opts...)
}
```

## Explanation

1. **Profiles** -- `sensitive`: lower threshold and shorter durations for wake word or quiet settings. `balanced`: default-like settings. `robust`: higher threshold and longer durations for noisy environments.

2. **Config options** -- `WithThreshold`, `WithMinSpeechDuration`, `WithMaxSilenceDuration`, and `WithEnablePreprocessing` are applied per profile. Add `WithModelPath` or `WithSampleRate` when needed.

3. **Runtime selection** -- Choose profile via environment variable, config file, or request context. The same provider creation path works for all profiles.

**Key insight:** Centralizing profiles avoids scattered magic numbers and makes it easy to A/B test or deploy environment-specific defaults.

## Variations

### Per-Tenant Profiles

Store profile name in tenant or device config; resolve when creating the VAD provider.

### Custom Profiles from Config File

Load profiles from YAML/JSON (threshold, durations, etc.) and build `ConfigOption` slices dynamically.

### Hybrid (Silero + Energy)

Use `silero` for `balanced`/`robust` and `energy` for `sensitive` low-CPU cases; keep the same profile interface.

## Related Recipes

- **[Background Noise Reduction](./background-noise)** -- STT preprocessing for noisy environments
- **[ML-Based Barge-In](./ml-barge-in)** -- VAD combined with turn detection
- **[Voice Backends Configuration](./voice-backends)** -- Backend provider setup
