package session

import (
	"bytes"
	"context"
	"io"
	"testing"

	voiceiface "github.com/lookatitude/beluga-ai/pkg/voice/iface"
	sessioniface "github.com/lookatitude/beluga-ai/pkg/voice/session/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock providers for testing.
type mockSTTProvider struct{}

func (m *mockSTTProvider) Transcribe(ctx context.Context, audio []byte) (string, error) {
	return "", nil
}

func (m *mockSTTProvider) StartStreaming(ctx context.Context) (voiceiface.StreamingSession, error) {
	return nil, nil
}

type mockTTSProvider struct{}

func (m *mockTTSProvider) GenerateSpeech(ctx context.Context, text string) ([]byte, error) {
	return nil, nil
}

func (m *mockTTSProvider) StreamGenerate(ctx context.Context, text string) (io.Reader, error) {
	// Return a simple reader for testing
	return bytes.NewReader([]byte("mock audio data")), nil
}

func TestNewVoiceSession(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		opts    []VoiceOption
		wantErr bool
	}{
		{
			name: "valid options with required providers",
			opts: []VoiceOption{
				WithSTTProvider(&mockSTTProvider{}), // Mock provider
				WithTTSProvider(&mockTTSProvider{}),
			},
			wantErr: false,
		},
		{
			name:    "missing STT provider",
			opts:    []VoiceOption{},
			wantErr: true,
		},
		{
			name: "missing TTS provider",
			opts: []VoiceOption{
				WithSTTProvider(&mockSTTProvider{}),
			},
			wantErr: true,
		},
		{
			name: "with config",
			opts: []VoiceOption{
				WithSTTProvider(&mockSTTProvider{}),
				WithTTSProvider(&mockTTSProvider{}),
				WithConfig(DefaultConfig()),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session, err := NewVoiceSession(ctx, tt.opts...)
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, session)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, session)
			}
		})
	}
}

func TestNewVoiceSession_WithCallbacks(t *testing.T) {
	ctx := context.Background()

	agentCallback := func(ctx context.Context, transcript string) (string, error) {
		return "response", nil
	}
	stateCallback := func(state sessioniface.SessionState) {}

	session, err := NewVoiceSession(ctx,
		WithSTTProvider(&mockSTTProvider{}),
		WithTTSProvider(&mockTTSProvider{}),
		WithAgentCallback(agentCallback),
		WithOnStateChanged(stateCallback),
	)
	require.NoError(t, err)
	assert.NotNil(t, session)
}

func TestGenerateSessionID(t *testing.T) {
	id1 := generateSessionID()
	id2 := generateSessionID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2, "Session IDs should be unique")
}
