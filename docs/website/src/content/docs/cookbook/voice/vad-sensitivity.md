---
title: "VAD Sensitivity Profiles"
description: "Define named VAD sensitivity profiles for different environments from quiet offices to noisy factories."
---

## Problem

Voice activity detection must adapt to dramatically different acoustic environments and use cases. A VAD threshold tuned for a quiet office produces false triggers in a noisy factory, detecting machine sounds as speech. A threshold tuned for noisy environments misses quiet speech in office settings, causing the system to ignore users who speak softly. Beyond environment, different use cases require different sensitivity: wake word detection demands high sensitivity to catch every utterance, barge-in detection needs balanced sensitivity to distinguish intentional interruptions from background speech, and turn segmentation requires lower sensitivity to avoid fragmenting continuous speech. Hard-coding a single configuration forces compromises that work poorly in all contexts.

## Solution

Define sensitivity profiles as named configuration bundles that set threshold, minimum speech duration, maximum silence duration, and preprocessing flags together. Each profile represents a coherent set of parameters validated for specific conditions. The `sensitive` profile uses low thresholds and short durations for wake word detection in quiet environments. The `balanced` profile provides general-purpose settings that work adequately in most conditions. The `robust` profile uses high thresholds and longer durations to reduce false positives in noisy environments. Create VAD providers with the chosen profile at runtime, allowing environment-specific or per-user configuration without code changes.

This pattern leverages Beluga's registry pattern: providers are registered via `init()`, instantiated through `NewProvider()` with functional options, and discovered via `List()`. Profile selection becomes a configuration concern rather than a code concern, enabling A/B testing, tenant-specific defaults, and runtime switching based on measured noise levels.

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

The code shows profile selection as a runtime decision based on a string constant, which can come from environment variables, configuration files, or request metadata. Functional options (`vad.WithThreshold()`, etc.) compose to build provider configurations, following Beluga's standard pattern for extensible components.

## Explanation

1. **Profiles** -- Each profile bundles parameters that work together coherently. `sensitive` uses a 0.4 threshold (40% confidence of speech triggers detection), 150ms minimum speech duration (short fragments count), and 400ms maximum silence (quick turn boundaries). This configuration minimizes missed detections but increases false positives, appropriate for wake word scenarios where missing an utterance is worse than occasionally triggering on noise. `balanced` increases thresholds and durations moderately, trading some sensitivity for fewer false positives. `robust` prioritizes precision over recall, using high thresholds (0.6) and long durations (250ms speech, 600ms silence) to ensure detected speech is genuine, even at the cost of missing some quiet or fragmented utterances.

2. **Config options** -- Beluga's functional options pattern allows composing configuration incrementally. `WithThreshold` sets the speech probability threshold. `WithMinSpeechDuration` filters out transient noise spikes that briefly exceed the threshold. `WithMaxSilenceDuration` determines how long silence is tolerated before considering speech complete. `WithEnablePreprocessing` activates audio normalization and filtering, improving detection quality at the cost of increased CPU usage. Additional options like `WithModelPath` (for ONNX models) or `WithSampleRate` can be added per provider as needed.

3. **Runtime selection** -- Choosing profiles via configuration rather than code enables flexible deployment. Store profile names in tenant databases, device preferences, or environment variables. The same `NewVADProviderForProfile` function works for all profiles, avoiding conditional provider logic throughout the codebase. This pattern supports A/B testing different profiles, gradual rollouts of sensitivity changes, and user-customizable sensitivity without redeployment.

**Key insight:** Centralizing profiles avoids magic numbers scattered throughout the codebase and makes acoustic tuning manageable. Instead of developers guessing appropriate thresholds per use case, profiles capture validated configurations that work together. Acoustic engineers can tune profiles based on measured false positive and false negative rates without developer involvement. Document profile characteristics (expected environment, false positive rate, false negative rate) to guide selection.

## Variations

### Per-Tenant Profiles

Store profile name in tenant or device configuration, resolving it when creating VAD providers. This allows tailoring sensitivity per customer based on their typical usage environment without maintaining separate codebases.

### Custom Profiles from Config File

Load profiles from YAML or JSON files with fields for threshold, speech duration, silence duration, and preprocessing flags. Build `ConfigOption` slices dynamically from parsed configuration, enabling ops teams to tune profiles without code changes.

### Hybrid (Silero + Energy)

Use different underlying providers per profile: `silero` for `balanced` and `robust` (high accuracy, higher CPU), `energy` for `sensitive` (fast, lower accuracy). Keep the same profile interface while switching implementation based on accuracy versus performance tradeoffs.

## Related Recipes

- **[Background Noise Reduction](./background-noise)** -- STT preprocessing for noisy environments
- **[ML-Based Barge-In](./ml-barge-in)** -- VAD combined with turn detection
- **[Voice Backends Configuration](./voice-backends)** -- Backend provider setup
