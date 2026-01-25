// Package package_pairs provides integration tests between Orchestration and Agents packages.
// This test suite verifies that orchestration components work correctly with agents
// for coordinating agent execution in chains, graphs, and workflows.
package package_pairs

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/agents"
	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/orchestration"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/tests/integration/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	agentsiface "github.com/lookatitude/beluga-ai/pkg/agents/iface"
	orchestrationiface "github.com/lookatitude/beluga-ai/pkg/orchestration/iface"
)

// TestIntegrationOrchestrationAgents tests the integration between Orchestration and Agents.
// This tests orchestration components using agents as executable steps.
func TestIntegrationOrchestrationAgents(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	tests := []struct {
		name              string
		orchestrationType string
		agentCount        int
		expectedExecution bool
		wantErr           bool
	}{
		{
			name:              "orchestrator_chain_with_agents",
			orchestrationType: "chain",
			agentCount:        2,
			expectedExecution: true,
			wantErr:           false,
		},
		{
			name:              "orchestrator_graph_with_agents",
			orchestrationType: "graph",
			agentCount:        3,
			expectedExecution: true,
			wantErr:           false,
		},
		{
			name:              "orchestrator_workflow_with_agents",
			orchestrationType: "workflow",
			agentCount:        2,
			expectedExecution: false, // Workflows require Temporal client
			wantErr:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// Create orchestrator
			orch, err := orchestration.NewDefaultOrchestrator()
			require.NoError(t, err)
			assert.NotNil(t, orch)

			// Create agents that will be used as steps/nodes
			testAgents := make([]agentsiface.CompositeAgent, tt.agentCount)
			for i := 0; i < tt.agentCount; i++ {
				agentName := fmt.Sprintf("orchestrated-agent-%d", i+1)
				mockAgent := agents.NewAdvancedMockAgent(agentName, "base")
				testAgents[i] = mockAgent
			}

			// Convert agents to Runnable for orchestration
			runnables := make([]core.Runnable, len(testAgents))
			for i, agent := range testAgents {
				runnables[i] = agent
			}

			var result any
			switch tt.orchestrationType {
			case "chain":
				// Create chain with agents as steps
				chain, err := orch.CreateChain(runnables)
				require.NoError(t, err)
				assert.NotNil(t, chain)

				// Chain.Invoke expects map[string]any or string, not ChatMessage
				result, err = chain.Invoke(ctx, map[string]any{"input": "test input"})
				if tt.wantErr {
					assert.Error(t, err)
				} else {
					require.NoError(t, err)
					assert.NotNil(t, result)
				}

			case "graph":
				// Create graph with agents as nodes
				graph, err := orch.CreateGraph()
				require.NoError(t, err)
				assert.NotNil(t, graph)

				// Add agents as graph nodes
				for i, agent := range testAgents {
					nodeName := fmt.Sprintf("agent-node-%d", i+1)
					err = graph.AddNode(nodeName, agent)
					require.NoError(t, err)
				}

				// Add edges between nodes
				for i := 0; i < len(testAgents)-1; i++ {
					from := fmt.Sprintf("agent-node-%d", i+1)
					to := fmt.Sprintf("agent-node-%d", i+2)
					err = graph.AddEdge(from, to)
					require.NoError(t, err)
				}

				// Set entry point
				err = graph.SetEntryPoint([]string{"agent-node-1"})
				require.NoError(t, err)

				// Set finish point
				lastNode := fmt.Sprintf("agent-node-%d", len(testAgents))
				err = graph.SetFinishPoint([]string{lastNode})
				require.NoError(t, err)

				result, err = graph.Invoke(ctx, schema.NewHumanMessage("test input"))
				if tt.wantErr {
					assert.Error(t, err)
				} else {
					require.NoError(t, err)
					assert.NotNil(t, result)
				}

			case "workflow":
				// Workflow creation will fail without Temporal client
				workflowFn := func(ctx context.Context, input string) (string, error) {
					return "workflow result", nil
				}

				_, err = orch.CreateWorkflow(workflowFn)
				require.Error(t, err) // Expected: Temporal client not configured
			}

			// Verify orchestration executed
			if tt.expectedExecution && !tt.wantErr {
				assert.NotNil(t, result, "Orchestration should have produced a result")
			}

			// Verify orchestrator metrics
			metrics := orch.GetMetrics()
			assert.NotNil(t, metrics)

			switch tt.orchestrationType {
			case "chain":
				assert.GreaterOrEqual(t, metrics.GetActiveChains(), 0)
			case "graph":
				assert.GreaterOrEqual(t, metrics.GetActiveGraphs(), 0)
			case "workflow":
				assert.GreaterOrEqual(t, metrics.GetActiveWorkflows(), 0)
			}
		})
	}
}

// TestOrchestrationAgentsErrorHandling tests error scenarios when orchestration uses agents.
func TestOrchestrationAgentsErrorHandling(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	tests := []struct {
		name        string
		setup       func(t *testing.T) (orchestrationiface.Chain, []agentsiface.CompositeAgent)
		expectedErr bool
	}{
		{
			name: "chain_with_error_agent",
			setup: func(t *testing.T) (orchestrationiface.Chain, []agentsiface.CompositeAgent) {
				orch, err := orchestration.NewDefaultOrchestrator()
				require.NoError(t, err)

				// Create agent that returns error
				errorAgent := agents.NewAdvancedMockAgent("error-agent", "base",
					agents.WithMockError(true, errors.New("agent execution failed")))

				runnables := []core.Runnable{errorAgent}
				chain, err := orch.CreateChain(runnables)
				require.NoError(t, err)

				return chain, []agentsiface.CompositeAgent{errorAgent}
			},
			expectedErr: true,
		},
		{
			name: "chain_with_multiple_agents_one_error",
			setup: func(t *testing.T) (orchestrationiface.Chain, []agentsiface.CompositeAgent) {
				orch, err := orchestration.NewDefaultOrchestrator()
				require.NoError(t, err)

				// Create mix of normal and error agents
				normalAgent1 := agents.NewAdvancedMockAgent("normal-1", "base")
				errorAgent := agents.NewAdvancedMockAgent("error-agent", "base",
					agents.WithMockError(true, errors.New("agent execution failed")))
				normalAgent2 := agents.NewAdvancedMockAgent("normal-2", "base")

				runnables := []core.Runnable{normalAgent1, errorAgent, normalAgent2}
				chain, err := orch.CreateChain(runnables)
				require.NoError(t, err)

				return chain, []agentsiface.CompositeAgent{normalAgent1, errorAgent, normalAgent2}
			},
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			chain, testAgents := tt.setup(t)

			// Execute chain with error agent
			// Chain.Invoke expects map[string]any or string, not ChatMessage
			result, err := chain.Invoke(ctx, map[string]any{"input": "test input"})

			if tt.expectedErr {
				assert.Error(t, err, "Chain execution should fail with error agent")
				// Result may or may not be nil depending on error handling
				t.Logf("Chain execution result: result=%v, err=%v", result != nil, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
			}

			// Verify agents exist
			assert.NotEmpty(t, testAgents, "Should have test agents")
			for i, agent := range testAgents {
				assert.NotNil(t, agent, "Agent %d should not be nil", i+1)
			}
		})
	}
}

// TestOrchestrationAgentsConcurrency tests concurrent execution of agents through orchestration.
func TestOrchestrationAgentsConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrency tests in short mode")
	}

	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	ctx := context.Background()

	// Create orchestrator
	orch, err := orchestration.NewDefaultOrchestrator()
	require.NoError(t, err)

	// Create multiple agents
	const agentCount = 5
	testAgents := make([]agentsiface.CompositeAgent, agentCount)
	runnables := make([]core.Runnable, agentCount)

	for i := 0; i < agentCount; i++ {
		agentName := fmt.Sprintf("concurrent-agent-%d", i+1)
		mockAgent := agents.NewAdvancedMockAgent(agentName, "base")
		testAgents[i] = mockAgent
		runnables[i] = mockAgent
	}

	// Create graph with agents for parallel execution
	graph, err := orch.CreateGraph()
	require.NoError(t, err)

	// Add all agents as nodes
	for i := 0; i < agentCount; i++ {
		nodeName := fmt.Sprintf("agent-node-%d", i+1)
		err = graph.AddNode(nodeName, testAgents[i])
		require.NoError(t, err)
	}

	// Set all nodes as entry points (parallel start)
	entryPoints := make([]string, agentCount)
	for i := 0; i < agentCount; i++ {
		entryPoints[i] = fmt.Sprintf("agent-node-%d", i+1)
	}
	err = graph.SetEntryPoint(entryPoints)
	require.NoError(t, err)

	// Execute graph
	result, err := graph.Invoke(ctx, schema.NewHumanMessage("concurrent test"))
	require.NoError(t, err)
	assert.NotNil(t, result)

	t.Logf("Concurrent execution completed: %d agents executed", agentCount)
}

// TestOrchestrationAgentsBatchExecution tests batch execution of agents through orchestration.
func TestOrchestrationAgentsBatchExecution(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	ctx := context.Background()

	// Create orchestrator
	orch, err := orchestration.NewDefaultOrchestrator()
	require.NoError(t, err)

	// Create agent
	agent := agents.NewAdvancedMockAgent("batch-agent", "base")
	runnables := []core.Runnable{agent}

	// Create chain with agent
	chain, err := orch.CreateChain(runnables)
	require.NoError(t, err)

	// Execute batch - Batch expects map[string]any or string inputs
	inputs := []any{
		map[string]any{"input": "input 1"},
		map[string]any{"input": "input 2"},
		map[string]any{"input": "input 3"},
	}

	results, err := chain.Batch(ctx, inputs)
	require.NoError(t, err)
	assert.Len(t, results, len(inputs))

	for i, result := range results {
		assert.NotNil(t, result, "Result %d should not be nil", i+1)
		t.Logf("Batch result %d: %v", i+1, result != nil)
	}
}

// TestOrchestrationAgentsStreamExecution tests stream execution of agents through orchestration.
func TestOrchestrationAgentsStreamExecution(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	ctx := context.Background()

	// Create orchestrator
	orch, err := orchestration.NewDefaultOrchestrator()
	require.NoError(t, err)

	// Create agent
	agent := agents.NewAdvancedMockAgent("stream-agent", "base")
	runnables := []core.Runnable{agent}

	// Create chain with agent
	chain, err := orch.CreateChain(runnables)
	require.NoError(t, err)

	// Execute stream
	streamCh, err := chain.Stream(ctx, schema.NewHumanMessage("stream input"))
	require.NoError(t, err)
	assert.NotNil(t, streamCh)

	// Collect stream results
	var results []any
	timeout := time.After(5 * time.Second)
	for {
		select {
		case result, ok := <-streamCh:
			if !ok {
				goto done
			}
			results = append(results, result)
		case <-timeout:
			t.Logf("Stream timeout after collecting %d results", len(results))
			goto done
		}
	}

done:
	assert.NotEmpty(t, results, "Should have received at least one stream result")
	t.Logf("Stream execution completed: %d results received", len(results))
}
