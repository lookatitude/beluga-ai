package main

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"
)

// TestInfiniteLoop demonstrates an infinite loop without exit condition
// This test is skipped because it intentionally runs forever to demonstrate the pattern
func TestInfiniteLoop(t *testing.T) {
	t.Skip("Skipping infinite loop test - this is an example test that intentionally runs forever")
	for {
		// This will run forever
		time.Sleep(100 * time.Millisecond)
	}
}

// TestMissingTimeout demonstrates a test without timeout
func TestMissingTimeout(t *testing.T) {
	// This test has no timeout mechanism
	time.Sleep(5 * time.Second)
}

// TestLargeIterations demonstrates a loop with many iterations
func TestLargeIterations(t *testing.T) {
	// Simple operation but many iterations
	for i := 0; i < 1000; i++ {
		_ = i * 2
	}
}

// TestComplexLoop demonstrates complex operations in a loop
func TestComplexLoop(t *testing.T) {
	// Complex operations in loop
	for i := 0; i < 50; i++ {
		// Simulated network call
		http.Get("https://example.com")
		// Simulated file I/O
		os.ReadFile("test.txt")
	}
}

// TestSleepDelays demonstrates excessive sleep usage
func TestSleepDelays(t *testing.T) {
	time.Sleep(50 * time.Millisecond)
	time.Sleep(30 * time.Millisecond)
	time.Sleep(40 * time.Millisecond)
	// Total: 120ms, exceeds 100ms threshold
}

// TestActualImplementation demonstrates actual implementation usage in unit test
func TestActualImplementation(t *testing.T) {
	// Using actual implementation instead of mock
	client := http.Client{}
	client.Get("https://example.com")
}

// TestWithTimeout demonstrates proper timeout usage (should not be flagged)
func TestWithTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	select {
	case <-ctx.Done():
		return
	case <-time.After(1 * time.Second):
		// Test logic
	}
}

// TestConcurrentTestRunner demonstrates timer-based infinite loop pattern
func TestConcurrentTestRunner(t *testing.T) {
	timer := time.NewTicker(100 * time.Millisecond)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			// Do something
			return
		}
	}
}
