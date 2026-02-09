---
title: Custom Message Types
description: Extend Beluga AI's schema with custom message types for structured enterprise data in conversation flows.
---

Standard message types (`HumanMessage`, `AIMessage`, `SystemMessage`) handle text well, but enterprise applications often need to pass structured data — customer profiles, transaction records, compliance events — through AI pipelines. Custom message types maintain type safety while passing this data alongside standard conversation messages.

## What You Will Build

A `TransactionMessage` type that carries structured financial data through the conversation pipeline, implementing the `schema.Message` interface.

## Prerequisites

- Familiarity with Go interfaces
- Understanding of the [schema package](/guides/schema) message types

## The Message Interface

In Beluga AI v2, every message implements the `schema.Message` interface:

```go
type Message interface {
    GetRole() Role
    GetContent() []ContentPart
    GetMetadata() map[string]any
}
```

Messages carry multimodal content parts (`TextPart`, `ImagePart`, `AudioPart`, etc.) and arbitrary metadata. The role constants are:

| Role | Constant | Purpose |
|:---|:---|:---|
| System | `schema.RoleSystem` | Instructions for the AI model |
| Human | `schema.RoleHuman` | User input |
| AI | `schema.RoleAI` | Model responses |
| Tool | `schema.RoleTool` | Tool execution results |

## Step 1: Define the Custom Message

Create a message type that carries structured transaction data:

```go
package main

import (
    "fmt"
    "time"

    "github.com/lookatitude/beluga-ai/schema"
)

// TransactionData holds structured financial payload.
type TransactionData struct {
    TransactionID string    `json:"transaction_id"`
    Amount        float64   `json:"amount"`
    Currency      string    `json:"currency"`
    Timestamp     time.Time `json:"timestamp"`
    Status        string    `json:"status"`
}

// TransactionMessage carries transaction data in a conversation.
type TransactionMessage struct {
    Data     TransactionData
    metadata map[string]any
}

// Compile-time interface compliance check.
var _ schema.Message = (*TransactionMessage)(nil)

func NewTransactionMessage(data TransactionData) *TransactionMessage {
    return &TransactionMessage{
        Data: data,
        metadata: map[string]any{
            "message_type": "transaction",
        },
    }
}
```

## Step 2: Implement the Message Interface

The `GetContent` method must return `[]schema.ContentPart`. For structured data, create a `TextPart` with a formatted representation that the LLM can read:

```go
func (m *TransactionMessage) GetRole() schema.Role {
    return schema.RoleSystem
}

func (m *TransactionMessage) GetContent() []schema.ContentPart {
    text := fmt.Sprintf(
        "Transaction %s: %s %.2f (%s) at %s",
        m.Data.TransactionID,
        m.Data.Currency,
        m.Data.Amount,
        m.Data.Status,
        m.Data.Timestamp.Format(time.RFC3339),
    )
    return []schema.ContentPart{schema.TextPart{Text: text}}
}

func (m *TransactionMessage) GetMetadata() map[string]any {
    return m.metadata
}
```

## Step 3: Use Custom Messages in a Conversation

Custom messages integrate into standard `[]schema.Message` slices:

```go
func main() {
    txData := TransactionData{
        TransactionID: "TX-998877",
        Amount:        1250.00,
        Currency:      "USD",
        Timestamp:     time.Now(),
        Status:        "Pending",
    }

    // Build conversation with custom message
    messages := []schema.Message{
        schema.NewSystemMessage("You are a financial assistant. Analyze the transaction data provided."),
        NewTransactionMessage(txData),
        schema.NewHumanMessage("Is this transaction suspicious?"),
    }

    // Pass to any ChatModel
    resp, err := model.Generate(ctx, messages)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }
    fmt.Println(resp.Text())
}
```

The LLM sees the `TextPart` content from `GetContent()` for the custom message, alongside the standard system and human messages.

## Step 4: JSON Content for Structured Parsing

For cases where the LLM should receive structured JSON data instead of formatted text, return JSON in the content part:

```go
import "encoding/json"

func (m *TransactionMessage) GetContent() []schema.ContentPart {
    data, err := json.Marshal(m.Data)
    if err != nil {
        return []schema.ContentPart{schema.TextPart{Text: "error: " + err.Error()}}
    }
    return []schema.ContentPart{schema.TextPart{Text: string(data)}}
}
```

## Step 5: Multimodal Custom Messages

Custom messages can carry multiple content parts including images, audio, and files:

```go
type ReportMessage struct {
    Title    string
    Summary  string
    ChartPNG []byte
}

func (m *ReportMessage) GetRole() schema.Role { return schema.RoleSystem }

func (m *ReportMessage) GetContent() []schema.ContentPart {
    parts := []schema.ContentPart{
        schema.TextPart{Text: fmt.Sprintf("Report: %s\n%s", m.Title, m.Summary)},
    }
    if len(m.ChartPNG) > 0 {
        parts = append(parts, schema.ImagePart{
            Data:     m.ChartPNG,
            MimeType: "image/png",
        })
    }
    return parts
}

func (m *ReportMessage) GetMetadata() map[string]any {
    return map[string]any{"message_type": "report"}
}
```

## Verification

1. Confirm your custom message compiles and satisfies the `schema.Message` interface.
2. Verify `GetContent()` returns the expected formatted data.
3. Confirm the custom message integrates into `[]schema.Message` alongside standard types.

## Next Steps

- [Multi-turn Conversations](/tutorials/foundation/multiturn-conversations) — Manage conversation state and history
- [Custom Runnable](/tutorials/foundation/custom-runnable) — Build custom processing steps
