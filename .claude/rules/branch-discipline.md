# Branch Discipline

Branch-and-PR discipline for the framework repo. This file mirrors the
mandatory rules already stated in `CLAUDE.md` and `workflow.md`, and adds the
Linear-integrated branch-naming convention introduced in A1.

## Rules

1. **Never commit directly to `main`.** Every change — code, docs, tests — starts with `git checkout -b <type>/<short-desc>` where `<type>` is `fix`, `feat`, `refactor`, `docs`, or `chore`.
2. **Every branch ends with `gh pr create`.** Never merge directly to `main`. CI must be green before merge.
3. **Verify `git branch --show-current` is not `main`** before any `git commit`.
4. **Never skip hooks** (`--no-verify`, `--no-gpg-sign`, etc.) unless the user explicitly requests it. If a hook fails, investigate and fix the underlying issue.
5. **Never force-push to `main`**, and never force-push without explicit user approval on any shared branch.
6. **Never delete files, branches, or tags destructively** without explicit user approval. Measure twice, cut once.

## Linear-integrated branch naming (A1)

When work is tracked via a Linear sub-issue:

- Branch name format: `<type>/loo-NN-<short-slug>` where:
  - `<type>` is derived from Linear labels: `Feature` → `feat/`, `Bug` → `fix/`, `Improvement` → `refactor/`, otherwise `chore/` (workspace `/plan-feature` does not apply a `documentation` label — doc-only changes flow through to `chore/`)
  - `loo-NN` is the Linear sub-issue ID (lowercase, matches auto-link behavior)
  - `<short-slug>` is a 3-5 word kebab-case summary of the sub-issue title
- Example: sub-issue `LOO-43 "Implement cmd/beluga/ cobra scaffold"` labeled `Feature, layer:framework` → `feat/loo-43-beluga-cli-foundation`
- Linear's GitHub integration auto-links PRs and auto-closes sub-issues when the `loo-NN` token appears anywhere in the branch name, PR title, or PR body. The type prefix is workspace convention, not a Linear requirement.

For work not tracked in Linear (small fixes, exploration), use the pre-A1 format: `<type>/<short-desc>`.

## Why

- CI (tests, linters, security scanners) is the last line of defense against regressions. Direct `main` pushes skip that line.
- Every change becomes a reviewable, revertable unit with its own history.
- Pre-existing findings in files you did NOT change do not block your commit, but MUST be documented in the commit message so reviewers know to ignore or fix them separately.
