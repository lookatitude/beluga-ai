// Example: Multiple concurrent voice agents
// This example demonstrates running multiple voice sessions with different agent instances concurrently.
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
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

	// Create multiple agent instances
	agentInstances := make([]iface.StreamingAgent, 3)
	for i := 0; i < 3; i++ {
		llm := &mockStreamingLLM{id: i}
		agent, err := agents.NewBaseAgent(
			fmt.Sprintf("agent-%d", i),
			llm,
			nil,
			agents.WithStreaming(true),
		)
		if err != nil {
			log.Fatalf("Failed to create agent %d: %v", i, err)
		}
		streamingAgent, ok := agent.(iface.StreamingAgent)
		if !ok {
			log.Fatalf("Agent %d does not implement StreamingAgent", i)
		}
		agentInstances[i] = streamingAgent
	}

	// Create mock providers (shared across sessions)
	sttProvider := &mockSTTProvider{}
	ttsProvider := &mockTTSProvider{}

	// Create multiple voice sessions concurrently
	var wg sync.WaitGroup
	for i, agent := range agentInstances {
		wg.Add(1)
		go func(id int, ag iface.StreamingAgent) {
			defer wg.Done()

			voiceSession, err := session.NewVoiceSession(ctx,
				session.WithSTTProvider(sttProvider),
				session.WithTTSProvider(ttsProvider),
				session.WithAgentInstance(ag, &schema.AgentConfig{
					Name:            fmt.Sprintf("agent-%d", id),
					LLMProviderName: "mock",
				}),
			)
			if err != nil {
				log.Printf("Failed to create session %d: %v", id, err)
				return
			}

			fmt.Printf("Starting session %d...\n", id)
			err = voiceSession.Start(ctx)
			if err != nil {
				log.Printf("Failed to start session %d: %v", id, err)
				return
			}
			defer voiceSession.Stop(ctx)

			// Simulate concurrent audio processing
			audio := []byte{byte(id), 2, 3, 4, 5}
			err = voiceSession.ProcessAudio(ctx, audio)
			if err != nil {
				log.Printf("Error processing audio in session %d: %v", id, err)
			}

			fmt.Printf("Session %d processing audio...\n", id)
		}(i, agent)
	}

	fmt.Println("All sessions started concurrently. Waiting...")
	wg.Wait()

	<-ctx.Done()
	fmt.Println("Shutting down all sessions...")
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

type mockStreamingLLM struct {
	id int
}

func (m *mockStreamingLLM) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	return fmt.Sprintf("Mock response from agent %d", m.id), nil
}

func (m *mockStreamingLLM) GetModelName() string {
	return fmt.Sprintf("mock-model-%d", m.id)
}

func (m *mockStreamingLLM) GetProviderName() string {
	return "mock-provider"
}

func (m *mockStreamingLLM) StreamChat(ctx context.Context, messages []schema.Message, options ...interface{}) (<-chan llmsiface.AIMessageChunk, error) {
	ch := make(chan llmsiface.AIMessageChunk, 10)
	go func() {
		defer close(ch)
		chunks := []string{fmt.Sprintf("Agent%d", m.id), "says", "hello", "!"}
		for _, content := range chunks {
			ch <- llmsiface.AIMessageChunk{Content: content}
			time.Sleep(10 * time.Millisecond)
		}
	}()
	return ch, nil
}
