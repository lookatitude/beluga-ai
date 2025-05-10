package integration_tests

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"


	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools/providers"
	"github.com/lookatitude/beluga-ai/pkg/config"
	"github.com/lookatitude/beluga-ai/pkg/embeddings"
	// "github.com/lookatitude/beluga-ai/pkg/embeddings/iface" // Unused import removed
	"github.com/lookatitude/beluga-ai/pkg/llms"
	"github.com/lookatitude/beluga-ai/pkg/memory"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	// Ensure provider packages are imported for their init() side effects (registration)
	_ "github.com/lookatitude/beluga-ai/pkg/embeddings/openai"
	_ "github.com/lookatitude/beluga-ai/pkg/llms/mock" // Ensure mock LLM provider is registered
	_ "github.com/lookatitude/beluga-ai/pkg/llms/openai"
)

// mockPhase1LLM is a mock implementation of the llms.LLM interface for testing.
type mockPhase1LLM struct {
	t                 *testing.T
	callCount         int
	expectedToolInput string
	// mockAgent         base.Agent // To access agent methods if needed, e.g. for parsing. Not used in current mock logic.
}

func newMockPhase1LLM(t *testing.T, expectedToolInput string) *mockPhase1LLM {
	return &mockPhase1LLM{t: t, expectedToolInput: expectedToolInput}
}

// Invoke matches the llms.LLM interface, using schema.LLMOption.
func (m *mockPhase1LLM) Invoke(ctx context.Context, prompt string, callOptions ...schema.LLMOption) (string, error) {
	m.callCount++ // Counting Invoke calls as well for completeness, though test flow uses Chat.
	m.t.Logf("MockLLM.Invoke called (Call #%d) with prompt: %s", m.callCount, prompt)
	// This mock will primarily use its Chat-like behavior for structured output simulation.
	// If an agent directly called Invoke for a simple completion, this is where it would be handled.
	// For this test, we simulate the agent making a decision that leads to a tool call or final answer.

	// Simulate a scenario where Invoke is used to get a tool call (less common for BaseAgent which prefers Chat for structured I/O)
	if m.callCount == 1 {
		// This part is more conceptual as BaseAgent would use Chat for tool decisions.
		// However, to make Invoke testable in a similar flow if needed:
		argsJSON := fmt.Sprintf(`{"input": "%s"}`, m.expectedToolInput)
		// Invoke returns a string. How to represent a tool call?
		// This highlights why Chat (returning structured AIChatMessage) is better for tool use.
		// For this test, we will rely on the test calling a Chat-like method on the mock directly.
		// So, this Invoke implementation is more of a basic placeholder.
		return fmt.Sprintf("LLM decided to call EchoTool with args: %s", argsJSON), nil
	}
	return "Invoke response: Final answer from LLM after tool use.", nil
}

// Chat is a method on the mock, not necessarily on the llms.LLM interface itself.
// The test will call this method to simulate an agent receiving structured messages from an LLM.
func (m *mockPhase1LLM) Chat(ctx context.Context, messages []schema.Message, callOptions ...schema.LLMOption) (*schema.AIChatMessage, error) {
	m.callCount++ // This callCount is specific to this Chat method for the test flow.
	m.t.Logf("MockLLM.Chat called (Chat Call #%d)", m.callCount)

	if m.callCount == 1 {
		m.t.Log("MockLLM (Chat Call 1): Responding with EchoTool call.")
		argsJSON := fmt.Sprintf(`{"input": "%s"}`, m.expectedToolInput)
		return &schema.AIChatMessage{
			BaseMessage: schema.BaseMessage{Content: "I will use the EchoTool to process your request."},
			Role:        schema.RoleAssistant,
			ToolCalls: []schema.ToolCall{
				{
					ID:   "mock-tool-call-id-123",
					Type: "function",
					Function: schema.ToolFunction{
						Name:      "EchoTool",
						Arguments: argsJSON,
					},
				},
			},
		}, nil
	} else if m.callCount == 2 {
		m.t.Log("MockLLM (Chat Call 2): Responding with final answer based on tool observation.")
		var observationContent string
		for _, msg := range messages {
			if toolMsg, ok := msg.(*schema.ToolMessage); ok {
				if toolMsg.ToolCallID == "mock-tool-call-id-123" {
					observationContent = toolMsg.GetContent()
					break
				}
			}
		}
		require.NotEmpty(m.t, observationContent, "LLM did not receive observation from tool in messages")
		finalAnswer := fmt.Sprintf("The phrase has been echoed by the tool: %s", observationContent)
		return &schema.AIChatMessage{
			BaseMessage: schema.BaseMessage{Content: finalAnswer},
			Role:        schema.RoleAssistant,
			ToolCalls:   nil,
		}, nil
	}
	return nil, fmt.Errorf("mockPhase1LLM.Chat received unexpected call number: %d", m.callCount)
}

func (m *mockPhase1LLM) GetModelName() string {
	return "mock-phase1-llm"
}

func (m *mockPhase1LLM) GetProviderName() string {
	return "mock"
}

var _ llms.LLM = (*mockPhase1LLM)(nil) // Ensures Invoke, GetModelName, GetProviderName match.

func createTempPhase1ConfigFile(t *testing.T, openAILLMAPIKey, openAIEmbedderAPIKey string) (string, string, func()) {
	t.Helper()
	configContent := fmt.Sprintf(`
llm_providers:
  openai_default:
    provider: "openai"
    model_name: "gpt-3.5-turbo-instruct"
    api_key: "%s"
  mock_llm:
    provider: "mock"
    model_name: "mock-phase1-llm"

embeddings:
  provider: "openai"
  openai:
    api_key: "%s"
    model: "text-embedding-ada-002"

tools:
`, openAILLMAPIKey, openAIEmbedderAPIKey)

	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test_phase1_config.yaml")
	err := os.WriteFile(configFile, []byte(configContent), 0600)
	require.NoError(t, err)
	return tempDir, "test_phase1_config", func() { os.RemoveAll(tempDir) }
}

func TestPhase1AgentIntegration(t *testing.T) {
	ctx := context.Background()
	tempDir, configName, cleanup := createTempPhase1ConfigFile(t, "sk-test-llm-key-placeholder", "sk-test-embedder-key-placeholder")
	defer cleanup()

	vp, err := config.NewViperProvider(configName, []string{tempDir}, "BELUGA_TEST_PHASE1")
	require.NoError(t, err, "Failed to create ViperProvider")
	require.NotNil(t, vp, "ViperProvider is nil")

	embedderProvider, err := embeddings.NewEmbedderProvider(vp)
	require.NoError(t, err, "Failed to create EmbedderProvider from config")
	require.NotNil(t, embedderProvider, "EmbedderProvider is nil")

	if openAIEmb, ok := embedderProvider.(interface{ GetProviderName() string }); ok {
		assert.Equal(t, "openai", openAIEmb.GetProviderName(), "Embedder provider name mismatch after type assertion")
	} else {
		t.Log("Embedder provider does not have GetProviderName method, skipping name check or specific type assertion failed.")
	}

	userInputPhrase := "Hello Beluga, please echo this for me!"
	mockLLM := newMockPhase1LLM(t, userInputPhrase)

	toolRegistry := tools.NewInMemoryToolRegistry()
	echoTool, err := providers.NewEchoTool(config.ToolConfig{Name: "EchoTool", Description: "Echoes input"})
	require.NoError(t, err, "Failed to create EchoTool")
	err = toolRegistry.RegisterTool(echoTool)
	require.NoError(t, err, "Failed to register EchoTool")

	// Corrected BufferMemory initialization
	// NewBufferMemory(returnMessages bool, inputKey, outputKey, memoryKey string)
	bufferMemory := memory.NewBufferMemory(true, "input", "output", "history")
	require.NotNil(t, bufferMemory, "BufferMemory is nil")

	// --- Manual Orchestration of Agent Flow ---

	// 1. Initial User Input - Save to Memory
	// BufferMemory.SaveContext expects inputs[InputKey] and outputs[OutputKey]
	// For initial user message, there's no prior AI output to pair with for SaveContext.
	// So, we directly add the HumanMessage to the ChatHistory.
	initialUserMessage := schema.NewHumanMessage(userInputPhrase)
	err = bufferMemory.ChatHistory.AddMessage(initialUserMessage)
	require.NoError(t, err, "Failed to add initial user message to memory")

	// 2. First LLM Call (Simulated)
	memOutput1, err := bufferMemory.LoadMemoryVariables(ctx, nil)
	require.NoError(t, err)
	history1, ok := memOutput1[bufferMemory.MemoryKey].([]schema.Message)
	require.True(t, ok, "Memory did not return []schema.Message for history1")

	aiResponse1, err := mockLLM.Chat(ctx, history1, schema.WithStreaming(false)) // Pass some LLMOption
	require.NoError(t, err, "mockLLM.Chat (call 1) failed")
	require.NotNil(t, aiResponse1, "mockLLM.Chat (call 1) returned nil response")
	require.NotEmpty(t, aiResponse1.ToolCalls, "mockLLM.Chat (call 1) did not return tool calls")

	// Save AI's response (tool call proposal) to memory
	err = bufferMemory.ChatHistory.AddMessage(aiResponse1)
	require.NoError(t, err, "Failed to add AI tool call message to memory")

	// 3. Tool Execution
	toolCall := aiResponse1.ToolCalls[0]
	assert.Equal(t, "EchoTool", toolCall.Function.Name)

	var toolInputArgs map[string]interface{}
	err = json.Unmarshal([]byte(toolCall.Function.Arguments), &toolInputArgs)
	require.NoError(t, err, "Failed to unmarshal tool arguments")

	toolOutputString, err := echoTool.Execute(ctx, toolInputArgs)
	require.NoError(t, err, "EchoTool.Execute failed")

	// Save Tool Observation to memory
	toolMessage := schema.NewToolMessage(toolOutputString, toolCall.ID)
	err = bufferMemory.ChatHistory.AddMessage(toolMessage)
	require.NoError(t, err, "Failed to add tool observation message to memory")

	// 4. Second LLM Call (Simulated)
	memOutput2, err := bufferMemory.LoadMemoryVariables(ctx, nil)
	require.NoError(t, err)
	history2, ok := memOutput2[bufferMemory.MemoryKey].([]schema.Message)
	require.True(t, ok, "Memory did not return []schema.Message for history2")

	aiResponse2, err := mockLLM.Chat(ctx, history2, schema.WithStreaming(false)) // Pass some LLMOption
	require.NoError(t, err, "mockLLM.Chat (call 2) failed")
	require.NotNil(t, aiResponse2, "mockLLM.Chat (call 2) returned nil response")
	require.Empty(t, aiResponse2.ToolCalls, "mockLLM.Chat (call 2) should not have tool calls")

	// Save final AI answer to memory
	err = bufferMemory.ChatHistory.AddMessage(aiResponse2)
	require.NoError(t, err, "Failed to add final AI answer to memory")

	// Construct FinalAnswer for assertion based on the orchestrated flow
	finalResult := schema.FinalAnswer{Output: aiResponse2.GetContent()}

	// --- Assertions ---
	require.NotNil(t, finalResult, "Final result is nil")
	expectedFinalAnswerContent := fmt.Sprintf("The phrase has been echoed by the tool: Echo: %s", userInputPhrase)
	assert.Equal(t, expectedFinalAnswerContent, finalResult.Output, "Final answer output mismatch")

	// Assert Memory State
	finalMemOutput, err := bufferMemory.LoadMemoryVariables(ctx, nil)
	require.NoError(t, err, "Failed to load final memory variables")
	finalChatHistory, ok := finalMemOutput[bufferMemory.MemoryKey].([]schema.Message)
	require.True(t, ok, "Final memory did not return []schema.Message")
	require.Len(t, finalChatHistory, 4, "Incorrect number of messages in history")

	// Message 1: User Input (HumanMessage)
	humanMsg, ok := finalChatHistory[0].(*schema.HumanMessage)
	require.True(t, ok, "First message not HumanMessage, got %T", finalChatHistory[0])
	assert.Equal(t, userInputPhrase, humanMsg.GetContent(), "User input in memory mismatch")

	// Message 2: AI Action (AIChatMessage with ToolCall)
	aiToolCallMsg, ok := finalChatHistory[1].(*schema.AIChatMessage)
	require.True(t, ok, "Second message not AIChatMessage, got %T", finalChatHistory[1])
	require.Len(t, aiToolCallMsg.ToolCalls, 1, "AI message should have one tool call")
	assert.Equal(t, "EchoTool", aiToolCallMsg.ToolCalls[0].Function.Name, "Tool call name mismatch")
	expectedArgsJSON := fmt.Sprintf(`{"input": "%s"}`, userInputPhrase)
	assert.JSONEq(t, expectedArgsJSON, aiToolCallMsg.ToolCalls[0].Function.Arguments, "Tool call arguments mismatch")

	// Message 3: Tool Observation (ToolMessage)
	toolResultMsg, ok := finalChatHistory[2].(*schema.ToolMessage)
	require.True(t, ok, "Third message not ToolMessage, got %T", finalChatHistory[2])
	assert.Equal(t, fmt.Sprintf("Echo: %s", userInputPhrase), toolResultMsg.GetContent(), "Tool observation content mismatch")
	assert.Equal(t, "mock-tool-call-id-123", toolResultMsg.ToolCallID, "Tool call ID in observation mismatch")

	// Message 4: AI Final Answer (AIChatMessage)
	aiFinalAnswerMsg, ok := finalChatHistory[3].(*schema.AIChatMessage)
	require.True(t, ok, "Fourth message not AIChatMessage, got %T", finalChatHistory[3])
	assert.Equal(t, expectedFinalAnswerContent, aiFinalAnswerMsg.GetContent(), "AI final answer in memory mismatch")
	assert.Empty(t, aiFinalAnswerMsg.ToolCalls, "Final AI message should not have tool calls")

	t.Log("Phase 1 Component Integration Test (Manual Orchestration) Passed!")
}

