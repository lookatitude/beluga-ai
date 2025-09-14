package providers

import (
	"context"
	"fmt"
	"strings"

	"github.com/lookatitude/beluga-ai/pkg/prompts"
	"github.com/lookatitude/beluga-ai/pkg/prompts/iface"
)

// MockTemplateEngine is a mock implementation of TemplateEngine for testing
type MockTemplateEngine struct {
	Templates map[string]string
}

// NewMockTemplateEngine creates a new mock template engine
func NewMockTemplateEngine() *MockTemplateEngine {
	return &MockTemplateEngine{
		Templates: make(map[string]string),
	}
}

// Parse parses a template string (mock implementation)
func (m *MockTemplateEngine) Parse(name, template string) (iface.ParsedTemplate, error) {
	m.Templates[name] = template
	return &MockParsedTemplate{template: template}, nil
}

// ExtractVariables extracts variables from a template string (mock implementation)
func (m *MockTemplateEngine) ExtractVariables(template string) ([]string, error) {
	// Simple mock extraction
	vars := []string{}
	if strings.Contains(template, "{{.name}}") {
		vars = append(vars, "name")
	}
	if strings.Contains(template, "{{.value}}") {
		vars = append(vars, "value")
	}
	return vars, nil
}

// MockParsedTemplate is a mock parsed template
type MockParsedTemplate struct {
	template string
}

// Execute executes the mock template
func (m *MockParsedTemplate) Execute(data interface{}) (string, error) {
	// Simple mock execution
	result := m.template
	if vars, ok := data.(map[string]interface{}); ok {
		for key, value := range vars {
			if str, ok := value.(string); ok {
				placeholder := fmt.Sprintf("{{.%s}}", key)
				result = strings.ReplaceAll(result, placeholder, str)
			}
		}
	}
	return result, nil
}

// MockVariableValidator is a mock implementation of VariableValidator for testing
type MockVariableValidator struct {
	ShouldFail bool
}

// NewMockVariableValidator creates a new mock variable validator
func NewMockVariableValidator() *MockVariableValidator {
	return &MockVariableValidator{}
}

// Validate validates variables (mock implementation)
func (m *MockVariableValidator) Validate(required []string, provided map[string]interface{}) error {
	if m.ShouldFail {
		return fmt.Errorf("mock validation failure")
	}

	for _, req := range required {
		if _, ok := provided[req]; !ok {
			return prompts.NewVariableMissingError("mock_validate", req, "test")
		}
	}
	return nil
}

// ValidateTypes validates variable types (mock implementation)
func (m *MockVariableValidator) ValidateTypes(variables map[string]interface{}) error {
	if m.ShouldFail {
		return fmt.Errorf("mock type validation failure")
	}
	return nil
}

// MockHealthChecker is a mock implementation of HealthChecker for testing
type MockHealthChecker struct {
	ShouldFail bool
}

// NewMockHealthChecker creates a new mock health checker
func NewMockHealthChecker() *MockHealthChecker {
	return &MockHealthChecker{}
}

// Check performs a mock health check
func (m *MockHealthChecker) Check(ctx context.Context) error {
	if m.ShouldFail {
		return fmt.Errorf("mock health check failure")
	}
	return nil
}
