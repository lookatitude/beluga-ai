# Safety Middleware

Wraps agents to add safety checks before planning.

## Usage
```go
unsafeAgent := agents.NewReActAgent(...)
safeAgent := safety.NewSafetyMiddleware(unsafeAgent)
```

## How It Works
```go
type SafetyMiddleware struct {
    iface.CompositeAgent  // Embeds wrapped agent
    checker *SafetyChecker
}

func (sm *SafetyMiddleware) Plan(ctx, steps, inputs) {
    // 1. Extract input string from inputs["input"]
    // 2. Run safety check
    // 3. If unsafe, return AgentFinish with error info
    // 4. Otherwise, delegate to wrapped agent
    return sm.CompositeAgent.Plan(ctx, steps, inputs)
}
```

## Opt-In Design
- Safety is NOT enabled by default
- Caller wraps agent explicitly
- Consistent with `agents.WithEnableSafety` option pattern

## Unsafe Response
```go
AgentFinish{
    ReturnValues: map[string]any{
        "error":  "Content failed safety validation",
        "issues": result.Issues,
    },
}
```
