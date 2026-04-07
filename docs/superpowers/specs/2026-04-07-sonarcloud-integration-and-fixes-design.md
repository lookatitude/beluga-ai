# SonarCloud Integration Fix & Issue Resolution

## Summary

Fix SonarCloud's GitHub PR integration for dependabot PRs, resolve the single security vulnerability, fix all production code smells and test file quality issues, and configure SonarCloud to exclude test files from the string duplication rule (`go:S1192`).

## Problem

1. **SonarCloud fails on dependabot PRs** — The `pr.yml` workflow's `if` guard (`github.event.pull_request.head.repo.full_name == github.repository`) passes for dependabot (same-repo), but dependabot can't access repo secrets. The `SONAR_TOKEN` is empty, causing the scan to fail with "Project not found."

2. **962 open SonarCloud issues** across the codebase:
   - 1 vulnerability (`githubactions:S8233`) in `release.yml`
   - ~14 critical code smells (`go:S1192`) in production code
   - 54 critical code smells (`go:S3776` — cognitive complexity) in test files
   - 34 critical code smells (`go:S1186` — empty functions) in test files
   - ~870+ critical code smells (`go:S1192`) in test files (to be handled by config exclusion)

## Design

### Workstream 1: CI Integration Fix

#### 1a. Fix `pr.yml` — Dependabot SonarCloud Handling

The `if` guard needs to also check that `SONAR_TOKEN` is actually available. Dependabot PRs are same-repo so they pass the fork check, but secrets are not exposed to dependabot.

**Fix:** Add a secret availability check to the `if` condition:

```yaml
sonarcloud:
  name: SonarCloud Analysis
  needs: ci
  runs-on: ubuntu-latest
  if: |
    github.event.pull_request.head.repo.full_name == github.repository &&
    github.actor != 'dependabot[bot]'
  steps:
    # ... existing steps
```

This explicitly skips SonarCloud for dependabot PRs. SonarCloud will still run on main pushes (via `main.yml`) after dependabot PRs are merged, so coverage is not lost.

#### 1b. Fix `release.yml` — Vulnerability `githubactions:S8233`

Move write permissions from workflow level to job level at line 22.

**Current:** Permissions declared at workflow level.
**Fix:** Move `contents: write` to the specific job that needs it.

### Workstream 2: Production Code Fixes (`go:S1192`)

Extract per-file constants for duplicated string literals in these production files:

| File | Duplicated Strings | Count |
|------|-------------------|-------|
| `workflow/providers/inngest/inngest.go` | `"Bearer "`, header literal | 2 |
| `eval/metrics/cost.go` | `"cost: missing metadata key %q"` | 1 |
| `protocol/a2a/server.go` | `"application/json"`, `"Content-Type"` | 2 |
| `protocol/a2a/client.go` | error format strings | 3 |
| `agent/lats.go` | `"Step %d: %s\n"` | 1 |
| `voice/stt/providers/deepgram/deepgram.go` | `"Token "`, `"/listen?"` | 2 |
| `orchestration/chain.go` | error format string | 1 |
| `orchestration/supervisor.go` | error format string | 1 |
| `cmd/docgen/main.go` | `` "```go\n" `` | 1 |

**Approach:** Add `const` blocks at the top of each file (or near usage) with descriptive names. Keep scope file-local (unexported).

### Workstream 3: Test File Fixes

#### 3a. Cognitive Complexity (`go:S3776`) — 54 issues

Break complex test functions into subtests using `t.Run()` where natural boundaries exist. Priority by complexity excess:

| File | Line | Complexity | Threshold |
|------|------|-----------|-----------|
| `voice/stt/providers/assemblyai/assemblyai_test.go` | 382 | 104 | 15 |
| `voice/stt/providers/gladia/gladia_test.go` | 327 | 89 | 15 |
| `voice/stt/providers/assemblyai/assemblyai_test.go` | 50 | 50 | 15 |
| `core/runnable_test.go` | 188 | 46 | 15 |
| `core/stream_test.go` | 415 | 43 | 15 |
| `core/runnable_test.go` | 317 | 40 | 15 |
| `rag/loader/providers/unstructured/unstructured_test.go` | 441 | 34 | 15 |
| `voice/stt/providers/elevenlabs/elevenlabs_test.go` | 190 | 29 | 15 |
| `voice/stt/providers/whisper/whisper_test.go` | 185 | 29 | 15 |
| `agent/persona_test.go` | 10 | 28 | 15 |
| `core/runnable_test.go` | 462 | 25 | 15 |
| `rag/loader/providers/notion/notion_test.go` | 44 | 24 | 15 |
| `voice/stt/providers/groq/groq_test.go` | 171 | 24 | 15 |
| `voice/tts/providers/elevenlabs/elevenlabs_test.go` | 177 | 23 | 15 |
| `server/handler_test.go` | 99 | 22 | 15 |
| `voice/stt/providers/gladia/gladia_test.go` | 930 | 22 | 15 |
| `voice/tts/providers/cartesia/cartesia_test.go` | 168 | 21 | 15 |
| `voice/tts/providers/fish/fish_test.go` | 174 | 21 | 15 |
| `voice/tts/providers/groq/groq_test.go` | 162 | 21 | 15 |
| `voice/tts/providers/lmnt/lmnt_test.go` | 190 | 21 | 15 |
| `voice/tts/providers/playht/playht_test.go` | 167 | 21 | 15 |
| `voice/tts/providers/smallest/smallest_test.go` | 153 | 21 | 15 |
| `rag/embedding/embedding_test.go` | 247 | 20 | 15 |
| `voice/processor_test.go` | 199 | 20 | 15 |
| `cache/providers/inmemory/inmemory_test.go` | 283 | 20 | 15 |
| `cache/semantic_test.go` | 255 | 20 | 15 |
| `core/stream_test.go` | 32 | 18 | 15 |
| `core/runnable_test.go` | 60 | 18 | 15 |
| `core/stream_test.go` | 84 | 17 | 15 |
| `voice/s2s/providers/gemini/gemini_test.go` | 495 | 16 | 15 |
| `agent/selfdiscover_test.go` | 117 | 16 | 15 |
| `core/option_test.go` | 75 | 16 | 15 |

**Strategy:** For each function:
- Identify logical test case groupings
- Extract into `t.Run("descriptive name", func(t *testing.T) { ... })` subtests
- Extract shared setup into helper functions where it reduces complexity
- For very high complexity (>40), consider extracting test helper functions

#### 3b. Empty Functions (`go:S1186`) — 34 issues

All are in test files — likely interface stub implementations or callback placeholders.

| File | Lines |
|------|-------|
| `protocol/a2a/a2a_test.go` | 623 |
| `protocol/rest/rest_test.go` | 486, 493, 498, 550, 672 |
| `server/sse_test.go` | 125, 143, 145 |
| `voice/stt/providers/assemblyai/assemblyai_test.go` | 632, 706 |
| `voice/stt/providers/gladia/gladia_test.go` | 411, 438, 468, 660, 1051 |
| `voice/stt/providers/groq/groq_test.go` | 211 |
| `agent/bus_test.go` | 115 |
| `agent/reflection_test.go` | 476 |
| `config/watch_test.go` | 131, 151, 180 |
| `core/option_test.go` | 72 |
| `core/runnable_test.go` | 150, 151, 430 |
| `internal/syncutil/pool_test.go` | 84 |
| `internal/testutil/helpers_test.go` | 16, 128 |
| `orchestration/hooks_test.go` | 159 |
| `rag/retriever/mocks_test.go` | 29 |
| `rag/retriever/retriever_test.go` | 89 |
| `state/hooks_test.go` | 200 |
| `voice/pipeline_test.go` | 225 |

**Strategy:** For each empty function, read the context to determine:
- If it's an interface stub → add `// no-op: satisfies <InterfaceName> for testing`
- If it's unused → remove it

### Workstream 4: SonarCloud Configuration

Update `sonar-project.properties` to exclude test files from the `S1192` (string duplication) rule:

```properties
# Existing config
sonar.projectKey=lookatitude_beluga-ai
sonar.organization=lookatitude
sonar.projectName=Beluga AI

sonar.sources=.
sonar.exclusions=**/*_test.go,**/testutil/**,**/vendor/**,docs/**
sonar.tests=.
sonar.test.inclusions=**/*_test.go

sonar.go.coverage.reportPaths=coverage.out

sonar.cpd.exclusions=**/providers/**

# NEW: Exclude test files from string duplication rule
sonar.issue.ignore.multicriteria=e1
sonar.issue.ignore.multicriteria.e1.ruleKey=go:S1192
sonar.issue.ignore.multicriteria.e1.resourceKey=**/*_test.go
```

## Testing

- All existing tests must continue to pass (`go test ./...`)
- After merge to main, verify SonarCloud dashboard shows reduced issue count
- Create a test dependabot-like PR or verify next dependabot PR doesn't fail SonarCloud

## Out of Scope

- Fixing `S1192` in test files (handled by SonarCloud config exclusion)
- Changing SonarCloud quality gates or other rules
- Adding new CI checks or workflows
