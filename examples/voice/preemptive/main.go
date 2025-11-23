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

// This example demonstrates preemptive generation
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

	// Create streaming STT provider for preemptive generation
	sttProvider := &streamingSTTProvider{}
	ttsProvider := &mockTTSProvider{}

	responses := []string{}
	agentCallback := func(ctx context.Context, transcript string) (string, error) {
		responses = append(responses, transcript)
		return fmt.Sprintf("Preemptive response to: %s", transcript), nil
	}

	voiceSession, err := session.NewVoiceSession(ctx,
		session.WithSTTProvider(sttProvider),
		session.WithTTSProvider(ttsProvider),
		session.WithAgentCallback(agentCallback),
	)
	if err != nil {
		log.Fatalf("Failed to create voice session: %v", err)
	}

	fmt.Println("Starting voice session with preemptive generation...")
	err = voiceSession.Start(ctx)
	if err != nil {
		log.Fatalf("Failed to start session: %v", err)
	}

	// Process audio - should generate responses based on interim transcripts
	audio := []byte{1, 2, 3, 4, 5}
	err = voiceSession.ProcessAudio(ctx, audio)
	if err != nil {
		log.Printf("Error processing audio: %v", err)
	}

	fmt.Printf("Received %d responses\n", len(responses))

	<-ctx.Done()

	err = voiceSession.Stop(ctx)
	if err != nil {
		log.Printf("Error stopping session: %v", err)
	}
}

// Streaming STT provider
type streamingSTTProvider struct{}

func (s *streamingSTTProvider) Transcribe(ctx context.Context, audio []byte) (string, error) {
	return "final transcript", nil
}
func (s *streamingSTTProvider) StartStreaming(ctx context.Context) (voiceiface.StreamingSession, error) {
	return &mockStreamingSession{}, nil
}

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

type mockTTSProvider struct{}

func (m *mockTTSProvider) GenerateSpeech(ctx context.Context, text string) ([]byte, error) {
	return []byte{1, 2, 3}, nil
}
func (m *mockTTSProvider) StreamGenerate(ctx context.Context, text string) (io.Reader, error) {
	return nil, nil
}
