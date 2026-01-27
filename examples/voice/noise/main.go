package main

import (
	"context"
	"fmt"
	"log"

	// Mock provider not available - remove blank import
	"github.com/lookatitude/beluga-ai/pkg/noisereduction"
)

func main() {
	fmt.Println("🔇 Beluga AI - Noise Cancellation Example")
	fmt.Println("==========================================")

	ctx := context.Background()

	// Step 1: Create Noise Cancellation configuration
	fmt.Println("\n📋 Step 1: Creating Noise Cancellation configuration...")
	config := noisereduction.DefaultConfig()
	config.Provider = "mock" // Use mock provider for this example
	// In production, use: "rnnoise", etc.
	fmt.Println("✅ Configuration created")

	// Step 2: Create Noise Cancellation provider
	fmt.Println("\n📋 Step 2: Creating Noise Cancellation provider...")
	provider, err := noisereduction.NewProvider(ctx, config.Provider, config)
	if err != nil {
		log.Fatalf("Failed to create Noise Cancellation provider: %v", err)
	}
	fmt.Printf("✅ Provider created: %s\n", config.Provider)

	// Step 3: Cancel noise from audio data
	fmt.Println("\n📋 Step 3: Canceling noise from audio...")
	// In a real application, this would be actual audio data with background noise
	audioData := []byte{1, 2, 3, 4, 5} // Placeholder audio data
	cleanedAudio, err := provider.Process(ctx, audioData)
	if err != nil {
		log.Fatalf("Failed to process noise: %v", err)
	}
	fmt.Printf("✅ Noise processed: %d bytes input, %d bytes output\n", len(audioData), len(cleanedAudio))

	// Step 4: Process streaming audio (optional)
	fmt.Println("\n📋 Step 4: Processing streaming audio...")
	audioCh := make(chan []byte, 1)
	audioCh <- audioData
	close(audioCh)
	cleanedCh, err := provider.ProcessStream(ctx, audioCh)
	if err != nil {
		log.Printf("Note: Streaming not available with mock provider: %v", err)
	} else {
		fmt.Println("✅ Streaming session started")
		for cleaned := range cleanedCh {
			fmt.Printf("   Processed chunk: %d bytes\n", len(cleaned))
		}
	}

	fmt.Println("\n✨ Example completed successfully!")
	fmt.Println("\nNext steps:")
	fmt.Println("- Use a real noise cancellation provider (rnnoise)")
	fmt.Println("- Configure aggressiveness and quality settings")
	fmt.Println("- Integrate with audio capture for real-time noise reduction")
}
