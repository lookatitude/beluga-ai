---
title: Multi-Agent Customer Support
description: Build an intelligent customer support system with specialized agents, handoffs, and human escalation using Beluga AI.
---

Customer support teams face diverse inquiries requiring different expertise — billing disputes, technical troubleshooting, account management, and general questions. A single monolithic chatbot struggles with this breadth. Multi-agent systems solve this by routing each inquiry to a specialized agent that has the right tools and context for the job, with seamless handoffs between agents when issues cross domains.

## Solution Architecture

Beluga AI's agent system models handoffs as tools. When a triage agent determines that an inquiry requires billing expertise, it calls a `transfer_to_billing` tool, which transparently transfers the conversation to the billing agent. Each specialized agent has its own persona, tools, and LLM configuration optimized for its domain.

```
                    ┌─────────────────┐
                    │  Triage Agent   │
                    │  (Classify &    │
                    │   Route)        │
                    └───────┬─────────┘
                            │
              ┌─────────────┼─────────────┐
              ▼             ▼             ▼
     ┌────────────┐ ┌────────────┐ ┌────────────┐
     │  Billing   │ │ Technical  │ │  General   │
     │  Agent     │ │  Agent     │ │  Agent     │
     │            │ │            │ │            │
     │ Tools:     │ │ Tools:     │ │ Tools:     │
     │ - Refund   │ │ - Logs     │ │ - FAQ      │
     │ - Invoice  │ │ - Restart  │ │ - Account  │
     │ - Payment  │ │ - Diagnose │ │ - Transfer │
     └────────────┘ └────────────┘ └────────────┘
              │             │             │
              └─────────────┼─────────────┘
                            ▼
                    ┌─────────────────┐
                    │  HITL Manager   │
                    │  (Escalate to   │
                    │   Human Agent)  │
                    └─────────────────┘
```

## Building Specialized Agents

Each agent has a focused persona, a curated set of tools, and domain-specific instructions.

```go
package main

import (
    "context"
    "fmt"

    "github.com/lookatitude/beluga-ai/agent"
    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/tool"
    "github.com/lookatitude/beluga-ai/schema"

    _ "github.com/lookatitude/beluga-ai/llm/providers/openai"
)

// createBillingAgent builds an agent specialized in billing inquiries.
func createBillingAgent(ctx context.Context) (agent.Agent, error) {
    model, err := llm.New("openai", nil)
    if err != nil {
        return nil, fmt.Errorf("create model: %w", err)
    }

    // Domain-specific tools
    refundTool := tool.NewFuncTool[RefundInput](
        "process_refund",
        "Process a refund for a customer order",
        func(ctx context.Context, input RefundInput) (*tool.Result, error) {
            // Call billing system API
            result, err := billingAPI.ProcessRefund(ctx, input.OrderID, input.Amount)
            if err != nil {
                return tool.ErrorResult(err), nil
            }
            return tool.TextResult(fmt.Sprintf("Refund of $%.2f processed for order %s", input.Amount, input.OrderID)), nil
        },
    )

    invoiceTool := tool.NewFuncTool[InvoiceInput](
        "lookup_invoice",
        "Look up invoice details by invoice or customer ID",
        func(ctx context.Context, input InvoiceInput) (*tool.Result, error) {
            invoice, err := billingAPI.GetInvoice(ctx, input.InvoiceID)
            if err != nil {
                return tool.ErrorResult(err), nil
            }
            return tool.TextResult(fmt.Sprintf("Invoice %s: $%.2f, status: %s", invoice.ID, invoice.Amount, invoice.Status)), nil
        },
    )

    billingAgent, err := agent.New(
        agent.WithID("billing-agent"),
        agent.WithPersona(agent.Persona{
            Role:      "Billing Support Specialist",
            Goal:      "Resolve billing inquiries accurately and efficiently",
            Backstory: "You handle refunds, invoice questions, and payment issues. " +
                "Always verify the customer's identity before processing financial transactions.",
        }),
        agent.WithModel(model),
        agent.WithTools(refundTool, invoiceTool),
    )
    if err != nil {
        return nil, fmt.Errorf("create billing agent: %w", err)
    }

    return billingAgent, nil
}

type RefundInput struct {
    OrderID string  `json:"order_id" jsonschema:"description=The order ID to refund"`
    Amount  float64 `json:"amount" jsonschema:"description=Refund amount in dollars"`
}

type InvoiceInput struct {
    InvoiceID string `json:"invoice_id" jsonschema:"description=The invoice ID to look up"`
}
```

## Handoffs as Tools

Beluga AI implements agent transfers as tools. When you register child agents, Beluga automatically generates `transfer_to_{agent_id}` tools that the parent agent can call to hand off the conversation.

```go
func createTriageAgent(ctx context.Context) (agent.Agent, error) {
    model, err := llm.New("openai", nil)
    if err != nil {
        return nil, fmt.Errorf("create model: %w", err)
    }

    billingAgent, err := createBillingAgent(ctx)
    if err != nil {
        return nil, err
    }

    techAgent, err := createTechAgent(ctx)
    if err != nil {
        return nil, err
    }

    generalAgent, err := createGeneralAgent(ctx)
    if err != nil {
        return nil, err
    }

    // Triage agent with child agents — handoff tools are auto-generated
    triageAgent, err := agent.New(
        agent.WithID("triage-agent"),
        agent.WithPersona(agent.Persona{
            Role: "Customer Support Triage",
            Goal: "Classify customer inquiries and route to the right specialist",
            Backstory: "You are the first point of contact. Quickly determine " +
                "whether the issue is billing, technical, or general, then " +
                "transfer to the appropriate specialist agent.",
        }),
        agent.WithModel(model),
        agent.WithChildren(billingAgent, techAgent, generalAgent),
    )
    if err != nil {
        return nil, fmt.Errorf("create triage agent: %w", err)
    }

    return triageAgent, nil
}
```

When the triage agent calls `transfer_to_billing-agent`, the framework transparently transfers the conversation along with its full context to the billing agent.

## Streaming Agent Events

Monitor agent execution in real time using Beluga AI's streaming event system:

```go
func handleInquiry(ctx context.Context, triageAgent agent.Agent, inquiry string) error {
    for event, err := range triageAgent.Stream(ctx, inquiry) {
        if err != nil {
            return fmt.Errorf("agent error: %w", err)
        }

        switch event.Type {
        case agent.EventText:
            fmt.Print(event.Text) // Stream response text
        case agent.EventToolCall:
            fmt.Printf("[Tool: %s]\n", event.ToolCall.Name)
        case agent.EventHandoff:
            fmt.Printf("[Handoff to: %s]\n", event.AgentID)
        case agent.EventDone:
            fmt.Println("\n[Done]")
        }
    }
    return nil
}
```

## Human-in-the-Loop Escalation

Not every issue can be resolved by AI. Beluga AI's HITL (Human-in-the-Loop) system provides confidence-based escalation. When an agent's confidence is low or the action carries high risk, the system routes to a human reviewer.

```go
import (
    "github.com/lookatitude/beluga-ai/hitl"
    _ "github.com/lookatitude/beluga-ai/hitl/providers/default"
)

func setupEscalation(ctx context.Context) (hitl.Manager, error) {
    manager, err := hitl.New("default", hitl.Config{})
    if err != nil {
        return nil, fmt.Errorf("create hitl manager: %w", err)
    }

    // Auto-approve low-risk actions, require human approval for high-risk
    err = manager.AddPolicy(hitl.ApprovalPolicy{
        Name:          "refund-approval",
        ToolPattern:   "process_refund",
        MinConfidence: 0.9,
        MaxRiskLevel:  hitl.RiskDataModification,
        RequireExplicit: true,
    })
    if err != nil {
        return nil, fmt.Errorf("add policy: %w", err)
    }

    // Read-only lookups don't need approval
    err = manager.AddPolicy(hitl.ApprovalPolicy{
        Name:          "lookup-auto-approve",
        ToolPattern:   "lookup_*",
        MinConfidence: 0.5,
        MaxRiskLevel:  hitl.RiskReadOnly,
    })
    if err != nil {
        return nil, fmt.Errorf("add policy: %w", err)
    }

    return manager, nil
}
```

When a tool execution triggers HITL review, the system pauses the agent, notifies a human reviewer, and resumes once the decision is made:

```go
func escalateToHuman(ctx context.Context, manager hitl.Manager, toolName string, input map[string]any) error {
    resp, err := manager.RequestInteraction(ctx, hitl.InteractionRequest{
        Type:        hitl.TypeApproval,
        ToolName:    toolName,
        Description: "Customer is requesting a refund over $500",
        Input:       input,
        RiskLevel:   hitl.RiskDataModification,
        Confidence:  0.75,
        Timeout:     5 * time.Minute,
    })
    if err != nil {
        return fmt.Errorf("request interaction: %w", err)
    }

    if resp.Decision != hitl.DecisionApprove {
        return fmt.Errorf("refund rejected by reviewer: %s", resp.Feedback)
    }
    return nil
}
```

## Workflow Orchestration

For complex multi-step support cases, use workflow agents to coordinate a sequence of actions:

```go
import "github.com/lookatitude/beluga-ai/agent/workflow"

// Sequential workflow: verify identity → diagnose → resolve → follow up
supportWorkflow, err := workflow.NewSequential(
    agent.WithID("support-workflow"),
    agent.WithModel(model),
    workflow.WithSteps(
        identityVerifier,
        diagnosisAgent,
        resolutionAgent,
        followUpAgent,
    ),
)
```

## Production Considerations

### Observability

Each agent execution produces OpenTelemetry spans with `gen_ai.*` attributes, making it straightforward to trace a customer inquiry through triage, handoff, tool execution, and response generation:

```go
import "go.opentelemetry.io/otel/attribute"

// Spans are automatically created for each agent invocation.
// Add custom attributes for business metrics:
span.SetAttributes(
    attribute.String("support.inquiry_type", "billing"),
    attribute.String("support.customer_tier", "enterprise"),
    attribute.Bool("support.escalated", false),
)
```

### Safety Guards

Use Beluga AI's guard pipeline to screen agent inputs and outputs:

```go
import "github.com/lookatitude/beluga-ai/guard"

// Input guard: prevent prompt injection attacks
// Output guard: prevent PII leakage in responses
// Tool guard: validate tool inputs before execution
pipeline := guard.NewPipeline(
    guard.WithInputGuards(injectionGuard, toxicityGuard),
    guard.WithOutputGuards(piiGuard),
    guard.WithToolGuards(authorizationGuard),
)
```

### Authentication and Authorization

Restrict agent capabilities per customer tier using Beluga AI's auth system:

```go
import "github.com/lookatitude/beluga-ai/auth"

policy, err := auth.New("rbac", auth.Config{})

// Check if agent can execute a tool for this customer
allowed, err := policy.Authorize(ctx, customerID, auth.PermToolExec, "process_refund")
if !allowed {
    return fmt.Errorf("customer not authorized for refund processing")
}
```

### Scaling

- **Agent pooling**: Reuse agent instances across requests. Agents are stateless between invocations when memory is externalized.
- **Concurrent execution**: Multiple inquiries execute in parallel; each gets its own context and memory scope.
- **Rate limiting**: Use Beluga AI's resilience package to rate-limit expensive tool calls (billing API, external systems).
- **Load balancing**: Deploy multiple service instances behind a load balancer. Route by customer tier for SLA compliance.

## Related Resources

- [Building Your First Agent](/guides/first-agent/) for planner strategies (ReAct, Reflexion, Self-Discover)
- [Tools & MCP](/guides/tools-and-mcp/) for building custom tools
- [Safety & Guards](/guides/safety-and-guards/) for human-in-the-loop patterns
