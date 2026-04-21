# SonarCloud Hotspot Triage Reference

Scope: recurring SonarCloud security hotspots that fire on scaffolded templates, golden fixtures, CI workflows, and other "intentional by design" code paths in the framework repo. Each row lists the rule, the pattern, and the canonical "Reviewed — Safe" comment to attach via `sonar api post /api/hotspots/change_status` after a CI scan.

**Why this file exists.** Some SonarCloud rules fire on code whose pattern is correct-by-design — the rule is asking a reasonable question, but the answer is "yes, we reviewed this, it's safe". The answer does not change between scans, but the `new_security_hotspots_reviewed` gate requires 100% review on **new** code, so every time we add or edit a scaffolded template we re-incur the review cost. This table makes each re-review a one-command operation using `sonar api`.

**Prerequisite:** `sonar auth status` shows `[✓ Connected]`. If not, run `sonar auth login`.

## How to use

1. After the SonarCloud scan fails on the gate condition `new_security_hotspots_reviewed < 100`, list the hotspots:
   ```bash
   sonar api get '/api/hotspots/search?projectKey=lookatitude_beluga-ai&pullRequest=<N>&status=TO_REVIEW&ps=500' | jq '.hotspots[] | {key, rule: .ruleKey, file: (.component|split(":")[-1]), line, msg: .message}'
   ```
2. For each hotspot whose rule + file match a row below, copy the canonical comment and issue:
   ```bash
   sonar api post /api/hotspots/change_status -d '{"hotspot":"<KEY>","status":"REVIEWED","resolution":"SAFE","comment":"<COMMENT FROM TABLE>"}'
   ```
3. For hotspots whose rule does **not** match any row below, treat them as a genuine new concern: fix the code, OR review + justify inline in the PR, OR extend this table with a new entry after the justification is merged.

## Canonical template-inherent patterns

### `docker:S6470` — glob COPY / recursive COPY in Dockerfile

**Files this fires on (as of 2026-04-21):** `cmd/beluga/scaffold/templates/basic/Dockerfile.tmpl` + every golden fixture mirroring it (`cmd/beluga/scaffold/testdata/golden/**/Dockerfile`).

**Why safe:** Both lines are canonical Go multi-stage build patterns. `COPY go.mod go.sum* ./` uses a glob that only matches sibling `go.sum` variants in the project root. `COPY . .` runs in a fresh build stage whose WORKDIR is populated only by `go.mod` + `go.sum` (from the prior step) plus the user's source tree; the runtime stage is `distroless/static`, which contains only the compiled binary.

**Canonical comment (glob line):**
> Scaffolded Dockerfile template. `COPY go.mod go.sum* ./` is the canonical Go multi-stage-build pattern; glob only matches sibling go.sum variants in the project root. Distroless runtime contains only the compiled binary.

**Canonical comment (recursive line):**
> Scaffolded Dockerfile template. `COPY . .` runs in a fresh build stage populated only by go.mod/go.sum in the prior step. `.dockerignore` is the user-level filter; distroless runtime contains only the compiled binary.

**Canonical comment (golden-fixture mirror, both lines):**
> Golden fixture mirroring the Dockerfile template. Reviewed on the template; this file is deterministic generated output.

### `githubactions:S7637` — mutable action tag in scaffolded `ci.yml`

**Files this fires on:** `cmd/beluga/scaffold/templates/basic/.github/workflows/ci.yml.tmpl` and its golden fixture.

**Why safe:** The scaffolded `ci.yml` uses `@v6` / `@v4` / `@v5` to match the repo-wide framework convention across `release.yml`, `_ci-checks.yml`, `_security-checks.yml`, and every other workflow. Hardening scaffolded user projects to SHA-pins is a downstream choice the user can make; forcing it in the scaffolder would diverge from the rest of the ecosystem.

**Canonical comment:**
> Scaffolded ci.yml template uses `@v6` / `@v4` / `@v5` matching repo-wide framework convention across all workflows. SHA-pinning in scaffolded user projects is a downstream hardening choice.

### `githubactions:S6506` — HTTPS redirect risk on `curl` in workflow

**Files this fires on:** any workflow `run:` block that downloads a binary via `curl` or `wget`.

**Fix preferred over Safe-review.** Add `--proto '=https' --tlsv1.2` to every such `curl`. This refuses any redirect to plain HTTP and any TLS below 1.2. The rule then stops firing on the next scan. **Only mark Safe if the fix is impossible** (e.g., the downstream endpoint requires an intermediate HTTP redirect, which is itself a concerning pattern and deserves a separate discussion).

**Canonical Safe comment (use only if the curl fix isn't possible):**
> curl uses `--proto '=https' --tlsv1.2`, enforcing HTTPS and TLS 1.2+; no redirect to plain HTTP is possible. <explain why additional hardening doesn't apply here>

## Hotspots that should NEVER be marked Safe wholesale

- `go:S2245` / `go:S2068` / `go:S2077` — these flag real secrets, weak random, and SQL injection. Fix in code or pull the code off the PR.
- `go:S6819` / `go:S6931` — real concurrency and I/O issues. Fix in code.
- Any rule firing on **non-scaffolded** production code. Template-inherent review is a concession to the scaffolder's code-generation contract; the framework's own production code should not need Safe-review blanket statements.

## Adding a new row

When a new scaffolder template lands and trips a recurring hotspot pattern:

1. Propose the row in the PR that introduced the template, including: (a) the rule key, (b) the file pattern, (c) a specific "why safe" paragraph, (d) the canonical comment.
2. After the PR merges, update this file with the row.
3. Subsequent PRs that touch the same template can copy-paste the canonical comment for faster `sonar api` triage.

## Related

- Workspace `.wiki/corrections.md` W-014 — discovery of the `sonar` CLI as the correct mutation path for hotspot state transitions.
- `.claude/rules/workflow.md` — the "Before push" gate that references this file for scaffolded-template diffs.
