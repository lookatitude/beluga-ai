# Wiki Validation Report — 2026-04-11

## 3-Step Retrieval Protocol Execution

### Step 1: Index Lookup
**Status:** ✓ PASS

- `.wiki/index.md` exists with 8 patterns routed to correct files
- All pattern links resolve: provider-registration, middleware, hooks, streaming, testing, otel-instrumentation, error-handling, security-guards
- Architecture links present: package-map, invariants
- Retrieval protocol documented in index

### Step 2: Query Execution (wiki-query.sh)
**Status:** ✓ PASS

- `wiki-query.sh` created and executable
- Successfully queries all 8 patterns
- Successfully queries architecture docs (invariants, package-map)
- Returns pattern headers and file:line references

### Step 3: Source Validation

#### Pattern Validation Results

| Pattern | File:Line Refs | Status |
|---------|---|---|
| provider-registration | 3 | ✓ All refs exist |
| middleware | 3 | ✓ All refs exist |
| hooks | 1 | ✓ Ref exists |
| streaming | 2 | ✓ All refs exist |
| testing | 3 | ✓ All refs exist |
| otel-instrumentation | 3 | ✓ All refs exist |
| error-handling | 3 | ✓ All refs exist |
| security-guards | 1 | ✓ Ref exists |

**Summary:** 19/19 file:line references validated against actual codebase

#### Architecture Files

| Document | Type | Status |
|----------|------|--------|
| invariants.md | Reference doc | ✓ 10 invariants with file:line refs |
| package-map.md | Reference doc | ✓ 7 packages documented |

---

## Discovered Patterns

### 1. Provider Registration
**Canonical:** `llm/registry.go:19-27`
**Variations:** init() pattern at `llm/providers/anthropic/anthropic.go:19-23`; instance registry at `tool/registry.go:17-35`

### 2. Middleware
**Canonical:** `tool/middleware.go:11-22`
**Variations:** WithTimeout at `:27-56`; WithRetry at `:58-85`

### 3. Hooks
**Canonical:** `tool/hooks.go:9-44`
**Source:** ComposeHooks pattern with nil-safe field checks

### 4. Streaming
**Canonical:** `core/stream.go:49-56`
**Variations:** MapStream producer at `:73-90`

### 5. Testing
**Canonical:** `tool/tool_test.go:11-39`
**Variations:** Retry tests at `tool/middleware_test.go:92-118`; context tests at `:164-188`

### 6. OTel Instrumentation
**Canonical:** `o11y/tracer.go:15-47`
**Variations:** StartSpan at `:118-121`; status mapping at `:100-107`

### 7. Error Handling
**Canonical:** `core/errors.go:8-110`
**Variations:** NewError at `:66-73`; Error method at `:77-82`

### 8. Security Guards
**Canonical:** `guard/guard.go:1-52`
**Source:** Three-stage guard pipeline interface

---

## Invariants Validated

All 10 architectural invariants documented with file:line references:

1. Error Classification via ErrorCode → `core/errors.go:8-39`
2. Retryable Code Classification → `core/errors.go:42-46`
3. Middleware Application Order → `tool/middleware.go:11-22`
4. Hooks Are Optional and Nil-Safe → `tool/hooks.go:9-44`
5. Registry Registration Before main() → `llm/registry.go:19-27`
6. Stream Respects Backpressure → `core/stream.go:73-90`
7. Context Cancellation Stops Retries → `tool/middleware_test.go:164-188`
8. GenAI Attributes Use Standard Prefixes → `o11y/tracer.go:15-47`
9. Guard Pipeline Runs All Stages → `guard/guard.go:1-52`
10. Test Subtests Isolate Failures → `tool/tool_test.go:11-39`

---

## Wiki System Status

✓ Index functional (8 patterns + 2 architecture docs)
✓ Pattern files created (`.wiki/patterns/*.md`)
✓ Architecture docs created (`.wiki/architecture/*.md`)
✓ Retrieval tool functional (`wiki-query.sh`)
✓ All file:line references validated
✓ Log entry created (`.wiki/log.md`)
✓ Scan artifact archived (`raw/research/wiki-scan-2026-04-11.md`)

---

## Conclusion

The wiki system is **fully operational** with all patterns discoverable and source-validated through the 3-step retrieval protocol.

**Validation Date:** 2026-04-11
**Validated By:** wiki-query.sh automated protocol
**References:** 19/19 file:line citations verified against live codebase
