package inngest

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/lookatitude/beluga-ai/workflow"
)

// mockServer creates an httptest server that simulates the Inngest API.
func mockServer() *httptest.Server {
	mu := sync.Mutex{}
	data := make(map[string][]byte)

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract workflow ID from URL (last segment).
		path := r.URL.Path
		parts := splitPath(path)
		if len(parts) < 3 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		id := parts[len(parts)-1]

		switch r.Method {
		case http.MethodPut:
			body, _ := io.ReadAll(r.Body)
			mu.Lock()
			data[id] = body
			mu.Unlock()
			w.WriteHeader(http.StatusOK)

		case http.MethodGet:
			mu.Lock()
			val, ok := data[id]
			mu.Unlock()
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(val)

		case http.MethodDelete:
			mu.Lock()
			delete(data, id)
			mu.Unlock()
			w.WriteHeader(http.StatusOK)

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
}

func splitPath(path string) []string {
	var parts []string
	current := ""
	for _, c := range path {
		if c == '/' {
			if current != "" {
				parts = append(parts, current)
			}
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

func newTestStore(t *testing.T) (*Store, *httptest.Server) {
	t.Helper()
	srv := mockServer()
	t.Cleanup(srv.Close)

	store, err := New(Config{
		BaseURL: srv.URL,
		Client:  srv.Client(),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return store, srv
}

func TestNew(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		store, err := New(Config{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if store.baseURL != "http://localhost:8288" {
			t.Errorf("expected default URL, got %q", store.baseURL)
		}
	})

	t.Run("custom config", func(t *testing.T) {
		store, err := New(Config{
			BaseURL:  "http://custom:9999",
			EventKey: "test-key",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if store.baseURL != "http://custom:9999" {
			t.Errorf("expected custom URL, got %q", store.baseURL)
		}
		if store.eventKey != "test-key" {
			t.Errorf("expected test-key, got %q", store.eventKey)
		}
	})
}

func TestSaveAndLoad(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	state := workflow.WorkflowState{
		WorkflowID: "wf-1",
		RunID:      "run-1",
		Status:     workflow.StatusRunning,
	}

	if err := store.Save(ctx, state); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := store.Load(ctx, "wf-1")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected non-nil state")
	}
	if loaded.WorkflowID != "wf-1" {
		t.Errorf("expected 'wf-1', got %q", loaded.WorkflowID)
	}
	if loaded.Status != workflow.StatusRunning {
		t.Errorf("expected running, got %s", loaded.Status)
	}
}

func TestLoadNotFound(t *testing.T) {
	store, _ := newTestStore(t)

	loaded, err := store.Load(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded != nil {
		t.Error("expected nil for nonexistent workflow")
	}
}

func TestSaveEmptyID(t *testing.T) {
	store, _ := newTestStore(t)

	err := store.Save(context.Background(), workflow.WorkflowState{})
	if err == nil {
		t.Fatal("expected error for empty workflow ID")
	}
}

func TestSaveOverwrite(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	store.Save(ctx, workflow.WorkflowState{WorkflowID: "wf-1", Status: workflow.StatusRunning})
	store.Save(ctx, workflow.WorkflowState{WorkflowID: "wf-1", Status: workflow.StatusCompleted})

	loaded, _ := store.Load(ctx, "wf-1")
	if loaded.Status != workflow.StatusCompleted {
		t.Errorf("expected completed, got %s", loaded.Status)
	}
}

func TestList(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	store.Save(ctx, workflow.WorkflowState{WorkflowID: "wf-1", Status: workflow.StatusRunning})
	store.Save(ctx, workflow.WorkflowState{WorkflowID: "wf-2", Status: workflow.StatusCompleted})
	store.Save(ctx, workflow.WorkflowState{WorkflowID: "wf-3", Status: workflow.StatusRunning})

	// List all.
	all, err := store.List(ctx, workflow.WorkflowFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("expected 3 workflows, got %d", len(all))
	}

	// Filter by status.
	running, err := store.List(ctx, workflow.WorkflowFilter{Status: workflow.StatusRunning})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(running) != 2 {
		t.Errorf("expected 2 running workflows, got %d", len(running))
	}
}

func TestDelete(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	store.Save(ctx, workflow.WorkflowState{WorkflowID: "wf-1"})
	if err := store.Delete(ctx, "wf-1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	loaded, _ := store.Load(ctx, "wf-1")
	if loaded != nil {
		t.Error("expected nil after delete")
	}
}

func TestSaveWithHistory(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	state := workflow.WorkflowState{
		WorkflowID: "wf-1",
		Status:     workflow.StatusRunning,
		History: []workflow.HistoryEvent{
			{ID: 1, Type: workflow.EventWorkflowStarted},
			{ID: 2, Type: workflow.EventActivityStarted, ActivityName: "task1"},
		},
	}

	if err := store.Save(ctx, state); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, _ := store.Load(ctx, "wf-1")
	if len(loaded.History) != 2 {
		t.Errorf("expected 2 history events, got %d", len(loaded.History))
	}
}

func TestJSONRoundTrip(t *testing.T) {
	state := workflow.WorkflowState{
		WorkflowID: "wf-rt",
		RunID:      "run-rt",
		Status:     workflow.StatusCompleted,
	}

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var loaded workflow.WorkflowState
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if loaded.WorkflowID != state.WorkflowID {
		t.Errorf("expected %q, got %q", state.WorkflowID, loaded.WorkflowID)
	}
}

func TestInterfaceCompliance(t *testing.T) {
	var _ workflow.WorkflowStore = (*Store)(nil)
}
