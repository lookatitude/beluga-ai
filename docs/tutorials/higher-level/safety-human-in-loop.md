# Human-in-the-Loop Approval Flows

<!--
Persona: Pair Programmer Colleague
- Treat the reader as a competent peer
- Be helpful, direct, and slightly informal but professional
- Focus on getting results quickly
- Keep it concise and functional
- Provide runnable code examples
-->

## What you will build
In this tutorial, we'll insert human review steps into your AI workflows for critical decisions. You'll learn how to identify HITL trigger points, implement approval breakpoints, and build a review interface for both CLI and API-based flows.

## Learning Objectives
- ✅ Identify when to use Human-in-the-Loop (HITL)
- ✅ Implement an approval breakpoint
- ✅ Resume execution after approval
- ✅ Build a review CLI/API

## Introduction
Welcome, colleague! Autonomous agents are great, but some actions—like transferring money or deleting production data—need a human "sanity check." Let's look at how to build a robust Human-in-the-Loop system that pauses execution until a human gives the green light.

## Prerequisites

- [Orchestration Basics](../../getting-started/06-orchestration-basics.md)

## Why HITL?

Use HITL for:
- **Sensitive actions**: Sending emails, transferring money, deploying code.
- **Ambiguity**: When the model has low confidence.
- **Training**: Collecting feedback to improve the model.

## Step 1: The Approval Interface

Define how the system asks for help.
```go
type ApprovalRequest struct {
    Action string
    Payload any
}

type Approver interface {
    RequestApproval(ctx context.Context, req ApprovalRequest) (bool, error)
}
```

## Step 2: CLI Approver

A simple implementation for testing.
```go
type CLIApprover struct{}

func (c *CLIApprover) RequestApproval(ctx context.Context, req ApprovalRequest) (bool, error) {
    fmt.Printf("Agent wants to perform: %s\nPayload: %v\nAllow? (y/n): ", req.Action, req.Payload)
    
    var response string
    fmt.Scanln(&response)

    
    return response == "y", nil
}
```

## Step 3: Integrating into a Tool

Make the tool require approval before executing.
```go
type SecureTool struct \{
    inner Tool
    approver Approver
}
go
func (s *SecureTool) Execute(ctx context.Context, input any) (any, error) {
    allowed, _ := s.approver.RequestApproval(ctx, ApprovalRequest{
        Action: s.inner.Name(),
        Payload: input,
    })

    

    if !allowed {
        return "Action denied by user.", nil
    }
    
    return s.inner.Execute(ctx, input)
}
```

## Step 4: Asynchronous Approval (Web/Temporal)

For web apps, you can't block the thread (`fmt.Scanln`). You need to suspend execution.

**Using Callbacks/State:**
1. Agent reaches tool.
2. Tool saves state to DB ("PENDING_APPROVAL").
3. Tool returns "Waiting for approval".
4. User clicks "Approve" in UI.
5. System resumes agent with approved=true.

(See [Temporal Workflows](./orchestration-temporal-workflows.md) for the robust implementation of this pattern).

## Verification

1. Wrap a `DeleteFileTool` with `SecureTool`.
2. Ask the agent to delete a file.
3. Verify the prompt appears.
4. Deny it. Verify file remains.
5. Approve it. Verify file is deleted.

## Next Steps

- **[Temporal Workflows](./orchestration-temporal-workflows.md)** - Async HITL
- **[Content Moderation](./safety-content-moderation.md)** - Automated safety
