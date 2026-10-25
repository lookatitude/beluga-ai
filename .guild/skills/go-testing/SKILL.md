---
derived_from_template: guild.skill_template.v1
name: go-testing
description: "Beluga AI v2 Go test conventions: table-driven subtests, testify assert/require, stream and context-cancellation tests, internal/testutil/mock<pkg> mocks with error injection and call counting, //go:build integration files, *_bench_test.go benchmarks, -race always. Pulled by developer-go and qa. TRIGGER: \"write tests for X\", \"add a table-driven test\", \"test context cancellation on the stream\", \"add a mock for the embedder\", \"write an integration test\", \"add a benchmark\", \"is this test suite good enough\". DO NOT TRIGGER for: deciding what the package API looks like (go-interfaces / go-framework); provider HTTP error mapping itself (provider-implementation); CI workflow changes under .github/ (out of scope for framework specialists); security probes such as injection or authz tests (the security specialist's lane)."
when_to_use: developer-go is writing the failing test of a Red/Green TDD cycle, or qa is judging test quality and coverage for acceptance.
type: specialist
---

# go-testing

Canonical body (test templates and rules): `.claude/skills/go-testing/SKILL.md`. Read it first; this file is the Guild routing wrapper.

# When to use it

- The RED step of Red/Green TDD (`AGENTS.md` critical rule 7).
- Adding mocks under `internal/testutil/mock<package>/`.
- Integration tests (`//go:build integration`, `*_integration_test.go`) and benchmarks.
- QA judging whether tests cover happy path, error paths, edge cases and context cancellation.

# When not to use it

- API or package design → `go-interfaces`, `go-framework`.
- CI configuration under `.github/` → out of scope.

# Required inputs

The acceptance criteria being pinned; the package under test; existing mocks in `internal/testutil/`.

# Output format

`*_test.go` alongside source, or a QA finding table (criterion · PASS/FAIL · evidence command + output).

# Workflow steps

1. Read `.claude/skills/go-testing/SKILL.md` and `.guild/wiki/patterns/testing.md`.
2. Write the failing table-driven test; run it and confirm RED.
3. Implement until GREEN; refactor with tests green.
4. Run `go test -race -count=1 ./<package>/...`; coverage via `go test -coverprofile=cover.out ./<package>/... && go tool cover -func=cover.out` (target >80% on critical paths).

# Evidence requirements

The exact command and its output. Never claim green for a command that did not run locally (CI parity: Go 1.27.3, `GOTOOLCHAIN=local`).

# Escalation rules

A behaviour that cannot be tested without changing the design → `architect`. A flaky test you cannot reproduce → report it with the reproduction attempts; do not quarantine silently.

# Safety constraints

Tests are deterministic; no real network calls in unit tests (use `httptest`); no secrets in fixtures beyond obvious fakes.

# Eval cases

See `evals.json` next to this file.
