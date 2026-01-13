package main

import (
	"context"
	"fmt"
	"log"

	// Mock provider not available - remove blank import
	"github.com/lookatitude/beluga-ai/pkg/voice/vad"
)

func main() {
	fmt.Println("🎯 Beluga AI - VAD (Voice Activity Detection) Example")
	fmt.Println("======================================================")

	ctx := context.Background()

	// Step 1: Create VAD configuration
	fmt.Println("\n📋 Step 1: Creating VAD configuration...")
	config := vad.DefaultConfig()
	config.Provider = "mock" // Use mock provider for this example
	// In production, use: "webrtc", "silero", etc.
	fmt.Println("✅ Configuration created")

	// Step 2: Create VAD provider
	fmt.Println("\n📋 Step 2: Creating VAD provider...")
	provider, err := vad.NewProvider(ctx, config.Provider, config)
	if err != nil {
		log.Fatalf("Failed to create VAD provider: %v", err)
	}
	fmt.Printf("✅ Provider created: %s\n", config.Provider)

	// Step 3: Detect voice activity in audio frame
	fmt.Println("\n📋 Step 3: Detecting voice activity...")
	// In a real application, this would be actual audio frame data
	audioFrame := []byte{1, 2, 3, 4, 5} // Placeholder audio frame
	isSpeech, err := provider.Process(ctx, audioFrame)
	if err != nil {
		log.Fatalf("Failed to detect voice activity: %v", err)
	}
	if isSpeech {
		fmt.Println("✅ Speech detected in audio frame")
	} else {
		fmt.Println("✅ No speech detected (silence or noise)")
	}

	// Step 4: Process streaming audio (optional)
	fmt.Println("\n📋 Step 4: Processing streaming audio...")
	audioCh := make(chan []byte, 1)
	audioCh <- audioFrame
	close(audioCh)
	resultsCh, err := provider.ProcessStream(ctx, audioCh)
	if err != nil {
		log.Printf("Note: Streaming not available with mock provider: %v", err)
	} else {
		fmt.Println("✅ Streaming session started")
		for result := range resultsCh {
			fmt.Printf("   VAD result: HasVoice=%v, Confidence=%.2f\n", result.HasVoice, result.Confidence)
		}
	}

	fmt.Println("\n✨ Example completed successfully!")
	fmt.Println("\nNext steps:")
	fmt.Println("- Use a real VAD provider (webrtc, silero)")
	fmt.Println("- Configure frame size and silence thresholds")
	fmt.Println("- Integrate with audio streams for real-time detection")
}
