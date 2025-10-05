package orchestration

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"
)

// RetryConfig holds configuration for retry behavior
type RetryConfig struct {
	MaxAttempts     int
	InitialDelay    time.Duration
	MaxDelay        time.Duration
	BackoffFactor   float64
	JitterFactor    float64
	RetryableErrors []error
}

// DefaultRetryConfig returns a default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:   3,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
		JitterFactor:  0.1,
	}
}

// RetryExecutor handles retry logic with exponential backoff
type RetryExecutor struct {
	config RetryConfig
}

// NewRetryExecutor creates a new retry executor
func NewRetryExecutor(config RetryConfig) *RetryExecutor {
	return &RetryExecutor{config: config}
}

// ExecuteWithRetry executes a function with retry logic
func (re *RetryExecutor) ExecuteWithRetry(ctx context.Context, operation func() error) error {
	var lastErr error

	for attempt := 1; attempt <= re.config.MaxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return fmt.Errorf("operation cancelled: %w", ctx.Err())
		default:
		}

		// Execute operation with panic recovery
		var err error
		func() {
			defer func() {
				if r := recover(); r != nil {
					if panicErr, ok := r.(error); ok {
						err = panicErr
					} else {
						err = fmt.Errorf("panic: %v", r)
					}
				}
			}()
			err = operation()
		}()

		if err == nil {
			return nil // Success
		}

		lastErr = err

		// Check if error is retryable
		if !re.isRetryableError(err) {
			return fmt.Errorf("non-retryable error on attempt %d: %w", attempt, err)
		}

		// Don't retry on last attempt
		if attempt == re.config.MaxAttempts {
			break
		}

		// Calculate delay with exponential backoff and jitter
		delay := re.calculateDelay(attempt)

		select {
		case <-time.After(delay):
			// Continue to next attempt
		case <-ctx.Done():
			return fmt.Errorf("operation cancelled during backoff: %w", ctx.Err())
		}
	}

	return fmt.Errorf("operation failed after %d attempts, last error: %w", re.config.MaxAttempts, lastErr)
}

// ExecuteTaskWithRetry executes a task with retry logic
func (re *RetryExecutor) ExecuteTaskWithRetry(ctx context.Context, task Task) error {
	return re.ExecuteWithRetry(ctx, task.Execute)
}

// calculateDelay calculates the delay for the given attempt using exponential backoff
func (re *RetryExecutor) calculateDelay(attempt int) time.Duration {
	// Exponential backoff: initialDelay * (backoffFactor ^ (attempt - 1))
	delay := float64(re.config.InitialDelay) * math.Pow(re.config.BackoffFactor, float64(attempt-1))

	// Cap at max delay
	if delay > float64(re.config.MaxDelay) {
		delay = float64(re.config.MaxDelay)
	}

	// Add jitter to prevent thundering herd
	jitter := delay * re.config.JitterFactor * float64(time.Now().UnixNano()%1000) / 1000.0

	return time.Duration(delay + jitter)
}

// isRetryableError checks if an error should be retried
func (re *RetryExecutor) isRetryableError(err error) bool {
	if len(re.config.RetryableErrors) == 0 {
		// If no specific retryable errors are configured, retry all errors
		return true
	}

	for _, retryableErr := range re.config.RetryableErrors {
		if err.Error() == retryableErr.Error() {
			return true
		}
	}

	return false
}

// CircuitBreaker provides circuit breaker functionality
type CircuitBreaker struct {
	mu               sync.RWMutex
	failureThreshold int
	resetTimeout     time.Duration
	failureCount     int
	lastFailureTime  time.Time
	state            CircuitState
}

// CircuitState represents the state of the circuit breaker
type CircuitState int

const (
	StateClosed CircuitState = iota
	StateOpen
	StateHalfOpen
)

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(failureThreshold int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		failureThreshold: failureThreshold,
		resetTimeout:     resetTimeout,
		state:            StateClosed,
	}
}

// Call executes a function with circuit breaker protection
func (cb *CircuitBreaker) Call(operation func() error) error {
	cb.mu.Lock()

	switch cb.state {
	case StateOpen:
		if time.Since(cb.lastFailureTime) < cb.resetTimeout {
			cb.mu.Unlock()
			return fmt.Errorf("circuit breaker is open")
		}
		cb.state = StateHalfOpen
	case StateHalfOpen:
		// Allow one request through
	}

	cb.mu.Unlock()

	err := operation()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.failureCount++
		cb.lastFailureTime = time.Now()

		if cb.failureCount >= cb.failureThreshold {
			cb.state = StateOpen
		} else if cb.state == StateHalfOpen {
			cb.state = StateOpen
		}

		return err
	}

	// Success
	if cb.state == StateHalfOpen {
		cb.state = StateClosed
	}
	cb.failureCount = 0

	return nil
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Bulkhead provides bulkhead pattern implementation for limiting concurrent operations
type Bulkhead struct {
	semaphore chan struct{}
}

// NewBulkhead creates a new bulkhead with the specified capacity
func NewBulkhead(capacity int) *Bulkhead {
	return &Bulkhead{
		semaphore: make(chan struct{}, capacity),
	}
}

// Execute executes a function within the bulkhead
func (b *Bulkhead) Execute(ctx context.Context, operation func() error) error {
	select {
	case b.semaphore <- struct{}{}:
		defer func() { <-b.semaphore }()
		return operation()
	case <-ctx.Done():
		return ctx.Err()
	default:
		return fmt.Errorf("bulkhead capacity exceeded")
	}
}

// GetCurrentConcurrency returns the current number of concurrent operations
func (b *Bulkhead) GetCurrentConcurrency() int {
	return len(b.semaphore)
}

// GetCapacity returns the total capacity of the bulkhead
func (b *Bulkhead) GetCapacity() int {
	return cap(b.semaphore)
}
