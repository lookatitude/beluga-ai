package session

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVoiceOptions(t *testing.T) {
	opts := &VoiceOptions{}

	WithSTTProvider(nil)(opts)
	assert.Nil(t, opts.STTProvider)

	WithTTSProvider(nil)(opts)
	assert.Nil(t, opts.TTSProvider)

	WithVADProvider(nil)(opts)
	assert.Nil(t, opts.VADProvider)

	WithTurnDetector(nil)(opts)
	assert.Nil(t, opts.TurnDetector)

	WithTransport(nil)(opts)
	assert.Nil(t, opts.Transport)

	WithNoiseCancellation(nil)(opts)
	assert.Nil(t, opts.NoiseCancellation)

	callback := func(ctx context.Context, transcript string) (string, error) {
		return "response", nil
	}
	WithAgentCallback(callback)(opts)
	assert.NotNil(t, opts.AgentCallback)

	stateCallback := func(state SessionState) {}
	WithOnStateChanged(stateCallback)(opts)
	assert.NotNil(t, opts.OnStateChanged)

	config := DefaultConfig()
	WithConfig(config)(opts)
	assert.Equal(t, config, opts.Config)
}

func TestSessionStateConstants(t *testing.T) {
	assert.Equal(t, "initial", string(SessionStateInitial))
	assert.Equal(t, "listening", string(SessionStateListening))
	assert.Equal(t, "processing", string(SessionStateProcessing))
	assert.Equal(t, "speaking", string(SessionStateSpeaking))
	assert.Equal(t, "away", string(SessionStateAway))
	assert.Equal(t, "ended", string(SessionStateEnded))
}
