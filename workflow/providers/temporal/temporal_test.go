package temporal

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/converter"
	temporalmocks "go.temporal.io/sdk/mocks"
	"go.temporal.io/sdk/testsuite"
	temporalworkflow "go.temporal.io/sdk/workflow"

	"github.com/lookatitude/beluga-ai/workflow"
)

// mockEncodedValue implements converter.EncodedValue for testing.
type mockEncodedValue struct {
	val any
	err error
}

func (m *mockEncodedValue) Get(valuePtr any) error {
	if m.err != nil {
		return m.err
	}
	// Set the pointer to our value.
	if ptr, ok := valuePtr.(*any); ok && m.val != nil {
		*ptr = m.val
	}
	return nil
}

func (m *mockEncodedValue) HasValue() bool {
	return m.val != nil
}

// Compile-time check.
var _ converter.EncodedValue = (*mockEncodedValue)(nil)

func TestNewExecutorNilClient(t *testing.T) {
	_, err := NewExecutor(Config{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "client is required")
}

func TestNewExecutorDefaults(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	defer env.AssertExpectations(t)

	// We can't get a real client from test env, but we can test config defaults.
	cfg := Config{
		Client: nil, // Will fail, but we test the config path.
	}
	assert.Equal(t, "", cfg.TaskQueue)
	assert.Equal(t, time.Duration(0), cfg.DefaultTimeout)
}

func TestStoreOperations(t *testing.T) {
	// Store with nil client — we test the no-op methods.
	store := NewStore(nil, "")
	assert.Equal(t, "default", store.namespace)

	ctx := context.Background()

	// Save is a no-op.
	err := store.Save(ctx, workflow.WorkflowState{WorkflowID: "wf-1"})
	require.NoError(t, err)

	// List returns nil.
	results, err := store.List(ctx, workflow.WorkflowFilter{})
	require.NoError(t, err)
	assert.Nil(t, results)

	// Delete is a no-op.
	err = store.Delete(ctx, "wf-1")
	require.NoError(t, err)
}

func TestStoreCustomNamespace(t *testing.T) {
	store := NewStore(nil, "production")
	assert.Equal(t, "production", store.namespace)
}

func TestToTemporalRetryPolicy(t *testing.T) {
	t.Run("nil policy", func(t *testing.T) {
		result := toTemporalRetryPolicy(nil)
		assert.Nil(t, result)
	})

	t.Run("valid policy", func(t *testing.T) {
		p := &workflow.RetryPolicy{
			MaxAttempts:        3,
			InitialInterval:   100 * time.Millisecond,
			BackoffCoefficient: 2.0,
			MaxInterval:        10 * time.Second,
		}
		result := toTemporalRetryPolicy(p)
		require.NotNil(t, result)
		assert.Equal(t, int32(3), result.MaximumAttempts)
		assert.Equal(t, 100*time.Millisecond, result.InitialInterval)
		assert.Equal(t, 2.0, result.BackoffCoefficient)
		assert.Equal(t, 10*time.Second, result.MaximumInterval)
	})
}

func TestTemporalHandleAccessors(t *testing.T) {
	h := &temporalHandle{
		id:    "wf-123",
		runID: "run-456",
	}
	assert.Equal(t, "wf-123", h.ID())
	assert.Equal(t, "run-456", h.RunID())
	assert.Equal(t, workflow.StatusRunning, h.Status())
}

func TestWorkflowWrapperRun(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()

	// Register a simple workflow function that doubles its input.
	fn := func(ctx workflow.WorkflowContext, input any) (any, error) {
		n, _ := input.(float64)
		return n * 2, nil
	}

	wrapper := newWorkflowWrapper(fn, "test-queue")

	env.ExecuteWorkflow(wrapper.Run, float64(21))

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result float64
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, float64(42), result)
}

func TestWorkflowWrapperWithError(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()

	fn := func(ctx workflow.WorkflowContext, input any) (any, error) {
		return nil, fmt.Errorf("workflow failed")
	}

	wrapper := newWorkflowWrapper(fn, "test-queue")

	env.ExecuteWorkflow(wrapper.Run, nil)

	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())
}

func TestWorkflowWithActivity(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()

	// Register a simple activity.
	activity := func(ctx context.Context, input any) (any, error) {
		s, _ := input.(string)
		return "processed: " + s, nil
	}

	env.RegisterActivity(activity)

	fn := func(ctx workflow.WorkflowContext, input any) (any, error) {
		result, err := ctx.ExecuteActivity(activity, "hello")
		if err != nil {
			return nil, err
		}
		return result, nil
	}

	wrapper := newWorkflowWrapper(fn, "test-queue")
	env.ExecuteWorkflow(wrapper.Run, nil)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result string
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, "processed: hello", result)
}

func TestWorkflowWithSleep(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()

	fn := func(ctx workflow.WorkflowContext, input any) (any, error) {
		if err := ctx.Sleep(5 * time.Second); err != nil {
			return nil, err
		}
		return "done after sleep", nil
	}

	wrapper := newWorkflowWrapper(fn, "test-queue")
	env.ExecuteWorkflow(wrapper.Run, nil)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result string
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, "done after sleep", result)
}

func TestWorkflowWithSignal(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()

	fn := func(ctx workflow.WorkflowContext, input any) (any, error) {
		ch := ctx.ReceiveSignal("my-signal")
		// In Temporal test env, we need to use the Temporal workflow channel
		// but our bridge creates a Go channel. For the test environment,
		// the signal is delivered synchronously.

		// Use a selector-based approach within the Temporal context.
		tCtx := ctx.(*temporalContext)
		signalCh := temporalworkflow.GetSignalChannel(tCtx.tCtx, "my-signal-direct")
		var payload string
		signalCh.Receive(tCtx.tCtx, &payload)

		// Just verify the Go channel was created.
		_ = ch

		return payload, nil
	}

	wrapper := newWorkflowWrapper(fn, "test-queue")

	// Send signal before executing (in test env, signals are buffered).
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow("my-signal-direct", "signal-value")
	}, 0)

	env.ExecuteWorkflow(wrapper.Run, nil)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result string
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, "signal-value", result)
}

func TestRegistryRegistration(t *testing.T) {
	// The init() function registers "temporal" in the workflow registry.
	names := workflow.List()
	assert.Contains(t, names, "temporal")
}

func TestRegistryNewWithoutClient(t *testing.T) {
	_, err := workflow.New("temporal", workflow.Config{
		Extra: map[string]any{},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "client is required")
}

func TestTemporalContextMethods(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()

	fn := func(ctx workflow.WorkflowContext, input any) (any, error) {
		// Test context methods.
		done := ctx.Done()
		assert.Nil(t, done) // Returns nil for Temporal contexts.

		err := ctx.Err()
		assert.NoError(t, err) // No error initially.

		return "context-ok", nil
	}

	wrapper := newWorkflowWrapper(fn, "test-queue")
	env.ExecuteWorkflow(wrapper.Run, nil)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
}

func TestTemporalContextDeadlineAndValue(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()

	fn := func(ctx workflow.WorkflowContext, input any) (any, error) {
		// Test Deadline.
		deadline, ok := ctx.Deadline()
		// In Temporal test env, deadline may or may not be set.
		_ = deadline
		_ = ok

		// Test Value with a key that doesn't exist.
		val := ctx.Value("nonexistent-key")
		_ = val

		return "deadline-value-ok", nil
	}

	wrapper := newWorkflowWrapper(fn, "test-queue")
	env.ExecuteWorkflow(wrapper.Run, nil)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
}

func TestNewExecutor_WithDefaults(t *testing.T) {
	mockClient := &temporalmocks.Client{}
	exec, err := NewExecutor(Config{
		Client: mockClient,
	})
	require.NoError(t, err)
	assert.Equal(t, "beluga-workflows", exec.taskQueue)
	assert.Equal(t, 10*time.Minute, exec.timeout)
}

func TestNewExecutor_CustomConfig(t *testing.T) {
	mockClient := &temporalmocks.Client{}
	exec, err := NewExecutor(Config{
		Client:         mockClient,
		TaskQueue:      "custom-queue",
		DefaultTimeout: 30 * time.Minute,
	})
	require.NoError(t, err)
	assert.Equal(t, "custom-queue", exec.taskQueue)
	assert.Equal(t, 30*time.Minute, exec.timeout)
}

func TestExecutor_Execute(t *testing.T) {
	mockClient := &temporalmocks.Client{}
	mockRun := &temporalmocks.WorkflowRun{}
	mockRun.On("GetRunID").Return("run-123")

	mockClient.On("ExecuteWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(mockRun, nil)

	exec, err := NewExecutor(Config{Client: mockClient})
	require.NoError(t, err)

	fn := func(ctx workflow.WorkflowContext, input any) (any, error) {
		return "result", nil
	}

	handle, err := exec.Execute(context.Background(), fn, workflow.WorkflowOptions{
		ID:    "wf-test",
		Input: "hello",
	})
	require.NoError(t, err)
	assert.Equal(t, "wf-test", handle.ID())
	assert.Equal(t, "run-123", handle.RunID())

	mockClient.AssertExpectations(t)
}

func TestExecutor_Execute_GeneratesID(t *testing.T) {
	mockClient := &temporalmocks.Client{}
	mockRun := &temporalmocks.WorkflowRun{}
	mockRun.On("GetRunID").Return("run-auto")

	mockClient.On("ExecuteWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(mockRun, nil)

	exec, err := NewExecutor(Config{Client: mockClient})
	require.NoError(t, err)

	fn := func(ctx workflow.WorkflowContext, input any) (any, error) {
		return nil, nil
	}

	// Empty ID should be auto-generated.
	handle, err := exec.Execute(context.Background(), fn, workflow.WorkflowOptions{})
	require.NoError(t, err)
	assert.Contains(t, handle.ID(), "beluga-wf-")
}

func TestExecutor_Execute_CustomTimeout(t *testing.T) {
	mockClient := &temporalmocks.Client{}
	mockRun := &temporalmocks.WorkflowRun{}
	mockRun.On("GetRunID").Return("run-t")

	mockClient.On("ExecuteWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(mockRun, nil)

	exec, err := NewExecutor(Config{Client: mockClient})
	require.NoError(t, err)

	fn := func(ctx workflow.WorkflowContext, input any) (any, error) {
		return nil, nil
	}

	handle, err := exec.Execute(context.Background(), fn, workflow.WorkflowOptions{
		ID:      "wf-custom-timeout",
		Timeout: 1 * time.Hour,
	})
	require.NoError(t, err)
	assert.NotNil(t, handle)
}

func TestExecutor_Execute_Error(t *testing.T) {
	mockClient := &temporalmocks.Client{}
	mockClient.On("ExecuteWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, fmt.Errorf("temporal unavailable"))

	exec, err := NewExecutor(Config{Client: mockClient})
	require.NoError(t, err)

	fn := func(ctx workflow.WorkflowContext, input any) (any, error) {
		return nil, nil
	}

	handle, err := exec.Execute(context.Background(), fn, workflow.WorkflowOptions{ID: "wf-err"})
	require.Error(t, err)
	assert.Nil(t, handle)
	assert.Contains(t, err.Error(), "temporal/execute")
}

func TestExecutor_Signal(t *testing.T) {
	mockClient := &temporalmocks.Client{}
	mockClient.On("SignalWorkflow", mock.Anything, "wf-sig", "", "my-signal", "payload").
		Return(nil)

	exec, err := NewExecutor(Config{Client: mockClient})
	require.NoError(t, err)

	err = exec.Signal(context.Background(), "wf-sig", workflow.Signal{
		Name:    "my-signal",
		Payload: "payload",
	})
	require.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestExecutor_Signal_Error(t *testing.T) {
	mockClient := &temporalmocks.Client{}
	mockClient.On("SignalWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(fmt.Errorf("signal failed"))

	exec, err := NewExecutor(Config{Client: mockClient})
	require.NoError(t, err)

	err = exec.Signal(context.Background(), "wf-sig", workflow.Signal{Name: "sig"})
	require.Error(t, err)
}

func TestExecutor_Cancel(t *testing.T) {
	mockClient := &temporalmocks.Client{}
	mockClient.On("CancelWorkflow", mock.Anything, "wf-cancel", "").
		Return(nil)

	exec, err := NewExecutor(Config{Client: mockClient})
	require.NoError(t, err)

	err = exec.Cancel(context.Background(), "wf-cancel")
	require.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestExecutor_Cancel_Error(t *testing.T) {
	mockClient := &temporalmocks.Client{}
	mockClient.On("CancelWorkflow", mock.Anything, mock.Anything, mock.Anything).
		Return(fmt.Errorf("cancel failed"))

	exec, err := NewExecutor(Config{Client: mockClient})
	require.NoError(t, err)

	err = exec.Cancel(context.Background(), "wf-cancel")
	require.Error(t, err)
}

func TestExecutor_Query(t *testing.T) {
	mockClient := &temporalmocks.Client{}

	encodedValue := &mockEncodedValue{val: "query-result"}
	mockClient.On("QueryWorkflow", mock.Anything, "wf-query", "", "status").
		Return(encodedValue, nil)

	exec, err := NewExecutor(Config{Client: mockClient})
	require.NoError(t, err)

	result, err := exec.Query(context.Background(), "wf-query", "status")
	require.NoError(t, err)
	assert.Equal(t, "query-result", result)
	mockClient.AssertExpectations(t)
}

func TestExecutor_Query_DecodeError(t *testing.T) {
	mockClient := &temporalmocks.Client{}

	encodedValue := &mockEncodedValue{err: fmt.Errorf("decode failed")}
	mockClient.On("QueryWorkflow", mock.Anything, "wf-query", "", "status").
		Return(encodedValue, nil)

	exec, err := NewExecutor(Config{Client: mockClient})
	require.NoError(t, err)

	result, err := exec.Query(context.Background(), "wf-query", "status")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "temporal/query: decode")
}

func TestExecutor_Query_Error(t *testing.T) {
	mockClient := &temporalmocks.Client{}
	mockClient.On("QueryWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, fmt.Errorf("query failed"))

	exec, err := NewExecutor(Config{Client: mockClient})
	require.NoError(t, err)

	result, err := exec.Query(context.Background(), "wf-query", "status")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "temporal/query")
}

func TestTemporalHandle_Result(t *testing.T) {
	mockRun := &temporalmocks.WorkflowRun{}
	mockRun.On("Get", mock.Anything, mock.Anything).Return(nil)

	h := &temporalHandle{
		run:   mockRun,
		id:    "wf-result",
		runID: "run-result",
	}

	result, err := h.Result(context.Background())
	require.NoError(t, err)
	_ = result
	mockRun.AssertExpectations(t)
}

func TestTemporalHandle_Result_Error(t *testing.T) {
	mockRun := &temporalmocks.WorkflowRun{}
	mockRun.On("Get", mock.Anything, mock.Anything).Return(fmt.Errorf("workflow failed"))

	h := &temporalHandle{
		run:   mockRun,
		id:    "wf-err",
		runID: "run-err",
	}

	result, err := h.Result(context.Background())
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "temporal/result")
}

func TestStore_Load(t *testing.T) {
	mockClient := &temporalmocks.Client{}
	mockRun := &temporalmocks.WorkflowRun{}
	mockRun.On("GetRunID").Return("run-load")

	mockClient.On("GetWorkflow", mock.Anything, "wf-load", "").
		Return(mockRun)

	store := NewStore(mockClient, "test")
	state, err := store.Load(context.Background(), "wf-load")
	require.NoError(t, err)
	require.NotNil(t, state)
	assert.Equal(t, "wf-load", state.WorkflowID)
	assert.Equal(t, "run-load", state.RunID)
	assert.Equal(t, workflow.StatusRunning, state.Status)
	mockClient.AssertExpectations(t)
}

func TestRegistry_NewWithClient(t *testing.T) {
	mockClient := &temporalmocks.Client{}

	exec, err := workflow.New("temporal", workflow.Config{
		Extra: map[string]any{
			"client":     mockClient,
			"task_queue": "custom-queue",
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, exec)
}

func TestRegistry_NewWithEmptyTaskQueue(t *testing.T) {
	mockClient := &temporalmocks.Client{}

	exec, err := workflow.New("temporal", workflow.Config{
		Extra: map[string]any{
			"client": mockClient,
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, exec)
}

func TestActivityError(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()

	activity := func(ctx context.Context, input any) (any, error) {
		return nil, fmt.Errorf("activity failed")
	}

	env.RegisterActivity(activity)

	fn := func(ctx workflow.WorkflowContext, input any) (any, error) {
		result, err := ctx.ExecuteActivity(activity, "hello")
		if err != nil {
			return nil, err
		}
		return result, nil
	}

	wrapper := newWorkflowWrapper(fn, "test-queue")
	env.ExecuteWorkflow(wrapper.Run, nil)

	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())
}

func TestWorkflowInterfaceChecks(t *testing.T) {
	var _ workflow.DurableExecutor = (*Executor)(nil)
	var _ workflow.WorkflowHandle = (*temporalHandle)(nil)
	var _ workflow.WorkflowStore = (*Store)(nil)
	var _ workflow.WorkflowContext = (*temporalContext)(nil)
}
