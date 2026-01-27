package agent

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/core"
)

func TestNewBuilder(t *testing.T) {
	builder := NewBuilder()

	if builder.name != "assistant" {
		t.Errorf("expected default name 'assistant', got %s", builder.name)
	}
	if builder.maxTurns != 10 {
		t.Errorf("expected default maxTurns 10, got %d", builder.maxTurns)
	}
	if builder.agentType != "react" {
		t.Errorf("expected default agentType 'react', got %s", builder.agentType)
	}
}

func TestBuilder_WithMethods(t *testing.T) {
	mockLLM := NewMockLLM()
	mockChatModel := NewMockChatModel()
	mockMemory := NewMockMemory()
	mockTool := NewMockTool("test-tool", "A test tool")

	builder := NewBuilder().
		WithName("test-agent").
		WithSystemPrompt("You are a test agent").
		WithMaxTurns(5).
		WithVerbose(true).
		WithAgentType("tool_calling").
		WithLLM(mockLLM).
		WithChatModel(mockChatModel).
		WithMemory(mockMemory).
		WithTool(mockTool).
		WithTimeout(1 * time.Minute)

	if builder.name != "test-agent" {
		t.Errorf("expected name 'test-agent', got %s", builder.name)
	}
	if builder.systemPrompt != "You are a test agent" {
		t.Errorf("expected systemPrompt 'You are a test agent', got %s", builder.systemPrompt)
	}
	if builder.maxTurns != 5 {
		t.Errorf("expected maxTurns 5, got %d", builder.maxTurns)
	}
	if !builder.verbose {
		t.Error("expected verbose to be true")
	}
	if builder.agentType != "tool_calling" {
		t.Errorf("expected agentType 'tool_calling', got %s", builder.agentType)
	}
	if builder.llm != mockLLM {
		t.Error("expected llm to be set")
	}
	if builder.chatModel != mockChatModel {
		t.Error("expected chatModel to be set")
	}
	if builder.memory != mockMemory {
		t.Error("expected memory to be set")
	}
	if len(builder.tools) != 1 {
		t.Errorf("expected 1 tool, got %d", len(builder.tools))
	}
	if builder.timeout != 1*time.Minute {
		t.Errorf("expected timeout 1m, got %v", builder.timeout)
	}
}

func TestBuilder_WithBufferMemory(t *testing.T) {
	builder := NewBuilder().WithBufferMemory(100)

	if !builder.enableMemory {
		t.Error("expected enableMemory to be true")
	}
	if builder.memoryType != "buffer" {
		t.Errorf("expected memoryType 'buffer', got %s", builder.memoryType)
	}
	if builder.memorySize != 100 {
		t.Errorf("expected memorySize 100, got %d", builder.memorySize)
	}
}

func TestBuilder_WithWindowMemory(t *testing.T) {
	builder := NewBuilder().WithWindowMemory(20)

	if !builder.enableMemory {
		t.Error("expected enableMemory to be true")
	}
	if builder.memoryType != "window" {
		t.Errorf("expected memoryType 'window', got %s", builder.memoryType)
	}
	if builder.memorySize != 20 {
		t.Errorf("expected memorySize 20, got %d", builder.memorySize)
	}
}

func TestBuilder_WithTools(t *testing.T) {
	tool1 := NewMockTool("tool1", "First tool")
	tool2 := NewMockTool("tool2", "Second tool")

	builder := NewBuilder().
		WithTool(tool1).
		WithTools([]core.Tool{tool2})

	if len(builder.tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(builder.tools))
	}
}

func TestBuilder_Build_MissingLLM(t *testing.T) {
	builder := NewBuilder()

	_, err := builder.Build(context.Background())
	if err == nil {
		t.Error("expected error for missing LLM")
	}

	var agentErr *Error
	if !errors.As(err, &agentErr) {
		t.Fatal("expected *Error type")
	}
	if agentErr.Code != ErrCodeMissingLLM {
		t.Errorf("expected error code %s, got %s", ErrCodeMissingLLM, agentErr.Code)
	}
}

func TestBuilder_Build_WithLLM(t *testing.T) {
	mockLLM := NewMockLLM()

	agent, err := NewBuilder().
		WithLLM(mockLLM).
		WithName("test-agent").
		Build(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agent == nil {
		t.Fatal("expected agent to be non-nil")
	}
	if agent.GetName() != "test-agent" {
		t.Errorf("expected name 'test-agent', got %s", agent.GetName())
	}
}

func TestBuilder_Build_WithChatModel(t *testing.T) {
	mockChatModel := NewMockChatModel()

	agent, err := NewBuilder().
		WithChatModel(mockChatModel).
		WithName("chat-agent").
		Build(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agent == nil {
		t.Fatal("expected agent to be non-nil")
	}
	if agent.GetName() != "chat-agent" {
		t.Errorf("expected name 'chat-agent', got %s", agent.GetName())
	}
}

func TestBuilder_Build_WithBufferMemory(t *testing.T) {
	mockLLM := NewMockLLM()

	agent, err := NewBuilder().
		WithLLM(mockLLM).
		WithBufferMemory(50).
		Build(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agent.GetMemory() == nil {
		t.Error("expected memory to be configured")
	}
}

func TestBuilder_Getters(t *testing.T) {
	builder := NewBuilder().
		WithName("my-agent").
		WithSystemPrompt("Hello").
		WithMaxTurns(15).
		WithAgentType("simple")

	if builder.GetName() != "my-agent" {
		t.Errorf("expected name 'my-agent', got %s", builder.GetName())
	}
	if builder.GetSystemPrompt() != "Hello" {
		t.Errorf("expected systemPrompt 'Hello', got %s", builder.GetSystemPrompt())
	}
	if builder.GetMaxTurns() != 15 {
		t.Errorf("expected maxTurns 15, got %d", builder.GetMaxTurns())
	}
	if builder.GetAgentType() != "simple" {
		t.Errorf("expected agentType 'simple', got %s", builder.GetAgentType())
	}
}
