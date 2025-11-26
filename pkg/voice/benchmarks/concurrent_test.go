package benchmarks

import (
	"context"
	"sync"
	"testing"
)

// BenchmarkConcurrentSessions benchmarks concurrent sessions (target 100+).
func BenchmarkConcurrentSessions(b *testing.B) {
	numSessions := 100

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		var wg sync.WaitGroup
		sessions := make([]any, numSessions)

		for j := 0; j < numSessions; j++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				// Simulate session creation and operations
				_ = ctx
				sessions[idx] = struct{}{}
			}(j)
		}

		wg.Wait()
	}
}

// BenchmarkConcurrentAudioProcessing benchmarks concurrent audio processing.
func BenchmarkConcurrentAudioProcessing(b *testing.B) {
	audio := make([]byte, 3200)
	concurrency := 50

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		var wg sync.WaitGroup
		sem := make(chan struct{}, concurrency)

		for j := 0; j < concurrency; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				// Simulate audio processing
				_ = ctx
				_ = audio
			}()
		}

		wg.Wait()
	}
}
