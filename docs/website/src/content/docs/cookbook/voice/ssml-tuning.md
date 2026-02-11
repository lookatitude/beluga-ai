---
title: "SSML Emphasis and Pause Tuning"
description: "Control emphasis, pauses, and prosody in TTS output with SSML markup for natural-sounding speech."
---

## Problem

You need to control emphasis, pauses, and intonation in text-to-speech output to make speech sound more natural and convey meaning effectively. Flat, monotone TTS output fails to communicate emotion, urgency, or sentence structure. Users interpret speech differently based on prosody, and without proper emphasis or pacing, TTS sounds robotic and difficult to understand. This becomes critical for applications like voice assistants, audiobooks, or customer service bots where user engagement depends on natural-sounding speech.

## Solution

Implement SSML (Speech Synthesis Markup Language) processing that inserts emphasis tags, pause breaks, and prosody controls into text before TTS synthesis. TTS providers support SSML, allowing fine-grained control over speech output. The strategy here is to detect emphasis cues in the source text (quotation marks, capitalization, markdown bold) and translate them into SSML tags that the TTS engine understands. Similarly, punctuation marks map to natural pause durations, recreating the rhythm of human speech.

This approach separates content generation from prosody tuning. Your agent generates plain text with emphasis markers, and the SSML processor applies voice-specific tuning without changing the agent's output logic. This makes it easy to adjust prosody across different TTS providers or voices without retraining or modifying upstream components.

## Code Example

```go
package main

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("beluga.voice.tts.ssml")

// SSMLProcessor processes text with SSML enhancements.
type SSMLProcessor struct {
	addEmphasis  bool
	addPauses    bool
	defaultPause time.Duration
}

func NewSSMLProcessor(addEmphasis, addPauses bool, defaultPause time.Duration) *SSMLProcessor {
	return &SSMLProcessor{
		addEmphasis:  addEmphasis,
		addPauses:    addPauses,
		defaultPause: defaultPause,
	}
}

// ProcessText applies SSML enhancements to text.
func (sp *SSMLProcessor) ProcessText(ctx context.Context, text string) (string, error) {
	ctx, span := tracer.Start(ctx, "ssml_processor.process")
	defer span.End()

	processed := text

	if sp.addEmphasis {
		processed = sp.addEmphasisTags(processed)
		span.SetAttributes(attribute.Bool("emphasis_added", true))
	}

	if sp.addPauses {
		processed = sp.addPauseTags(processed)
		span.SetAttributes(attribute.Bool("pauses_added", true))
	}

	ssml := fmt.Sprintf("<speak>%s</speak>", processed)

	span.SetAttributes(attribute.Int("output_length", len(ssml)))
	span.SetStatus(trace.StatusOK, "SSML processed")

	return ssml, nil
}

func (sp *SSMLProcessor) addEmphasisTags(text string) string {
	patterns := []struct {
		pattern *regexp.Regexp
		level   string
	}{
		{regexp.MustCompile(`"([^"]+)"`), "moderate"},
		{regexp.MustCompile(`\b([A-Z]{2,})\b`), "strong"},
		{regexp.MustCompile(`\*\*([^*]+)\*\*`), "moderate"},
	}

	for _, p := range patterns {
		text = p.pattern.ReplaceAllStringFunc(text, func(match string) string {
			content := p.pattern.FindStringSubmatch(match)[1]
			return fmt.Sprintf(`<emphasis level="%s">%s</emphasis>`, p.level, content)
		})
	}

	return text
}

func (sp *SSMLProcessor) addPauseTags(text string) string {
	pausePatterns := map[string]time.Duration{
		".": 500 * time.Millisecond,
		"!": 500 * time.Millisecond,
		"?": 500 * time.Millisecond,
		",": 250 * time.Millisecond,
		";": 300 * time.Millisecond,
		":": 300 * time.Millisecond,
	}

	for punct, duration := range pausePatterns {
		breakTag := fmt.Sprintf(`<break time="%dms"/>`, duration.Milliseconds())
		text = strings.ReplaceAll(text, punct, punct+breakTag)
	}

	return text
}

// AddCustomPause inserts a pause at a specific position.
func (sp *SSMLProcessor) AddCustomPause(text string, position int, duration time.Duration) string {
	if position < 0 || position > len(text) {
		return text
	}
	breakTag := fmt.Sprintf(`<break time="%dms"/>`, duration.Milliseconds())
	return text[:position] + breakTag + text[position:]
}

// SetProsody wraps text with prosody controls for rate, pitch, and volume.
func (sp *SSMLProcessor) SetProsody(text string, rate string, pitch string, volume string) string {
	attrs := []string{}
	if rate != "" {
		attrs = append(attrs, fmt.Sprintf(`rate="%s"`, rate))
	}
	if pitch != "" {
		attrs = append(attrs, fmt.Sprintf(`pitch="%s"`, pitch))
	}
	if volume != "" {
		attrs = append(attrs, fmt.Sprintf(`volume="%s"`, volume))
	}
	return fmt.Sprintf(`<prosody %s>%s</prosody>`, strings.Join(attrs, " "), text)
}

func main() {
	ctx := context.Background()

	processor := NewSSMLProcessor(true, true, 500*time.Millisecond)

	text := "Hello! This is IMPORTANT. How are you?"
	ssml, err := processor.ProcessText(ctx, text)
	if err != nil {
		log.Fatalf("Failed to process: %v", err)
	}
	fmt.Printf("SSML: %s\n", ssml)
}
```

## Explanation

1. **Emphasis detection** -- Important words are detected by pattern (quoted text, ALL CAPS, markdown bold) and wrapped in SSML emphasis tags with appropriate levels. This design leverages existing text conventions rather than requiring explicit markup from the agent. Quoted text receives moderate emphasis because it often represents reported speech or key terms. ALL CAPS signals strong emphasis (though use sparingly to avoid sounding aggressive). The regex patterns are applied sequentially to avoid nested tags, which many TTS engines reject.

2. **Pause insertion** -- Pause breaks are added after punctuation marks. Longer pauses after sentences (500ms), shorter after commas (250ms), creating natural rhythm. This mimics human speech pacing, where sentence boundaries receive longer pauses for comprehension, while mid-sentence commas provide brief breathing room. The durations here are conservative defaults; tune them per voice and speaking rate. Without pauses, TTS rushes through text, making it hard for listeners to parse sentence structure.

3. **Prosody control** -- Rate, pitch, and volume can be controlled using SSML prosody tags, allowing fine-tuning of speech characteristics per segment. Prosody adjustments let you shift tone dynamically: slow down for emphasis, raise pitch for questions, lower volume for asides. This is particularly valuable for multi-turn dialogues where the agent's mood or urgency changes. The functional option pattern (`SetProsody`) allows selective application without cluttering the main processor.

**Key insight:** Use SSML strategically to enhance naturalness. Too much SSML sounds robotic; too little sounds monotone. Balance is key. Over-tagging with emphasis makes every word sound urgent, which dilutes actual importance. Over-pausing creates awkward silence. Start with conservative defaults and tune based on user feedback or A/B testing. SSML is a tool for subtle correction, not a replacement for well-written prompts.

## Variations

### Dynamic Pause Adjustment

Adjust pauses based on context. Longer pauses for questions, shorter for lists.

### Voice-Specific SSML

Customize SSML for different voice providers, since SSML support varies between services.

## Related Recipes

- **[Multi-Speaker Dialogue Synthesis](./multi-speaker-synthesis)** -- Multi-speaker TTS with distinct voices
- **[Voice Backends Configuration](./voice-backends)** -- Backend provider setup
- **[Glass-to-Glass Latency](./glass-to-glass-latency)** -- End-to-end latency optimization
