package main

import (
	"context"
	"fmt"
	"log"

	// Mock provider not available - remove blank import
	"github.com/lookatitude/beluga-ai/pkg/voice/stt"
)

func main() {
	fmt.Println("🎤 Beluga AI - STT (Speech-to-Text) Example")
	fmt.Println("============================================")

	ctx := context.Background()

	// Step 1: Create STT configuration
	fmt.Println("\n📋 Step 1: Creating STT configuration...")
	config := stt.DefaultConfig()
	config.Provider = "mock" // Use mock provider for this example
	// In production, use: "deepgram", "azure", "google", "openai", "whisper"
	fmt.Println("✅ Configuration created")

	// Step 2: Create STT provider
	fmt.Println("\n📋 Step 2: Creating STT provider...")
	provider, err := stt.NewProvider(ctx, config.Provider, config)
	if err != nil {
		log.Fatalf("Failed to create STT provider: %v", err)
	}
	fmt.Printf("✅ Provider created: %s\n", config.Provider)

	// Step 3: Transcribe audio
	fmt.Println("\n📋 Step 3: Transcribing audio...")
	// In a real application, this would be actual audio data from a microphone or file
	audioData := []byte{1, 2, 3, 4, 5} // Placeholder audio data
	transcript, err := provider.Transcribe(ctx, audioData)
	if err != nil {
		log.Fatalf("Failed to transcribe audio: %v", err)
	}
	fmt.Printf("✅ Transcript: %s\n", transcript)

	// Step 4: Start streaming session (optional)
	fmt.Println("\n📋 Step 4: Starting streaming session...")
	streamingSession, err := provider.StartStreaming(ctx)
	if err != nil {
		log.Printf("Note: Streaming not available with mock provider: %v", err)
	} else {
		fmt.Println("✅ Streaming session started")
		defer streamingSession.Close()
	}

	fmt.Println("\n✨ Example completed successfully!")
	fmt.Println("\nNext steps:")
	fmt.Println("- Use a real STT provider (deepgram, azure, google, etc.)")
	fmt.Println("- Configure API keys and language settings")
	fmt.Println("- Process real audio streams from microphones or files")
}
