<!--
Sync Impact Report:
Version change: 1.0.0 → 2.0.0 (Focused restructuring for clarity and simplicity)
Modified principles: N/A (structure preserved but simplified)
Added sections: Git Workflow, Review & Acceptance Checklist
Removed sections: Detailed implementation standards (moved to framework patterns)
Templates requiring updates: ✅ plan-template.md updated, ✅ tasks-template.md updated, ✅ spec-template.md verified (no references to update)
Follow-up TODOs: None - all template references updated
-->

# Beluga AI Framework Constitution (Focused)

## Purpose
- Ensure consistent, extensible, configurable, and observable Go packages across the Beluga AI Framework.

## Core Principles (Required)
- Interface Segregation (ISP): small, focused interfaces.
- Dependency Inversion (DIP): depend on abstractions; constructor injection only.
- Single Responsibility (SRP): one clear reason to change per package/struct.
- Composition over Inheritance: prefer interface embedding + functional options.

## Framework Patterns (Required)
- Standard package layout:
  pkg/{name}/
  - iface/ (public interfaces, types)
  - internal/ (private implementation)
  - providers/ (only if multi-backend)
  - config.go (config structs with mapstructure,yaml,env,validate tags; defaults)
  - metrics.go (OpenTelemetry metrics/tracing integration)
  - errors.go (custom error: Op/Err/Code + codes)
  - {name}.go (interfaces + factories)
  - README.md (usage + examples)
- Multi-provider registry (if applicable):
  - Register(name string, creator func(ctx context.Context, cfg Config) (Interface, error))
  - NewProvider(ctx context.Context, name string, cfg Config) (Interface, error)
- Interfaces: "-er" for single-method (Embedder), nouns for multi-method (VectorStore). Use embedding to extend without breaking changes.
- Configuration: functional options; validate at creation; respect context for timeouts/cancellation.
- Observability: OTEL spans in public methods; structured logging; counters/histograms with labels; add health checks where relevant.
- Error handling: custom error type with codes; wrap and preserve chains; never swallow errors; always respect context.
- Dependencies: constructor injection; avoid globals/singletons (except safe registries).
- Testing: table-driven tests; mocks in internal/mock/; integration tests in tests/integration/; benchmarks for perf-critical paths; thread-safety tests when concurrency is involved.
- Documentation: package comment, function docs with examples, README for complex packages.
- Code generation: generate mocks/validation/metrics where beneficial.
- Evolution: SemVer; deprecate with notices; provide migration guides.

## Git Workflow (Required)
- Branching: one feature per branch, named `feature/<id>-<summary>` or `NNN-for-the-<package>`.
- Task-level commits: commit after each task (atomic, descriptive), referencing task IDs.
- Push + PR: upon feature completion, push the feature branch to origin and open a PR into `develop`.
- CI gates: PR must pass lint, unit, integration (and benchmarks if applicable) plus review.
- Merge policy: squash/rebase per repo convention; `develop` → `main` only after full validation.

## Review & Acceptance Checklist (Every PR)
- Package layout matches standard (iface/, internal/, providers/ if needed).
- Interfaces are small; dependencies injected; no global state.
- Config uses tags + validation; functional options applied; context respected.
- Observability: OTEL spans + metrics in public methods; structured logs; health checks if relevant.
- Error type with Op/Err/Code + error codes; errors wrapped; context respected.
- Tests: unit (table-driven), mocks, integration; benchmarks for perf-critical code; concurrency tests where applicable.
- Docs updated (README/examples); changelog when necessary.
- Git workflow followed (task commits, push, PR to develop).

## Governance
- Amend via PR with rationale, impact assessment, and migration notes; apply SemVer (MAJOR breaking, MINOR additive, PATCH clarifications).