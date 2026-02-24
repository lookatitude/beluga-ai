---
title: Human-in-the-Loop Approval Flows
description: "Insert human review steps into AI workflows in Go — configure confidence-based approval policies, webhook notifications, and async approval with Beluga AI's hitl package."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, Go, tutorial, human-in-the-loop, approval, hitl, confidence-based, safety"
---

Autonomous agents are powerful, but some actions -- transferring money, deleting data, sending emails to customers -- require a human "sanity check." Full automation is appropriate for low-risk, read-only operations, but high-stakes or irreversible actions need human oversight. The `hitl` package provides a `Manager` interface that routes interaction requests through configurable `ApprovalPolicy` rules, auto-approving low-risk actions while escalating uncertain or dangerous operations. This approach gives you the efficiency of automation where it is safe, with the safety of human review where it is needed.

## What You Will Build

A human-in-the-loop system with policy-based auto-approval, webhook notifications, and both synchronous and asynchronous approval flows. You will configure policies for different risk levels and tool categories.

## Prerequisites

- Familiarity with the `tool` and `agent` packages
- Understanding of the guard pipeline (recommended)

## Core Concepts

### Approval Policies

Policies determine when human approval is required. Each policy matches tools by **glob pattern** (using `path.Match()` semantics) and evaluates confidence and risk levels. The glob pattern enables flexible matching: `get_*` matches all read-only tools, `delete_*` matches all destructive tools, and `*` matches everything. This pattern-based approach means you do not need to enumerate every tool individually -- you define categories and let the pattern matching handle discovery.

```go
import "github.com/lookatitude/beluga-ai/hitl"

policy := hitl.ApprovalPolicy{
    Name:          "read-only-auto",
    ToolPattern:   "get_*",          // Matches get_weather, get_user, etc.
    MinConfidence: 0.5,              // Auto-approve if confidence >= 0.5
    MaxRiskLevel:  hitl.RiskReadOnly, // Only auto-approve read-only actions
}
```

### Risk Levels

Three risk levels categorize actions by their potential impact:

| Level | Description |
|-------|-------------|
| `RiskReadOnly` | Read-only operations with minimal risk |
| `RiskDataModification` | Operations that modify data |
| `RiskIrreversible` | Operations that cannot be undone |

### Interaction Types

The manager supports multiple interaction types beyond simple approval, covering different scenarios where human input is needed:

- `TypeApproval` -- Approve or reject an action
- `TypeFeedback` -- Request human feedback on a result
- `TypeInput` -- Request additional information from a human
- `TypeAnnotation` -- Request human annotation of data

## Step 1: Create and Configure the Manager

The manager is configured with functional options (`WithTimeout`, `WithNotifier`) following Beluga AI's standard configuration pattern. The timeout prevents workflows from blocking indefinitely when a human reviewer is unavailable. The notifier sends alerts when approval is needed, ensuring reviewers are aware of pending requests.

```go
package main

import (
    "context"
    "fmt"
    "log/slog"
    "time"

    "github.com/lookatitude/beluga-ai/hitl"
)

func main() {
    ctx := context.Background()

    // Create a manager with logging and timeout.
    mgr := hitl.NewManager(
        hitl.WithTimeout(30 * time.Second),
        hitl.WithNotifier(hitl.NewLogNotifier(slog.Default())),
    )

    // Add policies.
    if err := mgr.AddPolicy(hitl.ApprovalPolicy{
        Name:          "read-only-auto",
        ToolPattern:   "get_*",
        MinConfidence: 0.5,
        MaxRiskLevel:  hitl.RiskReadOnly,
    }); err != nil {
        fmt.Printf("policy error: %v\n", err)
        return
    }

    if err := mgr.AddPolicy(hitl.ApprovalPolicy{
        Name:            "destructive-always-human",
        ToolPattern:     "delete_*",
        RequireExplicit: true, // Always requires human approval.
    }); err != nil {
        fmt.Printf("policy error: %v\n", err)
        return
    }
}
```

## Step 2: Check Auto-Approval

The `ShouldApprove` method evaluates policies to determine if human approval is needed. It uses **first-match semantics**: the first policy whose `ToolPattern` matches the tool name determines the outcome. If the tool's confidence exceeds the policy's `MinConfidence` and the risk level is within the policy's `MaxRiskLevel`, the action is auto-approved. Otherwise, it requires human review.

```go
func checkApproval(ctx context.Context, mgr *hitl.DefaultManager) {
    // Read-only tool with high confidence: auto-approved.
    approved, err := mgr.ShouldApprove(ctx, "get_weather", 0.95, hitl.RiskReadOnly)
    if err != nil {
        fmt.Printf("error: %v\n", err)
        return
    }
    fmt.Printf("get_weather auto-approved: %v\n", approved) // true

    // Destructive tool: always requires human.
    approved, err = mgr.ShouldApprove(ctx, "delete_user", 0.99, hitl.RiskIrreversible)
    if err != nil {
        fmt.Printf("error: %v\n", err)
        return
    }
    fmt.Printf("delete_user auto-approved: %v\n", approved) // false
}
```

## Step 3: Request Human Interaction

When auto-approval is not granted, submit an interaction request that blocks until a human responds or the timeout expires. The `RequestInteraction` call is synchronous from the caller's perspective -- the agent's execution pauses until a decision is received. The request includes all the context a reviewer needs: tool name, description, input data, risk level, and the model's confidence score.

```go
func requestApproval(ctx context.Context, mgr *hitl.DefaultManager) {
    // This blocks until a human responds or the timeout expires.
    resp, err := mgr.RequestInteraction(ctx, hitl.InteractionRequest{
        Type:        hitl.TypeApproval,
        ToolName:    "delete_user",
        Description: "Agent wants to delete user account #12345",
        Input:       map[string]any{"user_id": "12345"},
        RiskLevel:   hitl.RiskIrreversible,
        Confidence:  0.85,
    })
    if err != nil {
        fmt.Printf("interaction error: %v\n", err)
        return
    }

    switch resp.Decision {
    case hitl.DecisionApprove:
        fmt.Println("Action approved by human reviewer.")
    case hitl.DecisionReject:
        fmt.Printf("Action rejected: %s\n", resp.Feedback)
    case hitl.DecisionModify:
        fmt.Printf("Action modified: %v\n", resp.Modified)
    }
}
```

## Step 4: Respond to Pending Requests

Human reviewers respond to pending requests via the `Respond` method. In a web application, this would be called from an API handler -- a review dashboard displays pending requests and the reviewer clicks approve or reject. The `requestID` correlates the response to the original blocking `RequestInteraction` call, unblocking the agent.

```go
func approveFromAPI(ctx context.Context, mgr *hitl.DefaultManager, requestID string) error {
    return mgr.Respond(ctx, requestID, hitl.InteractionResponse{
        Decision: hitl.DecisionApprove,
        Feedback: "Reviewed and approved by admin.",
    })
}

func rejectFromAPI(ctx context.Context, mgr *hitl.DefaultManager, requestID string) error {
    return mgr.Respond(ctx, requestID, hitl.InteractionResponse{
        Decision: hitl.DecisionReject,
        Feedback: "User account should not be deleted without manager approval.",
    })
}
```

## Step 5: Webhook Notifications

For production deployments, use the `WebhookNotifier` to alert humans via HTTP when approval is needed. This integrates with Slack, PagerDuty, or any webhook-compatible notification system, ensuring that approval requests are not silently queued without anyone being aware.

```go
func buildProductionManager() *hitl.DefaultManager {
    return hitl.NewManager(
        hitl.WithTimeout(5 * time.Minute),
        hitl.WithNotifier(hitl.NewWebhookNotifier("https://internal.example.com/approvals")),
    )
}
```

The webhook receives a JSON payload containing the full `InteractionRequest`:

```json
{
    "ID": "hitl-42",
    "Type": "approval",
    "ToolName": "delete_user",
    "Description": "Agent wants to delete user account #12345",
    "RiskLevel": "irreversible",
    "Confidence": 0.85
}
```

## Step 6: Policy Composition

In production, you define a comprehensive policy set that covers all tool categories from most permissive (read-only, auto-approve) to most restrictive (destructive, always human). Policy evaluation uses first-match semantics, so order your policies from most specific to most general. This layered approach ensures that new tools automatically fall into the correct approval tier based on their naming convention.

```go
func configureFullPolicies(mgr *hitl.DefaultManager) error {
    policies := []hitl.ApprovalPolicy{
        {
            Name:          "read-auto",
            ToolPattern:   "get_*",
            MinConfidence: 0.5,
            MaxRiskLevel:  hitl.RiskReadOnly,
        },
        {
            Name:          "search-auto",
            ToolPattern:   "search_*",
            MinConfidence: 0.7,
            MaxRiskLevel:  hitl.RiskReadOnly,
        },
        {
            Name:          "update-high-confidence",
            ToolPattern:   "update_*",
            MinConfidence: 0.9,
            MaxRiskLevel:  hitl.RiskDataModification,
        },
        {
            Name:            "delete-always-human",
            ToolPattern:     "delete_*",
            RequireExplicit: true,
        },
        {
            Name:            "send-always-human",
            ToolPattern:     "send_*",
            RequireExplicit: true,
        },
    }

    for _, p := range policies {
        if err := mgr.AddPolicy(p); err != nil {
            return fmt.Errorf("adding policy %q: %w", p.Name, err)
        }
    }
    return nil
}
```

Policy evaluation uses first-match semantics: the first policy whose `ToolPattern` matches the tool name determines the outcome. Order your policies from most specific to most general.

## Verification

1. Configure a policy that auto-approves `get_*` tools with confidence >= 0.5.
2. Call `ShouldApprove` with `get_weather` at 0.9 confidence. Verify it returns `true`.
3. Call `ShouldApprove` with `delete_file` with `RequireExplicit`. Verify it returns `false`.
4. Submit a `RequestInteraction` and respond with `Respond`. Verify the blocking call returns.
5. Test timeout behavior by not responding within the timeout window.

## Next Steps

- [Content Moderation](/docs/tutorials/safety/content-moderation) -- Automated safety guards for input/output filtering
- [Temporal Workflows](/docs/tutorials/orchestration/temporal-workflows) -- Async HITL with durable execution for long-running approvals
