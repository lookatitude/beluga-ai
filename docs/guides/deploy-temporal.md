# Guide: Deploy on Temporal

**Time:** ~30 minutes
**You will build:** a Beluga agent whose loop is durable — it survives crashes, supports day-long workflows, and uses signals for human approval.
**Prerequisites:** [Deploy on Docker guide](./deploy-docker.md), running Temporal server (locally via `temporal server start-dev`, or a cluster).

## When to use Temporal

Use Temporal mode when:

- Your workflows run longer than a single request's lifetime (minutes, hours, days).
- You need human-in-the-loop pauses (approval workflows).
- Crashes must not lose progress.
- You already run Temporal and want Beluga agents as part of your workflows.

Don't use Temporal when:
- Turns finish in seconds. The overhead isn't worth it.
- You don't have a Temporal cluster or the budget to run one.
- Simplicity is more valuable than durability.

**Status:** Temporal integration lives in `workflow/` as a provider backend. Confirm with `workflow.List()` that the Temporal backend is registered.

## Step 1 — run Temporal locally

```bash
temporal server start-dev --ui-port 8233
```

Temporal UI: http://localhost:8233.

## Step 2 — the agent with durable execution

```go
// cmd/durable-agent/main.go
package main

import (
    "context"
    "log"
    "time"

    "github.com/lookatitude/beluga-ai/agent"
    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/runtime"
    "github.com/lookatitude/beluga-ai/workflow"

    _ "github.com/lookatitude/beluga-ai/llm/providers/openai"
    _ "github.com/lookatitude/beluga-ai/workflow/backends/temporal"
)

func main() {
    ctx := context.Background()

    model, _ := llm.New("openai", llm.Config{"model": "gpt-4o"})
    a := agent.NewLLMAgent(
        agent.WithPersona(agent.Persona{Role: "researcher"}),
        agent.WithLLM(model),
        agent.WithTools(/* … */),
    )

    engine, err := workflow.NewEngine("temporal", workflow.Config{
        "host_port":  "localhost:7233",
        "namespace":  "default",
        "task_queue": "beluga-agent",
    })
    if err != nil {
        log.Fatalf("workflow engine: %v", err)
    }

    r := runtime.NewRunner(a,
        runtime.WithDurableExecution(engine),
        runtime.WithDurableActivityTimeout(5 * time.Minute),
        runtime.WithDurableWorkflowTimeout(24 * time.Hour),
        runtime.WithRESTEndpoint("/api/chat"),
    )

    log.Println("listening on :8080")
    if err := r.Serve(ctx, ":8080"); err != nil {
        log.Fatalf("serve: %v", err)
    }
}
```

## Step 3 — HITL approval workflow

Add a high-risk tool that requires human approval:

```go
// dangerous-tool.go
package main

import (
    "context"
    "github.com/lookatitude/beluga-ai/core"
    "github.com/lookatitude/beluga-ai/hitl"
    "github.com/lookatitude/beluga-ai/tool"
)

type deleteUserTool struct {
    approval hitl.Service
}

func (d *deleteUserTool) Name() string              { return "delete_user" }
func (d *deleteUserTool) Description() string       { return "Permanently delete a user account" }
func (d *deleteUserTool) InputSchema() map[string]any { return userIDSchema }
func (d *deleteUserTool) RiskLevel() hitl.Risk      { return hitl.RiskHigh }

func (d *deleteUserTool) Execute(ctx context.Context, in map[string]any) (*tool.Result, error) {
    userID, _ := in["user_id"].(string)

    // Pause the workflow and wait for approval.
    decision, err := d.approval.RequestApproval(ctx, hitl.Request{
        Action: "delete_user",
        Payload: map[string]any{"user_id": userID},
        Timeout: 24 * time.Hour,
    })
    if err != nil {
        return nil, core.Errorf(core.ErrGuardBlocked, "approval denied: %w", err)
    }
    if !decision.Approved {
        return nil, core.Errorf(core.ErrGuardBlocked, "human denied: %s", decision.Reason)
    }

    // Approved — do the delete.
    if err := deleteUser(ctx, userID); err != nil {
        return nil, core.Errorf(core.ErrToolFailed, "delete: %w", err)
    }
    return tool.TextResult("user deleted"), nil
}
```

Inside Temporal mode, `RequestApproval` translates to `workflow.WaitSignal("approval")`. The worker goes idle and uses no memory beyond the event log until the signal arrives.

## Step 4 — human approves via API

The approval service exposes an endpoint:

```bash
curl -X POST http://localhost:8080/hitl/approve \
  -H 'Content-Type: application/json' \
  -d '{"workflow_id":"chat-session-abc","approved":true,"reason":"verified identity"}'
```

This sends a Temporal signal that wakes the workflow. The agent resumes from exactly where it paused, completes the delete, and returns the response.

## Step 5 — crash recovery test

With a long-running turn:

```bash
curl -N http://localhost:8080/api/chat \
  -d '{"message":"Research every paper about RAG published in 2024"}'
```

Kill the worker mid-turn:

```bash
pkill -f durable-agent
```

Restart:

```bash
./durable-agent
```

Check the Temporal UI. The workflow is still there. When the worker comes back, it replays from the event log, reads the already-completed activities (the LLM calls and tool invocations that finished), and continues from the first unfinished activity. The client request (if it's still connected) receives the rest of the stream.

## Determinism rules (critical)

For replay to work, the workflow orchestration code must be deterministic:

- **Don't use `time.Now()`** in workflow code. Use `workflow.Now(ctx)`.
- **Don't use `rand`** in workflow code. Use `workflow.Random(ctx)`.
- **Don't use `go`** in workflow code. Use `workflow.Go(ctx, fn)`.
- **Don't make network calls** in workflow code. Wrap them in activities.

Activities are the non-deterministic escape hatch. An activity can do anything — its result is recorded in the event log and returned on replay.

Beluga's executor respects these rules when running under a durable engine. Your *tool* code runs inside activities, so it can use whatever it wants.

## Monitoring durable workflows

Temporal UI shows:

- Workflow ID (matches Beluga session ID).
- Current state (Running, Completed, Failed, Canceled).
- Event log (every activity, every signal, every retry).
- Pending activities.

Use it as a second observability surface alongside [OTel traces](../architecture/14-observability.md).

## Common mistakes

- **Non-deterministic code in orchestration.** Breaks replay. Move it into an activity.
- **Large payloads in workflow state.** Pass references (IDs, S3 keys), not the full data. The event log bloats and replay gets slow.
- **Activity timeout too short.** Long LLM calls need long timeouts. Start at 5 minutes and tune.
- **Not handling approval timeout.** A workflow can wait forever on a signal. Always set a timeout on `RequestApproval`.
- **Running Temporal in dev mode in production.** `temporal server start-dev` is for development. Use a real cluster.

## Related

- [16 — Durable Workflows](../architecture/16-durable-workflows.md) — deeper dive on determinism.
- [17 — Deployment Modes](../architecture/17-deployment-modes.md).
- [13 — Security Model](../architecture/13-security-model.md) — HITL gate details.
