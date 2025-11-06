// Package prompts provides advanced test utilities and comprehensive mocks for testing prompt implementations.
// This file contains utilities designed to support both unit tests and integration tests.
package prompts

import (
	"context"
	"fmt"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/prompts/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// AdvancedMockTemplate provides a comprehensive mock implementation for testing
type AdvancedMockTemplate struct {
	mock.Mock

	// Configuration
	name        string
	templateStr string
	callCount   int
	mu          sync.RWMutex

	// Configurable behavior
	shouldError   bool
	errorToReturn error
	variables     []string
	formatResults []interface{}
	resultIndex   int
	simulateDelay time.Duration

	// Template-specific data
	inputVariables  []string
	validationRules map[string]string

	// Health check data
	healthState     string
	lastHealthCheck time.Time
}

// NewAdvancedMockTemplate creates a new advanced mock with configurable behavior
func NewAdvancedMockTemplate(name, templateStr string, options ...MockTemplateOption) *AdvancedMockTemplate {
	mock := &AdvancedMockTemplate{
		name:            name,
		templateStr:     templateStr,
		variables:       []string{},
		formatResults:   []interface{}{},
		inputVariables:  []string{"input"},
		validationRules: make(map[string]string),
		healthState:     "healthy",
	}

	// Apply options
	for _, opt := range options {
		opt(mock)
	}

	// Extract variables from template if none provided
	if len(mock.inputVariables) <= 1 {
		mock.inputVariables = extractVariablesFromTemplate(templateStr)
	}

	return mock
}

// MockTemplateOption defines functional options for mock configuration
type MockTemplateOption func(*AdvancedMockTemplate)

// WithMockError configures the mock to return errors
func WithMockError(shouldError bool, err error) MockTemplateOption {
	return func(t *AdvancedMockTemplate) {
		t.shouldError = shouldError
		t.errorToReturn = err
	}
}

// WithMockVariables sets the input variables for the template
func WithMockVariables(variables []string) MockTemplateOption {
	return func(t *AdvancedMockTemplate) {
		t.inputVariables = make([]string, len(variables))
		copy(t.inputVariables, variables)
	}
}

// WithMockResults sets predefined format results
func WithMockResults(results []interface{}) MockTemplateOption {
	return func(t *AdvancedMockTemplate) {
		t.formatResults = make([]interface{}, len(results))
		copy(t.formatResults, results)
	}
}

// WithMockDelay adds artificial delay to mock operations
func WithMockDelay(delay time.Duration) MockTemplateOption {
	return func(t *AdvancedMockTemplate) {
		t.simulateDelay = delay
	}
}

// WithValidationRules sets validation rules for template variables
func WithValidationRules(rules map[string]string) MockTemplateOption {
	return func(t *AdvancedMockTemplate) {
		t.validationRules = make(map[string]string)
		for k, v := range rules {
			t.validationRules[k] = v
		}
	}
}

// Mock implementation methods for PromptFormatter interface
func (t *AdvancedMockTemplate) Format(ctx context.Context, inputs map[string]interface{}) (interface{}, error) {
	t.mu.Lock()
	t.callCount++
	t.mu.Unlock()

	if t.simulateDelay > 0 {
		time.Sleep(t.simulateDelay)
	}

	if t.shouldError {
		return nil, t.errorToReturn
	}

	// Validate required variables
	for _, variable := range t.inputVariables {
		if _, exists := inputs[variable]; !exists {
			return nil, fmt.Errorf("required variable '%s' is missing", variable)
		}
	}

	// Return predefined result if available
	if len(t.formatResults) > t.resultIndex {
		result := t.formatResults[t.resultIndex]
		t.resultIndex = (t.resultIndex + 1) % len(t.formatResults)
		return result, nil
	}

	// Generate default formatted result
	result := fmt.Sprintf("Template '%s' formatted with %d variables", t.name, len(inputs))
	for key, value := range inputs {
		result += fmt.Sprintf(" [%s: %v]", key, value)
	}

	return result, nil
}

func (t *AdvancedMockTemplate) GetInputVariables() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	result := make([]string, len(t.inputVariables))
	copy(result, t.inputVariables)
	return result
}

// Mock implementation methods for Template interface
func (t *AdvancedMockTemplate) Name() string {
	return t.name
}

func (t *AdvancedMockTemplate) Validate() error {
	if t.shouldError {
		return t.errorToReturn
	}

	if t.templateStr == "" {
		return fmt.Errorf("template string cannot be empty")
	}

	// Validate template variables
	for _, variable := range t.inputVariables {
		if variable == "" {
			return fmt.Errorf("template variable name cannot be empty")
		}
	}

	return nil
}

// Additional helper methods for testing
func (t *AdvancedMockTemplate) GetCallCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.callCount
}

func (t *AdvancedMockTemplate) GetTemplate() string {
	return t.templateStr
}

func (t *AdvancedMockTemplate) CheckHealth() map[string]interface{} {
	t.lastHealthCheck = time.Now()
	return map[string]interface{}{
		"status":           t.healthState,
		"name":             t.name,
		"template_length":  len(t.templateStr),
		"variable_count":   len(t.inputVariables),
		"call_count":       t.callCount,
		"validation_rules": len(t.validationRules),
		"last_checked":     t.lastHealthCheck,
	}
}

// AdvancedMockPromptValue provides a mock implementation of PromptValue
type AdvancedMockPromptValue struct {
	stringValue  string
	messageValue []schema.Message
}

func NewAdvancedMockPromptValue(str string, messages []schema.Message) *AdvancedMockPromptValue {
	return &AdvancedMockPromptValue{
		stringValue:  str,
		messageValue: messages,
	}
}

func (p *AdvancedMockPromptValue) ToString() string {
	return p.stringValue
}

func (p *AdvancedMockPromptValue) ToMessages() []schema.Message {
	result := make([]schema.Message, len(p.messageValue))
	copy(result, p.messageValue)
	return result
}

// AdvancedMockTemplateManager provides a mock template manager
type AdvancedMockTemplateManager struct {
	templates map[string]*AdvancedMockTemplate
	mu        sync.RWMutex
}

func NewAdvancedMockTemplateManager() *AdvancedMockTemplateManager {
	return &AdvancedMockTemplateManager{
		templates: make(map[string]*AdvancedMockTemplate),
	}
}

func (m *AdvancedMockTemplateManager) CreateTemplate(name, templateStr string) (iface.Template, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	template := NewAdvancedMockTemplate(name, templateStr)
	m.templates[name] = template

	return template, nil
}

func (m *AdvancedMockTemplateManager) GetTemplate(name string) (iface.Template, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	template, exists := m.templates[name]
	return template, exists
}

func (m *AdvancedMockTemplateManager) ListTemplates() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.templates))
	for name := range m.templates {
		names = append(names, name)
	}
	return names
}

func (m *AdvancedMockTemplateManager) DeleteTemplate(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.templates, name)
	return nil
}

func (m *AdvancedMockTemplateManager) GetTemplateCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.templates)
}

// Test data creation helpers

// CreateTestTemplate creates a test template string
func CreateTestTemplate(name string, variableCount int) string {
	template := fmt.Sprintf("This is test template '%s'", name)

	for i := 0; i < variableCount; i++ {
		varName := fmt.Sprintf("var%d", i+1)
		template += fmt.Sprintf(" Variable %s: {{.%s}}", varName, varName)
	}

	return template
}

// CreateTestInputs creates test input variables for templates
func CreateTestInputs(variableNames []string) map[string]interface{} {
	inputs := make(map[string]interface{})

	for i, name := range variableNames {
		inputs[name] = fmt.Sprintf("test_value_%d", i+1)
	}

	// Add common variables
	inputs["input"] = "test input content"
	inputs["context"] = "test context information"
	inputs["user_name"] = "test_user"

	return inputs
}

// CreateTestPromptConfig creates a test prompt configuration
func CreateTestPromptConfig() iface.Config {
	return iface.Config{
		DefaultTemplateTimeout: iface.Duration(30 * time.Second),
		MaxTemplateSize:        1048576, // 1MB
		ValidateVariables:      true,
		StrictVariableCheck:    false,
		EnableTemplateCache:    true,
		CacheTTL:               iface.Duration(5 * time.Minute),
		MaxCacheSize:           100,
		EnableMetrics:          true,
		EnableTracing:          true,
		DefaultAdapterType:     "default",
	}
}

// CreateTestMessages creates test chat messages for prompt value testing
func CreateTestPromptMessages(count int) []schema.Message {
	messages := make([]schema.Message, count)

	for i := 0; i < count; i++ {
		if i%2 == 0 {
			messages[i] = schema.NewHumanMessage(fmt.Sprintf("Human message %d", i+1))
		} else {
			messages[i] = schema.NewAIMessage(fmt.Sprintf("AI message %d", i+1))
		}
	}

	return messages
}

// Helper functions

func extractVariablesFromTemplate(template string) []string {
	// Extract variables from template string using regex (similar to real implementation)
	variables := make(map[string]struct{})
	re := regexp.MustCompile(`{{\.([\w]+)}}`)
	matches := re.FindAllStringSubmatch(template, -1)
	for _, match := range matches {
		if len(match) > 1 {
			variables[match[1]] = struct{}{}
		}
	}

	// If no variables found and template is not empty, add default "input"
	if len(variables) == 0 && len(template) > 0 {
		variables["input"] = struct{}{}
	}

	varList := make([]string, 0, len(variables))
	for v := range variables {
		varList = append(varList, v)
	}

	return varList
}

// Assertion helpers

// AssertTemplateFormat validates template formatting results
func AssertTemplateFormat(t *testing.T, result interface{}, expectedPattern string) {
	assert.NotNil(t, result)
	if str, ok := result.(string); ok {
		assert.Contains(t, str, expectedPattern)
	}
}

// AssertPromptValue validates prompt value results
func AssertPromptValue(t *testing.T, value iface.PromptValue, expectedStringPattern string, expectedMessageCount int) {
	assert.NotNil(t, value)

	// Test string representation
	str := value.ToString()
	assert.NotEmpty(t, str)
	if expectedStringPattern != "" {
		assert.Contains(t, str, expectedStringPattern)
	}

	// Test message representation
	messages := value.ToMessages()
	if expectedMessageCount > 0 {
		assert.Len(t, messages, expectedMessageCount)

		for i, msg := range messages {
			assert.NotEmpty(t, msg.GetContent(), "Message %d should have content", i)
		}
	}
}

// AssertTemplateVariables validates template input variables
func AssertTemplateVariables(t *testing.T, variables []string, expectedCount int) {
	assert.GreaterOrEqual(t, len(variables), expectedCount)

	for i, variable := range variables {
		assert.NotEmpty(t, variable, "Variable %d should not be empty", i)
	}
}

// AssertTemplateHealth validates template health check results
func AssertTemplateHealth(t *testing.T, health map[string]interface{}, expectedStatus string) {
	assert.Contains(t, health, "status")
	assert.Equal(t, expectedStatus, health["status"])
	assert.Contains(t, health, "name")
	assert.Contains(t, health, "call_count")
}

// AssertErrorType validates error types and codes
func AssertErrorType(t *testing.T, err error, expectedCode string) {
	assert.Error(t, err)
	var promptErr *iface.PromptError
	if assert.ErrorAs(t, err, &promptErr) {
		assert.Equal(t, expectedCode, promptErr.Code)
	}
}

// Performance testing helpers

// ConcurrentTestRunner runs prompt tests concurrently for performance testing
type ConcurrentTestRunner struct {
	NumGoroutines int
	TestDuration  time.Duration
	testFunc      func() error
}

func NewConcurrentTestRunner(numGoroutines int, duration time.Duration, testFunc func() error) *ConcurrentTestRunner {
	return &ConcurrentTestRunner{
		NumGoroutines: numGoroutines,
		TestDuration:  duration,
		testFunc:      testFunc,
	}
}

func (r *ConcurrentTestRunner) Run() error {
	var wg sync.WaitGroup
	errChan := make(chan error, r.NumGoroutines)
	stopChan := make(chan struct{})

	// Start timer
	timer := time.AfterFunc(r.TestDuration, func() {
		close(stopChan)
	})
	defer timer.Stop()

	// Start worker goroutines
	for i := 0; i < r.NumGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stopChan:
					return
				default:
					if err := r.testFunc(); err != nil {
						errChan <- err
						return
					}
				}
			}
		}()
	}

	// Wait for completion
	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

// RunLoadTest executes a load test scenario on templates
func RunLoadTest(t *testing.T, template *AdvancedMockTemplate, numOperations int, concurrency int) {
	var wg sync.WaitGroup
	errChan := make(chan error, numOperations)

	semaphore := make(chan struct{}, concurrency)
	testInputs := CreateTestInputs(template.GetInputVariables())

	for i := 0; i < numOperations; i++ {
		wg.Add(1)
		go func(opID int) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			ctx := context.Background()

			// Add operation-specific input
			operationInputs := make(map[string]interface{})
			for k, v := range testInputs {
				operationInputs[k] = fmt.Sprintf("%v_%d", v, opID)
			}

			_, err := template.Format(ctx, operationInputs)
			if err != nil {
				errChan <- err
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	// Verify no errors occurred
	for err := range errChan {
		assert.NoError(t, err)
	}

	// Verify expected call count
	assert.Equal(t, numOperations, template.GetCallCount())
}

// Integration test helpers

// IntegrationTestHelper provides utilities for integration testing
type IntegrationTestHelper struct {
	templates       map[string]*AdvancedMockTemplate
	templateManager *AdvancedMockTemplateManager
	promptValues    map[string]*AdvancedMockPromptValue
}

func NewIntegrationTestHelper() *IntegrationTestHelper {
	return &IntegrationTestHelper{
		templates:       make(map[string]*AdvancedMockTemplate),
		templateManager: NewAdvancedMockTemplateManager(),
		promptValues:    make(map[string]*AdvancedMockPromptValue),
	}
}

func (h *IntegrationTestHelper) AddTemplate(name string, template *AdvancedMockTemplate) {
	h.templates[name] = template
}

func (h *IntegrationTestHelper) AddPromptValue(name string, value *AdvancedMockPromptValue) {
	h.promptValues[name] = value
}

func (h *IntegrationTestHelper) GetTemplate(name string) *AdvancedMockTemplate {
	return h.templates[name]
}

func (h *IntegrationTestHelper) GetTemplateManager() *AdvancedMockTemplateManager {
	return h.templateManager
}

func (h *IntegrationTestHelper) GetPromptValue(name string) *AdvancedMockPromptValue {
	return h.promptValues[name]
}

func (h *IntegrationTestHelper) Reset() {
	for _, template := range h.templates {
		template.callCount = 0
		template.resultIndex = 0
	}
	h.templateManager = NewAdvancedMockTemplateManager()
	h.promptValues = make(map[string]*AdvancedMockPromptValue)
}

// PromptScenarioRunner runs common prompt scenarios
type PromptScenarioRunner struct {
	template iface.Template
	manager  iface.TemplateManager
}

func NewPromptScenarioRunner(template iface.Template, manager iface.TemplateManager) *PromptScenarioRunner {
	return &PromptScenarioRunner{
		template: template,
		manager:  manager,
	}
}

func (r *PromptScenarioRunner) RunTemplateFormattingScenario(ctx context.Context, inputSets []map[string]interface{}) ([]interface{}, error) {
	results := make([]interface{}, len(inputSets))

	for i, inputs := range inputSets {
		result, err := r.template.Format(ctx, inputs)
		if err != nil {
			return nil, fmt.Errorf("formatting scenario %d failed: %w", i+1, err)
		}
		results[i] = result
	}

	return results, nil
}

func (r *PromptScenarioRunner) RunTemplateManagementScenario(ctx context.Context, templateNames []string, templateStrings []string) error {
	if len(templateNames) != len(templateStrings) {
		return fmt.Errorf("template names and strings must have same length")
	}

	// Create templates
	for i, name := range templateNames {
		_, err := r.manager.CreateTemplate(name, templateStrings[i])
		if err != nil {
			return fmt.Errorf("failed to create template %s: %w", name, err)
		}
	}

	// Verify templates exist
	for _, name := range templateNames {
		template, exists := r.manager.GetTemplate(name)
		if !exists {
			return fmt.Errorf("template %s should exist after creation", name)
		}
		if template == nil {
			return fmt.Errorf("template %s should not be nil", name)
		}
	}

	// Test listing
	listedTemplates := r.manager.ListTemplates()
	if len(listedTemplates) < len(templateNames) {
		return fmt.Errorf("should list at least %d templates, got %d", len(templateNames), len(listedTemplates))
	}

	// Test deletion
	if len(templateNames) > 0 {
		err := r.manager.DeleteTemplate(templateNames[0])
		if err != nil {
			return fmt.Errorf("failed to delete template %s: %w", templateNames[0], err)
		}

		// Verify deletion worked
		_, exists := r.manager.GetTemplate(templateNames[0])
		if exists {
			return fmt.Errorf("template %s should be deleted", templateNames[0])
		}
	}

	return nil
}

// BenchmarkHelper provides benchmarking utilities for prompts
type BenchmarkHelper struct {
	template iface.Template
	inputs   []map[string]interface{}
}

func NewBenchmarkHelper(template iface.Template, inputCount int) *BenchmarkHelper {
	inputs := make([]map[string]interface{}, inputCount)
	variables := template.GetInputVariables()

	for i := 0; i < inputCount; i++ {
		inputs[i] = CreateTestInputs(variables)
		// Make each input set unique
		inputs[i]["operation_id"] = fmt.Sprintf("benchmark_%d", i)
	}

	return &BenchmarkHelper{
		template: template,
		inputs:   inputs,
	}
}

func (b *BenchmarkHelper) BenchmarkFormatting(iterations int) (time.Duration, error) {
	ctx := context.Background()

	start := time.Now()
	for i := 0; i < iterations; i++ {
		inputs := b.inputs[i%len(b.inputs)]
		_, err := b.template.Format(ctx, inputs)
		if err != nil {
			return 0, err
		}
	}

	return time.Since(start), nil
}

func (b *BenchmarkHelper) BenchmarkValidation(iterations int) (time.Duration, error) {
	start := time.Now()
	for i := 0; i < iterations; i++ {
		err := b.template.Validate()
		if err != nil {
			return 0, err
		}
	}

	return time.Since(start), nil
}

// TemplateQualityTester provides utilities for testing template quality
type TemplateQualityTester struct {
	template iface.Template
}

func NewTemplateQualityTester(template iface.Template) *TemplateQualityTester {
	return &TemplateQualityTester{template: template}
}

func (q *TemplateQualityTester) TestConsistency(ctx context.Context, inputs map[string]interface{}, iterations int) (bool, error) {
	if iterations < 2 {
		return true, nil
	}

	// Generate results multiple times
	results := make([]interface{}, iterations)
	for i := 0; i < iterations; i++ {
		result, err := q.template.Format(ctx, inputs)
		if err != nil {
			return false, err
		}
		results[i] = result
	}

	// Check if all results are consistent
	firstResult := results[0]
	for i := 1; i < iterations; i++ {
		if fmt.Sprintf("%v", results[i]) != fmt.Sprintf("%v", firstResult) {
			return false, nil
		}
	}

	return true, nil
}

func (q *TemplateQualityTester) TestVariableHandling(ctx context.Context, testCases []VariableTestCase) error {
	for i, testCase := range testCases {
		result, err := q.template.Format(ctx, testCase.Inputs)

		if testCase.ShouldError {
			if err == nil {
				return fmt.Errorf("test case %d should have failed but succeeded", i+1)
			}
		} else {
			if err != nil {
				return fmt.Errorf("test case %d should have succeeded but failed: %w", i+1, err)
			}
			if result == nil {
				return fmt.Errorf("test case %d produced nil result", i+1)
			}
		}
	}

	return nil
}

// VariableTestCase represents a test case for template variable handling
type VariableTestCase struct {
	Name        string
	Inputs      map[string]interface{}
	ShouldError bool
	Description string
}
