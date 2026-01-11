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
	config := &vbiface.Config{
		APIKey:    os.Getenv("LIVEKIT_API_KEY"),
		APISecret: os.Getenv("LIVEKIT_API_SECRET"),
		URL:       os.Getenv("LIVEKIT_URL"), // e.g., "wss://your-livekit-server.com"
	}
	if config.APIKey == "" || config.APISecret == "" || config.URL == "" {
		log.Println("⚠️  Note: LIVEKIT_API_KEY, LIVEKIT_API_SECRET, and LIVEKIT_URL environment variables are required for real usage")
		log.Println("Using placeholder values for demonstration...")
		config.APIKey = "placeholder-key"
		config.APISecret = "placeholder-secret"
		config.URL = "wss://localhost:7880"
	}
	fmt.Println("✅ Configuration created")

	// Step 2: Create backend instance
	fmt.Println("\n📋 Step 2: Creating backend instance...")
	backendInstance, err := backend.NewBackend(ctx, "livekit", config)
	if err != nil {
		log.Fatalf("Failed to create backend: %v", err)
	}
	fmt.Printf("✅ Backend created: %s\n", backendInstance.Name())

	// Step 3: Create a session
	fmt.Println("\n📋 Step 3: Creating session...")
	roomName := "example-room"
	sessionConfig := &vbiface.SessionConfig{
		RoomName: roomName,
	}
	session, err := backendInstance.CreateSession(ctx, sessionConfig)
	if err != nil {
		log.Printf("Note: Session creation may require a running LiveKit server: %v", err)
	} else {
		fmt.Printf("✅ Session created: %s\n", session.ID())
		defer session.Close()
	}

	// Step 4: Get session info
	fmt.Println("\n📋 Step 4: Getting session information...")
	if session != nil {
		info := session.Info()
		fmt.Printf("✅ Session info: %+v\n", info)
	}

	fmt.Println("\n✨ Example completed successfully!")
	fmt.Println("\nNext steps:")
	fmt.Println("- Set up a LiveKit server")
	fmt.Println("- Configure API keys and URL")
	fmt.Println("- Create rooms and manage participants")
	fmt.Println("- Handle audio tracks and WebRTC connections")
}
