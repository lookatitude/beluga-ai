package playground

import (
	"sync"
	"time"
)

// SpanEvent is the minimal wire shape emitted to the playground UI for
// each OTel span. It is deliberately flat (no parent-id graph) because
// the UI renders a flat newest-first table, not a waterfall. The
// [SpanExporter] in span_export.go builds one of these per span it
// receives.
type SpanEvent struct {
	Name         string    `json:"name"`
	StartTime    time.Time `json:"start_time"`
	DurationMs   int64     `json:"duration_ms"`
	Status       string    `json:"status"`
	Model        string    `json:"model,omitempty"`
	InputTokens  int64     `json:"input_tokens,omitempty"`
	OutputTokens int64     `json:"output_tokens,omitempty"`
	RestartSeq   int       `json:"restart_seq"`
}

// StderrLine carries one batch of stderr bytes from the supervised
// child plus the restart sequence it came from. Batches are line-
// oriented but not guaranteed complete — the UI consumer concatenates
// them and re-splits on '\n' for display.
type StderrLine struct {
	Bytes      []byte `json:"bytes"`
	RestartSeq int    `json:"restart_seq"`
}

// ringBuffer is a fixed-capacity FIFO. Push appends; newer items evict
// older ones. Snapshot returns a defensive copy oldest-first. It is
// safe for concurrent use.
type ringBuffer[T any] struct {
	mu   sync.Mutex
	buf  []T
	cap  int
	head int
	size int
}

func newRingBuffer[T any](cap int) *ringBuffer[T] {
	return &ringBuffer[T]{buf: make([]T, cap), cap: cap}
}

func (r *ringBuffer[T]) push(v T) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.size < r.cap {
		r.buf[(r.head+r.size)%r.cap] = v
		r.size++
		return
	}
	r.buf[r.head] = v
	r.head = (r.head + 1) % r.cap
}

func (r *ringBuffer[T]) snapshot() []T {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]T, r.size)
	for i := 0; i < r.size; i++ {
		out[i] = r.buf[(r.head+i)%r.cap]
	}
	return out
}

func (r *ringBuffer[T]) len() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.size
}
