---
title: Safety & Guards
description: Implement input validation, output filtering, PII redaction, and human-in-the-loop approval workflows.
---

The `guard` and `hitl` packages provide a comprehensive safety layer for AI applications. Guards validate content at three pipeline stages, while human-in-the-loop (HITL) enables confidence-based approval workflows for high-risk actions.

## Three-Stage Guard Pipeline

Guards validate content at three points in the agent lifecycle:

```
User Input → [Input Guards] → LLM → [Output Guards] → Response
                                ↓
                          Tool Call → [Tool Guards] → Execute
```

| Stage | Validates | Purpose |
|-------|-----------|---------|
| **Input** | User messages | Block prompt injection, validate input |
| **Output** | Model responses | Redact PII, filter harmful content |
| **Tool** | Tool arguments | Validate before execution |

## The Guard Interface

```go
type Guard interface {
	Name() string
	Validate(ctx context.Context, input GuardInput) (GuardResult, error)
}
```

`GuardResult` indicates whether content is allowed, optionally provides a modified version, and explains blocking reasons:

```go
type GuardResult struct {
	Allowed   bool   // True if content passes validation
	Reason    string // Why it was blocked or modified
	Modified  string // Optional sanitized version
	GuardName string // Which guard produced this result
}
```

## Building a Guard Pipeline

```go
import "github.com/lookatitude/beluga-ai/guard"

pipeline := guard.NewPipeline(
	guard.Input(
		guard.NewPromptInjectionDetector(),
		guard.NewContentFilter(),
	),
	guard.Output(
		guard.NewPIIRedactor(guard.DefaultPIIPatterns...),
	),
	guard.Tool(
		guard.NewToolValidator(),
	),
)

// Validate user input
result, err := pipeline.ValidateInput(ctx, "Tell me about security")
if err != nil {
	log.Fatal(err)
}
if !result.Allowed {
	fmt.Printf("Blocked: %s (by %s)\n", result.Reason, result.GuardName)
	return
}

// Validate model output before sending to user
result, err = pipeline.ValidateOutput(ctx, modelResponse)
if result.Modified != "" {
	modelResponse = result.Modified // Use sanitized version
}
```

## Built-in Guards

### Prompt Injection Detection

Detects attempts to override system instructions:

```go
injectionGuard := guard.NewPromptInjectionDetector(
	guard.WithInjectionThreshold(0.7),
)

result, err := injectionGuard.Validate(ctx, guard.GuardInput{
	Content: userMessage,
	Role:    "input",
})
if !result.Allowed {
	// Prompt injection detected
}
```

### PII Redaction

Automatically redact personally identifiable information from model responses:

```go
piiGuard := guard.NewPIIRedactor(
	guard.PIIPatternEmail,
	guard.PIIPatternPhone,
	guard.PIIPatternCreditCard,
	guard.PIIPatternSSN,
)

result, err := piiGuard.Validate(ctx, guard.GuardInput{
	Content: "Contact john@example.com at 555-123-4567",
	Role:    "output",
})

fmt.Println(result.Modified)
// Output: "Contact [EMAIL REDACTED] at [PHONE REDACTED]"
```

### Content Moderation

Filter harmful or inappropriate content:

```go
contentGuard := guard.NewContentFilter(
	guard.WithCategories("hate", "violence", "self-harm"),
	guard.WithThreshold(0.8),
)
```

### Spotlighting

Isolate untrusted input to prevent indirect prompt injection:

```go
spotlight := guard.NewSpotlighter(guard.SpotlightConfig{
	Delimiter: "<<<UNTRUSTED>>>",
})
```

## Custom Guards

Implement the `Guard` interface for domain-specific validation:

```go
type ComplianceGuard struct {
	bannedTopics []string
}

func (g *ComplianceGuard) Name() string { return "compliance" }

func (g *ComplianceGuard) Validate(ctx context.Context, input guard.GuardInput) (guard.GuardResult, error) {
	for _, topic := range g.bannedTopics {
		if strings.Contains(strings.ToLower(input.Content), topic) {
			return guard.GuardResult{
				Allowed:   false,
				Reason:    fmt.Sprintf("Content discusses banned topic: %s", topic),
				GuardName: g.Name(),
			}, nil
		}
	}
	return guard.GuardResult{Allowed: true, GuardName: g.Name()}, nil
}

// Use in pipeline
pipeline := guard.NewPipeline(
	guard.Output(&ComplianceGuard{
		bannedTopics: []string{"competitor pricing", "internal roadmap"},
	}),
)
```

## Human-in-the-Loop (HITL)

The `hitl` package manages approval workflows where human judgment is required:

### Setting Up a Manager

```go
import "github.com/lookatitude/beluga-ai/hitl"

mgr := hitl.NewManager(
	hitl.WithTimeout(30 * time.Second),
	hitl.WithNotifier(hitl.NewLogNotifier(slog.Default())),
)
```

### Defining Approval Policies

Policies determine which actions need human approval:

```go
// Auto-approve read-only operations with high confidence
mgr.AddPolicy(hitl.ApprovalPolicy{
	Name:          "read-only-auto",
	ToolPattern:   "get_*",
	MinConfidence: 0.5,
	MaxRiskLevel:  hitl.RiskReadOnly,
})

// Auto-approve data modifications with very high confidence
mgr.AddPolicy(hitl.ApprovalPolicy{
	Name:          "write-auto",
	ToolPattern:   "update_*",
	MinConfidence: 0.9,
	MaxRiskLevel:  hitl.RiskDataModification,
})

// Always require approval for destructive operations
mgr.AddPolicy(hitl.ApprovalPolicy{
	Name:            "delete-manual",
	ToolPattern:     "delete_*",
	RequireExplicit: true,
})
```

Policies are evaluated in order — the first matching policy wins.

### Risk Levels

| Level | Value | Description |
|-------|-------|-------------|
| `RiskReadOnly` | `"read_only"` | Read-only operations, minimal risk |
| `RiskDataModification` | `"data_modification"` | Modifies data, moderate risk |
| `RiskIrreversible` | `"irreversible"` | Cannot be undone, highest risk |

### Checking Auto-Approval

```go
autoApproved, err := mgr.ShouldApprove(ctx,
	"get_user_profile", // Tool name
	0.95,               // Model confidence
	hitl.RiskReadOnly,  // Risk level
)

if autoApproved {
	// Execute directly
} else {
	// Request human approval
}
```

### Requesting Human Interaction

```go
resp, err := mgr.RequestInteraction(ctx, hitl.InteractionRequest{
	Type:        hitl.TypeApproval,
	ToolName:    "delete_account",
	Description: "Delete user account #12345",
	Input:       map[string]any{"user_id": "12345"},
	RiskLevel:   hitl.RiskIrreversible,
	Confidence:  0.85,
	Timeout:     60 * time.Second,
})

if err != nil {
	log.Fatal(err)
}

switch resp.Decision {
case hitl.DecisionApprove:
	// Proceed with deletion
case hitl.DecisionReject:
	// Cancel the operation
	fmt.Printf("Rejected: %s\n", resp.Feedback)
case hitl.DecisionModify:
	// Use modified inputs
	fmt.Printf("Modified: %v\n", resp.Modified)
}
```

### Interaction Types

| Type | Purpose |
|------|---------|
| `TypeApproval` | Yes/no/modify decision on an action |
| `TypeFeedback` | Request feedback on a result |
| `TypeInput` | Request additional information |
| `TypeAnnotation` | Request data annotation |

## Integrating Guards with Agents

```go
// Create a guarded agent pipeline
pipeline := guard.NewPipeline(
	guard.Input(guard.NewPromptInjectionDetector()),
	guard.Output(guard.NewPIIRedactor(guard.DefaultPIIPatterns...)),
)

a := agent.New("safe-assistant",
	agent.WithLLM(model),
	agent.WithHooks(agent.Hooks{
		OnStart: func(ctx context.Context, input any) error {
			result, err := pipeline.ValidateInput(ctx, input.(string))
			if err != nil {
				return err
			}
			if !result.Allowed {
				return fmt.Errorf("input blocked: %s", result.Reason)
			}
			return nil
		},
	}),
)
```

## Integrating HITL with Tool Execution

```go
type ApprovedTool struct {
	inner   tool.Tool
	manager hitl.Manager
}

func (t *ApprovedTool) Execute(ctx context.Context, input map[string]any) (*tool.Result, error) {
	autoApproved, err := t.manager.ShouldApprove(ctx, t.inner.Name(), 0.8, hitl.RiskDataModification)
	if err != nil {
		return nil, err
	}

	if !autoApproved {
		resp, err := t.manager.RequestInteraction(ctx, hitl.InteractionRequest{
			Type:        hitl.TypeApproval,
			ToolName:    t.inner.Name(),
			Description: fmt.Sprintf("Execute %s", t.inner.Name()),
			Input:       input,
			RiskLevel:   hitl.RiskDataModification,
		})
		if err != nil {
			return nil, err
		}
		if resp.Decision != hitl.DecisionApprove {
			return tool.ErrorResult(fmt.Errorf("action rejected: %s", resp.Feedback)), nil
		}
	}

	return t.inner.Execute(ctx, input)
}
```

## Next Steps

- [Building Your First Agent](/guides/first-agent/) — Agent fundamentals
- [Tools & MCP](/guides/tools-and-mcp/) — Tool system and execution
- [Monitoring & Observability](/guides/observability/) — Audit guard decisions
- [Deploying to Production](/guides/deployment/) — Production safety configuration
