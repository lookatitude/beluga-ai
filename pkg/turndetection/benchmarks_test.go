package turndetection

import (
	"context"
	"testing"
	"time"
)

// BenchmarkTurnDetection_DetectTurn benchmarks turn detection.
func BenchmarkTurnDetection_DetectTurn(b *testing.B) {
	ctx := context.Background()
	audio := make([]byte, 3200) // 20ms at 16kHz

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Benchmark would call actual provider
		_ = ctx
		_ = audio
	}
}

// BenchmarkTurnDetection_DetectTurnWithSilence benchmarks turn detection with silence.
func BenchmarkTurnDetection_DetectTurnWithSilence(b *testing.B) {
	ctx := context.Background()
	audio := make([]byte, 3200)
	silenceDuration := 500 * time.Millisecond

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Benchmark would call actual provider
		_ = ctx
		_ = audio
		_ = silenceDuration
	}
}
