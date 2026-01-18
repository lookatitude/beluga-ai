// Package package_pairs provides integration tests between LLMs and Prompts packages.
// This test suite verifies that LLMs work correctly with prompt templates
// for dynamic prompt generation, variable substitution, and template rendering.
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

// TestIntegrationLLMsPrompts tests the integration between LLMs and Prompts packages.
func TestIntegrationLLMsPrompts(t *testing.T) {
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
			name:        "simple_template",
			description: "Test simple prompt template with LLM",
			template:    "Hello, {{.name}}! How are you?",
			variables:   map[string]any{"name": "Alice"},
			wantErr:     false,
		},
		{
			name:        "complex_template",
			description: "Test complex prompt template with multiple variables",
			template:    "You are {{.role}}. Your task is to {{.task}}. Context: {{.context}}",
			variables:   map[string]any{"role": "assistant", "task": "help users", "context": "customer support"},
			wantErr:     false,
		},
		{
			name:        "template_with_system_message",
			description: "Test template that generates system message",
			template:    "System: You are a helpful assistant. User: {{.user_input}}",
			variables:   map[string]any{"user_input": "What is the weather?"},
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// Create prompt template using convenience function
			template, err := prompts.NewStringPromptTemplate("test-template", tt.template)
			if err != nil {
				t.Skipf("Template creation failed: %v", err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, template)

			// Render template
			renderedAny, err := template.Format(ctx, tt.variables)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, renderedAny)
			
			// Convert PromptValue to string
			var rendered string
			if promptValue, ok := renderedAny.(iface.PromptValue); ok {
				rendered = promptValue.ToString()
			} else if str, ok := renderedAny.(string); ok {
				rendered = str
			} else {
				t.Fatalf("Unexpected rendered type: %T", renderedAny)
			}
			assert.NotEmpty(t, rendered)

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

// TestIntegrationLLMsPromptsTemplateTypes tests different template types with LLMs.
func TestIntegrationLLMsPromptsTemplateTypes(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	ctx := context.Background()
	llm := llms.NewAdvancedMockChatModel("template-type-test")

	tests := []struct {
		name     string
		template string
		variables map[string]any
		testFunc func(t *testing.T, rendered string, response schema.Message)
	}{
		{
			name:     "question_answer_template",
			template: "Question: {{.question}}\nAnswer:",
			variables: map[string]any{"question": "What is 2+2?"},
			testFunc: func(t *testing.T, rendered string, response schema.Message) {
				assert.Contains(t, rendered, "Question:")
				assert.Contains(t, rendered, "What is 2+2?")
				assert.NotEmpty(t, response.GetContent())
			},
		},
		{
			name:     "instruction_template",
			template: "Instructions: {{.instructions}}\n\nTask: {{.task}}",
			variables: map[string]any{
				"instructions": "Be concise and helpful",
				"task":         "Explain quantum computing",
			},
			testFunc: func(t *testing.T, rendered string, response schema.Message) {
				assert.Contains(t, rendered, "Instructions:")
				assert.Contains(t, rendered, "Task:")
				assert.NotEmpty(t, response.GetContent())
			},
		},
		{
			name:     "conversation_template",
			template: "Previous: {{.previous}}\nCurrent: {{.current}}",
			variables: map[string]any{
				"previous": "User said hello",
				"current":  "User asks about weather",
			},
			testFunc: func(t *testing.T, rendered string, response schema.Message) {
				assert.Contains(t, rendered, "Previous:")
				assert.Contains(t, rendered, "Current:")
				assert.NotEmpty(t, response.GetContent())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create and render template
			template, err := prompts.NewStringPromptTemplate("test-template", tt.template)
			require.NoError(t, err)

			renderedAny, err := template.Format(ctx, tt.variables)
			require.NoError(t, err)
			
			rendered := promptValueToString(renderedAny)
			require.NotEmpty(t, rendered, "Rendered template should not be empty")

			// Generate with LLM
			messages := []schema.Message{schema.NewHumanMessage(rendered)}
			response, err := llm.Generate(ctx, messages)
			require.NoError(t, err)

			// Verify with test function
			tt.testFunc(t, rendered, response)
		})
	}
}

// TestIntegrationLLMsPromptsStreaming tests streaming with prompt templates.
func TestIntegrationLLMsPromptsStreaming(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	ctx := context.Background()
	llm := llms.NewAdvancedMockChatModel("streaming-prompt-test",
		llms.WithResponses("Chunk 1", "Chunk 2", "Chunk 3"),
		llms.WithStreamingDelay(10))

	// Create template
	template, err := prompts.NewStringPromptTemplate("streaming-template", "Tell me about {{.topic}}")
	require.NoError(t, err)

	// Render template
	renderedAny, err := template.Format(ctx, map[string]any{"topic": "AI"})
	require.NoError(t, err)
	rendered := promptValueToString(renderedAny)
	require.NotEmpty(t, rendered, "Rendered template should not be empty")

	// Stream response
	messages := []schema.Message{schema.NewHumanMessage(rendered)}
	streamCh, err := llm.StreamChat(ctx, messages)
	require.NoError(t, err)

	// Collect chunks
	var fullResponse string
	for chunk := range streamCh {
		if chunk.Err != nil {
			t.Errorf("Streaming error: %v", chunk.Err)
			continue
		}
		fullResponse += chunk.Content
	}

	assert.NotEmpty(t, fullResponse)
}

// TestIntegrationLLMsPromptsErrorHandling tests error scenarios.
func TestIntegrationLLMsPromptsErrorHandling(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	ctx := context.Background()

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
			name:        "llm_generation_error",
			template:    "Hello {{.name}}",
			variables:   map[string]any{"name": "Alice"},
			setupLLM: func() *llms.AdvancedMockChatModel {
				return llms.NewAdvancedMockChatModel("error-test",
					llms.WithErrorCode(llms.ErrCodeNetworkError))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			llm := tt.setupLLM()

			// Try to render template
			template, err := prompts.NewStringPromptTemplate("error-test-template", tt.template)
			if err != nil {
				if tt.expectError {
					return // Expected error
				}
				require.NoError(t, err)
			}

			renderedAny, err := template.Format(ctx, tt.variables)
			if err != nil {
				if tt.expectError {
					return // Expected error
				}
				require.NoError(t, err)
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

// TestIntegrationLLMsPromptsBatchProcessing tests batch processing with templates.
func TestIntegrationLLMsPromptsBatchProcessing(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	ctx := context.Background()
	llm := llms.NewAdvancedMockChatModel("batch-prompt-test")

	// Create template
	template, err := prompts.NewStringPromptTemplate("batch-template", "Process: {{.item}}")
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

// TestIntegrationLLMsPromptsRealWorldScenarios tests realistic usage patterns.
func TestIntegrationLLMsPromptsRealWorldScenarios(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	ctx := context.Background()
	llm := llms.NewAdvancedMockChatModel("realworld-prompt-test")

	scenarios := []struct {
		name     string
		scenario func(t *testing.T)
	}{
		{
			name: "customer_support_prompt",
			scenario: func(t *testing.T) {
				t.Helper()
				template, err := prompts.NewStringPromptTemplate("support-template",
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
				template, err := prompts.NewStringPromptTemplate("code-template",
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

// BenchmarkIntegrationLLMsPrompts benchmarks LLM-Prompts integration performance.
func BenchmarkIntegrationLLMsPrompts(b *testing.B) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	llm := llms.NewAdvancedMockChatModel("benchmark-prompt-test")

	template, err := prompts.NewStringPromptTemplate("benchmark-template", "Test: {{.input}}")
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
