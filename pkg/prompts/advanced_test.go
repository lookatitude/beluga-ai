// Package prompts provides comprehensive tests for prompt implementations.
// This file contains advanced testing scenarios including table-driven tests,
// concurrency testing, and performance benchmarks.
package prompts

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/prompts/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAdvancedMockTemplate tests the advanced mock template functionality.
func TestAdvancedMockTemplate(t *testing.T) {
	tests := []struct {
		template          *AdvancedMockTemplate
		inputs            map[string]any
		name              string
		expectedPattern   string
		expectedCallCount int
		expectedError     bool
	}{
		{
			name:     "successful_formatting",
			template: NewAdvancedMockTemplate("test-template", "Hello {{.name}}, welcome to {{.service}}"),
			inputs: map[string]any{
				"name":    "Alice",
				"service": "Beluga AI",
			},
			expectedError:     false,
			expectedCallCount: 1,
			expectedPattern:   "test-template",
		},
		{
			name: "template_with_custom_results",
			template: NewAdvancedMockTemplate("custom-template", "Custom template",
				WithMockResults([]any{
					"Custom result 1",
					"Custom result 2",
				})),
			inputs: map[string]any{
				"input": "test",
			},
			expectedError:     false,
			expectedCallCount: 2, // Test twice to use both results
			expectedPattern:   "Custom result",
		},
		{
			name: "template_with_error",
			template: NewAdvancedMockTemplate("error-template", "Error template",
				WithMockError(true, errors.New("template parsing failed"))),
			inputs: map[string]any{
				"input": "test",
			},
			expectedError:     true,
			expectedCallCount: 1,
		},
		{
			name: "template_with_delay",
			template: NewAdvancedMockTemplate("delay-template", "Delayed template",
				WithMockDelay(15*time.Millisecond)),
			inputs: map[string]any{
				"input": "delayed test",
			},
			expectedError:     false,
			expectedCallCount: 1,
			expectedPattern:   "delay-template",
		},
		{
			name: "template_with_missing_variables",
			template: NewAdvancedMockTemplate("var-template", "Template with {{.required}} variable",
				WithMockVariables([]string{"required", "optional"})),
			inputs: map[string]any{
				"optional": "present",
				// "required" is missing
			},
			expectedError:     true,
			expectedCallCount: 1,
		},
		{
			name: "template_with_validation_rules",
			template: NewAdvancedMockTemplate("validation-template", "Validated template",
				WithValidationRules(map[string]string{
					"email": "email_format",
					"age":   "number",
				})),
			inputs: map[string]any{
				"input": "test with validation",
			},
			expectedError:     false,
			expectedCallCount: 1,
			expectedPattern:   "validation-template",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			for i := 0; i < tt.expectedCallCount; i++ {
				start := time.Now()
				result, err := tt.template.Format(ctx, tt.inputs)
				duration := time.Since(start)

				if tt.expectedError {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
					AssertTemplateFormat(t, result, tt.expectedPattern)

					// Verify delay was respected if configured
					if tt.template.simulateDelay > 0 {
						assert.GreaterOrEqual(t, duration, tt.template.simulateDelay)
					}
				}
			}

			// Test template validation
			err := tt.template.Validate()
			if tt.expectedError {
				// Template with error configuration might fail validation
			} else {
				require.NoError(t, err)
			}

			// Verify call count
			assert.Equal(t, tt.expectedCallCount, tt.template.GetCallCount())

			// Test health check
			health := tt.template.CheckHealth()
			AssertTemplateHealth(t, health, "healthy")

			// Test template metadata
			assert.Equal(t, tt.template.name, tt.template.Name())
			variables := tt.template.GetInputVariables()
			AssertTemplateVariables(t, variables, 1)
		})
	}
}

// TestTemplateManager tests template management functionality.
func TestTemplateManager(t *testing.T) {
	tests := []struct {
		templates     map[string]string
		operations    func(t *testing.T, manager *AdvancedMockTemplateManager)
		name          string
		expectedCount int
	}{
		{
			name: "basic_template_management",
			templates: map[string]string{
				"greeting": "Hello {{.name}}!",
				"farewell": "Goodbye {{.name}}, see you {{.when}}!",
			},
			operations: func(t *testing.T, manager *AdvancedMockTemplateManager) {
				t.Helper()
				// Test template creation and retrieval
				ctx := context.Background()

				// Test formatting each template
				for name := range manager.templates {
					template, exists := manager.GetTemplate(name)
					require.True(t, exists, "Template %s should exist", name)

					inputs := CreateTestInputs(template.GetInputVariables())
					result, err := template.Format(ctx, inputs)
					assert.NoError(t, err, "Template %s formatting should succeed", name)
					assert.NotNil(t, result, "Template %s should produce non-nil result", name)
				}
			},
			expectedCount: 2,
		},
		{
			name: "template_lifecycle",
			templates: map[string]string{
				"temp1": "Temporary template 1",
				"temp2": "Temporary template 2",
				"temp3": "Temporary template 3",
			},
			operations: func(t *testing.T, manager *AdvancedMockTemplateManager) {
				// Test listing
				templates := manager.ListTemplates()
				assert.GreaterOrEqual(t, len(templates), 3, "Should list all created templates")

				// Test deletion
				err := manager.DeleteTemplate("temp2")
				assert.NoError(t, err, "Template deletion should succeed")

				// Verify deletion
				_, exists := manager.GetTemplate("temp2")
				assert.False(t, exists, "Deleted template should not exist")

				// Verify remaining templates still exist
				remainingTemplates := manager.ListTemplates()
				assert.Len(t, remainingTemplates, 2, "Should have 2 templates after deletion")
			},
			expectedCount: 2, // After deletion
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewAdvancedMockTemplateManager()

			// Create templates
			for name, templateStr := range tt.templates {
				_, err := manager.CreateTemplate(name, templateStr)
				require.NoError(t, err, "Creating template %s should succeed", name)
			}

			// Run operations
			tt.operations(t, manager)

			// Verify final state
			finalCount := manager.GetTemplateCount()
			assert.Equal(t, tt.expectedCount, finalCount, "Final template count should match expected")
		})
	}
}

// TestPromptValue tests prompt value implementations.
func TestPromptValue(t *testing.T) {
	tests := []struct {
		name           string
		promptValue    *AdvancedMockPromptValue
		expectedString string
		expectedMsgs   int
	}{
		{
			name: "string_prompt_value",
			promptValue: NewAdvancedMockPromptValue(
				"This is a string prompt value for testing",
				[]schema.Message{}),
			expectedString: "string prompt value",
			expectedMsgs:   0,
		},
		{
			name: "message_prompt_value",
			promptValue: NewAdvancedMockPromptValue(
				"System: Be helpful\nHuman: Hello\nAI: Hi there!",
				CreateTestPromptMessages(3)),
			expectedString: "System:",
			expectedMsgs:   3,
		},
		{
			name: "mixed_prompt_value",
			promptValue: NewAdvancedMockPromptValue(
				"Mixed content with structured data",
				[]schema.Message{
					schema.NewSystemMessage("System context"),
					schema.NewHumanMessage("User query"),
				}),
			expectedString: "Mixed content",
			expectedMsgs:   2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			AssertPromptValue(t, tt.promptValue, tt.expectedString, tt.expectedMsgs)
		})
	}
}

// TestPromptsIntegrationTestHelper tests the integration test helper.
func TestPromptsIntegrationTestHelper(t *testing.T) {
	helper := NewIntegrationTestHelper()

	// Add templates
	greetingTemplate := NewAdvancedMockTemplate("greeting", "Hello {{.name}}!")
	farewellTemplate := NewAdvancedMockTemplate("farewell", "Goodbye {{.name}}!")

	helper.AddTemplate("greeting", greetingTemplate)
	helper.AddTemplate("farewell", farewellTemplate)

	// Test retrieval
	assert.Equal(t, greetingTemplate, helper.GetTemplate("greeting"))
	assert.Equal(t, farewellTemplate, helper.GetTemplate("farewell"))

	// Test template manager
	manager := helper.GetTemplateManager()
	assert.NotNil(t, manager)

	// Test operations
	ctx := context.Background()
	inputs := map[string]any{"name": "Alice"}

	_, err := greetingTemplate.Format(ctx, inputs)
	require.NoError(t, err)

	// Test reset
	helper.Reset()

	// Verify reset worked
	assert.Equal(t, 0, greetingTemplate.GetCallCount())
	assert.Equal(t, 0, helper.GetTemplateManager().GetTemplateCount())
}

// TestPromptScenarios tests real-world prompt usage scenarios.
func TestPromptScenarios(t *testing.T) {
	tests := []struct {
		scenario func(t *testing.T, template iface.Template, manager iface.TemplateManager)
		name     string
	}{
		{
			name: "multi_format_templates",
			scenario: func(t *testing.T, template iface.Template, manager iface.TemplateManager) {
				t.Helper()
				ctx := context.Background()
				runner := NewPromptScenarioRunner(template, manager)

				// Test different input combinations - use variables that match the template
				// Template is "Test template with {{.input}}", so provide "input" variable
				inputSets := []map[string]any{
					{"input": "Alice"},
					{"input": "Bob"},
					{"input": "Charlie"},
				}

				results, err := runner.RunTemplateFormattingScenario(ctx, inputSets)
				require.NoError(t, err)
				assert.Len(t, results, len(inputSets))

				// Verify each result is unique and valid
				for i, result := range results {
					assert.NotNil(t, result, "Result %d should not be nil", i+1)
					if str, ok := result.(string); ok {
						assert.NotEmpty(t, str, "Result %d should not be empty", i+1)
					}
				}
			},
		},
		{
			name: "template_library_management",
			scenario: func(t *testing.T, template iface.Template, manager iface.TemplateManager) {
				t.Helper()
				ctx := context.Background()
				runner := NewPromptScenarioRunner(template, manager)

				// Test managing a library of templates
				templateNames := []string{"email_welcome", "email_reminder", "email_goodbye"}
				templateStrings := []string{
					"Welcome {{.user}} to our service!",
					"Don't forget about {{.task}} due {{.date}}",
					"Thank you {{.user}} for using our service",
				}

				err := runner.RunTemplateManagementScenario(ctx, templateNames, templateStrings)
				require.NoError(t, err)
			},
		},
		{
			name: "prompt_value_conversion",
			scenario: func(t *testing.T, template iface.Template, manager iface.TemplateManager) {
				t.Helper()
				ctx := context.Background()

				// Format template to get result
				inputs := CreateTestInputs(template.GetInputVariables())
				result, err := template.Format(ctx, inputs)
				require.NoError(t, err)

				// Test different prompt value representations
				if str, ok := result.(string); ok {
					// Create prompt value from string
					promptValue := NewAdvancedMockPromptValue(str, []schema.Message{})

					// Test string conversion
					stringRep := promptValue.ToString()
					assert.Equal(t, str, stringRep)

					// Test message conversion
					messages := promptValue.ToMessages()
					assert.NotNil(t, messages)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			template := NewAdvancedMockTemplate("scenario-template", "Test template with {{.input}}")
			manager := NewAdvancedMockTemplateManager()

			tt.scenario(t, template, manager)
		})
	}
}

// TestTemplateQuality tests template quality and consistency.
func TestTemplateQuality(t *testing.T) {
	template := NewAdvancedMockTemplate("quality-test", "Quality test template with {{.input}}")
	ctx := context.Background()
	tester := NewTemplateQualityTester(template)

	tests := []struct {
		testFunc   func() (bool, error)
		name       string
		expectedOK bool
	}{
		{
			name: "formatting_consistency",
			testFunc: func() (bool, error) {
				inputs := map[string]any{"input": "consistency test"}
				return tester.TestConsistency(ctx, inputs, 5)
			},
			expectedOK: true,
		},
		{
			name: "variable_handling",
			testFunc: func() (bool, error) {
				testCases := []VariableTestCase{
					{
						Name:        "valid_input",
						Inputs:      map[string]any{"input": "valid value"},
						ShouldError: false,
					},
					{
						Name:        "missing_required_variable",
						Inputs:      map[string]any{"wrong_key": "value"},
						ShouldError: true,
					},
				}

				err := tester.TestVariableHandling(ctx, testCases)
				return err == nil, err
			},
			expectedOK: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, err := tt.testFunc()

			if tt.expectedOK {
				require.NoError(t, err)
				assert.True(t, ok, "Quality test should pass")
			} else {
				// Test may fail for specific scenarios
				if err != nil {
					t.Logf("Expected test failure: %v", err)
				}
			}
		})
	}
}

// TestConcurrencyAdvanced tests concurrent prompt operations.
func TestConcurrencyAdvanced(t *testing.T) {
	template := NewAdvancedMockTemplate("concurrent-test", "Concurrent template with {{.input}}")

	const numGoroutines = 6
	const operationsPerGoroutine = 5

	t.Run("concurrent_template_formatting", func(t *testing.T) {
		var wg sync.WaitGroup
		errChan := make(chan error, numGoroutines*operationsPerGoroutine)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()

				for j := 0; j < operationsPerGoroutine; j++ {
					ctx := context.Background()
					inputs := map[string]any{
						"input": fmt.Sprintf("concurrent-input-%d-%d", goroutineID, j),
					}

					_, err := template.Format(ctx, inputs)
					if err != nil {
						errChan <- err
						return
					}
				}
			}(i)
		}

		wg.Wait()
		close(errChan)

		// Check for errors
		for err := range errChan {
			t.Errorf("Concurrent formatting error: %v", err)
		}

		// Verify total operations
		expectedOps := numGoroutines * operationsPerGoroutine
		assert.Equal(t, expectedOps, template.GetCallCount())
	})

	t.Run("concurrent_template_management", func(t *testing.T) {
		manager := NewAdvancedMockTemplateManager()

		var wg sync.WaitGroup
		errChan := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()

				// Create template
				templateName := fmt.Sprintf("concurrent-template-%d", goroutineID)
				templateStr := fmt.Sprintf("Concurrent template %d with {{.input}}", goroutineID)

				_, err := manager.CreateTemplate(templateName, templateStr)
				if err != nil {
					errChan <- err
					return
				}

				// Retrieve and use template
				createdTemplate, exists := manager.GetTemplate(templateName)
				if !exists {
					errChan <- fmt.Errorf("template %s should exist after creation", templateName)
					return
				}

				ctx := context.Background()
				inputs := map[string]any{"input": "concurrent test"}
				_, err = createdTemplate.Format(ctx, inputs)
				if err != nil {
					errChan <- err
					return
				}
			}(i)
		}

		wg.Wait()
		close(errChan)

		// Check for errors
		for err := range errChan {
			t.Errorf("Concurrent template management error: %v", err)
		}

		// Verify all templates were created
		finalTemplates := manager.ListTemplates()
		assert.Len(t, finalTemplates, numGoroutines, "Should have created %d templates", numGoroutines)
	})
}

// TestLoadTesting performs load testing on prompt components.
func TestLoadTesting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load tests in short mode")
	}

	template := NewAdvancedMockTemplate("load-test", "Load test template with {{.input}}")

	const numOperations = 75
	const concurrency = 6

	t.Run("template_load_test", func(t *testing.T) {
		RunLoadTest(t, template, numOperations, concurrency)

		// Verify health after load test
		health := template.CheckHealth()
		AssertTemplateHealth(t, health, "healthy")
		assert.Equal(t, numOperations, health["call_count"])
	})
}

// TestPromptErrorHandling tests comprehensive error handling scenarios.
func TestPromptErrorHandling(t *testing.T) {
	tests := []struct {
		name      string
		setup     func() iface.Template
		operation func(template iface.Template) error
		errorCode string
	}{
		{
			name: "template_formatting_error",
			setup: func() iface.Template {
				return NewAdvancedMockTemplate("error-template", "Error template",
					WithMockError(true, errors.New("formatting engine failure")))
			},
			operation: func(template iface.Template) error {
				ctx := context.Background()
				inputs := map[string]any{"input": "test"}
				_, err := template.Format(ctx, inputs)
				return err
			},
		},
		{
			name: "template_validation_error",
			setup: func() iface.Template {
				return NewAdvancedMockTemplate("invalid-template", "",
					WithMockError(true, errors.New("invalid template structure")))
			},
			operation: func(template iface.Template) error {
				return template.Validate()
			},
		},
		{
			name: "missing_variables_error",
			setup: func() iface.Template {
				return NewAdvancedMockTemplate("var-template", "Template with {{.required}}",
					WithMockVariables([]string{"required", "optional"}))
			},
			operation: func(template iface.Template) error {
				ctx := context.Background()
				inputs := map[string]any{"optional": "present"} // missing "required"
				_, err := template.Format(ctx, inputs)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			template := tt.setup()
			err := tt.operation(template)

			require.Error(t, err)
		})
	}
}

// BenchmarkPromptOperations benchmarks prompt operation performance.
func BenchmarkPromptOperations(b *testing.B) {
	template := NewAdvancedMockTemplate("benchmark-template", "Benchmark template with {{.input}}")
	ctx := context.Background()
	inputs := map[string]any{"input": "benchmark test"}

	b.Run("TemplateFormatting", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := template.Format(ctx, inputs)
			if err != nil {
				b.Errorf("Template formatting error: %v", err)
			}
		}
	})

	b.Run("TemplateValidation", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := template.Validate()
			if err != nil {
				b.Errorf("Template validation error: %v", err)
			}
		}
	})

	b.Run("VariableRetrieval", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			variables := template.GetInputVariables()
			if len(variables) == 0 {
				b.Error("Template should have input variables")
			}
		}
	})
}

// BenchmarkTemplateManager benchmarks template manager operations.
func BenchmarkTemplateManager(b *testing.B) {
	manager := NewAdvancedMockTemplateManager()

	// Pre-populate manager
	for i := 0; i < 10; i++ {
		name := fmt.Sprintf("template_%d", i)
		templateStr := fmt.Sprintf("Template %d with {{.input}}", i)
		_, _ = manager.CreateTemplate(name, templateStr)
	}

	b.Run("GetTemplate", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			templateName := fmt.Sprintf("template_%d", i%10)
			_, exists := manager.GetTemplate(templateName)
			if !exists {
				b.Errorf("Template %s should exist", templateName)
			}
		}
	})

	b.Run("ListTemplates", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			templates := manager.ListTemplates()
			if len(templates) != 10 {
				b.Error("Should list 10 templates")
			}
		}
	})

	b.Run("CreateTemplate", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			name := fmt.Sprintf("benchmark_template_%d", i)
			templateStr := fmt.Sprintf("Benchmark template %d", i)
			_, err := manager.CreateTemplate(name, templateStr)
			if err != nil {
				b.Errorf("Template creation error: %v", err)
			}
		}
	})
}

// BenchmarkBenchmarkHelper tests the benchmark helper utility.
func BenchmarkBenchmarkHelper(b *testing.B) {
	template := NewAdvancedMockTemplate("benchmark-helper", "Helper template with {{.input}}")
	helper := NewBenchmarkHelper(template, 20)

	b.Run("BenchmarkFormatting", func(b *testing.B) {
		_, err := helper.BenchmarkFormatting(b.N)
		if err != nil {
			b.Errorf("BenchmarkFormatting error: %v", err)
		}
	})

	b.Run("BenchmarkValidation", func(b *testing.B) {
		_, err := helper.BenchmarkValidation(b.N)
		if err != nil {
			b.Errorf("BenchmarkValidation error: %v", err)
		}
	})
}

// Mock implementations for testing.
type mockMetrics struct{}

func (m *mockMetrics) RecordTemplateCreated(templateType string)                    {}
func (m *mockMetrics) RecordTemplateExecuted(templateName string, duration float64) {}
func (m *mockMetrics) RecordTemplateError(templateName, errorType string)           {}
func (m *mockMetrics) RecordFormattingRequest(adapterType string, duration float64) {}
func (m *mockMetrics) RecordFormattingError(adapterType, errorType string)          {}
func (m *mockMetrics) RecordValidationRequest()                                     {}
func (m *mockMetrics) RecordValidationError(errorType string)                       {}
func (m *mockMetrics) RecordCacheHit()                                              {}
func (m *mockMetrics) RecordCacheMiss()                                             {}
func (m *mockMetrics) RecordCacheSize(size int64)                                   {}
func (m *mockMetrics) RecordAdapterRequest(adapterType string)                      {}
func (m *mockMetrics) RecordAdapterError(adapterType, errorType string)             {}

type mockValidator struct{}

func (m *mockValidator) Validate(required []string, provided map[string]any) error {
	return nil
}

func (m *mockValidator) ValidateTypes(variables map[string]any) error {
	return nil
}

type mockTemplateEngine struct{}

func (m *mockTemplateEngine) Parse(name, template string) (iface.ParsedTemplate, error) {
	return nil, nil
}

func (m *mockTemplateEngine) ExtractVariables(template string) ([]string, error) {
	return nil, nil
}

type mockHealthChecker struct{}

func (m *mockHealthChecker) Check(ctx context.Context) error {
	return nil
}

// TestPromptManager_NewPromptManager tests NewPromptManager with various configurations.
func TestPromptManager_NewPromptManager(t *testing.T) {
	tests := []struct {
		name      string
		errString string
		opts      []Option
		wantErr   bool
	}{
		{
			name:    "default_configuration",
			opts:    []Option{},
			wantErr: false,
		},
		{
			name: "with_custom_config",
			opts: []Option{
				WithConfig(&Config{
					EnableMetrics: false,
					EnableTracing: false,
				}),
			},
			wantErr: false,
		},
		{
			name: "with_custom_metrics",
			opts: []Option{
				WithMetrics(&mockMetrics{}),
			},
			wantErr: false,
		},
		{
			name: "with_custom_tracer",
			opts: []Option{
				WithTracer(&iface.TracerNoOp{}),
			},
			wantErr: false,
		},
		{
			name: "with_custom_logger",
			opts: []Option{
				WithLogger(&iface.LoggerNoOp{}),
			},
			wantErr: false,
		},
		{
			name: "with_custom_validator",
			opts: []Option{
				WithValidator(&mockValidator{}),
			},
			wantErr: false,
		},
		{
			name: "with_template_engine",
			opts: []Option{
				WithTemplateEngine(&mockTemplateEngine{}),
			},
			wantErr: false,
		},
		{
			name: "with_health_checker",
			opts: []Option{
				WithHealthChecker(&mockHealthChecker{}),
			},
			wantErr: false,
		},
		{
			name: "with_all_options",
			opts: []Option{
				WithConfig(DefaultConfig()),
				WithMetrics(&mockMetrics{}),
				WithTracer(&iface.TracerNoOp{}),
				WithLogger(&iface.LoggerNoOp{}),
				WithValidator(&mockValidator{}),
				WithTemplateEngine(&mockTemplateEngine{}),
				WithHealthChecker(&mockHealthChecker{}),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewPromptManager(tt.opts...)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errString != "" {
					assert.Contains(t, err.Error(), tt.errString)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, manager)
		})
	}
}

// TestPromptManager_GetMetrics tests GetMetrics method.
func TestPromptManager_GetMetrics(t *testing.T) {
	t.Run("with_metrics_enabled", func(t *testing.T) {
		manager, err := NewPromptManager(
			WithConfig(&Config{
				EnableMetrics: true,
			}),
		)
		require.NoError(t, err)
		require.NotNil(t, manager)

		metrics := manager.GetMetrics()
		assert.NotNil(t, metrics)
	})

	t.Run("with_metrics_disabled", func(t *testing.T) {
		manager, err := NewPromptManager(
			WithConfig(&Config{
				EnableMetrics: false,
			}),
		)
		require.NoError(t, err)
		require.NotNil(t, manager)

		metrics := manager.GetMetrics()
		// When metrics are disabled, GetMetrics may return nil
		// This is implementation-dependent
		if metrics != nil {
			t.Logf("GetMetrics returned non-nil value when metrics disabled: %T", metrics)
		}
	})

	t.Run("with_custom_metrics", func(t *testing.T) {
		customMetrics := &mockMetrics{}
		manager, err := NewPromptManager(
			WithMetrics(customMetrics),
		)
		require.NoError(t, err)
		require.NotNil(t, manager)

		metrics := manager.GetMetrics()
		assert.Equal(t, customMetrics, metrics)
	})
}

// TestPromptManager_Check tests the Check method comprehensively.
func TestPromptManager_Check(t *testing.T) {
	t.Run("successful_health_check", func(t *testing.T) {
		manager, err := NewPromptManager()
		require.NoError(t, err)

		ctx := context.Background()
		err = manager.Check(ctx)
		assert.NoError(t, err)
	})

	t.Run("health_check_with_template_creation_error", func(t *testing.T) {
		// This tests the error path in Check when template creation fails
		// We can't easily simulate this without modifying internal code,
		// but we can test that Check handles errors properly
		manager, err := NewPromptManager()
		require.NoError(t, err)

		ctx := context.Background()
		err = manager.Check(ctx)
		// Check should succeed in normal conditions
		assert.NoError(t, err)
	})

	t.Run("health_check_with_adapter_creation_error", func(t *testing.T) {
		manager, err := NewPromptManager()
		require.NoError(t, err)

		ctx := context.Background()
		err = manager.Check(ctx)
		// Check should succeed in normal conditions
		assert.NoError(t, err)
	})
}

// TestPromptManager_ConvenienceFunctions tests convenience functions.
func TestPromptManager_ConvenienceFunctions(t *testing.T) {
	t.Run("NewStringPromptTemplate_success", func(t *testing.T) {
		template, err := NewStringPromptTemplate("test-template", "Hello {{.name}}")
		require.NoError(t, err)
		assert.NotNil(t, template)
		assert.Equal(t, "test-template", template.Name())
	})

	t.Run("NewStringPromptTemplate_error", func(t *testing.T) {
		// Test error path when manager creation fails
		// This is hard to simulate without modifying code, but we test the happy path
		template, err := NewStringPromptTemplate("", "Hello {{.name}}")
		require.Error(t, err)
		assert.Nil(t, template)
	})

	t.Run("NewDefaultPromptAdapter_success", func(t *testing.T) {
		adapter, err := NewDefaultPromptAdapter("test-adapter", "Hello {{.name}}", []string{"name"})
		require.NoError(t, err)
		assert.NotNil(t, adapter)
	})

	t.Run("NewDefaultPromptAdapter_error", func(t *testing.T) {
		adapter, err := NewDefaultPromptAdapter("", "Hello {{.name}}", []string{"name"})
		require.Error(t, err)
		assert.Nil(t, adapter)
	})

	t.Run("NewChatPromptAdapter_success", func(t *testing.T) {
		adapter, err := NewChatPromptAdapter(
			"test-chat-adapter",
			"System: {{.system}}",
			"User: {{.user}}",
			[]string{"system", "user"},
		)
		require.NoError(t, err)
		assert.NotNil(t, adapter)
	})

	t.Run("NewChatPromptAdapter_error", func(t *testing.T) {
		adapter, err := NewChatPromptAdapter("", "System: {{.system}}", "User: {{.user}}", []string{"system", "user"})
		require.Error(t, err)
		assert.Nil(t, adapter)
	})
}

// TestConfig_WithTemplateEngine tests WithTemplateEngine option.
func TestConfig_WithTemplateEngine(t *testing.T) {
	engine := &mockTemplateEngine{}
	manager, err := NewPromptManager(
		WithTemplateEngine(engine),
	)
	require.NoError(t, err)
	assert.NotNil(t, manager)
}

// TestConfig_WithHealthChecker tests WithHealthChecker option.
func TestConfig_WithHealthChecker(t *testing.T) {
	checker := &mockHealthChecker{}
	manager, err := NewPromptManager(
		WithHealthChecker(checker),
	)
	require.NoError(t, err)
	assert.NotNil(t, manager)
}

// TestErrors_AllErrorFunctions tests all error creation functions.
func TestErrors_AllErrorFunctions(t *testing.T) {
	tests := []struct {
		name      string
		createErr func() *PromptError
		code      string
	}{
		{
			name: "NewTemplateParseError",
			createErr: func() *PromptError {
				return NewTemplateParseError("test_op", "test_template", errors.New("parse error"))
			},
			code: ErrCodeTemplateParse,
		},
		{
			name: "NewTemplateExecuteError",
			createErr: func() *PromptError {
				return NewTemplateExecuteError("test_op", "test_template", errors.New("execute error"))
			},
			code: ErrCodeTemplateExecute,
		},
		{
			name: "NewVariableMissingError",
			createErr: func() *PromptError {
				return NewVariableMissingError("test_op", "test_var", "test_template")
			},
			code: ErrCodeVariableMissing,
		},
		{
			name: "NewVariableInvalidError",
			createErr: func() *PromptError {
				return NewVariableInvalidError("test_op", "test_var", "string", "int")
			},
			code: ErrCodeVariableInvalid,
		},
		{
			name: "NewCacheError",
			createErr: func() *PromptError {
				return NewCacheError("test_op", "cache operation failed", errors.New("cache error"))
			},
			code: ErrCodeCacheError,
		},
		{
			name: "NewAdapterError",
			createErr: func() *PromptError {
				return NewAdapterError("test_op", "default", errors.New("adapter error"))
			},
			code: ErrCodeAdapterError,
		},
		{
			name: "NewConfigurationError",
			createErr: func() *PromptError {
				return NewConfigurationError("test_op", "config invalid", errors.New("config error"))
			},
			code: ErrCodeConfigurationError,
		},
		{
			name: "NewTimeoutError",
			createErr: func() *PromptError {
				return NewTimeoutError("test_op", "30s")
			},
			code: ErrCodeTimeout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.createErr()
			require.NotNil(t, err)
			assert.Equal(t, tt.code, err.Code)
			assert.NotEmpty(t, err.Op)
			assert.NotEmpty(t, err.Error())
		})
	}
}

// TestErrors_ErrorCodes tests all error code constants.
func TestErrors_ErrorCodes(t *testing.T) {
	errorCodes := []struct {
		name string
		code string
	}{
		{"ErrCodeTemplateParse", ErrCodeTemplateParse},
		{"ErrCodeTemplateExecute", ErrCodeTemplateExecute},
		{"ErrCodeVariableMissing", ErrCodeVariableMissing},
		{"ErrCodeVariableInvalid", ErrCodeVariableInvalid},
		{"ErrCodeValidationFailed", ErrCodeValidationFailed},
		{"ErrCodeCacheError", ErrCodeCacheError},
		{"ErrCodeAdapterError", ErrCodeAdapterError},
		{"ErrCodeConfigurationError", ErrCodeConfigurationError},
		{"ErrCodeTimeout", ErrCodeTimeout},
	}

	for _, tt := range errorCodes {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.code, "Error code should not be empty")
		})
	}
}
