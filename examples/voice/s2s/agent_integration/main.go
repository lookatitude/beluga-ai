// Example: S2S with Agent Integration
// This example demonstrates S2S provider integration with Beluga AI agents for external reasoning mode.
// When external reasoning is enabled, audio is routed through agents for custom reasoning logic.
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/agents"
	"github.com/lookatitude/beluga-ai/pkg/agents/iface"
	"github.com/lookatitude/beluga-ai/pkg/core"
	llmsiface "github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/voice/s2s"
	"github.com/lookatitude/beluga-ai/pkg/voice/session"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	// Step 1: Create S2S provider with external reasoning mode
	s2sConfig := s2s.DefaultConfig()
	s2sConfig.Provider = "amazon_nova"
	s2sConfig.APIKey = os.Getenv("AWS_ACCESS_KEY_ID")
	s2sConfig.ReasoningMode = "external" // Enable external reasoning

	s2sProvider, err := s2s.NewProvider(ctx, s2sConfig.Provider, s2sConfig)
	if err != nil {
		log.Fatalf("Failed to create S2S provider: %v", err)
	}
	fmt.Println("✓ S2S provider created:", s2sProvider.Name())
	fmt.Println("✓ Reasoning mode: external")

	// Step 2: Create a streaming-compatible LLM
	llm := &mockStreamingLLM{}

	// Step 3: Create streaming agent
	agent, err := agents.NewBaseAgent("s2s-voice-assistant", llm, nil,
		agents.WithStreaming(true),
		agents.WithStreamingConfig(iface.StreamingConfig{
			EnableStreaming:     true,
			ChunkBufferSize:     20,
			SentenceBoundary:    true,
			InterruptOnNewInput: true,
			MaxStreamDuration:   30 * time.Minute,
		}),
	)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	// Cast to StreamingAgent interface
	streamingAgent, ok := agent.(iface.StreamingAgent)
	if !ok {
		log.Fatal("Agent does not implement StreamingAgent")
	}
	fmt.Println("✓ Streaming agent created")

	// Step 4: Create agent config
	agentConfig := &schema.AgentConfig{
		Name:            "s2s-voice-assistant",
		LLMProviderName: "mock",
	}

	// Step 5: Create voice session with S2S provider and agent
	// When both S2S provider and agent are present, external reasoning mode is enabled
	voiceSession, err := session.NewVoiceSession(ctx,
		session.WithS2SProvider(s2sProvider),
		session.WithAgentInstance(streamingAgent, agentConfig),
		session.WithConfig(session.DefaultConfig()),
	)
	if err != nil {
		log.Fatalf("Failed to create voice session: %v", err)
	}
	fmt.Println("✓ Voice session created with S2S + Agent integration")

	// Step 6: Start session
	fmt.Println("\nStarting voice session with S2S and external reasoning...")
	err = voiceSession.Start(ctx)
	if err != nil {
		log.Fatalf("Failed to start session: %v", err)
	}
	defer voiceSession.Stop(ctx)
	fmt.Println("✓ Session started")

	// Step 7: Process audio
	// Audio will be processed through S2S provider, then routed through agent for reasoning
	fmt.Println("\nProcessing audio through S2S with external reasoning...")
	audio := []byte{1, 2, 3, 4, 5}
	err = voiceSession.ProcessAudio(ctx, audio)
	if err != nil {
		log.Printf("Error processing audio: %v", err)
	} else {
		fmt.Println("✓ Audio processed with external reasoning")
	}

	// Step 8: Say something (this will also use S2S + agent)
	handle, err := voiceSession.Say(ctx, "Hello! I'm using S2S with external reasoning.")
	if err != nil {
		log.Printf("Error saying text: %v", err)
	} else {
		err = handle.WaitForPlayout(ctx)
		if err != nil {
			log.Printf("Error waiting for playout: %v", err)
		}
	}

	// Keep running until interrupted
	fmt.Println("\nSession running. Press Ctrl+C to stop...")
	<-ctx.Done()
	fmt.Println("\nShutting down...")
}

// Mock implementations for demonstration
type mockStreamingLLM struct{}

func (m *mockStreamingLLM) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	return "Mock response from agent", nil
}

func (m *mockStreamingLLM) GetModelName() string {
	return "mock-model"
}

func (m *mockStreamingLLM) GetProviderName() string {
	return "mock-provider"
}

func (m *mockStreamingLLM) StreamChat(ctx context.Context, messages []schema.Message, options ...interface{}) (<-chan llmsiface.AIMessageChunk, error) {
	ch := make(chan llmsiface.AIMessageChunk, 10)
	go func() {
		defer close(ch)
		chunks := []string{"Hello", "!", "I'm", "using", "S2S", "with", "external", "reasoning", "."}
		for _, content := range chunks {
			ch <- llmsiface.AIMessageChunk{Content: content}
			time.Sleep(10 * time.Millisecond)
		}
	}()
	return ch, nil
}
