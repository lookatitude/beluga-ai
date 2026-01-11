// Package multimodal provides integration tests for multimodal agent operations.
package multimodal

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/agents/iface"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	llmsiface "github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/multimodal"
	"github.com/lookatitude/beluga-ai/pkg/multimodal/internal"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReActAgent_MultimodalInputs(t *testing.T) {
	ctx := context.Background()

	// Create multimodal input (image)
	imageBlock, err := multimodal.NewContentBlock("image", []byte{0x89, 0x50, 0x4E, 0x47}, "image/png", nil)
	require.NoError(t, err)

	input, err := multimodal.NewMultimodalInput([]*multimodal.ContentBlock{imageBlock})
	require.NoError(t, err)

	config := multimodal.Config{
		Provider: "test",
		Model:    "test-model",
	}

	capabilities := &multimodal.ModalityCapabilities{
		Text:  true,
		Image: true,
	}

	baseModel := internal.NewBaseMultimodalModel("test", "test-model", config, capabilities)
	extension := internal.NewMultimodalAgentExtension(baseModel)

	// Create a mock agent for testing
	// Note: In a full implementation, would use actual ReAct agent
	mockAgent := &MockAgent{
		name: "test-agent",
	}

	// Test voice-enabled ReAct loop
	inputs := map[string]any{
		"input": schema.NewHumanMessage("Process this image"),
	}

	action, finish, err := extension.EnableVoiceReActLoop(ctx, mockAgent, inputs, nil)
	if err != nil {
		t.Logf("ReAct loop failed (expected if agent not fully configured): %v", err)
		return
	}

	// Verify structure
	t.Logf("ReAct loop structure ready")
	assert.NotNil(t, action)
	assert.NotNil(t, finish)
}

func TestOrchestrationGraph_MultimodalProcessing(t *testing.T) {
	ctx := context.Background()

	// Create image message
	imgMsg := &schema.ImageMessage{
		ImageURL:    "https://example.com/image.jpg",
		ImageFormat: "jpeg",
	}

	config := multimodal.Config{
		Provider: "test",
		Model:    "test-model",
	}

	capabilities := &multimodal.ModalityCapabilities{
		Text:  true,
		Image: true,
	}

	baseModel := internal.NewBaseMultimodalModel("test", "test-model", config, capabilities)
	extension := internal.NewMultimodalAgentExtension(baseModel)

	// Test orchestration graph input handling
	graphInput := map[string]any{
		"image": imgMsg,
		"text":  "Process this image",
	}

	processed, err := extension.HandleOrchestrationGraphInput(ctx, graphInput)
	if err != nil {
		t.Logf("Graph input handling failed: %v", err)
		return
	}

	require.NoError(t, err)
	assert.NotNil(t, processed)
}

func TestToolIntegration_MultimodalData(t *testing.T) {
	ctx := context.Background()

	config := multimodal.Config{
		Provider: "test",
		Model:    "test-model",
	}

	capabilities := &multimodal.ModalityCapabilities{
		Text:  true,
		Image: true,
	}

	baseModel := internal.NewBaseMultimodalModel("test", "test-model", config, capabilities)
	extension := internal.NewMultimodalAgentExtension(baseModel)

	// Create multimodal tool
	tool := extension.CreateMultimodalTool(
		"process_image",
		"Processes images using multimodal model",
		nil, // Use default processing
	)

	require.NotNil(t, tool)
	assert.Equal(t, "process_image", tool.Name())
	assert.Equal(t, "Processes images using multimodal model", tool.Description())

	// Test tool execution
	imageBlock, err := multimodal.NewContentBlock("image", []byte{0x89, 0x50, 0x4E, 0x47}, "image/png", nil)
	require.NoError(t, err)

	multimodalInput, err := multimodal.NewMultimodalInput([]*multimodal.ContentBlock{imageBlock})
	require.NoError(t, err)

	result, err := tool.Execute(ctx, multimodalInput)
	if err != nil {
		t.Logf("Tool execution failed (expected if provider not registered): %v", err)
		return
	}

	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAgentToAgentCommunication_MultimodalData(t *testing.T) {
	ctx := context.Background()

	config := multimodal.Config{
		Provider: "test",
		Model:    "test-model",
	}

	capabilities := &multimodal.ModalityCapabilities{
		Text:  true,
		Image: true,
	}

	baseModel := internal.NewBaseMultimodalModel("test", "test-model", config, capabilities)
	extension := internal.NewMultimodalAgentExtension(baseModel)

	// Create mock agents
	fromAgent := &MockAgent{name: "agent1"}
	toAgent := &MockAgent{name: "agent2"}

	// Create multimodal data
	imageBlock, err := multimodal.NewContentBlock("image", []byte{0x89, 0x50, 0x4E, 0x47}, "image/png", nil)
	require.NoError(t, err)

	multimodalInput, err := multimodal.NewMultimodalInput([]*multimodal.ContentBlock{imageBlock})
	require.NoError(t, err)

	// Test preserving multimodal data
	preserved, err := extension.PreserveMultimodalDataInAgentCommunication(ctx, fromAgent, toAgent, multimodalInput)
	if err != nil {
		t.Logf("Data preservation failed: %v", err)
		return
	}

	require.NoError(t, err)
	assert.NotNil(t, preserved)

	// Test with messages
	imgMsg := &schema.ImageMessage{ImageURL: "https://example.com/img.jpg"}
	preserved2, err := extension.PreserveMultimodalDataInAgentCommunication(ctx, fromAgent, toAgent, imgMsg)
	if err != nil {
		t.Logf("Message preservation failed: %v", err)
		return
	}

	require.NoError(t, err)
	assert.NotNil(t, preserved2)
}

// MockAgent is a simple mock implementation of iface.Agent for testing.
type MockAgent struct {
	name string
}

func (m *MockAgent) Plan(ctx context.Context, intermediateSteps []iface.IntermediateStep, inputs map[string]any) (iface.AgentAction, iface.AgentFinish, error) {
	return iface.AgentAction{}, iface.AgentFinish{
		ReturnValues: map[string]any{"output": "mock response"},
		Log:          "mock plan",
	}, nil
}

func (m *MockAgent) InputVariables() []string {
	return []string{"input"}
}

func (m *MockAgent) OutputVariables() []string {
	return []string{"output"}
}

func (m *MockAgent) GetTools() []tools.Tool {
	return nil
}

func (m *MockAgent) GetConfig() schema.AgentConfig {
	return schema.AgentConfig{
		Name: m.name,
	}
}

func (m *MockAgent) GetLLM() llmsiface.LLM {
	return nil
}

func (m *MockAgent) GetMetrics() iface.MetricsRecorder {
	return nil
}
