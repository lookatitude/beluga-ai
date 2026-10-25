---
description: Development workflow rules defining agent roles, handoff protocol, and quality gates.
alwaysApply: true
---

# Workflow Rules

## Roles

The **main session is the coordinator** (`.claude/agents/coordinator.md`). It routes tasks, dispatches the agents below (as agent-team teammates for multi-step or PR-bound work — plan-ack, CI-parity, no-over-scoping, correction-log handoff; see workspace `AGENTS.md` § Teammate dispatch), verifies their output, runs the Codex adversarial review, and owns every outward action: push, PR, merge, Linear/Sonar/Notion writes. Teammates prepare commits on their branch and never push.

In a `/guild` run the same roles are the Guild specialists in `.guild/agents/` (mapping in `AGENTS.md` § Agent team); the T0 session coordinates and Guild's `review: cross` engages Codex.

## Task Routing

| Task Type | Workflow | Agents |
|-----------|----------|--------|
| New feature / complex task | Full workflow | Architect → Researcher → Architect → Developer → Security Reviewer → QA |
| Bug fix / small change | Short workflow | Developer → Security Reviewer → QA |
| Documentation only | Direct | Doc Writer |
| Code review | Security review | Security Reviewer |
| Any branch before its PR | Adversarial review | Coordinator runs `/codex:adversarial-review` |

## Full Workflow

1. **Architect** analyzes the task, produces a research brief with specific topics.
2. **Researcher** investigates each topic, returns structured findings with evidence.
3. **Architect** receives findings, makes design decisions, produces an implementation plan with:
   - Interface definitions (Go code)
   - Implementation tasks in dependency order
   - Acceptance criteria per task (measurable, verifiable by QA)
4. **Developer** implements code AND tests per the plan.
   - Must pass the full "Before commit" gate below (build, vet, race tests, tidy, gofmt, golangci-lint, gosec, govulncheck) before submitting.
5. **Security Reviewer** performs thorough review against all security checklists.
   - If issues found → return to Developer with specific findings → Developer fixes → resubmit.
   - **Must achieve 2 consecutive clean passes with zero issues** before proceeding.
   - Any issue resets the clean pass counter to 0.
6. **QA Engineer** validates every acceptance criterion with evidence.
   - If any criterion fails → return to Developer → fix → back through Security Review.
7. **Coordinator** runs the Codex adversarial review on the branch (`/codex:adversarial-review`; needs the `codex@openai-codex` plugin and `codex login status` OK). Findings return to the Developer. If Codex is unavailable, record weak (same-family) independence in the PR body — it is not an equivalent pass.
8. **Coordinator** runs the "Before push" checks, pushes, and opens the PR.

## Quality Gates

### Branch discipline (MANDATORY, every change)

See [`branch-discipline.md`](./branch-discipline.md) for branch + PR rules and Linear-integrated naming (`<type>/loo-NN-<slug>`).

### Before commit (MANDATORY, every commit — not just before push)
Run the full suite locally BEFORE `git commit`. Only commit when all pass:

```bash
go build ./...
go vet ./...
go test -race ./...
go mod tidy && git diff --exit-code go.mod go.sum
test -z "$(gofmt -l $(git ls-files '*.go'))"   # fails if any tracked file is unformatted
golangci-lint run ./...
gosec -quiet ./...
govulncheck ./...
```

Pre-existing findings in files you did NOT change do not block your commit, but
MUST be documented in the commit message so reviewers know to ignore or fix
separately. New findings in files you DID change MUST be fixed before commit.

CI parity: CI runs Go 1.27.3 with `GOTOOLCHAIN=local` and pins golangci-lint
v2.15.0, gosec v2.30.0, govulncheck v1.9.0 (`make tools` installs them;
`make ci` mirrors the locally runnable blocking CI jobs; Trivy, Gitleaks and CodeQL run only in CI). Never claim a check green when the
tool could not run locally — say it did not run.

### Before push (MANDATORY on every branch that will open or update a PR)

Run by the **coordinator**, who is the only one that pushes.

PRs are expected to pass **both** SonarCloud quality gate **and** Greptile review on first submission. Most regressions that ship back after `git push` fall into a small set of recurring patterns that are cheap to catch locally. Run this pre-push pass on every changed file:

1. **Verify `git rev-list --left-right --count origin/<branch>...HEAD` is `0 0` after every push.** If it shows `N 0`, the push silently skipped (no tracking or `push.autoSetupRemote` disabled). Use `git push origin <branch>:<branch>` explicitly for new branches. See workspace `../.guild/wiki/standards/corrections.md` W-013.
2. **Greptile-pattern self-review on every file in the diff:**
   - Non-deterministic `for k, v := range map[...]...` writes with partial-failure risk → replace with a slice of pairs + rollback.
   - GitHub Actions uses mutable `@vN` tags on any new workflow that will exist as a security-scanned artifact (scaffolded CI templates, release workflows) → pin to a full SHA OR document that repo-wide `@vN` convention is deliberate.
   - `curl | tar` or `curl | sh` of binaries without `--proto '=https' --tlsv1.2` and without a `sha256sum -c` checksum verification step.
   - Exported interfaces renamed without `type OldName = NewName` back-compat aliases (S8196). See workspace `../.guild/wiki/standards/corrections.md` W-012 rule 2.
   - Dead exported accessor that could be unexported or removed (no external callers in `grep -rn <Name> .`).
   - Refactors that change observable behavior without a test asserting the new behavior — especially CC-reduction refactors that introduce new mutations.
3. **SonarCloud parity (STRONGLY RECOMMENDED, not blocking):** `sonar verify --file <changed-file>` per touched file catches S3776 (cognitive complexity), S1192 (duplicate literals), S8196 (interface naming), and S8193 (unnecessary vars) locally, before the full CI scan runs. Skip for doc-only or generated-file changes.
4. **Scaffolded-template diffs require a `hotspot-triage.md` lookup.** When a PR changes `cmd/beluga/scaffold/templates/`, consult `.guild/wiki/hotspot-triage.md` for the canonical Safe-review comment to attach after the next SonarCloud scan. Plan the `sonar api post /api/hotspots/change_status` follow-up at push time so it doesn't block the merge.

Failures caught in pre-push cost no CI cycle and preserve the PR's first-pass-clean reputation with Greptile.

### Before Security Review (additional, for full-workflow tasks)
- All of the "Before commit" checks above pass
- All of the "Before push" checks above pass
- No `TODO` or `FIXME` without associated tracking

### Security Review (2 clean passes required)
- Input validation and injection prevention
- Authentication and authorization
- Cryptography and data protection
- Concurrency and resource safety
- Error handling and information disclosure
- Dependencies and supply chain
- Architecture compliance

### QA Validation (final gate)
- Every acceptance criterion has explicit PASS/FAIL with evidence
- All verification commands run and output confirmed
- Architecture compliance verified (iter.Seq2, registry, context.Context, etc.)
- Test coverage on critical paths confirmed

## Handoff Protocol

- Teammates send a structured plan via `SendMessage` first and wait for an explicit `go` before mutating state.
- Each agent produces structured output for the next agent.
- Never skip an agent in the chain.
- If blocked, escalate to the coordinator (and through it to the user) — don't guess.
- Document decisions and rationale at each step.
