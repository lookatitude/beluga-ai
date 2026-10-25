package kafka

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/workflow"
)

// topicReader replays messages captured by a mockWriter, then returns EOF.
type topicReader struct {
	msgs []Message
	err  error
}

func (r *topicReader) ReadMessage(ctx context.Context) (Message, error) {
	if len(r.msgs) == 0 {
		if r.err != nil {
			return Message{}, r.err
		}
		return Message{}, io.EOF
	}
	m := r.msgs[0]
	r.msgs = r.msgs[1:]
	return m, nil
}

func (r *topicReader) Close() error { return nil }

// TestStore_RestoreSurvivesRestart is a regression test for FWC-12: the
// Kafka store only ever read an in-process cache, so nothing survived a
// restart.
func TestStore_RestoreSurvivesRestart(t *testing.T) {
	ctx := context.Background()
	w := &mockWriter{}
	first := NewWithWriterReader(w, nil)
	for _, st := range []workflow.WorkflowState{
		{WorkflowID: "a", Status: workflow.StatusRunning},
		{WorkflowID: "b", Status: workflow.StatusRunning},
		{WorkflowID: "a", Status: workflow.StatusCompleted},
	} {
		if err := first.Save(ctx, st); err != nil {
			t.Fatal(err)
		}
	}
	if err := first.Delete(ctx, "b"); err != nil {
		t.Fatal(err)
	}

	// "Restart": a new store reading the same topic.
	second := NewWithWriterReader(&mockWriter{}, &topicReader{msgs: w.messages})
	if st, _ := second.Load(ctx, "a"); st != nil {
		t.Fatal("index should be empty before Restore")
	}
	n, err := second.Restore(ctx)
	if err != nil || n != 4 {
		t.Fatalf("Restore = %d, %v; want 4, nil", n, err)
	}
	st, _ := second.Load(ctx, "a")
	if st == nil || st.Status != workflow.StatusCompleted {
		t.Fatalf("Load(a) after Restore = %+v, want completed", st)
	}
	if st, _ := second.Load(ctx, "b"); st != nil {
		t.Error("tombstoned workflow b restored")
	}
}

func TestStore_RestoreErrors(t *testing.T) {
	ctx := context.Background()
	if _, err := NewWithWriterReader(&mockWriter{}, nil).Restore(ctx); core.CodeOf(err, "") != core.ErrInvalidInput {
		t.Errorf("no reader: err = %v", err)
	}
	boom := errors.New("broker gone")
	_, err := NewWithWriterReader(&mockWriter{}, &topicReader{err: boom}).Restore(ctx)
	if !errors.Is(err, boom) {
		t.Errorf("read error = %v, want %v", err, boom)
	}
	bad := &topicReader{msgs: []Message{{Key: []byte("x"), Value: []byte("{not json")}, {Value: []byte("{}")}}}
	if _, err := NewWithWriterReader(&mockWriter{}, bad).Restore(ctx); err == nil {
		t.Error("expected decode error")
	}
	ctxDone, cancel := context.WithTimeout(ctx, time.Millisecond)
	defer cancel()
	time.Sleep(2 * time.Millisecond)
	if _, err := NewWithWriterReader(&mockWriter{}, &topicReader{err: context.DeadlineExceeded}).Restore(ctxDone); err != nil {
		t.Errorf("deadline at end of topic should not be an error: %v", err)
	}
	empty := &topicReader{msgs: []Message{{Key: nil, Value: []byte("{}")}}}
	if n, err := NewWithWriterReader(&mockWriter{}, empty).Restore(ctx); err != nil || n != 1 {
		t.Errorf("keyless message: n=%d err=%v", n, err)
	}
}
