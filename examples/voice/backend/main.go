package main

import (
	"context"
	"fmt"
	"log"
	"os"

	_ "github.com/lookatitude/beluga-ai/pkg/voice/backend/providers/livekit"
	vbiface "github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/backend"
)

func main() {
	fmt.Println("🏗️  Beluga AI - Voice Backend Example")
	fmt.Println("=====================================")

	ctx := context.Background()

	// Step 1: Create backend configuration
	fmt.Println("\n📋 Step 1: Creating backend configuration...")
	apiKey := os.Getenv("LIVEKIT_API_KEY")
	apiSecret := os.Getenv("LIVEKIT_API_SECRET")
	url := os.Getenv("LIVEKIT_URL") // e.g., "wss://your-livekit-server.com"
	if apiKey == "" || apiSecret == "" || url == "" {
		log.Println("⚠️  Note: LIVEKIT_API_KEY, LIVEKIT_API_SECRET, and LIVEKIT_URL environment variables are required for real usage")
		log.Println("Using placeholder values for demonstration...")
		apiKey = "placeholder-key"
		apiSecret = "placeholder-secret"
		url = "wss://localhost:7880"
	}
	config := &vbiface.Config{
		Provider:     "livekit",
		PipelineType: vbiface.PipelineTypeSTTTTS,
		STTProvider:  "mock",
		TTSProvider:  "mock",
		ProviderConfig: map[string]any{
			"api_key":    apiKey,
			"api_secret": apiSecret,
			"url":        url,
		},
	}
	fmt.Println("✅ Configuration created")

	// Step 2: Create backend instance
	fmt.Println("\n📋 Step 2: Creating backend instance...")
	backendInstance, err := backend.NewBackend(ctx, "livekit", config)
	if err != nil {
		log.Fatalf("Failed to create backend: %v", err)
	}
	fmt.Printf("✅ Backend created\n")

	// Step 3: Create a session
	fmt.Println("\n📋 Step 3: Creating session...")
	sessionConfig := &vbiface.SessionConfig{
		UserID:        "example-user",
		Transport:     "webrtc",
		ConnectionURL: url,
		PipelineType:  vbiface.PipelineTypeSTTTTS,
	}
	session, err := backendInstance.CreateSession(ctx, sessionConfig)
	if err != nil {
		log.Printf("Note: Session creation may require a running LiveKit server: %v", err)
	} else {
		fmt.Println("✅ Session created")
		defer func() {
			if err := session.Stop(ctx); err != nil {
				log.Printf("Error stopping session: %v", err)
			}
		}()
	}

	// Step 4: Start session (optional)
	fmt.Println("\n📋 Step 4: Starting session...")
	if session != nil {
		if err := session.Start(ctx); err != nil {
			log.Printf("Note: Session start may require a running LiveKit server: %v", err)
		} else {
			fmt.Println("✅ Session started")
		}
	}

	fmt.Println("\n✨ Example completed successfully!")
	fmt.Println("\nNext steps:")
	fmt.Println("- Set up a LiveKit server")
	fmt.Println("- Configure API keys and URL")
	fmt.Println("- Create rooms and manage participants")
	fmt.Println("- Handle audio tracks and WebRTC connections")
}
