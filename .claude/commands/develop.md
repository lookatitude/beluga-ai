---
name: develop
description: Implement a Linear sub-issue end-to-end via the framework agent chain (architect → developer-go → reviewer-security → reviewer-qa → Codex adversarial review → coordinator opens the PR).
---

Develop: $ARGUMENTS

$ARGUMENTS should be a Linear sub-issue ID (e.g., `LOO-43`) created by `/plan-feature` in the workspace. For work not tracked in Linear (small fixes, exploration), pass a brief path instead: `/develop research/briefs/<slug>.md` — then most of the Linear-specific steps below are skipped and the agents work from the brief alone.

## Who runs what

The main session is the **coordinator** (`.claude/agents/coordinator.md`): it runs pre-flight, dispatches each step as a teammate in an agent team (workspace `AGENTS.md` § Teammate dispatch — plan-ack checkpoint, CI-parity gate, no-over-scoping, correction-log handoff), verifies their output, and owns every outward action. Teammates prepare commits on the branch; they never push, open or merge PRs, or write to Linear/Sonar/Notion. Reading Linear in pre-flight is a coordinator action.

## Claude 5 chain notes

- **Acceptance criteria:** Treat the brief’s Framework tasks as a **literal** checklist—each item done or explicitly deferred with cause.
- **Reviewer prompts:** Security and QA reviewers should cite criteria by bullet, not loose paraphrase.
- **Docs:** Tech docs shipped in `docs/` are part of “done” when the brief says so—don’t rely on implicit completion.

## Pre-flight (Linear-integrated)

### 1. Fetch the sub-issue from Linear

If $ARGUMENTS matches `LOO-\d+`:

```
mcp__claude_ai_Linear__get_issue(id="$ARGUMENTS")
```

Capture: title, description, labels, parentId.

Then fetch the parent:

```
mcp__claude_ai_Linear__get_issue(id="<parentId>")
```

Capture: title, description, labels, any brief path referenced in the description.

### 2. Read the workspace brief

The parent issue's description should reference a brief at `research/briefs/<slug>.md` in the workspace repo (relative path from the workspace root — `../research/briefs/<slug>.md` when cd'd into framework/).

Read that brief fully. It is source of truth for Problem, Proposed solution, Success criteria, Framework tasks, Risks, and any open questions.

If the brief is not findable: proceed with just Linear parent + sub-issue content. Flag to the user that the brief was not located.

### 3. Derive the branch name

From the sub-issue's labels, pick the branch prefix. Mapping (matches workspace `/plan-feature`):

- `Feature` label → `feat/`
- `Bug` label → `fix/`
- `Improvement` label → `refactor/`
- Otherwise → `chore/` (workspace `/plan-feature` deliberately does NOT create a `documentation` type label for doc-only changes — they flow through untyped to `chore/`)

Branch name: `<prefix>/loo-NN-<short-slug>` where `<short-slug>` is a 3-5 word kebab-case summary of the sub-issue title. Linear auto-links via the `loo-NN` portion regardless of prefix.

Example: sub-issue `LOO-43 "Implement cmd/beluga/ cobra scaffold with version + providers"` labeled `layer:framework, Feature, size:regular` → `feat/loo-43-beluga-cli-foundation`.

If $ARGUMENTS is a brief path (no Linear ID), fall back to `feat/<brief-slug>` or let the architect/developer pick a branch during implementation.

### 4. Create the branch

```bash
git checkout main
git pull
git checkout -b <branch-name>
```

## Agent chain

With context loaded (sub-issue + parent + brief), proceed with the standard framework implementation chain per `.claude/rules/workflow.md`:

### Step 1 — Architect

`@agent-architect` analyzes the brief's Framework tasks + Proposed solution, produces a research request if gaps exist, then (after any research) outputs an implementation plan with interface definitions (Go code), tasks in dependency order, and acceptance criteria per task.

If the architect hits a design question outside their core expertise (OTel span naming, OWASP mapping, RAG strategy, etc.), they use `/consult <specialist-name> <question>` to bounce it to a workspace specialist and cite the resulting `docs/consultations/` file in the plan.

### Step 2 — Researcher (if requested by architect)

`@agent-researcher` investigates each specific research question, returns structured findings with evidence. Architect refines the plan with findings.

### Step 3 — Developer-go

`@agent-developer-go` implements code AND tests per the plan using Red/Green TDD. Must pass the full pre-commit gate in `.claude/rules/go-packages.md` (`go build`, `go vet`, `go test -race`, `go mod tidy` + diff, `gofmt -l`, `golangci-lint` v2.15.0, `gosec -quiet` v2.30.0, `govulncheck` v1.9.0) before committing on the branch. CI parity: Go 1.27.3 with `GOTOOLCHAIN=local`; `make ci` mirrors the blocking CI jobs. A tool that could not run locally is reported as not run, never as green.

### Step 4 — Reviewer-security

`@agent-reviewer-security` performs thorough review against security checklists. Requires 2 consecutive clean passes with zero issues. Any issue resets the counter.

### Step 5 — Reviewer-qa

`@agent-reviewer-qa` validates every acceptance criterion with evidence. Any failure returns to developer-go.

### Step 6 — Codex adversarial review (coordinator)

Run `/codex:adversarial-review` on the branch (needs the `codex@openai-codex` plugin and a logged-in `codex` CLI — `codex login status`). Route findings back to developer-go, then through Steps 4–5 again for any code change. If Codex is unavailable, say so in the PR body: the review ran with weak (same-family) independence and is not an equivalent pass.

## Open the PR (coordinator only)

### 1. Run the pre-push checks

The "Before push" list in `.claude/rules/workflow.md` (Greptile-pattern self-review, optional `sonar verify`, hotspot triage for scaffold templates).

### 2. Push the branch

```bash
git push -u origin <branch-name>
```

Verify `git rev-list --left-right --count origin/<branch-name>...HEAD` prints `0 0` (workspace W-013).

### 3. Create the PR

```bash
gh pr create --fill --title "<Conventional commit prefix>: <sub-issue title> (LOO-NN)" --body "<PR body referencing the brief, the sub-issue ID, and acceptance criteria>"
```

The PR title should include the `LOO-NN` ID so Linear auto-links the PR to the sub-issue (auto-link works on title/body/branch name). The PR body should link to the brief at `research/briefs/<slug>.md` in the workspace and summarize what was implemented against the acceptance criteria.

### 4. Wait for CI + human review

CI runs gosec, golangci-lint, govulncheck, SonarCloud, Snyk, Trivy, CodeQL, unit + integration tests. Human reviews and merges when CI is green. On merge, Linear auto-closes the sub-issue.

## If Linear MCP fails during pre-flight

Retry twice with ~3-second backoff. On persistent failure, proceed with whatever context was loaded. Print:

```
Linear MCP unavailable. Proceeding with:
  - Brief: <path if found>
  - Sub-issue: not fetched
  - Branch name: guessed from brief slug

Consider running `/feature-status <parent-id>` later to verify Linear state matches the PR.
```

Never silently skip the pre-flight. The user should know the context is incomplete.
