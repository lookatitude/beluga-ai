# StreamingAgent

**Interface:** `StreamingAgent` embeds `Agent` and adds `StreamExecute(ctx, inputs) (<-chan AgentStreamChunk, error)` and `StreamPlan(ctx, intermediateSteps, inputs) (<-chan AgentStreamChunk, error)`.

**Chunk:** `AgentStreamChunk` — `Err`, `Action`, `Finish`, `Metadata`, `Content`, `ToolCalls`. Stream ends when `Finish` or `Err` is set; channel is closed.

**Contract:** Start immediately (no blocking before first chunk); send chunks as they arrive; close on completion or error; include tool calls when present; respect `ctx` cancellation. **First-chunk latency:** treat ~200ms as a **performance goal** — do not enforce in tests; **emit warnings** (e.g. logging, metrics) when first chunk is later than that.

- **Optional:** Not all agents implement `StreamingAgent`. Depends on backend/LLM streaming support. Use type assertion or a separate `StreamingAgent`-capable factory when needed.
- `StreamingConfig`: `ChunkBufferSize`, `MaxStreamDuration`, `EnableStreaming`, `SentenceBoundary`, `InterruptOnNewInput`. Use `WithStreaming` / `WithStreamingConfig`.
