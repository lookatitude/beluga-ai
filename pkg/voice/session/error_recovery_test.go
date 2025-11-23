package session

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/session/internal"
	"github.com/stretchr/testify/assert"
)

func TestErrorRecovery_ShouldRetry(t *testing.T) {
	recovery := internal.NewErrorRecovery(3, 100*time.Millisecond)

	// Test retryable error
	err := errors.New("timeout error")
	assert.True(t, recovery.ShouldRetry("test", err))

	// Test non-retryable error
	err = errors.New("invalid input")
	assert.False(t, recovery.ShouldRetry("test", err))

	// Test max retries exceeded
	for i := 0; i < 4; i++ {
		recovery.ShouldRetry("test", err)
	}
	assert.False(t, recovery.ShouldRetry("test", err))
}

func TestErrorRecovery_RetryWithBackoff(t *testing.T) {
	recovery := internal.NewErrorRecovery(2, 10*time.Millisecond)

	ctx := context.Background()
	attempts := 0

	err := recovery.RetryWithBackoff(ctx, "test", func() error {
		attempts++
		if attempts < 3 {
			// Use a retryable error (contains "timeout" or "network")
			return errors.New("network timeout error")
		}
		return nil
	})

	// With maxRetries=2, the function will try attempts 0, 1, 2 (3 total)
	// The test expects success on attempt 3, which should happen
	assert.NoError(t, err)
	assert.Equal(t, 3, attempts)
}

func TestErrorRecovery_Reset(t *testing.T) {
	recovery := internal.NewErrorRecovery(3, 100*time.Millisecond)

	err := errors.New("timeout error")
	recovery.ShouldRetry("test", err)
	recovery.Reset("test")

	// Should be able to retry again after reset
	assert.True(t, recovery.ShouldRetry("test", err))
}
