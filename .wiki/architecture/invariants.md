# Architectural Invariants

## 1. Error Classification via ErrorCode
**Invariant:** All errors returned from provider/tool operations wrap in core.Error with a classified ErrorCode.

**Reference:** `core/errors.go:8-39`

**Rationale:** Enables programmatic retry decisions via core.IsRetryable(err); downstream consumers distinguish rate limits from auth failures without string matching.

**Violation Symptom:** catch (error) patterns that check strings ("timeout", "rate limit") instead of core.IsRetryable.

---

## 2. Retryable Code Classification
**Invariant:** Only ErrRateLimit, ErrTimeout, and ErrProviderDown are retryable. Auth, budget, guard block, and invalid input errors are non-retryable.

**Reference:** `core/errors.go:42-46` (retryableCodes map)

**Rationale:** Prevents futile retries (e.g., auth failure always fails, budget exhaustion needs human intervention).

**Violation Symptom:** Middleware retrying ErrAuth or ErrGuardBlocked; unbounded retry loops on these codes.

---

## 3. Middleware Application Order
**Invariant:** Middleware applied via reverse iteration; outermost middleware (first in slice) executes first.

**Reference:** `tool/middleware.go:11-22`

**Rationale:** Preserves intuitive left-to-right reading order: ApplyMiddleware(tool, timeout, retry) applies retry inside timeout.

**Violation Symptom:** Passing mws[0] to innermost, causing retry to wrap timeout (opposite intent).

---

## 4. Hooks Are Optional and Nil-Safe
**Invariant:** All Hooks struct fields (OnStart, OnEnd, OnError) are optional; ComposeHooks guarantees nil checks before invocation.

**Reference:** `tool/hooks.go:9-44`

**Rationale:** Zero allocation for consumers not using hooks; safe composition without defensive coding everywhere.

**Violation Symptom:** Direct hook invocation without nil check; panic on unset hook.

---

## 5. Registry Registration Before main()
**Invariant:** All providers and tools are registered via init() before main() executes; lookup always succeeds for registered names.

**Reference:** `llm/registry.go:19-27`, `llm/providers/anthropic/anthropic.go:19-23`

**Rationale:** Deterministic startup; no race conditions between registration and lookup.

**Violation Symptom:** Dynamic provider registration post-main; data races between Register and Get.

---

## 6. Stream Respects Backpressure
**Invariant:** Stream producer closes channel on context cancellation or early yield() return; never blocks caller indefinitely.

**Reference:** `core/stream.go:73-90`

**Rationale:** Prevents resource leaks (goroutines waiting on closed channels); respects cancellation semantics.

**Violation Symptom:** Leaked goroutines from streams that ignore yield() return false; hung calls on early break.

---

## 7. Context Cancellation Stops Retries
**Invariant:** WithRetry middleware checks ctx.Done() after each attempt; cancelled context halts retry loop immediately.

**Reference:** `tool/middleware_test.go:164-188`

**Rationale:** Respects deadline semantics; prevents retries on already-dead contexts.

**Violation Symptom:** Retries continuing after context cancellation; wasted attempts and delayed error return.

---

## 8. GenAI Attributes Use Standard Prefixes
**Invariant:** All OTel span attributes conform to OpenTelemetry GenAI v1.37+ semantics; all custom keys use gen_ai.* prefix.

**Reference:** `o11y/tracer.go:15-47`

**Rationale:** Observability tools and dashboards rely on standard attribute names; non-standard names are invisible.

**Violation Symptom:** Custom attribute keys like "agent" or "tokens" instead of "gen_ai.agent.name" and "gen_ai.usage.input_tokens".

---

## 9. Guard Pipeline Runs All Stages
**Invariant:** All three guard stages (InspectInput, InspectOutput, InspectTool) run in order; first Block decision halts processing.

**Reference:** `guard/guard.go:1-52`

**Rationale:** Complete security coverage; skipping Tool stage leaves tool-specific attacks undefended.

**Violation Symptom:** Guards skipping InspectTool; execution of blocked tools.

---

## 10. Test Subtests Isolate Failures
**Invariant:** All test scenarios use t.Run() subtests; each test case runs independently with isolated state.

**Reference:** `tool/tool_test.go:11-39`, `tool/middleware_test.go:92-118`

**Rationale:** Parallel test execution; clear failure attribution to specific scenarios.

**Violation Symptom:** Monolithic test functions; one failure masks others; sequential execution only.

