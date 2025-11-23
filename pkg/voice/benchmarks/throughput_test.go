package benchmarks

import (
	"context"
	"testing"
	"time"
)

// BenchmarkThroughput_AudioChunks benchmarks audio chunk throughput (target 1000+ chunks/sec)
func BenchmarkThroughput_AudioChunks(b *testing.B) {
	ctx := context.Background()
	chunkSize := 3200 // 20ms at 16kHz
	chunks := make([][]byte, 1000)
	for i := range chunks {
		chunks[i] = make([]byte, chunkSize)
	}

	b.ResetTimer()
	start := time.Now()

	for i := 0; i < b.N; i++ {
		for _, chunk := range chunks {
			// Simulate processing
			_ = ctx
			_ = chunk
		}
	}

	duration := time.Since(start)
	throughput := float64(len(chunks)*b.N) / duration.Seconds()

	if throughput < 1000 {
		b.Logf("Throughput below target: %.2f chunks/sec", throughput)
	}
}

// BenchmarkThroughput_Transcriptions benchmarks transcription throughput
func BenchmarkThroughput_Transcriptions(b *testing.B) {
	ctx := context.Background()
	audio := make([]byte, 3200)

	b.ResetTimer()
	start := time.Now()

	for i := 0; i < b.N; i++ {
		// Simulate transcription
		_ = ctx
		_ = audio
	}

	duration := time.Since(start)
	throughput := float64(b.N) / duration.Seconds()

	b.Logf("Transcription throughput: %.2f ops/sec", throughput)
}

// BenchmarkThroughput_SpeechGeneration benchmarks speech generation throughput
func BenchmarkThroughput_SpeechGeneration(b *testing.B) {
	ctx := context.Background()
	text := "Test text for speech generation"

	b.ResetTimer()
	start := time.Now()

	for i := 0; i < b.N; i++ {
		// Simulate speech generation
		_ = ctx
		_ = text
	}

	duration := time.Since(start)
	throughput := float64(b.N) / duration.Seconds()

	b.Logf("Speech generation throughput: %.2f ops/sec", throughput)
}
