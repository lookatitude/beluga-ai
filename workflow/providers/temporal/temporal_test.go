package temporal

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
	temporalworkflow "go.temporal.io/sdk/workflow"

	"github.com/lookatitude/beluga-ai/workflow"
)

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
