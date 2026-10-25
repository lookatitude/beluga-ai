---
derived_from_template: guild.skill_template.v1
name: go-framework
description: "Beluga AI v2 package-structure conventions: the Interface → Registry (Register/New/List) → Hooks → Middleware layout, init() provider auto-registration, functional options (WithX), lifecycle (Start/Stop/Health), core.Error + ErrorCode, context.Context first. Pulled by architect, developer-go, security, qa and doc-writer. TRIGGER: \"create a new package\", \"add a registry for X\", \"how should this package be laid out\", \"wire provider auto-registration\", \"add functional options\", \"does this package follow the framework pattern\". DO NOT TRIGGER for: designing the interface methods, hooks or middleware signatures (go-interfaces); iter.Seq2 producer/consumer code (streaming-patterns); a concrete provider for an existing registry (provider-implementation); test layout and mocks (go-testing); package docs and tutorials (doc-writing); CI, release or deployment (outside the framework specialists)."
when_to_use: A framework specialist is shaping, implementing, reviewing or documenting a Go package and needs the canonical Beluga package layout, registry, options, lifecycle and error conventions.
type: specialist
---

# go-framework

Canonical body (code templates and rules): `.claude/skills/go-framework/SKILL.md`. Read it first; this file is the Guild routing wrapper so both hosts resolve one copy of the patterns.

# When to use it

- A new extensible package, or a new registry inside an existing one.
- Reviewing whether a package follows Interface → Registry → Hooks → Middleware (`AGENTS.md` critical rule 2; `tool` uses an instance `tool.Registry` instead).
- Adding `WithX()` options, `init()` registration, lifecycle methods or typed errors.

# When not to use it

- Method sets, hook fields or middleware signatures → `go-interfaces`.
- Stream producers / consumers → `streaming-patterns`.
- A provider for an existing registry → `provider-implementation`.
- Test structure and mocks → `go-testing`.

# Required inputs

Target package path; the plan or acceptance criteria for the change; the package's current files (match its style exactly).

# Output format

Code or review findings that cite `file:line` and name the convention applied (registry, options, lifecycle, error code).

# Workflow steps

1. Read `.claude/skills/go-framework/SKILL.md` and `.guild/wiki/patterns/provider-registration.md`, `error-handling.md`, `middleware.md`, `hooks.md` as relevant.
2. Read `.guild/wiki/architecture/invariants.md` and `package-map.md` for the target package.
3. Apply the layout; add `var _ Interface = (*Impl)(nil)` for every implementation.
4. New registries panic on duplicate or empty names; `New()` returns a typed `core.Error` for unknown names.

# Evidence requirements

`go build ./...` and `go vet ./...` pass; registry, options and errors are visible in the diff with `file:line`.

# Escalation rules

Layer placement or a change to an invariant → `architect` (and an ADR in `.guild/wiki/architecture/decisions.md`). A cross-repo design question → `/consult <specialist>`.

# Safety constraints

No global mutable state outside registries; zero external deps in `core/` and `schema/`; the library never imports `k8s/`.

# Eval cases

See `evals.json` next to this file.
