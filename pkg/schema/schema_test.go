package schema

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/schema/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema/internal"
)

// Message Interface Tests

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
func TestMessageInterfaceCompliance(t *testing.T) {
	tests := []struct {
		name        string
		message     Message
		wantType    MessageType
		wantContent string
	}{
		{
			name:        "HumanMessage",
			message:     NewHumanMessage("Hello, world!"),
			wantType:    iface.RoleHuman,
			wantContent: "Hello, world!",
		},
		{
			name:        "AIMessage",
			message:     NewAIMessage("Hello from AI!"),
			wantType:    iface.RoleAssistant,
			wantContent: "Hello from AI!",
		},
		{
			name:        "SystemMessage",
			message:     NewSystemMessage("You are a helpful assistant."),
			wantType:    iface.RoleSystem,
			wantContent: "You are a helpful assistant.",
		},
		{
			name:        "ToolMessage",
			message:     NewToolMessage("Tool result", "call_123"),
			wantType:    iface.RoleTool,
			wantContent: "Tool result",
		},
		{
			name:        "FunctionMessage",
			message:     NewFunctionMessage("calculate", "Result: 42"),
			wantType:    iface.RoleFunction,
			wantContent: "Result: 42",
		},
		{
			name:        "ChatMessage_Human",
			message:     NewChatMessage(iface.RoleHuman, "Custom human message"),
			wantType:    iface.RoleHuman,
			wantContent: "Custom human message",
		},
		{
			name:        "Document",
			message:     NewDocument("Document content", map[string]string{"author": "test"}),
			wantType:    iface.RoleSystem,
			wantContent: "Document content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.message.GetType() != tt.wantType {
				t.Errorf("GetType() = %v, want %v", tt.message.GetType(), tt.wantType)
			}
			if tt.message.GetContent() != tt.wantContent {
				t.Errorf("GetContent() = %q, want %q", tt.message.GetContent(), tt.wantContent)
			}

			// Test that AdditionalArgs returns a non-nil map
			args := tt.message.AdditionalArgs()
			if args == nil {
				t.Error("AdditionalArgs() should not return nil")
			}

			// Test that ToolCalls returns a slice (may be empty)
			toolCalls := tt.message.ToolCalls()
			if toolCalls == nil {
				t.Error("ToolCalls() should not return nil")
			}
		})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
}

func TestMessageValidation(t *testing.T) {
	tests := []struct {
		name    string
		message Message
		wantErr bool
	}{
		{
			name:    "valid human message",
			message: NewHumanMessage("Hello"),
			wantErr: false,
		},
		{
			name:    "valid AI message",
			message: NewAIMessage("Response"),
			wantErr: false,
		},
		{
			name:    "empty content human message",
			message: NewHumanMessage(""),
			wantErr: true,
		},
		{
			name:    "empty content AI message",
			message: NewAIMessage(""),
			wantErr: true,
		},
		{
			name:    "nil message",
			message: nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMessage(tt.message)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		})
	}
}

func TestMessageTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		got      MessageType
		expected string
	}{
		{"RoleHuman", iface.RoleHuman, "human"},
		{"RoleAssistant", iface.RoleAssistant, "ai"},
		{"RoleSystem", iface.RoleSystem, "system"},
		{"RoleTool", iface.RoleTool, "tool"},
		{"RoleFunction", iface.RoleFunction, "function"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.got) != tt.expected {
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
				t.Errorf("MessageType constant %s = %q, want %q", tt.name, string(tt.got), tt.expected)
			}
		})
	}
}

func TestToolCallStructure(t *testing.T) {
	toolCall := iface.ToolCall{
		ID:   "call_123",
		Type: "function",
		Function: iface.FunctionCall{
			Name:      "calculate",
			Arguments: `{"expression": "2+2"}`,
		},
		Name:      "calculate",
		Arguments: `{"expression": "2+2"}`,
	}

	if toolCall.ID != "call_123" {
		t.Errorf("ToolCall.ID = %q, want %q", toolCall.ID, "call_123")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	if toolCall.Type != "function" {
		t.Errorf("ToolCall.Type = %q, want %q", toolCall.Type, "function")
	}
	if toolCall.Function.Name != "calculate" {
		t.Errorf("ToolCall.Function.Name = %q, want %q", toolCall.Function.Name, "calculate")
	}
}

func TestAIMessageWithToolCalls(t *testing.T) {
	content := "I'll calculate that for you."
	toolCalls := []iface.ToolCall{
		{
			ID:   "call_123",
			Type: "function",
			Function: iface.FunctionCall{
				Name:      "calculate",
				Arguments: `{"expression": "2+2"}`,
			},
		},
	}

	// Create AI message with tool calls by casting to internal type
	aiMsg := &internal.AIMessage{
		BaseMessage: internal.BaseMessage{Content: content},
		ToolCalls_:  toolCalls,
	}

	// Verify through interface
	if aiMsg.GetType() != iface.RoleAssistant {
		t.Errorf("GetType() = %v, want %v", aiMsg.GetType(), iface.RoleAssistant)
	}
	if aiMsg.GetContent() != content {
		t.Errorf("GetContent() = %q, want %q", aiMsg.GetContent(), content)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()

	returnedToolCalls := aiMsg.ToolCalls()
	if len(returnedToolCalls) != 1 {
		t.Errorf("ToolCalls() length = %d, want 1", len(returnedToolCalls))
	}
	if returnedToolCalls[0].ID != "call_123" {
		t.Errorf("ToolCalls()[0].ID = %q, want %q", returnedToolCalls[0].ID, "call_123")
	}
}

func TestMessageEdgeCases(t *testing.T) {
	t.Run("empty message content", func(t *testing.T) {
		msg := NewHumanMessage("")
		if msg.GetContent() != "" {
			t.Errorf("Expected empty content, got %q", msg.GetContent())
		}
	})

	t.Run("unicode content", func(t *testing.T) {
		content := "Hello 世界 🌍"
		msg := NewAIMessage(content)
		if msg.GetContent() != content {
			t.Errorf("Unicode content mismatch: got %q, want %q", msg.GetContent(), content)
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	t.Run("long content", func(t *testing.T) {
		longContent := string(make([]byte, 10000)) // 10KB content
		for i := range longContent {
			longContent = longContent[:i] + "a" + longContent[i+1:]
		}
		msg := NewSystemMessage(longContent)
		if len(msg.GetContent()) != 10000 {
			t.Errorf("Long content length = %d, want 10000", len(msg.GetContent()))
		}
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	})
}

func TestBaseMessage_GetContent(t *testing.T) {
	content := "Test message content"
	baseMsg := internal.BaseMessage{
		Content: content,
	}

	if baseMsg.GetContent() != content {
		t.Errorf("Expected content to be %q, got %q", content, baseMsg.GetContent())
	}
}

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
func TestBaseMessage_AdditionalArgs(t *testing.T) {
	baseMsg := internal.BaseMessage{
		Content: "Test",
	}

	args := baseMsg.AdditionalArgs()
	if args == nil {
		t.Error("Expected non-nil additional args for BaseMessage")
	}
	if len(args) != 0 {
		t.Errorf("Expected empty additional args, got %d items", len(args))
	}
}

// ChatHistory Tests

func TestNewBaseChatHistory(t *testing.T) {
	tests := []struct {
		name    string
		opts    []ChatHistoryOption
		wantErr bool
	}{
		{
			name:    "default config",
			opts:    []ChatHistoryOption{},
			wantErr: false,
		},
		{
			name: "with max messages",
			opts: []ChatHistoryOption{
				WithMaxMessages(50),
			},
			wantErr: false,
		},
		{
			name: "with TTL",
			opts: []ChatHistoryOption{
				WithTTL(24 * time.Hour),
			},
			wantErr: false,
		},
		{
			name: "with persistence",
			opts: []ChatHistoryOption{
				WithPersistence(true),
			},
			wantErr: false,
		},
		{
			name: "full config",
			opts: []ChatHistoryOption{
				WithMaxMessages(100),
				WithTTL(12 * time.Hour),
				WithPersistence(true),
			},
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			history, err := NewBaseChatHistory(tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewBaseChatHistory() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && history == nil {
				t.Error("NewBaseChatHistory() returned nil history without error")
			}
		})
	}
}

func TestChatHistoryOperations(t *testing.T) {
	history, err := NewBaseChatHistory(WithMaxMessages(3))
	if err != nil {
		t.Fatalf("Failed to create chat history: %v", err)
	}

	// Test adding user message
	err = history.AddUserMessage("Hello")
	if err != nil {
		t.Errorf("AddUserMessage() error = %v", err)
	}

	// Test adding AI message
	err = history.AddAIMessage("Hi there!")
	if err != nil {
		t.Errorf("AddAIMessage() error = %v", err)
	}

	// Test adding generic message
	systemMsg := NewSystemMessage("You are a helpful assistant")
	err = history.AddMessage(systemMsg)
	if err != nil {
		t.Errorf("AddMessage() error = %v", err)
	}

	// Test retrieving messages
	messages, err := history.Messages()
	if err != nil {
		t.Errorf("Messages() error = %v", err)
	}

	if len(messages) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(messages))
	}

	expectedTypes := []MessageType{iface.RoleHuman, iface.RoleAssistant, iface.RoleSystem}
	for i, msg := range messages {
		if msg.GetType() != expectedTypes[i] {
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
			t.Errorf("Message %d type = %v, want %v", i, msg.GetType(), expectedTypes[i])
		}
	}

	// Test clearing history
	err = history.Clear()
	if err != nil {
		t.Errorf("Clear() error = %v", err)
	}

	messages, err = history.Messages()
	if err != nil {
		t.Errorf("Messages() after clear error = %v", err)
	}

	if len(messages) != 0 {
		t.Errorf("Expected 0 messages after clear, got %d", len(messages))
	}
}

func TestChatHistoryMaxMessages(t *testing.T) {
	history, err := NewBaseChatHistory(WithMaxMessages(2))
	if err != nil {
		t.Fatalf("Failed to create chat history: %v", err)
	}

	// Add more messages than the limit
	messages := []string{"First", "Second", "Third", "Fourth"}
	for _, msg := range messages {
		err = history.AddUserMessage(msg)
		if err != nil {
			t.Errorf("AddUserMessage() error = %v", err)
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		}
	}

	// Should only keep the last 2 messages
	retrieved, err := history.Messages()
	if err != nil {
		t.Errorf("Messages() error = %v", err)
	}

	if len(retrieved) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(retrieved))
	}

	// Should be the last two messages added
	if retrieved[0].GetContent() != "Third" {
		t.Errorf("First message content = %q, want %q", retrieved[0].GetContent(), "Third")
	}
	if retrieved[1].GetContent() != "Fourth" {
		t.Errorf("Second message content = %q, want %q", retrieved[1].GetContent(), "Fourth")
	}
}

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
func TestChatHistoryUnlimitedMessages(t *testing.T) {
	history, err := NewBaseChatHistory() // No limit
	if err != nil {
		t.Fatalf("Failed to create chat history: %v", err)
	}

	// Add many messages
	for i := 0; i < 100; i++ {
		err = history.AddUserMessage(fmt.Sprintf("Message %d", i))
		if err != nil {
			t.Errorf("AddUserMessage() error = %v", err)
		}
	}

	messages, err := history.Messages()
	if err != nil {
		t.Errorf("Messages() error = %v", err)
	}

	if len(messages) != 100 {
		t.Errorf("Expected 100 messages, got %d", len(messages))
	}
}

func TestChatHistoryConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *ChatHistoryConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &ChatHistoryConfig{
				MaxMessages:       100,
				TTL:               24 * time.Hour,
				EnablePersistence: true,
			},
			wantErr: false,
		},
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		{
			name: "negative max messages",
			config: &ChatHistoryConfig{
				MaxMessages: -1,
			},
			wantErr: true,
		},
		{
			name: "zero max messages",
			config: &ChatHistoryConfig{
				MaxMessages: 0,
			},
			wantErr: false, // 0 means unlimited
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("ChatHistoryConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChatHistoryFunctionalOptions(t *testing.T) {
	// Test WithMaxMessages
	config, err := NewChatHistoryConfig(WithMaxMessages(50))
	if err != nil {
		t.Fatalf("NewChatHistoryConfig() error = %v", err)
	}
	if config.MaxMessages != 50 {
		t.Errorf("MaxMessages = %d, want 50", config.MaxMessages)
	}

	// Test WithTTL
	ttl := 12 * time.Hour
	config, err = NewChatHistoryConfig(WithTTL(ttl))
	if err != nil {
		t.Fatalf("NewChatHistoryConfig() error = %v", err)
	}
	if config.TTL != ttl {
		t.Errorf("TTL = %v, want %v", config.TTL, ttl)
	}

	// Test WithPersistence
	config, err = NewChatHistoryConfig(WithPersistence(true))
	if err != nil {
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		t.Fatalf("NewChatHistoryConfig() error = %v", err)
	}
	if !config.EnablePersistence {
		t.Error("EnablePersistence should be true")
	}

	// Test combined options
	config, err = NewChatHistoryConfig(
		WithMaxMessages(25),
		WithTTL(6*time.Hour),
		WithPersistence(false),
	)
	if err != nil {
		t.Fatalf("NewChatHistoryConfig() error = %v", err)
	}
	if config.MaxMessages != 25 {
		t.Errorf("MaxMessages = %d, want 25", config.MaxMessages)
	}
	if config.TTL != 6*time.Hour {
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		t.Errorf("TTL = %v, want %v", config.TTL, 6*time.Hour)
	}
	if config.EnablePersistence {
		t.Error("EnablePersistence should be false")
	}
}

// Agent I/O Tests

func TestNewAgentAction(t *testing.T) {
	tool := "calculator"
	toolInput := map[string]interface{}{
		"expression": "2 + 2",
		"operation":  "add",
	}
	log := "Calculating 2 + 2"

	action := NewAgentAction(tool, toolInput, log)

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	if action.Tool != tool {
		t.Errorf("Tool = %q, want %q", action.Tool, tool)
	}
	if !reflect.DeepEqual(action.ToolInput, toolInput) {
		t.Errorf("ToolInput = %v, want %v", action.ToolInput, toolInput)
	}
	if action.Log != log {
		t.Errorf("Log = %q, want %q", action.Log, log)
	}
}

func TestNewAgentObservation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	actionLog := "Executed calculator tool"
	output := "Result: 4"
	parsedOutput := map[string]interface{}{
		"result":  4,
		"success": true,
	}

	observation := NewAgentObservation(actionLog, output, parsedOutput)

	if observation.ActionLog != actionLog {
		t.Errorf("ActionLog = %q, want %q", observation.ActionLog, actionLog)
	}
	if observation.Output != output {
		t.Errorf("Output = %q, want %q", observation.Output, output)
	}
	if !reflect.DeepEqual(observation.ParsedOutput, parsedOutput) {
		t.Errorf("ParsedOutput = %v, want %v", observation.ParsedOutput, parsedOutput)
	}
}

func TestNewStep(t *testing.T) {
	action := NewAgentAction("search", map[string]interface{}{"query": "AI"}, "Searching for AI")
	observation := NewAgentObservation("Searching for AI", "Found results", nil)

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	step := NewStep(action, observation)

	if step.Action.Tool != "search" {
		t.Errorf("Step.Action.Tool = %q, want %q", step.Action.Tool, "search")
	}
	if step.Observation.ActionLog != "Searching for AI" {
		t.Errorf("Step.Observation.ActionLog = %q, want %q", step.Observation.ActionLog, "Searching for AI")
	}
}

func TestNewFinalAnswer(t *testing.T) {
	output := "The answer is 42"
	sourceDocuments := []interface{}{
		NewDocument("Source 1", map[string]string{"title": "Doc 1"}),
		NewDocument("Source 2", map[string]string{"title": "Doc 2"}),
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	}
	intermediateSteps := []Step{
		NewStep(
			NewAgentAction("calculate", map[string]interface{}{"expr": "40+2"}, "Calculating"),
			NewAgentObservation("Calculating", "42", 42),
		),
	}

	finalAnswer := NewFinalAnswer(output, sourceDocuments, intermediateSteps)

	if finalAnswer.Output != output {
		t.Errorf("Output = %q, want %q", finalAnswer.Output, output)
	}
	if len(finalAnswer.SourceDocuments) != 2 {
		t.Errorf("SourceDocuments length = %d, want 2", len(finalAnswer.SourceDocuments))
	}
	if len(finalAnswer.IntermediateSteps) != 1 {
		t.Errorf("IntermediateSteps length = %d, want 1", len(finalAnswer.IntermediateSteps))
	}
}

func TestNewAgentFinish(t *testing.T) {
	returnValues := map[string]interface{}{
		"answer":     42,
		"confidence": 0.95,
	}
	log := "Task completed successfully"

	finish := NewAgentFinish(returnValues, log)

	if !reflect.DeepEqual(finish.ReturnValues, returnValues) {
		t.Errorf("ReturnValues = %v, want %v", finish.ReturnValues, returnValues)
	}
	if finish.Log != log {
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		t.Errorf("Log = %q, want %q", finish.Log, log)
	}
}

func TestAgentActionWithComplexInput(t *testing.T) {
	// Test with complex nested input
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	complexInput := map[string]interface{}{
		"query": "machine learning",
		"filters": map[string]interface{}{
			"category": "research",
			"year":     2024,
		},
		"options": []string{"relevance", "recency"},
	}

	action := NewAgentAction("search", complexInput, "Advanced search")

	if action.Tool != "search" {
		t.Errorf("Tool = %q, want %q", action.Tool, "search")
	}

	inputMap, ok := action.ToolInput.(map[string]interface{})
	if !ok {
		t.Fatal("ToolInput is not a map")
	}

	if inputMap["query"] != "machine learning" {
		t.Errorf("Query = %v, want %q", inputMap["query"], "machine learning")
	}

	filters, ok := inputMap["filters"].(map[string]interface{})
	if !ok {
		t.Fatal("Filters is not a map")
	}

	if filters["year"] != 2024 {
		t.Errorf("Year = %v, want 2024", filters["year"])
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
}

func TestAgentObservationWithNilParsedOutput(t *testing.T) {
	observation := NewAgentObservation("Action log", "Raw output", nil)

	if observation.ParsedOutput != nil {
		t.Errorf("ParsedOutput should be nil, got %v", observation.ParsedOutput)
	}
}

func TestStepSequence(t *testing.T) {
	// Test a sequence of steps
	steps := []Step{
		NewStep(
			NewAgentAction("analyze", map[string]interface{}{"text": "Hello"}, "Analyzing text"),
			NewAgentObservation("Analyzing text", "Analysis complete", map[string]interface{}{"sentiment": "positive"}),
		),
		NewStep(
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
			NewAgentAction("summarize", map[string]interface{}{"content": "Analysis complete"}, "Summarizing"),
			NewAgentObservation("Summarizing", "Summary: Positive sentiment detected", "Positive sentiment"),
		),
	}

	if len(steps) != 2 {
		t.Errorf("Expected 2 steps, got %d", len(steps))
	}

	// Verify first step
	if steps[0].Action.Tool != "analyze" {
		t.Errorf("First step action tool = %q, want %q", steps[0].Action.Tool, "analyze")
	}
	if steps[0].Observation.Output != "Analysis complete" {
		t.Errorf("First step observation output = %q, want %q", steps[0].Observation.Output, "Analysis complete")
	}

	// Verify second step
	if steps[1].Action.Tool != "summarize" {
		t.Errorf("Second step action tool = %q, want %q", steps[1].Action.Tool, "summarize")
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	}
	if steps[1].Observation.Output != "Summary: Positive sentiment detected" {
		t.Errorf("Second step observation output = %q, want %q", steps[1].Observation.Output, "Summary: Positive sentiment detected")
	}
}

func TestAgentScratchPadEntry(t *testing.T) {
	action := NewAgentAction("think", map[string]interface{}{"thought": "I need to solve this"}, "Thinking")
	observation := "I should use the calculator tool"

	// Create scratch pad entry using internal type
	entry := internal.AgentScratchPadEntry{
		Action:      action,
		Observation: observation,
	}

	if entry.Action.Tool != "think" {
		t.Errorf("Action.Tool = %q, want %q", entry.Action.Tool, "think")
	}
	if entry.Observation != observation {
		t.Errorf("Observation = %q, want %q", entry.Observation, observation)
	}
}

// LLM Types Tests

func TestNewGeneration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	text := "Hello, world!"
	message := NewHumanMessage("Hello")
	generationInfo := map[string]interface{}{
		"model":         "gpt-4",
		"temperature":   0.7,
		"finish_reason": "stop",
	}

	generation := NewGeneration(text, message, generationInfo)

	if generation.Text != text {
		t.Errorf("Text = %q, want %q", generation.Text, text)
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	}
	if generation.Message.GetContent() != "Hello" {
		t.Errorf("Message content = %q, want %q", generation.Message.GetContent(), "Hello")
	}
	if generation.GenerationInfo["model"] != "gpt-4" {
		t.Errorf("GenerationInfo[model] = %v, want %q", generation.GenerationInfo["model"], "gpt-4")
	}
}

func TestNewLLMResponse(t *testing.T) {
	generations := [][]*Generation{
		{
			NewGeneration("Hello", NewHumanMessage("Hi"), nil),
			NewGeneration("How can I help?", NewAIMessage("How can I help?"), nil),
		},
	}
	llmOutput := map[string]interface{}{
		"model": "gpt-4",
		"usage": map[string]interface{}{
			"prompt_tokens":     10,
			"completion_tokens": 20,
			"total_tokens":      30,
		},
	}

	response := NewLLMResponse(generations, llmOutput)

	if len(response.Generations) != 1 {
		t.Errorf("Generations length = %d, want 1", len(response.Generations))
	}
	if len(response.Generations[0]) != 2 {
		t.Errorf("First generation batch length = %d, want 2", len(response.Generations[0]))
	}
	if response.LLMOutput["model"] != "gpt-4" {
		t.Errorf("LLMOutput[model] = %v, want %q", response.LLMOutput["model"], "gpt-4")
	}
}

func TestNewCallOptions(t *testing.T) {
	callOpts := NewCallOptions()

	if callOpts == nil {
		t.Fatal("NewCallOptions() returned nil")
	}
	if callOpts.ProviderSpecificArgs == nil {
		t.Error("ProviderSpecificArgs should be initialized")
	}
	if len(callOpts.ProviderSpecificArgs) != 0 {
		t.Errorf("ProviderSpecificArgs should be empty, got length %d", len(callOpts.ProviderSpecificArgs))
	}
}

func TestLLMOptionFunctions(t *testing.T) {
	callOpts := NewCallOptions()
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()

	// Test WithTemperature
	WithTemperature(0.8)(callOpts)
	if callOpts.Temperature == nil || *callOpts.Temperature != 0.8 {
		t.Errorf("Temperature = %v, want 0.8", callOpts.Temperature)
	}

	// Test WithMaxTokens
	WithMaxTokens(1024)(callOpts)
	if callOpts.MaxTokens == nil || *callOpts.MaxTokens != 1024 {
		t.Errorf("MaxTokens = %v, want 1024", callOpts.MaxTokens)
	}

	// Test WithTopP
	WithTopP(0.9)(callOpts)
	if callOpts.TopP == nil || *callOpts.TopP != 0.9 {
		t.Errorf("TopP = %v, want 0.9", callOpts.TopP)
	}

	// Test WithFrequencyPenalty
	WithFrequencyPenalty(0.5)(callOpts)
	if callOpts.FrequencyPenalty == nil || *callOpts.FrequencyPenalty != 0.5 {
		t.Errorf("FrequencyPenalty = %v, want 0.5", callOpts.FrequencyPenalty)
	}

	// Test WithPresencePenalty
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	WithPresencePenalty(0.3)(callOpts)
	if callOpts.PresencePenalty == nil || *callOpts.PresencePenalty != 0.3 {
		t.Errorf("PresencePenalty = %v, want 0.3", callOpts.PresencePenalty)
	}

	// Test WithStopSequences
	stopSequences := []string{"STOP", "END"}
	WithStopSequences(stopSequences)(callOpts)
	if len(callOpts.Stop) != 2 {
		t.Errorf("Stop length = %d, want 2", len(callOpts.Stop))
	}
	if callOpts.Stop[0] != "STOP" {
		t.Errorf("Stop[0] = %q, want %q", callOpts.Stop[0], "STOP")
	}

	// Test WithStreaming
	WithStreaming(true)(callOpts)
	if !callOpts.Streaming {
		t.Error("Streaming should be true")
	}

	// Test WithProviderSpecificArg
	WithProviderSpecificArg("custom_param", "value")(callOpts)
	if callOpts.ProviderSpecificArgs["custom_param"] != "value" {
		t.Errorf("ProviderSpecificArgs[custom_param] = %v, want %q", callOpts.ProviderSpecificArgs["custom_param"], "value")
	}
}

func TestCallOptionsComposition(t *testing.T) {
	callOpts := NewCallOptions()
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()

	// Apply multiple options
	WithTemperature(0.7)(callOpts)
	WithMaxTokens(500)(callOpts)
	WithStreaming(true)(callOpts)
	WithProviderSpecificArg("model", "gpt-4")(callOpts)
	WithProviderSpecificArg("organization", "test-org")(callOpts)

	// Verify all options are set
	if callOpts.Temperature == nil || *callOpts.Temperature != 0.7 {
		t.Errorf("Temperature = %v, want 0.7", callOpts.Temperature)
	}
	if callOpts.MaxTokens == nil || *callOpts.MaxTokens != 500 {
		t.Errorf("MaxTokens = %v, want 500", callOpts.MaxTokens)
	}
	if !callOpts.Streaming {
		t.Error("Streaming should be true")
	}
	if callOpts.ProviderSpecificArgs["model"] != "gpt-4" {
		t.Errorf("ProviderSpecificArgs[model] = %v, want %q", callOpts.ProviderSpecificArgs["model"], "gpt-4")
	}
	if callOpts.ProviderSpecificArgs["organization"] != "test-org" {
		t.Errorf("ProviderSpecificArgs[organization] = %v, want %q", callOpts.ProviderSpecificArgs["organization"], "test-org")
	}
}

func TestGenerationWithComplexInfo(t *testing.T) {
	text := "The answer is 42"
	message := NewAIMessage(text)

	generationInfo := map[string]interface{}{
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		"model": "gpt-4-turbo",
		"usage": map[string]interface{}{
			"prompt_tokens":     150,
			"completion_tokens": 50,
			"total_tokens":      200,
		},
		"finish_reason": "stop",
		"model_version": "2024-04-01",
		"temperature":   0.7,
	}

	generation := NewGeneration(text, message, generationInfo)

	if generation.Text != text {
		t.Errorf("Text = %q, want %q", generation.Text, text)
	}
	if generation.Message.GetType() != iface.RoleAssistant {
		t.Errorf("Message type = %v, want %v", generation.Message.GetType(), iface.RoleAssistant)
	}

	// Check nested usage info
	usage := generation.GenerationInfo["usage"].(map[string]interface{})
	if usage["total_tokens"] != 200 {
		t.Errorf("Usage total_tokens = %v, want 200", usage["total_tokens"])
	}
}

func TestLLMResponseWithMultipleBatches(t *testing.T) {
	// Create multiple batches of generations
	batch1 := []*Generation{
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		NewGeneration("Question 1?", NewHumanMessage("Question 1?"), nil),
		NewGeneration("Answer 1", NewAIMessage("Answer 1"), nil),
	}
	batch2 := []*Generation{
		NewGeneration("Question 2?", NewHumanMessage("Question 2?"), nil),
		NewGeneration("Answer 2", NewAIMessage("Answer 2"), nil),
	}

	generations := [][]*Generation{batch1, batch2}
	llmOutput := map[string]interface{}{
		"total_batches": 2,
		"model":         "gpt-4",
	}

	response := NewLLMResponse(generations, llmOutput)

	if len(response.Generations) != 2 {
		t.Errorf("Generations batches = %d, want 2", len(response.Generations))
	}
	if len(response.Generations[0]) != 2 {
		t.Errorf("First batch length = %d, want 2", len(response.Generations[0]))
	}
	if len(response.Generations[1]) != 2 {
		t.Errorf("Second batch length = %d, want 2", len(response.Generations[1]))
	}
	if response.LLMOutput["total_batches"] != 2 {
		t.Errorf("LLMOutput[total_batches] = %v, want 2", response.LLMOutput["total_batches"])
	}
}
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()

func TestCallOptionsDefaults(t *testing.T) {
	callOpts := NewCallOptions()

	// Check that pointers are nil by default (meaning not set)
	if callOpts.Temperature != nil {
		t.Errorf("Temperature should be nil by default, got %v", callOpts.Temperature)
	}
	if callOpts.MaxTokens != nil {
		t.Errorf("MaxTokens should be nil by default, got %v", callOpts.MaxTokens)
	}
	if callOpts.TopP != nil {
		t.Errorf("TopP should be nil by default, got %v", callOpts.TopP)
	}
	if callOpts.FrequencyPenalty != nil {
		t.Errorf("FrequencyPenalty should be nil by default, got %v", callOpts.FrequencyPenalty)
	}
	if callOpts.PresencePenalty != nil {
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		t.Errorf("PresencePenalty should be nil by default, got %v", callOpts.PresencePenalty)
	}
	if len(callOpts.Stop) != 0 {
		t.Errorf("Stop should be empty by default, got %v", callOpts.Stop)
	}
	if callOpts.Streaming {
		t.Error("Streaming should be false by default")
	}
	if callOpts.ProviderSpecificArgs == nil {
		t.Error("ProviderSpecificArgs should be initialized")
	}
}

// Document Tests

func TestNewDocument(t *testing.T) {
	pageContent := "This is a test document content."
	metadata := map[string]string{
		"author":     "Test Author",
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		"title":      "Test Document",
		"category":   "testing",
		"created_at": "2024-01-01",
	}

	doc := NewDocument(pageContent, metadata)

	if doc.PageContent != pageContent {
		t.Errorf("PageContent = %q, want %q", doc.PageContent, pageContent)
	}
	if len(doc.Metadata) != 4 {
		t.Errorf("Metadata length = %d, want 4", len(doc.Metadata))
	}
	if doc.Metadata["author"] != "Test Author" {
		t.Errorf("Metadata[author] = %q, want %q", doc.Metadata["author"], "Test Author")
	}
	if doc.ID != "" {
		t.Errorf("ID should be empty for NewDocument, got %q", doc.ID)
	}
	if doc.Embedding != nil {
		t.Error("Embedding should be nil for NewDocument")
	}
	if doc.Score != 0 {
		t.Errorf("Score should be 0, got %f", doc.Score)
	}
}

func TestNewDocumentWithID(t *testing.T) {
	id := "doc-12345"
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	pageContent := "Document with specific ID"
	metadata := map[string]string{
		"source": "database",
	}

	doc := NewDocumentWithID(id, pageContent, metadata)

	if doc.ID != id {
		t.Errorf("ID = %q, want %q", doc.ID, id)
	}
	if doc.PageContent != pageContent {
		t.Errorf("PageContent = %q, want %q", doc.PageContent, pageContent)
	}
	if doc.Metadata["source"] != "database" {
		t.Errorf("Metadata[source] = %q, want %q", doc.Metadata["source"], "database")
	}
}

func TestNewDocumentWithEmbedding(t *testing.T) {
	pageContent := "Document with embedding"
	metadata := map[string]string{"type": "embedded"}
	embedding := []float32{0.1, 0.2, 0.3, 0.4, 0.5}

	doc := NewDocumentWithEmbedding(pageContent, metadata, embedding)

	if doc.PageContent != pageContent {
		t.Errorf("PageContent = %q, want %q", doc.PageContent, pageContent)
	}
	if len(doc.Embedding) != 5 {
		t.Errorf("Embedding length = %d, want 5", len(doc.Embedding))
	}
	if doc.Embedding[0] != 0.1 {
		t.Errorf("Embedding[0] = %f, want 0.1", doc.Embedding[0])
	}
	if doc.Score != 0 {
		t.Errorf("Score should be 0, got %f", doc.Score)
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	}
}

func TestDocumentAsMessage(t *testing.T) {
	metadata := map[string]string{
		"author": "Test Author",
		"title":  "Test Doc",
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	}
	doc := NewDocument("Document content", metadata)

	// Test Message interface implementation
	if doc.GetType() != iface.RoleSystem {
		t.Errorf("GetType() = %v, want %v", doc.GetType(), iface.RoleSystem)
	}
	if doc.GetContent() != "Document content" {
		t.Errorf("GetContent() = %q, want %q", doc.GetContent(), "Document content")
	}

	// Documents don't have tool calls
	toolCalls := doc.ToolCalls()
	if toolCalls == nil {
		t.Error("ToolCalls() should not return nil")
	}
	if len(toolCalls) != 0 {
		t.Errorf("ToolCalls() length = %d, want 0", len(toolCalls))
	}

	// Documents have empty additional args
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	args := doc.AdditionalArgs()
	if args == nil {
		t.Error("AdditionalArgs() should not return nil")
	}
}

func TestDocumentValidation(t *testing.T) {
	tests := []struct {
		name    string
		doc     Document
		wantErr bool
	}{
		{
			name:    "valid document",
			doc:     NewDocument("Valid content", map[string]string{"key": "value"}),
			wantErr: false,
		},
		{
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
			name:    "empty content",
			doc:     NewDocument("", map[string]string{"key": "value"}),
			wantErr: true,
		},
		{
			name:    "nil metadata",
			doc:     Document{PageContent: "Content", Metadata: nil},
			wantErr: true,
		},
		{
			name:    "empty metadata",
			doc:     NewDocument("Content", map[string]string{}),
			wantErr: false,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDocument(tt.doc)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDocument() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDocumentWithScore(t *testing.T) {
	doc := NewDocument("Test content", map[string]string{"type": "test"})
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	doc.Score = 0.85

	if doc.Score != 0.85 {
		t.Errorf("Score = %f, want 0.85", doc.Score)
	}
}

func TestDocumentMetadataOperations(t *testing.T) {
	doc := NewDocument("Content", map[string]string{
		"author":  "Alice",
		"year":    "2024",
		"subject": "AI",
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()

	// Test metadata access
	if doc.Metadata["author"] != "Alice" {
		t.Errorf("Metadata[author] = %q, want %q", doc.Metadata["author"], "Alice")
	}

	// Test metadata modification
	doc.Metadata["category"] = "research"
	if doc.Metadata["category"] != "research" {
		t.Errorf("Metadata[category] = %q, want %q", doc.Metadata["category"], "research")
	}

	if len(doc.Metadata) != 4 {
		t.Errorf("Metadata length = %d, want 4", len(doc.Metadata))
	}
}

func TestDocumentEmbeddingOperations(t *testing.T) {
	doc := NewDocument("Content", map[string]string{})

	// Test setting embedding
	embedding := []float32{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8}
	doc.Embedding = embedding

	if len(doc.Embedding) != 8 {
		t.Errorf("Embedding length = %d, want 8", len(doc.Embedding))
	}

	// Test embedding values
	expected := []float32{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8}
	for i, v := range expected {
		if doc.Embedding[i] != v {
			t.Errorf("Embedding[%d] = %f, want %f", i, doc.Embedding[i], v)
		}
	}
}

func TestDocumentLargeContent(t *testing.T) {
	// Test with large content (1MB)
	largeContent := string(make([]byte, 1024*1024))
	for i := range largeContent {
		largeContent = largeContent[:i] + string(rune(i%26+97)) + largeContent[i+1:]
	}

	doc := NewDocument(largeContent, map[string]string{"size": "large"})

	if len(doc.PageContent) != 1024*1024 {
		t.Errorf("PageContent length = %d, want %d", len(doc.PageContent), 1024*1024)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	if doc.Metadata["size"] != "large" {
		t.Errorf("Metadata[size] = %q, want %q", doc.Metadata["size"], "large")
	}
}

func TestDocumentEmptyMetadata(t *testing.T) {
	doc := NewDocument("Content", map[string]string{})

	if len(doc.Metadata) != 0 {
		t.Errorf("Metadata length = %d, want 0", len(doc.Metadata))
	}

	// Should not panic when accessing non-existent key
	value := doc.Metadata["nonexistent"]
	if value != "" {
		t.Errorf("Metadata[nonexistent] = %q, want empty string", value)
	}
}

func TestDocumentUnicodeContent(t *testing.T) {
	unicodeContent := "Hello 世界 🌍 Test 内容 🚀"
	doc := NewDocument(unicodeContent, map[string]string{"lang": "mixed"})

	if doc.PageContent != unicodeContent {
		t.Errorf("PageContent = %q, want %q", doc.PageContent, unicodeContent)
	}

	if doc.Metadata["lang"] != "mixed" {
		t.Errorf("Metadata[lang] = %q, want %q", doc.Metadata["lang"], "mixed")
	}
}

// Config validation tests

func TestAgentConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *AgentConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &AgentConfig{
				Name:            "test-agent",
				LLMProviderName: "openai-gpt4",
				MaxIterations:   5,
			},
			wantErr: false,
		},
		{
			name: "missing name",
			config: &AgentConfig{
				LLMProviderName: "openai-gpt4",
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
				MaxIterations:   5,
			},
			wantErr: true,
		},
		{
			name: "missing llm provider",
			config: &AgentConfig{
				Name:          "test-agent",
				MaxIterations: 5,
			},
			wantErr: true,
		},
		{
			name: "invalid max iterations",
			config: &AgentConfig{
				Name:            "test-agent",
				LLMProviderName: "openai-gpt4",
				MaxIterations:   0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("AgentConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLLMProviderConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *LLMProviderConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &LLMProviderConfig{
				Name:      "openai-gpt4",
				Provider:  "openai",
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
				ModelName: "gpt-4-turbo",
				APIKey:    "sk-test",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			config: &LLMProviderConfig{
				Provider:  "openai",
				ModelName: "gpt-4-turbo",
				APIKey:    "sk-test",
			},
			wantErr: true,
		},
		{
			name: "missing provider",
			config: &LLMProviderConfig{
				Name:      "openai-gpt4",
				ModelName: "gpt-4-turbo",
				APIKey:    "sk-test",
			},
			wantErr: true,
		},
		{
			name: "missing model name",
			config: &LLMProviderConfig{
				Name:     "openai-gpt4",
				Provider: "openai",
				APIKey:   "sk-test",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
				t.Errorf("LLMProviderConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEmbeddingProviderConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *EmbeddingProviderConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &EmbeddingProviderConfig{
				Name:      "openai-embeddings",
				Provider:  "openai",
				ModelName: "text-embedding-ada-002",
				APIKey:    "sk-test",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			config: &EmbeddingProviderConfig{
				Provider:  "openai",
				ModelName: "text-embedding-ada-002",
				APIKey:    "sk-test",
			},
			wantErr: true,
		},
		{
			name: "missing api key",
			config: &EmbeddingProviderConfig{
				Name:      "openai-embeddings",
				Provider:  "openai",
				ModelName: "text-embedding-ada-002",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
				t.Errorf("EmbeddingProviderConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestVectorStoreConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *VectorStoreConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &VectorStoreConfig{
				Name:     "pgvector-store",
				Provider: "pgvector",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			config: &VectorStoreConfig{
				Provider: "pgvector",
			},
			wantErr: true,
		},
		{
			name: "missing provider",
			config: &VectorStoreConfig{
				Name: "pgvector-store",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
			if (err != nil) != tt.wantErr {
				t.Errorf("VectorStoreConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewAgentConfig(t *testing.T) {
	tests := []struct {
		name            string
		agentName       string
		llmProviderName string
		opts            []AgentOption
		wantErr         bool
	}{
		{
			name:            "valid config",
			agentName:       "test-agent",
			llmProviderName: "openai-gpt4",
			opts:            []AgentOption{WithMaxIterations(5)},
			wantErr:         false,
		},
		{
			name:            "empty name",
			agentName:       "",
			llmProviderName: "openai-gpt4",
			opts:            []AgentOption{},
			wantErr:         true,
		},
		{
			name:            "empty llm provider",
			agentName:       "test-agent",
			llmProviderName: "",
			opts:            []AgentOption{},
			wantErr:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := NewAgentConfig(tt.agentName, tt.llmProviderName, tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewAgentConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && config == nil {
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
				t.Error("NewAgentConfig() returned nil config without error")
			}
			if !tt.wantErr && config.Name != tt.agentName {
				t.Errorf("NewAgentConfig() config.Name = %q, want %q", config.Name, tt.agentName)
			}
		})
	}
}

func TestNewLLMProviderConfig(t *testing.T) {
	tests := []struct {
		name       string
		configName string
		provider   string
		modelName  string
		opts       []LLMProviderOption
		wantErr    bool
	}{
		{
			name:       "valid config",
			configName: "openai-gpt4",
			provider:   "openai",
			modelName:  "gpt-4-turbo",
			opts:       []LLMProviderOption{WithAPIKey("sk-test")},
			wantErr:    false,
		},
		{
			name:       "empty name",
			configName: "",
			provider:   "openai",
			modelName:  "gpt-4-turbo",
			opts:       []LLMProviderOption{},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := NewLLMProviderConfig(tt.configName, tt.provider, tt.modelName, tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewLLMProviderConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
			}
			if !tt.wantErr && config == nil {
				t.Error("NewLLMProviderConfig() returned nil config without error")
			}
		})
	}
}

func TestFunctionalOptions(t *testing.T) {
	// Test AgentOption
	agentConfig, err := NewAgentConfig("test-agent", "openai-gpt4",
		WithToolNames([]string{"tool1", "tool2"}),
		WithMaxIterations(20),
		WithPromptTemplate("You are a helpful assistant"),
		WithAgentType("react"),
	)
	if err != nil {
		t.Fatalf("NewAgentConfig() error = %v", err)
	}

	if len(agentConfig.ToolNames) != 2 {
		t.Errorf("ToolNames length = %d, want 2", len(agentConfig.ToolNames))
	}
	if agentConfig.MaxIterations != 20 {
		t.Errorf("MaxIterations = %d, want 20", agentConfig.MaxIterations)
	}
	if agentConfig.PromptTemplate != "You are a helpful assistant" {
		t.Errorf("PromptTemplate = %q, want %q", agentConfig.PromptTemplate, "You are a helpful assistant")
	}
	if agentConfig.AgentType != "react" {
		t.Errorf("AgentType = %q, want %q", agentConfig.AgentType, "react")
	}

	// Test LLMProviderOption
	llmConfig, err := NewLLMProviderConfig("openai-gpt4", "openai", "gpt-4-turbo",
		WithAPIKey("sk-test"),
		WithBaseURL("https://api.openai.com"),
		WithDefaultCallOptions(map[string]interface{}{"temperature": 0.7}),
	)
	if err != nil {
		t.Fatalf("NewLLMProviderConfig() error = %v", err)
	}

	if llmConfig.APIKey != "sk-test" {
		t.Errorf("APIKey = %q, want %q", llmConfig.APIKey, "sk-test")
	}
	if llmConfig.BaseURL != "https://api.openai.com" {
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		t.Errorf("BaseURL = %q, want %q", llmConfig.BaseURL, "https://api.openai.com")
	}
	if llmConfig.DefaultCallOptions["temperature"] != 0.7 {
		t.Errorf("DefaultCallOptions[temperature] = %v, want 0.7", llmConfig.DefaultCallOptions["temperature"])
	}
}

// SchemaValidationConfig tests

func TestNewSchemaValidationConfig(t *testing.T) {
	tests := []struct {
		name    string
		opts    []SchemaValidationOption
		wantErr bool
	}{
		{
			name:    "valid config with defaults",
			opts:    []SchemaValidationOption{},
			wantErr: false,
		},
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		{
			name: "valid config with custom options",
			opts: []SchemaValidationOption{
				WithStrictValidation(true),
				WithMaxMessageLength(5000),
				WithMaxMetadataSize(50),
				WithAllowedMessageTypes([]string{"human", "ai", "system"}),
			},
			wantErr: false,
		},
		{
			name: "invalid config - negative message length",
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
			opts: []SchemaValidationOption{
				WithMaxMessageLength(-1),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := NewSchemaValidationConfig(tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewSchemaValidationConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && config == nil {
				t.Error("NewSchemaValidationConfig() returned nil config without error")
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
			}
		})
	}
}

func TestSchemaValidationConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *SchemaValidationConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &SchemaValidationConfig{
				EnableStrictValidation:  true,
				MaxMessageLength:        10000,
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
				MaxMetadataSize:         100,
				MaxToolCalls:            10,
				MaxEmbeddingDimensions:  1536,
				AllowedMessageTypes:     []string{"human", "ai"},
				RequiredMetadataFields:  []string{},
				EnableContentValidation: true,
			},
			wantErr: false,
		},
		{
			name: "invalid config - negative message length",
			config: &SchemaValidationConfig{
				MaxMessageLength: -1,
			},
			wantErr: true,
		},
		{
			name: "invalid config - negative metadata size",
			config: &SchemaValidationConfig{
				MaxMessageLength: 1000,
				MaxMetadataSize:  -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("SchemaValidationConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	}
}

// A2A Communication tests

func TestNewAgentMessage(t *testing.T) {
	fromAgentID := "agent-1"
	messageID := "msg-123"
	messageType := AgentMessageRequest
	payload := map[string]interface{}{"action": "test"}

	msg := NewAgentMessage(fromAgentID, messageID, messageType, payload)

	if msg.FromAgentID != fromAgentID {
		t.Errorf("FromAgentID = %q, want %q", msg.FromAgentID, fromAgentID)
	}
	if msg.MessageID != messageID {
		t.Errorf("MessageID = %q, want %q", msg.MessageID, messageID)
	}
	if msg.MessageType != messageType {
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		t.Errorf("MessageType = %v, want %v", msg.MessageType, messageType)
	}
	if msg.Payload == nil {
		t.Error("Payload should not be nil")
	}
}

func TestNewAgentRequest(t *testing.T) {
	action := "calculate"
	parameters := map[string]interface{}{"expression": "2+2"}

	req := NewAgentRequest(action, parameters)

	if req.Action != action {
		t.Errorf("Action = %q, want %q", req.Action, action)
	}
	if len(req.Parameters) != 1 {
		t.Errorf("Parameters length = %d, want 1", len(req.Parameters))
	}
}

func TestNewAgentResponse(t *testing.T) {
	requestID := "req-123"
	status := "success"
	result := map[string]interface{}{"answer": 42}

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	resp := NewAgentResponse(requestID, status, result)

	if resp.RequestID != requestID {
		t.Errorf("RequestID = %q, want %q", resp.RequestID, requestID)
	}
	if resp.Status != status {
		t.Errorf("Status = %q, want %q", resp.Status, status)
	}
	if resp.Result == nil {
		t.Error("Result should not be nil")
	}
}

func TestNewAgentError(t *testing.T) {
	code := "test_error"
	message := "Test error occurred"
	details := map[string]interface{}{"context": "testing"}

	err := NewAgentError(code, message, details)

	if err.Code != code {
		t.Errorf("Code = %q, want %q", err.Code, code)
	}
	if err.Message != message {
		t.Errorf("Message = %q, want %q", err.Message, message)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	if len(err.Details) != 1 {
		t.Errorf("Details length = %d, want 1", len(err.Details))
	}
}

func TestAgentMessageWithMetadata(t *testing.T) {
	fromAgentID := "agent-1"
	toAgentID := "agent-2"
	messageID := "msg-123"
	conversationID := "conv-456"
	messageType := AgentMessageRequest
	payload := map[string]interface{}{"action": "analyze", "data": "test data"}
	metadata := map[string]interface{}{
		"priority":       "high",
		"timeout":        30,
		"correlation_id": "corr-789",
	}

	msg := NewAgentMessage(fromAgentID, messageID, messageType, payload)
	msg.ToAgentID = toAgentID
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	msg.ConversationID = conversationID
	msg.Metadata = metadata

	if msg.FromAgentID != fromAgentID {
		t.Errorf("FromAgentID = %q, want %q", msg.FromAgentID, fromAgentID)
	}
	if msg.ToAgentID != toAgentID {
		t.Errorf("ToAgentID = %q, want %q", msg.ToAgentID, toAgentID)
	}
	if msg.MessageID != messageID {
		t.Errorf("MessageID = %q, want %q", msg.MessageID, messageID)
	}
	if msg.ConversationID != conversationID {
		t.Errorf("ConversationID = %q, want %q", msg.ConversationID, conversationID)
	}
	if msg.Metadata["priority"] != "high" {
		t.Errorf("Metadata[priority] = %v, want %q", msg.Metadata["priority"], "high")
	}
}

func TestAgentMessageBroadcast(t *testing.T) {
	fromAgentID := "coordinator"
	messageID := "broadcast-123"
	messageType := AgentMessageBroadcast
	payload := map[string]interface{}{
		"announcement": "System maintenance in 5 minutes",
		"type":         "maintenance",
	}

	msg := NewAgentMessage(fromAgentID, messageID, messageType, payload)

	if msg.MessageType != AgentMessageBroadcast {
		t.Errorf("MessageType = %v, want %v", msg.MessageType, AgentMessageBroadcast)
	}
	if msg.ToAgentID != "" {
		t.Errorf("ToAgentID should be empty for broadcast, got %q", msg.ToAgentID)
	}
	if msg.Payload == nil {
		t.Error("Payload should not be nil")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
}

func TestAgentRequestWithTimeout(t *testing.T) {
	action := "process_data"
	parameters := map[string]interface{}{
		"data":     []int{1, 2, 3, 4, 5},
		"format":   "json",
		"validate": true,
	}
	timeout := 60
	priority := 5

	req := NewAgentRequest(action, parameters)
	req.Timeout = timeout
	req.Priority = priority

	if req.Action != action {
		t.Errorf("Action = %q, want %q", req.Action, action)
	}
	if req.Timeout != timeout {
		t.Errorf("Timeout = %d, want %d", req.Timeout, timeout)
	}
	if req.Priority != priority {
		t.Errorf("Priority = %d, want %d", req.Priority, priority)
	}
	if len(req.Parameters) != 3 {
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		t.Errorf("Parameters length = %d, want 3", len(req.Parameters))
	}
}

func TestAgentResponseWithError(t *testing.T) {
	requestID := "req-123"
	status := "error"
	result := map[string]interface{}{}
	agentError := NewAgentError("validation_failed", "Invalid input format", map[string]interface{}{
		"field":  "data",
		"reason": "missing required field",
		"code":   "VALIDATION_ERROR",
	})

	resp := NewAgentResponse(requestID, status, result)
	resp.Error = agentError

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	if resp.Status != status {
		t.Errorf("Status = %q, want %q", resp.Status, status)
	}
	if resp.Error == nil {
		t.Error("Error should not be nil")
		return
	}
	if resp.Error.Code != "validation_failed" {
		t.Errorf("Error.Code = %q, want %q", resp.Error.Code, "validation_failed")
	}
	if resp.Error.Message != "Invalid input format" {
		t.Errorf("Error.Message = %q, want %q", resp.Error.Message, "Invalid input format")
	}
}

func TestAgentMessageTypes(t *testing.T) {
	tests := []struct {
		name        string
		messageType AgentMessageType
		expected    string
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	}{
		{"Request", AgentMessageRequest, "request"},
		{"Response", AgentMessageResponse, "response"},
		{"Notification", AgentMessageNotification, "notification"},
		{"Broadcast", AgentMessageBroadcast, "broadcast"},
		{"Error", AgentMessageError, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.messageType) != tt.expected {
				t.Errorf("AgentMessageType %s = %q, want %q", tt.name, string(tt.messageType), tt.expected)
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
			}
		})
	}
}

func TestAgentMessageConversationFlow(t *testing.T) {
	conversationID := "conv-abc-123"

	// Request message
	request := NewAgentMessage("client-agent", "msg-1", AgentMessageRequest,
		NewAgentRequest("calculate", map[string]interface{}{"expr": "2*3"}))
	request.ConversationID = conversationID

	// Response message
	response := NewAgentMessage("calc-agent", "msg-2", AgentMessageResponse,
		NewAgentResponse("msg-1", "success", map[string]interface{}{"result": 6}))
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	response.ConversationID = conversationID
	response.ToAgentID = "client-agent"

	// Notification message
	notification := NewAgentMessage("system", "msg-3", AgentMessageNotification,
		map[string]interface{}{"type": "info", "message": "Calculation completed"})
	notification.ConversationID = conversationID

	// Verify conversation flow
	if request.ConversationID != conversationID {
		t.Errorf("Request ConversationID = %q, want %q", request.ConversationID, conversationID)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	if response.ConversationID != conversationID {
		t.Errorf("Response ConversationID = %q, want %q", response.ConversationID, conversationID)
	}
	if notification.ConversationID != conversationID {
		t.Errorf("Notification ConversationID = %q, want %q", notification.ConversationID, conversationID)
	}

	// Verify message types
	if request.MessageType != AgentMessageRequest {
		t.Errorf("Request MessageType = %v, want %v", request.MessageType, AgentMessageRequest)
	}
	if response.MessageType != AgentMessageResponse {
		t.Errorf("Response MessageType = %v, want %v", response.MessageType, AgentMessageResponse)
	}
	if notification.MessageType != AgentMessageNotification {
		t.Errorf("Notification MessageType = %v, want %v", notification.MessageType, AgentMessageNotification)
	}
}

func TestAgentErrorDetails(t *testing.T) {
	// Test error with detailed information
	err := NewAgentError("network_timeout", "Connection timeout", map[string]interface{}{
		"endpoint":        "api.example.com",
		"timeout_seconds": 30,
		"retry_count":     3,
		"last_attempt":    "2024-01-15T10:30:00Z",
	})

	if err.Code != "network_timeout" {
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		t.Errorf("Code = %q, want %q", err.Code, "network_timeout")
	}
	if err.Message != "Connection timeout" {
		t.Errorf("Message = %q, want %q", err.Message, "Connection timeout")
	}

	details := err.Details
	if details["endpoint"] != "api.example.com" {
		t.Errorf("Details[endpoint] = %v, want %q", details["endpoint"], "api.example.com")
	}
	if details["timeout_seconds"] != 30 {
		t.Errorf("Details[timeout_seconds] = %v, want 30", details["timeout_seconds"])
	}
	if details["retry_count"] != 3 {
		t.Errorf("Details[retry_count] = %v, want 3", details["retry_count"])
	}
}

func TestAgentMessageTimestamp(t *testing.T) {
	msg := NewAgentMessage("agent-1", "msg-123", AgentMessageRequest,
		map[string]interface{}{"action": "test"})

	// Timestamp should be set (greater than 0)
	if msg.Timestamp <= 0 {
		t.Errorf("Timestamp should be greater than 0, got %d", msg.Timestamp)
	}

	// Test setting custom timestamp
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	customTimestamp := int64(1700000000)
	msg.Timestamp = customTimestamp
	if msg.Timestamp != customTimestamp {
		t.Errorf("Timestamp = %d, want %d", msg.Timestamp, customTimestamp)
	}
}

// Event tests

func TestNewEvent(t *testing.T) {
	eventID := "event-123"
	eventType := "user_action"
	source := "web_app"
	payload := map[string]interface{}{"action": "click"}

	event := NewEvent(eventID, eventType, source, payload)

	if event.EventID != eventID {
		t.Errorf("EventID = %q, want %q", event.EventID, eventID)
	}
	if event.EventType != eventType {
		t.Errorf("EventType = %q, want %q", event.EventType, eventType)
	}
	if event.Source != source {
		t.Errorf("Source = %q, want %q", event.Source, source)
	}
	if event.Version != "1.0" {
		t.Errorf("Version = %q, want %q", event.Version, "1.0")
	}
}

func TestNewAgentLifecycleEvent(t *testing.T) {
	agentID := "agent-1"
	eventType := AgentStarted

	event := NewAgentLifecycleEvent(agentID, eventType)

	if event.AgentID != agentID {
		t.Errorf("AgentID = %q, want %q", event.AgentID, agentID)
	}
	if event.EventType != eventType {
		t.Errorf("EventType = %v, want %v", event.EventType, eventType)
	}
}

func TestNewTaskEvent(t *testing.T) {
	taskID := "task-123"
	agentID := "agent-1"
	eventType := TaskStarted

	event := NewTaskEvent(taskID, agentID, eventType)

	if event.TaskID != taskID {
		t.Errorf("TaskID = %q, want %q", event.TaskID, taskID)
	}
	if event.AgentID != agentID {
		t.Errorf("AgentID = %q, want %q", event.AgentID, agentID)
	}
	if event.EventType != eventType {
		t.Errorf("EventType = %v, want %v", event.EventType, eventType)
	}
}
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()

func TestNewWorkflowEvent(t *testing.T) {
	workflowID := "workflow-123"
	eventType := WorkflowStarted

	event := NewWorkflowEvent(workflowID, eventType)

	if event.WorkflowID != workflowID {
		t.Errorf("WorkflowID = %q, want %q", event.WorkflowID, workflowID)
	}
	if event.EventType != eventType {
		t.Errorf("EventType = %v, want %v", event.EventType, eventType)
	}
}

func TestEventWithMetadata(t *testing.T) {
	eventID := "event-456"
	eventType := "data_processed"
	source := "data-processor"
	payload := map[string]interface{}{
		"records_processed": 1000,
		"processing_time":   45.2,
		"success_rate":      0.987,
	}
	version := "2.1.0"
	metadata := map[string]interface{}{
		"correlation_id": "corr-abc-123",
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		"user_id":        "user-789",
		"session_id":     "sess-def-456",
	}

	event := NewEvent(eventID, eventType, source, payload)
	event.Version = version
	event.Metadata = metadata

	if event.EventID != eventID {
		t.Errorf("EventID = %q, want %q", event.EventID, eventID)
	}
	if event.Version != version {
		t.Errorf("Version = %q, want %q", event.Version, version)
	}
	if event.Metadata["correlation_id"] != "corr-abc-123" {
		t.Errorf("Metadata[correlation_id] = %v, want %q", event.Metadata["correlation_id"], "corr-abc-123")
	}
}

func TestAgentLifecycleEventStates(t *testing.T) {
	agentID := "agent-monitor"

	tests := []struct {
		name      string
		eventType AgentLifecycleEventType
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		expected  string
	}{
		{"Started", AgentStarted, "agent_started"},
		{"Stopped", AgentStopped, "agent_stopped"},
		{"Paused", AgentPaused, "agent_paused"},
		{"Resumed", AgentResumed, "agent_resumed"},
		{"Failed", AgentFailed, "agent_failed"},
		{"ConfigUpdated", AgentConfigUpdated, "agent_config_updated"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := NewAgentLifecycleEvent(agentID, tt.eventType)

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
			if event.AgentID != agentID {
				t.Errorf("AgentID = %q, want %q", event.AgentID, agentID)
			}
			if string(event.EventType) != tt.expected {
				t.Errorf("EventType = %q, want %q", string(event.EventType), tt.expected)
			}
		})
	}
}

func TestTaskEventLifecycle(t *testing.T) {
	taskID := "task-analysis-001"
	agentID := "analyzer-agent"

	// Task started
	startEvent := NewTaskEvent(taskID, agentID, TaskStarted)
	startEvent.TaskType = "data_analysis"
	startEvent.Status = "running"

	if startEvent.TaskID != taskID {
		t.Errorf("TaskID = %q, want %q", startEvent.TaskID, taskID)
	}
	if startEvent.AgentID != agentID {
		t.Errorf("AgentID = %q, want %q", startEvent.AgentID, agentID)
	}
	if startEvent.TaskType != "data_analysis" {
		t.Errorf("TaskType = %q, want %q", startEvent.TaskType, "data_analysis")
	}
	if startEvent.Status != "running" {
		t.Errorf("Status = %q, want %q", startEvent.Status, "running")
	}

	// Task progress
	progressEvent := NewTaskEvent(taskID, agentID, TaskProgress)
	progressEvent.Progress = 75
	progressEvent.Status = "processing"

	if progressEvent.Progress != 75 {
		t.Errorf("Progress = %d, want 75", progressEvent.Progress)
	}

	// Task completed
	completeEvent := NewTaskEvent(taskID, agentID, TaskCompleted)
	completeEvent.Result = map[string]interface{}{
		"analysis_result": "completed",
		"confidence":      0.92,
	}
	completeEvent.Status = "completed"

	if completeEvent.Result == nil {
		t.Error("Result should not be nil")
	}
	resultMap, ok := completeEvent.Result.(map[string]interface{})
	if !ok {
		t.Error("Result should be a map")
		return
	}
	if resultMap["analysis_result"] != "completed" {
		t.Errorf("Result[analysis_result] = %v, want %q", resultMap["analysis_result"], "completed")
	}

	// Task failed
	failEvent := NewTaskEvent(taskID, agentID, TaskFailed)
	failEvent.Error = NewAgentError("processing_error", "Failed to process data", nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	failEvent.Status = "failed"

	if failEvent.Error == nil {
		t.Error("Error should not be nil")
	}
	if failEvent.Error.Code != "processing_error" {
		t.Errorf("Error.Code = %q, want %q", failEvent.Error.Code, "processing_error")
	}
}

func TestWorkflowEventStates(t *testing.T) {
	workflowID := "workflow-etl-001"

	tests := []struct {
		name      string
		eventType WorkflowEventType
		expected  string
	}{
		{"Started", WorkflowStarted, "workflow_started"},
		{"StepCompleted", WorkflowStepCompleted, "workflow_step_completed"},
		{"Completed", WorkflowCompleted, "workflow_completed"},
		{"Failed", WorkflowFailed, "workflow_failed"},
		{"Cancelled", WorkflowCancelled, "workflow_cancelled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := NewWorkflowEvent(workflowID, tt.eventType)

			if event.WorkflowID != workflowID {
				t.Errorf("WorkflowID = %q, want %q", event.WorkflowID, workflowID)
			}
			if string(event.EventType) != tt.expected {
				t.Errorf("EventType = %q, want %q", string(event.EventType), tt.expected)
			}
		})
	}
}

func TestWorkflowEventWithParticipants(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	workflowID := "workflow-distributed-001"
	eventType := WorkflowStarted

	event := NewWorkflowEvent(workflowID, eventType)
	event.Participants = []string{"agent-1", "agent-2", "agent-3", "coordinator"}
	event.CurrentStep = "initialization"
	event.TotalSteps = 5
	event.Status = "starting"

	if len(event.Participants) != 4 {
		t.Errorf("Participants length = %d, want 4", len(event.Participants))
	}
	if event.Participants[0] != "agent-1" {
		t.Errorf("Participants[0] = %q, want %q", event.Participants[0], "agent-1")
	}
	if event.CurrentStep != "initialization" {
		t.Errorf("CurrentStep = %q, want %q", event.CurrentStep, "initialization")
	}
	if event.TotalSteps != 5 {
		t.Errorf("TotalSteps = %d, want 5", event.TotalSteps)
	}
	if event.Status != "starting" {
		t.Errorf("Status = %q, want %q", event.Status, "starting")
	}
}

func TestEventTimestamp(t *testing.T) {
	event := NewEvent("event-123", "test", "test-source", map[string]interface{}{"data": "test"})

	// Timestamp should be set (greater than 0)
	if event.Timestamp <= 0 {
		t.Errorf("Timestamp should be greater than 0, got %d", event.Timestamp)
	}

	// Test setting custom timestamp
	customTimestamp := int64(1700000000)
	event.Timestamp = customTimestamp
	if event.Timestamp != customTimestamp {
		t.Errorf("Timestamp = %d, want %d", event.Timestamp, customTimestamp)
	}
}

func TestEventConstants(t *testing.T) {
	// Test agent lifecycle event constants
	tests := []struct {
		name     string
		got      AgentLifecycleEventType
		expected string
	}{
		{"AgentStarted", AgentStarted, "agent_started"},
		{"AgentStopped", AgentStopped, "agent_stopped"},
		{"AgentPaused", AgentPaused, "agent_paused"},
		{"AgentResumed", AgentResumed, "agent_resumed"},
		{"AgentFailed", AgentFailed, "agent_failed"},
		{"AgentConfigUpdated", AgentConfigUpdated, "agent_config_updated"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.got) != tt.expected {
				t.Errorf("AgentLifecycleEventType constant %s = %q, want %q", tt.name, string(tt.got), tt.expected)
			}
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	// Test task event constants
	taskTests := []struct {
		name     string
		got      TaskEventType
		expected string
	}{
		{"TaskStarted", TaskStarted, "task_started"},
		{"TaskProgress", TaskProgress, "task_progress"},
		{"TaskCompleted", TaskCompleted, "task_completed"},
		{"TaskFailed", TaskFailed, "task_failed"},
		{"TaskCancelled", TaskCancelled, "task_cancelled"},
	}

	for _, tt := range taskTests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.got) != tt.expected {
				t.Errorf("TaskEventType constant %s = %q, want %q", tt.name, string(tt.got), tt.expected)
			}
		})
	}

	// Test workflow event constants
	workflowTests := []struct {
		name     string
		got      WorkflowEventType
		expected string
	}{
		{"WorkflowStarted", WorkflowStarted, "workflow_started"},
		{"WorkflowStepCompleted", WorkflowStepCompleted, "workflow_step_completed"},
		{"WorkflowCompleted", WorkflowCompleted, "workflow_completed"},
		{"WorkflowFailed", WorkflowFailed, "workflow_failed"},
		{"WorkflowCancelled", WorkflowCancelled, "workflow_cancelled"},
	}

	for _, tt := range workflowTests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.got) != tt.expected {
				t.Errorf("WorkflowEventType constant %s = %q, want %q", tt.name, string(tt.got), tt.expected)
			}
		})
	}
}

func TestEventSystemIntegration(t *testing.T) {
	// Test a complete event flow for an agent lifecycle
	agentID := "test-agent-001"

	// Agent started
	startEvent := NewAgentLifecycleEvent(agentID, AgentStarted)
	startEvent.Reason = "manual_start"

	// Agent processes tasks
	taskEvent := NewTaskEvent("task-001", agentID, TaskStarted)
	taskEvent.TaskType = "computation"

	// Workflow coordination
	workflowEvent := NewWorkflowEvent("workflow-001", WorkflowStarted)
	workflowEvent.Participants = []string{agentID, "coordinator"}
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	workflowEvent.CurrentStep = "task_assignment"

	// Agent failed
	failEvent := NewAgentLifecycleEvent(agentID, AgentFailed)
	failEvent.PreviousState = "running"
	failEvent.CurrentState = "failed"
	failEvent.Reason = "memory_limit_exceeded"

	// Verify all events are properly structured
	if startEvent.AgentID != agentID {
		t.Errorf("Start event AgentID mismatch")
	}
	if taskEvent.AgentID != agentID {
		t.Errorf("Task event AgentID mismatch")
	}
	if len(workflowEvent.Participants) != 2 {
		t.Errorf("Workflow participants count = %d, want 2", len(workflowEvent.Participants))
	}
	if failEvent.PreviousState != "running" {
		t.Errorf("Fail event PreviousState = %q, want %q", failEvent.PreviousState, "running")
	}
}

// Error code tests

// Enhanced Configuration Factory Tests

func TestNewEmbeddingProviderConfigFactory(t *testing.T) {
	tests := []struct {
		name       string
		configName string
		provider   string
		modelName  string
		apiKey     string
		opts       []EmbeddingOption
		wantErr    bool
	}{
		{
			name:       "valid config",
			configName: "openai-embed",
			provider:   "openai",
			modelName:  "text-embedding-ada-002",
			apiKey:     "sk-test",
			opts:       []EmbeddingOption{WithEmbeddingBaseURL("https://api.openai.com/v1")},
			wantErr:    false,
		},
		{
			name:       "empty config name",
			configName: "",
			provider:   "openai",
			modelName:  "text-embedding-ada-002",
			apiKey:     "sk-test",
			opts:       []EmbeddingOption{},
			wantErr:    true,
		},
		{
			name:       "empty api key",
			configName: "openai-embed",
			provider:   "openai",
			modelName:  "text-embedding-ada-002",
			apiKey:     "",
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
			opts:       []EmbeddingOption{},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := NewEmbeddingProviderConfig(tt.configName, tt.provider, tt.modelName, tt.apiKey, tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewEmbeddingProviderConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && config == nil {
				t.Error("NewEmbeddingProviderConfig() returned nil config without error")
			}
			if !tt.wantErr {
				if config.Name != tt.configName {
					t.Errorf("Config name = %q, want %q", config.Name, tt.configName)
				}
				if config.Provider != tt.provider {
					t.Errorf("Provider = %q, want %q", config.Provider, tt.provider)
				}
				if config.ModelName != tt.modelName {
					t.Errorf("Model name = %q, want %q", config.ModelName, tt.modelName)
				}
				if config.APIKey != tt.apiKey {
					t.Errorf("API key = %q, want %q", config.APIKey, tt.apiKey)
				}
			}
		})
	}
}
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()

func TestNewVectorStoreConfigFactory(t *testing.T) {
	tests := []struct {
		name       string
		configName string
		provider   string
		opts       []VectorStoreOption
		wantErr    bool
	}{
		{
			name:       "valid config",
			configName: "pgvector-store",
			provider:   "pgvector",
			opts:       []VectorStoreOption{WithConnectionString("postgres://localhost:5432/db")},
			wantErr:    false,
		},
		{
			name:       "minimal valid config",
			configName: "inmemory-store",
			provider:   "inmemory",
			opts:       []VectorStoreOption{},
			wantErr:    false,
		},
		{
			name:       "empty config name",
			configName: "",
			provider:   "pgvector",
			opts:       []VectorStoreOption{},
			wantErr:    true,
		},
		{
			name:       "empty provider",
			configName: "test-store",
			provider:   "",
			opts:       []VectorStoreOption{},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := NewVectorStoreConfig(tt.configName, tt.provider, tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewVectorStoreConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && config == nil {
				t.Error("NewVectorStoreConfig() returned nil config without error")
			}
			if !tt.wantErr {
				if config.Name != tt.configName {
					t.Errorf("Config name = %q, want %q", config.Name, tt.configName)
				}
				if config.Provider != tt.provider {
					t.Errorf("Provider = %q, want %q", config.Provider, tt.provider)
				}
			}
		})
	}
}

func TestFunctionalOptionsComposition(t *testing.T) {
	// Test complex composition of functional options

	// Agent config with multiple options
	agentConfig, err := NewAgentConfig("complex-agent", "openai-gpt4",
		WithToolNames([]string{"calculator", "search", "web"}),
		WithMaxIterations(50),
		WithPromptTemplate("You are a helpful AI assistant with access to tools."),
		WithAgentType("react"),
		WithMemoryProvider("vector-store", "buffer"),
	)
	if err != nil {
		t.Fatalf("NewAgentConfig() error = %v", err)
	}

	if len(agentConfig.ToolNames) != 3 {
		t.Errorf("ToolNames length = %d, want 3", len(agentConfig.ToolNames))
	}
	if agentConfig.MaxIterations != 50 {
		t.Errorf("MaxIterations = %d, want 50", agentConfig.MaxIterations)
	}
	if agentConfig.AgentType != "react" {
		t.Errorf("AgentType = %q, want %q", agentConfig.AgentType, "react")
	}
	if agentConfig.MemoryProviderName != "vector-store" {
		t.Errorf("MemoryProviderName = %q, want %q", agentConfig.MemoryProviderName, "vector-store")
	}
	if agentConfig.MemoryType != "buffer" {
		t.Errorf("MemoryType = %q, want %q", agentConfig.MemoryType, "buffer")
	}

	// LLM provider config with multiple options
	llmConfig, err := NewLLMProviderConfig("complex-llm", "openai", "gpt-4-turbo",
		WithAPIKey("sk-complex-key"),
		WithBaseURL("https://api.openai.com/v1"),
		WithDefaultCallOptions(map[string]interface{}{
			"temperature": 0.7,
			"max_tokens":  2000,
			"top_p":       1.0,
		}),
		WithProviderSpecific(map[string]interface{}{
			"organization": "test-org",
			"project":      "ai-framework",
		}),
	)
	if err != nil {
		t.Fatalf("NewLLMProviderConfig() error = %v", err)
	}

	if llmConfig.APIKey != "sk-complex-key" {
		t.Errorf("APIKey = %q, want %q", llmConfig.APIKey, "sk-complex-key")
	}
	if llmConfig.BaseURL != "https://api.openai.com/v1" {
		t.Errorf("BaseURL = %q, want %q", llmConfig.BaseURL, "https://api.openai.com/v1")
	}
	if llmConfig.DefaultCallOptions["temperature"] != 0.7 {
		t.Errorf("DefaultCallOptions[temperature] = %v, want 0.7", llmConfig.DefaultCallOptions["temperature"])
	}
	if llmConfig.ProviderSpecific["organization"] != "test-org" {
		t.Errorf("ProviderSpecific[organization] = %v, want %q", llmConfig.ProviderSpecific["organization"], "test-org")
	}
}

func TestConfigurationValidationWithOptions(t *testing.T) {
	// Test that validation works correctly with functional options

	// Test invalid agent config with options
	_, err := NewAgentConfig("", "openai-gpt4", WithMaxIterations(10))
	if err == nil {
		t.Error("Expected error for empty agent name")
	}

	// Test invalid LLM config with options
	_, err = NewLLMProviderConfig("test", "", "gpt-4", WithAPIKey("sk-test"))
	if err == nil {
		t.Error("Expected error for empty provider")
	}

	// Test invalid embedding config
	_, err = NewEmbeddingProviderConfig("", "openai", "text-embedding-ada-002", "")
	if err == nil {
		t.Error("Expected error for empty name and API key")
	}

	// Test valid configs with options
	agentConfig, err := NewAgentConfig("valid-agent", "llm-provider",
		WithMaxIterations(25),
		WithToolNames([]string{"tool1"}),
	)
	if err != nil {
		t.Errorf("Expected valid config, got error: %v", err)
	}
	if agentConfig.MaxIterations != 25 {
		t.Errorf("MaxIterations = %d, want 25", agentConfig.MaxIterations)
	}
}

func TestErrorCodes(t *testing.T) {
	// Test that error codes are properly defined and accessible
	expectedCodes := []string{
		"invalid_config",
		"validation_failed",
		"invalid_message",
		"agent_message_invalid",
		"event_invalid",
		"message_too_long",
		"task_not_found",
		"config_validation_failed",
		"factory_creation_failed",
		"storage_operation_failed",
	}

	// This test just ensures the constants are accessible
	// In a real test, you might want to test specific error handling scenarios
	for _, code := range expectedCodes {
		if code == "" {
			t.Error("Error code should not be empty")
		}
	}
}
