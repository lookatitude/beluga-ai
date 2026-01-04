// Example: Voice agent with custom streaming configuration
// This example demonstrates configuring a voice session with custom streaming settings.
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
	voiceiface "github.com/lookatitude/beluga-ai/pkg/voice/iface"
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

	// Create mock providers
	sttProvider := &mockSTTProvider{}
	ttsProvider := &mockTTSProvider{}

	// Create streaming LLM
	llm := &mockStreamingLLM{}

	// Create agent with custom streaming configuration
	// Custom config: larger buffer, no sentence boundaries for faster response
	agent, err := agents.NewBaseAgent("voice-assistant-custom", llm, nil,
		agents.WithStreamingConfig(iface.StreamingConfig{
			EnableStreaming:     true,
			ChunkBufferSize:     50,               // Larger buffer for high-throughput scenarios
			SentenceBoundary:    false,            // Disable for lower latency
			InterruptOnNewInput: true,             // Enable interruption
			MaxStreamDuration:   15 * time.Minute, // Shorter max duration
		}),
	)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	streamingAgent, ok := agent.(iface.StreamingAgent)
	if !ok {
		log.Fatal("Agent does not implement StreamingAgent")
	}

	// Create voice session with custom agent config
	voiceSession, err := session.NewVoiceSession(ctx,
		session.WithSTTProvider(sttProvider),
		session.WithTTSProvider(ttsProvider),
		session.WithAgentInstance(streamingAgent, &schema.AgentConfig{
			Name:            "voice-assistant-custom",
			LLMProviderName: "mock",
		}),
		session.WithConfig(&session.Config{
			Timeout:           15 * time.Minute, // Match agent max stream duration
			KeepAliveInterval: 30 * time.Second,
			MaxRetries:        3,
		}),
	)
	if err != nil {
		log.Fatalf("Failed to create voice session: %v", err)
	}

	fmt.Println("Starting voice session with custom streaming configuration...")
	fmt.Println("Configuration: Buffer=50, No sentence boundaries, Interruption enabled")
	err = voiceSession.Start(ctx)
	if err != nil {
		log.Fatalf("Failed to start session: %v", err)
	}
	defer voiceSession.Stop(ctx)

	fmt.Println("Voice session started with custom config.")

	// Simulate audio input
	audio := []byte{1, 2, 3, 4, 5}
	err = voiceSession.ProcessAudio(ctx, audio)
	if err != nil {
		log.Printf("Error processing audio: %v", err)
	}

	<-ctx.Done()
	fmt.Println("Shutting down...")
}

// Mock providers
type mockSTTProvider struct{}

func (m *mockSTTProvider) Transcribe(ctx context.Context, audio []byte) (string, error) {
	return "Hello", nil
}

func (m *mockSTTProvider) StartStreaming(ctx context.Context) (voiceiface.StreamingSession, error) {
	return &mockStreamingSTT{}, nil
}

type mockStreamingSTT struct{}

func (m *mockStreamingSTT) SendAudio(ctx context.Context, audio []byte) error {
	return nil
}

func (m *mockStreamingSTT) ReceiveTranscript() <-chan voiceiface.TranscriptResult {
	ch := make(chan voiceiface.TranscriptResult, 1)
	ch <- voiceiface.TranscriptResult{Text: "Hello", IsFinal: true}
	close(ch)
	return ch
}

func (m *mockStreamingSTT) Close() error {
	return nil
}

type mockTTSProvider struct{}

func (m *mockTTSProvider) GenerateSpeech(ctx context.Context, text string) ([]byte, error) {
	return []byte{1, 2, 3, 4, 5}, nil
}

func (m *mockTTSProvider) StreamGenerate(ctx context.Context, text string) (io.Reader, error) {
	return strings.NewReader(text), nil
}

type mockStreamingLLM struct{}

func (m *mockStreamingLLM) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	return "Mock response", nil
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
		chunks := []string{"Hi", "there", "!"}
		for _, content := range chunks {
			ch <- llmsiface.AIMessageChunk{Content: content}
			time.Sleep(5 * time.Millisecond) // Faster chunks for custom config
		}
	}()
	return ch, nil
}
