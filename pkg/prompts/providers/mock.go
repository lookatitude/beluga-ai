package providers

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/prompts"
	"github.com/lookatitude/beluga-ai/pkg/prompts/iface"
)

// MockTemplateEngine is a mock implementation of TemplateEngine for testing.
type MockTemplateEngine struct {
	Templates map[string]string
}

// NewMockTemplateEngine creates a new mock template engine.
func NewMockTemplateEngine() *MockTemplateEngine {
	return &MockTemplateEngine{
		Templates: make(map[string]string),
	}
}

// ThreadSafeMockTemplateEngine is a thread-safe version of MockTemplateEngine.
type ThreadSafeMockTemplateEngine struct {
	templates map[string]string
	mu        sync.RWMutex
}

// NewThreadSafeMockTemplateEngine creates a new thread-safe mock template engine.
func NewThreadSafeMockTemplateEngine() *ThreadSafeMockTemplateEngine {
	return &ThreadSafeMockTemplateEngine{
		templates: make(map[string]string),
	}
}

// Parse parses a template string (thread-safe mock implementation).
func (m *ThreadSafeMockTemplateEngine) Parse(name, template string) (iface.ParsedTemplate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.templates[name] = template
	return &MockParsedTemplate{template: template}, nil
}

// ExtractVariables extracts variables from a template string (thread-safe mock implementation).
func (m *ThreadSafeMockTemplateEngine) ExtractVariables(template string) ([]string, error) {
	// Extract all variables using regex
	vars := []string{}
	re := regexp.MustCompile(`\{\{\.(\w+)\}\}`)
	matches := re.FindAllStringSubmatch(template, -1)
	seen := make(map[string]bool)
	for _, match := range matches {
		if len(match) > 1 && !seen[match[1]] {
			vars = append(vars, match[1])
			seen[match[1]] = true
		}
	}
	return vars, nil
}

// Parse parses a template string (mock implementation).
func (m *MockTemplateEngine) Parse(name, template string) (iface.ParsedTemplate, error) {
	m.Templates[name] = template
	return &MockParsedTemplate{template: template}, nil
}

// ExtractVariables extracts variables from a template string (mock implementation).
func (m *MockTemplateEngine) ExtractVariables(template string) ([]string, error) {
	// Extract all variables using regex
	vars := []string{}
	re := regexp.MustCompile(`\{\{\.(\w+)\}\}`)
	matches := re.FindAllStringSubmatch(template, -1)
	seen := make(map[string]bool)
	for _, match := range matches {
		if len(match) > 1 && !seen[match[1]] {
			vars = append(vars, match[1])
			seen[match[1]] = true
		}
	}
	return vars, nil
}

// MockParsedTemplate is a mock parsed template.
type MockParsedTemplate struct {
	template string
}

// Execute executes the mock template.
func (m *MockParsedTemplate) Execute(data any) (string, error) {
	// Simple mock execution
	result := m.template
	if vars, ok := data.(map[string]any); ok {
		for key, value := range vars {
			if str, ok := value.(string); ok {
				placeholder := fmt.Sprintf("{{.%s}}", key)
				result = strings.ReplaceAll(result, placeholder, str)
			}
		}
	}
	return result, nil
}

// MockVariableValidator is a mock implementation of VariableValidator for testing.
type MockVariableValidator struct {
	ShouldFail bool
}

// NewMockVariableValidator creates a new mock variable validator.
func NewMockVariableValidator() *MockVariableValidator {
	return &MockVariableValidator{}
}

// Validate validates variables (mock implementation).
func (m *MockVariableValidator) Validate(required []string, provided map[string]any) error {
	if m.ShouldFail {
		return errors.New("mock validation failure")
	}

	for _, req := range required {
		if _, ok := provided[req]; !ok {
			return prompts.NewVariableMissingError("mock_validate", req, "test")
		}
	}
	return nil
}

// ValidateTypes validates variable types (mock implementation).
func (m *MockVariableValidator) ValidateTypes(variables map[string]any) error {
	if m.ShouldFail {
		return errors.New("mock type validation failure")
	}
	return nil
}

// MockHealthChecker is a mock implementation of HealthChecker for testing.
type MockHealthChecker struct {
	ShouldFail bool
	CheckCount int
	mu         sync.Mutex
}

// NewMockHealthChecker creates a new mock health checker.
func NewMockHealthChecker() *MockHealthChecker {
	return &MockHealthChecker{}
}

// Check performs a mock health check.
func (m *MockHealthChecker) Check(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CheckCount++
	if m.ShouldFail {
		return errors.New("mock health check failure")
	}
	return nil
}

// GetCheckCount returns the number of times Check was called.
func (m *MockHealthChecker) GetCheckCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.CheckCount
}

// ResetCheckCount resets the check count.
func (m *MockHealthChecker) ResetCheckCount() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CheckCount = 0
}

// AdvancedMockVariableValidator is a more advanced mock validator with configurable behavior.
type AdvancedMockVariableValidator struct {
	ValidationCalls      []ValidationCall
	mu                   sync.Mutex
	ShouldFailValidation bool
	ShouldFailTypeCheck  bool
}

// ValidationCall represents a call to Validate or ValidateTypes.
type ValidationCall struct {
	Provided  map[string]any
	Variables map[string]any
	Method    string
	Required  []string
	Timestamp int64
}

// NewAdvancedMockVariableValidator creates a new advanced mock validator.
func NewAdvancedMockVariableValidator() *AdvancedMockVariableValidator {
	return &AdvancedMockVariableValidator{
		ValidationCalls: make([]ValidationCall, 0),
	}
}

// Validate validates variables (advanced mock implementation).
func (m *AdvancedMockVariableValidator) Validate(required []string, provided map[string]any) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	call := ValidationCall{
		Method:    "Validate",
		Required:  make([]string, len(required)),
		Provided:  make(map[string]any),
		Timestamp: 0, // Would be time.Now().Unix() in real implementation
	}
	copy(call.Required, required)
	for k, v := range provided {
		call.Provided[k] = v
	}
	m.ValidationCalls = append(m.ValidationCalls, call)

	if m.ShouldFailValidation {
		return errors.New("mock validation failure")
	}

	for _, req := range required {
		if _, ok := provided[req]; !ok {
			return prompts.NewVariableMissingError("mock_validate", req, "test")
		}
	}
	return nil
}

// ValidateTypes validates variable types (advanced mock implementation).
func (m *AdvancedMockVariableValidator) ValidateTypes(variables map[string]any) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	call := ValidationCall{
		Method:    "ValidateTypes",
		Variables: make(map[string]any),
		Timestamp: 0, // Would be time.Now().Unix() in real implementation
	}
	for k, v := range variables {
		call.Variables[k] = v
	}
	m.ValidationCalls = append(m.ValidationCalls, call)

	if m.ShouldFailTypeCheck {
		return errors.New("mock type validation failure")
	}
	return nil
}

// GetValidationCalls returns all validation calls made.
func (m *AdvancedMockVariableValidator) GetValidationCalls() []ValidationCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	calls := make([]ValidationCall, len(m.ValidationCalls))
	copy(calls, m.ValidationCalls)
	return calls
}

// ResetValidationCalls clears the validation call history.
func (m *AdvancedMockVariableValidator) ResetValidationCalls() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ValidationCalls = make([]ValidationCall, 0)
}
