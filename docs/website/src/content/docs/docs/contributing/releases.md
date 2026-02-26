---
title: "Release Process — Beluga AI"
description: "How Beluga AI releases are managed. Semantic versioning, automated changelog generation with git-cliff, and the CI/CD release pipeline."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI releases, semantic versioning, Go module versioning, changelog generation, git-cliff, CI/CD pipeline"
---

This page describes how Beluga AI versions are managed, how releases are created, and what happens during the release pipeline. The release process is fully automated from tag to published release — this ensures consistency and reduces the chance of human error in release artifacts.

## Versioning

Beluga AI follows [Semantic Versioning](https://semver.org/) (SemVer). SemVer is especially important for a framework with 100+ providers: users need to know whether upgrading will break their code, introduce new features, or just fix bugs:

```
MAJOR.MINOR.PATCH
```

| Component | Incremented when |
|---|---|
| **MAJOR** | Breaking changes to the public API |
| **MINOR** | New features added in a backward-compatible manner |
| **PATCH** | Backward-compatible bug fixes |

**Examples:** `v1.0.0`, `v1.2.3`, `v2.0.0`

## Release Workflow

Releases are triggered by pushing a Git tag to the `main` branch:

```bash
# Tag a release commit
git tag v1.2.0
git push origin v1.2.0
```

The tag must match the pattern **`v*.*.*`** to trigger the release pipeline.

## What Happens on Release

When a version tag is pushed, the CI pipeline automatically:

1. **Generates the changelog** — [git-cliff](https://git-cliff.org/) generates `CHANGELOG.md` from conventional commit messages since the last release.
2. **Creates the GitHub Release** — [GoReleaser](https://goreleaser.com/) creates a GitHub Release with the generated changelog as release notes.
3. **Rebuilds documentation** — The documentation website is rebuilt and deployed with the new version information.

## Changelog

The changelog is **auto-generated** using [git-cliff](https://git-cliff.org/) from conventional commit messages. This is why following the [Conventional Commits](/docs/contributing/code-style/#commit-message-format) format is important — your commit messages directly become changelog entries. A well-written commit message like `feat(llm): add streaming support for Anthropic provider` produces a clear, useful changelog entry without any manual editing.

Commits are grouped by type:

- **Features** — `feat:` commits
- **Bug Fixes** — `fix:` commits
- **Performance** — `perf:` commits
- **Documentation** — `docs:` commits
- **Other** — `refactor:`, `test:`, `chore:`, `ci:` (may be excluded from public changelog)

## Go Module Versioning

Users install specific versions of Beluga AI using the standard Go module system:

```bash
# Install the latest release
go get github.com/lookatitude/beluga-ai@latest

# Install a specific version
go get github.com/lookatitude/beluga-ai@v1.2.0

# Update to the latest patch release
go get -u github.com/lookatitude/beluga-ai
```

For major version 2 and above, the module path includes the major version suffix per Go module conventions:

```bash
go get github.com/lookatitude/beluga-ai/v2@latest
```

## Pre-Releases

Pre-release versions use standard SemVer suffixes to communicate stability expectations. This graduated rollout process allows early adopters to test new features while ensuring that `@latest` always points to a stable release:

| Suffix | Stage | Stability |
|---|---|---|
| `-alpha.N` | Alpha | Experimental, API may change significantly |
| `-beta.N` | Beta | Feature-complete, API may have minor changes |
| `-rc.N` | Release Candidate | Production-ready, final testing |

**Examples:** `v1.0.0-alpha.1`, `v1.0.0-beta.3`, `v1.0.0-rc.1`

Pre-releases are not installed by default with `@latest` — users must explicitly request them:

```bash
go get github.com/lookatitude/beluga-ai@v1.0.0-rc.1
```

## Who Can Release

Only **project maintainers** have permission to push tags and trigger releases. If you believe a release is needed, open an issue or mention it in a discussion.

## CI Pipeline Overview

The CI pipeline is structured as a progression from fast, local checks to comprehensive, infrastructure-dependent validations. Each stage gates the next, ensuring that problems are caught early:

The full CI pipeline from code change to release:

```
PR Checks                Post-Merge               Tag / Release
─────────────           ──────────────           ──────────────
│ Lint             │    │ Full test suite   │    │ git-cliff changelog │
│ Build            │ →  │ Coverage report   │ →  │ GoReleaser          │
│ Unit tests       │    │ Security scans    │    │ GitHub Release      │
│ Integration tests│    │ SonarCloud        │    │ Docs rebuild        │
│ Security scans   │    │ Auto-version tag  │    │                     │
│ SonarCloud       │    │ Docs deploy       │    │                     │
│ Greptile review  │    │                   │    │                     │
─────────────           ──────────────           ──────────────
```

1. **PR Checks** — Run on every pull request. Must all pass before merge. Includes CI, security scans, SonarCloud analysis, and Greptile AI code review.
2. **Post-Merge** — Run after merge to `main`. Generates coverage reports, runs full security scans, deploys documentation, and auto-tags the next version.
3. **Tag / Release** — Triggered when a `v*.*.*` tag is pushed. Generates changelog, creates GitHub Release, and rebuilds docs.

## Security Scanning

Every PR and push to `main` runs a comprehensive, multi-layered security pipeline:

| Scanner | What it does |
|---|---|
| **Snyk** | Dependency vulnerability scanning with severity thresholds against the Snyk vulnerability database |
| **Trivy** | Filesystem and dependency scanning for CRITICAL/HIGH vulnerabilities (SARIF upload to GitHub Security tab) |
| **govulncheck** | Go team's official vulnerability scanner — uses symbol-level reachability to only flag vulnerabilities in code paths your project actually calls |
| **gosec** | Static analysis for Go security issues (SARIF upload to GitHub Security tab) |
| **CodeQL** | Deep semantic static analysis by GitHub (runs on push, PR, and weekly schedule) |
| **Gitleaks** | Scans for accidentally committed API keys, tokens, and passwords |
| **go-licenses** | Verifies all dependencies use compatible licenses (MIT, Apache-2.0, BSD-2-Clause, BSD-3-Clause, ISC, MPL-2.0) |

Security scans also run weekly (Monday 4am UTC) to catch newly disclosed CVEs in existing dependencies.

## Code Review

| Tool | What it does |
|---|---|
| **Greptile** | AI-powered code review on every PR — provides contextual feedback based on codebase understanding (internal PRs only) |
| **SonarCloud** | Code quality, duplication detection, and maintainability analysis |
