# DOC-21: Human-in-the-Loop

**Audience:** Operators, compliance engineers, and developers integrating approval gates into agent workflows.
**Prerequisites:** [03 — Extensibility Patterns](./03-extensibility-patterns.md), [16 — Durable Workflows](./16-durable-workflows.md).
**Related:** [13 — Security Model](./13-security-model.md), [16 — Durable Workflows](./16-durable-workflows.md).

## Overview

Automated agents make decisions at machine speed, but not all decisions should run unattended. High-risk tool calls — deleting records, sending emails, executing commands — warrant human review before execution. Low-confidence planner outputs benefit from human correction before they cascade into wasted work. Regulatory environments often require an audit trail of human sign-off on specific actions. The `hitl` package provides the infrastructure for all three cases.

`hitl` operates as a pause-and-wait gate inside an agent's tool-call or workflow step. When the agent reaches a decision point gated by a HITL check, execution blocks until a human responds — or a configurable timeout elapses. Approval policies let you fine-tune this: safe, high-confidence, read-only actions can be auto-approved without involving a human at all, while irreversible or uncertain actions always escalate.

The package sits in Layer 3 (Capability) of the 7-layer architecture, depends only on `core`, `schema`, and `o11y`, and follows the standard four-ring extension model: interface → registry → hooks → middleware. It integrates with `workflow/` via `HumanActivity`, so approval gates survive process restarts when backed by a durable workflow engine.

## Interface

The `Manager` interface is defined in `hitl/hitl.go:136-155`:

```go
type Manager interface {
    RequestInteraction(ctx context.Context, req InteractionRequest) (*InteractionResponse, error)
    AddPolicy(policy ApprovalPolicy) error
    ShouldApprove(ctx context.Context, toolName string, confidence float64, risk RiskLevel) (bool, error)
    Respond(ctx context.Context, requestID string, resp InteractionResponse) error
}
```

- `RequestInteraction` — sends a pending request, fires the `OnRequest` hook, evaluates auto-approval policies, optionally calls the configured `Notifier`, then blocks until `Respond` is called, the per-request `Timeout` elapses, or the context is cancelled.
- `AddPolicy` — registers an `ApprovalPolicy` (evaluated first-match). Takes a glob `ToolPattern`, a `MinConfidence` threshold, a `MaxRiskLevel` ceiling, and an optional `RequireExplicit` flag.
- `ShouldApprove` — pure policy evaluation. Returns `true` if the named tool at the given confidence and risk level satisfies a registered policy without human involvement.
- `Respond` — delivers a human decision to a waiting `RequestInteraction` call. Safe to call from a separate goroutine or process (via `workflow.SignalChannel`).

### Key types

```go
// hitl/hitl.go:66-94
type InteractionRequest struct {
    ID          string
    Type        InteractionType  // TypeApproval | TypeFeedback | TypeInput | TypeAnnotation
    ToolName    string
    Description string
    Input       map[string]any
    RiskLevel   RiskLevel        // RiskReadOnly | RiskDataModification | RiskIrreversible
    Confidence  float64          // 0.0–1.0, set by the calling agent/planner
    Timeout     time.Duration    // zero = use manager default
    Metadata    map[string]any
}

// hitl/hitl.go:96-112
type InteractionResponse struct {
    RequestID string
    Decision  Decision         // DecisionApprove | DecisionReject | DecisionModify
    Feedback  string
    Modified  map[string]any   // populated when Decision == DecisionModify
    Metadata  map[string]any
}

// hitl/hitl.go:114-134
type ApprovalPolicy struct {
    Name            string
    ToolPattern     string    // glob, e.g. "get_*", "delete_*", "*"
    MinConfidence   float64
    MaxRiskLevel    RiskLevel
    RequireExplicit bool
}
```

## Registry

The registry follows the standard Beluga pattern (`hitl/hitl.go:173-223`):

| Function | Signature | Notes |
|---|---|---|
| `Register` | `func(name string, f Factory)` | Panics on empty name, nil factory, or duplicate |
| `New` | `func(name string, cfg Config) (Manager, error)` | Returns `core.ErrNotFound` if name is unknown |
| `List` | `func() []string` | Returns sorted names of registered factories |

The built-in `DefaultManager` registers itself under the name `"default"` in `hitl/manager.go:284-293`:

```go
func init() {
    Register("default", func(cfg Config) (Manager, error) {
        opts := []ManagerOption{WithTimeout(cfg.DefaultTimeout)}
        if cfg.Notifier != nil {
            opts = append(opts, WithNotifier(cfg.Notifier))
        }
        return NewManager(opts...), nil
    })
}
```

To construct a manager via the registry:

```go
mgr, err := hitl.New("default", hitl.Config{
    DefaultTimeout: 10 * time.Minute,
    Notifier:       hitl.NewLogNotifier(slog.Default()),
})
```

## Confidence-based approval thresholds

`ShouldApprove` evaluates policies in registration order; the first matching `ToolPattern` wins (`hitl/manager.go:101-144`). The evaluation logic:

1. Match `toolName` against `ToolPattern` using `path.Match` glob semantics.
2. If `RequireExplicit` is set, return `false` — always escalate regardless of confidence.
3. If `confidence < MinConfidence`, return `false`.
4. If the action's `RiskLevel` exceeds `MaxRiskLevel` in the ordered scale `read_only < data_modification < irreversible`, return `false`.
5. If all checks pass, return `true` — auto-approve.
6. If no policy matches, default to `false` — require human approval.

```
RiskReadOnly (0) < RiskDataModification (1) < RiskIrreversible (2)
```

This ordering is defined in `hitl/hitl.go:57-63`.

Example policy setup:

```go
// Auto-approve read-only tools at any confidence.
err := mgr.AddPolicy(hitl.ApprovalPolicy{
    Name:          "read-only-auto",
    ToolPattern:   "get_*",
    MinConfidence: 0.0,
    MaxRiskLevel:  hitl.RiskReadOnly,
})

// Require explicit approval for all delete operations.
err = mgr.AddPolicy(hitl.ApprovalPolicy{
    Name:            "delete-always-explicit",
    ToolPattern:     "delete_*",
    RequireExplicit: true,
})
```

## Notification channels

The `Notifier` interface (`hitl/notifier.go:14-17`) decouples request delivery from the approval mechanism:

```go
type Notifier interface {
    Notify(ctx context.Context, req InteractionRequest) error
}
```

Two built-in implementations ship in `hitl/notifier.go`:

| Type | Constructor | Behaviour |
|---|---|---|
| `LogNotifier` | `NewLogNotifier(logger *slog.Logger)` | Logs at `slog.Warn` with request fields. Uses `slog.Default()` when `logger` is nil. |
| `WebhookNotifier` | `NewWebhookNotifier(url string)` | HTTP POST with JSON body. `NewWebhookNotifierWithClient` accepts a custom `*http.Client`. Returns `core.ErrProviderDown` on non-2xx responses. |

Notification failure is non-fatal. `DefaultManager.RequestInteraction` logs the error via `OnError` and continues waiting for a `Respond` call (`hitl/manager.go:251-260`).

Implement the `Notifier` interface to send Slack messages, PagerDuty alerts, email, or any other notification channel — then pass it to `WithNotifier`.

## Integration with workflow/ for pause-and-resume

Long-running agents use `workflow.HumanActivity` (`workflow/activity.go:42-54`) to wrap a HITL check as a durable workflow step:

```go
humanStep := workflow.HumanActivity(mgr)
```

This `ActivityFunc` accepts an `hitl.InteractionRequest` as its input, calls `mgr.RequestInteraction`, and returns `*hitl.InteractionResponse` as its output. Because it runs inside a workflow activity, the execution engine can checkpoint state before the call and resume on a fresh process after `Respond` is received.

External systems deliver the human decision via `workflow.SignalChannel`:

```go
// workflow/signal.go:14-23
type SignalChannel interface {
    Send(ctx context.Context, workflowID string, signal Signal) error
    Receive(ctx context.Context, workflowID string, signalName string) (*Signal, error)
}
```

`InMemorySignalChannel` ships for testing and single-instance deployments. For multi-process deployments, implement `SignalChannel` backed by Redis pub/sub, Temporal signals, or a database-backed queue.

## Middleware and hooks

### Middleware

`Middleware` is `func(Manager) Manager` (`hitl/middleware.go:8`). `ApplyMiddleware` wraps in outside-in order — the first middleware in the slice executes first:

```go
mgr = hitl.ApplyMiddleware(mgr,
    hitl.WithTracing(),
    hitl.WithHooks(auditHooks),
)
```

### WithTracing

`WithTracing()` (`hitl/tracing.go:17-21`) wraps every `Manager` method with an OTel span named `hitl.<op>`. Span attributes follow the GenAI semantic conventions:

| Operation | Span name | Key attributes |
|---|---|---|
| `RequestInteraction` | `hitl.request_interaction` | `hitl.request.type`, `hitl.request.tool`, `hitl.request.risk_level`, `hitl.request.confidence`, `hitl.response.decision` |
| `AddPolicy` | `hitl.add_policy` | `hitl.policy.name`, `hitl.policy.pattern` |
| `ShouldApprove` | `hitl.should_approve` | `hitl.tool`, `hitl.confidence`, `hitl.risk_level`, `hitl.auto_approved` |
| `Respond` | `hitl.respond` | `hitl.request_id`, `hitl.response.decision` |

### WithHooks

`WithHooks(h Hooks)` (`hitl/middleware.go:20-24`) injects lifecycle callbacks as a `Manager` wrapper. Hook fields (`hitl/hooks.go:11-29`):

| Field | Signature | When it fires |
|---|---|---|
| `OnRequest` | `func(ctx, req) error` | Before auto-approval check. Returning an error aborts the request. |
| `OnApprove` | `func(ctx, req, resp)` | After auto-approval or when a human approves. |
| `OnReject` | `func(ctx, req, resp)` | When a human rejects. |
| `OnTimeout` | `func(ctx, req)` | When the request times out. |
| `OnError` | `func(ctx, err) error` | On any error. Returning nil suppresses it. |

Compose multiple hook sets with `ComposeHooks` (`hitl/hooks.go:35-54`). For `OnRequest` and `OnError`, the first non-nil error short-circuits further hooks.

## End-to-end example

```go
package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/lookatitude/beluga-ai/hitl"
)

func main() {
	ctx := context.Background()

	// Build and configure the manager.
	mgr := hitl.NewManager(
		hitl.WithTimeout(5*time.Minute),
		hitl.WithNotifier(hitl.NewLogNotifier(slog.Default())),
	)

	// Auto-approve read-only lookups; always require review for deletes.
	if err := mgr.AddPolicy(hitl.ApprovalPolicy{
		Name:          "read-only-auto",
		ToolPattern:   "get_*",
		MinConfidence: 0.0,
		MaxRiskLevel:  hitl.RiskReadOnly,
	}); err != nil {
		slog.Error("add policy", "err", err)
		return
	}
	if err := mgr.AddPolicy(hitl.ApprovalPolicy{
		Name:            "delete-always-explicit",
		ToolPattern:     "delete_*",
		RequireExplicit: true,
	}); err != nil {
		slog.Error("add policy", "err", err)
		return
	}

	// Wrap with tracing and an audit hook.
	wrapped := hitl.ApplyMiddleware(mgr,
		hitl.WithTracing(),
		hitl.WithHooks(hitl.Hooks{
			OnApprove: func(_ context.Context, req hitl.InteractionRequest, _ hitl.InteractionResponse) {
				slog.Info("approved", "tool", req.ToolName, "id", req.ID)
			},
			OnReject: func(_ context.Context, req hitl.InteractionRequest, _ hitl.InteractionResponse) {
				slog.Warn("rejected", "tool", req.ToolName, "id", req.ID)
			},
		}),
	)

	// Simulate the agent requesting approval before a destructive tool call.
	req := hitl.InteractionRequest{
		ID:          "req-001",
		Type:        hitl.TypeApproval,
		ToolName:    "delete_user",
		Description: "Permanently delete user account ID 42",
		RiskLevel:   hitl.RiskIrreversible,
		Confidence:  0.95,
	}

	// In a real agent this runs in its own goroutine; the reviewer calls Respond
	// from an API handler or CLI command.
	go func() {
		time.Sleep(100 * time.Millisecond)
		if err := wrapped.Respond(ctx, req.ID, hitl.InteractionResponse{
			Decision: hitl.DecisionApprove,
			Feedback: "Reviewed and confirmed.",
		}); err != nil {
			slog.Error("respond", "err", err)
		}
	}()

	resp, err := wrapped.RequestInteraction(ctx, req)
	if err != nil {
		slog.Error("interaction failed", "err", err)
		return
	}
	if resp.Decision != hitl.DecisionApprove {
		slog.Warn("action rejected by reviewer", "feedback", resp.Feedback)
		return
	}
	slog.Info("proceeding with delete_user", "feedback", resp.Feedback)
}
```

## Common mistakes

**Blocking the agent goroutine indefinitely.** `RequestInteraction` blocks until `Respond` is called or the timeout fires. Always set a `Timeout` on the `InteractionRequest` or a `DefaultTimeout` on the manager. A zero timeout means the call can hang forever if the reviewer never responds.

**Calling `Respond` before `RequestInteraction` registers the pending entry.** The pending channel is created inside `RequestInteraction` (`hitl/manager.go:245-248`). A `Respond` call that arrives before the channel exists returns `core.ErrNotFound`. Use the `workflow.SignalChannel` pattern when the response arrives asynchronously from an external system.

**Forgetting `RequireExplicit` for genuinely dangerous operations.** A policy with a high `MinConfidence` and `MaxRiskLevel: RiskIrreversible` still auto-approves calls when confidence is above the threshold. Set `RequireExplicit: true` for actions that must never be auto-approved regardless of confidence.

**Skipping `WithTracing` in production.** HITL interactions are high-value observability events. Without tracing, diagnosing why an approval gate blocked or timed out requires log archaeology. Add `WithTracing()` as the outermost middleware wrapper.

**Ignoring the `DecisionModify` path.** When a reviewer returns `DecisionModify`, the `InteractionResponse.Modified` field contains revised input parameters. Agents that only check for `DecisionApprove`/`DecisionReject` drop this signal and execute the original (potentially incorrect) inputs.

## Related reading

- [03 — Extensibility Patterns](./03-extensibility-patterns.md) — the four-ring model this package implements.
- [13 — Security Model](./13-security-model.md) — 3-stage guard pipeline (Input → Output → Tool) that HITL gates complement.
- [16 — Durable Workflows](./16-durable-workflows.md) — `workflow.HumanActivity` and `SignalChannel` for pause-and-resume across process restarts.
- [`hitl/hitl.go`](../../hitl/hitl.go) — `Manager` interface and all core types.
- [`hitl/manager.go`](../../hitl/manager.go) — `DefaultManager` with policy evaluation logic.
- [`hitl/tracing.go`](../../hitl/tracing.go) — OTel span wrappers.
- [`workflow/activity.go`](../../workflow/activity.go) — `HumanActivity` adapter.
