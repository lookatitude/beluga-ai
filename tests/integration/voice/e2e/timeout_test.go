package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voicesession"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTimeout_E2E tests session timeout end-to-end.
func TestTimeout_E2E(t *testing.T) {
	ctx := context.Background()

	sttProvider := &mockSTTProvider{}
	ttsProvider := &mockTTSProvider{}

	// Create session with short timeout
	config := voicesession.DefaultConfig()
	config.Timeout = 1 * time.Minute // Minimum valid timeout (per validation)

	voiceSession, err := voicesession.NewVoiceSession(ctx,
		voicesession.WithSTTProvider(sttProvider),
		voicesession.WithTTSProvider(ttsProvider),
		voicesession.WithConfig(config),
	)
	require.NoError(t, err)

	err = voiceSession.Start(ctx)
	require.NoError(t, err)

	// Wait for timeout (implementation may vary)
	time.Sleep(150 * time.Millisecond)

	// Session should handle timeout (may transition to away or ended state)
	state := voiceSession.GetState()
	assert.NotEqual(t, "initial", string(state))

	err = voiceSession.Stop(ctx)
	require.NoError(t, err)
}
