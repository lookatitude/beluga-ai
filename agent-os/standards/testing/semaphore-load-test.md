# Semaphore Load Testing

Rate-limit concurrent operations to prevent thundering herd.

```go
func (h *IntegrationTestHelper) CrossPackageLoadTest(
    t *testing.T,
    scenario func() error,
    numOperations, concurrency int,
) {
    var wg sync.WaitGroup
    errChan := make(chan error, numOperations)
    semaphore := make(chan struct{}, concurrency)  // Rate limiting

    start := time.Now()

    for i := 0; i < numOperations; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()

            semaphore <- struct{}{}           // Acquire slot
            defer func() { <-semaphore }()    // Release slot

            if err := scenario(); err != nil {
                errChan <- err
            }
        }()
    }

    wg.Wait()
    close(errChan)

    duration := time.Since(start)

    // Check for errors
    for err := range errChan {
        require.NoError(t, err)
    }

    t.Logf("Load test: %d ops in %v (%.2f ops/sec)",
        numOperations, duration, float64(numOperations)/duration.Seconds())
}
```

## Why Semaphore Pattern
- Prevents overwhelming external services
- Controls memory usage under load
- Simulates realistic concurrency limits

## Guidelines
- Default concurrency: 5-10 for integration tests
- Log performance metrics for comparison
- Collect all errors before failing (don't fail fast)
