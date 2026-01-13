package agents_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/agents"
	"github.com/lookatitude/beluga-ai/pkg/agents/iface"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	"github.com/lookatitude/beluga-ai/pkg/core"
	llmsiface "github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"go.opentelemetry.io/otel/trace"
)

// Static error variables for testing (err113 compliance)
var (
	errTestOriginal = errors.New("original error")
	errTestTimeout  = errors.New("timeout")
	errTestInvalid  = errors.New("invalid")
	errTestErr      = errors.New("err")
	errTestHandler  = errors.New("handler error")
)

// mockLLM is a simple mock implementation of the LLM interface for testing.
type mockLLM struct{}

func (m *mockLLM) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	prompt, _ := input.(string) // Assume string for mock
	return "Mock LLM response to: " + prompt, nil
}

func (m *mockLLM) GetModelName() string {
	return "mock-llm"
}

func (m *mockLLM) GetProviderName() string {
	return "mock-provider"
}

// mockChatModel is a simple mock implementation of the ChatModel interface for testing.
type mockChatModel struct{}

func (m *mockChatModel) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	return schema.NewAIMessage("Mock chat model response"), nil
}

func (m *mockChatModel) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
	results := make([]any, len(inputs))
	for i := range inputs {
		results[i] = schema.NewAIMessage("Mock batch chat response")
	}
	return results, nil
}

func (m *mockChatModel) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
	ch := make(chan any, 1)
	ch <- schema.NewAIMessage("Mock streaming chat response")
	close(ch)
	return ch, nil
}

func (m *mockChatModel) Generate(ctx context.Context, messages []schema.Message, options ...core.Option) (schema.Message, error) {
	return schema.NewAIMessage("Mock chat model response"), nil
}

func (m *mockChatModel) StreamChat(ctx context.Context, messages []schema.Message, options ...core.Option) (<-chan llmsiface.AIMessageChunk, error) {
	ch := make(chan llmsiface.AIMessageChunk, 1)
	ch <- llmsiface.AIMessageChunk{
		Content: "Mock streaming chat response",
	}
	close(ch)
	return ch, nil
}

func (m *mockChatModel) BindTools(toolsToBind []tools.Tool) llmsiface.ChatModel {
	return m
}

func (m *mockChatModel) GetModelName() string {
	return "mock-chat-model"
}

func (m *mockChatModel) GetProviderName() string {
	return "mock-provider"
}

func (m *mockChatModel) CheckHealth() map[string]any {
	return map[string]any{
		"status": "healthy",
		"name":   "mock-chat-model",
	}
}

// mockTool is a simple mock implementation of the Tool interface for testing.
type mockTool struct {
	name        string
	description string
}

func (m *mockTool) Name() string {
	return m.name
}

func (m *mockTool) Description() string {
	return m.description
}

func (m *mockTool) Definition() tools.ToolDefinition {
	return tools.ToolDefinition{
		Name:        m.name,
		Description: m.description,
		InputSchema: map[string]any{"type": "string"},
	}
}

func (m *mockTool) Execute(ctx context.Context, input any) (any, error) {
	return "Mock tool result for input: " + input.(string), nil
}

func (m *mockTool) Batch(ctx context.Context, inputs []any) ([]any, error) {
	results := make([]any, len(inputs))
	for i, input := range inputs {
		results[i] = "Mock batch tool result for input: " + input.(string)
	}
	return results, nil
}

func createMockTools() []tools.Tool {
	return []tools.Tool{
		&mockTool{name: "calculator", description: "A calculator tool"},
		&mockTool{name: "webSearch", description: "A web search tool"},
	}
}

// mockErrorLLM simulates LLM errors for testing error scenarios.
type mockErrorLLM struct {
	err error
}

func (m *mockErrorLLM) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	return nil, m.err
}

func (m *mockErrorLLM) GetModelName() string {
	return "mock-error-llm"
}

func (m *mockErrorLLM) GetProviderName() string {
	return "mock-error-provider"
}

// mockErrorTool simulates tool errors for testing error scenarios.
type mockErrorTool struct {
	err         error
	name        string
	description string
}

func (m *mockErrorTool) Name() string {
	return m.name
}

func (m *mockErrorTool) Description() string {
	return m.description
}

func (m *mockErrorTool) Definition() tools.ToolDefinition {
	return tools.ToolDefinition{
		Name:        m.name,
		Description: m.description,
		InputSchema: map[string]any{"type": "string"},
	}
}

func (m *mockErrorTool) Execute(ctx context.Context, input any) (any, error) {
	return nil, m.err
}

func (m *mockErrorTool) Batch(ctx context.Context, inputs []any) ([]any, error) {
	return nil, m.err
}

// mockMetricsRecorder implements the MetricsRecorder interface for testing.
type mockMetricsRecorder struct {
	agentExecutions int
	planningCalls   int
	toolCalls       int
	executorRuns    int
	mu              sync.Mutex
}

func (m *mockMetricsRecorder) StartAgentSpan(ctx context.Context, agentName, operation string) (context.Context, iface.SpanEnder) {
	return ctx, &mockSpan{}
}

func (m *mockMetricsRecorder) RecordAgentExecution(ctx context.Context, agentName, agentType string, duration time.Duration, success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.agentExecutions++
}

func (m *mockMetricsRecorder) RecordPlanningCall(ctx context.Context, agentName string, duration time.Duration, success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.planningCalls++
}

func (m *mockMetricsRecorder) RecordExecutorRun(ctx context.Context, executorType string, duration time.Duration, steps int, success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.executorRuns++
}

func (m *mockMetricsRecorder) RecordToolCall(ctx context.Context, toolName string, duration time.Duration, success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.toolCalls++
}

type mockSpan struct{}

func (m *mockSpan) End(options ...trace.SpanEndOption) {}

// createMockMetricsRecorder creates a new mock metrics recorder.
func createMockMetricsRecorder() *mockMetricsRecorder {
	return &mockMetricsRecorder{}
}

func TestNewBaseAgent(t *testing.T) {
	tests := []struct {
		name      string
		agentName string
		llm       llmsiface.LLM
		tools     []tools.Tool
		opts      []iface.Option
		wantErr   bool
	}{
		{
			name:      "valid base agent creation",
			agentName: "test-agent",
			llm:       &mockLLM{},
			tools:     createMockTools(),
			opts:      []iface.Option{},
			wantErr:   false,
		},
		{
			name:      "agent with options",
			agentName: "test-agent-with-opts",
			llm:       &mockLLM{},
			tools:     createMockTools(),
			opts: []iface.Option{
				agents.WithMaxRetries(5),
				agents.WithTimeout(10 * time.Second),
			},
			wantErr: false,
		},
		{
			name:      "empty agent name",
			agentName: "",
			llm:       &mockLLM{},
			tools:     createMockTools(),
			opts:      []iface.Option{},
			wantErr:   false, // Base agent allows empty names
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent, err := agents.NewBaseAgent(tt.agentName, tt.llm, tt.tools, tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewBaseAgent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && agent == nil {
				t.Error("NewBaseAgent() returned nil agent without error")
			}
		})
	}
}

func TestNewReActAgent(t *testing.T) {
	tests := []struct {
		name      string
		agentName string
		llm       llmsiface.ChatModel
		tools     []tools.Tool
		prompt    any
		opts      []iface.Option
		wantErr   bool
	}{
		{
			name:      "valid react agent creation",
			agentName: "react-agent",
			llm:       &mockChatModel{},
			tools:     createMockTools(),
			prompt:    "Test prompt",
			opts:      []iface.Option{},
			wantErr:   false,
		},
		{
			name:      "react agent with options",
			agentName: "react-agent-with-opts",
			llm:       &mockChatModel{},
			tools:     createMockTools(),
			prompt:    "Test prompt with options",
			opts: []iface.Option{
				agents.WithMaxRetries(3),
				agents.WithTimeout(5 * time.Second),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent, err := agents.NewReActAgent(tt.agentName, tt.llm, tt.tools, tt.prompt, tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewReActAgent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && agent == nil {
				t.Error("NewReActAgent() returned nil agent without error")
			}
		})
	}
}

func TestAgentFactory(t *testing.T) {
	config := agents.DefaultConfig()
	factory := agents.NewAgentFactory(config)

	llm := &mockLLM{}
	ctx := context.Background()
	tools := createMockTools()

	t.Run("CreateBaseAgent", func(t *testing.T) {
		agent, err := factory.CreateBaseAgent(ctx, "factory-base-agent", llm, tools)
		if err != nil {
			t.Errorf("AgentFactory.CreateBaseAgent() error = %v", err)
			return
		}
		if agent == nil {
			t.Error("AgentFactory.CreateBaseAgent() returned nil agent")
		}
	})

	t.Run("CreateReActAgent", func(t *testing.T) {
		chatLLM := &mockChatModel{}
		agent, err := factory.CreateReActAgent(ctx, "factory-react-agent", chatLLM, tools, "Test prompt")
		if err != nil {
			t.Errorf("AgentFactory.CreateReActAgent() error = %v", err)
			return
		}
		if agent == nil {
			t.Error("AgentFactory.CreateReActAgent() returned nil agent")
		}
	})
}

func TestNewAgentExecutor(t *testing.T) {
	executor := agents.NewAgentExecutor()
	if executor == nil {
		t.Error("NewAgentExecutor() returned nil")
	}
}

func TestNewAgentExecutorWithOptions(t *testing.T) {
	executor := agents.NewExecutorWithMaxIterations(10)
	if executor == nil {
		t.Error("NewExecutorWithMaxIterations() returned nil")
	}

	executor2 := agents.NewExecutorWithReturnIntermediateSteps(true)
	if executor2 == nil {
		t.Error("NewExecutorWithReturnIntermediateSteps() returned nil")
	}

	executor3 := agents.NewExecutorWithHandleParsingErrors(false)
	if executor3 == nil {
		t.Error("NewExecutorWithHandleParsingErrors() returned nil")
	}
}

func TestNewToolRegistry(t *testing.T) {
	registry := agents.NewToolRegistry()
	if registry == nil {
		t.Error("NewToolRegistry() returned nil")
	}
}

func TestDefaultConfig(t *testing.T) {
	config := agents.DefaultConfig()
	if config == nil {
		t.Error("DefaultConfig() returned nil")
	}

	// Test default values
	if config.DefaultMaxRetries != 3 {
		t.Errorf("DefaultConfig().DefaultMaxRetries = %v, want %v", config.DefaultMaxRetries, 3)
	}
	if config.DefaultTimeout != 30*time.Second {
		t.Errorf("DefaultConfig().DefaultTimeout = %v, want %v", config.DefaultTimeout, 30*time.Second)
	}
	if !config.EnableMetrics {
		t.Error("DefaultConfig().EnableMetrics should be true by default")
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		config  *agents.Config
		name    string
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  agents.DefaultConfig(),
			wantErr: false,
		},
		{
			name: "negative max retries",
			config: &agents.Config{
				DefaultMaxRetries:    -1,
				DefaultRetryDelay:    2 * time.Second,
				DefaultTimeout:       30 * time.Second,
				DefaultMaxIterations: 15,
			},
			wantErr: true,
		},
		{
			name: "zero timeout",
			config: &agents.Config{
				DefaultMaxRetries:    3,
				DefaultRetryDelay:    2 * time.Second,
				DefaultTimeout:       0,
				DefaultMaxIterations: 15,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := agents.ValidateConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHealthCheck(t *testing.T) {
	llm := &mockLLM{}
	tools := createMockTools()
	agent, err := agents.NewBaseAgent("test-agent", llm, tools)
	if err != nil {
		t.Fatalf("Failed to create test agent: %v", err)
	}

	// Initialize the agent to change its state to ready
	err = agent.Initialize(map[string]any{})
	if err != nil {
		t.Fatalf("Failed to initialize test agent: %v", err)
	}

	status := agents.HealthCheck(agent)
	if status == nil {
		t.Error("HealthCheck() returned nil")
	}

	// Check that status contains expected fields
	if name, ok := status["name"].(string); !ok || name != "test-agent" {
		t.Errorf("HealthCheck() name = %v, want 'test-agent'", name)
	}

	if state, ok := status["state"].(iface.AgentState); !ok {
		t.Error("HealthCheck() missing or invalid state field")
	} else if state != iface.StateReady {
		t.Errorf("HealthCheck() state = %v, want %v", state, iface.StateReady)
	}
}

func TestListAgentStates(t *testing.T) {
	states := agents.ListAgentStates()
	expectedStates := []iface.AgentState{
		iface.StateInitializing,
		iface.StateReady,
		iface.StateRunning,
		iface.StatePaused,
		iface.StateError,
		iface.StateShutdown,
	}

	if len(states) != len(expectedStates) {
		t.Errorf("ListAgentStates() returned %d states, want %d", len(states), len(expectedStates))
	}

	for i, expected := range expectedStates {
		if i >= len(states) || states[i] != expected {
			t.Errorf("ListAgentStates()[%d] = %v, want %v", i, states[i], expected)
		}
	}
}

func TestGetAgentStateString(t *testing.T) {
	tests := []struct {
		state iface.AgentState
		want  string
	}{
		{iface.StateInitializing, "Initializing"},
		{iface.StateReady, "Ready"},
		{iface.StateRunning, "Running"},
		{iface.StatePaused, "Paused"},
		{iface.StateError, "Error"},
		{iface.StateShutdown, "Shutdown"},
		{iface.AgentState("unknown"), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			got := agents.GetAgentStateString(tt.state)
			if got != tt.want {
				t.Errorf("GetAgentStateString(%v) = %v, want %v", tt.state, got, tt.want)
			}
		})
	}
}

// Test agent as Runnable interface.
func TestAgentAsRunnable(t *testing.T) {
	llm := &mockLLM{}
	ctx := context.Background()
	tools := createMockTools()
	agent, err := agents.NewBaseAgent("runnable-agent", llm, tools)
	if err != nil {
		t.Fatalf("Failed to create test agent: %v", err)
	}

	input := map[string]any{"input": "test query"}

	// Test Invoke - expect error since BaseAgent doesn't implement executeWithInput
	_, err = agent.Invoke(ctx, input)
	if err == nil {
		t.Error("Agent.Invoke() expected error for unimplemented executeWithInput")
	}

	// Test Batch - expect error since BaseAgent doesn't implement executeWithInput
	inputs := []any{input, input}
	_, err = agent.Batch(ctx, inputs)
	if err == nil {
		t.Error("Agent.Batch() expected error for unimplemented executeWithInput")
	}

	// Test Stream - expect error since BaseAgent doesn't implement executeWithInput
	stream, err := agent.Stream(ctx, input)
	if err != nil {
		t.Errorf("Agent.Stream() unexpected error: %v", err)
	} else {
		select {
		case result := <-stream:
			if result == nil {
				t.Error("Agent.Stream() should return an error result")
			}
			// Should be an error
			if _, ok := result.(error); !ok {
				t.Error("Agent.Stream() expected error result")
			}
		case <-time.After(100 * time.Millisecond):
			t.Error("Agent.Stream() timed out waiting for result")
		}
	}
}

// Test error scenarios for agent creation.
func TestAgentCreationErrors(t *testing.T) {
	tests := []struct {
		llm       llmsiface.LLM
		name      string
		errString string
		tools     []tools.Tool
		wantErr   bool
	}{
		{
			name:      "nil LLM",
			llm:       nil,
			tools:     createMockTools(),
			wantErr:   true,
			errString: "LLM cannot be nil",
		},
		{
			name:    "nil tools slice",
			llm:     &mockLLM{},
			tools:   nil,
			wantErr: false, // BaseAgent allows nil tools
		},
		{
			name:    "empty tools slice",
			llm:     &mockLLM{},
			tools:   []tools.Tool{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent, err := agents.NewBaseAgent("test-agent", tt.llm, tt.tools)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.errString)
					return
				}
				// Use string comparison instead of errors.Is with dynamic errors (err113 compliance)
				if !strings.Contains(err.Error(), tt.errString) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errString, err)
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if agent == nil {
				t.Error("Expected non-nil agent")
			}
		})
	}
}

// Test comprehensive agent factory scenarios.
func TestAgentFactoryComprehensive(t *testing.T) {
	config := agents.DefaultConfig()
	config.DefaultMaxRetries = 10
	config.EnableMetrics = true // Enable metrics for the test
	config.EnableTracing = false

	factory := agents.NewAgentFactory(config)

	t.Run("FactoryWithMetrics", func(t *testing.T) {
		ctx := context.Background()
		llm := &mockLLM{}
		tools := createMockTools()

		agent, err := factory.CreateBaseAgent(ctx, "metrics-test-agent", llm, tools)
		if err != nil {
			t.Errorf("Factory agent creation failed: %v", err)
			return
		}
		if agent == nil {
			t.Error("Expected non-nil agent from factory")
		}

		// Verify agent has some metrics recorder
		if agent.GetMetrics() == nil {
			t.Error("Agent should have a metrics recorder")
		}
	})

	t.Run("FactoryConfigInheritance", func(t *testing.T) {
		llm := &mockLLM{}
		ctx := context.Background()
		tools := createMockTools()

		// Create agent without additional options - should inherit factory config
		agent, err := factory.CreateBaseAgent(ctx, "inherit-config-agent", llm, tools)
		if err != nil {
			t.Errorf("Factory agent creation failed: %v", err)
			return
		}

		// Initialize agent to test config inheritance
		err = agent.Initialize(map[string]any{})
		if err != nil {
			t.Errorf("Agent initialization failed: %v", err)
		}

		// Verify agent is in ready state
		if agent.GetState() != iface.StateReady {
			t.Errorf("Expected agent state Ready, got %v", agent.GetState())
		}
	})

	t.Run("FactoryWithOptionsOverride", func(t *testing.T) {
		llm := &mockLLM{}
		ctx := context.Background()
		tools := createMockTools()

		// Create agent with options that should override factory config
		agent, err := factory.CreateBaseAgent(ctx, "override-agent", llm, tools,
			agents.WithMaxRetries(20), // Override factory's 10
			agents.WithTimeout(60*time.Second),
		)
		if err != nil {
			t.Errorf("Factory agent creation with options failed: %v", err)
			return
		}
		if agent == nil {
			t.Error("Expected non-nil agent from factory with options")
		}
	})
}

// Test agent lifecycle management.
func TestAgentLifecycle(t *testing.T) {
	llm := &mockLLM{}
	ctx := context.Background()
	tools := createMockTools()
	agent, err := agents.NewBaseAgent("lifecycle-test", llm, tools)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Test initial state
	if agent.GetState() != iface.StateInitializing {
		t.Errorf("Expected initial state Initializing, got %v", agent.GetState())
	}

	// Test initialization
	config := map[string]any{
		"max_retries": 1,                    // Use fewer retries for faster test
		"retry_delay": 1 * time.Millisecond, // Use very short delay for test
		"timeout":     "30*time.Second",
	}
	err = agent.Initialize(config)
	if err != nil {
		t.Errorf("Initialization failed: %v", err)
	}

	if agent.GetState() != iface.StateReady {
		t.Errorf("Expected state Ready after initialization, got %v", agent.GetState())
	}

	// Test execution (will fail since BaseAgent doesn't implement doExecute)
	// Use a timeout context to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Execute in a goroutine with timeout
	done := make(chan error, 1)
	go func() {
		done <- agent.Execute()
	}()

	select {
	case err = <-done:
		// Execution completed
	case <-ctx.Done():
		t.Fatal("Execute() timed out")
	}
	if err == nil {
		t.Error("Expected execution to fail for BaseAgent (not implemented)")
	}

	// Test shutdown
	err = agent.Shutdown()
	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}

	if agent.GetState() != iface.StateShutdown {
		t.Errorf("Expected state Shutdown after shutdown, got %v", agent.GetState())
	}
}

// Test comprehensive configuration scenarios.
func TestConfigurationScenarios(t *testing.T) {
	t.Run("DefaultConfigValidation", func(t *testing.T) {
		config := agents.DefaultConfig()
		if err := agents.ValidateConfig(config); err != nil {
			t.Errorf("Default config should be valid: %v", err)
		}
	})

	t.Run("InvalidConfigScenarios", func(t *testing.T) {
		tests := []struct {
			config  *agents.Config
			name    string
			wantErr bool
		}{
			{
				name: "negative max retries",
				config: &agents.Config{
					DefaultMaxRetries:    -1,
					DefaultRetryDelay:    2 * time.Second,
					DefaultTimeout:       30 * time.Second,
					DefaultMaxIterations: 15,
				},
				wantErr: true,
			},
			{
				name: "zero timeout",
				config: &agents.Config{
					DefaultMaxRetries:    3,
					DefaultRetryDelay:    2 * time.Second,
					DefaultTimeout:       0,
					DefaultMaxIterations: 15,
				},
				wantErr: true,
			},
			{
				name: "negative retry delay",
				config: &agents.Config{
					DefaultMaxRetries:    3,
					DefaultRetryDelay:    -1 * time.Second,
					DefaultTimeout:       30 * time.Second,
					DefaultMaxIterations: 15,
				},
				wantErr: true,
			},
			{
				name: "zero max iterations",
				config: &agents.Config{
					DefaultMaxRetries:    3,
					DefaultRetryDelay:    2 * time.Second,
					DefaultTimeout:       30 * time.Second,
					DefaultMaxIterations: 0,
				},
				wantErr: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := agents.ValidateConfig(tt.config)
				if tt.wantErr && err == nil {
					t.Errorf("Expected validation error for config: %v", tt.name)
				}
				if !tt.wantErr && err != nil {
					t.Errorf("Unexpected validation error for config %v: %v", tt.name, err)
				}
			})
		}
	})

	t.Run("ConfigWithAgentConfigs", func(t *testing.T) {
		config := agents.DefaultConfig()
		config.AgentConfigs = map[string]schema.AgentConfig{
			"test-agent": {
				Name: "test-agent",
				Settings: map[string]any{
					"max_retries": 5,
				},
			},
		}

		if err := agents.ValidateConfig(config); err != nil {
			t.Errorf("Config with agent configs should be valid: %v", err)
		}
	})
}

// Test tool registry functionality.
func TestToolRegistry(t *testing.T) {
	registry := agents.NewToolRegistry()
	if registry == nil {
		t.Fatal("NewToolRegistry() returned nil")
	}

	// Test registering tools
	tool1 := &mockTool{name: "tool1", description: "first tool"}
	tool2 := &mockTool{name: "tool2", description: "second tool"}

	err := registry.RegisterTool(tool1)
	if err != nil {
		t.Errorf("Failed to register tool1: %v", err)
	}

	err = registry.RegisterTool(tool2)
	if err != nil {
		t.Errorf("Failed to register tool2: %v", err)
	}

	// Test duplicate registration
	err = registry.RegisterTool(&mockTool{name: "tool1", description: "duplicate"})
	if err == nil {
		t.Error("Expected error when registering duplicate tool")
	}

	// Test getting tools
	retrievedTool, err := registry.GetTool("tool1")
	if err != nil {
		t.Errorf("Failed to get tool1: %v", err)
	}
	if retrievedTool != tool1 {
		t.Error("Retrieved tool should be the same instance")
	}

	// Test getting non-existent tool
	_, err = registry.GetTool("non-existent")
	if err == nil {
		t.Error("Expected error when getting non-existent tool")
	}

	// Test listing tools
	tools := registry.ListTools()
	expectedTools := []string{"tool1", "tool2"}
	if len(tools) != len(expectedTools) {
		t.Errorf("Expected %d tools, got %d", len(expectedTools), len(tools))
	}

	// Test tool descriptions
	descriptions := registry.GetToolDescriptions()
	if descriptions == "" {
		t.Error("Expected non-empty tool descriptions")
	}
	if !strings.Contains(descriptions, "tool1") || !strings.Contains(descriptions, "tool2") {
		t.Error("Tool descriptions should contain registered tool names")
	}
}

// Test error handling and custom error types.
func TestErrorHandling(t *testing.T) {
	t.Run("AgentErrorCreation", func(t *testing.T) {
		originalErr := errTestOriginal
		agentErr := agents.NewAgentError("test_operation", "test_code", originalErr)

		if agentErr.Op != "test_operation" {
			t.Errorf("Expected operation 'test_operation', got '%s'", agentErr.Op)
		}
		if agentErr.Code != "test_code" {
			t.Errorf("Expected code 'test_code', got '%s'", agentErr.Code)
		}
		if !errors.Is(agentErr, originalErr) {
			t.Error("AgentError should wrap original error")
		}
	})

	t.Run("ValidationError", func(t *testing.T) {
		valErr := agents.NewValidationError("test_field", "test message")
		if valErr.Field != "test_field" {
			t.Errorf("Expected field 'test_field', got '%s'", valErr.Field)
		}
		if valErr.Message != "test message" {
			t.Errorf("Expected message 'test message', got '%s'", valErr.Message)
		}
	})

	t.Run("IsRetryable", func(t *testing.T) {
		retryableErr := agents.NewAgentError("test", agents.ErrCodeTimeout, errTestTimeout)
		if !agents.IsRetryable(retryableErr) {
			t.Error("Timeout error should be retryable")
		}

		nonRetryableErr := agents.NewAgentError("test", agents.ErrCodeInvalidInput, errTestInvalid)
		if agents.IsRetryable(nonRetryableErr) {
			t.Error("Invalid input error should not be retryable")
		}
	})

	t.Run("ErrorTypeChecking", func(t *testing.T) {
		valErr := agents.NewValidationError("field", "message")
		if !agents.IsValidationError(valErr) {
			t.Error("Should identify validation error")
		}

		agentErr := agents.NewAgentError("op", "code", errTestErr)
		if agents.IsValidationError(agentErr) {
			t.Error("Should not identify agent error as validation error")
		}
	})
}

// Test event handling scenarios.
func TestEventHandling(t *testing.T) {
	llm := &mockLLM{}
	tools := createMockTools()
	agent, err := agents.NewBaseAgent("event-test", llm, tools)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	t.Run("EventRegistrationAndEmission", func(t *testing.T) {
		eventReceived := false
		var receivedPayload any

		handler := func(payload any) error {
			eventReceived = true
			receivedPayload = payload
			return nil
		}

		agent.RegisterEventHandler("test_event", handler)
		agent.EmitEvent("test_event", "test_payload")

		if !eventReceived {
			t.Error("Event handler should have been called")
		}
		if receivedPayload != "test_payload" {
			t.Errorf("Expected payload 'test_payload', got %v", receivedPayload)
		}
	})

	t.Run("MultipleEventHandlers", func(t *testing.T) {
		handler1Called := false
		handler2Called := false

		handler1 := func(payload any) error {
			handler1Called = true
			return nil
		}
		handler2 := func(payload any) error {
			handler2Called = true
			return nil
		}

		agent.RegisterEventHandler("multi_event", handler1)
		agent.RegisterEventHandler("multi_event", handler2)
		agent.EmitEvent("multi_event", "multi_payload")

		if !handler1Called || !handler2Called {
			t.Error("Both event handlers should have been called")
		}
	})

	t.Run("EventHandlerError", func(t *testing.T) {
		handlerErr := errTestHandler
		handler := func(payload any) error {
			return handlerErr
		}

		agent.RegisterEventHandler("error_event", handler)

		// This should not panic or cause issues
		agent.EmitEvent("error_event", "error_payload")
		// Event emission should continue despite handler errors
	})
}

// Benchmark tests.
func BenchmarkNewBaseAgent(b *testing.B) {
	llm := &mockLLM{}
	tools := createMockTools()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = agents.NewBaseAgent("bench-agent", llm, tools)
	}
}

func BenchmarkAgentInitialization(b *testing.B) {
	llm := &mockLLM{}
	tools := createMockTools()
	agent, _ := agents.NewBaseAgent("bench-agent", llm, tools)

	config := map[string]any{
		"max_retries": 3,
		"timeout":     "30*time.Second",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = agent.Initialize(config)
	}
}

func BenchmarkHealthCheck(b *testing.B) {
	llm := &mockLLM{}
	tools := createMockTools()
	agent, _ := agents.NewBaseAgent("bench-agent", llm, tools)
	agent.Initialize(map[string]any{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = agents.HealthCheck(agent)
	}
}
