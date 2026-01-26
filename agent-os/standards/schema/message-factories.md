# Message Factories

**Pattern:** `NewHumanMessage(content)`, `NewAIMessage(content)`, `NewSystemMessage(content)`, `NewToolMessage(content, toolCallID)` — return `Message`. Multimodal: `NewImageMessage(url, text)`, `NewImageMessageWithData(data, format, text)`, `NewVideoMessage(...)`, `NewVoiceDocument(...)`.

**Return type:** Prefer `Message`. A factory may return a concrete type (e.g. `*ImageMessage`) when callers need type-specific APIs.

**OTEL:** A single factory per message type is enough. Tracing and metrics are handled where the message is used. OTEL is the default observability standard.
