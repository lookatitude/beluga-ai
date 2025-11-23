package quickstart

import (
	"context"
	"io"

	voiceiface "github.com/lookatitude/beluga-ai/pkg/voice/iface"
)

// Shared mock providers for quickstart tests
type mockSTTProvider struct{}

func (m *mockSTTProvider) Transcribe(ctx context.Context, audio []byte) (string, error) {
	return "test transcript", nil
}

func (m *mockSTTProvider) StartStreaming(ctx context.Context) (voiceiface.StreamingSession, error) {
	return nil, nil
}

type mockTTSProvider struct{}

func (m *mockTTSProvider) GenerateSpeech(ctx context.Context, text string) ([]byte, error) {
	return []byte{1, 2, 3}, nil
}

func (m *mockTTSProvider) StreamGenerate(ctx context.Context, text string) (io.Reader, error) {
	return nil, nil
}
