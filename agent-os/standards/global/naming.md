# Naming

**Package:** lowercase, singular. `llms`, `embeddings`, `orchestration` — not `LLMs`, `embeddings pkg`.

**Abbreviations:** Only these are allowed when widely used: `llms`, `stt`, `tts`, `vad`, `rag`. Prefer full words otherwise.

**Interfaces folder:** Always `iface/` (not `interfaces/`). Put public interfaces and shared types in `iface/`. Add `iface/errors.go` when those interfaces use custom error types.

**Main API file:** `{package_name}.go` (e.g. `llms.go`, `embeddings.go`).
