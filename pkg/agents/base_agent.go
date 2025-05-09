package agents

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
	"beluga/pkg/interfaces"
	"beluga/pkg/monitoring"
)

// AgentState represents the current state of an agent.
type AgentState string

const (
	StateInitializing AgentState = "initializing"
	StateReady        AgentState = "ready"
	StateRunning      AgentState = "running"
	StatePaused       AgentState = "paused"
	StateError        AgentState = "error"
	StateShutdown     AgentState = "shutdown"
)

// BaseAgent provides common functionality for all agents.
type BaseAgent struct {
	Name           string
	Config         map[string]interface{}
	State          AgentState
	CreatedAt      time.Time
	LastActiveTime time.Time
	Mutex          sync.RWMutex
	Context        context.Context
	CancelFunc     context.CancelFunc
	Logger         *monitoring.Logger
	ErrorCount     int
	MaxRetries     int
	RetryDelay     time.Duration
	EventHandlers  map[string][]func(interface{}) error
}

// NewBaseAgent creates a new BaseAgent with default values.
func NewBaseAgent(name string) *BaseAgent {
	ctx, cancel := context.WithCancel(context.Background())
	return &BaseAgent{
		Name:          name,
		State:         StateInitializing,
		CreatedAt:     time.Now(),
		Context:       ctx,
		CancelFunc:    cancel,
		Logger:        monitoring.NewLogger(name),
		MaxRetries:    3,
		RetryDelay:    time.Second * 2,
		EventHandlers: make(map[string][]func(interface{}) error),
	}
}

// Initialize sets up the agent with necessary configurations.
func (b *BaseAgent) Initialize(config map[string]interface{}) error {
	b.Mutex.Lock()
	defer b.Mutex.Unlock()

	if config == nil {
		return errors.New("config cannot be nil")
	}

	b.Config = config
	b.setState(StateReady)
	b.Logger.Info("Agent initialized with config: %v", b.Config)

	// Handle specific configuration options
	if maxRetries, ok := config["max_retries"].(int); ok {
		b.MaxRetries = maxRetries
	}
	if retryDelay, ok := config["retry_delay"].(int); ok {
		b.RetryDelay = time.Duration(retryDelay) * time.Second
	}

	return nil
}

// Execute performs the main task of the agent.
func (b *BaseAgent) Execute() error {
	b.Mutex.Lock()
	b.setState(StateRunning)
	b.LastActiveTime = time.Now()
	b.Mutex.Unlock()

	b.Logger.Info("Executing agent task")

	// Implement retry logic
	var err error
	for attempt := 0; attempt <= b.MaxRetries; attempt++ {
		if attempt > 0 {
			b.Logger.Warning("Retrying execution (attempt %d of %d)", attempt, b.MaxRetries)
			time.Sleep(b.RetryDelay)
		}

		err = b.doExecute()
		if err == nil {
			break
		}

		b.Mutex.Lock()
		b.ErrorCount++
		b.Mutex.Unlock()
		b.Logger.Error("Execution failed: %v", err)
	}

	b.Mutex.Lock()
	defer b.Mutex.Unlock()

	if err != nil {
		b.setState(StateError)
		return fmt.Errorf("agent %s execution failed after %d attempts: %w", b.Name, b.MaxRetries+1, err)
	}

	b.setState(StateReady)
	return nil
}

// doExecute is the internal execution method that should be overridden by subclasses.
func (b *BaseAgent) doExecute() error {
	// This method should be overridden by specific agent implementations
	return nil
}

// Shutdown gracefully stops the agent and cleans up resources.
func (b *BaseAgent) Shutdown() error {
	b.Mutex.Lock()
	defer b.Mutex.Unlock()

	if b.State == StateShutdown {
		return nil // Already shut down
	}

	b.Logger.Info("Shutting down")
	b.CancelFunc() // Cancel the context to signal all goroutines to stop
	b.setState(StateShutdown)

	// Perform resource cleanup here, such as closing files or connections.
	return nil
}

// GracefulShutdown ensures that ongoing tasks are completed or safely terminated.
func (b *BaseAgent) GracefulShutdown(timeout time.Duration) error {
	b.Logger.Info("Performing graceful shutdown (timeout: %v)", timeout)

	// Signal that we're shutting down but don't stop current tasks yet
	b.Mutex.Lock()
	if b.State == StateRunning {
		b.setState(StatePaused)
	}
	b.Mutex.Unlock()

	// Create a timeout context
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Wait for running tasks to complete or timeout
	shutdownComplete := make(chan struct{})
	go func() {
		// Add logic to wait for tasks to complete
		// This is a simplified example
		time.Sleep(100 * time.Millisecond) // Simulate waiting for tasks
		close(shutdownComplete)
	}()

	// Wait for completion or timeout
	select {
	case <-shutdownComplete:
		return b.Shutdown()
	case <-ctx.Done():
		b.Logger.Warning("Graceful shutdown timed out, forcing shutdown")
		return b.Shutdown()
	}
}

// GetState returns the current state of the agent.
func (b *BaseAgent) GetState() AgentState {
	b.Mutex.RLock()
	defer b.Mutex.RUnlock()
	return b.State
}

// setState updates the agent's state.
func (b *BaseAgent) setState(state AgentState) {
	b.State = state
	b.LastActiveTime = time.Now()
	b.Logger.Info("State changed to: %s", state)

	// Trigger state change event
	b.triggerEvent("state_change", state)
}

// RegisterEventHandler registers a handler function for a specific event type.
func (b *BaseAgent) RegisterEventHandler(eventType string, handler func(interface{}) error) {
	b.Mutex.Lock()
	defer b.Mutex.Unlock()

	if _, exists := b.EventHandlers[eventType]; !exists {
		b.EventHandlers[eventType] = make([]func(interface{}) error, 0)
	}

	b.EventHandlers[eventType] = append(b.EventHandlers[eventType], handler)
}

// triggerEvent calls all registered handlers for the given event type.
func (b *BaseAgent) triggerEvent(eventType string, payload interface{}) {
	b.Mutex.RLock()
	handlers := b.EventHandlers[eventType]
	b.Mutex.RUnlock()

	for _, handler := range handlers {
		if err := handler(payload); err != nil {
			b.Logger.Error("Event handler for %s failed: %v", eventType, err)
		}
	}
}

// CheckHealth returns the health status of the agent.
func (b *BaseAgent) CheckHealth() map[string]interface{} {
	b.Mutex.RLock()
	defer b.Mutex.RUnlock()

	return map[string]interface{}{
		"name":             b.Name,
		"state":            b.State,
		"up_time":          time.Since(b.CreatedAt).String(),
		"last_active_time": b.LastActiveTime,
		"error_count":      b.ErrorCount,
	}
}

// Ensure BaseAgent implements the Agent interface.
var _ interfaces.Agent = (*BaseAgent)(nil)

// DataFetcherAgent is responsible for retrieving data from various sources.
type DataFetcherAgent struct {
	*BaseAgent
	DataSource string
	DataFormat string
}

// NewDataFetcherAgent creates a new DataFetcherAgent.
func NewDataFetcherAgent(name string, dataSource string, dataFormat string) *DataFetcherAgent {
	return &DataFetcherAgent{
		BaseAgent:  NewBaseAgent(name),
		DataSource: dataSource,
		DataFormat: dataFormat,
	}
}

func (d *DataFetcherAgent) doExecute() error {
	d.Logger.Info("Fetching data from %s in format %s", d.DataSource, d.DataFormat)
	// Add data fetching logic here.
	return nil
}

// AnalyzerAgent processes and analyzes data to extract insights.
type AnalyzerAgent struct {
	*BaseAgent
	AnalysisType   string
	InputData      interface{}
	AnalysisResult interface{}
}

// NewAnalyzerAgent creates a new AnalyzerAgent.
func NewAnalyzerAgent(name string, analysisType string) *AnalyzerAgent {
	return &AnalyzerAgent{
		BaseAgent:    NewBaseAgent(name),
		AnalysisType: analysisType,
	}
}

func (a *AnalyzerAgent) SetInputData(data interface{}) {
	a.Mutex.Lock()
	defer a.Mutex.Unlock()
	a.InputData = data
}

func (a *AnalyzerAgent) GetAnalysisResult() interface{} {
	a.Mutex.RLock()
	defer a.Mutex.RUnlock()
	return a.AnalysisResult
}

func (a *AnalyzerAgent) doExecute() error {
	a.Mutex.RLock()
	inputData := a.InputData
	a.Mutex.RUnlock()

	if inputData == nil {
		return errors.New("no input data provided for analysis")
	}

	a.Logger.Info("Analyzing data using %s method", a.AnalysisType)
	// Add data analysis logic here.

	// Store analysis result
	a.Mutex.Lock()
	a.AnalysisResult = "Sample analysis result"
	a.Mutex.Unlock()

	return nil
}

// DecisionMakerAgent makes decisions based on analyzed data.
type DecisionMakerAgent struct {
	*BaseAgent
	AnalysisData  interface{}
	Decision      string
	DecisionRules map[string]interface{}
}

// NewDecisionMakerAgent creates a new DecisionMakerAgent.
func NewDecisionMakerAgent(name string) *DecisionMakerAgent {
	return &DecisionMakerAgent{
		BaseAgent:     NewBaseAgent(name),
		DecisionRules: make(map[string]interface{}),
	}
}

func (d *DecisionMakerAgent) SetAnalysisData(data interface{}) {
	d.Mutex.Lock()
	defer d.Mutex.Unlock()
	d.AnalysisData = data
}

func (d *DecisionMakerAgent) GetDecision() string {
	d.Mutex.RLock()
	defer d.Mutex.RUnlock()
	return d.Decision
}

func (d *DecisionMakerAgent) doExecute() error {
	d.Mutex.RLock()
	analysisData := d.AnalysisData
	d.Mutex.RUnlock()

	if analysisData == nil {
		return errors.New("no analysis data provided for decision making")
	}

	d.Logger.Info("Making decision based on analysis data")
	// Add decision-making logic here.

	// Store decision
	d.Mutex.Lock()
	d.Decision = "Sample decision"
	d.Mutex.Unlock()

	return nil
}

// ExecutorAgent executes actions or commands based on decisions.
type ExecutorAgent struct {
	*BaseAgent
	Action   string
	Target   string
	Params   map[string]interface{}
	Results  interface{}
}

// NewExecutorAgent creates a new ExecutorAgent.
func NewExecutorAgent(name string, action string, target string) *ExecutorAgent {
	return &ExecutorAgent{
		BaseAgent: NewBaseAgent(name),
		Action:    action,
		Target:    target,
		Params:    make(map[string]interface{}),
	}
}

func (e *ExecutorAgent) SetParams(params map[string]interface{}) {
	e.Mutex.Lock()
	defer e.Mutex.Unlock()
	e.Params = params
}

func (e *ExecutorAgent) GetResults() interface{} {
	e.Mutex.RLock()
	defer e.Mutex.RUnlock()
	return e.Results
}

func (e *ExecutorAgent) doExecute() error {
	e.Logger.Info("Executing action %s on target %s", e.Action, e.Target)

	// Add action execution logic here.

	// Store results
	e.Mutex.Lock()
	e.Results = "Sample execution results"
	e.Mutex.Unlock()

	return nil
}

// MonitorAgent monitors the performance and health of the system.
type MonitorAgent struct {
	*BaseAgent
	MonitorTargets []string
	MonitorResults map[string]interface{}
	Interval       time.Duration
	stopMonitoring chan struct{}
}

// NewMonitorAgent creates a new MonitorAgent.
func NewMonitorAgent(name string, interval time.Duration) *MonitorAgent {
	return &MonitorAgent{
		BaseAgent:      NewBaseAgent(name),
		MonitorTargets: make([]string, 0),
		MonitorResults: make(map[string]interface{}),
		Interval:       interval,
		stopMonitoring: make(chan struct{}),
	}
}

func (m *MonitorAgent) AddMonitorTarget(target string) {
	m.Mutex.Lock()
	defer m.Mutex.Unlock()
	m.MonitorTargets = append(m.MonitorTargets, target)
}

func (m *MonitorAgent) GetMonitorResults() map[string]interface{} {
	m.Mutex.RLock()
	defer m.Mutex.RUnlock()

	// Create a copy to avoid external modification
	results := make(map[string]interface{})
	for k, v := range m.MonitorResults {
		results[k] = v
	}

	return results
}

func (m *MonitorAgent) doExecute() error {
	m.Logger.Info("Starting continuous monitoring")

	// Start continuous monitoring in a goroutine
	go func() {
		ticker := time.NewTicker(m.Interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				m.collectMetrics()
			case <-m.stopMonitoring:
				m.Logger.Info("Stopping monitoring")
				return
			case <-m.Context.Done():
				m.Logger.Info("Context cancelled, stopping monitoring")
				return
			}
		}
	}()

	return nil
}

func (m *MonitorAgent) collectMetrics() {
	m.Mutex.Lock()
	defer m.Mutex.Unlock()

	for _, target := range m.MonitorTargets {
		// Simulate metric collection
		m.Logger.Debug("Collecting metrics for %s", target)
		m.MonitorResults[target] = map[string]interface{}{
			"status":     "healthy",
			"timestamp":  time.Now(),
			"cpu_usage":  30.5,
			"memory_use": 512,
		}
	}

	// Trigger metrics updated event
	m.triggerEvent("metrics_updated", m.MonitorResults)
}

func (m *MonitorAgent) Shutdown() error {
	close(m.stopMonitoring)
	return m.BaseAgent.Shutdown()
}