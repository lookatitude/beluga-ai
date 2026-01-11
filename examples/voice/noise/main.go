package main

import (
	"context"
	"fmt"
	"log"

	_ "github.com/lookatitude/beluga-ai/pkg/voice/noise/providers/mock"
	"github.com/lookatitude/beluga-ai/pkg/voice/noise"
)

func main() {
	fmt.Println("🔇 Beluga AI - Noise Cancellation Example")
	fmt.Println("==========================================")

	ctx := context.Background()

	// Step 1: Create Noise Cancellation configuration
	fmt.Println("\n📋 Step 1: Creating Noise Cancellation configuration...")
	config := noise.DefaultConfig()
	config.Provider = "mock" // Use mock provider for this example
	// In production, use: "rnnoise", etc.
	fmt.Println("✅ Configuration created")

	// Step 2: Create Noise Cancellation provider
	fmt.Println("\n📋 Step 2: Creating Noise Cancellation provider...")
	provider, err := noise.NewProvider(ctx, config.Provider, config)
	if err != nil {
		log.Fatalf("Failed to create Noise Cancellation provider: %v", err)
	}
	fmt.Printf("✅ Provider created: %s\n", config.Provider)

	// Step 3: Cancel noise from audio data
	fmt.Println("\n📋 Step 3: Canceling noise from audio...")
	// In a real application, this would be actual audio data with background noise
	audioData := []byte{1, 2, 3, 4, 5} // Placeholder audio data
	cleanedAudio, err := provider.CancelNoise(ctx, audioData)
	if err != nil {
		log.Fatalf("Failed to cancel noise: %v", err)
	}
	fmt.Printf("✅ Noise canceled: %d bytes input, %d bytes output\n", len(audioData), len(cleanedAudio))

	// Step 4: Process streaming audio (optional)
	fmt.Println("\n📋 Step 4: Processing streaming audio...")
	streamingSession, err := provider.StartStreaming(ctx)
	if err != nil {
		log.Printf("Note: Streaming not available with mock provider: %v", err)
	} else {
		fmt.Println("✅ Streaming session started")
		defer streamingSession.Close()
	}

	fmt.Println("\n✨ Example completed successfully!")
	fmt.Println("\nNext steps:")
	fmt.Println("- Use a real noise cancellation provider (rnnoise)")
	fmt.Println("- Configure aggressiveness and quality settings")
	fmt.Println("- Integrate with audio capture for real-time noise reduction")
}
