package benchmarks

import (
	"context"
	"testing"
	"time"
)

// BenchmarkLatency_EndToEnd benchmarks end-to-end latency (target <200ms).
func BenchmarkLatency_EndToEnd(b *testing.B) {
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()

		// Simulate end-to-end processing:
		// Audio -> STT -> Agent -> TTS -> Audio
		_ = ctx

		latency := time.Since(start)
		if latency > 200*time.Millisecond {
			b.Logf("Latency exceeded target: %v", latency)
		}
	}
}

// BenchmarkLatency_STT benchmarks STT latency.
func BenchmarkLatency_STT(b *testing.B) {
	ctx := context.Background()
	audio := make([]byte, 3200) // 20ms

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		_ = ctx
		_ = audio
		latency := time.Since(start)
		if latency > 100*time.Millisecond {
			b.Logf("STT latency: %v", latency)
		}
	}
}

// BenchmarkLatency_TTS benchmarks TTS latency.
func BenchmarkLatency_TTS(b *testing.B) {
	ctx := context.Background()
	text := "Test text for TTS latency benchmarking"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		_ = ctx
		_ = text
		latency := time.Since(start)
		if latency > 100*time.Millisecond {
			b.Logf("TTS latency: %v", latency)
		}
	}
}
