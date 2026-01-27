package main

import (
	"context"
	"fmt"
	"log"

	// Mock provider not available - remove blank import
	"github.com/lookatitude/beluga-ai/pkg/tts"
)

func main() {
	fmt.Println("🔊 Beluga AI - TTS (Text-to-Speech) Example")
	fmt.Println("===========================================")

	ctx := context.Background()

	// Step 1: Create TTS configuration
	fmt.Println("\n📋 Step 1: Creating TTS configuration...")
	config := tts.DefaultConfig()
	config.Provider = "mock" // Use mock provider for this example
	// In production, use: "deepgram", "azure", "google", "openai", "elevenlabs"
	fmt.Println("✅ Configuration created")

	// Step 2: Create TTS provider
	fmt.Println("\n📋 Step 2: Creating TTS provider...")
	provider, err := tts.NewProvider(ctx, config.Provider, config)
	if err != nil {
		log.Fatalf("Failed to create TTS provider: %v", err)
	}
	fmt.Printf("✅ Provider created: %s\n", config.Provider)

	// Step 3: Generate speech from text
	fmt.Println("\n📋 Step 3: Generating speech...")
	text := "Hello! This is a text-to-speech example using Beluga AI."
	audioData, err := provider.GenerateSpeech(ctx, text)
	if err != nil {
		log.Fatalf("Failed to generate speech: %v", err)
	}
	fmt.Printf("✅ Generated audio data: %d bytes\n", len(audioData))

	// Step 4: Start streaming session (optional)
	fmt.Println("\n📋 Step 4: Starting streaming session...")
	streamReader, err := provider.StreamGenerate(ctx, "This is a streaming test.")
	if err != nil {
		log.Printf("Note: Streaming not available with mock provider: %v", err)
	} else {
		fmt.Println("✅ Streaming session started")
		_ = streamReader
	}

	fmt.Println("\n✨ Example completed successfully!")
	fmt.Println("\nNext steps:")
	fmt.Println("- Use a real TTS provider (deepgram, azure, google, etc.)")
	fmt.Println("- Configure API keys and voice settings")
	fmt.Println("- Stream audio to speakers or save to files")
}
