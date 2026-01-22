# Configuring Voice Reasoning Modes

<!--
Persona: Pair Programmer Colleague
- Treat the reader as a competent peer
- Be helpful, direct, and slightly informal but professional
- Focus on getting results quickly
- Keep it concise and functional
- Provide runnable code examples
-->

## What you will build
In this tutorial, we'll tune Speech-to-Speech (S2S) models for different interaction styles. You'll learn how to optimize for ultra-fast "Concierge" mode vs. deep-thinking "Tutor" mode, and implement dynamic mode switching based on user intent.

## Learning Objectives
- ✅ Configure reasoning depth in S2S
- ✅ Optimize for latency vs. quality
- ✅ Implement context-aware mode switching
- ✅ Use OpenAI Realtime reasoning features

## Introduction
Welcome, colleague! S2S models are unique because they can "think" while they speak. But more thinking often means more latency. Let's look at how to balance speed and intelligence by configuring reasoning modes for different use cases.

## Prerequisites

- [Native S2S with Amazon Nova](./voice-s2s-amazon-nova.md) or OpenAI Realtime API Key

## The Trade-off: Speed vs. Thought

S2S models often have internal "reasoning" steps. 
- **Fast**: Immediate response, simple sentences.
- **Deep**: Thinking time, complex logic, nuanced emotion.

## Step 1: Configuring OpenAI Realtime Modes

If using the OpenAI provider, you can adjust the `reasoning_effort`.
```go
package main

import "github.com/lookatitude/beluga-ai/pkg/voice/s2s/providers/openai_realtime"

func main() {
    config := &openai_realtime.Config{
        APIKey: os.Getenv("OPENAI_API_KEY"),
        // "low" = fastest, "medium" = balanced, "high" = smart
        ReasoningEffort: "low", 
        
        // Turn Detection sensitivity
        TurnDetection: &openai_realtime.TurnDetectionConfig{
            Threshold: 0.5,
            SilenceDuration: 500, // ms
        },
    }
}
```

## Step 2: System Prompts for Reasoning

Just like LLMs, S2S models are guided by system prompts.
```go
const conciergePrompt = "You are a fast hotel concierge. Give short, 1-sentence answers."
const tutorPrompt = "You are a patient math tutor. Explain your reasoning step-by-step."

session.UpdateSession(s2s.SessionUpdate{
    Instructions: conciergePrompt,
})
```

## Step 3: Latency Optimization (Glass-to-Glass)

To achieve "Glass-to-Glass" (Mic to Speaker) latency < 1s:
1. **Reduce Chunk Size**: Send smaller audio packets (20ms).
2. **Use VAD**: Enable Voice Activity Detection on the server side (S2S native).
3. **Turn Detection**: Use "Server VAD" mode for faster response triggers.

```go
config.TurnDetectionType = "server_vad"
```

## Step 4: Dynamic Mode Switching

Switch modes based on the user's intent.
```go
func handleIntent(intent string, session s2s.Session) {
    if intent == "PROBLEM_SOLVING" {
        session.UpdateSession(s2s.SessionUpdate{
            Instructions: tutorPrompt,
            Temperature: 0.7,
        })
    } else {
        session.UpdateSession(s2s.SessionUpdate{
            Instructions: conciergePrompt,
            Temperature: 0.1,
        })
    }
}
```

## Step 5: Handling Speech-to-Speech Interruption

Configure how sensitive the model is to being interrupted.
config.InputTranscription = true // Get text of what user is saying
config.InterruptionThreshold = 0.8 // 0.0 to 1.0
```

## Verification

1. Set `ReasoningEffort` to `low`. Ask "What is 2+2?".
2. Set `ReasoningEffort` to `high`. Ask "Explain the theory of relativity".
3. Measure the time until audio starts playing in both cases.

## Next Steps

- **[Native S2S with Amazon Nova](./voice-s2s-amazon-nova.md)** - Try different providers.
- **[Real-time STT Streaming](./voice-stt-realtime-streaming.md)** - Traditional pipeline comparison.
