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

// This example demonstrates using multiple providers with fallback
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

	// Create primary and fallback providers
	primarySTT := &primarySTTProvider{}
	_ = &fallbackSTTProvider{} // Reserved for future fallback implementation
	primaryTTS := &primaryTTSProvider{}

	// In real implementation, you would configure fallback providers
	// For now, use primary provider
	sttProvider := primarySTT
	ttsProvider := primaryTTS

	agentCallback := func(ctx context.Context, transcript string) (string, error) {
		return fmt.Sprintf("Response from primary provider for: %s", transcript), nil
	}

	voiceSession, err := session.NewVoiceSession(ctx,
		session.WithSTTProvider(sttProvider),
		session.WithTTSProvider(ttsProvider),
		session.WithAgentCallback(agentCallback),
	)
	if err != nil {
		log.Fatalf("Failed to create voice session: %v", err)
	}

	fmt.Println("Starting multi-provider voice session...")
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

// Primary providers
type primarySTTProvider struct{}

func (p *primarySTTProvider) Transcribe(ctx context.Context, audio []byte) (string, error) {
	return "primary transcript", nil
}
func (p *primarySTTProvider) StartStreaming(ctx context.Context) (voiceiface.StreamingSession, error) {
	return nil, nil
}

type primaryTTSProvider struct{}

func (p *primaryTTSProvider) GenerateSpeech(ctx context.Context, text string) ([]byte, error) {
	return []byte{1, 2, 3}, nil
}
func (p *primaryTTSProvider) StreamGenerate(ctx context.Context, text string) (io.Reader, error) {
	return nil, nil
}

// Fallback providers
type fallbackSTTProvider struct{}

func (f *fallbackSTTProvider) Transcribe(ctx context.Context, audio []byte) (string, error) {
	return "fallback transcript", nil
}
func (f *fallbackSTTProvider) StartStreaming(ctx context.Context) (voiceiface.StreamingSession, error) {
	return nil, nil
}
