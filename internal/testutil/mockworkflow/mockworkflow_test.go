package mockworkflow

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name       string
		opts       []Option
		wantStates int
		wantErr    error
	}{
		{
			name:       "default configuration",
			wantStates: 0,
		},
		{
			name: "with states",
			opts: []Option{WithStates([]workflow.WorkflowState{
				{WorkflowID: "1", Status: workflow.StatusRunning},
				{WorkflowID: "2", Status: workflow.StatusCompleted},
			})},
			wantStates: 2,
		},
		{
			name:    "with error",
			opts:    []Option{WithError(errors.New("test error"))},
			wantErr: errors.New("test error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New(tt.opts...)
			require.NotNil(t, m)
			assert.Len(t, m.States(), tt.wantStates)
			if tt.wantErr != nil {
				assert.NotNil(t, m.err)
			}
		})
	}
}

func TestMockWorkflowStore_Save(t *testing.T) {
	tests := []struct {
		name    string
		opts    []Option
		state   workflow.WorkflowState
		wantErr bool
	}{
		{
			name: "save new workflow",
			state: workflow.WorkflowState{
				WorkflowID: "wf-1",
				RunID:      "run-1",
				Status:     workflow.StatusRunning,
				Input:      "test input",
				CreatedAt:  time.Now(),
			},
		},
		{
			name: "save completed workflow",
			state: workflow.WorkflowState{
				WorkflowID: "wf-2",
				RunID:      "run-2",
				Status:     workflow.StatusCompleted,
				Result:     "success",
				UpdatedAt:  time.Now(),
			},
		},
		{
			name:    "error path",
			opts:    []Option{WithError(errors.New("save failed"))},
			state:   workflow.WorkflowState{WorkflowID: "wf-3"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New(tt.opts...)
			ctx := context.Background()

			err := m.Save(ctx, tt.state)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, 1, m.SaveCalls())
			lastState := m.LastState()
			require.NotNil(t, lastState)
			assert.Equal(t, tt.state.WorkflowID, lastState.WorkflowID)
			assert.Equal(t, tt.state.Status, lastState.Status)
		})
	}
}

func TestMockWorkflowStore_Load(t *testing.T) {
	states := []workflow.WorkflowState{
		{WorkflowID: "wf-1", Status: workflow.StatusRunning},
		{WorkflowID: "wf-2", Status: workflow.StatusCompleted},
	}

	tests := []struct {
		name       string
		opts       []Option
		workflowID string
		wantStatus workflow.WorkflowStatus
		wantErr    bool
	}{
		{
			name:       "load existing workflow",
			opts:       []Option{WithStates(states)},
			workflowID: "wf-1",
			wantStatus: workflow.StatusRunning,
		},
		{
			name:       "load completed workflow",
			opts:       []Option{WithStates(states)},
			workflowID: "wf-2",
			wantStatus: workflow.StatusCompleted,
		},
		{
			name:       "load non-existent workflow",
			opts:       []Option{WithStates(states)},
			workflowID: "wf-999",
			wantErr:    true,
		},
		{
			name:       "error path",
			opts:       []Option{WithError(errors.New("load failed"))},
			workflowID: "wf-1",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New(tt.opts...)
			ctx := context.Background()

			state, err := m.Load(ctx, tt.workflowID)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, state)
			assert.Equal(t, tt.workflowID, state.WorkflowID)
			assert.Equal(t, tt.wantStatus, state.Status)
			assert.Equal(t, 1, m.LoadCalls())
		})
	}
}

func TestMockWorkflowStore_List(t *testing.T) {
	states := []workflow.WorkflowState{
		{WorkflowID: "wf-1", Status: workflow.StatusRunning},
		{WorkflowID: "wf-2", Status: workflow.StatusCompleted},
		{WorkflowID: "wf-3", Status: workflow.StatusRunning},
		{WorkflowID: "wf-4", Status: workflow.StatusFailed},
	}

	tests := []struct {
		name      string
		opts      []Option
		filter    workflow.WorkflowFilter
		wantCount int
		wantErr   bool
	}{
		{
			name:      "list all workflows",
			opts:      []Option{WithStates(states)},
			filter:    workflow.WorkflowFilter{},
			wantCount: 4,
		},
		{
			name:      "filter by running status",
			opts:      []Option{WithStates(states)},
			filter:    workflow.WorkflowFilter{Status: workflow.StatusRunning},
			wantCount: 2,
		},
		{
			name:      "filter by completed status",
			opts:      []Option{WithStates(states)},
			filter:    workflow.WorkflowFilter{Status: workflow.StatusCompleted},
			wantCount: 1,
		},
		{
			name:      "filter by failed status",
			opts:      []Option{WithStates(states)},
			filter:    workflow.WorkflowFilter{Status: workflow.StatusFailed},
			wantCount: 1,
		},
		{
			name:      "limit results",
			opts:      []Option{WithStates(states)},
			filter:    workflow.WorkflowFilter{Limit: 2},
			wantCount: 2,
		},
		{
			name:      "filter with limit",
			opts:      []Option{WithStates(states)},
			filter:    workflow.WorkflowFilter{Status: workflow.StatusRunning, Limit: 1},
			wantCount: 1,
		},
		{
			name:      "empty store",
			filter:    workflow.WorkflowFilter{},
			wantCount: 0,
		},
		{
			name:    "error path",
			opts:    []Option{WithError(errors.New("list failed"))},
			filter:  workflow.WorkflowFilter{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New(tt.opts...)
			ctx := context.Background()

			results, err := m.List(ctx, tt.filter)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, results, tt.wantCount)
			assert.Equal(t, 1, m.ListCalls())
			lastFilter := m.LastFilter()
			require.NotNil(t, lastFilter)
			assert.Equal(t, tt.filter.Status, lastFilter.Status)
			assert.Equal(t, tt.filter.Limit, lastFilter.Limit)
		})
	}
}

func TestMockWorkflowStore_Delete(t *testing.T) {
	states := []workflow.WorkflowState{
		{WorkflowID: "wf-1", Status: workflow.StatusRunning},
		{WorkflowID: "wf-2", Status: workflow.StatusCompleted},
	}

	tests := []struct {
		name          string
		opts          []Option
		workflowID    string
		wantErr       bool
		wantRemaining int
	}{
		{
			name:          "delete existing workflow",
			opts:          []Option{WithStates(states)},
			workflowID:    "wf-1",
			wantRemaining: 1,
		},
		{
			name:          "delete non-existent workflow",
			opts:          []Option{WithStates(states)},
			workflowID:    "wf-999",
			wantRemaining: 2,
		},
		{
			name:       "error path",
			opts:       []Option{WithError(errors.New("delete failed"))},
			workflowID: "wf-1",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New(tt.opts...)
			ctx := context.Background()

			err := m.Delete(ctx, tt.workflowID)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, 1, m.DeleteCalls())
			assert.Len(t, m.States(), tt.wantRemaining)
		})
	}
}

func TestMockWorkflowStore_WithSaveFunc(t *testing.T) {
	called := false
	customFn := func(ctx context.Context, state workflow.WorkflowState) error {
		called = true
		assert.Equal(t, "custom", state.WorkflowID)
		return nil
	}

	m := New(WithSaveFunc(customFn))
	err := m.Save(context.Background(), workflow.WorkflowState{WorkflowID: "custom"})

	require.NoError(t, err)
	assert.True(t, called)
}

func TestMockWorkflowStore_WithLoadFunc(t *testing.T) {
	called := false
	customFn := func(ctx context.Context, workflowID string) (*workflow.WorkflowState, error) {
		called = true
		return &workflow.WorkflowState{WorkflowID: workflowID, Status: workflow.StatusCompleted}, nil
	}

	m := New(WithLoadFunc(customFn))
	state, err := m.Load(context.Background(), "custom")

	require.NoError(t, err)
	assert.True(t, called)
	require.NotNil(t, state)
	assert.Equal(t, "custom", state.WorkflowID)
}

func TestMockWorkflowStore_WithListFunc(t *testing.T) {
	called := false
	customFn := func(ctx context.Context, filter workflow.WorkflowFilter) ([]workflow.WorkflowState, error) {
		called = true
		return []workflow.WorkflowState{{WorkflowID: "custom"}}, nil
	}

	m := New(WithListFunc(customFn))
	results, err := m.List(context.Background(), workflow.WorkflowFilter{})

	require.NoError(t, err)
	assert.True(t, called)
	assert.Len(t, results, 1)
}

func TestMockWorkflowStore_WithDeleteFunc(t *testing.T) {
	called := false
	customFn := func(ctx context.Context, workflowID string) error {
		called = true
		assert.Equal(t, "custom", workflowID)
		return nil
	}

	m := New(WithDeleteFunc(customFn))
	err := m.Delete(context.Background(), "custom")

	require.NoError(t, err)
	assert.True(t, called)
}

func TestMockWorkflowStore_SetError(t *testing.T) {
	m := New()
	ctx := context.Background()

	// Initial calls succeed
	err := m.Save(ctx, workflow.WorkflowState{WorkflowID: "wf-1"})
	require.NoError(t, err)

	// Set error
	testErr := errors.New("new error")
	m.SetError(testErr)
	err = m.Save(ctx, workflow.WorkflowState{WorkflowID: "wf-2"})
	require.Error(t, err)
	assert.Equal(t, testErr, err)
}

func TestMockWorkflowStore_Reset(t *testing.T) {
	states := []workflow.WorkflowState{{WorkflowID: "wf-1"}}
	m := New(
		WithStates(states),
		WithError(errors.New("test")),
	)

	// Make some calls
	_ = m.Save(context.Background(), workflow.WorkflowState{WorkflowID: "wf-2"})
	_, _ = m.Load(context.Background(), "wf-1")
	_, _ = m.List(context.Background(), workflow.WorkflowFilter{})
	_ = m.Delete(context.Background(), "wf-1")

	assert.Equal(t, 1, m.SaveCalls())
	assert.Equal(t, 1, m.LoadCalls())
	assert.Equal(t, 1, m.ListCalls())
	assert.Equal(t, 1, m.DeleteCalls())

	// Reset
	m.Reset()
	assert.Equal(t, 0, m.SaveCalls())
	assert.Equal(t, 0, m.LoadCalls())
	assert.Equal(t, 0, m.ListCalls())
	assert.Equal(t, 0, m.DeleteCalls())
	assert.Empty(t, m.States())
	assert.Nil(t, m.LastState())
	assert.Nil(t, m.LastFilter())
	assert.Nil(t, m.err)
}

func TestMockWorkflowStore_Concurrency(t *testing.T) {
	m := New()
	ctx := context.Background()

	// Multiple goroutines calling methods concurrently
	const goroutines = 10
	done := make(chan bool, goroutines*4)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			_ = m.Save(ctx, workflow.WorkflowState{WorkflowID: "test"})
			done <- true
		}(i)
		go func(id int) {
			_, _ = m.Load(ctx, "test")
			done <- true
		}(i)
		go func(id int) {
			_, _ = m.List(ctx, workflow.WorkflowFilter{})
			done <- true
		}(i)
		go func(id int) {
			_ = m.Delete(ctx, "test")
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < goroutines*4; i++ {
		<-done
	}

	assert.Equal(t, goroutines, m.SaveCalls())
	assert.Equal(t, goroutines, m.LoadCalls())
	assert.Equal(t, goroutines, m.ListCalls())
	assert.Equal(t, goroutines, m.DeleteCalls())
}

func TestMockWorkflowStore_SaveThenLoad(t *testing.T) {
	m := New()
	ctx := context.Background()

	// Save a workflow
	state := workflow.WorkflowState{
		WorkflowID: "wf-1",
		RunID:      "run-1",
		Status:     workflow.StatusRunning,
		Input:      "test input",
		CreatedAt:  time.Now(),
	}
	err := m.Save(ctx, state)
	require.NoError(t, err)

	// Load it back
	loaded, err := m.Load(ctx, "wf-1")
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Equal(t, state.WorkflowID, loaded.WorkflowID)
	assert.Equal(t, state.Status, loaded.Status)
	assert.Equal(t, state.Input, loaded.Input)
}

func TestMockWorkflowStore_SaveUpdateDelete(t *testing.T) {
	m := New()
	ctx := context.Background()

	// Save initial state
	state1 := workflow.WorkflowState{
		WorkflowID: "wf-1",
		Status:     workflow.StatusRunning,
	}
	err := m.Save(ctx, state1)
	require.NoError(t, err)

	// Update the state
	state2 := workflow.WorkflowState{
		WorkflowID: "wf-1",
		Status:     workflow.StatusCompleted,
		Result:     "done",
	}
	err = m.Save(ctx, state2)
	require.NoError(t, err)

	// Verify update
	loaded, err := m.Load(ctx, "wf-1")
	require.NoError(t, err)
	assert.Equal(t, workflow.StatusCompleted, loaded.Status)
	assert.Equal(t, "done", loaded.Result)

	// Delete
	err = m.Delete(ctx, "wf-1")
	require.NoError(t, err)

	// Verify deleted
	_, err = m.Load(ctx, "wf-1")
	require.Error(t, err)
}

func TestMockWorkflowStore_ListFiltering(t *testing.T) {
	m := New()
	ctx := context.Background()

	// Save workflows with different statuses
	states := []workflow.WorkflowState{
		{WorkflowID: "wf-1", Status: workflow.StatusRunning},
		{WorkflowID: "wf-2", Status: workflow.StatusRunning},
		{WorkflowID: "wf-3", Status: workflow.StatusCompleted},
		{WorkflowID: "wf-4", Status: workflow.StatusFailed},
	}
	for _, state := range states {
		err := m.Save(ctx, state)
		require.NoError(t, err)
	}

	// List only running workflows
	results, err := m.List(ctx, workflow.WorkflowFilter{Status: workflow.StatusRunning})
	require.NoError(t, err)
	assert.Len(t, results, 2)
	for _, r := range results {
		assert.Equal(t, workflow.StatusRunning, r.Status)
	}

	// List with limit
	results, err = m.List(ctx, workflow.WorkflowFilter{Limit: 2})
	require.NoError(t, err)
	assert.Len(t, results, 2)
}
