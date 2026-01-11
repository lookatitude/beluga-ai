package main

import (
	"context"
	"fmt"
	"log"

	_ "github.com/lookatitude/beluga-ai/pkg/voice/vad/providers/mock"
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
	isSpeech, err := provider.Detect(ctx, audioFrame)
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
	streamingSession, err := provider.StartStreaming(ctx)
	if err != nil {
		log.Printf("Note: Streaming not available with mock provider: %v", err)
	} else {
		fmt.Println("✅ Streaming session started")
		defer streamingSession.Close()
	}

	fmt.Println("\n✨ Example completed successfully!")
	fmt.Println("\nNext steps:")
	fmt.Println("- Use a real VAD provider (webrtc, silero)")
	fmt.Println("- Configure frame size and silence thresholds")
	fmt.Println("- Integrate with audio streams for real-time detection")
}
