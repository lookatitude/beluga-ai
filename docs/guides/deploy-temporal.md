# Guide: Deploy on Temporal

**Time:** ~30 minutes
**You will build:** an approval workflow (research, wait for a human, then act) on Beluga's workflow API, run it on the in-process executor, and see how it maps onto the Temporal executor.
**Prerequisites:** [Deploy on Docker guide](./deploy-docker.md). For the Temporal part, a running Temporal server (`temporal server start-dev`, or a cluster).

## When to use Temporal

Use a Temporal-backed executor when:

- Your workflows run longer than a single request's lifetime (minutes, hours, days).
- You need human-in-the-loop pauses (approval workflows).
- Crashes must not lose progress.
- You already run Temporal and want Beluga agents as part of your workflows.

Don't use Temporal when:
- Turns finish in seconds. The overhead isn't worth it.
- You don't have a Temporal cluster or the budget to run one.
- Simplicity is more valuable than durability.

> **Status.** `workflow/providers/temporal` implements `workflow.DurableExecutor` on a Temporal client: `Execute` starts a Temporal workflow, and `Signal`, `Query` and `Cancel` are forwarded. It does not yet ship a worker: the workflow wrapper it starts is unexported, so you cannot register it with a Temporal worker from your code, and `workflow.WithActivityTimeout` is not applied (activities use a fixed 5-minute start-to-close timeout). Until that lands, develop workflows on the in-process executor below. The in-process executor keeps state in memory and does not survive a restart.

## Step 1 — the workflow

A Beluga workflow is a `workflow.WorkflowFunc`: a function of a `workflow.WorkflowContext`. Non-deterministic work (LLM calls, tool calls, deletes) goes in activities via `ctx.ExecuteActivity`; waiting for a human is `ctx.ReceiveSignal`.

```go
// main.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/lookatitude/beluga-ai/v2/agent"
	"github.com/lookatitude/beluga-ai/v2/config"
	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/llm"
	"github.com/lookatitude/beluga-ai/v2/workflow"

	_ "github.com/lookatitude/beluga-ai/v2/llm/providers/openai"
)

const approvalSignal = "approval"

// Approval is the payload a human sends to resume the workflow.
type Approval struct {
	Approved bool   `json:"approved"`
	Reason   string `json:"reason"`
}

// deleteUser is the irreversible action, run as an activity.
func deleteUser(ctx context.Context, input any) (any, error) {
	userID, _ := input.(string)
	// Call your user service here.
	return "deleted " + userID, nil
}

// deleteUserWorkflow researches the account, waits for approval, then deletes.
func deleteUserWorkflow(researcher agent.Agent) workflow.WorkflowFunc {
	research := workflow.LLMActivity(func(ctx context.Context, prompt string) (string, error) {
		return researcher.Invoke(ctx, prompt)
	})

	return func(ctx workflow.WorkflowContext, input any) (any, error) {
		userID, _ := input.(string)

		summary, err := ctx.ExecuteActivity(research,
			"Summarise the recent account activity for user "+userID,
			workflow.WithActivityTimeout(5*time.Minute))
		if err != nil {
			return nil, err
		}

		// Pause until a human answers on the "approval" signal.
		var decision *Approval
		for payload, err := range ctx.ReceiveSignal(approvalSignal) {
			if err != nil {
				return nil, err
			}
			if a, ok := payload.(Approval); ok {
				decision = &a
				break
			}
		}
		if decision == nil || !decision.Approved {
			return nil, core.Errorf(core.ErrGuardBlocked, "delete_user for %s was not approved", userID)
		}

		if _, err := ctx.ExecuteActivity(deleteUser, userID); err != nil {
			return nil, core.Errorf(core.ErrToolFailed, "delete_user: %w", err)
		}
		return fmt.Sprintf("user %s deleted (%s). Research: %v", userID, decision.Reason, summary), nil
	}
}

func main() {
	ctx := context.Background()

	model, err := llm.New("openai", config.ProviderConfig{
		Provider: "openai",
		APIKey:   os.Getenv("OPENAI_API_KEY"),
		Model:    "gpt-4o",
	})
	if err != nil {
		log.Fatal(err)
	}
	researcher := agent.New("researcher",
		agent.WithPersona(agent.Persona{Role: "account researcher"}),
		agent.WithLLM(model),
	)

	// Step 2: run it on the in-process executor.
	executor := workflow.NewExecutor()
	handle, err := executor.Execute(ctx, deleteUserWorkflow(researcher), workflow.WorkflowOptions{
		ID:      "delete-user-42",
		Input:   "42",
		Timeout: 24 * time.Hour,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Step 3: a human approves through an HTTP endpoint, which sends the signal.
	http.HandleFunc("POST /approve", func(w http.ResponseWriter, r *http.Request) {
		var a Approval
		if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		err := executor.Signal(r.Context(), handle.ID(), workflow.Signal{
			Name:    approvalSignal,
			Payload: a,
			SentAt:  time.Now(),
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	})
	go func() {
		srv := &http.Server{Addr: ":8080", ReadHeaderTimeout: 10 * time.Second}
		log.Fatal(srv.ListenAndServe())
	}()

	result, err := handle.Result(ctx)
	if err != nil {
		log.Fatalf("workflow: %v", err)
	}
	log.Println(result)
}
```

## Step 2 — run it

```bash
export OPENAI_API_KEY=sk-...
go run .
```

The workflow runs the research activity, then waits on the `approval` signal.

## Step 3 — approve

```bash
curl -X POST http://localhost:8080/approve \
  -H 'Content-Type: application/json' \
  -d '{"approved":true,"reason":"verified identity"}'
```

`executor.Signal` delivers the payload to `ctx.ReceiveSignal("approval")`, the workflow runs the delete activity, and `handle.Result` returns.

For approvals routed through Beluga's HITL manager instead of a raw signal, wrap a `hitl.Manager` with `workflow.HumanActivity(mgr)` and pass it a `hitl.InteractionRequest`; a reviewer answers with `mgr.Respond(ctx, requestID, hitl.InteractionResponse{Decision: hitl.DecisionApprove})`.

## Step 4 — the Temporal executor

The workflow function does not change. Only the executor does:

```go
import (
	"go.temporal.io/sdk/client"

	"github.com/lookatitude/beluga-ai/v2/workflow/providers/temporal"
)

c, err := client.Dial(client.Options{HostPort: "localhost:7233", Namespace: "default"})
if err != nil {
	log.Fatal(err)
}
defer c.Close()

executor, err := temporal.NewExecutor(temporal.Config{
	Client:         c,
	TaskQueue:      "beluga-agent",
	DefaultTimeout: 24 * time.Hour,
})
```

Through the registry, the same executor is `workflow.New("temporal", workflow.Config{Extra: map[string]any{"client": c, "task_queue": "beluga-agent"}})`.

See the status note at the top: with the current provider the workflow is started on Temporal, but no worker in your process can pick it up yet. Signal payloads that cross Temporal are also serialised, so a workflow that must run on both executors should accept the decoded form (for example a `map[string]any`) as well as the typed `Approval`.

## Determinism rules

For replay on a durable engine to work, the workflow function must be deterministic:

- **Don't read the clock** (`time.Now()`) or call `time.Sleep` in workflow code. Use `ctx.Sleep(d)`; take timestamps inside activities.
- **Don't use `rand`** in workflow code. Generate random values in an activity.
- **Don't start goroutines** in workflow code.
- **Don't make network calls** in workflow code. Wrap them in activities.

Activities are the non-deterministic escape hatch: an activity can do anything, and a durable engine records its result.

## Monitoring durable workflows

Temporal UI shows:

- Workflow ID (the `WorkflowOptions.ID` you passed to `Execute`).
- Current state (Running, Completed, Failed, Canceled).
- Event log (every activity, every signal, every retry).
- Pending activities.

Use it as a second observability surface alongside [OTel traces](../architecture/14-observability.md).

## Common mistakes

- **Non-deterministic code in orchestration.** Breaks replay. Move it into an activity.
- **Large payloads in workflow state.** Pass references (IDs, S3 keys), not the full data. The event log bloats and replay gets slow.
- **Activity timeout too short.** Long LLM calls need long timeouts. Start at 5 minutes and tune.
- **Not handling approval timeout.** A workflow can wait forever on a signal. Set `WorkflowOptions.Timeout`, or a `Timeout` on the `hitl.InteractionRequest`.
- **Running Temporal in dev mode in production.** `temporal server start-dev` is for development. Use a real cluster.

## Related

- [16 — Durable Workflows](../architecture/16-durable-workflows.md) — deeper dive on determinism.
- [17 — Deployment Modes](../architecture/17-deployment-modes.md).
- [13 — Security Model](../architecture/13-security-model.md) — HITL gate details.
