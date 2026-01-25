---
title: "SSML Emphasis & Pause Tuning"
package: "voice/tts"
category: "audio"
complexity: "intermediate"
---

# SSML Emphasis & Pause Tuning

## Problem

You need to control emphasis, pauses, and intonation in text-to-speech output to make speech sound more natural and convey meaning effectively.

## Solution

Implement SSML (Speech Synthesis Markup Language) processing that inserts emphasis tags, pause breaks, and prosody controls into text before TTS synthesis. This works because TTS providers support SSML, allowing fine-grained control over speech output.

## Code Example
```go
package main

import (
    "context"
    "fmt"
    "regexp"
    "strings"
    "time"
    
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("beluga.voice.tts.ssml")

// SSMLProcessor processes text with SSML enhancements
type SSMLProcessor struct {
    addEmphasis bool
    addPauses   bool
    defaultPause time.Duration
}

// NewSSMLProcessor creates a new SSML processor
func NewSSMLProcessor(addEmphasis, addPauses bool, defaultPause time.Duration) *SSMLProcessor {
    return &SSMLProcessor{
        addEmphasis: addEmphasis,
        addPauses:   addPauses,
        defaultPause: defaultPause,
    }
}

// ProcessText processes text with SSML enhancements
func (sp *SSMLProcessor) ProcessText(ctx context.Context, text string) (string, error) {
    ctx, span := tracer.Start(ctx, "ssml_processor.process")
    defer span.End()
    
    processed := text
    
    // Add emphasis to important words/phrases
    if sp.addEmphasis {
        processed = sp.addEmphasisTags(ctx, processed)
        span.SetAttributes(attribute.Bool("emphasis_added", true))
    }
    
    // Add pauses at punctuation
    if sp.addPauses {
        processed = sp.addPauseTags(ctx, processed)
        span.SetAttributes(attribute.Bool("pauses_added", true))
    }
    
    // Wrap in SSML
    ssml := fmt.Sprintf("<speak >%s</speak>", processed)
    
    span.SetAttributes(attribute.Int("output_length", len(ssml)))
    span.SetStatus(trace.StatusOK, "SSML processed")
    
    return ssml, nil
}

// addEmphasisTags adds emphasis to important words
func (sp *SSMLProcessor) addEmphasisTags(ctx context.Context, text string) string {
    // Emphasis patterns (words in quotes, ALL CAPS, etc.)
    patterns := []struct {
        pattern *regexp.Regexp
        level   string
    }{
        {regexp.MustCompile(`"([^"]+)"`), "moderate"}, // Quoted text
        {regexp.MustCompile(`\b([A-Z]{2,})\b`), "strong"}, // ALL CAPS
        {regexp.MustCompile(`\*\*([^*]+)\*\*`), "moderate"}, // Markdown bold
    }
    
    for _, p := range patterns {
        text = p.pattern.ReplaceAllStringFunc(text, func(match string) string {
            // Extract content
            content := p.pattern.FindStringSubmatch(match)[1]
            return fmt.Sprintf(`<emphasis level="%s">%s</emphasis>`, p.level, content)
        })
    }
    
    return text
}

// addPauseTags adds pauses at punctuation
func (sp *SSMLProcessor) addPauseTags(ctx context.Context, text string) string {
    // Add pauses after punctuation
    pausePatterns := map[string]time.Duration{
        ".":  500 * time.Millisecond,
        "!":  500 * time.Millisecond,
        "?":  500 * time.Millisecond,
        ",":  250 * time.Millisecond,
        ";":  300 * time.Millisecond,
        ":":  300 * time.Millisecond,
    }

    for punct, duration := range pausePatterns {
        breakTag := fmt.Sprintf(`<break time="%dms"/>`, duration.Milliseconds())
        text = strings.ReplaceAll(text, punct, punct+breakTag)
    }
    
    return text
}

// AddCustomPause adds a custom pause
func (sp *SSMLProcessor) AddCustomPause(ctx context.Context, text string, position int, duration time.Duration) string {
    if position < 0 || position > len(text) {
        return text
    }
    
    breakTag := fmt.Sprintf(`<break time="%dms"/>`, duration.Milliseconds())
    return text[:position] + breakTag + text[position:]
}

// SetProsody sets prosody (rate, pitch, volume)
func (sp *SSMLProcessor) SetProsody(ctx context.Context, text string, rate string, pitch string, volume string) string {
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

    prosodyTag := fmt.Sprintf(`<prosody %s>%s</prosody>`, strings.Join(attrs, " "), text)
    return prosodyTag
}

func main() {
    ctx := context.Background()

    // Create processor
    processor := NewSSMLProcessor(true, true, 500*time.Millisecond)
    
    // Process text
    text := "Hello! This is IMPORTANT. How are you?"
    ssml, err := processor.ProcessText(ctx, text)
    if err != nil {
        log.Fatalf("Failed to process: %v", err)
    }
    fmt.Printf("SSML: %s\n", ssml)
}
```

## Explanation

Let's break down what's happening:

1. **Emphasis detection** - Notice how we detect important words (quotes, ALL CAPS, markdown) and wrap them in SSML emphasis tags. This adds natural stress to important content.

2. **Pause insertion** - We add pause breaks after punctuation marks. Longer pauses after sentences, shorter after commas, creating natural rhythm.

3. **Prosody control** - We can control rate, pitch, and volume using SSML prosody tags. This allows fine-tuning speech characteristics.

```go
**Key insight:** Use SSML strategically to enhance naturalness. Too much SSML sounds robotic; too little sounds monotone. Balance is key.

## Testing

```
Here's how to test this solution:
```go
func TestSSMLProcessor_AddsPauses(t *testing.T) {
    processor := NewSSMLProcessor(false, true, 500*time.Millisecond)
    
    text := "Hello. How are you?"
    ssml, err := processor.ProcessText(context.Background(), text)
    
    require.NoError(t, err)
    require.Contains(t, ssml, "<break")
}

## Variations

### Dynamic Pause Adjustment

Adjust pauses based on context:
func (sp *SSMLProcessor) ContextualPauses(ctx context.Context, text string) string {
    // Longer pauses for questions, shorter for lists
}
```

### Voice-Specific SSML

Customize SSML for different voices:
```go
func (sp *SSMLProcessor) ProcessForVoice(ctx context.Context, text string, voiceID string) string {
    // Voice-specific SSML enhancements
}
```

## Related Recipes

- **[Voice TTS Multi-speaker Dialogue Synthesis](./voice-tts-multi-speaker-dialogue-synthesis.md)** - Multi-speaker synthesis
- **[Voice Providers Guide](../guides/voice-providers.md)** - For a deeper understanding of TTS
