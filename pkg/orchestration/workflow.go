package orchestration

import (
	"github.com/trustmaster/goflow"
	"log"
)

// TaskComponent represents a single task in the workflow.
type TaskComponent struct {
	goflow.Component
	In  <-chan string
	Out chan<- string
}

// Process executes the task logic.
func (t *TaskComponent) Process() {
	for input := range t.In {
		log.Printf("Processing task with input: %s", input)
		output := input + " processed"
		t.Out <- output
	}
}

// WorkflowGraph defines the workflow structure.
type WorkflowGraph struct {
	goflow.Graph
}

// NewWorkflowGraph initializes a new workflow graph.
func NewWorkflowGraph() *WorkflowGraph {
	graph := new(WorkflowGraph)
	graph.Graph = *goflow.NewGraph()

	// Add components
	task1 := &TaskComponent{}
	task2 := &TaskComponent{}
	graph.Add("Task1", task1)
	graph.Add("Task2", task2)

	// Connect components
	graph.Connect("Task1", "Out", "Task2", "In")

	// Set I/O channels
	graph.MapInPort("In", "Task1", "In")
	graph.MapOutPort("Out", "Task2", "Out")

	return graph
}

// WorkflowOrchestrator ties the messaging system, scheduler, and workflow graph together.
type WorkflowOrchestrator struct {
	MessagingSystem *MessagingSystem
	Scheduler       *Scheduler
	WorkflowGraph   *WorkflowGraph
}

// NewWorkflowOrchestrator initializes a new WorkflowOrchestrator with concrete dependencies.
func NewWorkflowOrchestrator(messagingSystem *MessagingSystem, scheduler *Scheduler) *WorkflowOrchestrator {
	return &WorkflowOrchestrator{
		MessagingSystem: messagingSystem,
		Scheduler:       scheduler,
		WorkflowGraph:   NewWorkflowGraph(),
	}
}

// Start begins the workflow orchestration process.
func (wo *WorkflowOrchestrator) Start() error {
	return wo.Scheduler.ExecuteSequential()
}