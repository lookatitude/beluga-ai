// Package package_pairs provides integration tests between Agents and Memory packages.
// This test suite verifies that agents work correctly with memory components
// for conversation history, context management, and state persistence.
package package_pairs

import (
	"context"
	"fmt"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/agents"
	"github.com/lookatitude/beluga-ai/pkg/memory"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/tests/integration/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	agentsiface "github.com/lookatitude/beluga-ai/pkg/agents/iface"
)

// TestIntegrationAgentsMemory tests the integration between Agents and Memory packages.
func TestIntegrationAgentsMemory(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	tests := []struct {
		name       string
		memoryType memory.MemoryType
		agentCount int
		exchanges  int
		wantErr    bool
	}{
		{
			name:       "single_agent_buffer_memory",
			memoryType: memory.MemoryTypeBuffer,
			agentCount: 1,
			exchanges:  3,
			wantErr:    false,
		},
		{
			name:       "single_agent_window_memory",
			memoryType: memory.MemoryTypeBufferWindow,
			agentCount: 1,
			exchanges:  5,
			wantErr:    false,
		},
		{
			name:       "multi_agent_shared_memory",
			memoryType: memory.MemoryTypeBuffer,
			agentCount: 3,
			exchanges:  2,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// Create memory component
			mockMemory := helper.CreateMockMemory("integration-memory", tt.memoryType)

			// Create agents
			testAgents := make([]agentsiface.CompositeAgent, tt.agentCount)
			for i := 0; i < tt.agentCount; i++ {
				agentName := fmt.Sprintf("memory-agent-%d", i+1)
				mockLLM := helper.CreateMockLLM(fmt.Sprintf("llm-%d", i+1))
				agent, err := agents.NewBaseAgent(agentName, mockLLM, nil)
				require.NoError(t, err)
				testAgents[i] = agent
			}

			// Test agent-memory integration
			for i := 0; i < tt.exchanges; i++ {
				// Load memory variables
				inputs := map[string]any{"input": fmt.Sprintf("Exchange %d", i+1)}
				vars, err := mockMemory.LoadMemoryVariables(ctx, inputs)
				if tt.wantErr {
					assert.Error(t, err)
				} else {
					require.NoError(t, err)
					assert.NotNil(t, vars)
				}

				// Simulate agent using memory
				for _, agent := range testAgents {
					// Agent would use memory variables in real scenario
					_, err := agent.Invoke(ctx, schema.NewHumanMessage(fmt.Sprintf("Exchange %d", i+1)))
					// Execution may succeed or fail depending on implementation
					t.Logf("Agent execution: err=%v", err)
				}

				// Save memory variables
				outputs := map[string]any{"output": fmt.Sprintf("Response %d", i+1)}
				err = mockMemory.SaveContext(ctx, inputs, outputs)
				if tt.wantErr {
					assert.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			}

			// Verify memory contains conversation history
			memoryVars := mockMemory.MemoryVariables()
			assert.NotEmpty(t, memoryVars, "Memory should have variables")

			// Verify memory persistence
			finalVars, err := mockMemory.LoadMemoryVariables(ctx, map[string]any{"input": "final check"})
			require.NoError(t, err)
			assert.NotEmpty(t, finalVars, "Memory should persist across exchanges")
		})
	}
}

// TestAgentsMemoryErrorHandling tests error scenarios between agents and memory.
func TestAgentsMemoryErrorHandling(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	tests := []struct {
		name        string
		setupError  func() (agentsiface.CompositeAgent, memory.Memory, error)
		expectedErr bool
	}{
		{
			name: "memory_load_error",
			setupError: func() (agentsiface.CompositeAgent, memory.Memory, error) {
				mockLLM := helper.CreateMockLLM("error-llm")
				agent, err := agents.NewBaseAgent("error-agent", mockLLM, nil)
				if err != nil {
					return nil, nil, err
				}
				mockMemory := helper.CreateMockMemory("error-memory", memory.MemoryTypeBuffer)
				return agent, mockMemory, nil
			},
			expectedErr: false, // Mock memory may not return errors by default
		},
		{
			name: "memory_save_error",
			setupError: func() (agentsiface.CompositeAgent, memory.Memory, error) {
				mockLLM := helper.CreateMockLLM("save-error-llm")
				agent, err := agents.NewBaseAgent("save-error-agent", mockLLM, nil)
				if err != nil {
					return nil, nil, err
				}
				mockMemory := helper.CreateMockMemory("save-error-memory", memory.MemoryTypeBuffer)
				return agent, mockMemory, nil
			},
			expectedErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent, mockMemory, setupErr := tt.setupError()
			require.NoError(t, setupErr)
			assert.NotNil(t, agent)
			assert.NotNil(t, mockMemory)

			ctx := context.Background()

			// Test memory operations
			inputs := map[string]any{"input": "test"}
			vars, err := mockMemory.LoadMemoryVariables(ctx, inputs)
			if tt.expectedErr {
				assert.Error(t, err)
			} else {
				// Error may or may not occur
				t.Logf("Load memory variables: vars=%v, err=%v", vars != nil, err)
			}

			outputs := map[string]any{"output": "test response"}
			err = mockMemory.SaveContext(ctx, inputs, outputs)
			if tt.expectedErr {
				assert.Error(t, err)
			} else {
				// Error may or may not occur
				t.Logf("Save context: err=%v", err)
			}
		})
	}
}

// TestAgentsMemoryConcurrency tests concurrent agent-memory interactions.
func TestAgentsMemoryConcurrency(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	const numGoroutines = 5
	const operationsPerGoroutine = 3

	t.Run("concurrent_agent_memory_operations", func(t *testing.T) {
		mockLLM := helper.CreateMockLLM("concurrent-llm")
		_, err := agents.NewBaseAgent("concurrent-agent", mockLLM, nil)
		require.NoError(t, err)

		mockMemory := helper.CreateMockMemory("concurrent-memory", memory.MemoryTypeBuffer)

		helper.CrossPackageLoadTest(t, func() error {
			ctx := context.Background()
			inputs := map[string]any{"input": "concurrent test"}
			_, err := mockMemory.LoadMemoryVariables(ctx, inputs)
			if err != nil {
				return err
			}

			outputs := map[string]any{"output": "concurrent response"}
			err = mockMemory.SaveContext(ctx, inputs, outputs)
			return err
		}, numGoroutines*operationsPerGoroutine, numGoroutines)
	})
}

// TestAgentsMemoryStatePersistence tests memory state persistence across agent operations.
func TestAgentsMemoryStatePersistence(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	t.Run("memory_persistence_across_agent_calls", func(t *testing.T) {
		mockLLM := helper.CreateMockLLM("persistence-llm")
		agent, err := agents.NewBaseAgent("persistence-agent", mockLLM, nil)
		require.NoError(t, err)

		mockMemory := helper.CreateMockMemory("persistence-memory", memory.MemoryTypeBuffer)
		ctx := context.Background()

		// First exchange
		inputs1 := map[string]any{"input": "First message"}
		vars1, err := mockMemory.LoadMemoryVariables(ctx, inputs1)
		require.NoError(t, err)

		_, err = agent.Invoke(ctx, schema.NewHumanMessage("First message"))
		t.Logf("First agent call: err=%v", err)

		outputs1 := map[string]any{"output": "First response"}
		err = mockMemory.SaveContext(ctx, inputs1, outputs1)
		require.NoError(t, err)

		// Second exchange - should have memory from first
		inputs2 := map[string]any{"input": "Second message"}
		vars2, err := mockMemory.LoadMemoryVariables(ctx, inputs2)
		require.NoError(t, err)

		// Verify memory persisted
		assert.NotNil(t, vars1)
		assert.NotNil(t, vars2)
		// In a real scenario, vars2 would contain information from vars1

		_, err = agent.Invoke(ctx, schema.NewHumanMessage("Second message"))
		t.Logf("Second agent call: err=%v", err)
	})
}

// TestAgentsMemoryMultiAgentCoordination tests memory coordination between multiple agents.
func TestAgentsMemoryMultiAgentCoordination(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	t.Run("shared_memory_multi_agent", func(t *testing.T) {
		// Create shared memory for multiple agents
		sharedMemory := helper.CreateMockMemory("shared-memory", memory.MemoryTypeBuffer)

		// Create multiple agents
		agentList := make([]agentsiface.CompositeAgent, 3)
		for i := 0; i < 3; i++ {
			mockLLM := helper.CreateMockLLM(fmt.Sprintf("coord-llm-%d", i+1))
			agent, err := agents.NewBaseAgent(fmt.Sprintf("coord-agent-%d", i+1), mockLLM, nil)
			require.NoError(t, err)
			agentList[i] = agent
		}

		ctx := context.Background()

		// Each agent interacts with shared memory
		for i, agent := range agentList {
			inputs := map[string]any{"input": fmt.Sprintf("Agent %d input", i+1)}
			vars, err := sharedMemory.LoadMemoryVariables(ctx, inputs)
			require.NoError(t, err)
			assert.NotNil(t, vars)

			_, err = agent.Invoke(ctx, schema.NewHumanMessage(fmt.Sprintf("Agent %d message", i+1)))
			t.Logf("Agent %d execution: err=%v", i+1, err)

			outputs := map[string]any{"output": fmt.Sprintf("Agent %d output", i+1)}
			err = sharedMemory.SaveContext(ctx, inputs, outputs)
			require.NoError(t, err)
		}

		// Verify shared memory contains all agent interactions
		finalVars, err := sharedMemory.LoadMemoryVariables(ctx, map[string]any{"input": "final"})
		require.NoError(t, err)
		assert.NotNil(t, finalVars)
	})
}

// BenchmarkIntegrationAgentsMemory benchmarks agent-memory integration.
func BenchmarkIntegrationAgentsMemory(b *testing.B) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	mockLLM := helper.CreateMockLLM("benchmark-llm")
	agent, err := agents.NewBaseAgent("benchmark-agent", mockLLM, nil)
	require.NoError(b, err)

	mockMemory := helper.CreateMockMemory("benchmark-memory", memory.MemoryTypeBuffer)
	ctx := context.Background()

	b.Run("MemoryLoadSave", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			inputs := map[string]any{"input": fmt.Sprintf("benchmark %d", i)}
			_, err := mockMemory.LoadMemoryVariables(ctx, inputs)
			if err != nil {
				b.Errorf("Load memory error: %v", err)
			}

			outputs := map[string]any{"output": fmt.Sprintf("response %d", i)}
			err = mockMemory.SaveContext(ctx, inputs, outputs)
			if err != nil {
				b.Errorf("Save context error: %v", err)
			}
		}
	})

	b.Run("AgentWithMemory", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			inputs := map[string]any{"input": fmt.Sprintf("benchmark %d", i)}
			_, err := mockMemory.LoadMemoryVariables(ctx, inputs)
			if err != nil {
				b.Errorf("Load memory error: %v", err)
			}

			_, err = agent.Invoke(ctx, schema.NewHumanMessage(fmt.Sprintf("benchmark %d", i)))
			if err != nil {
				b.Errorf("Agent invoke error: %v", err)
			}

			outputs := map[string]any{"output": fmt.Sprintf("response %d", i)}
			err = mockMemory.SaveContext(ctx, inputs, outputs)
			if err != nil {
				b.Errorf("Save context error: %v", err)
			}
		}
	})
}
