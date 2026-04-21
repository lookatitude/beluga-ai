package playground

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	s, err := New(Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = s.Close(ctx)
	})
	return s
}

func TestServer_BindsLoopbackOnly(t *testing.T) {
	t.Parallel()
	s := newTestServer(t)
	host, _, err := net.SplitHostPort(s.Addr())
	if err != nil {
		t.Fatal(err)
	}
	if host != "127.0.0.1" {
		t.Fatalf("bound to %q want 127.0.0.1", host)
	}
}

func TestServer_SnapshotEmpty(t *testing.T) {
	t.Parallel()
	s := newTestServer(t)
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://"+s.Addr()+"/snapshot", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	var body struct {
		Spans  []SpanEvent  `json:"spans"`
		Stderr []StderrLine `json:"stderr"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Spans) != 0 || len(body.Stderr) != 0 {
		t.Fatalf("want empty, got %d spans / %d stderr", len(body.Spans), len(body.Stderr))
	}
}

func TestServer_SnapshotIncludesPushedEvents(t *testing.T) {
	t.Parallel()
	s := newTestServer(t)

	s.SpanSink() <- SpanEvent{Name: "agent.invoke", DurationMs: 123, Status: "ok", RestartSeq: 1}
	s.StderrSink() <- StderrLine{Bytes: []byte("hello stderr\n"), RestartSeq: 1}

	deadline := time.Now().Add(2 * time.Second)
	for {
		if s.spans.len() >= 1 && s.stderr.len() >= 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("fanout never populated rings")
		}
		time.Sleep(10 * time.Millisecond)
	}

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://"+s.Addr()+"/snapshot", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	var body struct {
		Spans  []SpanEvent  `json:"spans"`
		Stderr []StderrLine `json:"stderr"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Spans) != 1 || body.Spans[0].Name != "agent.invoke" {
		t.Fatalf("spans=%+v", body.Spans)
	}
	if len(body.Stderr) != 1 || string(body.Stderr[0].Bytes) != "hello stderr\n" {
		t.Fatalf("stderr=%+v", body.Stderr)
	}
}

func TestServer_EventsStream(t *testing.T) {
	t.Parallel()
	s := newTestServer(t)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://"+s.Addr()+"/events", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if got := resp.Header.Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("content-type=%q", got)
	}
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "http://"+s.Addr() {
		t.Fatalf("cors=%q want exact origin", got)
	}

	// Give the SSE handler time to register the subscriber before we push.
	time.Sleep(50 * time.Millisecond)
	s.SpanSink() <- SpanEvent{Name: "llm.stream", DurationMs: 42, Status: "ok"}

	lines := readUntilEvent(t, resp.Body, "span", 2*time.Second)
	if !strings.Contains(lines, "llm.stream") {
		t.Fatalf("want span frame, got %q", lines)
	}
}

func TestServer_EventsRejectsCrossOrigin(t *testing.T) {
	t.Parallel()
	s := newTestServer(t)
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://"+s.Addr()+"/events", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	req.Header.Set("Origin", "http://evil.example")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status=%d want 403", resp.StatusCode)
	}
}

func TestServer_EventsRejectsPostFromUnknownOrigin(t *testing.T) {
	t.Parallel()
	s := newTestServer(t)
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "http://"+s.Addr()+"/events", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Origin", "http://evil.example")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusForbidden && resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d want 403/405", resp.StatusCode)
	}
}

func TestServer_StaticIndexServed(t *testing.T) {
	t.Parallel()
	s := newTestServer(t)
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://"+s.Addr()+"/", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "beluga dev — playground") {
		t.Fatalf("index missing title: first bytes=%q", string(body)[:min(200, len(body))])
	}
}

func TestServer_RingBufferEvictsOldestSpan(t *testing.T) {
	t.Parallel()
	s, err := New(Config{MaxSpans: 3})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = s.Close(ctx)
	})
	for i := 0; i < 5; i++ {
		s.SpanSink() <- SpanEvent{Name: "s", RestartSeq: i}
	}
	deadline := time.Now().Add(2 * time.Second)
	for s.spans.len() < 3 {
		if time.Now().After(deadline) {
			t.Fatal("ring never filled")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if got := s.spans.len(); got != 3 {
		t.Fatalf("len=%d want 3", got)
	}
}

func readUntilEvent(t *testing.T, r io.Reader, want string, timeout time.Duration) string {
	t.Helper()
	type line struct{ s string }
	ch := make(chan line, 32)
	done := make(chan struct{})
	defer close(done)
	go func() {
		sc := bufio.NewScanner(r)
		for sc.Scan() {
			select {
			case ch <- line{s: sc.Text()}:
			case <-done:
				return
			}
		}
	}()

	deadline := time.After(timeout)
	var b strings.Builder
	sawEvent := false
	for {
		select {
		case <-deadline:
			t.Fatalf("timeout waiting for event %q; got so far: %q", want, b.String())
		case l := <-ch:
			b.WriteString(l.s + "\n")
			if strings.HasPrefix(l.s, "event: "+want) {
				sawEvent = true
			}
			if sawEvent && strings.HasPrefix(l.s, "data: ") {
				return b.String()
			}
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
