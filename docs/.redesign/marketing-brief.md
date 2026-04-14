# Marketing Brief — Beluga AI v2 Site Redesign

**Scope:** Six surfaces — `/`, `/product`, `/compare`, `/enterprise`, `/community`, `/providers`
**Date:** 2026-04-12
**Author:** marketeer agent
**Status:** draft for coordinator review

---

## 1. Positioning Statement

Beluga AI is the Go-native agentic framework for backend teams shipping production agents: complete from LLM abstraction to voice pipelines, with OTel, circuit breakers, and crash-durable execution built in — not bolted on.

---

## 2. Proof Pillars

### Pillar 1 — Streaming-first via `iter.Seq2`

**Claim:** The only Go agent framework whose public streaming API is a typed `iter.Seq2[T, error]` — no channels, no callbacks, no goroutine leaks.

**Evidence:**
- `core/stream.go:49-56` defines `Stream[T]` backed by `iter.Seq2` (`docs/architecture/01-overview.md` §"Streaming first").
- `.wiki/patterns/streaming.md` documents the canonical producer and explicitly names channels as an anti-pattern.
- `docs/architecture/03-extensibility-patterns.md` states `Invoke()` is "a convenience wrapper around `Stream()` + collect" — the stream path is the real contract.

**Code that proves it:**

```go
import (
    "context"
    "fmt"
    _ "github.com/lookatitude/beluga-ai/llm/providers/openai"
    "github.com/lookatitude/beluga-ai/llm"
)

func main() {
    model, err := llm.New("openai", llm.Config{Model: "gpt-4o"})
    if err != nil {
        panic(err)
    }
    stream, err := model.Stream(context.Background(), []schema.Message{
        {Role: schema.RoleUser, Content: "explain iter.Seq2 in one sentence"},
    })
    if err != nil {
        panic(err)
    }
    for _, chunk := range stream.Range {
        fmt.Print(chunk.Content)
    }
}
```

---

### Pillar 2 — 8 reasoning strategies, one-line swap

**Claim:** Beluga ships eight planner strategies — ReAct to Mixture-of-Agents — that share one interface. Switching strategies is a one-line change.

**Evidence:**
- `docs/architecture/06-reasoning-strategies.md` lists all eight with LLM-calls-per-turn and best-use guidance.
- `docs/architecture/05-agent-anatomy.md` shows the `Planner` interface is ≤4 methods, following the universal package shape (`docs/architecture/03-extensibility-patterns.md` §Ring 1).
- `docs/architecture/06-reasoning-strategies.md`: "upgrading is a one-line change because every planner implements the same `Planner` interface."

**Code that proves it:**

```go
// swap strategy without touching any other agent config
agent, err := agent.New(ctx,
    agent.WithLLM(model),
    agent.WithPlanner("lats"),   // was "react" — one line
    agent.WithTools(tools...),
)
```

---

### Pillar 3 — OTel GenAI spans at every boundary

**Claim:** Every extensible package ships `WithTracing()` middleware that emits `gen_ai.*` spans. No custom instrumentation required.

**Evidence:**
- `docs/architecture/14-observability.md` §Overview: "Every package boundary opens a span. Every span attribute uses the `gen_ai.*` namespace."
- `docs/architecture/03-extensibility-patterns.md` §"`WithTracing()` — the canonical framework-wide middleware": lists 17 packages that follow the template.
- `docs/architecture/14-observability.md` span hierarchy diagram shows `agent.invoke` → `planner.plan`, `llm.generate`, `tool.execute`, `memory.load`, `guard.pipeline` as nested child spans.

**Code that proves it:**

```go
// nothing extra required — tracing is middleware on the registered instance
model, _ := llm.New("anthropic", llm.Config{})
instrumented := llm.ApplyMiddleware(model, llm.WithTracing())
// gen_ai.system, gen_ai.request.model, gen_ai.usage.input_tokens
// appear in your Jaeger/Grafana/Honeycomb dashboard automatically
```

---

### Pillar 4 — Crash-durable execution, no checkpoint boilerplate

**Claim:** The `workflow/` package provides durable execution via event log replay. Agents survive process restarts without application-level checkpointing. Temporal is a drop-in provider backend.

**Evidence:**
- `docs/architecture/16-durable-workflows.md` §Overview: "Durable workflows persist their state so a crash, redeploy, or machine failure doesn't lose progress."
- `docs/reference/providers.md` §"Workflow engines": 6 providers including `temporal`, `inngest`, `dapr`, `nats`, `kafka`, `inmemory`.
- `docs/architecture/01-overview.md` §"Production-ready by default": "The `workflow/` package provides crash-durable execution via a Temporal-compatible backend so agent runs survive process restarts without application-level checkpointing."

**Code that proves it:**

```go
import (
    "github.com/lookatitude/beluga-ai/workflow"
    _ "github.com/lookatitude/beluga-ai/workflow/providers/temporal"
)

store, _ := workflow.New("temporal", workflow.Config{Endpoint: "temporal:7233"})
wf, _ := store.Resume(ctx, runID) // picks up from last durable checkpoint
```

---

### Pillar 5 — 110 providers, 19 categories, one import

**Claim:** 110 providers across LLM, embedding, vector stores, memory, voice, guard, workflow, and more. Every provider registers in `init()` — one import makes it available.

**Evidence:**
- `docs/reference/providers.md` footer: "Total providers: 110 across 19 categories." (as of 2026-04-12 scan).
- `docs/architecture/03-extensibility-patterns.md` §Ring 2 — Registry: `init()`-based registration, blank import activates the provider.
- `docs/architecture/01-overview.md` §"Minimal core, rich ecosystem": "`core/` has zero external dependencies...Providers live in `*/providers/` subdirectories and pull their own SDKs."

**Code that proves it:**

```go
import (
    _ "github.com/lookatitude/beluga-ai/llm/providers/anthropic"
    _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/pgvector"
    _ "github.com/lookatitude/beluga-ai/voice/stt/providers/deepgram"
    _ "github.com/lookatitude/beluga-ai/workflow/providers/temporal"
)
// all four now available via llm.New / vectorstore.New / stt.New / workflow.New
```

---

## 3. Homepage Block Spec

### Block H1 — Hero

**Purpose:** Establish the product claim and prove it with code in the first viewport.

**Copy:**
- Headline: `The Go-native agent framework for production.`
- Subhead: `Not a Python port. Not a prototype scaffold. Beluga is a complete 7-layer framework — streaming via iter.Seq2, OTel at every boundary, crash-durable execution, 110 providers.`
- Body: omit — code does the work.

**Visual/layout note:** Two-column. Left: headline + subhead + primary CTA. Right: a short, runnable Go snippet (≤25 lines) showing an agent streaming a response — complete imports, real error handling, real package path. The code block is visible without scrolling on a 1280px viewport.

**CTA:** `Read the quickstart` → `/docs/guides/first-agent` (primary); `Browse the architecture` → `/docs/architecture/01-overview` (secondary, text link below button).

---

### Block H2 — The 7-layer stack

**Purpose:** Give the mental model before any feature list. Readers should understand the shape of the framework in 30 seconds.

**Copy:**
- Headline: `Seven layers. One import path.`
- Subhead: `From Foundation primitives to Application code, each layer depends only on the layers below it. No hidden global state. No runtime reflection.`

**Visual/layout note:** The SVG at `docs/beluga_full_layered_architecture.svg`. Full-width or near-full-width, center-aligned. Below it, one sentence linking to `docs/architecture/01-overview.md`: "Read the architecture" text link.

**CTA:** None — this block is informational. The architecture link is the secondary action.

---

### Block H3 — Proof pillars

**Purpose:** Evidence-anchored claims. Not a feature grid — three to five sharp statements, each with a code fragment or specific number.

**Copy:**
- Headline: `Built for the production side of Go.`
- Subhead: omit — the pillars speak.
- Pillars (two-column or staggered list, not icon cards):
  1. `iter.Seq2, not channels.` — "The streaming primitive is a typed range-over-func iterator. Backpressure is free. Goroutine leaks are impossible."
  2. `8 reasoning strategies, one interface.` — "ReAct to LATS to Mixture-of-Agents. Swap with one line. Build your own in the same slot."
  3. `OTel GenAI spans everywhere.` — "17 packages emit gen_ai.* spans at every boundary. Works with Jaeger, Grafana Tempo, Honeycomb out of the box."
  4. `Crash-durable without checkpointing.` — "workflow/ replays from an event log. Temporal, Inngest, NATS, Dapr — or in-process for tests."

**Visual/layout note:** Each pillar is a short text block with a one- or two-line code fragment inline (not a full code block). Asymmetric layout — pillars stagger left/right, not a uniform grid.

**CTA:** None per-pillar. Section ends with: `See what's under the hood` → `/product`.

---

### Block H4 — Provider marquee

**Purpose:** Signal breadth without claiming irrelevant numbers.

**Copy:**
- Headline: `110 providers. 19 categories.`
- Subhead: `LLM, embedding, vector stores, memory, voice, guard, workflow, observability. Every provider registers in init() — one import, fully wired.`

**Visual/layout note:** Scrolling horizontal marquee of provider logos/names, two rows. Not a grid. Sourced from `docs/reference/providers.md`. Link to `/providers` for the filterable catalog.

**CTA:** `Browse all providers` → `/providers`.

---

### Block H5 — Final CTA

**Purpose:** Single exit point for readers who have scrolled the page.

**Copy:**
- Headline: `Start with the five-minute quickstart.`
- Body: `Build a streaming agent, wire a tool, read the architecture. No account required.`

**Visual/layout note:** Full-width editorial block. Left-aligned. Two buttons side by side.

**CTA:** `Read the quickstart` → `/docs/guides/first-agent` (primary); `Star on GitHub` → `https://github.com/lookatitude/beluga-ai` (secondary).

---

## 4. /product Page Block Spec

### Block P1 — Intro

**Purpose:** Orient the reader. This page is the consolidated replacement for 11 features pages.

**Copy:**
- Headline: `What Beluga can do.`
- Subhead: `A complete agent stack organized around three concerns: Build — where agents come from; Know — what agents remember and retrieve; Ship — how agents behave in production.`

No grid of 11 icons. The three groups below are the navigation.

---

### Block P2 — Build (agent runtime, LLM, tools, prompts)

**Purpose:** Cover the agent core — planning, execution, LLM abstraction, tool use, prompt management.

**Copy:**
- Section label: `BUILD`
- Headline: `Agents that reason, act, and recover.`
- Body: `The agent runtime runs a Plan → Act → Observe → Replan loop on every turn. Eight planning strategies from ReAct to Mixture-of-Agents share one interface — swap them with a config change. LLM calls route across 22 providers with a unified ChatModel interface. Tools are any Go function wrapped with a schema. Handoffs between agents are auto-generated tools.`

**Visual/layout note:** One code sample (≤30 lines): a minimal agent wiring an LLM, two tools, and a planner, then streaming a response. Reference: `docs/guides/first-agent.md`, `docs/architecture/05-agent-anatomy.md`, `docs/architecture/06-reasoning-strategies.md`.

**CTA:** `Agent anatomy docs` → `/docs/architecture/05-agent-anatomy`.

---

### Block P3 — Know (memory, RAG)

**Purpose:** Cover the knowledge layer — what agents remember between turns and what they retrieve from documents.

**Copy:**
- Section label: `KNOW`
- Headline: `Memory that persists. Retrieval that finds the right thing.`
- Body: `Three-tier memory (working / recall / archival) with graph store support. RAG pipeline with hybrid retrieval: BM25, dense vector, graph traversal — fused with Reciprocal Rank Fusion. Strategies include CRAG, Adaptive RAG, HyDE, and GraphRAG. 13 vector store backends, 9 embedding providers.`

**Visual/layout note:** One code sample showing a hybrid retrieval setup with a vector store and BM25 reranker. Reference: `docs/architecture/09-memory-architecture.md`, `docs/architecture/10-rag-pipeline.md`.

**CTA:** `RAG pipeline docs` → `/docs/architecture/10-rag-pipeline`.

---

### Block P4 — Ship (guardrails, observability, resilience, durability, protocols)

**Purpose:** The production story — what makes this safe, observable, resilient, durable, and interoperable.

**Copy:**
- Section label: `SHIP`
- Headline: `Production defaults, not production afterthoughts.`
- Body: `Guard pipeline runs three stages — Input, Output, Tool — before and after every LLM interaction. Circuit breakers, rate limits, and retry are middleware on the same interface as your LLM call. OTel GenAI spans emit from 17 packages at every boundary. Durable workflows replay from an event log — agents survive restarts. MCP and A2A are first-class protocols; REST, gRPC, and WebSocket ship the same runner.`

**Visual/layout note:** One code sample showing `ApplyMiddleware` stacking `WithGuardrails`, `WithTracing`, `WithRateLimit`, `WithRetry` on a model — four lines, real API. Reference: `docs/architecture/03-extensibility-patterns.md` §middleware, `docs/architecture/14-observability.md`, `docs/architecture/15-resilience.md`, `docs/architecture/16-durable-workflows.md`.

**CTA:** `Production checklist` → `/docs/production-checklist`.

---

### Block P5 — Voice (standalone, not folded into Ship)

**Purpose:** Voice is a non-obvious capability for a Go framework. It earns its own paragraph.

**Copy:**
- Section label: `VOICE`
- Headline: `Frame-based voice, built in.`
- Body: `STT to LLM to TTS as a typed pipeline. 6 STT providers, 7 TTS providers, 3 speech-to-speech providers. LiveKit, Daily, and Pipecat transports. VAD with Silero and WebRTC. No Go competitor includes this.`

**Visual/layout note:** The voice pipeline diagram from `docs/architecture/01-overview.md` §"Voice agent" data flow (Mermaid graph). Reference: `docs/architecture/11-voice-pipeline.md`.

**CTA:** `Voice pipeline docs` → `/docs/architecture/11-voice-pipeline`.

---

## 5. /compare Page Spec

### Competitors to include

1. **LangChainGo** — the direct Go peer; Go developers will land here first as a reference point. `.wiki/competitors/langchaingo.md`.
2. **Google ADK-Go** — Go SDK, enterprise pedigree; will appear in the same evaluation shortlists. `.wiki/competitors/adk-go.md`.
3. **Eino (ByteDance/cloudwego)** — active Go framework with growing traction. `.wiki/competitors/eino.md`.
4. **LangChain/LangGraph (Python)** — the most common prior art for Go engineers who have tried Python first; relevant for framing the language-choice argument.

Exclude Mastra (Node.js), CrewAI (Python), Pydantic AI (Python) — different language target, different evaluation context. A Go engineer choosing between Beluga and Mastra has already decided on Go.

### Comparison dimensions

| Dimension | Source |
|---|---|
| Primary language | Observable |
| Streaming primitive | `.wiki/competitors/langchaingo.md`; `docs/architecture/01-overview.md` |
| Reasoning strategies (count) | `docs/architecture/06-reasoning-strategies.md` |
| Built-in OTel GenAI spans | `docs/architecture/14-observability.md` |
| Durable workflow (built-in) | `docs/architecture/16-durable-workflows.md`; `.wiki/competitors/adk-go.md` |
| Voice pipeline (built-in) | `docs/architecture/11-voice-pipeline.md`; `.wiki/competitors/*.md` |
| Provider integrations (count) | `docs/reference/providers.md` |
| License | Observable |

Eight rows. Every cell is verifiable, not a subjective score. Beluga cells cite the doc; competitor cells cite the competitor's public documentation or a stub note where the wiki entry is unpopulated.

### Tone guidance

State facts. Where a competitor lacks a capability, write "not included" — not a dash, not a cross, not a frown emoji. Where a competitor has a genuine strength (LangChain Python's 50+ LLM providers, LangGraph's checkpointing maturity), acknowledge it. The table footnotes are the place for "as of April 2026 — verify before deciding." Competitor wiki entries are stubs; flag claims that need external verification before the page ships.

### Closing narrative

Beluga is the right choice when the team ships Go, needs the full agent stack in one consistent framework, and cannot afford to debug Python interop in production. It is not the right choice when: the team is Python-native, the project requires a mature third-party ecosystem of LangChain plugins, or the primary constraint is time-to-prototype rather than production operability. An honest comparison page converts the readers who would have churned anyway, and keeps the ones who fit.

---

## 6. /enterprise Page Spec

### Value proposition

Enterprise teams get the same OSS framework plus documented operational patterns for the concerns that are not optional at scale: access control, audit trails, cost accountability, safety compliance, and durable execution across restarts and deployments.

### Capabilities mapped to architecture

| Capability | Package | Doc reference |
|---|---|---|
| RBAC / ABAC access control | `auth/` | `docs/architecture/18-package-dependency-map.md` §Layer 2 |
| Full audit trail (every agent action logged) | `audit/` | `docs/architecture/18-package-dependency-map.md` §Layer 2 |
| Cost tracking and enforcement | `cost/` | `docs/architecture/18-package-dependency-map.md` §Layer 2 |
| Safety pipeline (Input → Output → Tool) | `guard/` + 5 providers | `docs/architecture/01-overview.md` §Layer 3; `.wiki/patterns/security-guards.md` |
| Crash-durable workflows | `workflow/` + Temporal provider | `docs/architecture/16-durable-workflows.md` |
| OTel GenAI observability | `o11y/` + 4 exporters | `docs/architecture/14-observability.md` |

Six capabilities. Each maps to a real package and a doc. No capability claims a package that does not exist.

### Contact / CTA

We do not have a sales team. The CTA routes to an enterprise inquiry GitHub issue template: `github.com/lookatitude/beluga-ai/issues/new?template=enterprise-inquiry.md`. The reply SLA and what to expect in the reply (architecture call, deployment review, roadmap alignment) are stated directly on the page — no vague "we'll be in touch."

### Customer-type scenario

A platform team at a fintech consolidating agent tooling across 40 services needs a framework where every agent emits the same OTel spans, every LLM call is cost-attributed to a team, every tool call passes through the same guard pipeline, and no agent run silently drops state on a pod restart. They need audit logs that satisfy compliance review. They evaluate Beluga because it ships all six of those concerns as first-class packages — not because a vendor pitch deck promises them.

---

## 7. /community Page Spec

Single page, not a kitchen sink. Fold in `about.astro` content.

### Sections

**Mission paragraph (from `about.astro`):** Go teams building AI products face a real choice: Python interop, fragile bindings, or incomplete libraries. Beluga is the answer built entirely in Go — testable, deployable, reasoned about with standard Go tooling. MIT licensed.

**Philosophy:** Three sentences on why Go for agents: deterministic concurrency model, single-binary deployment, compile-time safety. No manifestos.

**Channels:**
- GitHub: source, issues, discussions, releases. `github.com/lookatitude/beluga-ai`.
- Discord: async questions, show-and-tell, release announcements. Link TBD at launch.
- Blog: `/blog` placeholder — link activates when first post ships.

**Contributor ladder:** one-paragraph reference to `docs/CONTRIBUTING.md` (or equivalent). Good first issues labeled, architecture doc is the onboarding path.

**Roadmap:** the four phases from `community.astro` (all marked complete as of the current codebase). Add a "What's next" line pointing to GitHub Discussions for roadmap input — not a promise grid.

**Maintainers:** list names and GitHub handles when ready. Stub with "core team" until populated.

---

## 8. Navigation Map

Top nav item order, left to right:

| Item | Destination | Active state |
|---|---|---|
| Product | `/product` | Active on `/product` |
| Docs | `/docs` | Active on `/docs/*` |
| Providers | `/providers` | Active on `/providers` |
| Recipes | `/docs/guides` | Active on `/docs/guides/*` |
| Enterprise | `/enterprise` | Active on `/enterprise` |
| Community | `/community` | Active on `/community` |

Right side of nav: GitHub icon link → `https://github.com/lookatitude/beluga-ai`; theme toggle.

Active-state rule: a nav item is active when the current URL matches its destination prefix. `/docs/guides/first-agent` activates both `Docs` and `Recipes` — resolve the tie by activating the more specific match (`Recipes`). The root `/` has no active nav item; the logo is the active indicator.

---

## 9. Copy Guardrails

Words and constructions banned from all marketing copy on this site:

**Banned words:**
- unlock, unleash, empower, revolutionize, transform, supercharge
- seamless, effortless, intuitive, frictionless
- next-gen, next-generation, cutting-edge, state-of-the-art, best-in-class
- AI-powered (redundant — it's an AI framework)
- scalable (unless a specific number follows it)
- robust (weak filler — say what the robustness property is)
- simple (show the code; let readers decide)
- world-class
- game-changing, groundbreaking, paradigm-shifting

**Banned constructions:**
- Hype stats without a `file:line` or doc reference.
- Marketing ellipses: "And so much more..."
- Rhetorical questions in headlines: "What if your agents could...?"
- Multi-em-dash sentences: "Beluga — the framework that — ships production agents."
- Passive voice for capability claims: "Agents are made reliable by..." → "Beluga makes agents reliable by..."
- "Easy to use" without a code example proving it.
- Numbered lists disguised as claims: "3 reasons why Beluga is the best..." — cite evidence instead.
- Claims about competitors not backed by `.wiki/competitors/` or public documentation.
- Sentence fragments as section intros: "Built for production." as a standalone paragraph.
- Any use of "modern" as a standalone adjective ("modern Go framework") — it means nothing.

**Verification rule:** Before any claim ships in copy, it must have a `docs/` or source file citation in the page's implementation comments. The compare page is especially high-risk — all competitor cells must be re-verified against current public documentation before launch.
