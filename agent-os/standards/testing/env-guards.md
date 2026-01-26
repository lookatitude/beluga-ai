# Two-Factor Environment Guards

Require both API key AND explicit flag for expensive tests.

```go
func shouldUseRealProviders() bool {
    return os.Getenv("OPENAI_API_KEY") != "" &&
           os.Getenv("INTEGRATION_TEST_REAL_PROVIDERS") == "true"
}
```

## Why Two Factors
- **API key alone is not enough**: Developers may have keys set but not want to run expensive tests
- **Explicit opt-in**: Prevents accidental API calls and costs
- **CI/CD control**: Can set both in CI but not locally

## Skip Guard Functions
```go
func SkipIfNoRealProviders(t *testing.T) {
    t.Helper()
    if !shouldUseRealProviders() {
        t.Skip("Skipping integration test - real providers not configured")
    }
}

func SkipIfShortMode(t *testing.T) {
    t.Helper()
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }
}

func GetEnvOrSkip(t *testing.T, key string) string {
    t.Helper()
    value := os.Getenv(key)
    if value == "" {
        t.Skipf("Skipping test: %s environment variable not set", key)
    }
    return value
}
```

## Convention
- Each optional feature gets its own skip function
- Always call `t.Helper()` for proper stack traces
- Use descriptive skip messages
