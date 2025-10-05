// Package core provides advanced test utilities and comprehensive mocks for testing core implementations.
// This file contains utilities designed to support both unit tests and integration tests.
// T006: Create test_utils.go with AdvancedMockRunnable and AdvancedMockContainer testing utilities
package core

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/core/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// AdvancedMockRunnable provides a comprehensive mock implementation for testing
type AdvancedMockRunnable struct {
	mock.Mock

	// Configuration
	name      string
	callCount int
	mu        sync.RWMutex

	// Configurable behavior
	shouldError      bool
	errorToReturn    error
	responses        []any
	responseIndex    int
	simulateDelay    bool
	delayDuration    time.Duration
	simulateFailures bool

	// Health check data
	healthState     string
	lastHealthCheck time.Time
}

// NewAdvancedMockRunnable creates a new advanced mock with configurable behavior
func NewAdvancedMockRunnable(name string, options ...MockRunnableOption) *AdvancedMockRunnable {
	mock := &AdvancedMockRunnable{
		name:            name,
		responses:       []any{"default mock response"},
		healthState:     "healthy",
		lastHealthCheck: time.Now(),
		delayDuration:   0,
	}

	// Apply options
	for _, opt := range options {
		opt(mock)
	}

	return mock
}

// MockRunnableOption defines functional options for mock configuration
type MockRunnableOption func(*AdvancedMockRunnable)

// WithMockResponses sets the responses to return
func WithMockResponses(responses []any) MockRunnableOption {
	return func(m *AdvancedMockRunnable) {
		m.responses = make([]any, len(responses))
		copy(m.responses, responses)
	}
}

// WithMockError configures the mock to return an error
func WithMockError(err error) MockRunnableOption {
	return func(m *AdvancedMockRunnable) {
		m.shouldError = true
		m.errorToReturn = err
	}
}

// WithMockDelay configures simulated execution delay
func WithMockDelay(delay time.Duration) MockRunnableOption {
	return func(m *AdvancedMockRunnable) {
		m.simulateDelay = true
		m.delayDuration = delay
	}
}

// WithMockFailureRate configures random failure simulation
func WithMockFailureRate(failureRate float32) MockRunnableOption {
	return func(m *AdvancedMockRunnable) {
		if failureRate > 0 {
			m.simulateFailures = true
		}
	}
}

// Mock implementation methods for Runnable interface
func (m *AdvancedMockRunnable) Invoke(ctx context.Context, input any, options ...iface.Option) (any, error) {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()

	// Simulate delay if configured
	if m.simulateDelay && m.delayDuration > 0 {
		select {
		case <-time.After(m.delayDuration):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// Return configured error if set
	if m.shouldError {
		return nil, m.errorToReturn
	}

	// Return next response in sequence
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.responses) == 0 {
		return "default response", nil
	}

	response := m.responses[m.responseIndex%len(m.responses)]
	m.responseIndex++
	return response, nil
}

func (m *AdvancedMockRunnable) Batch(ctx context.Context, inputs []any, options ...iface.Option) ([]any, error) {
	results := make([]any, len(inputs))

	for i, input := range inputs {
		result, err := m.Invoke(ctx, input, options...)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}

	return results, nil
}

func (m *AdvancedMockRunnable) Stream(ctx context.Context, input any, options ...iface.Option) (<-chan any, error) {
	ch := make(chan any, 1)

	go func() {
		defer close(ch)

		// Simulate streaming by sending response
		result, err := m.Invoke(ctx, input, options...)
		if err != nil {
			ch <- err
			return
		}

		ch <- result
	}()

	return ch, nil
}

// GetCallCount returns the number of times the mock was called
func (m *AdvancedMockRunnable) GetCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callCount
}

// AdvancedMockContainer provides a comprehensive mock implementation for testing Container operations
type AdvancedMockContainer struct {
	mock.Mock

	// Configuration
	registrations map[reflect.Type]any
	singletons    map[reflect.Type]any
	callCount     int
	mu            sync.RWMutex

	// Configurable behavior
	shouldFailResolve   bool
	shouldFailRegister  bool
	resolveDelay        time.Duration
	registerDelay       time.Duration
	simulateHealthIssue bool

	// Health monitoring
	healthState     string
	lastHealthCheck time.Time
}

// NewAdvancedMockContainer creates a new advanced container mock
func NewAdvancedMockContainer(options ...MockContainerOption) *AdvancedMockContainer {
	mock := &AdvancedMockContainer{
		registrations:   make(map[reflect.Type]any),
		singletons:      make(map[reflect.Type]any),
		healthState:     "healthy",
		lastHealthCheck: time.Now(),
	}

	for _, opt := range options {
		opt(mock)
	}

	return mock
}

// MockContainerOption defines functional options for container mock configuration
type MockContainerOption func(*AdvancedMockContainer)

// WithMockResolveFailure configures the mock to fail resolution
func WithMockResolveFailure(shouldFail bool) MockContainerOption {
	return func(m *AdvancedMockContainer) {
		m.shouldFailResolve = shouldFail
	}
}

// WithMockRegisterFailure configures the mock to fail registration
func WithMockRegisterFailure(shouldFail bool) MockContainerOption {
	return func(m *AdvancedMockContainer) {
		m.shouldFailRegister = shouldFail
	}
}

// WithMockResolveDelay configures simulated resolve delay
func WithMockResolveDelay(delay time.Duration) MockContainerOption {
	return func(m *AdvancedMockContainer) {
		m.resolveDelay = delay
	}
}

// Mock implementation methods for Container interface
func (m *AdvancedMockContainer) Register(factoryFunc interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.callCount++

	if m.registerDelay > 0 {
		time.Sleep(m.registerDelay)
	}

	if m.shouldFailRegister {
		return NewConfigurationError("mock_register_failure", fmt.Errorf("Mock configured to fail registration"))
	}

	// Store factory function by its return type
	funcType := reflect.TypeOf(factoryFunc)
	if funcType.Kind() != reflect.Func {
		return NewValidationError("invalid_factory", fmt.Errorf("Factory must be a function"))
	}

	if funcType.NumOut() == 0 {
		return NewValidationError("invalid_factory", fmt.Errorf("Factory must return at least one value"))
	}

	returnType := funcType.Out(0)
	m.registrations[returnType] = factoryFunc
	return nil
}

func (m *AdvancedMockContainer) Resolve(target interface{}) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.callCount++

	if m.resolveDelay > 0 {
		time.Sleep(m.resolveDelay)
	}

	if m.shouldFailResolve {
		return NewInternalError("mock_resolve_failure", fmt.Errorf("Mock configured to fail resolution"))
	}

	// Basic mock resolution - just set target to a default value
	targetValue := reflect.ValueOf(target)
	if targetValue.Kind() != reflect.Ptr {
		return NewValidationError("invalid_target", fmt.Errorf("Target must be a pointer"))
	}

	elem := targetValue.Elem()
	if elem.CanSet() {
		// Set a mock value based on type
		switch elem.Kind() {
		case reflect.String:
			elem.SetString("mock_value")
		case reflect.Int:
			elem.SetInt(42)
		case reflect.Interface:
			// For interfaces, create a basic mock
			if elem.Type() == reflect.TypeOf((*iface.Runnable)(nil)).Elem() {
				mockRunnable := NewAdvancedMockRunnable("mock_runnable")
				elem.Set(reflect.ValueOf(mockRunnable))
			}
		}
	}

	return nil
}

func (m *AdvancedMockContainer) MustResolve(target interface{}) {
	if err := m.Resolve(target); err != nil {
		panic(fmt.Sprintf("MustResolve failed: %v", err))
	}
}

func (m *AdvancedMockContainer) Has(typ reflect.Type) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, hasRegistration := m.registrations[typ]
	_, hasSingleton := m.singletons[typ]
	return hasRegistration || hasSingleton
}

func (m *AdvancedMockContainer) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.registrations = make(map[reflect.Type]any)
	m.singletons = make(map[reflect.Type]any)
}

func (m *AdvancedMockContainer) Singleton(instance interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	instanceType := reflect.TypeOf(instance)
	m.singletons[instanceType] = instance
}

func (m *AdvancedMockContainer) CheckHealth(ctx context.Context) error {
	m.mu.Lock()
	m.lastHealthCheck = time.Now()
	m.mu.Unlock()

	if m.simulateHealthIssue {
		return NewInternalError("mock_health_failure", fmt.Errorf("Mock configured to simulate health issue"))
	}

	return nil
}

// GetCallCount returns the number of operations performed on the mock container
func (m *AdvancedMockContainer) GetCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callCount
}

// GetRegistrationCount returns the number of registered factories
func (m *AdvancedMockContainer) GetRegistrationCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.registrations)
}

// ConcurrentTestRunner provides utilities for running concurrent test scenarios
type ConcurrentTestRunner struct {
	workers    int
	operations int
	timeout    time.Duration
}

// NewConcurrentTestRunner creates a new concurrent test runner
func NewConcurrentTestRunner(workers, operations int, timeout time.Duration) *ConcurrentTestRunner {
	return &ConcurrentTestRunner{
		workers:    workers,
		operations: operations,
		timeout:    timeout,
	}
}

// Run executes the test operation concurrently and reports results
func (r *ConcurrentTestRunner) Run(t *testing.T, operation func(workerID, operationID int) error) {
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()

	var wg sync.WaitGroup
	errorsChan := make(chan error, r.workers*r.operations)
	operationsPerWorker := r.operations / r.workers

	for i := 0; i < r.workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < operationsPerWorker; j++ {
				select {
				case <-ctx.Done():
					errorsChan <- fmt.Errorf("worker %d: context timeout", workerID)
					return
				default:
					if err := operation(workerID, j); err != nil {
						errorsChan <- fmt.Errorf("worker %d operation %d: %w", workerID, j, err)
					}
				}
			}
		}(i)
	}

	wg.Wait()
	close(errorsChan)

	// Check for errors
	var errors []error
	for err := range errorsChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		for _, err := range errors {
			t.Logf("Concurrent operation error: %v", err)
		}
		t.Fatalf("Concurrent operations failed with %d errors", len(errors))
	}
}

// PerformanceTestRunner provides utilities for performance testing
type PerformanceTestRunner struct {
	name         string
	target       time.Duration
	iterations   int
	warmupRounds int
}

// NewPerformanceTestRunner creates a new performance test runner
func NewPerformanceTestRunner(name string, target time.Duration, iterations int) *PerformanceTestRunner {
	return &PerformanceTestRunner{
		name:         name,
		target:       target,
		iterations:   iterations,
		warmupRounds: iterations / 10, // 10% warmup
	}
}

// Run executes the performance test and validates against target
func (r *PerformanceTestRunner) Run(t *testing.T, operation func() error) {
	// Warmup
	for i := 0; i < r.warmupRounds; i++ {
		operation()
	}

	// Actual test
	start := time.Now()
	for i := 0; i < r.iterations; i++ {
		if err := operation(); err != nil {
			t.Fatalf("Performance test operation failed: %v", err)
		}
	}
	elapsed := time.Since(start)
	avgTime := elapsed / time.Duration(r.iterations)

	t.Logf("%s: %d iterations, avg time: %v, target: %v", r.name, r.iterations, avgTime, r.target)

	if avgTime > r.target {
		t.Errorf("%s performance target not met: %v > %v", r.name, avgTime, r.target)
	}
}

// TestScenarioBuilder helps build complex test scenarios
type TestScenarioBuilder struct {
	name         string
	setup        func() error
	teardown     func() error
	operations   []func() error
	expectations []func(t *testing.T)
}

// NewTestScenarioBuilder creates a new test scenario builder
func NewTestScenarioBuilder(name string) *TestScenarioBuilder {
	return &TestScenarioBuilder{
		name:         name,
		operations:   make([]func() error, 0),
		expectations: make([]func(t *testing.T), 0),
	}
}

// WithSetup adds a setup function to the scenario
func (tsb *TestScenarioBuilder) WithSetup(setup func() error) *TestScenarioBuilder {
	tsb.setup = setup
	return tsb
}

// WithTeardown adds a teardown function to the scenario
func (tsb *TestScenarioBuilder) WithTeardown(teardown func() error) *TestScenarioBuilder {
	tsb.teardown = teardown
	return tsb
}

// AddOperation adds an operation to the scenario
func (tsb *TestScenarioBuilder) AddOperation(op func() error) *TestScenarioBuilder {
	tsb.operations = append(tsb.operations, op)
	return tsb
}

// AddExpectation adds an expectation to verify after operations
func (tsb *TestScenarioBuilder) AddExpectation(exp func(t *testing.T)) *TestScenarioBuilder {
	tsb.expectations = append(tsb.expectations, exp)
	return tsb
}

// Execute runs the complete test scenario
func (tsb *TestScenarioBuilder) Execute(t *testing.T) {
	// Setup
	if tsb.setup != nil {
		if err := tsb.setup(); err != nil {
			t.Fatalf("Scenario setup failed: %v", err)
		}
	}

	// Teardown
	defer func() {
		if tsb.teardown != nil {
			if err := tsb.teardown(); err != nil {
				t.Errorf("Scenario teardown failed: %v", err)
			}
		}
	}()

	// Execute operations
	for i, op := range tsb.operations {
		if err := op(); err != nil {
			t.Fatalf("Operation %d failed: %v", i, err)
		}
	}

	// Verify expectations
	for i, exp := range tsb.expectations {
		t.Run(fmt.Sprintf("expectation_%d", i), func(t *testing.T) {
			exp(t)
		})
	}
}

// MockContainerFactory provides factory methods for creating configured mock containers
type MockContainerFactory struct {
	defaultConfig MockContainerConfig
}

// MockContainerConfig defines configuration for mock containers
type MockContainerConfig struct {
	SimulateFailures bool
	ResolveDelay     time.Duration
	RegisterDelay    time.Duration
	HealthIssueRate  float32
	MaxRegistrations int
}

// NewMockContainerFactory creates a new factory for mock containers
func NewMockContainerFactory(config MockContainerConfig) *MockContainerFactory {
	return &MockContainerFactory{
		defaultConfig: config,
	}
}

// CreateMockContainer creates a mock container with the factory's default configuration
func (mcf *MockContainerFactory) CreateMockContainer() *AdvancedMockContainer {
	var options []MockContainerOption

	if mcf.defaultConfig.SimulateFailures {
		options = append(options, WithMockResolveFailure(true))
	}

	if mcf.defaultConfig.ResolveDelay > 0 {
		options = append(options, WithMockResolveDelay(mcf.defaultConfig.ResolveDelay))
	}

	return NewAdvancedMockContainer(options...)
}

// TestHelper provides utility functions for core package testing
type TestHelper struct {
	t *testing.T
}

// NewTestHelper creates a new test helper
func NewTestHelper(t *testing.T) *TestHelper {
	return &TestHelper{t: t}
}

// AssertRunnableCompliance verifies that a runnable properly implements the interface
func (th *TestHelper) AssertRunnableCompliance(runnable iface.Runnable, testInput any) {
	ctx := context.Background()

	// Test Invoke
	result, err := runnable.Invoke(ctx, testInput)
	assert.NoError(th.t, err, "Invoke should not return error")
	assert.NotNil(th.t, result, "Invoke should return non-nil result")

	// Test Batch
	inputs := []any{testInput}
	results, err := runnable.Batch(ctx, inputs)
	assert.NoError(th.t, err, "Batch should not return error")
	assert.Len(th.t, results, 1, "Batch should return one result for one input")

	// Test Stream
	ch, err := runnable.Stream(ctx, testInput)
	assert.NoError(th.t, err, "Stream should not return error")
	assert.NotNil(th.t, ch, "Stream should return non-nil channel")

	// Read from stream
	select {
	case result := <-ch:
		assert.NotNil(th.t, result, "Stream should produce result")
	case <-time.After(time.Second):
		th.t.Error("Stream should produce result within reasonable time")
	}
}

// AssertContainerCompliance verifies that a container properly implements the interface
func (th *TestHelper) AssertContainerCompliance(container Container) {
	// Test registration
	testFactory := func() string { return "test" }
	err := container.Register(testFactory)
	assert.NoError(th.t, err, "Register should not return error for valid factory")

	// Test resolution
	var result string
	err = container.Resolve(&result)
	assert.NoError(th.t, err, "Resolve should not return error for registered type")
	assert.Equal(th.t, "test", result, "Resolve should return factory result")

	// Test health check
	ctx := context.Background()
	err = container.CheckHealth(ctx)
	assert.NoError(th.t, err, "CheckHealth should not return error for healthy container")
}

// Global test utilities
var (
	DefaultMockRunnable  *AdvancedMockRunnable
	DefaultMockContainer *AdvancedMockContainer
	TestHelperOnce       sync.Once
)

// InitializeTestUtilities initializes global test utilities
func InitializeTestUtilities() {
	TestHelperOnce.Do(func() {
		DefaultMockRunnable = NewAdvancedMockRunnable("default")
		DefaultMockContainer = NewAdvancedMockContainer()
	})
}

// GetDefaultMockRunnable returns a default mock runnable for testing
func GetDefaultMockRunnable() *AdvancedMockRunnable {
	InitializeTestUtilities()
	return DefaultMockRunnable
}

// GetDefaultMockContainer returns a default mock container for testing
func GetDefaultMockContainer() *AdvancedMockContainer {
	InitializeTestUtilities()
	return DefaultMockContainer
}
