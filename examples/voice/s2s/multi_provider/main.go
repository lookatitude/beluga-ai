package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/s2s"
	s2siface "github.com/lookatitude/beluga-ai/pkg/voice/s2s/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/session"
)

// This example demonstrates multi-provider S2S configuration with automatic fallback.
// When the primary provider fails, the system automatically falls back to alternative providers.
func main() {
	ctx := context.Background()

	// Step 1: Create primary S2S provider
	primaryConfig := s2s.DefaultConfig()
	primaryConfig.Provider = "amazon_nova"
	primaryConfig.APIKey = os.Getenv("AWS_ACCESS_KEY_ID")

	primaryProvider, err := s2s.NewProvider(ctx, primaryConfig.Provider, primaryConfig)
	if err != nil {
		log.Fatalf("Failed to create primary provider: %v", err)
	}
	fmt.Println("✓ Primary provider created:", primaryProvider.Name())

	// Step 2: Create fallback providers
	fallbackConfigs := []struct {
		name   string
		apiKey string
	}{
		{"grok", os.Getenv("XAI_API_KEY")},
		{"gemini", os.Getenv("GOOGLE_API_KEY")},
		{"openai_realtime", os.Getenv("OPENAI_API_KEY")},
	}

	var fallbackProviders []s2siface.S2SProvider
	for _, fbConfig := range fallbackConfigs {
		if fbConfig.apiKey == "" {
			fmt.Printf("⚠ Skipping %s (no API key)\n", fbConfig.name)
			continue
		}

		config := s2s.DefaultConfig()
		config.Provider = fbConfig.name
		config.APIKey = fbConfig.apiKey

		provider, err := s2s.NewProvider(ctx, config.Provider, config)
		if err != nil {
			log.Printf("Failed to create fallback provider %s: %v", fbConfig.name, err)
			continue
		}
		fallbackProviders = append(fallbackProviders, provider)
		fmt.Printf("✓ Fallback provider created: %s\n", provider.Name())
	}

	// Step 3: Create provider manager with fallback support
	manager, err := s2s.NewProviderManager(primaryProvider, fallbackProviders)
	if err != nil {
		log.Fatalf("Failed to create provider manager: %v", err)
	}
	fmt.Printf("✓ Provider manager created with %d fallback(s)\n", len(fallbackProviders))

	// Step 4: Create voice session with provider manager
	// Note: We need to use the primary provider for the session,
	// but the manager handles fallback internally
	voiceSession, err := session.NewVoiceSession(ctx,
		session.WithS2SProvider(primaryProvider),
		session.WithConfig(session.DefaultConfig()),
	)
	if err != nil {
		log.Fatalf("Failed to create voice session: %v", err)
	}
	fmt.Println("✓ Voice session created with fallback support")

	// Step 5: Start the session
	err = voiceSession.Start(ctx)
	if err != nil {
		log.Fatalf("Failed to start session: %v", err)
	}
	fmt.Println("✓ Session started")

	// Step 6: Process audio
	// If primary provider fails, the system will automatically try fallback providers
	audioData := []byte{1, 2, 3, 4, 5}
	fmt.Println("\nProcessing audio (with automatic fallback if needed)...")
	err = voiceSession.ProcessAudio(ctx, audioData)
	if err != nil {
		log.Printf("Error processing audio: %v", err)
	} else {
		fmt.Println("✓ Audio processed successfully")
		fmt.Printf("Current provider: %s\n", manager.GetCurrentProviderName())
		if manager.IsUsingFallback() {
			fmt.Println("⚠ Using fallback provider")
		}
	}

	// Step 7: Wait and stop
	time.Sleep(1 * time.Second)
	err = voiceSession.Stop(ctx)
	if err != nil {
		log.Fatalf("Failed to stop session: %v", err)
	}
	fmt.Println("✓ Session stopped")

	fmt.Println("\nExample completed successfully!")
	fmt.Println("\nKey features demonstrated:")
	fmt.Println("- Multi-provider configuration")
	fmt.Println("- Automatic fallback on failure")
	fmt.Println("- Provider manager for fallback coordination")
}
