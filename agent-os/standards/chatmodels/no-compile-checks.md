# No Compile-Time Interface Checks

ChatModels intentionally omits compile-time interface verification.

```go
// At end of chatmodels.go:

// Compile-time interface checks are removed to avoid import cycles.
// Provider packages (openai, mock, etc.) should verify their own interface implementations.
// These checks can be added to provider-specific test files if needed.
```

## Why Removed?
Standard Go pattern:
```go
var _ iface.ChatModel = (*OpenAIChatModel)(nil)  // Would cause import cycle
```

This requires chatmodels.go to import provider packages, which import iface, creating a cycle.

## Where Checks Belong
Each provider package should verify its own implementation:
```go
// In providers/openai/openai.go
var _ iface.ChatModel = (*OpenAIChatModel)(nil)
```

Or in test files:
```go
// In providers/openai/openai_test.go
func TestInterfaceCompliance(t *testing.T) {
    var _ iface.ChatModel = (*OpenAIChatModel)(nil)
}
```

## Guideline
- Root package: No compile-time checks
- Provider packages: Must include compile-time checks
- Test files: Good alternative location
