package workflow

import (
	"context"
	"errors"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/embedded"
	"go.temporal.io/sdk/mocks"

	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/orchestration/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRunnable is a mock implementation of core.Runnable for testing
type MockRunnable struct {
	mock.Mock
	name string
}

func (m *MockRunnable) Invoke(ctx context.Context, input any, opts ...core.Option) (any, error) {
	args := m.Called(ctx, input, opts)
	return args.Get(0), args.Error(1)
}

func (m *MockRunnable) Batch(ctx context.Context, inputs []any, opts ...core.Option) ([]any, error) {
	args := m.Called(ctx, inputs, opts)
	return args.Get(0).([]any), args.Error(1)
}

func (m *MockRunnable) Stream(ctx context.Context, input any, opts ...core.Option) (<-chan any, error) {
	args := m.Called(ctx, input, opts)
	return args.Get(0).(<-chan any), args.Error(1)
}

// MockTracer is a mock implementation of trace.Tracer for testing
type MockTracer struct {
	mock.Mock
	embedded.Tracer
}

func (m *MockTracer) Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	args := m.Called(ctx, spanName, opts)
	return args.Get(0).(context.Context), args.Get(1).(trace.Span)
}

// MockSpan is a mock implementation of trace.Span
type MockSpan struct {
	mock.Mock
	embedded.Span
}

func (m *MockSpan) End(options ...trace.SpanEndOption) {
	m.Called(options)
}

func (m *MockSpan) AddEvent(name string, options ...trace.EventOption) {
	m.Called(name, options)
}

func (m *MockSpan) AddLink(link trace.Link) {
	m.Called(link)
}

func (m *MockSpan) IsRecording() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockSpan) RecordError(err error, options ...trace.EventOption) {
	m.Called(err, options)
}

func (m *MockSpan) SpanContext() trace.SpanContext {
	args := m.Called()
	return args.Get(0).(trace.SpanContext)
}

func (m *MockSpan) SetStatus(code codes.Code, description string) {
	m.Called(code, description)
}

func (m *MockSpan) SetName(name string) {
	m.Called(name)
}

func (m *MockSpan) SetAttributes(kv ...attribute.KeyValue) {
	m.Called(kv)
}

func (m *MockSpan) TracerProvider() trace.TracerProvider {
	args := m.Called()
	return args.Get(0).(trace.TracerProvider)
}

// Use temporal SDK mocks instead of custom mocks

// Use temporal SDK mocks for WorkflowRun and Value as well

func TestNewTemporalWorkflow(t *testing.T) {
	config := iface.WorkflowConfig{
		Name:      "test-workflow",
		TaskQueue: "test-queue",
		Timeout:   300,
	}

	mockClient := &mocks.Client{}
	mockTracer := &MockTracer{}
	workflowFn := func(ctx context.Context, input string) (string, error) {
		return "result", nil
	}

	temporalWorkflow := NewTemporalWorkflow(config, mockClient, mockTracer, workflowFn)

	assert.NotNil(t, temporalWorkflow)
	assert.Equal(t, config, temporalWorkflow.config)
	assert.Equal(t, mockClient, temporalWorkflow.client)
	assert.Equal(t, mockTracer, temporalWorkflow.tracer)
	assert.NotNil(t, temporalWorkflow.workflowFn)
}

func TestTemporalWorkflow_Execute_Success(t *testing.T) {
	config := iface.WorkflowConfig{
		Name:      "test-workflow",
		TaskQueue: "test-queue",
		Timeout:   300,
	}

	mockClient := &mocks.Client{}
	mockTracer := &MockTracer{}
	mockSpan := &MockSpan{}
	mockWorkflowRun := &mocks.WorkflowRun{}

	// Setup mocks
	mockTracer.On("Start", mock.Anything, "temporal_workflow.execute", mock.Anything).Return(context.Background(), mockSpan)
	mockSpan.On("SetAttributes", mock.Anything).Return()
	mockSpan.On("End", mock.Anything).Return()
	mockClient.On("ExecuteWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(mockWorkflowRun, nil)
	mockWorkflowRun.On("GetID").Return("test-workflow-id")
	mockWorkflowRun.On("GetRunID").Return("test-run-id")

	workflowFn := func(ctx context.Context, input string) (string, error) {
		return "result", nil
	}

	temporalWorkflow := NewTemporalWorkflow(config, mockClient, mockTracer, workflowFn)

	workflowID, runID, err := temporalWorkflow.Execute(context.Background(), "test-input")

	assert.NoError(t, err)
	assert.Equal(t, "test-workflow-id", workflowID)
	assert.Equal(t, "test-run-id", runID)

	mockClient.AssertExpectations(t)
	mockTracer.AssertExpectations(t)
	mockSpan.AssertExpectations(t)
	mockWorkflowRun.AssertExpectations(t)
}

func TestTemporalWorkflow_Execute_Error(t *testing.T) {
	config := iface.WorkflowConfig{
		Name:      "test-workflow",
		TaskQueue: "test-queue",
		Timeout:   300,
	}

	mockClient := &mocks.Client{}
	mockTracer := &MockTracer{}
	mockSpan := &MockSpan{}

	// Setup mocks
	mockTracer.On("Start", mock.Anything, "temporal_workflow.execute", mock.Anything).Return(context.Background(), mockSpan)
	mockSpan.On("RecordError", mock.Anything, mock.Anything).Return()
	mockSpan.On("End", mock.Anything).Return()
	mockClient.On("ExecuteWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("temporal execution failed"))

	workflowFn := func(ctx context.Context, input string) (string, error) {
		return "result", nil
	}

	temporalWorkflow := NewTemporalWorkflow(config, mockClient, mockTracer, workflowFn)

	workflowID, runID, err := temporalWorkflow.Execute(context.Background(), "test-input")

	assert.Error(t, err)
	assert.Empty(t, workflowID)
	assert.Empty(t, runID)
	assert.Contains(t, err.Error(), "temporal execution failed")

	mockClient.AssertExpectations(t)
	mockTracer.AssertExpectations(t)
	mockSpan.AssertExpectations(t)
}

func TestTemporalWorkflow_GetResult_Success(t *testing.T) {
	config := iface.WorkflowConfig{
		Name:      "test-workflow",
		TaskQueue: "test-queue",
	}

	mockClient := &mocks.Client{}
	mockTracer := &MockTracer{}
	mockSpan := &MockSpan{}
	mockWorkflowRun := &mocks.WorkflowRun{}

	// Setup mocks
	mockTracer.On("Start", mock.Anything, "temporal_workflow.get_result", mock.Anything).Return(context.Background(), mockSpan)
	mockSpan.On("End", mock.Anything).Return()
	mockClient.On("GetWorkflow", mock.Anything, "workflow-id", "run-id").Return(mockWorkflowRun)
	mockWorkflowRun.On("Get", mock.Anything, mock.AnythingOfType("*interface {}")).Run(func(args mock.Arguments) {
		resultPtr := args.Get(1).(*interface{})
		*resultPtr = "workflow_result"
	}).Return(nil)

	temporalWorkflow := NewTemporalWorkflow(config, mockClient, mockTracer, nil)

	result, err := temporalWorkflow.GetResult(context.Background(), "workflow-id", "run-id")

	assert.NoError(t, err)
	assert.Equal(t, "workflow_result", result)

	mockClient.AssertExpectations(t)
	mockTracer.AssertExpectations(t)
	mockSpan.AssertExpectations(t)
	mockWorkflowRun.AssertExpectations(t)
}

func TestTemporalWorkflow_GetResult_Error(t *testing.T) {
	config := iface.WorkflowConfig{
		Name:      "test-workflow",
		TaskQueue: "test-queue",
	}

	mockClient := &mocks.Client{}
	mockTracer := &MockTracer{}
	mockSpan := &MockSpan{}
	mockWorkflowRun := &mocks.WorkflowRun{}

	// Setup mocks
	mockTracer.On("Start", mock.Anything, "temporal_workflow.get_result", mock.Anything).Return(context.Background(), mockSpan)
	mockSpan.On("RecordError", mock.Anything, mock.Anything).Return()
	mockSpan.On("End", mock.Anything).Return()
	mockClient.On("GetWorkflow", mock.Anything, "workflow-id", "run-id").Return(mockWorkflowRun)
	mockWorkflowRun.On("Get", mock.Anything, mock.AnythingOfType("*interface {}")).Return(errors.New("workflow not found"))

	temporalWorkflow := NewTemporalWorkflow(config, mockClient, mockTracer, nil)

	result, err := temporalWorkflow.GetResult(context.Background(), "workflow-id", "run-id")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "workflow not found")

	mockClient.AssertExpectations(t)
	mockTracer.AssertExpectations(t)
	mockSpan.AssertExpectations(t)
	mockWorkflowRun.AssertExpectations(t)
}

func TestTemporalWorkflow_Signal_Success(t *testing.T) {
	config := iface.WorkflowConfig{
		Name:      "test-workflow",
		TaskQueue: "test-queue",
	}

	mockClient := &mocks.Client{}
	mockTracer := &MockTracer{}
	mockSpan := &MockSpan{}

	// Setup mocks
	mockTracer.On("Start", mock.Anything, "temporal_workflow.signal", mock.Anything).Return(context.Background(), mockSpan)
	mockSpan.On("End", mock.Anything).Return()
	mockClient.On("SignalWorkflow", mock.Anything, "workflow-id", "run-id", "test-signal", "signal-data").Return(nil)

	temporalWorkflow := NewTemporalWorkflow(config, mockClient, mockTracer, nil)

	err := temporalWorkflow.Signal(context.Background(), "workflow-id", "run-id", "test-signal", "signal-data")

	assert.NoError(t, err)

	mockClient.AssertExpectations(t)
	mockTracer.AssertExpectations(t)
}

func TestTemporalWorkflow_Signal_Error(t *testing.T) {
	config := iface.WorkflowConfig{
		Name:      "test-workflow",
		TaskQueue: "test-queue",
	}

	mockClient := &mocks.Client{}
	mockTracer := &MockTracer{}
	mockSpan := &MockSpan{}

	// Setup mocks
	mockTracer.On("Start", mock.Anything, "temporal_workflow.signal", mock.Anything).Return(context.Background(), mockSpan)
	mockSpan.On("RecordError", mock.Anything, mock.Anything).Return()
	mockSpan.On("End", mock.Anything).Return()
	mockClient.On("SignalWorkflow", mock.Anything, "workflow-id", "run-id", "test-signal", "signal-data").Return(errors.New("signal failed"))

	temporalWorkflow := NewTemporalWorkflow(config, mockClient, mockTracer, nil)

	err := temporalWorkflow.Signal(context.Background(), "workflow-id", "run-id", "test-signal", "signal-data")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "signal failed")

	mockClient.AssertExpectations(t)
	mockTracer.AssertExpectations(t)
	mockSpan.AssertExpectations(t)
}

func TestTemporalWorkflow_Query_Success(t *testing.T) {
	config := iface.WorkflowConfig{
		Name:      "test-workflow",
		TaskQueue: "test-queue",
	}

	mockClient := &mocks.Client{}
	mockTracer := &MockTracer{}
	mockSpan := &MockSpan{}
	mockValue := &mocks.Value{}

	// Setup mocks
	mockTracer.On("Start", mock.Anything, "temporal_workflow.query", mock.Anything).Return(context.Background(), mockSpan)
	mockSpan.On("End", mock.Anything).Return() // Mock span.End() call
	// QueryWorkflow accepts variadic args, so we need to match all possible arguments
	mockClient.On("QueryWorkflow", mock.Anything, "workflow-id", "run-id", "test-query", "arg1", "arg2").Return(mockValue, nil)
	mockValue.On("Get", mock.AnythingOfType("*interface {}")).Run(func(args mock.Arguments) {
		resultPtr := args.Get(0).(*interface{})
		*resultPtr = "query_result"
	}).Return(nil)

	temporalWorkflow := NewTemporalWorkflow(config, mockClient, mockTracer, nil)

	result, err := temporalWorkflow.Query(context.Background(), "workflow-id", "run-id", "test-query", "arg1", "arg2")

	assert.NoError(t, err)
	assert.Equal(t, "query_result", result)

	mockClient.AssertExpectations(t)
	mockTracer.AssertExpectations(t)
	mockSpan.AssertExpectations(t)
	mockValue.AssertExpectations(t)
}

func TestTemporalWorkflow_Query_Error(t *testing.T) {
	config := iface.WorkflowConfig{
		Name:      "test-workflow",
		TaskQueue: "test-queue",
	}

	mockClient := &mocks.Client{}
	mockTracer := &MockTracer{}
	mockSpan := &MockSpan{}

	// Setup mocks
	mockTracer.On("Start", mock.Anything, "temporal_workflow.query", mock.Anything).Return(context.Background(), mockSpan)
	mockClient.On("QueryWorkflow", mock.Anything, "workflow-id", "run-id", "test-query", mock.Anything).Return(nil, errors.New("query failed"))
	mockSpan.On("RecordError", mock.Anything, mock.Anything).Return()
	mockSpan.On("End", mock.Anything).Return()

	temporalWorkflow := NewTemporalWorkflow(config, mockClient, mockTracer, nil)

	result, err := temporalWorkflow.Query(context.Background(), "workflow-id", "run-id", "test-query")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "query failed")

	mockClient.AssertExpectations(t)
	mockTracer.AssertExpectations(t)
	mockSpan.AssertExpectations(t)
}

func TestTemporalWorkflow_Cancel_Success(t *testing.T) {
	config := iface.WorkflowConfig{
		Name:      "test-workflow",
		TaskQueue: "test-queue",
	}

	mockClient := &mocks.Client{}
	mockTracer := &MockTracer{}
	mockSpan := &MockSpan{}

	// Setup mocks
	mockTracer.On("Start", mock.Anything, "temporal_workflow.cancel", mock.Anything).Return(context.Background(), mockSpan)
	mockClient.On("CancelWorkflow", mock.Anything, "workflow-id", "run-id").Return(nil)
	mockSpan.On("End", mock.Anything).Return()

	temporalWorkflow := NewTemporalWorkflow(config, mockClient, mockTracer, nil)

	err := temporalWorkflow.Cancel(context.Background(), "workflow-id", "run-id")

	assert.NoError(t, err)

	mockClient.AssertExpectations(t)
	mockTracer.AssertExpectations(t)
	mockSpan.AssertExpectations(t)
}

func TestTemporalWorkflow_Cancel_Error(t *testing.T) {
	config := iface.WorkflowConfig{
		Name:      "test-workflow",
		TaskQueue: "test-queue",
	}

	mockClient := &mocks.Client{}
	mockTracer := &MockTracer{}
	mockSpan := &MockSpan{}

	// Setup mocks
	mockTracer.On("Start", mock.Anything, "temporal_workflow.cancel", mock.Anything).Return(context.Background(), mockSpan)
	mockClient.On("CancelWorkflow", mock.Anything, "workflow-id", "run-id").Return(errors.New("cancel failed"))
	mockSpan.On("RecordError", mock.Anything, mock.Anything).Return()
	mockSpan.On("End", mock.Anything).Return()

	temporalWorkflow := NewTemporalWorkflow(config, mockClient, mockTracer, nil)

	err := temporalWorkflow.Cancel(context.Background(), "workflow-id", "run-id")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cancel failed")

	mockClient.AssertExpectations(t)
	mockTracer.AssertExpectations(t)
	mockSpan.AssertExpectations(t)
}

func TestTemporalWorkflow_Terminate_Success(t *testing.T) {
	config := iface.WorkflowConfig{
		Name:      "test-workflow",
		TaskQueue: "test-queue",
	}

	mockClient := &mocks.Client{}
	mockTracer := &MockTracer{}
	mockSpan := &MockSpan{}

	// Setup mocks
	mockTracer.On("Start", mock.Anything, "temporal_workflow.terminate", mock.Anything).Return(context.Background(), mockSpan)
	// Match variadic arguments - use mock.Anything for all variadic args
	mockClient.On("TerminateWorkflow", mock.Anything, "workflow-id", "run-id", "test-reason", mock.Anything, mock.Anything).Return(nil)
	mockSpan.On("End", mock.Anything).Return()

	temporalWorkflow := NewTemporalWorkflow(config, mockClient, mockTracer, nil)

	err := temporalWorkflow.Terminate(context.Background(), "workflow-id", "run-id", "test-reason", "detail1", "detail2")

	assert.NoError(t, err)

	mockClient.AssertExpectations(t)
	mockTracer.AssertExpectations(t)
	mockSpan.AssertExpectations(t)
}

func TestTemporalWorkflow_Terminate_Error(t *testing.T) {
	config := iface.WorkflowConfig{
		Name:      "test-workflow",
		TaskQueue: "test-queue",
	}

	mockClient := &mocks.Client{}
	mockTracer := &MockTracer{}
	mockSpan := &MockSpan{}

	// Setup mocks
	mockTracer.On("Start", mock.Anything, "temporal_workflow.terminate", mock.Anything).Return(context.Background(), mockSpan)
	// Terminate with no details - variadic args are empty
	mockClient.On("TerminateWorkflow", mock.Anything, "workflow-id", "run-id", "test-reason").Return(errors.New("terminate failed"))
	mockSpan.On("RecordError", mock.Anything, mock.Anything).Return()
	mockSpan.On("End", mock.Anything).Return()

	temporalWorkflow := NewTemporalWorkflow(config, mockClient, mockTracer, nil)

	err := temporalWorkflow.Terminate(context.Background(), "workflow-id", "run-id", "test-reason")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "terminate failed")

	mockClient.AssertExpectations(t)
	mockTracer.AssertExpectations(t)
}

func TestNewTemporalActivityWrapper(t *testing.T) {
	mockRunnable := &MockRunnable{name: "test-activity"}
	mockTracer := &MockTracer{}

	wrapper := NewTemporalActivityWrapper(mockRunnable, mockTracer)

	assert.NotNil(t, wrapper)
	assert.Equal(t, mockRunnable, wrapper.Runnable)
	assert.Equal(t, mockTracer, wrapper.tracer)
}

func TestTemporalActivityWrapper_Execute_Success(t *testing.T) {
	mockRunnable := &MockRunnable{name: "test-activity"}
	mockTracer := &MockTracer{}
	mockSpan := &MockSpan{}

	// Setup mocks
	mockTracer.On("Start", mock.Anything, "temporal_activity.execute", mock.Anything).Return(context.Background(), mockSpan)
	mockRunnable.On("Invoke", mock.Anything, "test-input", mock.Anything).Return("activity_result", nil)
	mockSpan.On("End", mock.Anything).Return()

	wrapper := NewTemporalActivityWrapper(mockRunnable, mockTracer)

	result, err := wrapper.Execute(context.Background(), "test-input")

	assert.NoError(t, err)
	assert.Equal(t, "activity_result", result)

	mockRunnable.AssertExpectations(t)
	mockTracer.AssertExpectations(t)
	mockSpan.AssertExpectations(t)
}

func TestTemporalActivityWrapper_Execute_Error(t *testing.T) {
	mockRunnable := &MockRunnable{name: "test-activity"}
	mockTracer := &MockTracer{}
	mockSpan := &MockSpan{}

	// Setup mocks
	mockTracer.On("Start", mock.Anything, "temporal_activity.execute", mock.Anything).Return(context.Background(), mockSpan)
	mockRunnable.On("Invoke", mock.Anything, "test-input", mock.Anything).Return(nil, errors.New("activity failed"))
	mockSpan.On("RecordError", mock.Anything, mock.Anything).Return()
	mockSpan.On("End", mock.Anything).Return()

	wrapper := NewTemporalActivityWrapper(mockRunnable, mockTracer)

	result, err := wrapper.Execute(context.Background(), "test-input")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "activity failed")

	mockRunnable.AssertExpectations(t)
	mockTracer.AssertExpectations(t)
	mockSpan.AssertExpectations(t)
}

func TestRegisterActivities(t *testing.T) {
	mockTracer := &MockTracer{}

	runnables := map[string]core.Runnable{
		"activity1": &MockRunnable{name: "activity1"},
		"activity2": &MockRunnable{name: "activity2"},
	}

	// This function doesn't do much in the current implementation
	// but we test that it doesn't panic and creates wrappers
	assert.NotPanics(t, func() {
		RegisterActivities(runnables, mockTracer)
	})
}

// Note: The client package imports need to be adjusted for the actual Temporal client interfaces
// This test file uses mock implementations that match the expected interface
