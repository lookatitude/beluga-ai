---
description: Development workflow rules defining agent roles, handoff protocol, and quality gates.
alwaysApply: true
---

# Workflow Rules

## Task Routing

| Task Type | Workflow | Agents |
|-----------|----------|--------|
| New feature / complex task | Full workflow | Architect → Researcher → Architect → Developer → Security Reviewer → QA |
| Bug fix / small change | Short workflow | Developer → Security Reviewer → QA |
| Documentation only | Direct | Doc Writer |
| Code review | Security review | Security Reviewer |

## Full Workflow

1. **Architect** analyzes the task, produces a research brief with specific topics.
2. **Researcher** investigates each topic, returns structured findings with evidence.
3. **Architect** receives findings, makes design decisions, produces an implementation plan with:
   - Interface definitions (Go code)
   - Implementation tasks in dependency order
   - Acceptance criteria per task (measurable, verifiable by QA)
4. **Developer** implements code AND tests per the plan.
   - Must pass: `go build ./...`, `go vet ./...`, `go test ./...` before submitting.
5. **Security Reviewer** performs thorough review against all security checklists.
   - If issues found → return to Developer with specific findings → Developer fixes → resubmit.
   - **Must achieve 2 consecutive clean passes with zero issues** before proceeding.
   - Any issue resets the clean pass counter to 0.
6. **QA Engineer** validates every acceptance criterion with evidence.
   - If any criterion fails → return to Developer → fix → back through Security Review.

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
gofmt -l . | grep -v ".claude/worktrees"
golangci-lint run ./...
gosec -quiet ./...
govulncheck ./...
```

Pre-existing findings in files you did NOT change do not block your commit, but
MUST be documented in the commit message so reviewers know to ignore or fix
separately. New findings in files you DID change MUST be fixed before commit.

### Before push (MANDATORY on every branch that will open or update a PR)

PRs are expected to pass **both** SonarCloud quality gate **and** Greptile review on first submission. Most regressions that ship back after `git push` fall into a small set of recurring patterns that are cheap to catch locally. Run this pre-push pass on every changed file:

1. **Verify `git rev-list --left-right --count origin/<branch>...HEAD` is `0 0` after every push.** If it shows `N 0`, the push silently skipped (no tracking or `push.autoSetupRemote` disabled). Use `git push origin <branch>:<branch>` explicitly for new branches. See workspace `.wiki/corrections.md` W-013.
2. **Greptile-pattern self-review on every file in the diff:**
   - Non-deterministic `for k, v := range map[...]...` writes with partial-failure risk → replace with a slice of pairs + rollback.
   - GitHub Actions uses mutable `@vN` tags on any new workflow that will exist as a security-scanned artifact (scaffolded CI templates, release workflows) → pin to a full SHA OR document that repo-wide `@vN` convention is deliberate.
   - `curl | tar` or `curl | sh` of binaries without `--proto '=https' --tlsv1.2` and without a `sha256sum -c` checksum verification step.
   - Exported interfaces renamed without `type OldName = NewName` back-compat aliases (S8196). See workspace `.wiki/corrections.md` W-012 rule 2.
   - Dead exported accessor that could be unexported or removed (no external callers in `grep -rn <Name> .`).
   - Refactors that change observable behavior without a test asserting the new behavior — especially CC-reduction refactors that introduce new mutations.
3. **SonarCloud parity (STRONGLY RECOMMENDED, not blocking):** `sonar verify --file <changed-file>` per touched file catches S3776 (cognitive complexity), S1192 (duplicate literals), S8196 (interface naming), and S8193 (unnecessary vars) locally, before the full CI scan runs. Skip for doc-only or generated-file changes.
4. **Scaffolded-template diffs require a `hotspot-triage.md` lookup.** When a PR changes `cmd/beluga/scaffold/templates/`, consult `.wiki/hotspot-triage.md` for the canonical Safe-review comment to attach after the next SonarCloud scan. Plan the `sonar api post /api/hotspots/change_status` follow-up at push time so it doesn't block the merge.

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

- Each agent produces structured output for the next agent.
- Never skip an agent in the chain.
- If blocked, escalate to the user — don't guess.
- Document decisions and rationale at each step.
