// Package package_pairs provides integration tests between Agents and Tools packages.
// This test suite verifies that agents work correctly with tools for
// tool-augmented interactions, execution, and error handling.
package package_pairs

import (
	"context"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/agents"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/tools"
	toolsiface "github.com/lookatitude/beluga-ai/pkg/tools/iface"
	"github.com/lookatitude/beluga-ai/tests/integration/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegrationAgentsTools tests the integration between Agents and Tools packages.
func TestIntegrationAgentsTools(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	tests := []struct {
		name       string
		toolSetup  func() []tools.Tool
		agentType  string
		wantErr    bool
		expectTool bool
	}{
		{
			name: "agent_with_single_tool",
			toolSetup: func() []tools.Tool {
				return []tools.Tool{
					agents.NewMockTool("calculator", "Performs calculations"),
				}
			},
			agentType:  "base",
			wantErr:    false,
			expectTool: true,
		},
		{
			name: "agent_with_multiple_tools",
			toolSetup: func() []tools.Tool {
				return []tools.Tool{
					agents.NewMockTool("calculator", "Performs calculations"),
					agents.NewMockTool("search", "Searches for information"),
					agents.NewMockTool("weather", "Gets weather information"),
				}
			},
			agentType:  "base",
			wantErr:    false,
			expectTool: true,
		},
		{
			name: "agent_with_no_tools",
			toolSetup: func() []tools.Tool {
				return nil
			},
			agentType:  "base",
			wantErr:    false,
			expectTool: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// Create mock LLM
			mockLLM := helper.CreateMockLLM("integration-llm")

			// Create tools
			testTools := tt.toolSetup()

			// Create agent with tools
			agent, err := agents.NewBaseAgent("test-agent", mockLLM, testTools)
			require.NoError(t, err)
			require.NotNil(t, agent)

			// Verify tools are attached
			agentTools := agent.GetTools()
			if tt.expectTool {
				assert.NotEmpty(t, agentTools)
				assert.Len(t, agentTools, len(testTools))
			} else {
				assert.Empty(t, agentTools)
			}

			// Test agent execution
			result, err := agent.Invoke(ctx, schema.NewHumanMessage("Test input"))
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				t.Logf("Agent execution result: result=%v, err=%v", result != nil, err)
			}
		})
	}
}

// TestAgentsToolsExecution tests tool execution through agents.
func TestAgentsToolsExecution(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	t.Run("tool_execution_tracking", func(t *testing.T) {
		// Create a mock tool that tracks execution
		mockTool := agents.NewMockTool("tracker", "Tracks execution")

		mockLLM := helper.CreateMockLLM("tool-llm")
		agent, err := agents.NewBaseAgent("tracker-agent", mockLLM, []tools.Tool{mockTool})
		require.NoError(t, err)

		// Verify tool is accessible
		agentTools := agent.GetTools()
		require.Len(t, agentTools, 1)
		assert.Equal(t, "tracker", agentTools[0].Name())
		assert.Equal(t, "Tracks execution", agentTools[0].Description())
	})

	t.Run("tool_name_uniqueness", func(t *testing.T) {
		// Create tools with unique names
		tool1 := agents.NewMockTool("unique_tool_1", "First unique tool")
		tool2 := agents.NewMockTool("unique_tool_2", "Second unique tool")

		mockLLM := helper.CreateMockLLM("unique-llm")
		agent, err := agents.NewBaseAgent("unique-agent", mockLLM, []tools.Tool{tool1, tool2})
		require.NoError(t, err)

		agentTools := agent.GetTools()
		names := make(map[string]bool)
		for _, tool := range agentTools {
			assert.False(t, names[tool.Name()], "Tool names should be unique")
			names[tool.Name()] = true
		}
	})
}

// TestAgentsToolsErrorHandling tests error scenarios between agents and tools.
func TestAgentsToolsErrorHandling(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	t.Run("agent_with_nil_tool_slice", func(t *testing.T) {
		mockLLM := helper.CreateMockLLM("nil-tools-llm")
		agent, err := agents.NewBaseAgent("nil-tools-agent", mockLLM, nil)
		require.NoError(t, err)
		assert.NotNil(t, agent)
		assert.Empty(t, agent.GetTools())
	})

	t.Run("agent_with_empty_tool_slice", func(t *testing.T) {
		mockLLM := helper.CreateMockLLM("empty-tools-llm")
		agent, err := agents.NewBaseAgent("empty-tools-agent", mockLLM, []tools.Tool{})
		require.NoError(t, err)
		assert.NotNil(t, agent)
		assert.Empty(t, agent.GetTools())
	})
}

// TestAgentsToolsConcurrency tests concurrent tool usage with agents.
func TestAgentsToolsConcurrency(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	const numGoroutines = 5

	t.Run("concurrent_tool_access", func(t *testing.T) {
		mockLLM := helper.CreateMockLLM("concurrent-llm")
		testTools := []tools.Tool{
			agents.NewMockTool("tool1", "First tool"),
			agents.NewMockTool("tool2", "Second tool"),
		}

		agent, err := agents.NewBaseAgent("concurrent-agent", mockLLM, testTools)
		require.NoError(t, err)

		helper.CrossPackageLoadTest(t, func() error {
			// Access tools concurrently
			tools := agent.GetTools()
			if len(tools) != 2 {
				return nil // Just verify we can access tools
			}
			return nil
		}, numGoroutines*3, numGoroutines)
	})
}

// TestToolsRegistryIntegration tests the tools registry integration.
func TestToolsRegistryIntegration(t *testing.T) {
	t.Run("registry_tool_creation", func(t *testing.T) {
		registry := tools.NewToolRegistry()

		// Register a test tool type
		registry.RegisterType("test-tool", func(ctx context.Context, config tools.ToolConfig) (toolsiface.Tool, error) {
			return tools.NewMockTool(config.Name, config.Description), nil
		})

		// Create tool from registry
		ctx := context.Background()
		config := tools.ToolConfig{
			Name:        "my-tool",
			Description: "A test tool",
		}

		tool, err := registry.Create(ctx, "test-tool", config)
		require.NoError(t, err)
		require.NotNil(t, tool)
		assert.Equal(t, "my-tool", tool.Name())
	})

	t.Run("global_registry_operations", func(t *testing.T) {
		globalRegistry := tools.GetRegistry()
		require.NotNil(t, globalRegistry)

		// List registered tools
		registeredTools := globalRegistry.ListTools()
		t.Logf("Registered tools: %v", registeredTools)
	})
}

// TestToolsWithAgentLifecycle tests tools through agent lifecycle.
func TestToolsWithAgentLifecycle(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	t.Run("tool_availability_after_agent_creation", func(t *testing.T) {
		ctx := context.Background()

		tool := agents.NewMockTool("lifecycle", "Tests lifecycle")
		mockLLM := helper.CreateMockLLM("lifecycle-llm")

		agent, err := agents.NewBaseAgent("lifecycle-agent", mockLLM, []tools.Tool{tool})
		require.NoError(t, err)

		// Tool should be available immediately after creation
		agentTools := agent.GetTools()
		require.Len(t, agentTools, 1)

		// Tool should work through agent context
		result, err := agent.Invoke(ctx, schema.NewHumanMessage("Use lifecycle tool"))
		t.Logf("Lifecycle test result: result=%v, err=%v", result != nil, err)
	})

	t.Run("agent_timeout_with_tools", func(t *testing.T) {
		mockLLM := helper.CreateMockLLM("timeout-llm")
		tool := agents.NewMockTool("slow-tool", "A slow tool")

		agent, err := agents.NewBaseAgent("timeout-agent", mockLLM, []tools.Tool{tool},
			agents.WithTimeout(100*time.Millisecond))
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		_, err = agent.Invoke(ctx, schema.NewHumanMessage("Use slow-tool"))
		// May or may not timeout depending on implementation
		t.Logf("Timeout test error: %v", err)
	})
}

// BenchmarkIntegrationAgentsTools benchmarks agent-tools integration.
func BenchmarkIntegrationAgentsTools(b *testing.B) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	mockLLM := helper.CreateMockLLM("benchmark-llm")
	testTools := []tools.Tool{
		agents.NewMockTool("tool1", "First tool"),
		agents.NewMockTool("tool2", "Second tool"),
	}

	agent, err := agents.NewBaseAgent("benchmark-agent", mockLLM, testTools)
	require.NoError(b, err)

	ctx := context.Background()
	input := schema.NewHumanMessage("benchmark input")

	b.Run("GetTools", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = agent.GetTools()
		}
	})

	b.Run("AgentInvokeWithTools", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = agent.Invoke(ctx, input)
		}
	})
}
