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

// This example demonstrates creating a custom provider
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

	// Create custom providers
	customSTT := &CustomSTTProvider{}
	customTTS := &CustomTTSProvider{}

	agentCallback := func(ctx context.Context, transcript string) (string, error) {
		return fmt.Sprintf("Custom provider response to: %s", transcript), nil
	}

	voiceSession, err := session.NewVoiceSession(ctx,
		session.WithSTTProvider(customSTT),
		session.WithTTSProvider(customTTS),
		session.WithAgentCallback(agentCallback),
	)
	if err != nil {
		log.Fatalf("Failed to create voice session: %v", err)
	}

	fmt.Println("Starting voice session with custom providers...")
	err = voiceSession.Start(ctx)
	if err != nil {
		log.Fatalf("Failed to start session: %v", err)
	}

	// Process audio
	audio := []byte{1, 2, 3, 4, 5}
	err = voiceSession.ProcessAudio(ctx, audio)
	if err != nil {
		log.Printf("Error processing audio: %v", err)
	}

	<-ctx.Done()

	err = voiceSession.Stop(ctx)
	if err != nil {
		log.Printf("Error stopping session: %v", err)
	}
}

// CustomSTTProvider is a custom STT provider implementation
type CustomSTTProvider struct{}

func (c *CustomSTTProvider) Transcribe(ctx context.Context, audio []byte) (string, error) {
	// Custom transcription logic
	return "custom transcript", nil
}

func (c *CustomSTTProvider) StartStreaming(ctx context.Context) (voiceiface.StreamingSession, error) {
	return &CustomStreamingSession{}, nil
}

// CustomTTSProvider is a custom TTS provider implementation
type CustomTTSProvider struct{}

func (c *CustomTTSProvider) GenerateSpeech(ctx context.Context, text string) ([]byte, error) {
	// Custom speech generation logic
	return []byte{1, 2, 3}, nil
}

func (c *CustomTTSProvider) StreamGenerate(ctx context.Context, text string) (io.Reader, error) {
	return &CustomReader{}, nil
}

// CustomStreamingSession implements StreamingSession
type CustomStreamingSession struct{}

func (c *CustomStreamingSession) SendAudio(ctx context.Context, audio []byte) error {
	return nil
}

func (c *CustomStreamingSession) ReceiveTranscript() <-chan voiceiface.TranscriptResult {
	ch := make(chan voiceiface.TranscriptResult, 1)
	ch <- voiceiface.TranscriptResult{Text: "custom", IsFinal: true}
	close(ch)
	return ch
}

func (c *CustomStreamingSession) Close() error {
	return nil
}

// CustomReader implements io.Reader
type CustomReader struct{}

func (c *CustomReader) Read(p []byte) (n int, err error) {
	return 0, io.EOF
}
