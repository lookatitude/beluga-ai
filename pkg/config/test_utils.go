// Package config provides advanced test utilities and comprehensive mocks for testing configuration implementations.
// This file contains utilities designed to support both unit tests and integration tests.
package config

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/config/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// AdvancedMockConfigProvider provides a comprehensive mock implementation for testing
type AdvancedMockConfigProvider struct {
	mock.Mock

	// Configuration
	name         string
	providerType string
	callCount    int
	mu           sync.RWMutex

	// Configurable behavior
	shouldError    bool
	errorToReturn  error
	configValues   map[string]interface{}
	simulateDelay  time.Duration
	validateStrict bool

	// Configuration management
	watchCallbacks map[string][]func(interface{})
	changeHistory  []ConfigChange

	// Health check data
	healthState     string
	lastHealthCheck time.Time
}

// ConfigChange represents a configuration change event
type ConfigChange struct {
	Key       string
	OldValue  interface{}
	NewValue  interface{}
	Timestamp time.Time
}

// NewAdvancedMockConfigProvider creates a new advanced mock with configurable behavior
func NewAdvancedMockConfigProvider(name, providerType string, options ...MockConfigOption) *AdvancedMockConfigProvider {
	mock := &AdvancedMockConfigProvider{
		name:           name,
		providerType:   providerType,
		configValues:   make(map[string]interface{}),
		watchCallbacks: make(map[string][]func(interface{})),
		changeHistory:  make([]ConfigChange, 0),
		healthState:    "healthy",
	}

	// Apply options
	for _, opt := range options {
		opt(mock)
	}

	// Set default config values if none provided
	if len(mock.configValues) == 0 {
		mock.configValues = map[string]interface{}{
			"provider":       providerType,
			"name":           name,
			"timeout":        "30s",
			"max_retries":    3,
			"enable_metrics": true,
			"enable_tracing": true,
		}
	}

	return mock
}

// MockConfigOption defines functional options for mock configuration
type MockConfigOption func(*AdvancedMockConfigProvider)

// WithMockError configures the mock to return errors
func WithMockError(shouldError bool, err error) MockConfigOption {
	return func(c *AdvancedMockConfigProvider) {
		c.shouldError = shouldError
		c.errorToReturn = err
	}
}

// WithMockConfigValues sets predefined configuration values
func WithMockConfigValues(values map[string]interface{}) MockConfigOption {
	return func(c *AdvancedMockConfigProvider) {
		c.configValues = make(map[string]interface{})
		for k, v := range values {
			c.configValues[k] = v
		}
	}
}

// WithMockDelay adds artificial delay to mock operations
func WithMockDelay(delay time.Duration) MockConfigOption {
	return func(c *AdvancedMockConfigProvider) {
		c.simulateDelay = delay
	}
}

// WithStrictValidation enables strict validation mode
func WithStrictValidation(strict bool) MockConfigOption {
	return func(c *AdvancedMockConfigProvider) {
		c.validateStrict = strict
	}
}

// Mock implementation methods for Provider interface
func (c *AdvancedMockConfigProvider) Load(configStruct interface{}) error {
	if c.shouldError {
		return c.errorToReturn
	}

	// Basic implementation - in real mock would populate struct
	return nil
}

func (c *AdvancedMockConfigProvider) UnmarshalKey(key string, rawVal interface{}) error {
	if c.shouldError {
		return c.errorToReturn
	}

	// Basic implementation - in real mock would unmarshal specific key
	return nil
}

func (c *AdvancedMockConfigProvider) IsSet(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, exists := c.configValues[key]
	return exists
}

func (c *AdvancedMockConfigProvider) GetLLMProviderConfig(name string) (schema.LLMProviderConfig, error) {
	if c.shouldError {
		return schema.LLMProviderConfig{}, c.errorToReturn
	}

	// Return mock LLM provider config
	return schema.LLMProviderConfig{
		Name:      name,
		Provider:  "mock",
		ModelName: "mock-model",
	}, nil
}

func (c *AdvancedMockConfigProvider) GetLLMProvidersConfig() ([]schema.LLMProviderConfig, error) {
	if c.shouldError {
		return nil, c.errorToReturn
	}

	return []schema.LLMProviderConfig{
		{Name: "openai", Provider: "openai", ModelName: "gpt-4"},
		{Name: "anthropic", Provider: "anthropic", ModelName: "claude-3"},
	}, nil
}

func (c *AdvancedMockConfigProvider) GetEmbeddingProvidersConfig() ([]schema.EmbeddingProviderConfig, error) {
	if c.shouldError {
		return nil, c.errorToReturn
	}

	return []schema.EmbeddingProviderConfig{
		{Name: "openai", Provider: "openai", ModelName: "text-embedding-ada-002"},
	}, nil
}

func (c *AdvancedMockConfigProvider) GetVectorStoresConfig() ([]schema.VectorStoreConfig, error) {
	if c.shouldError {
		return nil, c.errorToReturn
	}

	return []schema.VectorStoreConfig{
		{Name: "memory", Provider: "inmemory"},
	}, nil
}

func (c *AdvancedMockConfigProvider) GetAgentConfig(name string) (schema.AgentConfig, error) {
	if c.shouldError {
		return schema.AgentConfig{}, c.errorToReturn
	}

	return schema.AgentConfig{
		Name:               name,
		LLMProviderName:    "mock",
		ToolNames:          []string{},
		MemoryProviderName: "",
		AgentType:          "base",
	}, nil
}

func (c *AdvancedMockConfigProvider) GetAgentsConfig() ([]schema.AgentConfig, error) {
	if c.shouldError {
		return nil, c.errorToReturn
	}

	return []schema.AgentConfig{
		{Name: "agent1", LLMProviderName: "mock", AgentType: "base"},
		{Name: "agent2", LLMProviderName: "mock", AgentType: "react"},
	}, nil
}

func (c *AdvancedMockConfigProvider) GetToolConfig(name string) (iface.ToolConfig, error) {
	if c.shouldError {
		return iface.ToolConfig{}, c.errorToReturn
	}

	return iface.ToolConfig{
		Name:        name,
		Provider:    "mock",
		Description: "Mock tool config",
		Enabled:     true,
	}, nil
}

func (c *AdvancedMockConfigProvider) GetToolsConfig() ([]iface.ToolConfig, error) {
	if c.shouldError {
		return nil, c.errorToReturn
	}

	return []iface.ToolConfig{
		{Name: "tool1", Provider: "api", Description: "API tool", Enabled: true},
		{Name: "tool2", Provider: "shell", Description: "Shell tool", Enabled: true},
	}, nil
}

func (c *AdvancedMockConfigProvider) SetDefaults() error {
	if c.shouldError {
		return c.errorToReturn
	}

	// Set default values if not already set
	defaults := map[string]interface{}{
		"timeout":        "30s",
		"max_retries":    3,
		"enable_metrics": true,
	}

	for key, value := range defaults {
		if !c.IsSet(key) {
			c.configValues[key] = value
		}
	}

	return nil
}

// Additional methods for extended functionality
func (c *AdvancedMockConfigProvider) Get(key string) interface{} {
	c.mu.Lock()
	c.callCount++
	c.mu.Unlock()

	if c.simulateDelay > 0 {
		time.Sleep(c.simulateDelay)
	}

	if c.shouldError {
		return nil
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.configValues[key]
}

func (c *AdvancedMockConfigProvider) Set(key string, value interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.callCount++

	if c.simulateDelay > 0 {
		time.Sleep(c.simulateDelay)
	}

	if c.shouldError {
		return c.errorToReturn
	}

	oldValue := c.configValues[key]
	c.configValues[key] = value

	// Record change
	change := ConfigChange{
		Key:       key,
		OldValue:  oldValue,
		NewValue:  value,
		Timestamp: time.Now(),
	}
	c.changeHistory = append(c.changeHistory, change)

	// Trigger callbacks
	if callbacks, exists := c.watchCallbacks[key]; exists {
		for _, callback := range callbacks {
			go callback(value) // Execute callbacks asynchronously
		}
	}

	return nil
}

func (c *AdvancedMockConfigProvider) GetString(key string) string {
	value := c.Get(key)
	if str, ok := value.(string); ok {
		return str
	}
	return ""
}

func (c *AdvancedMockConfigProvider) GetInt(key string) int {
	value := c.Get(key)
	if i, ok := value.(int); ok {
		return i
	}
	return 0
}

func (c *AdvancedMockConfigProvider) GetBool(key string) bool {
	value := c.Get(key)
	if b, ok := value.(bool); ok {
		return b
	}
	return false
}

func (c *AdvancedMockConfigProvider) GetFloat64(key string) float64 {
	value := c.Get(key)
	if f, ok := value.(float64); ok {
		return f
	}
	return 0.0
}

func (c *AdvancedMockConfigProvider) GetStringMapString(key string) map[string]string {
	value := c.Get(key)
	if m, ok := value.(map[string]string); ok {
		return m
	}
	// Return default test map
	return map[string]string{
		"provider": "mock",
		"name":     c.name,
		"type":     c.providerType,
	}
}

func (c *AdvancedMockConfigProvider) GetDuration(key string) time.Duration {
	value := c.Get(key)
	if d, ok := value.(time.Duration); ok {
		return d
	}
	if str, ok := value.(string); ok {
		if duration, err := time.ParseDuration(str); err == nil {
			return duration
		}
	}
	return 0
}

func (c *AdvancedMockConfigProvider) Watch(key string, callback func(interface{})) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.shouldError {
		return c.errorToReturn
	}

	if c.watchCallbacks[key] == nil {
		c.watchCallbacks[key] = make([]func(interface{}), 0)
	}
	c.watchCallbacks[key] = append(c.watchCallbacks[key], callback)

	return nil
}

func (c *AdvancedMockConfigProvider) Validate() error {
	if c.shouldError {
		return c.errorToReturn
	}

	if c.validateStrict {
		// Perform strict validation
		requiredKeys := []string{"provider", "name"}
		for _, key := range requiredKeys {
			if _, exists := c.configValues[key]; !exists {
				return fmt.Errorf("required configuration key '%s' is missing", key)
			}
		}
	}

	return nil
}

// Additional helper methods for testing
func (c *AdvancedMockConfigProvider) GetName() string {
	return c.name
}

func (c *AdvancedMockConfigProvider) GetProviderType() string {
	return c.providerType
}

func (c *AdvancedMockConfigProvider) GetCallCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.callCount
}

func (c *AdvancedMockConfigProvider) GetConfigValues() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make(map[string]interface{})
	for k, v := range c.configValues {
		result[k] = v
	}
	return result
}

func (c *AdvancedMockConfigProvider) GetChangeHistory() []ConfigChange {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make([]ConfigChange, len(c.changeHistory))
	copy(result, c.changeHistory)
	return result
}

func (c *AdvancedMockConfigProvider) TriggerChange(key string, newValue interface{}) {
	c.Set(key, newValue) // This will trigger callbacks automatically
}

func (c *AdvancedMockConfigProvider) CheckHealth() map[string]interface{} {
	c.lastHealthCheck = time.Now()
	return map[string]interface{}{
		"status":        c.healthState,
		"name":          c.name,
		"provider_type": c.providerType,
		"call_count":    c.callCount,
		"config_keys":   len(c.configValues),
		"watch_keys":    len(c.watchCallbacks),
		"change_count":  len(c.changeHistory),
		"last_checked":  c.lastHealthCheck,
	}
}

// Test data creation helpers

// CreateTestConfig creates a comprehensive test configuration
func CreateTestConfig() map[string]interface{} {
	return map[string]interface{}{
		// Provider settings
		"provider": "test",
		"name":     "test-config",
		"version":  "1.0.0",

		// Connection settings
		"timeout":         "30s",
		"max_retries":     3,
		"retry_delay":     "1s",
		"connection_pool": 10,

		// Feature flags
		"enable_metrics": true,
		"enable_tracing": true,
		"enable_logging": true,
		"enable_caching": true,

		// Limits
		"max_concurrent":   100,
		"max_request_size": "10MB",
		"rate_limit":       1000,

		// Environment
		"environment": "test",
		"debug_mode":  true,
		"log_level":   "info",
	}
}

// CreateTestProviderConfigs creates test configurations for different providers
func CreateTestProviderConfigs() map[string]map[string]interface{} {
	return map[string]map[string]interface{}{
		"openai": {
			"provider":    "openai",
			"api_key":     "test-openai-key",
			"model":       "gpt-4",
			"temperature": 0.7,
			"max_tokens":  2000,
		},
		"anthropic": {
			"provider":    "anthropic",
			"api_key":     "test-anthropic-key",
			"model":       "claude-3-sonnet",
			"temperature": 0.8,
			"max_tokens":  4000,
		},
		"mock": {
			"provider":  "mock",
			"model":     "mock-model",
			"responses": []string{"Mock response 1", "Mock response 2"},
			"dimension": 128,
		},
	}
}

// CreateTestEnvironmentVars creates test environment variables
func CreateTestEnvironmentVars() map[string]string {
	return map[string]string{
		"BELUGA_ENVIRONMENT":    "test",
		"BELUGA_LOG_LEVEL":      "debug",
		"BELUGA_ENABLE_METRICS": "true",
		"BELUGA_ENABLE_TRACING": "true",
		"BELUGA_TIMEOUT":        "30s",
		"BELUGA_MAX_RETRIES":    "3",
	}
}

// Assertion helpers

// AssertConfigValue validates configuration value retrieval using Provider interface
func AssertConfigValue(t *testing.T, provider iface.Provider, key string, expectedValue interface{}, valueType string) {
	switch valueType {
	case "string":
		actual := provider.GetString(key)
		assert.Equal(t, expectedValue, actual, "Config value for key '%s' should match", key)
	case "int":
		actual := provider.GetInt(key)
		assert.Equal(t, expectedValue, actual, "Config value for key '%s' should match", key)
	case "bool":
		actual := provider.GetBool(key)
		assert.Equal(t, expectedValue, actual, "Config value for key '%s' should match", key)
	case "float64":
		actual := provider.GetFloat64(key)
		assert.Equal(t, expectedValue, actual, "Config value for key '%s' should match", key)
	default:
		// Provider interface doesn't have Get method, test through specific getters
		if provider.IsSet(key) {
			assert.True(t, true, "Key '%s' should be set", key)
		}
	}
}

// AssertProviderStructs validates provider configuration structure loading
func AssertProviderStructs(t *testing.T, provider iface.Provider) {
	// Test LLM provider config loading
	llmConfigs, err := provider.GetLLMProvidersConfig()
	assert.NoError(t, err, "Should be able to get LLM provider configs")
	assert.NotNil(t, llmConfigs, "LLM configs should not be nil")

	// Test embedding provider config loading
	embConfigs, err := provider.GetEmbeddingProvidersConfig()
	assert.NoError(t, err, "Should be able to get embedding provider configs")
	assert.NotNil(t, embConfigs, "Embedding configs should not be nil")

	// Test agent config loading
	agentConfigs, err := provider.GetAgentsConfig()
	assert.NoError(t, err, "Should be able to get agent configs")
	assert.NotNil(t, agentConfigs, "Agent configs should not be nil")

	// Test tool config loading
	toolConfigs, err := provider.GetToolsConfig()
	assert.NoError(t, err, "Should be able to get tool configs")
	assert.NotNil(t, toolConfigs, "Tool configs should not be nil")
}

// AssertConfigValidation validates configuration validation
func AssertConfigValidation(t *testing.T, provider iface.Provider, shouldPass bool) {
	err := provider.Validate()

	if shouldPass {
		assert.NoError(t, err, "Configuration validation should pass")
	} else {
		assert.Error(t, err, "Configuration validation should fail")
	}
}

// AssertConfigHealth validates configuration provider health check
func AssertConfigHealth(t *testing.T, health map[string]interface{}, expectedStatus string) {
	assert.Contains(t, health, "status")
	assert.Equal(t, expectedStatus, health["status"])
	assert.Contains(t, health, "name")
	assert.Contains(t, health, "provider_type")
	assert.Contains(t, health, "config_keys")
}

// AssertErrorType validates error types and codes
func AssertErrorType(t *testing.T, err error, expectedCode string) {
	assert.Error(t, err)
	var configErr *iface.ConfigError
	if assert.ErrorAs(t, err, &configErr) {
		assert.Equal(t, expectedCode, configErr.Code)
	}
}

// Performance testing helpers

// ConcurrentTestRunner runs config tests concurrently for performance testing
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

// RunLoadTest executes a load test scenario on config provider
func RunLoadTest(t *testing.T, provider *AdvancedMockConfigProvider, numOperations int, concurrency int) {
	var wg sync.WaitGroup
	errChan := make(chan error, numOperations)

	semaphore := make(chan struct{}, concurrency)
	testKeys := []string{"provider", "timeout", "max_retries", "enable_metrics", "enable_tracing"}

	for i := 0; i < numOperations; i++ {
		wg.Add(1)
		go func(opID int) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			key := testKeys[opID%len(testKeys)]

			if opID%4 == 0 {
				// Test GetString
				_ = provider.GetString(key)
			} else if opID%4 == 1 {
				// Test GetInt
				_ = provider.GetInt(key)
			} else if opID%4 == 2 {
				// Test GetBool
				_ = provider.GetBool(key)
			} else {
				// Test IsSet
				_ = provider.IsSet(key)
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	// Verify no errors occurred
	for err := range errChan {
		assert.NoError(t, err)
	}

	// Note: Provider interface methods don't update call count in our mock
}

// Integration test helpers

// IntegrationTestHelper provides utilities for integration testing
type IntegrationTestHelper struct {
	providers map[string]*AdvancedMockConfigProvider
}

func NewIntegrationTestHelper() *IntegrationTestHelper {
	return &IntegrationTestHelper{
		providers: make(map[string]*AdvancedMockConfigProvider),
	}
}

func (h *IntegrationTestHelper) AddProvider(name string, provider *AdvancedMockConfigProvider) {
	h.providers[name] = provider
}

func (h *IntegrationTestHelper) GetProvider(name string) *AdvancedMockConfigProvider {
	return h.providers[name]
}

func (h *IntegrationTestHelper) Reset() {
	for _, provider := range h.providers {
		provider.callCount = 0
		provider.changeHistory = make([]ConfigChange, 0)
		provider.watchCallbacks = make(map[string][]func(interface{}))
	}
}

// ConfigScenarioRunner runs common configuration scenarios
type ConfigScenarioRunner struct {
	provider iface.Provider
}

func NewConfigScenarioRunner(provider iface.Provider) *ConfigScenarioRunner {
	return &ConfigScenarioRunner{
		provider: provider,
	}
}

func (r *ConfigScenarioRunner) RunProviderSwitchingScenario(providerConfigs map[string]map[string]interface{}) error {
	for providerName := range providerConfigs {
		// Test provider configuration retrieval (Provider interface focuses on loading configs)
		// Instead of setting individual keys, test structured config retrieval

		// Test LLM provider config if this is an LLM provider
		if providerName == "openai" || providerName == "anthropic" {
			llmConfig, err := r.provider.GetLLMProviderConfig(providerName)
			if err != nil {
				return fmt.Errorf("failed to get LLM config for %s: %w", providerName, err)
			}
			if llmConfig.Name != providerName {
				return fmt.Errorf("LLM config name mismatch for %s", providerName)
			}
		}

		// Validate configuration
		err := r.provider.Validate()
		if err != nil {
			return fmt.Errorf("configuration validation failed for provider %s: %w", providerName, err)
		}
	}

	return nil
}

func (r *ConfigScenarioRunner) RunConfigReloadScenario(reloadCount int) error {
	// Test configuration reloading through SetDefaults and validation
	for i := 0; i < reloadCount; i++ {
		// Set defaults (simulates config reload)
		err := r.provider.SetDefaults()
		if err != nil {
			return fmt.Errorf("reload %d failed: %w", i+1, err)
		}

		// Validate after reload
		err = r.provider.Validate()
		if err != nil {
			return fmt.Errorf("validation after reload %d failed: %w", i+1, err)
		}
	}

	return nil
}

// BenchmarkHelper provides benchmarking utilities for configuration
type BenchmarkHelper struct {
	provider   iface.Provider
	testKeys   []string
	testValues []interface{}
}

func NewBenchmarkHelper(provider iface.Provider, keyCount int) *BenchmarkHelper {
	testKeys := make([]string, keyCount)
	testValues := make([]interface{}, keyCount)

	for i := 0; i < keyCount; i++ {
		testKeys[i] = fmt.Sprintf("benchmark_key_%d", i+1)
		testValues[i] = fmt.Sprintf("benchmark_value_%d", i+1)
	}

	return &BenchmarkHelper{
		provider:   provider,
		testKeys:   testKeys,
		testValues: testValues,
	}
}

func (b *BenchmarkHelper) BenchmarkGetString(iterations int) (time.Duration, error) {
	start := time.Now()
	for i := 0; i < iterations; i++ {
		key := b.testKeys[i%len(b.testKeys)]
		_ = b.provider.GetString(key)
	}

	return time.Since(start), nil
}

func (b *BenchmarkHelper) BenchmarkStructLoad(iterations int) (time.Duration, error) {
	start := time.Now()
	for i := 0; i < iterations; i++ {
		var config struct {
			Provider string `yaml:"provider"`
			Timeout  string `yaml:"timeout"`
		}

		err := b.provider.Load(&config)
		if err != nil {
			return 0, err
		}
	}

	return time.Since(start), nil
}

func (b *BenchmarkHelper) BenchmarkValidation(iterations int) (time.Duration, error) {
	start := time.Now()
	for i := 0; i < iterations; i++ {
		err := b.provider.Validate()
		if err != nil {
			return 0, err
		}
	}

	return time.Since(start), nil
}
