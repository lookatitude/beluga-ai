// Package package_pairs provides integration tests between Prompts and LLMs packages.
// This test suite verifies that Prompts work correctly with LLMs
// for dynamic prompt generation, template rendering, and LLM integration.
package package_pairs

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/llms"
	"github.com/lookatitude/beluga-ai/pkg/prompts"
	"github.com/lookatitude/beluga-ai/pkg/prompts/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/tests/integration/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// promptValueToString converts a PromptValue or string to string.
func promptValueToString(renderedAny any) string {
	if promptValue, ok := renderedAny.(iface.PromptValue); ok {
		return promptValue.ToString()
	}
	if str, ok := renderedAny.(string); ok {
		return str
	}
	return ""
}

// TestIntegrationPromptsLLMs tests the integration between Prompts and LLMs packages.
func TestIntegrationPromptsLLMs(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	tests := []struct {
		name        string
		template    string
		variables   map[string]any
		wantErr     bool
		description string
	}{
		{
			name:        "simple_template_with_llm",
			description: "Test simple prompt template rendered and used with LLM",
			template:    "Hello, {{.name}}! How are you?",
			variables:   map[string]any{"name": "Alice"},
			wantErr:     false,
		},
		{
			name:        "complex_template_with_llm",
			description: "Test complex prompt template with multiple variables used with LLM",
			template:    "You are {{.role}}. Your task is to {{.task}}. Context: {{.context}}",
			variables:   map[string]any{"role": "assistant", "task": "help users", "context": "customer support"},
			wantErr:     false,
		},
		{
			name:        "template_with_system_message",
			description: "Test template that generates system message for LLM",
			template:    "System: You are a helpful assistant. User: {{.user_input}}",
			variables:   map[string]any{"user_input": "What is the weather?"},
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// Create prompt manager
			manager, err := prompts.NewPromptManager()
			require.NoError(t, err)
			require.NotNil(t, manager)

			// Create prompt template
			template, err := manager.NewStringTemplate("test-template", tt.template)
			if err != nil {
				if tt.wantErr {
					return // Expected error
				}
				require.NoError(t, err)
				return
			}
			require.NotNil(t, template)

			// Render template
			renderedAny, err := template.Format(ctx, tt.variables)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, renderedAny)

			// Convert to string
			rendered := promptValueToString(renderedAny)
			require.NotEmpty(t, rendered, "Rendered template should not be empty")

			// Create LLM
			llm := llms.NewAdvancedMockChatModel("prompt-test",
				llms.WithResponses("Response to: "+rendered))

			// Convert rendered prompt to messages
			messages := []schema.Message{
				schema.NewHumanMessage(rendered),
			}

			// Generate response with LLM
			response, err := llm.Generate(ctx, messages)
			require.NoError(t, err)
			assert.NotNil(t, response)
			assert.NotEmpty(t, response.GetContent())
		})
	}
}

// TestIntegrationPromptsLLMsAdapterTypes tests different adapter types with LLMs.
func TestIntegrationPromptsLLMsAdapterTypes(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	ctx := context.Background()
	manager, err := prompts.NewPromptManager()
	require.NoError(t, err)

	llm := llms.NewAdvancedMockChatModel("adapter-type-test")

	tests := []struct {
		name      string
		setup     func() (iface.PromptFormatter, error)
		variables map[string]any
		testFunc  func(t *testing.T, rendered any, response schema.Message)
	}{
		{
			name: "default_adapter",
			setup: func() (iface.PromptFormatter, error) {
				return manager.NewDefaultAdapter("default-adapter", "Process: {{.input}}", []string{"input"})
			},
			variables: map[string]any{"input": "test input"},
			testFunc: func(t *testing.T, rendered any, response schema.Message) {
				assert.NotNil(t, rendered)
				assert.NotEmpty(t, response.GetContent())
			},
		},
		{
			name: "chat_adapter",
			setup: func() (iface.PromptFormatter, error) {
				return manager.NewChatAdapter(
					"chat-adapter",
					"System: {{.system}}",
					"User: {{.user}}",
					[]string{"system", "user"},
				)
			},
			variables: map[string]any{
				"system": "You are a helpful assistant",
				"user":   "Hello",
			},
			testFunc: func(t *testing.T, rendered any, response schema.Message) {
				assert.NotNil(t, rendered)
				assert.NotEmpty(t, response.GetContent())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create adapter
			adapter, err := tt.setup()
			require.NoError(t, err)
			require.NotNil(t, adapter)

			// Format prompt
			renderedAny, err := adapter.Format(ctx, tt.variables)
			require.NoError(t, err)
			assert.NotNil(t, renderedAny)

			// Convert to messages if needed
			var messages []schema.Message
			switch v := renderedAny.(type) {
			case []schema.Message:
				// ChatPromptAdapter returns []schema.Message directly
				messages = v
			case iface.PromptValue:
				// Use ToMessages() for ChatPromptValue
				messages = v.ToMessages()
			case string:
				// Default adapter returns string
				messages = []schema.Message{schema.NewHumanMessage(v)}
			default:
				// Try promptValueToString as fallback
				if renderedStr := promptValueToString(renderedAny); renderedStr != "" {
					messages = []schema.Message{schema.NewHumanMessage(renderedStr)}
				} else {
					t.Fatalf("Unexpected rendered type: %T", renderedAny)
				}
			}

			// Generate with LLM
			response, err := llm.Generate(ctx, messages)
			require.NoError(t, err)

			// Verify with test function
			tt.testFunc(t, renderedAny, response)
		})
	}
}

// TestIntegrationPromptsLLMsErrorHandling tests error scenarios.
func TestIntegrationPromptsLLMsErrorHandling(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	ctx := context.Background()
	manager, err := prompts.NewPromptManager()
	require.NoError(t, err)

	tests := []struct {
		name        string
		template    string
		variables   map[string]any
		setupLLM    func() *llms.AdvancedMockChatModel
		expectError bool
	}{
		{
			name:        "template_rendering_error",
			template:    "Hello {{.missing}}",
			variables:   map[string]any{}, // Missing variable
			setupLLM:    func() *llms.AdvancedMockChatModel { return llms.NewAdvancedMockChatModel("test") },
			expectError: true, // Template rendering may fail
		},
		{
			name:      "llm_generation_error",
			template:  "Hello {{.name}}",
			variables: map[string]any{"name": "Alice"},
			setupLLM: func() *llms.AdvancedMockChatModel {
				return llms.NewAdvancedMockChatModel("error-test",
					llms.WithErrorCode(llms.ErrCodeNetworkError))
			},
			expectError: true,
		},
		{
			name:        "invalid_template_name",
			template:    "Hello {{.name}}",
			variables:   map[string]any{"name": "Alice"},
			setupLLM:    func() *llms.AdvancedMockChatModel { return llms.NewAdvancedMockChatModel("test") },
			expectError: false, // Template creation will fail, not LLM
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			llm := tt.setupLLM()

			// Try to create template
			template, err := manager.NewStringTemplate("error-test-template", tt.template)
			if err != nil {
				if tt.expectError {
					return // Expected error
				}
				require.NoError(t, err)
			}

			if template == nil {
				if tt.expectError {
					return // Expected error
				}
				t.Fatal("Template should not be nil")
			}

			// Try to render template
			renderedAny, err := template.Format(ctx, tt.variables)
			if err != nil {
				if tt.expectError {
					return // Expected error
				}
				require.NoError(t, err)
			}

			if renderedAny == nil {
				if tt.expectError {
					return // Expected error
				}
				t.Fatal("Rendered template should not be nil")
			}

			rendered := promptValueToString(renderedAny)
			if rendered == "" && renderedAny != nil {
				t.Fatalf("Rendered template should be convertible to string, got %T", renderedAny)
			}

			// Try to generate with LLM
			messages := []schema.Message{schema.NewHumanMessage(rendered)}
			_, err = llm.Generate(ctx, messages)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestIntegrationPromptsLLMsBatchProcessing tests batch processing with templates.
func TestIntegrationPromptsLLMsBatchProcessing(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	ctx := context.Background()
	manager, err := prompts.NewPromptManager()
	require.NoError(t, err)

	llm := llms.NewAdvancedMockChatModel("batch-prompt-test")

	// Create template
	template, err := manager.NewStringTemplate("batch-template", "Process: {{.item}}")
	require.NoError(t, err)

	// Create batch of inputs
	items := []string{"item1", "item2", "item3", "item4"}
	inputs := make([]any, len(items))

	for i, item := range items {
		renderedAny, err := template.Format(ctx, map[string]any{"item": item})
		require.NoError(t, err)
		rendered := promptValueToString(renderedAny)
		require.NotEmpty(t, rendered, "Rendered template should not be empty")
		inputs[i] = rendered
	}

	// Process batch
	results, err := llm.Batch(ctx, inputs)
	require.NoError(t, err)
	assert.Len(t, results, len(items))

	// Verify all results
	for i, result := range results {
		assert.NotNil(t, result, "Result %d should not be nil", i)
		if msg, ok := result.(schema.Message); ok {
			assert.NotEmpty(t, msg.GetContent(), "Result %d should have content", i)
		}
	}
}

// TestIntegrationPromptsLLMsRealWorldScenarios tests realistic usage patterns.
func TestIntegrationPromptsLLMsRealWorldScenarios(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	ctx := context.Background()
	manager, err := prompts.NewPromptManager()
	require.NoError(t, err)

	llm := llms.NewAdvancedMockChatModel("realworld-prompt-test")

	scenarios := []struct {
		name     string
		scenario func(t *testing.T)
	}{
		{
			name: "customer_support_prompt",
			scenario: func(t *testing.T) {
				t.Helper()
				template, err := manager.NewStringTemplate("support-template",
					"You are a customer support agent.\n"+
						"Customer issue: {{.issue}}\n"+
						"Customer name: {{.name}}\n"+
						"Previous interactions: {{.history}}\n"+
						"Please provide a helpful response.")
				require.NoError(t, err)

				renderedAny, err := template.Format(ctx, map[string]any{
					"issue":   "Cannot log in",
					"name":    "Alice",
					"history": "Previous attempt at 10:00 AM",
				})
				require.NoError(t, err)
				rendered := promptValueToString(renderedAny)
				require.NotEmpty(t, rendered, "Rendered template should not be empty")

				messages := []schema.Message{schema.NewHumanMessage(rendered)}
				response, err := llm.Generate(ctx, messages)
				require.NoError(t, err)
				assert.NotEmpty(t, response.GetContent())
			},
		},
		{
			name: "code_generation_prompt",
			scenario: func(t *testing.T) {
				t.Helper()
				template, err := manager.NewStringTemplate("code-template",
					"Generate {{.language}} code for: {{.task}}\n"+
						"Requirements: {{.requirements}}")
				require.NoError(t, err)

				renderedAny, err := template.Format(ctx, map[string]any{
					"language":     "Go",
					"task":         "HTTP server",
					"requirements": "Use standard library, handle errors",
				})
				require.NoError(t, err)
				rendered := promptValueToString(renderedAny)
				require.NotEmpty(t, rendered, "Rendered template should not be empty")

				messages := []schema.Message{schema.NewHumanMessage(rendered)}
				response, err := llm.Generate(ctx, messages)
				require.NoError(t, err)
				assert.NotEmpty(t, response.GetContent())
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			scenario.scenario(t)
		})
	}
}

// BenchmarkIntegrationPromptsLLMs benchmarks Prompts-LLMs integration performance.
func BenchmarkIntegrationPromptsLLMs(b *testing.B) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	manager, err := prompts.NewPromptManager()
	if err != nil {
		b.Fatalf("Manager creation failed: %v", err)
	}

	llm := llms.NewAdvancedMockChatModel("benchmark-prompt-test")

	template, err := manager.NewStringTemplate("benchmark-template", "Test: {{.input}}")
	if err != nil {
		b.Fatalf("Template creation failed: %v", err)
	}

	b.Run("TemplateRenderAndGenerate", func(b *testing.B) {
		b.ResetTimer()
		ctx := context.Background()
		for i := 0; i < b.N; i++ {
			renderedAny, err := template.Format(ctx, map[string]any{"input": "test"})
			if err != nil {
				b.Errorf("Template rendering error: %v", err)
				continue
			}
			rendered := promptValueToString(renderedAny)
			if rendered == "" {
				b.Errorf("Rendered template should not be empty, got %T", renderedAny)
				continue
			}

			messages := []schema.Message{schema.NewHumanMessage(rendered)}
			_, err = llm.Generate(ctx, messages)
			if err != nil {
				b.Errorf("Generation error: %v", err)
			}
		}
	})
}
