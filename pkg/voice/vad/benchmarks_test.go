package vad

import (
	"context"
	"testing"
)

// BenchmarkVAD_Process benchmarks VAD processing
func BenchmarkVAD_Process(b *testing.B) {
	ctx := context.Background()
	audio := make([]byte, 3200) // 20ms at 16kHz

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Benchmark would call actual provider
		_ = ctx
		_ = audio
	}
}

// BenchmarkVAD_ProcessStream benchmarks streaming VAD
func BenchmarkVAD_ProcessStream(b *testing.B) {
	ctx := context.Background()
	audioCh := make(chan []byte, 10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Benchmark would call actual provider
		_ = ctx
		_ = audioCh
	}
}
