---
title: "HITL Package"
description: "Human-in-the-loop: confidence-based approval, escalation policies"
---

```go
import "github.com/lookatitude/beluga-ai/hitl"
```

Package hitl provides human-in-the-loop interaction management for the
Beluga AI framework. It supports approval workflows, feedback collection,
and confidence-based auto-approval policies.

## Manager Interface

The Manager interface is the core abstraction with four methods:

- RequestInteraction sends an interaction request and blocks until a human
  responds, the request times out, or the context is cancelled.
- AddPolicy registers an ApprovalPolicy for auto-approval decisions.
- ShouldApprove checks whether an action can be auto-approved based on
  registered policies, confidence level, and risk level.
- Respond delivers a human response to a pending interaction request.

## Approval Policies

ApprovalPolicy rules determine when human approval is required. Policies
match tool names via glob patterns and evaluate confidence and risk levels.
The first matching policy wins. When no policy matches, the default is to
require human approval.

Risk levels are ordered: RiskReadOnly < RiskDataModification < RiskIrreversible.

## Interaction Types

Four interaction types are supported:

- TypeApproval requests permission to proceed with an action.
- TypeFeedback requests human feedback on a result.
- TypeInput requests additional human-provided information.
- TypeAnnotation requests human annotation of data.

## Notifiers

The Notifier interface sends notifications about pending requests. Built-in
implementations include LogNotifier (logs via slog) and WebhookNotifier
(sends HTTP POST to a webhook URL).

## Middleware and Hooks

The package supports the standard Beluga middleware and hooks patterns.
Hooks provide callbacks for OnRequest, OnApprove, OnReject, OnTimeout,
and OnError lifecycle events. Hooks can be composed via ComposeHooks and
applied as middleware via WithHooks.

## Registry

Manager implementations register via the standard Beluga registry pattern
with Register, New, and List. The built-in DefaultManager registers under
the name "default".

## Usage

```go
mgr := hitl.NewManager(
    hitl.WithTimeout(30 * time.Second),
    hitl.WithNotifier(hitl.NewLogNotifier(slog.Default())),
)
err := mgr.AddPolicy(hitl.ApprovalPolicy{
    Name:          "read-only-auto",
    ToolPattern:   "get_*",
    MinConfidence: 0.5,
    MaxRiskLevel:  hitl.RiskReadOnly,
})
if err != nil {
    log.Fatal(err)
}
resp, err := mgr.RequestInteraction(ctx, hitl.InteractionRequest{
    Type:       hitl.TypeApproval,
    ToolName:   "delete_user",
    RiskLevel:  hitl.RiskIrreversible,
    Confidence: 0.9,
})
```
