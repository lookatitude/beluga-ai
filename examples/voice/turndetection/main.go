package main

import (
	"context"
	"fmt"
	"log"
	"time"

	// Mock provider not available - remove blank import
	"github.com/lookatitude/beluga-ai/pkg/voice/turndetection"
)

func main() {
	fmt.Println("🔄 Beluga AI - Turn Detection Example")
	fmt.Println("======================================")

	ctx := context.Background()

	// Step 1: Create Turn Detection configuration
	fmt.Println("\n📋 Step 1: Creating Turn Detection configuration...")
	config := turndetection.DefaultConfig()
	config.Provider = "mock" // Use mock provider for this example
	// In production, use: "silence", "energy", etc.
	fmt.Println("✅ Configuration created")

	// Step 2: Create Turn Detection provider
	fmt.Println("\n📋 Step 2: Creating Turn Detection provider...")
	provider, err := turndetection.NewProvider(ctx, config.Provider, config)
	if err != nil {
		log.Fatalf("Failed to create Turn Detection provider: %v", err)
	}
	fmt.Printf("✅ Provider created: %s\n", config.Provider)

	// Step 3: Detect turn completion in audio stream
	fmt.Println("\n📋 Step 3: Detecting turn completion...")
	// In a real application, this would be actual audio stream data
	audioStream := []byte{1, 2, 3, 4, 5} // Placeholder audio stream
	isTurnComplete, err := provider.DetectTurn(ctx, audioStream)
	if err != nil {
		log.Fatalf("Failed to detect turn: %v", err)
	}
	if isTurnComplete {
		fmt.Println("✅ Turn detected - speaker has finished speaking")
	} else {
		fmt.Println("✅ Turn not complete - speaker is still speaking")
	}

	// Step 4: Additional turn detection with silence duration (optional)
	fmt.Println("\n📋 Step 4: Detecting turn with silence duration...")
	// Note: TurnDetector doesn't have streaming, but has DetectTurnWithSilence
	isTurnCompleteWithSilence, err := provider.DetectTurnWithSilence(ctx, audioStream, 2*time.Second)
	if err != nil {
		log.Printf("Note: DetectTurnWithSilence not available with mock provider: %v", err)
	} else {
		if isTurnCompleteWithSilence {
			fmt.Println("✅ Turn detected with silence duration")
		} else {
			fmt.Println("✅ Turn not complete (silence threshold not met)")
		}
	}

	fmt.Println("\n✨ Example completed successfully!")
	fmt.Println("\nNext steps:")
	fmt.Println("- Use a real turn detection provider (silence-based, energy-based)")
	fmt.Println("- Configure silence duration and minimum speech duration")
	fmt.Println("- Integrate with voice sessions for natural conversation flow")
}
