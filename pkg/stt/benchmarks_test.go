package stt

import (
	"context"
	"testing"
)

// BenchmarkSTT_Transcribe benchmarks the Transcribe operation.
func BenchmarkSTT_Transcribe(b *testing.B) {
	// This is a placeholder - actual benchmarks would use real providers
	ctx := context.Background()
	audio := make([]byte, 16000) // 1 second of audio at 16kHz

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Benchmark would call actual provider
		_ = ctx
		_ = audio
	}
}

// BenchmarkSTT_StartStreaming benchmarks streaming STT.
func BenchmarkSTT_StartStreaming(b *testing.B) {
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Benchmark would call actual provider
		_ = ctx
	}
}
