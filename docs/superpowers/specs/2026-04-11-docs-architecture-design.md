# Docs Architecture Redesign

**Date:** 2026-04-11
**Status:** Approved — ready for execution
**Source plan:** `docs/beluga-ai-v2-docs-plan.md`
**Visual sources:** `docs/beluga_full_layered_architecture.svg`, `docs/beluga_request_lifecycle.svg`
**Supersedes:** `docs/architecture.md`, `docs/concepts.md`, `docs/packages.md`, `docs/beluga-ai-v2-comprehensive-architecture.md`

## Purpose

Replace the current monolithic documentation (4 large markdown files predating v2) with the 29-document structure defined in the v2 docs plan. The new structure maps one document per architectural concern, grounds every explanation in real `file:line` references from the codebase, and uses Mermaid for all diagrams with automatic SVG rendering via the website build.

## Scope

In scope:
- Delete the 4 superseded monolithic docs.
- Create `docs/{architecture,patterns,guides,reference}/` with 29 documents total:
  - 18 architecture docs (`architecture/01-overview.md` … `architecture/18-package-dependency-map.md`)
  - 8 pattern docs (`patterns/registry-factory.md` … `patterns/context-propagation.md`)
  - 7 guide docs (`guides/first-agent.md` … `guides/deploy-docker.md`)
  - 4 reference docs (`reference/interfaces.md`, `configuration.md`, `glossary.md`, `providers.md`)
- Embed the two new SVGs directly in their matching docs (DOC-01 and DOC-04).
- Update `CLAUDE.md` `@-imports` to point at the new canonical docs.
- Update `.wiki/index.md` retrieval routing to reference new doc paths.
- Add a top-level `docs/README.md` as the index for the new structure.

Out of scope (explicit):
- Website (`docs/website/`) updates — separate concern, handled by `developer-web` later.
- The older `docs/plans/*` files (SEO, search, gradient homepage) — unrelated content, leave as-is.
- `docs/beluga-ai-website-blueprint-v2.md` — separate initiative.
- `docs/beluga-ai-v2-agents-plan.md` — historical, kept as archive.
- Running the actual Beluga Go framework code. No Go changes in this spec.

## Target directory layout

```
docs/
├── README.md                          # top-level index (new)
├── architecture/
│   ├── README.md                      # architecture index with the 7-layer map
│   ├── 01-overview.md                 # master map (embeds beluga_full_layered_architecture.svg)
│   ├── 02-core-primitives.md
│   ├── 03-extensibility-patterns.md
│   ├── 04-data-flow.md                # (embeds beluga_request_lifecycle.svg)
│   ├── 05-agent-anatomy.md
│   ├── 06-reasoning-strategies.md
│   ├── 07-orchestration-patterns.md
│   ├── 08-runner-and-lifecycle.md
│   ├── 09-memory-architecture.md
│   ├── 10-rag-pipeline.md
│   ├── 11-voice-pipeline.md
│   ├── 12-protocol-layer.md
│   ├── 13-security-model.md
│   ├── 14-observability.md
│   ├── 15-resilience.md
│   ├── 16-durable-workflows.md
│   ├── 17-deployment-modes.md
│   └── 18-package-dependency-map.md
├── patterns/
│   ├── README.md
│   ├── registry-factory.md
│   ├── middleware-chain.md
│   ├── hooks-lifecycle.md
│   ├── streaming-iter-seq2.md
│   ├── functional-options.md
│   ├── provider-template.md
│   ├── error-handling.md
│   └── context-propagation.md
├── guides/
│   ├── README.md
│   ├── first-agent.md
│   ├── custom-provider.md
│   ├── custom-planner.md
│   ├── multi-agent-team.md
│   ├── deploy-kubernetes.md
│   ├── deploy-temporal.md
│   └── deploy-docker.md
├── reference/
│   ├── README.md
│   ├── interfaces.md
│   ├── configuration.md
│   ├── glossary.md
│   └── providers.md                   # NEW: addresses broken CLAUDE.md @-import
├── plans/                             # preserved (unrelated)
├── superpowers/                       # preserved
├── website/                           # preserved
├── beluga-ai-v2-agents-plan.md        # preserved (historical)
├── beluga-ai-v2-docs-plan.md          # preserved (source of truth for this work)
├── beluga-ai-website-blueprint-v2.md  # preserved (out of scope)
├── beluga_full_layered_architecture.svg   # preserved (embedded in 01-overview.md)
└── beluga_request_lifecycle.svg           # preserved (embedded in 04-data-flow.md)
```

## Deletion list

Files to remove (content preserved in git history):

1. `docs/architecture.md` — superseded by `docs/architecture/*`.
2. `docs/concepts.md` — superseded by `docs/architecture/02-core-primitives.md`, `03-extensibility-patterns.md`, `05-agent-anatomy.md`, and `reference/glossary.md`.
3. `docs/packages.md` — superseded by `docs/architecture/18-package-dependency-map.md` and `docs/reference/interfaces.md`.
4. `docs/beluga-ai-v2-comprehensive-architecture.md` — entirely superseded by the new 18-doc architecture set.
5. `docs/beluga_agent_workflow_system.svg` — stale visual (the new agent system is now documented via `.claude/` directly).
6. `docs/beluga_full_runtime_architecture.svg` — stale visual, superseded by `beluga_full_layered_architecture.svg`.

Files updated to reference the new structure:

- `CLAUDE.md` — replace broken `@docs/providers.md` and update `@docs/concepts.md`, `@docs/packages.md`, `@docs/architecture.md` to new paths.
- `.wiki/index.md` — retrieval routing table points at the new docs.

## Document template

Every new document has the same structure (scaled to content):

```markdown
# DOC-NN: <title>

**Audience:** <who is this for>
**Prerequisites:** <what to read first, if any>
**Related:** <cross-refs to other new docs>

## Overview
2-3 sentence TL;DR.

## <Concept 1>
Explanation with rationale (the "why").

```mermaid
...diagram...
```

```go
// canonical code example from the codebase, with file:line reference
```

## <Concept 2>
...

## Common mistakes
Sourced from `.wiki/corrections.md`.

## Related reading
- Cross-links to other docs
```

Word target: 1000–2000 words per document.

## Diagram strategy

All diagrams use **Mermaid**. Sources live inline in the markdown files so the single source of truth is the doc itself. Exported SVGs for sharing land in `docs/diagrams/*.svg` during the website build (the Astro/Starlight pipeline handles this — no hand-exported artifacts in this spec's scope).

Two exceptions: DOC-01 and DOC-04 embed the existing hand-drawn SVGs (`beluga_full_layered_architecture.svg`, `beluga_request_lifecycle.svg`) via standard markdown image syntax. The Mermaid source for the same diagrams is also included as a secondary fallback for non-SVG-capable renderers and for agents that need to query structural information via text.

## Code examples

Every code example:
- Is sourced from an actual `file:line` in the Beluga codebase (use `.wiki/architecture/package-map.md` and `.wiki/patterns/*` for canonical references).
- Has full `github.com/lookatitude/beluga-ai/...` imports.
- Handles errors explicitly.
- Compiles when extracted to a throwaway file.

If a feature hasn't been implemented yet, the doc marks it as `**Status:** Planned (not yet in v2)` and gives the target interface without pretending it exists.

## Execution order (29 docs)

Per plan priorities, plus the new `providers.md`:

1. **Phase 1 — Foundation (3 docs, unblocks everything)**
   - DOC-01, DOC-02, DOC-03
2. **Phase 2 — Runtime core (5 docs)**
   - DOC-05, DOC-04, DOC-06, DOC-07, DOC-08
3. **Phase 3 — Capabilities (5 docs)**
   - DOC-09, DOC-10, DOC-11, DOC-12, DOC-13
4. **Phase 4 — Infra & cross-cutting (5 docs)**
   - DOC-14, DOC-15, DOC-16, DOC-17, DOC-18
5. **Phase 5 — Patterns (8 docs, parallel-safe)**
   - PAT-01..08
6. **Phase 6 — Guides (7 docs, parallel-safe)**
   - first-agent, custom-provider, custom-planner, multi-agent-team, deploy-kubernetes, deploy-temporal, deploy-docker
7. **Phase 7 — Reference (4 docs, parallel-safe)**
   - interfaces, configuration, glossary, providers

Parallel tracks (phases 5–7) can be dispatched to multiple agents simultaneously since they have no inter-doc dependencies beyond phases 1–4.

## Acceptance criteria

- [ ] Four monolithic docs deleted; the two stale SVGs removed.
- [ ] `docs/{architecture,patterns,guides,reference}/` directories exist with the 29 files listed above.
- [ ] `docs/README.md` exists and links to all four sub-indexes.
- [ ] Each sub-directory has a `README.md` index.
- [ ] `CLAUDE.md` `@-imports` updated; no broken references.
- [ ] `.wiki/index.md` retrieval routing table updated to reference new doc paths.
- [ ] DOC-01 embeds `beluga_full_layered_architecture.svg`.
- [ ] DOC-04 embeds `beluga_request_lifecycle.svg`.
- [ ] Every doc has a "Related reading" section with working relative links.
- [ ] Every code block has a `file:line` reference comment.
- [ ] Common mistakes sections source from `.wiki/corrections.md`.
- [ ] Commit(s) created in logical chunks (spec, cleanup, per-phase content).

## Out-of-scope trade-offs

- **Diagram SVG export.** Only the two existing hand-drawn SVGs live as files. All other diagrams stay as Mermaid source in the markdown. Rendering to SVG happens at website build time. If the user wants static SVG artifacts for each diagram, that's a follow-up.
- **Guide compilability.** Guide code examples are aspirationally compilable, but until someone runs them through a real build check, consider them "reviewed-by-eye only."
- **Reference/interfaces.md automation.** Manually curated from `.wiki/architecture/package-map.md`. Future: generate from `go doc` via `/wiki-learn` extension.

## Known risks

- **29 docs is a lot of content to write in one session.** Risk of quality drift in later phases. Mitigation: strict template adherence, each doc explicitly references its source file:line, phases 5–7 dispatched in parallel to reduce context pressure.
- **Some diagrams in the plan describe features not yet in the code** (e.g., LATS planner, S2S voice, Temporal integration). Mark these as "Status: Planned" and document the target interface.
- **Cross-reference drift.** As docs move between phases, relative links might get stale. Mitigation: final smoke pass grepping for broken `../` links before commit.
