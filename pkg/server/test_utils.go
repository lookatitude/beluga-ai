// Package server provides advanced test utilities and comprehensive mocks for testing server implementations.
// This file contains utilities designed to support both unit tests and integration tests.
package server

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// AdvancedMockServer provides a comprehensive mock implementation for testing
type AdvancedMockServer struct {
	mock.Mock

	// Configuration
	name       string
	serverType string
	port       int
	callCount  int
	mu         sync.RWMutex

	// Configurable behavior
	shouldError   bool
	errorToReturn error
	responseDelay time.Duration
	simulateLoad  bool

	// Server state
	isRunning       bool
	connections     int
	requestHistory  []RequestRecord
	handlerRegistry map[string]http.HandlerFunc

	// Health check data
	healthState     string
	lastHealthCheck time.Time
}

// RequestRecord tracks server request history for testing
type RequestRecord struct {
	Method     string
	Path       string
	StatusCode int
	Duration   time.Duration
	Timestamp  time.Time
}

// NewAdvancedMockServer creates a new advanced mock server
func NewAdvancedMockServer(name, serverType string, port int, options ...MockServerOption) *AdvancedMockServer {
	mock := &AdvancedMockServer{
		name:            name,
		serverType:      serverType,
		port:            port,
		requestHistory:  make([]RequestRecord, 0),
		handlerRegistry: make(map[string]http.HandlerFunc),
		healthState:     "healthy",
	}

	// Apply options
	for _, opt := range options {
		opt(mock)
	}

	return mock
}

// MockServerOption defines functional options for mock configuration
type MockServerOption func(*AdvancedMockServer)

// WithMockError configures the mock to return errors
func WithMockError(shouldError bool, err error) MockServerOption {
	return func(s *AdvancedMockServer) {
		s.shouldError = shouldError
		s.errorToReturn = err
	}
}

// WithResponseDelay adds artificial delay to mock operations
func WithResponseDelay(delay time.Duration) MockServerOption {
	return func(s *AdvancedMockServer) {
		s.responseDelay = delay
	}
}

// WithLoadSimulation enables load simulation
func WithLoadSimulation(enabled bool) MockServerOption {
	return func(s *AdvancedMockServer) {
		s.simulateLoad = enabled
	}
}

// Mock server operations
func (s *AdvancedMockServer) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.callCount++

	if s.shouldError {
		return s.errorToReturn
	}

	if s.isRunning {
		return fmt.Errorf("server %s is already running", s.name)
	}

	s.isRunning = true
	return nil
}

func (s *AdvancedMockServer) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.callCount++

	if s.shouldError {
		return s.errorToReturn
	}

	if !s.isRunning {
		return fmt.Errorf("server %s is not running", s.name)
	}

	s.isRunning = false
	s.connections = 0
	return nil
}

func (s *AdvancedMockServer) HandleRequest(method, path string) (int, time.Duration, error) {
	s.mu.Lock()
	s.callCount++
	start := time.Now()
	s.mu.Unlock()

	if s.responseDelay > 0 {
		time.Sleep(s.responseDelay)
	}

	duration := time.Since(start)
	statusCode := 200

	if s.shouldError {
		statusCode = 500
	}

	// Record request
	s.mu.Lock()
	s.requestHistory = append(s.requestHistory, RequestRecord{
		Method:     method,
		Path:       path,
		StatusCode: statusCode,
		Duration:   duration,
		Timestamp:  time.Now(),
	})
	s.mu.Unlock()

	if s.shouldError {
		return statusCode, duration, s.errorToReturn
	}

	return statusCode, duration, nil
}

func (s *AdvancedMockServer) RegisterHandler(pattern string, handler http.HandlerFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlerRegistry[pattern] = handler
}

func (s *AdvancedMockServer) AddConnection() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.connections++
}

func (s *AdvancedMockServer) RemoveConnection() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.connections > 0 {
		s.connections--
	}
}

// Helper methods for testing
func (s *AdvancedMockServer) GetName() string {
	return s.name
}

func (s *AdvancedMockServer) GetServerType() string {
	return s.serverType
}

func (s *AdvancedMockServer) GetPort() int {
	return s.port
}

func (s *AdvancedMockServer) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isRunning
}

func (s *AdvancedMockServer) GetCallCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.callCount
}

func (s *AdvancedMockServer) GetConnectionCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.connections
}

func (s *AdvancedMockServer) GetRequestHistory() []RequestRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]RequestRecord, len(s.requestHistory))
	copy(result, s.requestHistory)
	return result
}

func (s *AdvancedMockServer) GetHandlerCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.handlerRegistry)
}

func (s *AdvancedMockServer) CheckHealth() map[string]interface{} {
	s.lastHealthCheck = time.Now()
	return map[string]interface{}{
		"status":        s.healthState,
		"name":          s.name,
		"type":          s.serverType,
		"port":          s.port,
		"running":       s.isRunning,
		"connections":   s.connections,
		"call_count":    s.callCount,
		"request_count": len(s.requestHistory),
		"handler_count": len(s.handlerRegistry),
		"last_checked":  s.lastHealthCheck,
	}
}

// Test data creation helpers

// CreateTestServerConfig creates a test server configuration
func CreateTestServerConfig() Config {
	return Config{
		Host:            "localhost",
		Port:            8080,
		ReadTimeout:     30 * time.Second,
		WriteTimeout:    30 * time.Second,
		IdleTimeout:     120 * time.Second,
		MaxHeaderBytes:  1048576, // 1MB
		EnableCORS:      true,
		CORSOrigins:     []string{"*"},
		EnableMetrics:   true,
		EnableTracing:   true,
		LogLevel:        "info",
		ShutdownTimeout: 30 * time.Second,
	}
}

// CreateTestRequests creates test HTTP requests for simulation
func CreateTestRequests(count int) []TestRequest {
	requests := make([]TestRequest, count)
	methods := []string{"GET", "POST", "PUT", "DELETE"}
	paths := []string{"/api/health", "/api/chat", "/api/embeddings", "/api/vector-search"}

	for i := 0; i < count; i++ {
		requests[i] = TestRequest{
			Method: methods[i%len(methods)],
			Path:   paths[i%len(paths)],
			Body:   fmt.Sprintf("Test request body %d", i+1),
		}
	}

	return requests
}

type TestRequest struct {
	Method string
	Path   string
	Body   string
}

// Assertion helpers

// AssertServerStatus validates server status
func AssertServerStatus(t *testing.T, server *AdvancedMockServer, expectedRunning bool) {
	assert.Equal(t, expectedRunning, server.IsRunning(), "Server running status should match expected")
}

// AssertRequestHandling validates request handling
func AssertRequestHandling(t *testing.T, statusCode int, duration time.Duration, err error, expectError bool) {
	if expectError {
		assert.Error(t, err)
		assert.GreaterOrEqual(t, statusCode, 400, "Error responses should have status >= 400")
	} else {
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, statusCode, 200, "Success responses should have status >= 200")
		assert.LessOrEqual(t, statusCode, 299, "Success responses should have status <= 299")
	}
	assert.Greater(t, duration, time.Duration(0), "Request should take some time")
}

// AssertServerHealth validates server health check results
func AssertServerHealth(t *testing.T, health map[string]interface{}, expectedStatus string) {
	assert.Contains(t, health, "status")
	assert.Equal(t, expectedStatus, health["status"])
	assert.Contains(t, health, "name")
	assert.Contains(t, health, "type")
	assert.Contains(t, health, "running")
	assert.Contains(t, health, "connections")
}

// Performance testing helpers

// RunLoadTest executes a load test scenario on server
func RunLoadTest(t *testing.T, server *AdvancedMockServer, numRequests int, concurrency int) {
	var wg sync.WaitGroup
	errChan := make(chan error, numRequests)

	semaphore := make(chan struct{}, concurrency)
	testRequests := CreateTestRequests(10) // Reuse test requests

	// Start server
	ctx := context.Background()
	err := server.Start(ctx)
	assert.NoError(t, err, "Server should start successfully")

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(reqID int) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			req := testRequests[reqID%len(testRequests)]
			statusCode, duration, err := server.HandleRequest(req.Method, req.Path)

			if err != nil && statusCode < 500 {
				errChan <- err
				return
			}

			// Validate response
			if statusCode < 200 || statusCode >= 600 {
				errChan <- fmt.Errorf("invalid status code: %d", statusCode)
				return
			}

			// Validate duration is reasonable
			if duration < 0 {
				errChan <- fmt.Errorf("negative duration: %v", duration)
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	// Verify no unexpected errors occurred
	for err := range errChan {
		t.Errorf("Load test error: %v", err)
	}

	// Verify request history
	history := server.GetRequestHistory()
	assert.Equal(t, numRequests, len(history), "Should record all requests")

	// Stop server
	err = server.Stop(ctx)
	assert.NoError(t, err, "Server should stop successfully")
}

// Integration test helpers

// IntegrationTestHelper provides utilities for integration testing
type IntegrationTestHelper struct {
	servers map[string]*AdvancedMockServer
}

func NewIntegrationTestHelper() *IntegrationTestHelper {
	return &IntegrationTestHelper{
		servers: make(map[string]*AdvancedMockServer),
	}
}

func (h *IntegrationTestHelper) AddServer(name string, server *AdvancedMockServer) {
	h.servers[name] = server
}

func (h *IntegrationTestHelper) GetServer(name string) *AdvancedMockServer {
	return h.servers[name]
}

func (h *IntegrationTestHelper) Reset() {
	for _, server := range h.servers {
		server.callCount = 0
		server.requestHistory = make([]RequestRecord, 0)
		server.isRunning = false
		server.connections = 0
	}
}

// ServerScenarioRunner runs common server scenarios
type ServerScenarioRunner struct {
	server *AdvancedMockServer
}

func NewServerScenarioRunner(server *AdvancedMockServer) *ServerScenarioRunner {
	return &ServerScenarioRunner{
		server: server,
	}
}

func (r *ServerScenarioRunner) RunAPIEndpointScenario(ctx context.Context, endpoints []APIEndpoint) error {
	// Start server
	err := r.server.Start(ctx)
	if err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}
	defer r.server.Stop(ctx)

	// Test each endpoint
	for i, endpoint := range endpoints {
		statusCode, duration, err := r.server.HandleRequest(endpoint.Method, endpoint.Path)

		if endpoint.ExpectError {
			if err == nil && statusCode < 400 {
				return fmt.Errorf("endpoint %d should have failed", i+1)
			}
		} else {
			if err != nil {
				return fmt.Errorf("endpoint %d failed: %w", i+1, err)
			}
			if statusCode < 200 || statusCode >= 400 {
				return fmt.Errorf("endpoint %d returned error status: %d", i+1, statusCode)
			}
		}

		if duration <= 0 {
			return fmt.Errorf("endpoint %d should have positive duration", i+1)
		}
	}

	return nil
}

type APIEndpoint struct {
	Method      string
	Path        string
	ExpectError bool
}

func (r *ServerScenarioRunner) RunConnectionManagementScenario(ctx context.Context, connectionCount int) error {
	// Start server
	err := r.server.Start(ctx)
	if err != nil {
		return err
	}
	defer r.server.Stop(ctx)

	// Add connections
	for i := 0; i < connectionCount; i++ {
		r.server.AddConnection()
	}

	// Verify connection count
	actualConnections := r.server.GetConnectionCount()
	if actualConnections != connectionCount {
		return fmt.Errorf("expected %d connections, got %d", connectionCount, actualConnections)
	}

	// Remove connections
	for i := 0; i < connectionCount; i++ {
		r.server.RemoveConnection()
	}

	// Verify connections are removed
	finalConnections := r.server.GetConnectionCount()
	if finalConnections != 0 {
		return fmt.Errorf("expected 0 connections after removal, got %d", finalConnections)
	}

	return nil
}

// BenchmarkHelper provides benchmarking utilities for servers
type BenchmarkHelper struct {
	server   *AdvancedMockServer
	requests []TestRequest
}

func NewBenchmarkHelper(server *AdvancedMockServer, requestCount int) *BenchmarkHelper {
	return &BenchmarkHelper{
		server:   server,
		requests: CreateTestRequests(requestCount),
	}
}

func (b *BenchmarkHelper) BenchmarkRequestHandling(iterations int) (time.Duration, error) {
	ctx := context.Background()

	// Start server
	err := b.server.Start(ctx)
	if err != nil {
		return 0, err
	}
	defer b.server.Stop(ctx)

	start := time.Now()
	for i := 0; i < iterations; i++ {
		req := b.requests[i%len(b.requests)]
		_, _, err := b.server.HandleRequest(req.Method, req.Path)
		if err != nil && b.server.shouldError {
			// Expected error
			continue
		} else if err != nil {
			return 0, err
		}
	}

	return time.Since(start), nil
}

func (b *BenchmarkHelper) BenchmarkServerLifecycle(iterations int) (time.Duration, error) {
	ctx := context.Background()

	start := time.Now()
	for i := 0; i < iterations; i++ {
		err := b.server.Start(ctx)
		if err != nil {
			return 0, err
		}

		err = b.server.Stop(ctx)
		if err != nil {
			return 0, err
		}
	}

	return time.Since(start), nil
}
