---
derived_from_template: guild.skill_template.v1
name: provider-implementation
description: "Implementing a provider for an existing Beluga AI v2 registry (LLM, embedding, vectorstore, STT/TTS, memory, …): full interface implementation, init() + Register(), HTTP/SDK error mapping to core.Error codes (401→ErrAuth, 429→ErrRateLimit, 408/504→ErrTimeout, 400→ErrInvalidInput), context cancellation, usage metrics, httptest-based tests and testdata recordings, regenerating docs/reference/providers.md. Pulled by developer-go. TRIGGER: \"add a provider for X\", \"implement the Mistral ChatModel\", \"new vectorstore backend\", \"map this provider's errors\", \"register the provider\", \"provider catalog is stale\". DO NOT TRIGGER for: designing a new extension interface or registry (go-interfaces / go-framework); generic stream composition (streaming-patterns); provider comparison research before choosing one (the researcher specialist); provider documentation pages beyond the generated catalog (doc-writing)."
when_to_use: developer-go is adding or fixing a concrete provider under `<package>/providers/<name>/` for a registry that already exists.
type: specialist
---

# provider-implementation

Canonical body (checklist, file layout, template, error mapping): `.claude/skills/provider-implementation/SKILL.md`. Read it first; this file is the Guild routing wrapper.

# When to use it

- New or changed provider package under an existing registry (`AGENTS.md` critical rule 3: providers auto-register in `init()`).
- Provider error mapping, auth handling, usage metrics.

# When not to use it

- New interfaces or registries → `go-interfaces`, `go-framework`.
- Choosing which provider to support → `researcher` (input) and `architect` (decision).

# Required inputs

The registry's interface and factory type, the provider's API docs, existing sibling providers to match.

# Output format

`<package>/providers/<name>/` with implementation, `init()` registration, error mapping, tests with `httptest` and `testdata/` recordings.

# Workflow steps

1. Read `.claude/skills/provider-implementation/SKILL.md` and `.guild/wiki/patterns/provider-registration.md`.
2. Write failing tests against recorded responses (`go-testing`).
3. Implement; add `var _ Interface = (*Impl)(nil)`; map every error status to a `core.Error` code.
4. Regenerate the catalog: `go run ./internal/tools/provider-catalog > docs/reference/providers.md`; `make docs-providers` must pass (CI runs it).

# Evidence requirements

Tests pass with `-race`; `make docs-providers` passes; error-mapping table covered by tests.

# Escalation rules

The provider needs an interface change → `architect`. Untrusted-input or credential-handling questions → `security` (or `/consult security-architect`).

# Safety constraints

API keys come from `config.ProviderConfig` / environment — never hardcoded or logged; no secrets in `testdata/`; tests never hit the live API.

# Eval cases

See `evals.json` next to this file.
