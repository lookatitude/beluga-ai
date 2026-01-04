// Example: Voice agent with tools
// This example demonstrates a voice session with a streaming agent that can execute tools.
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
	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
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

	// Create tools for the agent
	calculator := &mockCalculatorTool{}
	agentTools := []tools.Tool{calculator}

	// Create streaming LLM
	llm := &mockStreamingLLM{}

	// Create streaming agent with tools
	agent, err := agents.NewBaseAgent("voice-assistant-with-tools", llm, agentTools,
		agents.WithStreaming(true),
	)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	streamingAgent, ok := agent.(iface.StreamingAgent)
	if !ok {
		log.Fatal("Agent does not implement StreamingAgent")
	}

	// Create voice session with agent instance
	voiceSession, err := session.NewVoiceSession(ctx,
		session.WithSTTProvider(sttProvider),
		session.WithTTSProvider(ttsProvider),
		session.WithAgentInstance(streamingAgent, &schema.AgentConfig{
			Name:            "voice-assistant-with-tools",
			LLMProviderName: "mock",
		}),
	)
	if err != nil {
		log.Fatalf("Failed to create voice session: %v", err)
	}

	fmt.Println("Starting voice session with agent and tools...")
	err = voiceSession.Start(ctx)
	if err != nil {
		log.Fatalf("Failed to start session: %v", err)
	}
	defer voiceSession.Stop(ctx)

	fmt.Println("Voice session started. Agent can now execute tools during conversations.")

	// Simulate audio input
	audio := []byte{1, 2, 3, 4, 5}
	err = voiceSession.ProcessAudio(ctx, audio)
	if err != nil {
		log.Printf("Error processing audio: %v", err)
	}

	<-ctx.Done()
	fmt.Println("Shutting down...")
}

// Mock calculator tool
type mockCalculatorTool struct{}

func (m *mockCalculatorTool) Name() string {
	return "calculator"
}

func (m *mockCalculatorTool) Description() string {
	return "Performs basic arithmetic calculations"
}

func (m *mockCalculatorTool) Definition() tools.ToolDefinition {
	return tools.ToolDefinition{
		Name:        "calculator",
		Description: "Performs basic arithmetic calculations",
	}
}

func (m *mockCalculatorTool) Execute(ctx context.Context, input any) (any, error) {
	// Mock calculation
	return "42", nil
}

func (m *mockCalculatorTool) Batch(ctx context.Context, inputs []any) ([]any, error) {
	results := make([]any, len(inputs))
	for i := range inputs {
		results[i] = "42"
	}
	return results, nil
}

// Mock providers
type mockSTTProvider struct{}

func (m *mockSTTProvider) Transcribe(ctx context.Context, audio []byte) (string, error) {
	return "Calculate 2 plus 2", nil
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
	ch <- voiceiface.TranscriptResult{Text: "Calculate 2 plus 2", IsFinal: true}
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
		chunks := []string{"The", "answer", "is", "4", "."}
		for _, content := range chunks {
			ch <- llmsiface.AIMessageChunk{Content: content}
			time.Sleep(10 * time.Millisecond)
		}
	}()
	return ch, nil
}
