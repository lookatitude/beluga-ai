package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"

	voiceiface "github.com/lookatitude/beluga-ai/pkg/voice/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/session"
)

// This example demonstrates streaming voice session functionality
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	// Create streaming-capable providers
	sttProvider := &streamingSTTProvider{}
	ttsProvider := &streamingTTSProvider{}

	agentCallback := func(ctx context.Context, transcript string) (string, error) {
		return fmt.Sprintf("Streaming response to: %s", transcript), nil
	}

	voiceSession, err := session.NewVoiceSession(ctx,
		session.WithSTTProvider(sttProvider),
		session.WithTTSProvider(ttsProvider),
		session.WithAgentCallback(agentCallback),
	)
	if err != nil {
		log.Fatalf("Failed to create voice session: %v", err)
	}

	fmt.Println("Starting streaming voice session...")
	err = voiceSession.Start(ctx)
	if err != nil {
		log.Fatalf("Failed to start session: %v", err)
	}

	// Process streaming audio chunks
	chunks := [][]byte{
		{1, 2, 3},
		{4, 5, 6},
		{7, 8, 9},
	}

	for i, chunk := range chunks {
		fmt.Printf("Processing chunk %d...\n", i+1)
		err = voiceSession.ProcessAudio(ctx, chunk)
		if err != nil {
			log.Printf("Error processing chunk: %v", err)
		}
	}

	// Use streaming TTS
	handle, err := voiceSession.Say(ctx, "This is a streaming response")
	if err != nil {
		log.Printf("Error saying text: %v", err)
	} else {
		_ = handle.WaitForPlayout(ctx)
	}

	<-ctx.Done()

	err = voiceSession.Stop(ctx)
	if err != nil {
		log.Printf("Error stopping session: %v", err)
	}
}

// Streaming providers
type streamingSTTProvider struct{}

func (s *streamingSTTProvider) Transcribe(ctx context.Context, audio []byte) (string, error) {
	return "streaming transcript", nil
}
func (s *streamingSTTProvider) StartStreaming(ctx context.Context) (voiceiface.StreamingSession, error) {
	return &mockStreamingSession{}, nil
}

type streamingTTSProvider struct{}

func (s *streamingTTSProvider) GenerateSpeech(ctx context.Context, text string) ([]byte, error) {
	return []byte{1, 2, 3}, nil
}
func (s *streamingTTSProvider) StreamGenerate(ctx context.Context, text string) (io.Reader, error) {
	return &mockReader{}, nil
}

type mockStreamingSession struct{}

func (m *mockStreamingSession) SendAudio(ctx context.Context, audio []byte) error {
	return nil
}
func (m *mockStreamingSession) ReceiveTranscript() <-chan voiceiface.TranscriptResult {
	ch := make(chan voiceiface.TranscriptResult, 1)
	ch <- voiceiface.TranscriptResult{Text: "test", IsFinal: true}
	close(ch)
	return ch
}
func (m *mockStreamingSession) Close() error {
	return nil
}

type mockReader struct{}

func (m *mockReader) Read(p []byte) (n int, err error) {
	return 0, io.EOF
}
