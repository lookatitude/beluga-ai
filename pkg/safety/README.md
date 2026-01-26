# Safety Package

The safety package provides content safety validation and ethical AI checks for the Beluga AI Framework. It enables opt-in safety validation for agents and LLM interactions.

## Overview

- **SafetyChecker**: Pattern-based content validation
- **SafetyMiddleware**: Wraps agents with automatic safety checks
- **OTEL Metrics**: Full observability integration

## Installation

```go
import "github.com/lookatitude/beluga-ai/pkg/safety"
```

## Quick Start

### Basic Usage

```go
// Create a safety checker
checker := safety.NewSafetyChecker()

// Check content
result, err := checker.CheckContent(ctx, "Hello, how can I help?")
if err != nil {
    log.Fatal(err)
}

if result.Safe {
    fmt.Println("Content is safe")
} else {
    fmt.Printf("Unsafe content detected: %d issues, risk score: %.2f\n",
        len(result.Issues), result.RiskScore)
}
```

### Agent Middleware

```go
// Wrap an agent with safety checks
agent := agents.NewReActAgent(llm, tools)
safeAgent := safety.NewSafetyMiddleware(agent)

// Plan with automatic safety validation
action, finish, err := safeAgent.Plan(ctx, steps, inputs)
```

## Configuration

```go
cfg := &safety.Config{
    Enabled:        true,
    RiskThreshold:  0.3,
    ToxicityWeight: 0.4,
    BiasWeight:     0.2,
    HarmfulWeight:  0.5,
    EnableMetrics:  true,
}

// Or use functional options
cfg := safety.NewTestConfig(
    safety.WithRiskThreshold(0.5),
    safety.WithToxicityWeight(0.3),
)
```

### YAML Configuration

```yaml
safety:
  enabled: true
  risk_threshold: 0.3
  toxicity_weight: 0.4
  bias_weight: 0.2
  harmful_weight: 0.5
  enable_metrics: true
  custom_patterns:
    toxicity_patterns:
      - "(?i)\\bcustom_bad_word\\b"
```

## Risk Scoring

Content is evaluated against three pattern categories:

| Category | Default Weight | Description |
|----------|---------------|-------------|
| Toxicity | 0.4 | Hate speech, profanity, slurs |
| Bias | 0.2 | Generalizations, stereotypes |
| Harmful | 0.5 | Dangerous content, illegal activities |

Content is considered **safe** if `risk_score < risk_threshold` (default 0.3).

## Safety Result

```go
type SafetyResult struct {
    Timestamp time.Time     // When check was performed
    Issues    []SafetyIssue // Detected issues
    RiskScore float64       // Cumulative risk score
    Safe      bool          // Whether content passed validation
}

type SafetyIssue struct {
    Type        string // "toxicity", "bias", "harmful"
    Description string // Human-readable description
    Severity    string // "low", "medium", "high"
}
```

## Metrics

When `EnableMetrics` is true, the following OTEL metrics are recorded:

| Metric | Type | Description |
|--------|------|-------------|
| `safety.checks.total` | Counter | Total safety checks performed |
| `safety.issues.total` | Counter | Total issues detected |
| `safety.unsafe.total` | Counter | Total unsafe content detections |
| `safety.errors.total` | Counter | Total errors during checks |
| `safety.check.duration` | Histogram | Check duration in seconds |
| `safety.risk.score` | Histogram | Distribution of risk scores |

## Testing

```go
// Use mock for testing
mock := safety.NewMockSafetyChecker(
    safety.WithMockSafe(false),
    safety.WithMockRiskScore(0.8),
    safety.WithMockIssues([]iface.SafetyIssue{
        safety.MakeToxicityIssue(),
    }),
)

result, err := mock.CheckContent(ctx, "test")
// result.Safe == false
// result.RiskScore == 0.8
```

## Error Handling

```go
var (
    ErrUnsafe            // Content failed safety validation
    ErrUnsafeContent     // Content contains unsafe material
    ErrSafetyCheckFailed // Safety check process failed
    ErrHighRiskContent   // Content has high safety risk
)
```

## Opt-In Design

Safety validation is **opt-in** by design:

1. **Performance**: Skip checks when not needed
2. **Customization**: Use custom checkers for domain-specific needs
3. **Flexibility**: Apply selectively to specific operations

Enable safety in agent configuration:

```go
agentCfg := schema.AgentConfig{
    EnableSafety: true,
}
```

## Related Packages

- `pkg/agents` - Agent framework with safety middleware support
- `pkg/monitoring` - OTEL metrics and tracing
- `pkg/llms` - LLM providers that can use safety checks

## Standards

This package follows the Beluga AI Framework standards:

- `agent-os/standards/safety/middleware-pattern.md`
- `agent-os/standards/safety/risk-scoring.md`
