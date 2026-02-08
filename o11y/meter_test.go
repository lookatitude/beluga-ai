package o11y

import (
	"context"
	"testing"
)

func TestTokenUsage(t *testing.T) {
	// TokenUsage should not panic even without explicit InitMeter.
	ctx := context.Background()
	TokenUsage(ctx, 100, 50)
}

func TestOperationDuration(t *testing.T) {
	ctx := context.Background()
	OperationDuration(ctx, 123.45)
}

func TestCost(t *testing.T) {
	ctx := context.Background()
	Cost(ctx, 0.0042)
}

func TestCounter(t *testing.T) {
	ctx := context.Background()
	Counter(ctx, "test.counter", 5)
}

func TestHistogram(t *testing.T) {
	ctx := context.Background()
	Histogram(ctx, "test.histogram", 99.9)
}

func TestInitMeter(t *testing.T) {
	err := InitMeter("test-meter-service")
	if err != nil {
		t.Fatalf("InitMeter: %v", err)
	}

	// After init, all instrument functions should work.
	ctx := context.Background()
	TokenUsage(ctx, 10, 20)
	OperationDuration(ctx, 50.0)
	Cost(ctx, 0.001)
	Counter(ctx, "post_init.counter", 1)
	Histogram(ctx, "post_init.histogram", 42.0)
}
