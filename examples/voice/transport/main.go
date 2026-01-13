package main

import (
	"context"
	"fmt"
	"log"

	// Mock provider not available - remove blank import
	"github.com/lookatitude/beluga-ai/pkg/voice/transport"
)

func main() {
	fmt.Println("📡 Beluga AI - Transport Example")
	fmt.Println("=================================")

	ctx := context.Background()

	// Step 1: Create Transport configuration
	fmt.Println("\n📋 Step 1: Creating Transport configuration...")
	config := transport.DefaultConfig()
	config.Provider = "mock" // Use mock provider for this example
	// In production, use: "websocket", "webrtc", etc.
	fmt.Println("✅ Configuration created")

	// Step 2: Create Transport provider
	fmt.Println("\n📋 Step 2: Creating Transport provider...")
	provider, err := transport.NewProvider(ctx, config.Provider, config)
	if err != nil {
		log.Fatalf("Failed to create Transport provider: %v", err)
	}
	fmt.Printf("✅ Provider created: %s\n", config.Provider)

	// Step 3: Send audio data
	fmt.Println("\n📋 Step 3: Sending audio data...")
	audioData := []byte{1, 2, 3, 4, 5} // Placeholder audio data
	err = provider.SendAudio(ctx, audioData)
	if err != nil {
		log.Fatalf("Failed to send audio: %v", err)
	}
	fmt.Println("✅ Audio data sent successfully")

	// Step 4: Receive audio data (optional)
	fmt.Println("\n📋 Step 4: Receiving audio data...")
	// Set up callback for received audio
	provider.OnAudioReceived(func(audio []byte) {
		fmt.Printf("✅ Received audio data via callback: %d bytes\n", len(audio))
	})
	// Also get the receive channel
	audioCh := provider.ReceiveAudio()
	select {
	case receivedAudio := <-audioCh:
		fmt.Printf("✅ Received audio data via channel: %d bytes\n", len(receivedAudio))
	default:
		fmt.Println("✅ Receive channel set up (no data available yet)")
	}

	// Step 5: Close connection
	fmt.Println("\n📋 Step 5: Closing connection...")
	err = provider.Close()
	if err != nil {
		log.Printf("Error closing connection: %v", err)
	} else {
		fmt.Println("✅ Connection closed")
	}

	fmt.Println("\n✨ Example completed successfully!")
	fmt.Println("\nNext steps:")
	fmt.Println("- Use a real transport provider (websocket, webrtc)")
	fmt.Println("- Configure connection URLs and authentication")
	fmt.Println("- Handle bidirectional audio streaming")
}
