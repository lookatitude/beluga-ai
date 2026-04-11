# Architecture Decision Records

ADR log for Beluga AI v2. Each decision is immutable — supersede with a new entry rather than editing old ones.

## Format

```
### ADR-NNN | YYYY-MM-DD | <title>
**Status:** Accepted / Superseded by ADR-XXX / Deprecated
**Context:** what problem needs solving
**Decision:** what we decided
**Rationale:** why this over alternatives
**Consequences:** positive + negative impacts
**Alternatives considered:** brief
```

---

### ADR-001 | 2026-04-11 | Adopt unified self-evolving multi-agent system

**Status:** Accepted
**Context:** The v1 agent setup had monolithic CLAUDE.md, single-file learnings, and no enforcement layer. A parallel `.claude/teams/` system existed with better learning infrastructure but was bound to a one-shot migration workflow.
**Decision:** Fuse both systems into a 5-layer design: deterministic hooks (L1), two-tier knowledge (L2), 10 lean agents (L3), 15 standalone-composable workflow commands (L4), and an evolution pipeline (L5).
**Rationale:** Keeps the proven learning hooks from teams/ (cross-pollinating per-agent rules) while adopting the plan's architectural boundaries (file-scoped auto-loaded rules, retrieval-on-demand wiki, <2500-token CLAUDE.md).
**Consequences:**
- Positive: learning is both fast (automatic per-agent) and curated (wiki-promoted). Every workflow is independently triggerable. Enforcement is deterministic.
- Negative: two learning stores means some duplication; coordinator must periodically promote per-agent findings to wiki.
**Alternatives considered:** Full replacement (lost the existing hook infrastructure); additive migration (left duplication without integration).

---

### ADR-002 | 2026-04-11 | `WithTracing` canonicalized as the universal Ring 4 middleware

**Status:** Accepted
**Context:** Before the arch-validate sweep, OTel instrumentation was inconsistent across the framework. `memory/` had none at all. Some packages had custom span builders with ad-hoc attribute keys (`llm.tokens.input`, `tool.name`), which were invisible to GenAI-aware backends and drifted per package. Invariant #8 required GenAI spans at every package boundary but did not specify *how*.
**Decision:** The `memory/tracing.go` template is now the canonical pattern for OTel instrumentation across every extensible package. Each package must expose a `WithTracing() Middleware` that wraps its core interface, opens a span named `<pkg>.<method>` per public method via `o11y.StartSpan`, attaches attributes exclusively through the typed `o11y.Attr*` constants, records errors with `span.RecordError`, and sets `StatusError` on failure. Opt-in is a one-line composition: `pkg.ApplyMiddleware(base, pkg.WithTracing())`.
**Rationale:** After the pattern proved itself across 17 packages in a single sweep, enshrining it as the standard eliminates per-package invention. Middleware is the right Ring (not hooks) because tracing applies uniformly to every call on the interface; a `func(T) T` wrapper covers the entire interface with ~40 lines of code. The `o11y.Attr*` constants centralise the GenAI v1.37 semconv so any future semconv bump is a single-file edit.
**Consequences:**
- Positive: Uniform trace shape across 17 capability packages; any GenAI-aware backend (Datadog, Honeycomb, Grafana, Tempo) renders dashboards without config; new packages have a copy-paste template.
- Positive: `WithTracing()` composes cleanly with `WithRetry()`, rate-limit, and logging middleware via `ApplyMiddleware`.
- Negative: Every new extensible package must ship `WithTracing()` plus its paired test using `tracetest.InMemoryExporter` — slight ceremony for contributors.
- Enforcement: `/arch-validate` flags extensible packages missing a `WithTracing()` export. New packages without it fail review.
**Alternatives considered:** Single top-level `otel.Wrap(any)` reflector (rejected — `interface{}` in public API, no generics, breaks invariant #8); per-package hook wiring (rejected — hooks observe specific points, tracing needs every point); central `o11y.Trace[T]()` generic (rejected — still requires per-package method walking, no simpler than the middleware).

---

### ADR-003 | 2026-04-11 | `iter.Seq2[T, error]` required in public streaming APIs (enforcement complete)

**Status:** Accepted
**Context:** Invariant #6 stated "public streaming APIs use `iter.Seq2[T, error]` — never channels" but five user-facing interfaces still returned `<-chan T`: `memory/shared.SharedMemory.Watch`, `workflow.WorkflowContext.ReceiveSignal`, `state.Store.Watch`, `voice/s2s.SessionReceiver.Recv`, and the coupled pair `voice.FrameProcessor.Process` / `voice.Transport.Recv`. Channel-based public APIs leak goroutines when callers forget cleanup, allow close-after-send races, and cannot express typed errors without a parallel error channel.
**Decision:** All five cascades were converted to `iter.Seq2[T, error]` (commits `e97d0771`, `19741097`, `f9c06d30`). Public streaming APIs now exclusively use `iter.Seq2`. Internal channels remain legitimate for producer/consumer buffering inside struct fields — the invariant applies only to the *exported surface*.
**Rationale:** `iter.Seq2` expresses values + errors in one iterator, matches Go 1.23+ idioms, makes cleanup follow lexical scope (`defer stop()` or range-break), and prevents the `<-chan T`-plus-separate-error-chan anti-pattern. Enforcement is now empirical: `grep` + `/arch-validate` confirm zero `<-chan T` exports across user-facing packages.
**Consequences:**
- Positive: Callers consume streams with plain `for v, err := range s { … }`; no manual drain on errors, no leaked watchers, no dual-return `(ch, err)` shapes.
- Positive: Novel streaming constraints discovered during the sweep are now documented for pattern authors — C-004 (Temporal `Context.Done() == nil`), C-005 (fan-in single-writer rule), and the `iter.Pull2`-is-single-goroutine-only rule — see `.wiki/corrections.md`.
- Negative: `iter.Seq2` composition with bidirectional fan-in is non-trivial; the `voice/s2s.AsFrameProcessor` layout documents the canonical pattern (`outResults` + `inputErr` + `inputDone` + select-nil-channel).
- Enforcement: Reviewer-security and `/arch-validate` reject any new public `<-chan T` return. Future cross-package channel cascades must land in a single coordinated commit, not incrementally.
**Alternatives considered:** Keep channels with a rigorous "always `defer close`" discipline (rejected — not enforceable, lost races); wrap channels in a helper type (rejected — `iter.Seq2` already exists and is the language-native answer); defer the work indefinitely (rejected — the invariant was in the spec but not in the code, so the spec was fiction until now).
