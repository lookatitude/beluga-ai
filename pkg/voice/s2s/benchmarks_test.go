package s2s

import (
	"context"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/s2s/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/s2s/internal"
)

// BenchmarkProcess benchmarks the Process operation for different providers.
func BenchmarkProcess(b *testing.B) {
	ctx := context.Background()
	mockProvider := NewAdvancedMockS2SProvider("benchmark-provider",
		WithMockDelay(10*time.Millisecond))

	input := &internal.AudioInput{
		Data: make([]byte, 16000), // 1 second of audio at 16kHz
		Format: internal.AudioFormat{
			SampleRate: 24000,
			Channels:   1,
			BitDepth:   16,
			Encoding:   "PCM",
		},
		Timestamp: time.Now(),
	}

	convCtx := &internal.ConversationContext{
		SessionID: "benchmark-session",
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := mockProvider.Process(ctx, input, convCtx)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkProcessLatency benchmarks latency for Process operations.
func BenchmarkProcessLatency(b *testing.B) {
	ctx := context.Background()
	mockProvider := NewAdvancedMockS2SProvider("benchmark-provider",
		WithMockDelay(50*time.Millisecond))

	input := &internal.AudioInput{
		Data: make([]byte, 16000),
		Format: internal.AudioFormat{
			SampleRate: 24000,
			Channels:   1,
			BitDepth:   16,
			Encoding:   "PCM",
		},
		Timestamp: time.Now(),
	}

	convCtx := &internal.ConversationContext{
		SessionID: "benchmark-session",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		_, err := mockProvider.Process(ctx, input, convCtx)
		if err != nil {
			b.Fatal(err)
		}
		latency := time.Since(start)
		b.ReportMetric(float64(latency.Nanoseconds())/1e6, "ms/op")
	}
}

// BenchmarkConcurrentSessions benchmarks concurrent session processing.
func BenchmarkConcurrentSessions(b *testing.B) {
	ctx := context.Background()
	mockProvider := NewAdvancedMockS2SProvider("benchmark-provider",
		WithMockDelay(10*time.Millisecond))

	input := &internal.AudioInput{
		Data: make([]byte, 16000),
		Format: internal.AudioFormat{
			SampleRate: 24000,
			Channels:   1,
			BitDepth:   16,
			Encoding:   "PCM",
		},
		Timestamp: time.Now(),
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		convCtx := &internal.ConversationContext{
			SessionID: "benchmark-session",
		}
		for pb.Next() {
			_, err := mockProvider.Process(ctx, input, convCtx)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkThroughput benchmarks throughput (operations per second).
func BenchmarkThroughput(b *testing.B) {
	ctx := context.Background()
	mockProvider := NewAdvancedMockS2SProvider("benchmark-provider",
		WithMockDelay(5*time.Millisecond))

	input := &internal.AudioInput{
		Data: make([]byte, 8000), // Smaller chunk for throughput testing
		Format: internal.AudioFormat{
			SampleRate: 24000,
			Channels:   1,
			BitDepth:   16,
			Encoding:   "PCM",
		},
		Timestamp: time.Now(),
	}

	convCtx := &internal.ConversationContext{
		SessionID: "benchmark-session",
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := mockProvider.Process(ctx, input, convCtx)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkFallback benchmarks fallback mechanism performance.
func BenchmarkFallback(b *testing.B) {
	ctx := context.Background()

	primary := NewAdvancedMockS2SProvider("primary",
		WithError(NewS2SError("Process", ErrCodeNetworkError, nil)))
	fallback := NewAdvancedMockS2SProvider("fallback",
		WithMockDelay(10*time.Millisecond))

	fallbackManager := NewProviderFallback(primary, []iface.S2SProvider{fallback}, nil)

	input := &internal.AudioInput{
		Data: make([]byte, 16000),
		Format: internal.AudioFormat{
			SampleRate: 24000,
			Channels:   1,
			BitDepth:   16,
			Encoding:   "PCM",
		},
		Timestamp: time.Now(),
	}

	convCtx := &internal.ConversationContext{
		SessionID: "benchmark-session",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := fallbackManager.ProcessWithFallback(ctx, input, convCtx)
		if err != nil {
			b.Fatal(err)
		}
	}
}
