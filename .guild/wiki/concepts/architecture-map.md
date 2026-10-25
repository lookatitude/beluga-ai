---
type: concept
title: "Architecture map (stub)"
description: "Cheap-scan architecture stub for framework. Confidence-tagged candidate; the deep knowledge graph is deferred."
owner: orchestrator
confidence: medium
trust_tier: synthesized
status: candidate
importance: low
source_refs: [codebase-map]
created_at: 2026-10-25
updated_at: 2026-10-25
expires_at: null
supersedes: null
sensitivity: internal
generated_from_commit: 41f1c83f2f54abbc84a70ace4e31b3843f74f2bd
recall_importance: 3
---
# Architecture map (stub)

Built from the cheap scan (codebase map, a derived cache outside the repo; 1522 files at
commit `41f1c83f`; languages: go, html, json, makefile, markdown, shell, toml, yaml; frameworks: none detected) plus this repo's
`CLAUDE.md`. Candidate page: every line carries a confidence tag. The deep knowledge
graph and onboarding tour were **not** built at init; run `/guild:learn` to build them.

## What this repo is

- **[high]** `beluga-ai`: the Go-native agentic AI framework, module `github.com/lookatitude/beluga-ai/v2`, Go 1.26+. The engineering layer of the Beluga project (code, tech docs, releases). Source: `CLAUDE.md` §Project scope.
- **[high]** Public streaming APIs use `iter.Seq2[T, error]`, never channels; `Invoke()` is stream plus collect. Source: `CLAUDE.md` §Critical rules.
- **[high]** Every extensible package follows interface → registry (`Register`/`New`/`List`) → hooks → middleware; providers self-register in `init()`. Source: `CLAUDE.md` §Critical rules.

## 7-layer model

Authoritative: `docs/architecture/01-overview.md` (DOC-01) and `docs/architecture/18-package-dependency-map.md`. Layer N imports only from layers 1..N-1.

- **[high]** Layer 1, Foundation: `core`, `schema`, `config`, `o11y`. Zero external deps beyond stdlib + otel in `core/` and `schema/`.
- **[medium]** Layers 2–5 (cross-cutting, capability, protocol gateway, orchestration): see DOC-01; package-to-layer mapping not re-derived by this scan.
- **[high]** Layer 6, Agent runtime: Runner, Agent, Executor, Planner (seven strategies, ReAct to LATS), Team.
- **[high]** `k8s/` is an optional overlay the library never imports. Source: `CLAUDE.md` rule 4.

## Modules by size (from the scan)

| Path | Files |
|---|---|
| `rag` | 178 |
| `agent` | 127 |
| `memory` | 114 |
| `llm` | 103 |
| `voice` | 99 |
| `eval` | 86 |
| `docs` | 76 |
| `cmd` | 73 |
| `guard` | 61 |
| `runtime` | 54 |
| `internal` | 47 |
| `tool` | 45 |

## Entrypoints

- **[high]** `cmd/beluga/main.go`
- **[high]** `cmd/docgen/main.go`
- **[high]** `internal/tools/provider-catalog/main.go`

## Knowledge that stays where it is

- **[high]** This repo's wiki (patterns, architecture invariants, package map, corrections C-NNN) lives in `.guild/wiki/` with its legacy structure (moved from `.wiki/` on 2026-10-25); `.claude/hooks/wiki-query.sh` reads it.
- **[high]** `docs/architecture/` (numbered DOC-NN) is the canonical prose architecture.
