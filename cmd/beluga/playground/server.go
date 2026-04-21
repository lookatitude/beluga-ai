package playground

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Config governs a playground Server. The zero value is invalid; use
// [New] which applies sensible defaults.
type Config struct {
	// Port to bind on 127.0.0.1. Zero picks an ephemeral port (useful
	// in tests). Callers can read the chosen port via [Server.Addr]
	// after [Server.Start] returns.
	Port int

	// MaxSpans bounds the span ring buffer (default 50).
	MaxSpans int

	// MaxStderrLines bounds the stderr ring buffer (default 100).
	MaxStderrLines int

	// Now is indirected for deterministic tests.
	Now func() time.Time
}

// Server is the minimal dev-UI HTTP server. It is safe for concurrent
// use; SpanSink and StderrSink may be written from multiple goroutines.
type Server struct {
	cfg    Config
	addr   string
	listen net.Listener
	mux    *http.ServeMux
	srv    *http.Server

	spans   *ringBuffer[SpanEvent]
	stderr  *ringBuffer[StderrLine]
	spanCh  chan SpanEvent
	errCh   chan StderrLine
	closing chan struct{}

	subsMu sync.Mutex
	subs   map[chan sseFrame]struct{}

	staticHandler http.Handler
	origin        string
}

type sseFrame struct {
	event string
	data  []byte
}

// New constructs a Server ready to be started. Assets are read from
// embeddedAssets (see assets.go) unless overridden for tests.
func New(cfg Config) (*Server, error) {
	if cfg.MaxSpans <= 0 {
		cfg.MaxSpans = 50
	}
	if cfg.MaxStderrLines <= 0 {
		cfg.MaxStderrLines = 100
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	sub, err := fs.Sub(embeddedAssets, "assets")
	if err != nil {
		return nil, fmt.Errorf("assets sub-fs: %w", err)
	}
	s := &Server{
		cfg:           cfg,
		spans:         newRingBuffer[SpanEvent](cfg.MaxSpans),
		stderr:        newRingBuffer[StderrLine](cfg.MaxStderrLines),
		spanCh:        make(chan SpanEvent, 256),
		errCh:         make(chan StderrLine, 256),
		closing:       make(chan struct{}),
		subs:          make(map[chan sseFrame]struct{}),
		staticHandler: http.FileServer(http.FS(sub)),
	}
	s.mux = http.NewServeMux()
	s.mux.HandleFunc("/events", s.handleEvents)
	s.mux.HandleFunc("/snapshot", s.handleSnapshot)
	s.mux.Handle("/", s.staticHandler)
	return s, nil
}

// Start binds the listener and begins serving. It returns as soon as
// the listener is ready; the actual server loop runs in a goroutine.
func (s *Server) Start() error {
	addr := fmt.Sprintf("127.0.0.1:%d", s.cfg.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", addr, err)
	}
	s.listen = ln
	s.addr = ln.Addr().String()
	s.origin = "http://" + s.addr
	s.srv = &http.Server{
		Handler:           s.mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      0, // SSE requires unbounded write; handler enforces its own deadlines
		IdleTimeout:       60 * time.Second,
	}
	go s.fanout()
	go func() {
		if serveErr := s.srv.Serve(ln); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			return
		}
	}()
	return nil
}

// Addr returns the address the server is bound to, e.g. "127.0.0.1:8089".
// Valid only after [Start].
func (s *Server) Addr() string { return s.addr }

// SpanSink returns the channel the devloop supervisor writes OTel span
// exports to. The channel is buffered; slow UI subscribers cannot
// back-pressure trace export.
func (s *Server) SpanSink() chan<- SpanEvent { return s.spanCh }

// StderrSink returns the channel the devloop supervisor writes tee'd
// stderr bytes to.
func (s *Server) StderrSink() chan<- StderrLine { return s.errCh }

// Close shuts the server down. It blocks until in-flight requests
// drain or ctx expires.
func (s *Server) Close(ctx context.Context) error {
	close(s.closing)
	if s.srv == nil {
		return nil
	}
	return s.srv.Shutdown(ctx)
}

// fanout multiplexes the two ingress channels onto the ring buffers
// and all active SSE subscribers. It exits when closing is signalled.
func (s *Server) fanout() {
	for {
		select {
		case <-s.closing:
			return
		case ev := <-s.spanCh:
			s.spans.push(ev)
			s.broadcast("span", ev)
		case ln := <-s.errCh:
			s.stderr.push(ln)
			s.broadcast("stderr", ln)
		}
	}
}

func (s *Server) broadcast(event string, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	s.subsMu.Lock()
	defer s.subsMu.Unlock()
	for sub := range s.subs {
		select {
		case sub <- sseFrame{event: event, data: data}:
		default:
		}
	}
}

// handleEvents is the SSE endpoint. It replays the current ring
// contents on connect, then streams live events until the client
// disconnects or the server shuts down.
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.sameOrigin(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	h.Set("X-Accel-Buffering", "no")
	h.Set("Access-Control-Allow-Origin", s.origin)
	w.WriteHeader(http.StatusOK)
	// Force headers to the socket so the client's Do() returns before
	// we block on the event channel. Without this, Go's http server
	// buffers the status line until the first body write, deadlocking
	// callers that push events only after Do() returns.
	flusher.Flush()

	sub := make(chan sseFrame, 64)
	s.subsMu.Lock()
	s.subs[sub] = struct{}{}
	s.subsMu.Unlock()
	defer func() {
		s.subsMu.Lock()
		delete(s.subs, sub)
		s.subsMu.Unlock()
	}()

	for _, ev := range s.spans.snapshot() {
		writeSSE(w, flusher, "span", ev)
	}
	for _, ln := range s.stderr.snapshot() {
		writeSSE(w, flusher, "stderr", ln)
	}

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.closing:
			return
		case frame := <-sub:
			writeFrame(w, flusher, frame)
		}
	}
}

// handleSnapshot returns the current ring contents as a single JSON
// blob. It is a convenience endpoint for tests and for UI cold-load
// without SSE replay churn.
func (s *Server) handleSnapshot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.sameOrigin(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	h := w.Header()
	h.Set("Content-Type", "application/json")
	h.Set("Access-Control-Allow-Origin", s.origin)
	body := struct {
		Spans  []SpanEvent  `json:"spans"`
		Stderr []StderrLine `json:"stderr"`
	}{
		Spans:  s.spans.snapshot(),
		Stderr: s.stderr.snapshot(),
	}
	_ = json.NewEncoder(w).Encode(body)
}

// sameOrigin enforces Sec-Fetch-Site=same-origin or an Origin header
// that matches the listener. Requests from outside the playground
// origin — including curl without an Origin — are allowed only for GET
// (SSE replay, snapshot cold-load). This keeps dev UX usable while
// still blocking cross-site browser JS from scraping the playground.
func (s *Server) sameOrigin(r *http.Request) bool {
	if r.Method == http.MethodGet {
		site := r.Header.Get("Sec-Fetch-Site")
		if site == "" || site == "same-origin" || site == "none" {
			origin := r.Header.Get("Origin")
			if origin == "" || origin == s.origin {
				return true
			}
		}
		return false
	}
	return r.Header.Get("Sec-Fetch-Site") == "same-origin" &&
		r.Header.Get("Origin") == s.origin
}

func writeSSE(w http.ResponseWriter, f http.Flusher, event string, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	writeFrame(w, f, sseFrame{event: event, data: data})
}

func writeFrame(w http.ResponseWriter, f http.Flusher, frame sseFrame) {
	if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", frame.event, escapeSSE(frame.data)); err != nil {
		return
	}
	f.Flush()
}

func escapeSSE(b []byte) string {
	s := string(b)
	if !strings.ContainsAny(s, "\r\n") {
		return s
	}
	return strings.NewReplacer("\r", "", "\n", "\\n").Replace(s)
}
