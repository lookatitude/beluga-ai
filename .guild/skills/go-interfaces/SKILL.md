---
derived_from_template: guild.skill_template.v1
name: go-interfaces
description: "Beluga AI v2 extension-contract design: interfaces of 1–4 methods, optional capabilities via type assertion, embedding for composition, nil-safe Hooks structs (Before/After/On naming, ComposeHooks), and func(T) T middleware with ApplyMiddleware ordering. Pulled by architect, developer-go, security and doc-writer. TRIGGER: \"design the interface for X\", \"this interface has too many methods\", \"add hooks to the agent\", \"add a middleware for Y\", \"how should ComposeHooks chain\", \"split this interface\". DO NOT TRIGGER for: package layout, registries and functional options (go-framework); stream producer code (streaming-patterns); a provider for an existing interface (provider-implementation); test code (go-testing); architecture-level layer placement or ADRs (the architect specialist decides, this skill only shapes the contract)."
when_to_use: A framework specialist is defining or reviewing a public extension contract — an interface, its Hooks struct or its Middleware type.
type: specialist
---

# go-interfaces

Canonical body (contract examples and rules): `.claude/skills/go-interfaces/SKILL.md`. Read it first; this file is the Guild routing wrapper.

# When to use it

- Defining or changing a public interface, a `Hooks` struct or a `Middleware` type.
- Reviewing an interface for the ≤4-method rule (`AGENTS.md` critical rule 9).
- Choosing between a new method and an optional capability (`if x, ok := v.(BatchRetriever); ok`).

# When not to use it

- Registry, options, lifecycle → `go-framework`.
- `iter.Seq2` producers → `streaming-patterns`.
- Writing a provider → `provider-implementation`.

# Required inputs

The consumer package (interfaces live with the consumer), the behaviour to expose, existing interfaces in the package.

# Output format

Go interface / hooks / middleware definitions, or review findings with `file:line`, each tied to the rule it applies.

# Workflow steps

1. Read `.claude/skills/go-interfaces/SKILL.md`, `.guild/wiki/patterns/hooks.md` and `middleware.md`.
2. Keep each interface at 1–4 methods; compose larger surfaces by embedding.
3. Hooks: every field optional, nil = skip, errors handled not swallowed.
4. Middleware: `func(T) T`; `ApplyMiddleware(x, a, b)` makes `a` outermost.
5. `context.Context` first on every method; no `interface{}` — use generics.

# Evidence requirements

Method counts stated; compile-time checks present; `go build ./...` passes.

# Escalation rules

Renaming an exported interface needs a `type Old = New` alias (Sonar S8196) and an `architect` sign-off; breaking changes need an ADR.

# Safety constraints

Never widen an interface past four methods "just this once"; never expose internals through hook signatures.

# Eval cases

See `evals.json` next to this file.
