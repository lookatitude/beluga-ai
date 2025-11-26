package transport

import (
	"context"
	"testing"
)

// BenchmarkTransport_SendAudio benchmarks audio sending.
func BenchmarkTransport_SendAudio(b *testing.B) {
	ctx := context.Background()
	audio := make([]byte, 3200) // 20ms at 16kHz

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Benchmark would call actual provider
		_ = ctx
		_ = audio
	}
}

// BenchmarkTransport_ReceiveAudio benchmarks audio receiving.
func BenchmarkTransport_ReceiveAudio(b *testing.B) {
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Benchmark would call actual provider
		_ = ctx
	}
}
