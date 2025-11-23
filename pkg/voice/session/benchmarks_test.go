package session

import (
	"context"
	"testing"
	"time"
)

// BenchmarkSession_Start benchmarks the Start method
func BenchmarkSession_Start(b *testing.B) {
	session := NewAdvancedMockSession("benchmark",
		WithActive(false),
		WithProcessingDelay(0),
	)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = session.Start(ctx)
		session.Stop(ctx)
	}
}

// BenchmarkSession_Stop benchmarks the Stop method
func BenchmarkSession_Stop(b *testing.B) {
	session := NewAdvancedMockSession("benchmark",
		WithActive(true),
		WithProcessingDelay(0),
	)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		session.Start(ctx)
		_ = session.Stop(ctx)
	}
}

// BenchmarkSession_ProcessAudio benchmarks the ProcessAudio method
func BenchmarkSession_ProcessAudio(b *testing.B) {
	session := NewAdvancedMockSession("benchmark",
		WithActive(true),
		WithProcessingDelay(0),
	)

	ctx := context.Background()
	audio := make([]byte, 1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = session.ProcessAudio(ctx, audio)
	}
}

// BenchmarkConfig_Validate benchmarks configuration validation
func BenchmarkConfig_Validate(b *testing.B) {
	config := DefaultConfig()
	config.SessionID = "test-session"
	config.Timeout = 30 * time.Minute

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = config.Validate()
	}
}
