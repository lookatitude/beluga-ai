---
title: Audio Analysis with Multimodal Models
description: Analyze audio files beyond simple transcription using Beluga AI's AudioPart type for meeting summarization, sentiment detection, and question answering.
---

Audio is a rich data source often trapped in large files. Multimodal models that accept audio input allow you to "chat" with recordings -- summarizing meetings, detecting speaker sentiment, and answering questions about specific moments. Beluga AI's `schema.AudioPart` provides a uniform interface for sending audio to capable providers.

## What You Will Build

An audio analysis pipeline that sends audio files to a multimodal model for summarization, sentiment detection, and interactive question answering. You will build a meeting summarizer that extracts key decisions and action items.

## Prerequisites

- An LLM provider API key for a model with audio support (Google Gemini 1.5 Pro or later)
- Audio files in supported formats (MP3, WAV, M4A)

## Core Concepts

### Audio Content Parts

The `schema.AudioPart` type carries raw audio data with format metadata:

```go
import "github.com/lookatitude/beluga-ai/schema"

audioPart := schema.AudioPart{
    Data:       audioBytes,
    Format:     "mp3",
    SampleRate: 44100,
}
```

### Messages with Audio

Combine text prompts with audio parts in a single message:

```go
msg := &schema.HumanMessage{
    Parts: []schema.ContentPart{
        schema.TextPart{Text: "Summarize the key decisions from this meeting."},
        schema.AudioPart{Data: audioBytes, Format: "mp3"},
    },
}
```

## Step 1: Initialize an Audio-Capable Model

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/schema"

    _ "github.com/lookatitude/beluga-ai/llm/providers/google"
)

func main() {
    model, err := llm.New("google", llm.ProviderConfig{
        Options: map[string]any{
            "api_key": os.Getenv("GOOGLE_API_KEY"),
            "model":   "gemini-1.5-pro",
        },
    })
    if err != nil {
        fmt.Printf("model creation failed: %v\n", err)
        return
    }

    ctx := context.Background()
    _ = model
    _ = ctx
}
```

## Step 2: Summarize a Meeting Recording

Load an audio file and request a meeting summary:

```go
func summarizeMeeting(ctx context.Context, model llm.ChatModel) {
    audioData, err := os.ReadFile("meeting_recording.mp3")
    if err != nil {
        fmt.Printf("read error: %v\n", err)
        return
    }

    msgs := []schema.Message{
        schema.NewSystemMessage(
            "You are a meeting summarizer. Extract key decisions, action items, " +
                "and participants from audio recordings. Format as structured bullet points.",
        ),
        &schema.HumanMessage{
            Parts: []schema.ContentPart{
                schema.TextPart{Text: "Summarize the key decisions made in this meeting."},
                schema.AudioPart{
                    Data:   audioData,
                    Format: "mp3",
                },
            },
        },
    }

    aiMsg, err := model.Generate(ctx, msgs)
    if err != nil {
        fmt.Printf("error: %v\n", err)
        return
    }

    fmt.Println("Meeting Summary:")
    fmt.Println(aiMsg.Text())
}
```

## Step 3: Detect Sentiment

Ask the model to analyze the emotional tone at specific points in the recording:

```go
func analyzeSentiment(ctx context.Context, model llm.ChatModel, audioData []byte) {
    msgs := []schema.Message{
        &schema.HumanMessage{
            Parts: []schema.ContentPart{
                schema.TextPart{
                    Text: "Analyze the speakers' tone throughout this recording. " +
                        "For each notable shift in sentiment, describe what changed and approximately when.",
                },
                schema.AudioPart{
                    Data:   audioData,
                    Format: "mp3",
                },
            },
        },
    }

    aiMsg, err := model.Generate(ctx, msgs)
    if err != nil {
        fmt.Printf("error: %v\n", err)
        return
    }

    fmt.Println("Sentiment Analysis:")
    fmt.Println(aiMsg.Text())
}
```

## Step 4: Interactive Question Answering

Ask specific questions about the audio content:

```go
func askAboutAudio(ctx context.Context, model llm.ChatModel, audioData []byte, question string) {
    msgs := []schema.Message{
        &schema.HumanMessage{
            Parts: []schema.ContentPart{
                schema.TextPart{Text: question},
                schema.AudioPart{
                    Data:   audioData,
                    Format: "mp3",
                },
            },
        },
    }

    aiMsg, err := model.Generate(ctx, msgs)
    if err != nil {
        fmt.Printf("error: %v\n", err)
        return
    }

    fmt.Printf("Q: %s\nA: %s\n", question, aiMsg.Text())
}
```

Usage:

```go
func main() {
    // ... initialize model ...

    audioData, err := os.ReadFile("lecture.mp3")
    if err != nil {
        fmt.Printf("read error: %v\n", err)
        return
    }

    ctx := context.Background()
    askAboutAudio(ctx, model, audioData, "What was the main topic discussed?")
    askAboutAudio(ctx, model, audioData, "Did anyone disagree with the proposal?")
    askAboutAudio(ctx, model, audioData, "What action items were assigned?")
}
```

## Step 5: Build a Meeting Summarizer Service

Combine the patterns into a reusable service:

```go
// MeetingSummarizer provides audio analysis capabilities.
type MeetingSummarizer struct {
    model llm.ChatModel
}

func NewMeetingSummarizer(model llm.ChatModel) *MeetingSummarizer {
    return &MeetingSummarizer{model: model}
}

func (s *MeetingSummarizer) Summarize(ctx context.Context, audioPath string) (string, error) {
    audioData, err := os.ReadFile(audioPath)
    if err != nil {
        return "", fmt.Errorf("read audio: %w", err)
    }

    msgs := []schema.Message{
        schema.NewSystemMessage(
            "You are a meeting summarizer. Provide:\n" +
                "1. Key decisions\n" +
                "2. Action items with owners\n" +
                "3. Open questions\n" +
                "4. Next steps",
        ),
        &schema.HumanMessage{
            Parts: []schema.ContentPart{
                schema.TextPart{Text: "Analyze this meeting recording."},
                schema.AudioPart{Data: audioData, Format: "mp3"},
            },
        },
    }

    aiMsg, err := s.model.Generate(ctx, msgs)
    if err != nil {
        return "", fmt.Errorf("generate: %w", err)
    }

    return aiMsg.Text(), nil
}
```

## Supported Audio Formats

Audio format support depends on the provider:

| Provider | Formats | Max Duration | Notes |
|----------|---------|-------------|-------|
| Google Gemini | MP3, WAV, M4A, FLAC, OGG | ~1 hour | Best audio support |
| OpenAI | WAV | Limited | Via Whisper for transcription |

For long audio files, consider splitting into segments and processing them sequentially, or using Google Cloud Storage URIs for large files.

## Verification

1. Record a short audio clip of a conversation.
2. Run the summarizer. Verify it identifies the main topics discussed.
3. Ask a specific question about the audio content. Verify the answer is relevant.
4. Test with different audio formats to verify format handling.

## Next Steps

- [Visual Reasoning](/tutorials/multimodal/visual-reasoning) -- Analyze images alongside text
- [Lazy-Loading Documents](/tutorials/documents/lazy-loading) -- Process large collections of media files efficiently
