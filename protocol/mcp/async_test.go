package mcp

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestInMemoryAsyncHandler_StartAndPoll(t *testing.T) {
	h := NewInMemoryAsyncHandler()

	id, err := h.Start(context.Background(), func(ctx context.Context) (any, error) {
		return "result-value", nil
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty ID")
	}

	// Wait for completion.
	var op *AsyncOperation
	for i := 0; i < 50; i++ {
		op, err = h.Poll(context.Background(), id)
		if err != nil {
			t.Fatalf("Poll: %v", err)
		}
		if op.Status == AsyncStatusCompleted || op.Status == AsyncStatusFailed {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if op.Status != AsyncStatusCompleted {
		t.Errorf("expected completed, got %q", op.Status)
	}
	if op.Result != "result-value" {
		t.Errorf("expected result 'result-value', got %v", op.Result)
	}
	if op.Progress != 1.0 {
		t.Errorf("expected progress 1.0, got %f", op.Progress)
	}
}

func TestInMemoryAsyncHandler_StartAndFail(t *testing.T) {
	h := NewInMemoryAsyncHandler()

	id, err := h.Start(context.Background(), func(ctx context.Context) (any, error) {
		return nil, fmt.Errorf("something broke")
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	var op *AsyncOperation
	for i := 0; i < 50; i++ {
		op, err = h.Poll(context.Background(), id)
		if err != nil {
			t.Fatalf("Poll: %v", err)
		}
		if op.Status == AsyncStatusCompleted || op.Status == AsyncStatusFailed {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if op.Status != AsyncStatusFailed {
		t.Errorf("expected failed, got %q", op.Status)
	}
	if op.Error != "something broke" {
		t.Errorf("expected error 'something broke', got %q", op.Error)
	}
}

func TestInMemoryAsyncHandler_Cancel(t *testing.T) {
	h := NewInMemoryAsyncHandler()

	started := make(chan struct{})
	id, err := h.Start(context.Background(), func(ctx context.Context) (any, error) {
		close(started)
		<-ctx.Done()
		return nil, ctx.Err()
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Wait for the function to start running.
	<-started

	err = h.Cancel(context.Background(), id)
	if err != nil {
		t.Fatalf("Cancel: %v", err)
	}

	op, err := h.Poll(context.Background(), id)
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}
	if op.Status != AsyncStatusCancelled {
		t.Errorf("expected cancelled, got %q", op.Status)
	}
}

func TestInMemoryAsyncHandler_Cancel_NotFound(t *testing.T) {
	h := NewInMemoryAsyncHandler()

	err := h.Cancel(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent operation")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error %q should mention 'not found'", err)
	}
}

func TestInMemoryAsyncHandler_Cancel_TerminalState(t *testing.T) {
	h := NewInMemoryAsyncHandler()

	id, err := h.Start(context.Background(), func(ctx context.Context) (any, error) {
		return "done", nil
	})
	if err != nil {
		t.Fatal(err)
	}

	// Wait for completion.
	for i := 0; i < 50; i++ {
		op, _ := h.Poll(context.Background(), id)
		if op.Status == AsyncStatusCompleted {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	err = h.Cancel(context.Background(), id)
	if err == nil {
		t.Fatal("expected error when cancelling completed operation")
	}
	if !strings.Contains(err.Error(), "terminal state") {
		t.Errorf("error %q should mention 'terminal state'", err)
	}
}

func TestInMemoryAsyncHandler_Poll_NotFound(t *testing.T) {
	h := NewInMemoryAsyncHandler()

	_, err := h.Poll(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent operation")
	}
}

func TestInMemoryAsyncHandler_MaxOps(t *testing.T) {
	h := NewInMemoryAsyncHandler(WithMaxOps(1))

	blocker := make(chan struct{})
	_, err := h.Start(context.Background(), func(ctx context.Context) (any, error) {
		<-blocker
		return nil, nil
	})
	if err != nil {
		t.Fatalf("Start first: %v", err)
	}

	_, err = h.Start(context.Background(), func(ctx context.Context) (any, error) {
		return nil, nil
	})
	if err == nil {
		t.Fatal("expected error when max ops exceeded")
	}
	if !strings.Contains(err.Error(), "maximum operations") {
		t.Errorf("error %q should mention 'maximum operations'", err)
	}

	close(blocker)
}

func TestInMemoryAsyncHandler_Timeout(t *testing.T) {
	h := NewInMemoryAsyncHandler(WithAsyncTimeout(50 * time.Millisecond))

	id, err := h.Start(context.Background(), func(ctx context.Context) (any, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	})
	if err != nil {
		t.Fatal(err)
	}

	// Wait for the timeout to trigger.
	time.Sleep(200 * time.Millisecond)

	op, err := h.Poll(context.Background(), id)
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}
	if op.Status != AsyncStatusFailed {
		t.Errorf("expected failed after timeout, got %q", op.Status)
	}
}

func TestAsyncStatus_Values(t *testing.T) {
	statuses := []AsyncStatus{
		AsyncStatusPending,
		AsyncStatusRunning,
		AsyncStatusCompleted,
		AsyncStatusFailed,
		AsyncStatusCancelled,
	}
	for _, s := range statuses {
		if s == "" {
			t.Error("status should not be empty")
		}
	}
}
