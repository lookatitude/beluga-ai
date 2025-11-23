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

// This example demonstrates a simple voice agent session
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

	// Create mock providers (in real usage, use actual providers)
	sttProvider := &mockSTTProvider{}
	ttsProvider := &mockTTSProvider{}

	// Create agent callback
	agentCallback := func(ctx context.Context, transcript string) (string, error) {
		fmt.Printf("User said: %s\n", transcript)
		return fmt.Sprintf("I heard you say: %s", transcript), nil
	}

	// Create voice session
	voiceSession, err := session.NewVoiceSession(ctx,
		session.WithSTTProvider(sttProvider),
		session.WithTTSProvider(ttsProvider),
		session.WithAgentCallback(agentCallback),
	)
	if err != nil {
		log.Fatalf("Failed to create voice session: %v", err)
	}

	// Start session
	fmt.Println("Starting voice session...")
	err = voiceSession.Start(ctx)
	if err != nil {
		log.Fatalf("Failed to start session: %v", err)
	}
	fmt.Println("Voice session started. Listening...")

	// Simulate audio processing
	audio := []byte{1, 2, 3, 4, 5}
	err = voiceSession.ProcessAudio(ctx, audio)
	if err != nil {
		log.Printf("Error processing audio: %v", err)
	}

	// Say something
	handle, err := voiceSession.Say(ctx, "Hello! I'm ready to help.")
	if err != nil {
		log.Printf("Error saying text: %v", err)
	} else {
		err = handle.WaitForPlayout(ctx)
		if err != nil {
			log.Printf("Error waiting for playout: %v", err)
		}
	}

	// Wait for shutdown signal
	<-ctx.Done()

	// Stop session
	fmt.Println("Stopping voice session...")
	err = voiceSession.Stop(ctx)
	if err != nil {
		log.Printf("Error stopping session: %v", err)
	}
	fmt.Println("Voice session stopped.")
}

// Mock providers for demonstration
type mockSTTProvider struct{}

func (m *mockSTTProvider) Transcribe(ctx context.Context, audio []byte) (string, error) {
	return "mock transcript", nil
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
