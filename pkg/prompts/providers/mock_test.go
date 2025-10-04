package providers

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/prompts"
)

func TestMockTemplateEngine_Parse(t *testing.T) {
	engine := NewMockTemplateEngine()

	name := "test_template"
	template := "Hello {{.name}}!"

	parsed, err := engine.Parse(name, template)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if parsed == nil {
		t.Error("Parse() returned nil")
	}

	// Check that template was stored
	if stored, ok := engine.Templates[name]; !ok || stored != template {
		t.Errorf("Template not stored correctly: got %v, want %v", stored, template)
	}
}

func TestMockTemplateEngine_ExtractVariables(t *testing.T) {
	tests := []struct {
		name     string
		template string
		expected []string
	}{
		{
			name:     "template with name variable",
			template: "Hello {{.name}}!",
			expected: []string{"name"},
		},
		{
			name:     "template with value variable",
			template: "Value: {{.value}}",
			expected: []string{"value"},
		},
		{
			name:     "template with both variables",
			template: "{{.name}}: {{.value}}",
			expected: []string{"name", "value"},
		},
		{
			name:     "template without variables",
			template: "Hello World!",
			expected: []string{},
		},
		{
			name:     "template with duplicate variables",
			template: "{{.name}} and {{.name}}",
			expected: []string{"name"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewMockTemplateEngine()
			vars, err := engine.ExtractVariables(tt.template)
			if err != nil {
				t.Fatalf("ExtractVariables() error = %v", err)
			}

			if len(vars) != len(tt.expected) {
				t.Errorf("ExtractVariables() len = %d, want %d", len(vars), len(tt.expected))
				return
			}

			for i, v := range vars {
				if v != tt.expected[i] {
					t.Errorf("ExtractVariables()[%d] = %v, want %v", i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestMockParsedTemplate_Execute(t *testing.T) {
	tests := []struct {
		name     string
		template string
		data     interface{}
		expected string
	}{
		{
			name:     "simple replacement",
			template: "Hello {{.name}}!",
			data: map[string]interface{}{
				"name": "Alice",
			},
			expected: "Hello Alice!",
		},
		{
			name:     "multiple replacements",
			template: "{{.greeting}} {{.name}}!",
			data: map[string]interface{}{
				"greeting": "Hi",
				"name":     "Bob",
			},
			expected: "Hi Bob!",
		},
		{
			name:     "no replacements",
			template: "Hello World!",
			data:     map[string]interface{}{},
			expected: "Hello World!",
		},
		{
			name:     "non-string data",
			template: "Hello {{.name}}!",
			data: map[string]interface{}{
				"name": 123, // int instead of string
			},
			expected: "Hello {{.name}}!", // Should not replace
		},
		{
			name:     "wrong data type",
			template: "Hello {{.name}}!",
			data:     "not a map",
			expected: "Hello {{.name}}!", // Should not replace
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed := &MockParsedTemplate{template: tt.template}
			result, err := parsed.Execute(tt.data)
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}

			if result != tt.expected {
				t.Errorf("Execute() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestMockVariableValidator_Validate(t *testing.T) {
	tests := []struct {
		name        string
		shouldFail  bool
		required    []string
		provided    map[string]interface{}
		expectError bool
	}{
		{
			name:       "successful validation",
			shouldFail: false,
			required:   []string{"name", "age"},
			provided: map[string]interface{}{
				"name": "Alice",
				"age":  "25",
			},
			expectError: false,
		},
		{
			name:       "missing required variable",
			shouldFail: false,
			required:   []string{"name", "age"},
			provided: map[string]interface{}{
				"name": "Alice",
			},
			expectError: true,
		},
		{
			name:       "forced failure",
			shouldFail: true,
			required:   []string{"name"},
			provided: map[string]interface{}{
				"name": "Alice",
			},
			expectError: true,
		},
		{
			name:        "no required variables",
			shouldFail:  false,
			required:    []string{},
			provided:    map[string]interface{}{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewMockVariableValidator()
			validator.ShouldFail = tt.shouldFail

			err := validator.Validate(tt.required, tt.provided)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Check error type when expecting specific error
			if tt.expectError && !tt.shouldFail && err != nil {
				var promptErr *prompts.PromptError
				if !errors.As(err, &promptErr) {
					t.Errorf("Expected PromptError, got %T", err)
				}
			}
		})
	}
}

func TestMockVariableValidator_ValidateTypes(t *testing.T) {
	tests := []struct {
		name        string
		shouldFail  bool
		variables   map[string]interface{}
		expectError bool
	}{
		{
			name:       "successful type validation",
			shouldFail: false,
			variables: map[string]interface{}{
				"name": "Alice",
				"age":  25,
			},
			expectError: false,
		},
		{
			name:       "forced failure",
			shouldFail: true,
			variables: map[string]interface{}{
				"name": "Alice",
			},
			expectError: true,
		},
		{
			name:        "empty variables",
			shouldFail:  false,
			variables:   map[string]interface{}{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewMockVariableValidator()
			validator.ShouldFail = tt.shouldFail

			err := validator.ValidateTypes(tt.variables)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestMockHealthChecker_Check(t *testing.T) {
	tests := []struct {
		name        string
		shouldFail  bool
		expectError bool
	}{
		{
			name:        "successful health check",
			shouldFail:  false,
			expectError: false,
		},
		{
			name:        "forced failure",
			shouldFail:  true,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := NewMockHealthChecker()
			checker.ShouldFail = tt.shouldFail

			err := checker.Check(context.Background())

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestMockHealthChecker_CheckCount(t *testing.T) {
	checker := NewMockHealthChecker()

	// Initial count should be 0
	if count := checker.GetCheckCount(); count != 0 {
		t.Errorf("Initial check count = %d, want 0", count)
	}

	// Check multiple times
	checker.Check(context.Background())
	checker.Check(context.Background())
	checker.Check(context.Background())

	if count := checker.GetCheckCount(); count != 3 {
		t.Errorf("Check count after 3 calls = %d, want 3", count)
	}

	// Reset count
	checker.ResetCheckCount()
	if count := checker.GetCheckCount(); count != 0 {
		t.Errorf("Check count after reset = %d, want 0", count)
	}
}

func TestThreadSafeMockTemplateEngine_Concurrency(t *testing.T) {
	engine := NewThreadSafeMockTemplateEngine()

	numGoroutines := 10
	numOperations := 100

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*numOperations)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				templateName := fmt.Sprintf("template_g%d_i%d", goroutineID, j)
				template := fmt.Sprintf("Hello {{.name}} from goroutine %d iteration %d!", goroutineID, j)

				// Parse template
				parsed, err := engine.Parse(templateName, template)
				if err != nil {
					errors <- err
					continue
				}

				// Extract variables
				vars, err := engine.ExtractVariables(template)
				if err != nil {
					errors <- err
					continue
				}

				// Execute template
				data := map[string]interface{}{
					"name": fmt.Sprintf("User%d", goroutineID),
				}
				result, err := parsed.Execute(data)
				if err != nil {
					errors <- err
					continue
				}

				// Verify result contains expected content
				expectedContent := fmt.Sprintf("Hello User%d from goroutine %d iteration %d!", goroutineID, goroutineID, j)
				if result != expectedContent {
					errors <- fmt.Errorf("unexpected result: got %q, want %q", result, expectedContent)
				}

				// Verify variables extraction
				if len(vars) != 1 || vars[0] != "name" {
					errors <- fmt.Errorf("unexpected variables: got %v, want [name]", vars)
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for any errors
	errorCount := 0
	for err := range errors {
		t.Errorf("Concurrent operation error: %v", err)
		errorCount++
	}

	if errorCount > 0 {
		t.Errorf("Total concurrent operation errors: %d", errorCount)
	}
}

func TestAdvancedMockVariableValidator_ValidationCalls(t *testing.T) {
	validator := NewAdvancedMockVariableValidator()

	// Test Validate calls
	required := []string{"name", "age"}
	provided := map[string]interface{}{
		"name": "Alice",
		"age":  "25",
	}

	err := validator.Validate(required, provided)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	err = validator.ValidateTypes(provided)
	if err != nil {
		t.Fatalf("ValidateTypes() error = %v", err)
	}

	// Check validation calls
	calls := validator.GetValidationCalls()
	if len(calls) != 2 {
		t.Errorf("Expected 2 validation calls, got %d", len(calls))
	}

	// Check first call (Validate)
	if calls[0].Method != "Validate" {
		t.Errorf("First call method = %s, want Validate", calls[0].Method)
	}
	if !reflect.DeepEqual(calls[0].Required, required) {
		t.Errorf("First call required = %v, want %v", calls[0].Required, required)
	}
	if !reflect.DeepEqual(calls[0].Provided, provided) {
		t.Errorf("First call provided = %v, want %v", calls[0].Provided, provided)
	}

	// Check second call (ValidateTypes)
	if calls[1].Method != "ValidateTypes" {
		t.Errorf("Second call method = %s, want ValidateTypes", calls[1].Method)
	}
	if !reflect.DeepEqual(calls[1].Variables, provided) {
		t.Errorf("Second call variables = %v, want %v", calls[1].Variables, provided)
	}

	// Reset calls
	validator.ResetValidationCalls()
	calls = validator.GetValidationCalls()
	if len(calls) != 0 {
		t.Errorf("Expected 0 calls after reset, got %d", len(calls))
	}
}

func TestAdvancedMockVariableValidator_ErrorScenarios(t *testing.T) {
	validator := NewAdvancedMockVariableValidator()

	// Test forced validation failure
	validator.ShouldFailValidation = true
	err := validator.Validate([]string{"name"}, map[string]interface{}{"name": "test"})
	if err == nil {
		t.Error("Expected validation error but got none")
	}

	// Test forced type check failure
	validator.ShouldFailValidation = false
	validator.ShouldFailTypeCheck = true
	err = validator.ValidateTypes(map[string]interface{}{"name": "test"})
	if err == nil {
		t.Error("Expected type validation error but got none")
	}

	// Check that calls were recorded
	calls := validator.GetValidationCalls()
	if len(calls) != 2 {
		t.Errorf("Expected 2 validation calls, got %d", len(calls))
	}
}

func TestMockHealthChecker_Concurrency(t *testing.T) {
	checker := NewMockHealthChecker()
	ctx := context.Background()

	numGoroutines := 10
	numChecks := 100

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*numChecks)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for j := 0; j < numChecks; j++ {
				err := checker.Check(ctx)
				if err != nil {
					errors <- err
				}
			}
		}()
	}

	wg.Wait()
	close(errors)

	// Check for any errors
	errorCount := 0
	for err := range errors {
		t.Errorf("Concurrent health check error: %v", err)
		errorCount++
	}

	if errorCount > 0 {
		t.Errorf("Total concurrent health check errors: %d", errorCount)
	}

	// Verify total check count
	expectedChecks := numGoroutines * numChecks
	if count := checker.GetCheckCount(); count != expectedChecks {
		t.Errorf("Total check count = %d, want %d", count, expectedChecks)
	}
}

func BenchmarkMockTemplateEngine_Parse(b *testing.B) {
	engine := NewMockTemplateEngine()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		name := fmt.Sprintf("template_%d", i)
		template := fmt.Sprintf("Hello {{.name%d}}!", i)
		_, err := engine.Parse(name, template)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMockVariableValidator_Validate(b *testing.B) {
	validator := NewMockVariableValidator()

	required := []string{"name", "age", "email"}
	provided := map[string]interface{}{
		"name":  "Alice",
		"age":   "25",
		"email": "alice@example.com",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := validator.Validate(required, provided)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMockHealthChecker_Check(b *testing.B) {
	checker := NewMockHealthChecker()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := checker.Check(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// TestMockIntegration tests the mocks working together
func TestMockIntegration(t *testing.T) {
	// Create all mock components
	templateEngine := NewMockTemplateEngine()
	variableValidator := NewMockVariableValidator()
	healthChecker := NewMockHealthChecker()

	ctx := context.Background()

	// Test template operations
	templateName := "integration_test"
	template := "Hello {{.name}}, welcome to {{.place}}!"

	parsed, err := templateEngine.Parse(templateName, template)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	vars, err := templateEngine.ExtractVariables(template)
	if err != nil {
		t.Fatalf("ExtractVariables() error = %v", err)
	}

	expectedVars := []string{"name", "place"}
	if len(vars) != len(expectedVars) {
		t.Errorf("Expected %d variables, got %d", len(expectedVars), len(vars))
	}

	// Test validation
	err = variableValidator.Validate(vars, map[string]interface{}{
		"name":  "Alice",
		"place": "Wonderland",
	})
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	// Test template execution
	result, err := parsed.Execute(map[string]interface{}{
		"name":  "Alice",
		"place": "Wonderland",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	expected := "Hello Alice, welcome to Wonderland!"
	if result != expected {
		t.Errorf("Execute() = %q, want %q", result, expected)
	}

	// Test health check
	err = healthChecker.Check(ctx)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	// Verify health check was called
	if count := healthChecker.GetCheckCount(); count != 1 {
		t.Errorf("Health check count = %d, want 1", count)
	}
}
