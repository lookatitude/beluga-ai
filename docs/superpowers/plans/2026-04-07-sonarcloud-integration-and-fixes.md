# SonarCloud Integration Fix & Issue Resolution — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix SonarCloud CI integration for dependabot PRs, resolve all SonarCloud issues in production and test code, and configure SonarCloud to exclude test files from string duplication checks.

**Architecture:** Four workstreams executed sequentially: (1) CI workflow fixes, (2) SonarCloud config, (3) production code string constant extraction, (4) test file quality fixes (empty functions + cognitive complexity). Each workstream is committed independently.

**Tech Stack:** GitHub Actions YAML, Go, SonarCloud configuration

**Spec:** `docs/superpowers/specs/2026-04-07-sonarcloud-integration-and-fixes-design.md`

---

## Task 1: Fix `pr.yml` — Skip SonarCloud for Dependabot PRs

**Files:**
- Modify: `.github/workflows/pr.yml`

- [ ] **Step 1: Update the `if` condition on the sonarcloud job**

In `.github/workflows/pr.yml`, change the `sonarcloud` job's `if` from:

```yaml
if: github.event.pull_request.head.repo.full_name == github.repository
```

to:

```yaml
if: |
  github.event.pull_request.head.repo.full_name == github.repository &&
  github.actor != 'dependabot[bot]'
```

This skips SonarCloud for dependabot PRs (which can't access `SONAR_TOKEN`). SonarCloud still runs on main after merge via `main.yml`.

- [ ] **Step 2: Commit**

```bash
git add .github/workflows/pr.yml
git commit -m "fix(ci): skip SonarCloud analysis for dependabot PRs

Dependabot PRs cannot access repo secrets, causing SonarCloud to
fail with 'Project not found'. Skip the job for dependabot and
rely on the main branch analysis after merge."
```

---

## Task 2: Fix `release.yml` — Move Permissions to Job Level

**Files:**
- Modify: `.github/workflows/release.yml`

- [ ] **Step 1: Remove workflow-level permissions**

In `.github/workflows/release.yml`, remove the top-level `permissions` block:

```yaml
# REMOVE this block (currently at line ~22):
permissions:
  contents: write
```

- [ ] **Step 2: Add job-level permissions to each job that needs them**

Add `permissions` to the jobs that need write access:

```yaml
  create-tag:
    name: Create and push tag (manual only)
    if: github.event_name == 'workflow_dispatch'
    runs-on: ubuntu-latest
    permissions:
      contents: write
    steps:
      # ... existing steps unchanged

  changes:
    name: Check Release-Relevant Changes
    if: github.event_name == 'push'
    runs-on: ubuntu-latest
    permissions:
      contents: read
    # ... rest unchanged

  changelog:
    name: Generate Changelog
    needs: changes
    if: needs.changes.outputs.release_needed == 'true'
    runs-on: ubuntu-latest
    permissions:
      contents: write
    # ... rest unchanged

  release:
    name: GoReleaser
    needs: [changes, changelog]
    if: needs.changes.outputs.release_needed == 'true'
    runs-on: ubuntu-latest
    permissions:
      contents: write
    # ... rest unchanged

  trigger-docs:
    name: Trigger Docs Rebuild
    needs: [changes, release]
    if: needs.changes.outputs.release_needed == 'true'
    runs-on: ubuntu-latest
    permissions:
      contents: read
      actions: write
    # ... rest unchanged
```

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/release.yml
git commit -m "fix(ci): move permissions from workflow to job level in release.yml

Resolves SonarCloud vulnerability githubactions:S8233. Each job now
declares only the permissions it needs rather than inheriting a
broad workflow-level write permission."
```

---

## Task 3: Update SonarCloud Configuration

**Files:**
- Modify: `sonar-project.properties`

- [ ] **Step 1: Add S1192 exclusion for test files**

Append to `sonar-project.properties`:

```properties
sonar.issue.ignore.multicriteria=e1
sonar.issue.ignore.multicriteria.e1.ruleKey=go:S1192
sonar.issue.ignore.multicriteria.e1.resourceKey=**/*_test.go
```

The full file should now be:

```properties
sonar.projectKey=lookatitude_beluga-ai
sonar.organization=lookatitude
sonar.projectName=Beluga AI

sonar.sources=.
sonar.exclusions=**/*_test.go,**/testutil/**,**/vendor/**,docs/**
sonar.tests=.
sonar.test.inclusions=**/*_test.go

sonar.go.coverage.reportPaths=coverage.out

sonar.cpd.exclusions=**/providers/**

sonar.issue.ignore.multicriteria=e1
sonar.issue.ignore.multicriteria.e1.ruleKey=go:S1192
sonar.issue.ignore.multicriteria.e1.resourceKey=**/*_test.go
```

- [ ] **Step 2: Commit**

```bash
git add sonar-project.properties
git commit -m "chore(sonar): exclude test files from string duplication rule

Test files intentionally repeat string literals for readability.
Suppressing go:S1192 in *_test.go eliminates ~870 false-positive
code smell reports."
```

---

## Task 4: Fix S1192 in `protocol/a2a/server.go`

**Files:**
- Modify: `protocol/a2a/server.go`

- [ ] **Step 1: Add constants at the top of the file (after imports)**

Add a const block after the import section:

```go
const (
	contentTypeHeader = "Content-Type"
	contentTypeJSON   = "application/json"
)
```

- [ ] **Step 2: Replace all occurrences**

Replace every `"Content-Type"` with `contentTypeHeader` and every `"application/json"` with `contentTypeJSON` in `w.Header().Set(...)` calls throughout the file. These appear at lines ~79, ~117, ~175, ~208.

- [ ] **Step 3: Run tests**

```bash
go test ./protocol/a2a/... -count=1
```

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add protocol/a2a/server.go
git commit -m "refactor(a2a): extract HTTP header constants in server.go"
```

---

## Task 5: Fix S1192 in `protocol/a2a/client.go`

**Files:**
- Modify: `protocol/a2a/client.go`

- [ ] **Step 1: Identify the duplicated error prefix strings**

The duplicated strings are the error format prefixes used 3+ times each:
- `"a2a/get_card: %w"` — 3 occurrences
- `"a2a/create_task: %w"` — 4 occurrences
- `"a2a/get_task: %w"` — 3 occurrences

Since these are error format strings with `%w`/`%s` verbs (not plain strings), and each occurrence has a slightly different context, the cleanest fix is to extract the operation name prefixes as constants:

```go
const (
	opGetCard    = "a2a/get_card: "
	opCreateTask = "a2a/create_task: "
	opGetTask    = "a2a/get_task: "
	opCancelTask = "a2a/cancel_task: "
	opInvoke     = "a2a/invoke: "
)
```

- [ ] **Step 2: Replace all error format strings**

Replace usages like:
```go
fmt.Errorf("a2a/get_card: %w", err)
```
with:
```go
fmt.Errorf(opGetCard+"%w", err)
```

Apply this pattern to all `fmt.Errorf` calls in the file that use these prefixes.

- [ ] **Step 3: Run tests**

```bash
go test ./protocol/a2a/... -count=1
```

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add protocol/a2a/client.go
git commit -m "refactor(a2a): extract error prefix constants in client.go"
```

---

## Task 6: Fix S1192 in `voice/stt/providers/deepgram/deepgram.go`

**Files:**
- Modify: `voice/stt/providers/deepgram/deepgram.go`

- [ ] **Step 1: Add constants**

```go
const (
	authPrefix = "Token "
	listenPath = "/listen?"
)
```

- [ ] **Step 2: Replace all occurrences**

Replace `"Token "` with `authPrefix` (lines ~65, ~110, ~139) and `"/listen?"` with `listenPath` (lines ~106, ~132).

- [ ] **Step 3: Run tests**

```bash
go test ./voice/stt/providers/deepgram/... -count=1
```

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add voice/stt/providers/deepgram/deepgram.go
git commit -m "refactor(deepgram): extract auth and path constants"
```

---

## Task 7: Fix S1192 in `agent/lats.go`

**Files:**
- Modify: `agent/lats.go`

- [ ] **Step 1: Add constant**

```go
const stepFmt = "Step %d: %s\n"
```

- [ ] **Step 2: Replace all occurrences**

Replace `"Step %d: %s\n"` with `stepFmt` at lines ~238, ~276, ~385.

- [ ] **Step 3: Run tests**

```bash
go test ./agent/... -count=1
```

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add agent/lats.go
git commit -m "refactor(agent): extract step format constant in lats.go"
```

---

## Task 8: Fix S1192 in `orchestration/chain.go`

**Files:**
- Modify: `orchestration/chain.go`

- [ ] **Step 1: Add constant**

```go
const chainStepErrFmt = "orchestration/chain: step %d: %w"
```

- [ ] **Step 2: Replace all occurrences**

Replace `"orchestration/chain: step %d: %w"` with `chainStepErrFmt` at lines ~38, ~69, ~81.

- [ ] **Step 3: Run tests**

```bash
go test ./orchestration/... -count=1
```

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add orchestration/chain.go
git commit -m "refactor(orchestration): extract error format constant in chain.go"
```

---

## Task 9: Fix S1192 in `orchestration/supervisor.go`

**Files:**
- Modify: `orchestration/supervisor.go`

- [ ] **Step 1: Add constant**

```go
const supervisorAgentErrFmt = "orchestration/supervisor: agent %q: %w"
```

- [ ] **Step 2: Replace all occurrences**

Replace `"orchestration/supervisor: agent %q: %w"` with `supervisorAgentErrFmt` at lines ~67, ~109, ~120.

- [ ] **Step 3: Run tests**

```bash
go test ./orchestration/... -count=1
```

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add orchestration/supervisor.go
git commit -m "refactor(orchestration): extract error format constant in supervisor.go"
```

---

## Task 10: Fix S1192 in `eval/metrics/cost.go`

**Files:**
- Modify: `eval/metrics/cost.go`

- [ ] **Step 1: Add constant**

```go
const missingMetaKeyFmt = "cost: missing metadata key %q"
```

- [ ] **Step 2: Replace all occurrences**

Replace `"cost: missing metadata key %q"` with `missingMetaKeyFmt` at lines ~57, ~71, ~80.

- [ ] **Step 3: Run tests**

```bash
go test ./eval/metrics/... -count=1
```

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add eval/metrics/cost.go
git commit -m "refactor(eval): extract error format constant in cost.go"
```

---

## Task 11: Fix S1192 in `workflow/providers/inngest/inngest.go`

**Files:**
- Modify: `workflow/providers/inngest/inngest.go`

- [ ] **Step 1: Add constant**

```go
const workflowPathFmt = "/v1/workflows/%s"
```

- [ ] **Step 2: Replace all occurrences**

Replace `"/v1/workflows/%s"` with `workflowPathFmt` in the `fmt.Sprintf` calls at lines ~71, ~101, ~151.

Example:
```go
// Before:
u := fmt.Sprintf("%s/v1/workflows/%s", s.baseURL, state.WorkflowID)
// After:
u := fmt.Sprintf("%s"+workflowPathFmt, s.baseURL, state.WorkflowID)
```

- [ ] **Step 3: Run tests**

```bash
go test ./workflow/providers/inngest/... -count=1
```

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add workflow/providers/inngest/inngest.go
git commit -m "refactor(inngest): extract workflow path constant"
```

---

## Task 12: Fix S1192 in `cmd/docgen/main.go`

**Files:**
- Modify: `cmd/docgen/main.go`

- [ ] **Step 1: Add constant**

```go
const goCodeFence = "```go\n"
```

- [ ] **Step 2: Replace all occurrences**

Replace the 3 occurrences of the go code fence string literal with `goCodeFence`.

- [ ] **Step 3: Run build**

```bash
go build ./cmd/docgen/...
```

Expected: BUILD OK (docgen is a CLI tool, may not have separate tests)

- [ ] **Step 4: Commit**

```bash
git add cmd/docgen/main.go
git commit -m "refactor(docgen): extract code fence constant"
```

---

## Task 13: Fix S1186 — Empty Functions in Test Files

**Files (34 empty functions across 18 files):**
- Modify: `core/runnable_test.go` (lines 150, 151, 430)
- Modify: `core/option_test.go` (line 72)
- Modify: `protocol/a2a/a2a_test.go` (line 623)
- Modify: `protocol/rest/rest_test.go` (lines 486, 493, 498, 550, 672)
- Modify: `server/sse_test.go` (lines 125, 143, 145)
- Modify: `voice/stt/providers/assemblyai/assemblyai_test.go` (lines 632, 706)
- Modify: `voice/stt/providers/gladia/gladia_test.go` (lines 411, 438, 468, 660, 1051)
- Modify: `voice/stt/providers/groq/groq_test.go` (line 211)
- Modify: `agent/bus_test.go` (line 115)
- Modify: `agent/reflection_test.go` (line 476)
- Modify: `config/watch_test.go` (lines 131, 151, 180)
- Modify: `internal/syncutil/pool_test.go` (line 84)
- Modify: `internal/testutil/helpers_test.go` (lines 16, 128)
- Modify: `orchestration/hooks_test.go` (line 159)
- Modify: `rag/retriever/mocks_test.go` (line 29)
- Modify: `rag/retriever/retriever_test.go` (line 89)
- Modify: `state/hooks_test.go` (line 200)
- Modify: `voice/pipeline_test.go` (line 225)

- [ ] **Step 1: Add explanatory comments to all empty functions**

For each empty function, read the surrounding context to understand its purpose, then add an appropriate comment inside the function body. The patterns are:

**Pattern A — Interface stub (most common):**
```go
func (m *mockThing) Method() {
	// no-op: stub satisfies InterfaceName for testing
}
```

**Pattern B — No-op callback/handler:**
```go
bus.Subscribe(func(event AgentEvent) {
	// no-op: subscription is immediately cancelled by context
})
```

**Pattern C — No-op option func:**
```go
OptionFunc(func(_ any) {
	// no-op: tests option passthrough, not behavior
})
```

Read each file at the specified line, determine which pattern applies, and add the appropriate comment. The comment must explain *why* the function is empty.

- [ ] **Step 2: Run all tests**

```bash
go test ./... -count=1 -timeout 120s 2>&1 | tail -5
```

Expected: PASS (comments don't change behavior)

- [ ] **Step 3: Commit**

```bash
git add -A '*.go'
git commit -m "refactor(tests): add explanatory comments to empty test functions

Resolves SonarCloud go:S1186 across 18 test files. Each empty
function now documents why it is intentionally empty."
```

---

## Task 14: Fix S3776 — Cognitive Complexity in `core/` Test Files

**Files:**
- Modify: `core/runnable_test.go` (lines 60, 188, 317, 462 — complexities 18, 46, 40, 25)
- Modify: `core/stream_test.go` (lines 32, 84, 415 — complexities 18, 17, 43)
- Modify: `core/option_test.go` (line 75 — complexity 16)

- [ ] **Step 1: Refactor complex test functions**

For each flagged function:
1. Read the function at the specified line
2. Identify logical subtest boundaries (different test scenarios, error paths, edge cases)
3. Extract each scenario into a `t.Run("descriptive_name", func(t *testing.T) { ... })` subtest
4. Extract repeated setup logic into `t.Helper()` functions where it reduces complexity
5. For table-driven tests, ensure assertions inside the loop body are flat (no nested if/else)

**Key principle:** Each `t.Run` creates a new function scope that resets cognitive complexity. Moving test cases into subtests naturally reduces per-function complexity.

**Example transformation for table-driven tests with nested assertions:**

Before (high complexity):
```go
func TestFoo(t *testing.T) {
    tests := []struct{ ... }{...}
    for _, tt := range tests {
        result, err := Foo(tt.input)
        if tt.wantErr {
            if err == nil {
                t.Errorf("expected error")
            }
        } else {
            if err != nil {
                t.Fatalf("unexpected error: %v", err)
            }
            if result != tt.want {
                t.Errorf("got %v, want %v", result, tt.want)
            }
        }
    }
}
```

After (lower complexity):
```go
func TestFoo(t *testing.T) {
    tests := []struct{ ... }{...}
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := Foo(tt.input)
            if tt.wantErr {
                require.Error(t, err)
                return
            }
            require.NoError(t, err)
            assert.Equal(t, tt.want, result)
        })
    }
}
```

If the project doesn't use testify, use the same pattern with standard library assertions but keep the `t.Run` wrapping.

- [ ] **Step 2: Run tests**

```bash
go test ./core/... -count=1 -v 2>&1 | tail -20
```

Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add core/runnable_test.go core/stream_test.go core/option_test.go
git commit -m "refactor(core): reduce cognitive complexity in test files

Break complex test functions into subtests using t.Run() to bring
cognitive complexity below the 15-point threshold."
```

---

## Task 15: Fix S3776 — Cognitive Complexity in `agent/` Test Files

**Files:**
- Modify: `agent/persona_test.go` (line 10 — complexity 28)
- Modify: `agent/selfdiscover_test.go` (line 117 — complexity 16)

- [ ] **Step 1: Refactor `TestPersona_ToSystemMessage` in `agent/persona_test.go`**

This is a table-driven test with 6 cases. Wrap the loop body in `t.Run(tt.name, ...)` and flatten assertions using early returns.

- [ ] **Step 2: Refactor the function at line 117 in `agent/selfdiscover_test.go`**

Complexity is 16 (threshold 15). Wrap test cases in `t.Run` subtests to bring below threshold.

- [ ] **Step 3: Run tests**

```bash
go test ./agent/... -count=1
```

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add agent/persona_test.go agent/selfdiscover_test.go
git commit -m "refactor(agent): reduce cognitive complexity in test files"
```

---

## Task 16: Fix S3776 — Cognitive Complexity in `voice/stt/` Test Files

**Files:**
- Modify: `voice/stt/providers/assemblyai/assemblyai_test.go` (lines 50, 382 — complexities 50, 104)
- Modify: `voice/stt/providers/gladia/gladia_test.go` (lines 327, 930 — complexities 89, 22)
- Modify: `voice/stt/providers/groq/groq_test.go` (line 171 — complexity 24)
- Modify: `voice/stt/providers/whisper/whisper_test.go` (line 185 — complexity 29)
- Modify: `voice/stt/providers/elevenlabs/elevenlabs_test.go` (line 190 — complexity 29)

- [ ] **Step 1: Refactor each flagged function**

These are the highest-complexity functions in the codebase (up to 104). For each:
1. Read the function
2. Identify distinct test scenarios (success path, error cases, edge cases, streaming behavior)
3. Extract each scenario into `t.Run("name", func(t *testing.T) { ... })`
4. For functions with complexity >40, also extract shared setup into helper functions

- [ ] **Step 2: Run tests**

```bash
go test ./voice/stt/... -count=1 -timeout 60s
```

Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add voice/stt/
git commit -m "refactor(voice/stt): reduce cognitive complexity in test files

Break high-complexity test functions (up to 104) into focused
subtests using t.Run()."
```

---

## Task 17: Fix S3776 — Cognitive Complexity in `voice/tts/` Test Files

**Files:**
- Modify: `voice/tts/providers/cartesia/cartesia_test.go` (line 168 — complexity 21)
- Modify: `voice/tts/providers/elevenlabs/elevenlabs_test.go` (line 177 — complexity 23)
- Modify: `voice/tts/providers/fish/fish_test.go` (line 174 — complexity 21)
- Modify: `voice/tts/providers/groq/groq_test.go` (line 162 — complexity 21)
- Modify: `voice/tts/providers/lmnt/lmnt_test.go` (line 190 — complexity 21)
- Modify: `voice/tts/providers/playht/playht_test.go` (line 167 — complexity 21)
- Modify: `voice/tts/providers/smallest/smallest_test.go` (line 153 — complexity 21)

- [ ] **Step 1: Refactor each flagged function**

These TTS provider tests likely follow a similar pattern (complexity 21 across 6 of 7 files). For each:
1. Read the function at the specified line
2. Wrap test cases in `t.Run` subtests
3. Flatten nested conditionals with early returns

- [ ] **Step 2: Run tests**

```bash
go test ./voice/tts/... -count=1 -timeout 60s
```

Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add voice/tts/
git commit -m "refactor(voice/tts): reduce cognitive complexity in test files"
```

---

## Task 18: Fix S3776 — Cognitive Complexity in Remaining Test Files

**Files:**
- Modify: `voice/processor_test.go` (line 199 — complexity 20)
- Modify: `voice/s2s/providers/gemini/gemini_test.go` (line 495 — complexity 16)
- Modify: `voice/pipeline_test.go` (if flagged)
- Modify: `rag/embedding/embedding_test.go` (line 247 — complexity 20)
- Modify: `rag/loader/providers/notion/notion_test.go` (line 44 — complexity 24)
- Modify: `rag/loader/providers/unstructured/unstructured_test.go` (line 441 — complexity 34)
- Modify: `server/handler_test.go` (line 99 — complexity 22)
- Modify: `cache/providers/inmemory/inmemory_test.go` (line 283 — complexity 20)
- Modify: `cache/semantic_test.go` (line 255 — complexity 20)

- [ ] **Step 1: Refactor each flagged function**

Same approach as previous tasks: wrap test cases in `t.Run` subtests, flatten assertions, extract helpers where needed.

- [ ] **Step 2: Run all tests to verify nothing is broken**

```bash
go test ./... -count=1 -timeout 120s 2>&1 | tail -10
```

Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add voice/ rag/ server/ cache/
git commit -m "refactor: reduce cognitive complexity in remaining test files

Fix S3776 issues in voice, rag, server, and cache test files."
```

---

## Task 19: Final Verification

- [ ] **Step 1: Run full test suite**

```bash
go test ./... -count=1 -timeout 180s
```

Expected: all PASS

- [ ] **Step 2: Run linter**

```bash
go vet ./...
```

Expected: no issues

- [ ] **Step 3: Verify build**

```bash
go build ./...
```

Expected: no errors

- [ ] **Step 4: Review the change summary**

```bash
git log --oneline origin/main..HEAD
```

Expected output should show commits for each task.
