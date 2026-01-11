package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/s2s"
	"github.com/lookatitude/beluga-ai/pkg/voice/session"
)

// This example demonstrates basic S2S (Speech-to-Speech) usage with a voice session.
// S2S providers enable end-to-end speech conversations without explicit intermediate text steps.
func main() {
	ctx := context.Background()

	// Step 1: Create S2S provider configuration
	config := s2s.DefaultConfig()
	config.Provider = "amazon_nova" // or "grok", "gemini", "openai_realtime"
	config.APIKey = os.Getenv("AWS_ACCESS_KEY_ID") // Set your API key

	// Configure audio settings
	config.SampleRate = 24000
	config.Channels = 1
	config.Language = "en-US"

	// Configure latency target
	config.LatencyTarget = "low" // Options: low, medium, high

	// Step 2: Create S2S provider
	provider, err := s2s.NewProvider(ctx, config.Provider, config)
	if err != nil {
		log.Fatalf("Failed to create S2S provider: %v", err)
	}
	fmt.Println("✓ S2S provider created:", provider.Name())

	// Step 3: Create voice session with S2S provider
	// Note: S2S is an alternative to STT+TTS, so we don't need STT/TTS providers
	voiceSession, err := session.NewVoiceSession(ctx,
		session.WithS2SProvider(provider),
		session.WithConfig(session.DefaultConfig()),
	)
	if err != nil {
		log.Fatalf("Failed to create voice session: %v", err)
	}
	fmt.Println("✓ Voice session created")

	// Step 4: Start the session
	err = voiceSession.Start(ctx)
	if err != nil {
		log.Fatalf("Failed to start session: %v", err)
	}
	fmt.Println("✓ Session started")

	// Step 5: Process audio through S2S provider
	// In a real application, this would come from a microphone or audio stream
	audioData := []byte{1, 2, 3, 4, 5} // Placeholder audio data

	fmt.Println("\nProcessing audio through S2S provider...")
	err = voiceSession.ProcessAudio(ctx, audioData)
	if err != nil {
		log.Printf("Error processing audio: %v", err)
	} else {
		fmt.Println("✓ Audio processed successfully")
	}

	// Step 6: Wait a bit for processing
	time.Sleep(1 * time.Second)

	// Step 7: Stop the session
	err = voiceSession.Stop(ctx)
	if err != nil {
		log.Fatalf("Failed to stop session: %v", err)
	}
	fmt.Println("✓ Session stopped")

	fmt.Println("\nExample completed successfully!")
	fmt.Println("\nNext steps:")
	fmt.Println("- Use a real audio source (microphone, file, WebRTC)")
	fmt.Println("- Configure fallback providers for reliability")
	fmt.Println("- Enable external reasoning mode with agent integration")
}
