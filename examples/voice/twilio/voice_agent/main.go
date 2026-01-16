package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/backend"
	vbiface "github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	// Configure Twilio voice backend
	config := &vbiface.Config{
		Provider:     "twilio",
		PipelineType: vbiface.PipelineTypeSTTTTS,
		STTProvider:  "openai", // Configure your STT provider
		TTSProvider:  "openai", // Configure your TTS provider
		ProviderConfig: map[string]any{
			"account_sid":  os.Getenv("TWILIO_ACCOUNT_SID"),
			"auth_token":   os.Getenv("TWILIO_AUTH_TOKEN"),
			"phone_number": os.Getenv("TWILIO_PHONE_NUMBER"),
			"webhook_url":  os.Getenv("TWILIO_WEBHOOK_URL"),
			// Optional: Advanced features
			// "vad_provider": "silero",
			// "vad_config": map[string]any{
			//     "model_path": "/path/to/vad/model",
			// },
			// "turn_detector_provider": "silence",
			// "turn_detector_config": map[string]any{
			//     "min_silence_duration": "1s",
			// },
			// "noise_cancellation_provider": "rnnoise",
			// "noise_cancellation_config": map[string]any{
			//     "model_path": "/path/to/noise/model",
			// },
			// "memory_config": map[string]any{
			//     "type": "buffer",
			//     "window_size": 10,
			// },
			// For S2S mode, use:
			// "s2s_provider": "amazon_nova",
			// "s2s_config": map[string]any{
			//     "api_key": os.Getenv("AWS_ACCESS_KEY_ID"),
			// },
		},
		LatencyTarget:         2 * time.Second, // FR-009: <2s latency
		MaxConcurrentSessions: 100,             // SC-003: 100 concurrent calls
	}

	// Create backend
	voiceBackend, err := backend.NewBackend(ctx, "twilio", config)
	if err != nil {
		log.Fatalf("Failed to create voice backend: %v", err)
	}

	// Start backend
	if err := voiceBackend.Start(ctx); err != nil {
		log.Fatalf("Failed to start voice backend: %v", err)
	}
	defer voiceBackend.Stop(ctx)

	log.Println("Twilio voice backend started. Waiting for calls...")

	// Agent callback function
	agentCallback := func(ctx context.Context, transcript string) (string, error) {
		// Simple echo agent - replace with your agent logic
		return fmt.Sprintf("You said: %s", transcript), nil
	}

	// Example: Create a session for an outbound call
	sessionConfig := &vbiface.SessionConfig{
		UserID:        "user-123",
		Transport:     "websocket",
		ConnectionURL: "wss://example.com/voice",
		PipelineType:  vbiface.PipelineTypeSTTTTS,
		AgentCallback: agentCallback,
		Metadata: map[string]any{
			"to":   "+15559876543", // Destination phone number
			"from": config.ProviderConfig["phone_number"].(string),
		},
	}

	session, err := voiceBackend.CreateSession(ctx, sessionConfig)
	if err != nil {
		log.Fatalf("Failed to create session: %v", err)
	}

	// Start session
	if err := session.Start(ctx); err != nil {
		log.Fatalf("Failed to start session: %v", err)
	}
	defer session.Stop(ctx)

	log.Printf("Session started: %s", session.GetID())

	// Wait for context cancellation
	<-ctx.Done()
	log.Println("Shutting down...")
}
