package providers

import (
	"context"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/s2s"
	"github.com/lookatitude/beluga-ai/pkg/voice/s2s/internal"
)

// BenchmarkAmazonNova benchmarks Amazon Nova provider performance.
func BenchmarkAmazonNova(b *testing.B) {
	ctx := context.Background()
	config := s2s.DefaultConfig()
	config.Provider = "amazon_nova"
	// Use mock for benchmarking (real provider would require API key)
	mockProvider := s2s.NewAdvancedMockS2SProvider("amazon_nova",
		s2s.WithMockDelay(50*time.Millisecond))

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
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := mockProvider.Process(ctx, input, convCtx)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkGrok benchmarks Grok provider performance.
func BenchmarkGrok(b *testing.B) {
	ctx := context.Background()
	mockProvider := s2s.NewAdvancedMockS2SProvider("grok",
		s2s.WithMockDelay(60*time.Millisecond))

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
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := mockProvider.Process(ctx, input, convCtx)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkGemini benchmarks Gemini provider performance.
func BenchmarkGemini(b *testing.B) {
	ctx := context.Background()
	mockProvider := s2s.NewAdvancedMockS2SProvider("gemini",
		s2s.WithMockDelay(55*time.Millisecond))

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
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := mockProvider.Process(ctx, input, convCtx)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkOpenAIRealtime benchmarks OpenAI Realtime provider performance.
func BenchmarkOpenAIRealtime(b *testing.B) {
	ctx := context.Background()
	mockProvider := s2s.NewAdvancedMockS2SProvider("openai_realtime",
		s2s.WithMockDelay(45*time.Millisecond))

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
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := mockProvider.Process(ctx, input, convCtx)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkProviderComparison compares all providers.
func BenchmarkProviderComparison(b *testing.B) {
	providers := []struct {
		name     string
		delay    time.Duration
		provider *s2s.AdvancedMockS2SProvider
	}{
		{
			name:     "amazon_nova",
			delay:    50 * time.Millisecond,
			provider: s2s.NewAdvancedMockS2SProvider("amazon_nova", s2s.WithMockDelay(50*time.Millisecond)),
		},
		{
			name:     "grok",
			delay:    60 * time.Millisecond,
			provider: s2s.NewAdvancedMockS2SProvider("grok", s2s.WithMockDelay(60*time.Millisecond)),
		},
		{
			name:     "gemini",
			delay:    55 * time.Millisecond,
			provider: s2s.NewAdvancedMockS2SProvider("gemini", s2s.WithMockDelay(55*time.Millisecond)),
		},
		{
			name:     "openai_realtime",
			delay:    45 * time.Millisecond,
			provider: s2s.NewAdvancedMockS2SProvider("openai_realtime", s2s.WithMockDelay(45*time.Millisecond)),
		},
	}

	ctx := context.Background()
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

	for _, p := range providers {
		b.Run(p.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := p.provider.Process(ctx, input, convCtx)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
