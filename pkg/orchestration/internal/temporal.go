// Package orchestrator provides interfaces and components for managing complex flows.
// This file specifically deals with Temporal integration.
package orchestration

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/orchestration/iface"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

// TemporalWorkflow implements the Workflow interface using Temporal.
type TemporalWorkflow struct {
	Client     client.Client
	TaskQueue  string
	WorkflowFn any // The actual Temporal workflow function (e.g., func(workflow.Context, InputType) (OutputType, error))
	// TODO: Add options like ID, timeouts, retry policies
}

// NewTemporalWorkflow creates a new TemporalWorkflow adapter.
func NewTemporalWorkflow(temporalClient client.Client, taskQueue string, workflowFn any) (*TemporalWorkflow, error) {
	if temporalClient == nil {
		return nil, fmt.Errorf("temporal client cannot be nil")
	}
	if taskQueue == "" {
		return nil, fmt.Errorf("temporal task queue cannot be empty")
	}
	if workflowFn == nil {
		return nil, fmt.Errorf("temporal workflow function cannot be nil")
	}
	// TODO: Validate workflowFn signature?
	return &TemporalWorkflow{
		Client:     temporalClient,
		TaskQueue:  taskQueue,
		WorkflowFn: workflowFn,
	}, nil
}

// Execute starts a new Temporal workflow execution.
func (tw *TemporalWorkflow) Execute(ctx context.Context, input any) (string, string, error) {
	options := client.StartWorkflowOptions{
		ID:        "beluga-workflow-" + fmt.Sprintf("%d", time.Now().UnixNano()), // Example ID
		TaskQueue: tw.TaskQueue,
		// TODO: Add WorkflowExecutionTimeout, WorkflowRunTimeout, WorkflowTaskTimeout from options
	}

	we, err := tw.Client.ExecuteWorkflow(ctx, options, tw.WorkflowFn, input)
	if err != nil {
		return "", "", fmt.Errorf("failed to execute Temporal workflow: %w", err)
	}
	return we.GetID(), we.GetRunID(), nil
}

// GetResult retrieves the result of a completed Temporal workflow.
func (tw *TemporalWorkflow) GetResult(ctx context.Context, workflowID string, runID string) (any, error) {
	we := tw.Client.GetWorkflow(ctx, workflowID, runID)
	var result any
	err := we.Get(ctx, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to get Temporal workflow result: %w", err)
	}
	return result, nil
}

// Signal sends a signal to a running Temporal workflow.
func (tw *TemporalWorkflow) Signal(ctx context.Context, workflowID string, runID string, signalName string, data any) error {
	err := tw.Client.SignalWorkflow(ctx, workflowID, runID, signalName, data)
	if err != nil {
		return fmt.Errorf("failed to signal Temporal workflow: %w", err)
	}
	return nil
}

// Query queries the state of a running Temporal workflow.
func (tw *TemporalWorkflow) Query(ctx context.Context, workflowID string, runID string, queryType string, args ...any) (any, error) {
	value, err := tw.Client.QueryWorkflow(ctx, workflowID, runID, queryType, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query Temporal workflow: %w", err)
	}
	var result any
	if err := value.Get(&result); err != nil {
		return nil, fmt.Errorf("failed to decode Temporal query result: %w", err)
	}
	return result, nil
}

// Cancel requests cancellation of a running Temporal workflow.
func (tw *TemporalWorkflow) Cancel(ctx context.Context, workflowID string, runID string) error {
	err := tw.Client.CancelWorkflow(ctx, workflowID, runID)
	if err != nil {
		return fmt.Errorf("failed to cancel Temporal workflow: %w", err)
	}
	return nil
}

// Terminate forcefully stops a running Temporal workflow.
func (tw *TemporalWorkflow) Terminate(ctx context.Context, workflowID string, runID string, reason string, details ...any) error {
	err := tw.Client.TerminateWorkflow(ctx, workflowID, runID, reason, details...)
	if err != nil {
		return fmt.Errorf("failed to terminate Temporal workflow: %w", err)
	}
	return nil
}

// Ensure TemporalWorkflow implements the interface.
var _ iface.Workflow = (*TemporalWorkflow)(nil)

// --- Activity Wrapper ---

// RunnableActivityWrapper wraps a Beluga core.Runnable to be used as a Temporal Activity.
type RunnableActivityWrapper struct {
	Runnable core.Runnable
}

// NewRunnableActivityWrapper creates a wrapper.
func NewRunnableActivityWrapper(runnable core.Runnable) *RunnableActivityWrapper {
	return &RunnableActivityWrapper{Runnable: runnable}
}

// Execute runs the underlying Beluga Runnable.
// It uses Temporal's context and handles input/output conversion.
func (w *RunnableActivityWrapper) Execute(ctx context.Context, input any) (any, error) {
	activity.GetLogger(ctx).Info("Executing Beluga Runnable activity", "runnableType", reflect.TypeOf(w.Runnable).String())

	// Note: Temporal uses a DataConverter for serialization. Input/Output types
	// need to be compatible with the configured converter (usually JSON).
	// We assume the input `any` can be directly passed to the Runnable's Invoke.
	// If Runnables expect specific structs, the workflow definition needs to ensure
	// the correct type is passed.

	// TODO: Extract core.Options from activity context/input if needed?
	output, err := w.Runnable.Invoke(ctx, input)
	if err != nil {
		activity.GetLogger(ctx).Error("Beluga Runnable activity failed", "error", err)
		return nil, err
	}

	activity.GetLogger(ctx).Info("Beluga Runnable activity completed")
	return output, nil
}

// RegisterRunnableActivities registers wrapped Beluga Runnables with a Temporal worker.
// It takes a map where keys are activity names and values are core.Runnables.
func RegisterRunnableActivities(w worker.Worker, runnables map[string]core.Runnable) {
	for name, runnable := range runnables {
		wrapper := NewRunnableActivityWrapper(runnable)
		// Register the Execute method of the wrapper
		w.RegisterActivityWithOptions(wrapper.Execute, activity.RegisterOptions{Name: name})
	}
}

// --- Example Workflow Definition (Illustrative) ---

// Define input/output structs for the example workflow
type SimpleChainWorkflowInput struct {
	InitialInput  map[string]any
	ActivityNames []string // Names of registered activities to run in sequence
}

// SimpleChainWorkflow executes a sequence of registered Beluga activities.
func SimpleChainWorkflow(ctx workflow.Context, input SimpleChainWorkflowInput) (any, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("SimpleChainWorkflow started", "activities", input.ActivityNames)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute, // Example timeout
		// TODO: Add HeartbeatTimeout, RetryPolicy etc.
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var currentOutput any = input.InitialInput
	var err error

	for i, activityName := range input.ActivityNames {
		logger.Info("Executing activity", "index", i, "name", activityName)
		// ExecuteActivity requires the result to be decoded into a variable.
		// We use a generic `any` here, assuming JSON compatibility.
		// For complex types, define specific structs.
		var stepOutput any
		err = workflow.ExecuteActivity(ctx, activityName, currentOutput).Get(ctx, &stepOutput)
		if err != nil {
			logger.Error("Activity failed", "name", activityName, "error", err)
			return nil, fmt.Errorf("activity 	%s	 failed: %w", activityName, err)
		}
		currentOutput = stepOutput
		logger.Info("Activity completed", "name", activityName)
	}

	logger.Info("SimpleChainWorkflow completed successfully.")
	return currentOutput, nil
}

// TODO:
// - Implement Graph orchestration using Temporal (e.g., executing activities concurrently based on dependencies).
// - Add more robust error handling and retry policies.
// - Integrate Beluga Memory with workflows (e.g., load/save state between activities or workflow runs).
// - Provide helper functions to easily create and run workers.
