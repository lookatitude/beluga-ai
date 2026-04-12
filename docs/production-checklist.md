# Production Checklist

Before deploying a Beluga agent to production, verify every item below.
Each item links to the package or architecture doc that implements the capability.
A checked item means the feature is wired in and verified — not just imported.

---

## Observability

Beluga emits OpenTelemetry GenAI semantic-convention spans (`gen_ai.*`) at every
package boundary. Verify that your exporter is configured and that spans land in
your trace backend before going live.

- [ ] **OTel exporter configured** — `o11y/exporter.go` · [`DOC-14`](architecture/14-observability.md)
      Call `o11y.ConfigureTracer(ctx, res, exporter)` with your OTLP/Jaeger/Zipkin exporter.
      `o11y/providers/` ships Langfuse, LangSmith, Opik, and Phoenix adapters.

- [ ] **`WithTracing()` applied to every extensible package** — `o11y/tracer.go`
      Wrap each capability: `llm.ApplyMiddleware(model, llm.WithTracing())`, and
      equivalently for `tool`, `memory`, `rag/embedding`, `rag/retriever`,
      `rag/vectorstore`, `guard`, `auth`, `hitl`, `workflow`, `server`, `state`,
      `rag/splitter`, `voice/s2s`, `orchestration`, `prompt`. Missing `WithTracing()`
      means blind spots in your traces.

- [ ] **GenAI span attributes verified in trace backend** — `o11y/tracer.go:16-40`
      Confirm `gen_ai.request.model`, `gen_ai.usage.input_tokens`,
      `gen_ai.usage.output_tokens`, `gen_ai.agent.name`, and `gen_ai.system` appear
      on spans.

- [ ] **Metrics configured** — `o11y/meter.go`
      Call `o11y.ConfigureMeter(ctx, res, exporter)`. Token-usage and latency
      histograms depend on this.

- [ ] **Health endpoint enabled** — `o11y/health.go`
      Expose `/healthz` (or equivalent) using `o11y.NewHealthHandler()` so your
      orchestrator can gate traffic on readiness.

- [ ] **Audit log store configured** — `audit/registry.go`
      Register a persistent `Store` via `audit.Register("name", factory)`. The
      in-memory default loses all events on restart. See `audit/store.go` for the
      `Store` interface. Every `audit.Entry` includes `TenantID`, `AgentID`,
      `SessionID`, and an `Action` — redact `Input`/`Output` before logging
      (`audit/audit.go:37`).

---

## Resilience

Agents make external calls to LLM providers and tools. None of those calls are
unconditionally reliable. Configure every resilience layer before production traffic
hits.

- [ ] **Circuit breakers on external calls** — `resilience/circuitbreaker.go`
      Wrap each external dependency with `resilience.NewCircuitBreaker(cfg)`. States
      are Closed → Open → HalfOpen. Tune `FailureThreshold`, `ResetTimeout`, and
      `HalfOpenMaxCalls` to your SLA.

- [ ] **Rate limits set (RPM, TPM, concurrency)** — `resilience/ratelimit.go`
      Configure `ProviderLimits{RPM: …, TPM: …, MaxConcurrent: …}` per model. The
      `RateLimiter` uses a token-bucket for RPM/TPM and a semaphore for concurrency.

- [ ] **Retry policy with jitter on LLM calls** — `resilience/retry.go`
      Use `resilience.DefaultRetryPolicy()` as the baseline (3 attempts, 500 ms
      initial backoff, jitter enabled). Pass `RetryPolicy.RetryableErrors` to
      restrict retries to `core.IsRetryable(err) == true` errors.

- [ ] **Hedged requests for latency-sensitive paths** — `resilience/hedge.go`
      Call `resilience.Hedge(ctx, primary, secondary, delay)` on p99-critical paths.
      The primary fires immediately; the secondary is started only if the primary
      does not return within `delay`. The faster result wins.

---

## Safety and Guards

All agent input, output, and tool calls must pass through the 3-stage guard pipeline.
Skipping any stage leaves an attack surface.

- [ ] **Input guard enabled** — `guard/pipeline.go`
      Build a `guard.NewPipeline(guard.Input(guards...))` and call
      `p.ValidateInput(ctx, content)` before passing user messages to the LLM.
      Minimum guards: `guard.NewInjectionDetector()` (prompt injection,
      `guard/injection.go`) and PII detection (`guard/pii.go`).

- [ ] **Output guard enabled** — `guard/pipeline.go`
      Call `p.ValidateOutput(ctx, content)` on every LLM response before
      returning to the user. Minimum guards: PII redaction (`guard/pii.go`),
      content moderation (`guard/content.go`).

- [ ] **Tool guard enabled** — `guard/pipeline.go`
      Call `p.ValidateTool(ctx, toolName, input)` before executing any tool
      call produced by the LLM. Minimum guards: capability check
      (`guard/providers/`) and schema validation.

- [ ] **Spotlighting configured for untrusted content** — `guard/spotlighting.go`
      Wrap all untrusted external content (documents, search results, user
      uploads) with `guard.NewSpotlighting(delimiter)` before embedding in
      prompts. The default delimiter is `^^^`. This isolates untrusted data
      from trusted instructions.

- [ ] **Agentic guard pipeline configured for multi-agent deployments** — `guard/agentic/pipeline.go`
      In multi-step or multi-agent deployments: configure exfiltration detection
      (`guard/agentic/exfiltration_guard.go`), cascade-abuse detection
      (`guard/agentic/cascade_guard.go`), and tool-overreach detection
      (`guard/agentic/tool_guard.go`). Wire with `agentic.NewPipeline(...)`.

- [ ] **Memory poisoning detection enabled** — `guard/memory/guard.go`
      Wrap memory writes with `memory.NewMemoryGuard(memory.WithDetectors(...))`.
      The `MemoryGuard` runs `AnomalyDetector` instances and flags content above
      the configured score threshold (`guard/memory/detector.go`).

- [ ] **Graceful degradation policy set** — `guard/degradation/degrader.go`
      Configure `degradation.NewRuntimeDegrader(monitor, policy, opts...)` to
      enforce autonomy level restrictions (`AutonomyLevel`) when the
      `SecurityMonitor` detects elevated risk. Define your `DegradationPolicy`
      to map severity to tool allowlists and capability restrictions.

---

## Authentication and Authorization

Every API surface must enforce authentication and every tool call must be
capability-checked.

- [ ] **Auth policy configured** — `auth/auth.go`
      Choose at least one policy implementation. `auth/rbac.go` for role-based
      control (`NewRBACPolicy`); `auth/abac.go` for attribute-based control with
      `Condition` predicates; `auth/opa.go` for policy-as-code via OPA
      (`NewOPAPolicy(name, endpoint)`). Compose multiple with `auth/composite.go`.

- [ ] **`WithTracing()` applied to the auth middleware** — `auth/tracing.go`
      Wrap your auth enforcer: `auth.ApplyMiddleware(enforcer, auth.WithTracing())`.
      Missing auth spans make security incidents impossible to trace.

- [ ] **Tenant isolation via context** — `core/tenant.go`
      Every request must carry a tenant ID: `core.WithTenant(ctx, id)`. Downstream
      packages call `core.GetTenant(ctx)` to scope data access. Without this, a
      multi-tenant deployment leaks data between tenants.

- [ ] **Capability-based tool permissions** — `auth/auth.go` + `guard/agentic/tool_guard.go`
      Restrict which tools each tenant or role can invoke. Grant permissions
      explicitly — deny by default. Wire the capability check into the tool guard
      stage of the guard pipeline.

- [ ] **JWT/OAuth2 credential handling** — `auth/credential/`
      Configure `auth/credential/` for your identity provider. Validate signatures
      and expiry before trusting any claim.

---

## Durability

LLM agents are stateful, long-running processes. A restart without durable state
loses all in-progress work.

- [ ] **Workflow backend configured** — `workflow/providers/`
      Select a durable backend: Temporal (`workflow/providers/temporal/`), Inngest,
      NATS, Dapr, or Kafka. The in-memory backend (`workflow/providers/inmemory/`)
      is for development only — it does not survive process restarts.
      See [`DOC-16`](architecture/16-durable-workflows.md).

- [ ] **Activity retry policies set** — `workflow/activity.go`
      Every `ActivityFunc` must have an explicit retry policy. Use
      `workflow.LLMActivity(invoker)` and `workflow.ToolActivity(executor)` to
      wrap LLM and tool calls as retryable activities.

- [ ] **HITL approval gates on high-risk tools** — `hitl/manager.go`
      Register a `hitl.Manager` and use `hitl.HumanActivity(manager, ...)` in
      `workflow/activity.go` to block execution until a human approves. Configure
      `hitl.TypeApproval` for actions with irreversible side effects (writes,
      sends, deployments). Set a timeout — unanswered requests must not block
      indefinitely.

- [ ] **HITL middleware applied** — `hitl/middleware.go`
      Wrap agents that perform high-risk operations:
      `hitl.ApplyMiddleware(agent, hitl.WithApproval(manager, policy))`.

---

## Cost and Budget

LLM token spend can spike without warning. Budget enforcement prevents runaway costs.

- [ ] **Cost tracker configured** — `cost/tracker.go`
      Instantiate a `cost.Tracker` and register it. The tracker records per-request
      token counts and maps them to USD cost using model pricing from
      `cost/registry.go`.

- [ ] **Budget limits set** — `cost/budget.go`
      Define a `cost.Budget` with `MaxTokensPerHour` and `MaxCostPerDay`. Set
      `AlertThreshold` (e.g. `0.8` for 80%) and choose `BudgetActionThrottle` or
      `BudgetActionReject` as the enforcement action.

- [ ] **Token usage visible in traces** — `o11y/tracer.go`
      Verify `gen_ai.usage.input_tokens` and `gen_ai.usage.output_tokens` appear
      on LLM spans. The cost tracker depends on these attributes; missing values
      produce incorrect budget calculations.

---

## Configuration

- [ ] **Hot-reload configured for environment-specific settings** — `config/watch.go`
      Use `config.NewFileWatcher(cfg)` or implement `config.Watcher` for your
      config store (Consul, etcd, AWS AppConfig). Register a callback that applies
      new values without a process restart.

- [ ] **Prompt cache enabled for repeated system prompts** — `cache/`
      Wrap your LLM with `cache.NewSemanticCache(embedder, threshold, store)`
      (`cache/semantic.go`) or use provider-native prompt caching for Anthropic and
      OpenAI. Cache hits reduce latency and token spend on shared system prompts.

- [ ] **Declarative agent configuration validated** — `config/declarative/`
      If you deploy from Agentfile (`config/agentfile/`), validate the schema
      at startup. Unknown fields should error, not silently default.

---

## Evaluation and Continuous Quality

Evaluation is not a one-time activity — regression in faithfulness, hallucination
rate, or latency is a production incident.

- [ ] **Eval runner wired into CI** — `eval/runner.go`
      Configure an `eval.EvalRunner` with `eval.WithMetrics(...)` and
      `eval.WithHooks(hooks)`. Run it on a representative dataset
      (`eval/dataset.go`) on every merge that touches agent logic.

- [ ] **Key metrics tracked** — `eval/metrics/`
      Include at minimum:
      - Faithfulness (`eval/metrics/faithfulness.go`)
      - Hallucination rate (`eval/metrics/hallucination.go`)
      - Relevance (`eval/metrics/relevance.go`)
      - Latency p50/p99 (`eval/metrics/latency.go`)
      - Cost per query (`eval/metrics/cost.go`)
      - Toxicity for user-facing outputs (`eval/metrics/toxicity.go`)

- [ ] **Red-team evaluation run** — `eval/redteam/`
      Run adversarial probes against your guard pipeline before initial launch and
      after any guard configuration change.

---

## Tool Execution Safety

- [ ] **Sandboxed tool execution configured** — `tool/sandbox/`
      Wrap dangerous tools with `sandbox.NewSandboxedTool(inner, pool)`.
      The `sandbox.Pool` manages isolated processes (`tool/sandbox/pool.go`).
      Configure process limits in `tool/sandbox/types.go`.

- [ ] **Tool middleware applied (tracing, retry, rate-limit)** — `tool/middleware.go`
      Apply `tool.ApplyMiddleware(t, tool.WithTracing(), ...)` to every registered
      tool. Bare tool executions produce no spans and no retry on failure.

---

## Pre-Launch Verification Commands

Run these locally and in CI before every production deployment:

```bash
# Compilation and static analysis
go build ./...
go vet ./...

# Race detector
go test -race ./...

# Module hygiene
go mod tidy && git diff --exit-code go.mod go.sum

# Formatting
gofmt -l . | grep -v ".claude/worktrees"

# Linters and security scanners
golangci-lint run ./...
gosec -quiet ./...
govulncheck ./...
```

All must pass with zero new findings in files you changed. Pre-existing findings
in files you did not change must be documented in the commit message.

---

## Related Reading

- [`architecture/14-observability.md`](architecture/14-observability.md) — OTel GenAI conventions, span naming, exporter setup.
- [`architecture/15-resilience.md`](architecture/15-resilience.md) — circuit breakers, retry, hedging, rate limits.
- [`architecture/16-durable-workflows.md`](architecture/16-durable-workflows.md) — workflow backends, activity retry, HITL.
- [`architecture/17-deployment-modes.md`](architecture/17-deployment-modes.md) — Docker, Kubernetes, Temporal, edge.
- [`reference/providers.md`](reference/providers.md) — full provider catalog per capability.
- [`.wiki/patterns/security-guards.md`](../.wiki/patterns/security-guards.md) — guard pipeline canonical implementation.
- [`.wiki/architecture/invariants.md`](../.wiki/architecture/invariants.md) — the ten architectural invariants.
