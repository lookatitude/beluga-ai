---
derived_from_template: guild.skill_template.v1
name: streaming-patterns
description: "Beluga AI v2 streaming with iter.Seq2[T, error] (Go 1.26+): producer shape, honouring yield=false and ctx cancellation, Invoke = stream + collect, and the real composition helpers in core (core.Stream, CollectStream, MapStream, FilterStream, MergeStreams, FanOut, NewBufferedStream; core.Pipe/Parallel for Runnables). Pulled by architect and developer-go. TRIGGER: \"implement Stream for X\", \"convert this channel API to iter.Seq2\", \"fan out the stream\", \"merge two streams\", \"buffer the stream\", \"build Invoke from Stream\", \"backpressure in the stream\". DO NOT TRIGGER for: interface method design (go-interfaces); registry and package layout (go-framework); provider HTTP/SSE error mapping (provider-implementation); stream test code (go-testing); OTel span naming (consult observability-expert)."
when_to_use: An architect or developer-go lane is designing or writing a public streaming API, or reviewing one that might leak channels into the public surface.
type: specialist
---

# streaming-patterns

Canonical body (producer template, helper list, rules): `.claude/skills/streaming-patterns/SKILL.md`. Read it first; this file is the Guild routing wrapper.

# When to use it

- Any public streaming API (`AGENTS.md` critical rule 1: `iter.Seq2[T, error]`, never channels).
- Composing streams with the `core` helpers (`core/stream.go`) or Runnables (`core/runnable.go`).
- Deriving `Invoke()` from `Stream()`.

# When not to use it

- Interface shape → `go-interfaces`. Provider wire protocol → `provider-implementation`. Tests → `go-testing`.

# Required inputs

The producer's source (client stream, generator), cancellation semantics, whether pull semantics are truly needed.

# Output format

An `iter.Seq2` producer or a review finding citing `file:line`.

# Workflow steps

1. Read `.claude/skills/streaming-patterns/SKILL.md` and `.guild/wiki/patterns/streaming.md`.
2. Check `ctx.Done()` in the producer loop; return immediately when `yield` returns false.
3. Use only helpers that exist in `core/stream.go` / `core/runnable.go` — there is no generic `Pipe`/`Collect` over plain `iter.Seq2`.
4. Channels are fine internally; never in the exported signature.

# Evidence requirements

A cancellation test (see `go-testing`) and `go test -race` output.

# Escalation rules

A need for pull semantics or a new core helper → `architect`.

# Safety constraints

No goroutine without a cancellation path; no unbounded buffering of external input.

# Eval cases

See `evals.json` next to this file.
