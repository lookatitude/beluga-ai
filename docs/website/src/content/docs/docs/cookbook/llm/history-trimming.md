---
title: "Multi-step History Trimming"
description: "Recipe for managing conversation history beyond token limits in Go with intelligent trimming, summarization, and sliding windows using Beluga AI."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, history trimming, Go context window, conversation management, token limits, sliding window, ContextManager recipe"
---

## Problem

You need to manage conversation history that grows beyond token limits, intelligently trimming old messages while preserving important context and maintaining conversation coherence.

## Solution

Implement a history trimmer that prioritizes recent messages, always preserves system messages, and optionally summarizes older messages using an LLM call. When summarization is unavailable or produces a result that is still too large, aggressive trimming removes the oldest non-system messages until the history fits.

## Why This Matters

Every LLM has a finite context window, and conversation history grows without bound. Without management, a chatbot that works perfectly for 10 exchanges will fail silently on exchange 50 when the history exceeds the model's token limit. Simple truncation from the beginning loses the system prompt and early context that defines the agent's behavior.

The three-tier strategy (preserve system messages → keep recent messages → summarize or drop old messages) ensures the most valuable context survives trimming. Summarization compresses potentially thousands of tokens into a concise recap so the agent retains awareness of earlier discussion without paying the full token cost.

## Code Example

```go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
)

var tracer = otel.Tracer("beluga.chatmodels.history_trimming")

// HistoryTrimmer manages conversation history with intelligent trimming.
type HistoryTrimmer struct {
	maxTokens  int
	keepRecent int
	summarizer llm.ChatModel // Optional. Pass nil to skip summarization.
}

// NewHistoryTrimmer creates a new history trimmer.
// maxTokens: approximate token budget. keepRecent: number of non-system messages to always retain.
func NewHistoryTrimmer(maxTokens, keepRecent int, summarizer llm.ChatModel) *HistoryTrimmer {
	return &HistoryTrimmer{
		maxTokens:  maxTokens,
		keepRecent: keepRecent,
		summarizer: summarizer,
	}
}

// TrimHistory trims messages to fit the token budget.
func (ht *HistoryTrimmer) TrimHistory(ctx context.Context, messages []schema.Message) ([]schema.Message, error) {
	ctx, span := tracer.Start(ctx, "history_trimmer.trim")
	defer span.End()

	span.SetAttributes(
		attribute.Int("input_message_count", len(messages)),
		attribute.Int("max_tokens", ht.maxTokens),
	)

	if ht.estimateTokens(messages) <= ht.maxTokens {
		span.SetAttributes(attribute.Bool("trimming_needed", false))
		span.SetStatus(trace.StatusOK, "no trimming needed")
		return messages, nil
	}

	systemMsgs, recentMsgs, oldMsgs := ht.categorize(messages)

	// Start with the must-keep messages.
	trimmed := append(systemMsgs, recentMsgs...)

	// Attempt to inject a summary of old messages if there is budget.
	usedTokens := ht.estimateTokens(trimmed)
	remainingTokens := ht.maxTokens - usedTokens

	if len(oldMsgs) > 0 && remainingTokens > 100 && ht.summarizer != nil {
		summary, err := ht.summarize(ctx, oldMsgs)
		if err == nil {
			summaryTokens := ht.estimateTokens([]schema.Message{summary})
			if summaryTokens <= remainingTokens {
				// Insert summary between system messages and recent messages.
				trimmed = append(systemMsgs, append([]schema.Message{summary}, recentMsgs...)...)
			}
		} else {
			slog.WarnContext(ctx, "summarization failed, skipping", "error", err)
		}
	}

	// Safety net: if still too large, drop oldest non-system messages.
	if ht.estimateTokens(trimmed) > ht.maxTokens {
		trimmed = ht.aggressiveTrim(trimmed)
	}

	span.SetAttributes(
		attribute.Int("output_message_count", len(trimmed)),
		attribute.Int("trimmed_count", len(messages)-len(trimmed)),
	)
	span.SetStatus(trace.StatusOK, "history trimmed")
	return trimmed, nil
}

// categorize splits messages into system, recent, and old buckets.
func (ht *HistoryTrimmer) categorize(messages []schema.Message) (system, recent, old []schema.Message) {
	for _, msg := range messages {
		if msg.GetRole() == schema.RoleSystem {
			system = append(system, msg)
		}
	}

	nonSystem := make([]schema.Message, 0, len(messages))
	for _, msg := range messages {
		if msg.GetRole() != schema.RoleSystem {
			nonSystem = append(nonSystem, msg)
		}
	}

	cutoff := len(nonSystem) - ht.keepRecent
	if cutoff < 0 {
		cutoff = 0
	}
	old = nonSystem[:cutoff]
	recent = nonSystem[cutoff:]
	return
}

// summarize creates a brief summary of old messages via an LLM call.
func (ht *HistoryTrimmer) summarize(ctx context.Context, messages []schema.Message) (schema.Message, error) {
	ctx, span := tracer.Start(ctx, "history_trimmer.summarize")
	defer span.End()

	var sb strings.Builder
	sb.WriteString("Summarize the following conversation history in 2-3 sentences:\n\n")
	for _, msg := range messages {
		// Use the typed Text() method to extract text from ContentPart slices.
		switch m := msg.(type) {
		case *schema.HumanMessage:
			fmt.Fprintf(&sb, "Human: %s\n", m.Text())
		case *schema.AIMessage:
			fmt.Fprintf(&sb, "AI: %s\n", m.Text())
		case *schema.SystemMessage:
			fmt.Fprintf(&sb, "System: %s\n", m.Text())
		default:
			fmt.Fprintf(&sb, "[%s message]\n", msg.GetRole())
		}
	}

	prompt := []schema.Message{
		schema.NewSystemMessage("You are a conversation summarizer. Create a concise summary that preserves key facts and decisions."),
		schema.NewHumanMessage(sb.String()),
	}

	response, err := ht.summarizer.Generate(ctx, prompt)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(trace.StatusError, err.Error())
		return nil, err
	}

	// response is *schema.AIMessage; extract text via the Text() method.
	summaryText := response.Text()
	span.SetStatus(trace.StatusOK, "summary created")
	return schema.NewSystemMessage("Previous conversation summary: " + summaryText), nil
}

// aggressiveTrim keeps system messages and as many recent messages as fit.
func (ht *HistoryTrimmer) aggressiveTrim(messages []schema.Message) []schema.Message {
	var sys, rest []schema.Message
	for _, m := range messages {
		if m.GetRole() == schema.RoleSystem {
			sys = append(sys, m)
		} else {
			rest = append(rest, m)
		}
	}

	tokens := ht.estimateTokens(sys)
	var kept []schema.Message

	// Add from most recent backwards.
	for i := len(rest) - 1; i >= 0; i-- {
		msgTokens := ht.estimateTokens([]schema.Message{rest[i]})
		if tokens+msgTokens > ht.maxTokens {
			break
		}
		kept = append([]schema.Message{rest[i]}, kept...)
		tokens += msgTokens
	}

	return append(sys, kept...)
}

// estimateTokens approximates token count using the ~4 characters-per-token rule.
func (ht *HistoryTrimmer) estimateTokens(messages []schema.Message) int {
	total := 0
	for _, msg := range messages {
		for _, part := range msg.GetContent() {
			if tp, ok := part.(schema.TextPart); ok {
				total += len(tp.Text) / 4
			}
		}
		total += 5 // Per-message overhead.
	}
	return total
}

func main() {
	ctx := context.Background()

	// Pass nil summarizer to use trimming without summarization.
	trimmer := NewHistoryTrimmer(200, 5, nil)

	messages := []schema.Message{
		schema.NewSystemMessage("You are a helpful assistant."),
		schema.NewHumanMessage("Message 1: tell me about Go."),
		schema.NewAIMessage("Go is a statically typed, compiled language."),
		schema.NewHumanMessage("Message 2: what about Python?"),
		schema.NewAIMessage("Python is dynamically typed and interpreted."),
		schema.NewHumanMessage("Current question: compare them."),
	}

	trimmed, err := trimmer.TrimHistory(ctx, messages)
	if err != nil {
		slog.Error("trim failed", "error", err)
		return
	}
	fmt.Printf("Trimmed from %d to %d messages\n", len(messages), len(trimmed))
}
```

## Explanation

1. **`GetRole() schema.Role`** — Used to classify messages as system, human, or AI rather than `GetType()` (which does not exist on `schema.Message`). Role comparison uses the typed constants `schema.RoleSystem`, `schema.RoleHuman`, `schema.RoleAI`.

2. **`msg.Text()` via type switch** — `schema.Message.GetContent()` returns `[]schema.ContentPart`, not a string. To extract text for summarization, type-assert the message to the concrete type and call the `.Text()` method (`*schema.HumanMessage`, `*schema.AIMessage`, `*schema.SystemMessage` all implement it). Alternatively, iterate `GetContent()` and accumulate `schema.TextPart` values directly.

3. **`response.Text()`** — `ht.summarizer.Generate()` returns `*schema.AIMessage`. Call `.Text()` on it to get the concatenated text content.

4. **Three-tier trimming** — System messages are always preserved. The `keepRecent` most recent non-system messages are always kept. Old messages are summarized when possible, dropped when not.

## Testing

```go
func TestHistoryTrimmer_TrimsWhenNeeded(t *testing.T) {
	trimmer := NewHistoryTrimmer(50, 2, nil)

	messages := []schema.Message{
		schema.NewSystemMessage("System instructions."),
	}
	for i := 0; i < 10; i++ {
		messages = append(messages, schema.NewHumanMessage(fmt.Sprintf("Message %d with enough text to consume tokens", i)))
	}

	trimmed, err := trimmer.TrimHistory(context.Background(), messages)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(trimmed) >= len(messages) {
		t.Errorf("expected trimming to reduce message count, got %d -> %d", len(messages), len(trimmed))
	}
	// System message must be preserved.
	if trimmed[0].GetRole() != schema.RoleSystem {
		t.Error("system message was not preserved")
	}
}
```

## Variations

### Token-aware Trimming

Replace the character-ratio estimator with actual tokenizer counts from `chunk.Usage` returned during streaming:

```go
func (ht *HistoryTrimmer) TrimWithUsage(ctx context.Context, messages []schema.Message, usage schema.Usage) ([]schema.Message, error) {
	if usage.TotalTokens <= ht.maxTokens {
		return messages, nil
	}
	return ht.TrimHistory(ctx, messages)
}
```

## Related Recipes

- **[Streaming Chunks with Metadata](./streaming-metadata)** — Capture `Usage` from streaming responses
- **[Token Counting without Latency](./token-counting)** — Estimate token counts without API round-trips
