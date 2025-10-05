package embeddings

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric/noop"
)

func TestNewMetrics(t *testing.T) {
	// Test with valid meter and tracer
	meter := noop.Meter{}
	tracer := otel.Tracer("test")

	metrics, err := NewMetrics(&meter, tracer)
	if err != nil {
		t.Fatalf("NewMetrics failed: %v", err)
	}

	if metrics == nil {
		t.Fatal("NewMetrics returned nil metrics")
	}

	// Verify metrics struct is properly initialized (but metrics are no-ops)
	// We can't easily test the actual metric creation without a real meter
	ctx := context.Background()

	// Test that no-op metrics don't panic
	metrics.RecordRequest(ctx, "test_provider", "test_model", time.Second, 1, 128)
	metrics.RecordError(ctx, "test_provider", "test_model", "test_error")
	metrics.RecordTokensProcessed(ctx, "test_provider", "test_model", 100)
	metrics.StartRequest(ctx, "test_provider", "test_model")
	metrics.EndRequest(ctx, "test_provider", "test_model")
}

func TestNoOpMetrics(t *testing.T) {
	metrics := NoOpMetrics()

	if metrics == nil {
		t.Fatal("NoOpMetrics returned nil")
	}

	// Verify all fields are nil (no-op behavior)
	if metrics.requestsTotal != nil {
		t.Error("NoOpMetrics should have nil requestsTotal")
	}
	if metrics.requestDuration != nil {
		t.Error("NoOpMetrics should have nil requestDuration")
	}
	if metrics.requestsInFlight != nil {
		t.Error("NoOpMetrics should have nil requestsInFlight")
	}
	if metrics.errorsTotal != nil {
		t.Error("NoOpMetrics should have nil errorsTotal")
	}
	if metrics.tokensProcessed != nil {
		t.Error("NoOpMetrics should have nil tokensProcessed")
	}

	// Test that all operations complete without panic
	ctx := context.Background()

	// These should not panic with nil metrics
	metrics.RecordRequest(ctx, "test_provider", "test_model", time.Second, 1, 128)
	metrics.RecordError(ctx, "test_provider", "test_model", "test_error")
	metrics.RecordTokensProcessed(ctx, "test_provider", "test_model", 100)
	metrics.StartRequest(ctx, "test_provider", "test_model")
	metrics.EndRequest(ctx, "test_provider", "test_model")
}

func TestMetrics_RecordRequest(t *testing.T) {
	// Test with NoOpMetrics (nil metrics)
	metrics := NoOpMetrics()
	ctx := context.Background()

	// Should not panic
	metrics.RecordRequest(ctx, "openai", "text-embedding-ada-002", 100*time.Millisecond, 5, 1536)

	// Test with edge case values
	metrics.RecordRequest(ctx, "", "", 0, 0, 0)
	metrics.RecordRequest(ctx, "very_long_provider_name_that_might_cause_issues", "very_long_model_name", time.Hour, 1000000, 1000000)
}

func TestMetrics_RecordError(t *testing.T) {
	metrics := NoOpMetrics()
	ctx := context.Background()

	// Test various error types
	errorTypes := []string{
		"connection_failed",
		"timeout",
		"rate_limit",
		"authentication",
		"invalid_request",
		"",
		"very_long_error_type_name_that_might_cause_issues",
	}

	for _, errorType := range errorTypes {
		metrics.RecordError(ctx, "test_provider", "test_model", errorType)
	}
}

func TestMetrics_RecordTokensProcessed(t *testing.T) {
	metrics := NoOpMetrics()
	ctx := context.Background()

	// Test various token counts
	tokenCounts := []int{0, 1, 100, 1000, 1000000, -1} // Including edge cases

	for _, count := range tokenCounts {
		metrics.RecordTokensProcessed(ctx, "test_provider", "test_model", count)
	}
}

func TestMetrics_InFlightRequests(t *testing.T) {
	metrics := NoOpMetrics()
	ctx := context.Background()

	// Test start/end request cycle
	metrics.StartRequest(ctx, "test_provider", "test_model")
	metrics.EndRequest(ctx, "test_provider", "test_model")

	// Test multiple concurrent requests
	for i := 0; i < 10; i++ {
		metrics.StartRequest(ctx, "test_provider", "test_model")
	}
	for i := 0; i < 10; i++ {
		metrics.EndRequest(ctx, "test_provider", "test_model")
	}

	// Test with empty strings
	metrics.StartRequest(ctx, "", "")
	metrics.EndRequest(ctx, "", "")
}

func TestMetrics_Concurrency(t *testing.T) {
	metrics := NoOpMetrics()
	ctx := context.Background()

	// Test concurrent access to metrics
	done := make(chan bool, 100)

	for i := 0; i < 100; i++ {
		go func(id int) {
			// Mix of different operations
			operation := id % 5
			switch operation {
			case 0:
				metrics.RecordRequest(ctx, "provider", "model", time.Millisecond, 1, 128)
			case 1:
				metrics.RecordError(ctx, "provider", "model", "test_error")
			case 2:
				metrics.RecordTokensProcessed(ctx, "provider", "model", 100)
			case 3:
				metrics.StartRequest(ctx, "provider", "model")
			case 4:
				metrics.EndRequest(ctx, "provider", "model")
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 100; i++ {
		<-done
	}
}

func BenchmarkNoOpMetrics_RecordRequest(b *testing.B) {
	metrics := NoOpMetrics()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.RecordRequest(ctx, "provider", "model", time.Millisecond, 1, 128)
	}
}

func BenchmarkNoOpMetrics_RecordError(b *testing.B) {
	metrics := NoOpMetrics()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.RecordError(ctx, "provider", "model", "error")
	}
}

func BenchmarkNoOpMetrics_InFlightRequests(b *testing.B) {
	metrics := NoOpMetrics()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i%2 == 0 {
			metrics.StartRequest(ctx, "provider", "model")
		} else {
			metrics.EndRequest(ctx, "provider", "model")
		}
	}
}
