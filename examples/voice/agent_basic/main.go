// Example: Basic voice agent with streaming support
// This example demonstrates creating a voice session with a streaming agent instance.
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

	// Create mock providers (in real usage, use actual providers)
	sttProvider := &mockSTTProvider{}
	ttsProvider := &mockTTSProvider{}

	// Create a streaming-compatible LLM (must implement ChatModel interface)
	llm := &mockStreamingLLM{}

	// Create streaming agent
	agent, err := agents.NewBaseAgent("voice-assistant", llm, nil,
		agents.WithStreaming(true),
		agents.WithStreamingConfig(iface.StreamingConfig{
			EnableStreaming:      true,
			ChunkBufferSize:      20,
			SentenceBoundary:     true,
			InterruptOnNewInput:  true,
			MaxStreamDuration:    30 * time.Minute,
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

	// Create agent config
	agentConfig := &schema.AgentConfig{
		Name:            "voice-assistant",
		LLMProviderName: "mock",
	}

	// Create voice session with agent instance
	voiceSession, err := session.NewVoiceSession(ctx,
		session.WithSTTProvider(sttProvider),
		session.WithTTSProvider(ttsProvider),
		session.WithAgentInstance(streamingAgent, agentConfig),
		session.WithConfig(session.DefaultConfig()),
	)
	if err != nil {
		log.Fatalf("Failed to create voice session: %v", err)
	}

	// Start session
	fmt.Println("Starting voice session with streaming agent...")
	err = voiceSession.Start(ctx)
	if err != nil {
		log.Fatalf("Failed to start session: %v", err)
	}
	defer voiceSession.Stop(ctx)

	fmt.Println("Voice session started. Say something...")

	// Simulate audio input
	audio := []byte{1, 2, 3, 4, 5}
	err = voiceSession.ProcessAudio(ctx, audio)
	if err != nil {
		log.Printf("Error processing audio: %v", err)
	}

	// Keep running until interrupted
	<-ctx.Done()
	fmt.Println("Shutting down...")
}

// Mock implementations
type mockSTTProvider struct{}

func (m *mockSTTProvider) Transcribe(ctx context.Context, audio []byte) (string, error) {
	return "Hello, how are you?", nil
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
		chunks := []string{"Hello", "!", "How", "can", "I", "help", "?"}
		for _, content := range chunks {
			ch <- llmsiface.AIMessageChunk{Content: content}
			time.Sleep(10 * time.Millisecond)
		}
	}()
	return ch, nil
}
