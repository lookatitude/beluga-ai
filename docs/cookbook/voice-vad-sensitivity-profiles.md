---
title: "VAD Sensitivity Profiles"
package: "voice/vad"
category: "voice"
complexity: "intermediate"
---

# VAD Sensitivity Profiles

## Problem

You need different VAD sensitivity settings per environment (e.g. quiet office vs noisy factory vs vehicle) or per use case (wake word vs barge-in vs segmentation). Hard-coding a single config leads to either false triggers or missed speech when switching contexts.

## Solution

Define **sensitivity profiles** (e.g. `sensitive`, `balanced`, `robust`) as named configs. Each profile sets `Threshold`, `MinSpeechDuration`, `MaxSilenceDuration`, and optionally `EnablePreprocessing`. Create the VAD provider with the chosen profile at runtime. This works because `pkg/voice/vad` accepts `Config` and options; you map profile names to option sets and reuse them across sessions or deployments.

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

	"github.com/lookatitude/beluga-ai/pkg/voice/vad"
	vadiface "github.com/lookatitude/beluga-ai/pkg/voice/vad/iface"
	_ "github.com/lookatitude/beluga-ai/pkg/voice/vad/providers/silero"
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
	default:
		opts = nil
	}
text
	return vad.NewProvider(ctx, providerName, cfg, opts...)
}
```

## Explanation

1. **Profiles** — `sensitive`: lower threshold, shorter durations for wake word or quiet settings. `balanced`: default-like. `robust`: higher threshold, longer durations for noisy environments.

2. **Config options** — `WithThreshold`, `WithMinSpeechDuration`, `WithMaxSilenceDuration`, and `WithEnablePreprocessing` are applied per profile. Add `WithModelPath` or `WithSampleRate` when needed.

3. **Runtime selection** — Choose profile via env var, config, or request context. Use the same provider creation path for all profiles.

4. **OTEL** — Record `vad.profile` and `vad.speech` for tuning and debugging.

```go
**Key insight:** Centralizing profiles avoids scattered magic numbers and makes it easy to A/B test or deploy env-specific defaults.

## Testing

- Unit-test `NewVADProviderForProfile` for each profile; assert provider is created and (if possible) that `Process` behavior differs.
- Integration-test with synthetic audio; compare speech vs non-speech rates across profiles.

## Variations

### Per-tenant or per-device profiles

Store profile name in tenant or device config; resolve when creating the VAD provider.

### Custom profile from config file

Load profiles from YAML/JSON (threshold, durations, etc.) and build `ConfigOption` slices from them.

### Hybrid (Silero + energy)

Use `silero` for `balanced`/`robust` and `energy` for `sensitive` low-CPU cases; keep the same profile interface.

## Related Recipes

- **[Custom VAD with Silero](../tutorials/voice/voice-vad-custom-silero.md)** — Silero setup.
- **[Noise-Resistant VAD](../use-cases/voice-vad-noise-resistant.md)** — Noisy environments.
- **[Overcoming Background Noise](./voice-stt-overcoming-background-noise.md)** — STT in noise.

```
