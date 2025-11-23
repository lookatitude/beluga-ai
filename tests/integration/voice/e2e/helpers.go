package e2e

import (
	"context"
	"io"

	voiceiface "github.com/lookatitude/beluga-ai/pkg/voice/iface"
)

// Shared mock providers for e2e tests
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

// Streaming STT provider for streaming tests
type streamingSTTProvider struct{}

func (s *streamingSTTProvider) Transcribe(ctx context.Context, audio []byte) (string, error) {
	return "final transcript", nil
}

func (s *streamingSTTProvider) StartStreaming(ctx context.Context) (voiceiface.StreamingSession, error) {
	return &mockStreamingSession{}, nil
}

// Mock streaming session
type mockStreamingSession struct{}

func (m *mockStreamingSession) SendAudio(ctx context.Context, audio []byte) error {
	return nil
}

func (m *mockStreamingSession) ReceiveTranscript() <-chan voiceiface.TranscriptResult {
	ch := make(chan voiceiface.TranscriptResult, 2)
	ch <- voiceiface.TranscriptResult{Text: "interim", IsFinal: false}
	ch <- voiceiface.TranscriptResult{Text: "final", IsFinal: true}
	close(ch)
	return ch
}

func (m *mockStreamingSession) Close() error {
	return nil
}
