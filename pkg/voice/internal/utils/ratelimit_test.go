package utils

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewRateLimiter(t *testing.T) {
	rl := NewRateLimiter(10, time.Second)

	assert.NotNil(t, rl)
	assert.Equal(t, 10, rl.rate)
	assert.Equal(t, time.Second, rl.per)
	assert.NotNil(t, rl.requests)
}

func TestRateLimiter_Allow_WithinLimit(t *testing.T) {
	rl := NewRateLimiter(5, time.Second)

	// Allow 5 requests
	for i := 0; i < 5; i++ {
		err := rl.Allow(context.Background())
		assert.NoError(t, err, "request %d should be allowed", i)
	}
}

func TestRateLimiter_Allow_ExceedsLimit(t *testing.T) {
	rl := NewRateLimiter(3, time.Second)

	// Allow 3 requests
	for i := 0; i < 3; i++ {
		err := rl.Allow(context.Background())
		assert.NoError(t, err)
	}

	// 4th request should be rate limited
	err := rl.Allow(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rate limit exceeded")
}

func TestRateLimiter_Allow_AfterWindow(t *testing.T) {
	rl := NewRateLimiter(2, 100*time.Millisecond)

	// Use up the limit
	_ = rl.Allow(context.Background())
	_ = rl.Allow(context.Background())

	// Should be rate limited
	err := rl.Allow(context.Background())
	assert.Error(t, err)

	// Wait for window to expire
	time.Sleep(150 * time.Millisecond)

	// Should be allowed again
	err = rl.Allow(context.Background())
	assert.NoError(t, err)
}

func TestRateLimiter_Wait_Success(t *testing.T) {
	rl := NewRateLimiter(5, time.Second)
	ctx := context.Background()

	// Should succeed immediately
	err := rl.Wait(ctx)
	assert.NoError(t, err)
}

func TestRateLimiter_Wait_ContextCancellation(t *testing.T) {
	rl := NewRateLimiter(1, 5*time.Second)
	ctx, cancel := context.WithCancel(context.Background())

	// Use up the limit
	_ = rl.Allow(ctx)

	// Start waiting in a goroutine
	done := make(chan error, 1)
	go func() {
		done <- rl.Wait(ctx)
	}()

	// Cancel context
	cancel()

	// Should return context error
	select {
	case err := <-done:
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Wait should have returned after context cancellation")
	}
}

func TestRateLimiter_Wait_EventuallySucceeds(t *testing.T) {
	rl := NewRateLimiter(1, 100*time.Millisecond)
	ctx := context.Background()

	// Use up the limit
	_ = rl.Allow(ctx)

	// Wait should eventually succeed after window expires
	start := time.Now()
	err := rl.Wait(ctx)
	duration := time.Since(start)

	assert.NoError(t, err)
	// Should have waited at least 100ms
	assert.GreaterOrEqual(t, duration, 90*time.Millisecond)
}

func TestRateLimiter_Allow_RequestCleanup(t *testing.T) {
	rl := NewRateLimiter(2, 100*time.Millisecond)

	// Make requests
	_ = rl.Allow(context.Background())
	time.Sleep(50 * time.Millisecond)
	_ = rl.Allow(context.Background())

	// Wait for first request to expire
	time.Sleep(60 * time.Millisecond)

	// Should be able to make another request
	err := rl.Allow(context.Background())
	assert.NoError(t, err)
}

func TestRateLimiter_Allow_Concurrent(t *testing.T) {
	rl := NewRateLimiter(10, time.Second)
	ctx := context.Background()

	// Make concurrent requests
	errors := make(chan error, 20)
	for i := 0; i < 20; i++ {
		go func() {
			errors <- rl.Allow(ctx)
		}()
	}

	// Collect results
	successCount := 0
	errorCount := 0
	for i := 0; i < 20; i++ {
		err := <-errors
		if err == nil {
			successCount++
		} else {
			errorCount++
		}
	}

	// Should have exactly 10 successes and 10 errors
	assert.Equal(t, 10, successCount)
	assert.Equal(t, 10, errorCount)
}

