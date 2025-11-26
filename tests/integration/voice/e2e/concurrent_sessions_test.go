package e2e

import (
	"context"
	"sync"
	"testing"

	voiceiface "github.com/lookatitude/beluga-ai/pkg/voice/iface"
	voicesession "github.com/lookatitude/beluga-ai/pkg/voice/session"
	"github.com/stretchr/testify/assert"
)

// TestConcurrentSessions_E2E tests concurrent sessions (100+ sessions).
func TestConcurrentSessions_E2E(t *testing.T) {
	ctx := context.Background()

	numSessions := 100
	sessions := make([]voiceiface.VoiceSession, numSessions)
	var wg sync.WaitGroup
	var mu sync.Mutex
	errors := []error{}

	// Create sessions concurrently
	for i := 0; i < numSessions; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			sttProvider := &mockSTTProvider{}
			ttsProvider := &mockTTSProvider{}

			sess, err := voicesession.NewVoiceSession(ctx,
				voicesession.WithSTTProvider(sttProvider),
				voicesession.WithTTSProvider(ttsProvider),
			)

			mu.Lock()
			if err != nil {
				errors = append(errors, err)
			} else {
				sessions[idx] = sess
			}
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	// Verify all sessions were created
	createdCount := 0
	for _, sess := range sessions {
		if sess != nil {
			createdCount++
		}
	}

	assert.GreaterOrEqual(t, createdCount, numSessions-5, "Should create most sessions")
	assert.Empty(t, errors, "Should not have errors creating sessions")

	// Start all sessions concurrently
	wg = sync.WaitGroup{}
	errors = []error{}
	for i := 0; i < createdCount; i++ {
		if sessions[i] == nil {
			continue
		}
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			err := sessions[idx].Start(ctx)
			mu.Lock()
			if err != nil {
				errors = append(errors, err)
			}
			mu.Unlock()
		}(i)
	}

	wg.Wait()
	assert.Empty(t, errors, "Should not have errors starting sessions")

	// Stop all sessions concurrently
	wg = sync.WaitGroup{}
	errors = []error{}
	for i := 0; i < createdCount; i++ {
		if sessions[i] == nil {
			continue
		}
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			err := sessions[idx].Stop(ctx)
			mu.Lock()
			if err != nil {
				errors = append(errors, err)
			}
			mu.Unlock()
		}(i)
	}

	wg.Wait()
	assert.Empty(t, errors, "Should not have errors stopping sessions")
}
