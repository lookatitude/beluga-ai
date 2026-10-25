package a2a

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// postResult is the status, headers and body of a POST /tasks call.
type postResult struct {
	StatusCode int
	Header     http.Header
}

func postTask(t *testing.T, url, body string) (postResult, []byte) {
	t.Helper()
	resp, err := testPost(t, url+"/tasks", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	return postResult{StatusCode: resp.StatusCode, Header: resp.Header}, raw
}

func waitStatus(t *testing.T, srv *A2AServer, id string, want TaskStatus) Task {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		srv.mu.RLock()
		task, ok := srv.tasks[id]
		var snap Task
		if ok {
			snap = *task
		}
		srv.mu.RUnlock()
		if ok && snap.Status == want {
			return snap
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("task %s never reached %s", id, want)
	return Task{}
}

func TestA2AServer_FailedTaskDoesNotLeakError(t *testing.T) {
	srv := NewServer(&mockAgent{id: "x", invokeFn: func(context.Context, string) (string, error) {
		return "", fmt.Errorf("db password=SECRET at /etc/x")
	}}, AgentCard{Name: "x"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	_, raw := postTask(t, ts.URL, `{"input":"hi"}`)
	var tr TaskResponse
	if err := json.Unmarshal(raw, &tr); err != nil {
		t.Fatal(err)
	}
	waitStatus(t, srv, tr.Task.ID, StatusFailed)

	resp, err := testGet(t, ts.URL+"/tasks/"+tr.Task.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if strings.Contains(string(body), "SECRET") {
		t.Fatalf("internal error leaked: %s", body)
	}
	if !strings.Contains(string(body), "internal error (request_id=") {
		t.Fatalf("expected generic error with request id: %s", body)
	}
}

func TestA2AServer_InvalidBodyDoesNotEchoDecoderError(t *testing.T) {
	_, ts := setupA2ATestServer()
	defer ts.Close()
	resp, raw := postTask(t, ts.URL, `{"input": nope}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if strings.Contains(string(raw), "invalid character") || !strings.Contains(string(raw), `"invalid request body"`) {
		t.Fatalf("unexpected body %s", raw)
	}
}

func TestA2AServer_BodyLimit(t *testing.T) {
	_, ts := setupA2ATestServer()
	defer ts.Close()
	resp, _ := postTask(t, ts.URL, `{"input":"`+strings.Repeat("a", 2<<20)+`"}`)
	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("status %d, want 413", resp.StatusCode)
	}
}

func TestA2AServer_ConcurrencyLimit(t *testing.T) {
	release := make(chan struct{})
	started := make(chan struct{}, 4)
	srv := NewServer(&mockAgent{id: "x", invokeFn: func(ctx context.Context, _ string) (string, error) {
		started <- struct{}{}
		select {
		case <-release:
		case <-ctx.Done():
		}
		return "ok", nil
	}}, AgentCard{Name: "x"}, WithMaxConcurrentTasks(2))
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for i := 0; i < 2; i++ {
		if resp, raw := postTask(t, ts.URL, `{"input":"hi"}`); resp.StatusCode != http.StatusCreated {
			t.Fatalf("task %d: status %d %s", i, resp.StatusCode, raw)
		}
	}
	<-started
	<-started
	resp, _ := postTask(t, ts.URL, `{"input":"hi"}`)
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("third task: status %d, want 503", resp.StatusCode)
	}
	if resp.Header.Get("Retry-After") == "" {
		t.Error("missing Retry-After")
	}
	close(release)

	// Slots are released once tasks finish.
	deadline := time.Now().Add(5 * time.Second)
	for {
		resp, _ := postTask(t, ts.URL, `{"input":"hi"}`)
		if resp.StatusCode == http.StatusCreated {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("slots never released")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestA2AServer_TaskTTLEviction(t *testing.T) {
	var mu sync.Mutex
	now := time.Unix(1000, 0)
	srv := NewServer(&mockAgent{id: "x"}, AgentCard{Name: "x"}, WithTaskTTL(time.Minute))
	srv.now = func() time.Time { mu.Lock(); defer mu.Unlock(); return now }
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	_, raw := postTask(t, ts.URL, `{"input":"hi"}`)
	var tr TaskResponse
	_ = json.Unmarshal(raw, &tr)
	waitStatus(t, srv, tr.Task.ID, StatusCompleted)

	resp, err := testGet(t, ts.URL+"/tasks/"+tr.Task.ID)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("before TTL: status %d", resp.StatusCode)
	}

	mu.Lock()
	now = now.Add(2 * time.Minute)
	mu.Unlock()

	resp, err = testGet(t, ts.URL+"/tasks/"+tr.Task.ID)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("after TTL: status %d, want 404", resp.StatusCode)
	}
	srv.mu.RLock()
	n := len(srv.tasks) + len(srv.finished)
	srv.mu.RUnlock()
	if n != 0 {
		t.Fatalf("expected maps empty, got %d entries", n)
	}
}

func TestA2AServer_CanceledUnstartedTaskEvicted(t *testing.T) {
	srv := NewServer(&mockAgent{id: "x"}, AgentCard{Name: "x"}, WithTaskTTL(time.Minute))
	srv.mu.Lock()
	srv.tasks["t1"] = &Task{ID: "t1", Status: StatusCompleted}
	srv.mu.Unlock()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()
	resp, err := testPost(t, ts.URL+"/tasks/t1/cancel", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	srv.mu.RLock()
	_, tracked := srv.finished["t1"]
	srv.mu.RUnlock()
	if !tracked {
		t.Fatal("canceled task not tracked for TTL eviction")
	}
}

func TestA2AServer_TaskTimeout(t *testing.T) {
	srv := NewServer(&mockAgent{id: "x", invokeFn: func(ctx context.Context, _ string) (string, error) {
		<-ctx.Done()
		return "", ctx.Err()
	}}, AgentCard{Name: "x"}, WithTaskTimeout(20*time.Millisecond))
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()
	_, raw := postTask(t, ts.URL, `{"input":"hi"}`)
	var tr TaskResponse
	_ = json.Unmarshal(raw, &tr)
	got := waitStatus(t, srv, tr.Task.ID, StatusFailed)
	if got.Error != "task timed out" {
		t.Fatalf("error = %q", got.Error)
	}
}

// TestA2AServer_TaskSurvivesRequestCancel: the task context is detached from
// the request but keeps its values.
func TestA2AServer_TaskSurvivesRequestCancel(t *testing.T) {
	type key struct{}
	gotVal := make(chan any, 1)
	srv := NewServer(&mockAgent{id: "x", invokeFn: func(ctx context.Context, _ string) (string, error) {
		time.Sleep(20 * time.Millisecond)
		gotVal <- ctx.Value(key{})
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		if _, ok := ctx.Deadline(); !ok {
			return "", fmt.Errorf("no deadline")
		}
		return "ok", nil
	}}, AgentCard{Name: "x"})

	reqCtx, cancel := context.WithCancel(context.WithValue(context.Background(), key{}, "v"))
	req := httptest.NewRequestWithContext(reqCtx, http.MethodPost, "/tasks", strings.NewReader(`{"input":"hi"}`))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	cancel()
	var tr TaskResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &tr)
	waitStatus(t, srv, tr.Task.ID, StatusCompleted)
	if v := <-gotVal; v != "v" {
		t.Fatalf("context value lost: %v", v)
	}
}

func TestA2AServer_OptionsIgnoreNonPositive(t *testing.T) {
	srv := NewServer(&mockAgent{id: "x"}, AgentCard{}, WithTaskTTL(0), WithMaxConcurrentTasks(-1), WithTaskTimeout(-time.Second))
	if srv.taskTTL != DefaultTaskTTL || cap(srv.sem) != DefaultMaxConcurrentTasks || srv.taskTimeout != DefaultTaskTimeout {
		t.Fatalf("defaults not kept: %v %d %v", srv.taskTTL, cap(srv.sem), srv.taskTimeout)
	}
}
