package adapter

import (
	"beluga/pkg/interfaces"
	"beluga/pkg/orchestration"
	"context"
	"fmt"
	"sync"
	"time"
)

// AgentTask adapts an Agent to work with the Task-based workflow system.
// It wraps an Agent into a Task that can be scheduled by the orchestration system.
type AgentTask struct {
	Agent         interfaces.Agent
	ID            string
	DependsOn     []string
	InputData     interface{}
	OutputData    interface{}
	ResultHandler func(interface{}) error
	mutex         sync.RWMutex
	context       context.Context
	cancelFunc    context.CancelFunc
}

// NewAgentTask creates a new task that wraps an agent.
func NewAgentTask(agent interfaces.Agent, id string) *AgentTask {
	ctx, cancel := context.WithCancel(context.Background())
	return &AgentTask{
		Agent:      agent,
		ID:         id,
		DependsOn:  make([]string, 0),
		context:    ctx,
		cancelFunc: cancel,
	}
}

// WithDependencies specifies task dependencies.
func (at *AgentTask) WithDependencies(deps ...string) *AgentTask {
	at.DependsOn = append(at.DependsOn, deps...)
	return at
}

// WithInput sets the input data for the agent.
func (at *AgentTask) WithInput(input interface{}) *AgentTask {
	at.mutex.Lock()
	defer at.mutex.Unlock()
	at.InputData = input
	return at
}

// WithResultHandler sets a callback for processing the agent's output.
func (at *AgentTask) WithResultHandler(handler func(interface{}) error) *AgentTask {
	at.ResultHandler = handler
	return at
}

// GetOutput returns the output data from the agent's execution.
func (at *AgentTask) GetOutput() interface{} {
	at.mutex.RLock()
	defer at.mutex.RUnlock()
	return at.OutputData
}

// SetOutput sets the output data.
func (at *AgentTask) SetOutput(output interface{}) {
	at.mutex.Lock()
	defer at.mutex.Unlock()
	at.OutputData = output
}

// ToTask converts the AgentTask to an orchestration.Task that can be scheduled.
func (at *AgentTask) ToTask() *orchestration.Task {
	return &orchestration.Task{
		ID:       at.ID,
		DependsOn: at.DependsOn,
		Execute: func() error {
			return at.Execute()
		},
	}
}

// Execute runs the agent and captures its output.
func (at *AgentTask) Execute() error {
	// Execute the agent
	if err := at.Agent.Execute(); err != nil {
		return fmt.Errorf("agent execution failed: %w", err)
	}

	// Process results if a handler is specified
	if at.ResultHandler != nil {
		// Here we assume that the agent has stored its output somehow
		// and we can retrieve it. This is a simplified example.
		if err := at.ResultHandler(at.GetOutput()); err != nil {
			return fmt.Errorf("result handler failed: %w", err)
		}
	}

	return nil
}

// Cancel aborts the agent task execution.
func (at *AgentTask) Cancel() {
	at.cancelFunc()
}

// AgentWorkflow represents a workflow composed of multiple agents.
type AgentWorkflow struct {
	ID     string
	Tasks  []*AgentTask
	Scheduler *orchestration.Scheduler
}

// NewAgentWorkflow creates a new agent workflow.
func NewAgentWorkflow(id string) *AgentWorkflow {
	return &AgentWorkflow{
		ID:     id,
		Tasks:  make([]*AgentTask, 0),
		Scheduler: orchestration.NewScheduler(),
	}
}

// AddTask adds a task to the workflow.
func (aw *AgentWorkflow) AddTask(task *AgentTask) error {
	aw.Tasks = append(aw.Tasks, task)
	return aw.Scheduler.AddTask(task.ToTask())
}

// Execute runs the workflow by executing all tasks in the correct order.
func (aw *AgentWorkflow) Execute() error {
	return aw.Scheduler.Run()
}

// ExecuteSequential runs the workflow in a strictly sequential manner.
func (aw *AgentWorkflow) ExecuteSequential() error {
	return aw.Scheduler.ExecuteSequential()
}

// ExecuteParallel runs all tasks in parallel without considering dependencies.
func (aw *AgentWorkflow) ExecuteParallel() error {
	return aw.Scheduler.ExecuteAutonomous()
}

// AgentMessagingAdapter connects agents to the messaging system.
type AgentMessagingAdapter struct {
	MessagingSystem  *orchestration.MessagingSystem
	AgentRegistry    map[string]interfaces.Agent
	MessageHandlers  map[string]func(orchestration.Message) error
	mutex            sync.RWMutex
	stopChan         chan struct{}
}

// NewAgentMessagingAdapter creates a new messaging adapter for agents.
func NewAgentMessagingAdapter(ms *orchestration.MessagingSystem) *AgentMessagingAdapter {
	return &AgentMessagingAdapter{
		MessagingSystem: ms,
		AgentRegistry:   make(map[string]interfaces.Agent),
		MessageHandlers: make(map[string]func(orchestration.Message) error),
		stopChan:        make(chan struct{}),
	}
}

// RegisterAgent adds an agent to the messaging adapter.
func (ama *AgentMessagingAdapter) RegisterAgent(name string, agent interfaces.Agent) {
	ama.mutex.Lock()
	defer ama.mutex.Unlock()
	ama.AgentRegistry[name] = agent
}

// RegisterMessageHandler adds a message handler for a specific agent.
func (ama *AgentMessagingAdapter) RegisterMessageHandler(agentName string, handler func(orchestration.Message) error) {
	ama.mutex.Lock()
	defer ama.mutex.Unlock()
	ama.MessageHandlers[agentName] = handler
}

// SendMessage sends a message from an agent to another component.
func (ama *AgentMessagingAdapter) SendMessage(sender string, receiver string, msgType string, payload map[string]interface{}) error {
	msg := orchestration.Message{
		ID:        fmt.Sprintf("msg-%d", time.Now().UnixNano()),
		Timestamp: time.Now(),
		Sender:    sender,
		Receiver:  receiver,
		Type:      msgType,
		Payload:   payload,
	}

	return ama.MessagingSystem.SendMessageWithRetry(msg, 3, time.Second)
}

// StartMessageProcessing begins processing incoming messages for agents.
func (ama *AgentMessagingAdapter) StartMessageProcessing() {
	go func() {
		for {
			select {
			case <-ama.stopChan:
				return
			default:
				msg, err := ama.MessagingSystem.ReceiveMessage()
				if err != nil {
					// No message available or error, continue polling
					continue
				}

				ama.mutex.RLock()
				handler, exists := ama.MessageHandlers[msg.Receiver]
				ama.mutex.RUnlock()

				if exists {
					go func(m orchestration.Message, h func(orchestration.Message) error) {
						if err := h(m); err != nil {
							fmt.Printf("Error handling message: %v\n", err)
						}
					}(msg, handler)
				} else {
					fmt.Printf("No handler registered for agent %s\n", msg.Receiver)
				}
			}
		}
	}()
}

// StopMessageProcessing halts the message processing loop.
func (ama *AgentMessagingAdapter) StopMessageProcessing() {
	close(ama.stopChan)
}