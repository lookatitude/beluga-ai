# Architecture Invariants

The 10 design decisions that define Beluga AI v2. Never violate these without a documented ADR in `decisions.md`.

`/wiki-learn` enriches each invariant below with a real `file:line` reference to its canonical implementation. Until then, the WHY is the contract.

## 1. Streaming uses `iter.Seq2[T, error]` — never channels

**Why:** Channels in public APIs force consumers into select statements, leak goroutines on early return, and block composition. `iter.Seq2` is pull-based, composable, integrates with `context.Context`, and idiomatic in Go 1.23+.

**Canonical example:** (populate via /wiki-learn — look for `iter.Seq2` in `core/stream.go`)

## 2. Handoffs are tools — auto-generate `transfer_to_{name}`

**Why:** Treating handoffs as normal tools unifies the agent's decision surface. LLMs already reason about tools; a separate handoff mechanism would require separate prompt engineering.

## 3. MemGPT 3-tier memory: Core / Recall / Archival + graph

**Why:** Single-tier memory either runs out of context (all in-prompt) or loses relevance (all external). The 3-tier model keeps working set hot while maintaining long-term recall.

## 4. Guard pipeline is 3-stage: Input → Output → Tool

**Why:** Any stage alone is insufficient. Input guards can miss model-generated issues; output guards can't protect tool calls. All three are required for defense in depth.

## 5. Own durable execution engine; Temporal is a provider option

**Why:** Forcing Temporal on every user imports significant infrastructure. Owning the engine lets us provide a simple default with Temporal as an opt-in backend.

## 6. Frame-based voice: `FrameProcessor` interface

**Why:** Voice pipelines need streaming transformation (STT → LLM → TTS) with backpressure and interruption. Frames are the minimum useful unit for this.

## 7. Registry pattern everywhere

**Why:** Extensibility without recompilation. `Register()` + `New()` + `List()` is the contract for every pluggable package (LLM providers, vector stores, tools, etc.).

## 8. OTel GenAI conventions — `gen_ai.*` namespace

**Why:** Aligning with OpenTelemetry GenAI semantic conventions means observability tools work out of the box and the code stays portable across backends.

## 9. Hybrid search default — Vector + BM25 + RRF fusion

**Why:** Pure vector search misses exact matches; pure BM25 misses semantic similarity. Reciprocal Rank Fusion combines both without tuning weights.

## 10. Prompt cache optimization — static content first

**Why:** Providers like Anthropic cache prefixes. Putting static (system, tools) content first and dynamic (user, history) content last maximizes cache hit rate and minimizes cost.

---

## Hard rules (derived from invariants)

- Interfaces have 1–4 methods. Compose larger surfaces.
- `context.Context` is the first parameter of every public function.
- Providers auto-register in `init()`.
- `core/` and `schema/` have zero external deps beyond stdlib + otel.
- No circular imports. Dependencies flow downward.
- Errors: `(T, error)`, typed via `core/errors.go` with `ErrorCode`.
- No `interface{}` in public APIs — use generics.
- No global mutable state outside registries.
- Goroutines are cancellable via context and bounded.
- Table-driven tests; `*_test.go` alongside source.
