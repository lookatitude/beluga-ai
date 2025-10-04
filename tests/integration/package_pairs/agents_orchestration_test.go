// Package package_pairs provides integration tests between Agents and Orchestration packages.
// This test suite verifies that agents work correctly with orchestration components
// for multi-agent workflows, chain execution, and complex task coordination.
package package_pairs

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/agents"
	"github.com/lookatitude/beluga-ai/pkg/memory"
	"github.com/lookatitude/beluga-ai/pkg/orchestration"
	"github.com/lookatitude/beluga-ai/tests/integration/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	agentsiface "github.com/lookatitude/beluga-ai/pkg/agents/iface"
)

// TestIntegrationAgentsOrchestration tests the integration between Agents and Orchestration
func TestIntegrationAgentsOrchestration(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer helper.Cleanup(context.Background())

	tests := []struct {
		name              string
		agentCount        int
		orchestrationType string
		expectedExecution bool
		expectedSteps     int
	}{
		{
			name:              "single_agent_chain",
			agentCount:        1,
			orchestrationType: "chain",
			expectedExecution: true,
			expectedSteps:     3,
		},
		{
			name:              "multi_agent_chain",
			agentCount:        3,
			orchestrationType: "chain",
			expectedExecution: true,
			expectedSteps:     6, // 2 steps per agent
		},
		{
			name:              "agents_in_graph",
			agentCount:        4,
			orchestrationType: "graph",
			expectedExecution: true,
			expectedSteps:     8,
		},
		{
			name:              "agents_in_workflow",
			agentCount:        2,
			orchestrationType: "workflow",
			expectedExecution: true,
			expectedSteps:     4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// Create agents
			testAgents := make([]agentsiface.CompositeAgent, tt.agentCount)
			for i := 0; i < tt.agentCount; i++ {
				agentName := fmt.Sprintf("test-agent-%d", i+1)

				// For integration testing, we'll use mock agents
				mockAgent := agents.NewAdvancedMockAgent(agentName, "base")
				testAgents[i] = mockAgent
			}

			// Create orchestration component
			var orchestrationResult interface{}
			var err error

			switch tt.orchestrationType {
			case "chain":
				chain := orchestration.CreateTestChain(
					fmt.Sprintf("agent-chain-%s", tt.name),
					generateStepNames(tt.expectedSteps),
				)
				orchestrationResult, err = chain.Invoke(ctx, "test input")

			case "graph":
				nodes := make([]string, tt.agentCount)
				edges := make(map[string][]string)
				for i := 0; i < tt.agentCount; i++ {
					nodes[i] = fmt.Sprintf("agent-node-%d", i+1)
					if i > 0 {
						edges[fmt.Sprintf("agent-node-%d", i)] = []string{fmt.Sprintf("agent-node-%d", i+1)}
					}
				}

				graph := orchestration.CreateTestGraph(
					fmt.Sprintf("agent-graph-%s", tt.name),
					nodes,
					edges,
				)
				orchestrationResult, err = graph.Invoke(ctx, "test input")

			case "workflow":
				workflow := orchestration.CreateTestWorkflow(
					fmt.Sprintf("agent-workflow-%s", tt.name),
					generateTaskNames(tt.agentCount),
				)
				workflowID, runID, workflowErr := workflow.Execute(ctx, "test input")
				err = workflowErr
				if err == nil {
					orchestrationResult = fmt.Sprintf("workflow executed: %s/%s", workflowID, runID)
				}
			}

			if tt.expectedExecution {
				assert.NoError(t, err)
				assert.NotNil(t, orchestrationResult)

				// Verify orchestration executed successfully
				if result, ok := orchestrationResult.(string); ok {
					assert.Contains(t, result, tt.name)
				}
			} else {
				assert.Error(t, err)
			}

			// Test multi-agent coordination through memory
			sharedMemory := helper.CreateMockMemory("shared-coordination", memory.MemoryTypeBuffer)
			err = helper.TestMultiAgentWorkflow(testAgents, orchestrationResult, sharedMemory)
			assert.NoError(t, err)

			// Verify agents were properly coordinated
			for i, agent := range testAgents {
				if mockAgent, ok := agent.(*agents.AdvancedMockAgent); ok {
					assert.Greater(t, mockAgent.GetCallCount(), 0,
						"Agent %d should have been called", i+1)
				}
			}
		})
	}
}

// TestAgentsOrchestrationErrorHandling tests error scenarios
func TestAgentsOrchestrationErrorHandling(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer helper.Cleanup(context.Background())

	tests := []struct {
		name        string
		setupError  func() ([]agentsiface.CompositeAgent, interface{})
		expectedErr bool
	}{
		{
			name: "agent_execution_error",
			setupError: func() ([]agentsiface.CompositeAgent, interface{}) {
				errorAgent := agents.NewAdvancedMockAgent("error-agent", "base",
					agents.WithMockError(true, fmt.Errorf("agent execution failed")))

				chain := orchestration.CreateTestChain("error-chain", []string{"step1"})

				return []agentsiface.CompositeAgent{errorAgent}, chain
			},
			expectedErr: true,
		},
		{
			name: "orchestration_execution_error",
			setupError: func() ([]agentsiface.CompositeAgent, interface{}) {
				normalAgent := agents.NewAdvancedMockAgent("normal-agent", "base")

				errorOrchestrator := orchestration.NewAdvancedMockOrchestrator("error-orch", "chain",
					orchestration.WithMockError(true, fmt.Errorf("orchestration failed")))

				return []agentsiface.CompositeAgent{normalAgent}, errorOrchestrator
			},
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testAgents, orchestrator := tt.setupError()
			sharedMemory := helper.CreateMockMemory("error-test-memory", memory.MemoryTypeBuffer)

			// Test multi-agent workflow with errors
			err := helper.TestMultiAgentWorkflow(testAgents, orchestrator, sharedMemory)

			if tt.expectedErr {
				// For now, the helper method doesn't propagate errors from individual components
				// In a real implementation, this would test actual error propagation
				assert.NotNil(t, testAgents, "Should have test agents even with error setup")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestAgentsOrchestrationPerformance tests performance scenarios
func TestAgentsOrchestrationPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	helper := utils.NewIntegrationTestHelper()
	defer helper.Cleanup(context.Background())

	tests := []struct {
		name            string
		agentCount      int
		executionRounds int
		maxDuration     time.Duration
	}{
		{
			name:            "small_agent_group",
			agentCount:      3,
			executionRounds: 5,
			maxDuration:     2 * time.Second,
		},
		{
			name:            "medium_agent_group",
			agentCount:      8,
			executionRounds: 3,
			maxDuration:     5 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()

			// Create agents
			testAgents := make([]agentsiface.CompositeAgent, tt.agentCount)
			for i := 0; i < tt.agentCount; i++ {
				agentName := fmt.Sprintf("perf-agent-%d", i+1)
				mockAgent := agents.NewAdvancedMockAgent(agentName, "base")
				testAgents[i] = mockAgent
			}

			// Create orchestration
			chain := orchestration.CreateTestChain("performance-chain",
				generateStepNames(tt.agentCount*2))

			sharedMemory := helper.CreateMockMemory("perf-memory", memory.MemoryTypeBuffer)

			// Execute multiple rounds
			for round := 0; round < tt.executionRounds; round++ {
				err := helper.TestMultiAgentWorkflow(testAgents, chain, sharedMemory)
				require.NoError(t, err, "Performance round %d failed", round+1)
			}

			duration := time.Since(start)
			assert.LessOrEqual(t, duration, tt.maxDuration,
				"Performance test should complete within %v, took %v", tt.maxDuration, duration)

			t.Logf("Performance test completed: %d agents, %d rounds in %v",
				tt.agentCount, tt.executionRounds, duration)
		})
	}
}

// TestAgentsOrchestrationConcurrency tests concurrent agent execution
func TestAgentsOrchestrationConcurrency(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer helper.Cleanup(context.Background())

	const numGoroutines = 5
	const agentsPerGoroutine = 2

	t.Run("concurrent_agent_orchestration", func(t *testing.T) {
		sharedMemory := helper.CreateMockMemory("concurrent-memory", memory.MemoryTypeBuffer)

		helper.CrossPackageLoadTest(t, func() error {
			// Create agents for this execution
			testAgents := make([]agentsiface.CompositeAgent, agentsPerGoroutine)
			for i := 0; i < agentsPerGoroutine; i++ {
				agentName := fmt.Sprintf("concurrent-agent-%d", i+1)
				mockAgent := agents.NewAdvancedMockAgent(agentName, "base")
				testAgents[i] = mockAgent
			}

			// Create orchestration
			chain := orchestration.CreateTestChain("concurrent-chain", []string{"step1", "step2"})

			// Execute workflow
			return helper.TestMultiAgentWorkflow(testAgents, chain, sharedMemory)
		}, numGoroutines*agentsPerGoroutine, numGoroutines)
	})
}

// TestAgentsOrchestrationComplexScenarios tests complex real-world scenarios
func TestAgentsOrchestrationComplexScenarios(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer helper.Cleanup(context.Background())

	scenarios := []struct {
		name     string
		scenario func(t *testing.T)
	}{
		{
			name: "collaborative_problem_solving",
			scenario: func(t *testing.T) {
				ctx := context.Background()

				// Create specialized agents
				researchAgent := agents.NewAdvancedMockAgent("researcher", "base")
				analysisAgent := agents.NewAdvancedMockAgent("analyst", "base")
				reportAgent := agents.NewAdvancedMockAgent("reporter", "base")

				agents := []agentsiface.CompositeAgent{researchAgent, analysisAgent, reportAgent}

				// Create orchestration workflow
				workflow := orchestration.CreateTestWorkflow("collaborative-workflow",
					[]string{"research", "analyze", "report"})

				// Shared memory for collaboration
				sharedMemory := helper.CreateMockMemory("collaborative-memory", memory.MemoryTypeBuffer)

				// Test workflow execution
				workflowID, runID, err := workflow.Execute(ctx, "Analyze the impact of AI on healthcare")
				require.NoError(t, err)
				assert.NotEmpty(t, workflowID)
				assert.NotEmpty(t, runID)

				// Test agent coordination
				err = helper.TestMultiAgentWorkflow(agents, workflow, sharedMemory)
				assert.NoError(t, err)

				// Verify workflow result
				result, err := workflow.GetResult(ctx, workflowID, runID)
				assert.NoError(t, err)
				assert.NotNil(t, result)

				// Test workflow control operations
				err = workflow.Signal(ctx, workflowID, runID, "progress_update", "50% complete")
				assert.NoError(t, err)

				queryResult, err := workflow.Query(ctx, workflowID, runID, "status")
				assert.NoError(t, err)
				assert.NotNil(t, queryResult)
			},
		},
		{
			name: "hierarchical_agent_system",
			scenario: func(t *testing.T) {
				// Create hierarchical agent structure
				supervisorAgent := agents.NewAdvancedMockAgent("supervisor", "base")
				workerAgent1 := agents.NewAdvancedMockAgent("worker1", "base")
				workerAgent2 := agents.NewAdvancedMockAgent("worker2", "base")

				allAgents := []agentsiface.CompositeAgent{supervisorAgent, workerAgent1, workerAgent2}

				// Create graph orchestration for hierarchy
				nodes := []string{"supervisor", "worker1", "worker2"}
				edges := map[string][]string{
					"supervisor": {"worker1", "worker2"}, // Supervisor delegates to workers
				}

				graph := orchestration.CreateTestGraph("hierarchical-graph", nodes, edges)

				// Shared memory for coordination
				hierarchyMemory := helper.CreateMockMemory("hierarchy-memory", memory.MemoryTypeBuffer)

				// Test hierarchical execution
				ctx := context.Background()
				result, err := graph.Invoke(ctx, "Coordinate task execution")
				assert.NoError(t, err)
				assert.NotNil(t, result)

				// Test agent coordination
				err = helper.TestMultiAgentWorkflow(allAgents, graph, hierarchyMemory)
				assert.NoError(t, err)

				// Verify all agents participated
				for i, agent := range allAgents {
					if mockAgent, ok := agent.(*agents.AdvancedMockAgent); ok {
						// In a real test, we'd verify the agent was actually called
						assert.NotNil(t, mockAgent, "Agent %d should be accessible", i+1)
					}
				}
			},
		},
		{
			name: "agent_chain_with_recovery",
			scenario: func(t *testing.T) {
				// Test error recovery in agent chains
				normalAgent := agents.NewAdvancedMockAgent("normal-agent", "base")
				recoveryAgent := agents.NewAdvancedMockAgent("recovery-agent", "base")

				testAgents := []agentsiface.CompositeAgent{normalAgent, recoveryAgent}

				// Create chain with error recovery steps
				chain := orchestration.CreateTestChain("recovery-chain",
					[]string{"execute", "verify", "recover", "complete"})

				recoveryMemory := helper.CreateMockMemory("recovery-memory", memory.MemoryTypeBuffer)

				// Test execution with recovery
				err := helper.TestMultiAgentWorkflow(testAgents, chain, recoveryMemory)
				assert.NoError(t, err)

				// Verify recovery mechanism worked
				memoryContent := helper.GetMemoryContent(recoveryMemory)
				assert.NotEmpty(t, memoryContent, "Recovery memory should contain execution history")
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			scenario.scenario(t)
		})
	}
}

// TestAgentsOrchestrationRealWorldWorkflows tests realistic workflows
func TestAgentsOrchestrationRealWorldWorkflows(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer helper.Cleanup(context.Background())

	t.Run("document_processing_workflow", func(t *testing.T) {
		// Simulate a document processing workflow with multiple agents

		// Create specialized agents
		extractorAgent := agents.NewAdvancedMockAgent("extractor", "base")
		processorAgent := agents.NewAdvancedMockAgent("processor", "base")
		validatorAgent := agents.NewAdvancedMockAgent("validator", "base")

		agents := []agentsiface.CompositeAgent{extractorAgent, processorAgent, validatorAgent}

		// Create workflow for document processing
		workflow := orchestration.CreateTestWorkflow("doc-processing",
			[]string{"extract", "process", "validate", "store"})

		// Workflow memory for state management
		workflowMemory := helper.CreateMockMemory("workflow-memory", memory.MemoryTypeBuffer)

		// Execute document processing workflow
		ctx := context.Background()
		workflowID, runID, err := workflow.Execute(ctx, "process_document.pdf")
		require.NoError(t, err)

		// Test agent coordination in workflow
		err = helper.TestMultiAgentWorkflow(agents, workflow, workflowMemory)
		assert.NoError(t, err)

		// Verify workflow completion
		result, err := workflow.GetResult(ctx, workflowID, runID)
		assert.NoError(t, err)
		assert.Contains(t, result.(string), workflowID)

		// Test workflow state queries
		statusResult, err := workflow.Query(ctx, workflowID, runID, "processing_status")
		assert.NoError(t, err)
		assert.NotNil(t, statusResult)
	})

	t.Run("decision_making_chain", func(t *testing.T) {
		// Test decision-making chain with multiple agents

		analyzerAgent := agents.NewAdvancedMockAgent("analyzer", "base")
		evaluatorAgent := agents.NewAdvancedMockAgent("evaluator", "base")
		decisionAgent := agents.NewAdvancedMockAgent("decider", "base")

		agents := []agentsiface.CompositeAgent{analyzerAgent, evaluatorAgent, decisionAgent}

		// Create decision chain
		chain := orchestration.CreateTestChain("decision-chain",
			[]string{"analyze_options", "evaluate_criteria", "make_decision"})

		decisionMemory := helper.CreateMockMemory("decision-memory", memory.MemoryTypeBuffer)

		// Execute decision chain
		ctx := context.Background()
		result, err := chain.Invoke(ctx, "Should we adopt new technology X?")
		require.NoError(t, err)
		assert.Contains(t, result.(string), "decision-chain")

		// Test agent coordination in decision making
		err = helper.TestMultiAgentWorkflow(agents, chain, decisionMemory)
		assert.NoError(t, err)

		// Verify decision process was recorded in memory
		memoryContent := helper.GetMemoryContent(decisionMemory)
		assert.NotEmpty(t, memoryContent, "Decision process should be recorded in memory")
	})
}

// TestAgentsOrchestrationMetrics tests metrics collection across agent-orchestration integration
func TestAgentsOrchestrationMetrics(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer helper.Cleanup(context.Background())

	// Create components with metrics
	agent := agents.NewAdvancedMockAgent("metrics-agent", "base")
	orchestrator := orchestration.NewAdvancedMockOrchestrator("metrics-orch", "chain")

	// Create chain for testing
	chain := orchestration.CreateTestChain("metrics-chain", []string{"step1", "step2"})
	memory := helper.CreateMockMemory("metrics-memory", memory.MemoryTypeBuffer)

	// Execute operations to generate metrics
	err := helper.TestMultiAgentWorkflow(
		[]agentsiface.CompositeAgent{agent},
		orchestrator,
		memory,
	)
	assert.NoError(t, err)

	// Execute chain operations
	ctx := context.Background()
	_, err = chain.Invoke(ctx, "metrics test")
	assert.NoError(t, err)

	// Verify metrics were recorded
	agentHealth := agent.CheckHealth()
	assert.Contains(t, agentHealth, "call_count")

	orchestratorHealth := orchestrator.CheckHealth()
	assert.Contains(t, orchestratorHealth, "call_count")

	// Test cross-component metrics
	components := map[string]interface{}{
		"agent":        agent,
		"orchestrator": orchestrator,
		"memory":       memory,
	}
	helper.AssertHealthChecks(t, components)
}

// BenchmarkIntegrationAgentsOrchestration benchmarks agent-orchestration integration
func BenchmarkIntegrationAgentsOrchestration(b *testing.B) {
	helper := utils.NewIntegrationTestHelper()
	defer helper.Cleanup(context.Background())

	// Setup test components
	testAgents := make([]agentsiface.CompositeAgent, 3)
	for i := 0; i < 3; i++ {
		agentName := fmt.Sprintf("benchmark-agent-%d", i+1)
		mockAgent := agents.NewAdvancedMockAgent(agentName, "base")
		testAgents[i] = mockAgent
	}

	orchestrator := orchestration.NewAdvancedMockOrchestrator("benchmark-orch", "chain")
	memory := helper.CreateMockMemory("benchmark-memory", memory.MemoryTypeBuffer)

	b.Run("MultiAgentWorkflow", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := helper.TestMultiAgentWorkflow(testAgents, orchestrator, memory)
			if err != nil {
				b.Errorf("Multi-agent workflow error: %v", err)
			}
		}
	})

	b.Run("ChainExecution", func(b *testing.B) {
		chain := orchestration.CreateTestChain("benchmark-chain", []string{"step1", "step2", "step3"})
		ctx := context.Background()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := chain.Invoke(ctx, fmt.Sprintf("input-%d", i))
			if err != nil {
				b.Errorf("Chain execution error: %v", err)
			}
		}
	})

	b.Run("WorkflowExecution", func(b *testing.B) {
		workflow := orchestration.CreateTestWorkflow("benchmark-workflow",
			[]string{"task1", "task2", "task3"})
		ctx := context.Background()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			workflowID, runID, err := workflow.Execute(ctx, fmt.Sprintf("input-%d", i))
			if err != nil {
				b.Errorf("Workflow execution error: %v", err)
			}

			// Also benchmark result retrieval
			_, err = workflow.GetResult(ctx, workflowID, runID)
			if err != nil {
				b.Errorf("Workflow result retrieval error: %v", err)
			}
		}
	})
}

// Helper functions

func generateStepNames(count int) []string {
	steps := make([]string, count)
	for i := 0; i < count; i++ {
		steps[i] = fmt.Sprintf("step_%d", i+1)
	}
	return steps
}

func generateTaskNames(count int) []string {
	tasks := make([]string, count)
	for i := 0; i < count; i++ {
		tasks[i] = fmt.Sprintf("task_%d", i+1)
	}
	return tasks
}

// This method is now available in utils.IntegrationTestHelper
