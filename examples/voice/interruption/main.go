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

// This example demonstrates interruption handling
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

	sttProvider := &mockSTTProvider{}
	ttsProvider := &mockTTSProvider{}

	agentCallback := func(ctx context.Context, transcript string) (string, error) {
		return "Response", nil
	}

	voiceSession, err := session.NewVoiceSession(ctx,
		session.WithSTTProvider(sttProvider),
		session.WithTTSProvider(ttsProvider),
		session.WithAgentCallback(agentCallback),
	)
	if err != nil {
		log.Fatalf("Failed to create voice session: %v", err)
	}

	fmt.Println("Starting voice session with interruption handling...")
	err = voiceSession.Start(ctx)
	if err != nil {
		log.Fatalf("Failed to start session: %v", err)
	}

	// Start saying something long
	handle, err := voiceSession.Say(ctx, "This is a long response that can be interrupted")
	if err != nil {
		log.Printf("Error saying text: %v", err)
	} else {
		// Simulate user interruption
		fmt.Println("User interrupts...")
		audio := []byte{1, 2, 3, 4, 5}
		err = voiceSession.ProcessAudio(ctx, audio)
		if err != nil {
			log.Printf("Error processing interruption: %v", err)
		}

		// Cancel current say operation
		err = handle.Cancel()
		if err != nil {
			log.Printf("Error canceling: %v", err)
		} else {
			fmt.Println("Interruption handled successfully")
		}
	}

	<-ctx.Done()

	err = voiceSession.Stop(ctx)
	if err != nil {
		log.Printf("Error stopping session: %v", err)
	}
}

// Mock providers
type mockSTTProvider struct{}

func (m *mockSTTProvider) Transcribe(ctx context.Context, audio []byte) (string, error) {
	return "interruption", nil
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
