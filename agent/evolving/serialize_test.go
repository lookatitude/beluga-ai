package evolving

import (
	"context"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// slowDistiller counts concurrent Distill calls and records the interaction
// count it saw on each call.
type slowDistiller struct {
	active, maxActive atomic.Int32
	mu                sync.Mutex
	seen              []int
}

func (d *slowDistiller) Distill(_ context.Context, in []Interaction) ([]Experience, error) {
	n := d.active.Add(1)
	for {
		m := d.maxActive.Load()
		if n <= m || d.maxActive.CompareAndSwap(m, n) {
			break
		}
	}
	time.Sleep(20 * time.Millisecond)
	d.mu.Lock()
	d.seen = append(d.seen, len(in))
	d.mu.Unlock()
	d.active.Add(-1)
	return []Experience{{Category: "c", Pattern: strconv.Itoa(len(in))}}, nil
}

// TestEvolvingAgent_LearningSerialized is a regression test for FWC-20:
// every Nth call started an unbounded fire-and-forget learn goroutine, so
// runs overlapped and an older run could overwrite newer results.
func TestEvolvingAgent_LearningSerialized(t *testing.T) {
	d := &slowDistiller{}
	ea := New(&mockAgent{id: "a"}, WithDistiller(d), WithOptimizer(&stubOptimizer{}), WithLearnEveryN(1))
	const calls = 20
	for range calls {
		if _, err := ea.Invoke(context.Background(), "x"); err != nil {
			t.Fatal(err)
		}
	}
	deadline := time.Now().Add(3 * time.Second)
	for {
		ea.learnMu.Lock()
		running := ea.learnRunning
		ea.learnMu.Unlock()
		if !running {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("learning worker never finished")
		}
		time.Sleep(5 * time.Millisecond)
	}
	if m := d.maxActive.Load(); m != 1 {
		t.Errorf("max concurrent learn runs = %d, want 1", m)
	}
	d.mu.Lock()
	runs := len(d.seen)
	last := d.seen[len(d.seen)-1]
	d.mu.Unlock()
	if runs >= calls {
		t.Errorf("learn runs = %d, want triggers coalesced (< %d)", runs, calls)
	}
	if last != calls {
		t.Errorf("last learn saw %d interactions, want %d (newest)", last, calls)
	}
	if exps := ea.Experiences(); len(exps) != 1 || exps[0].Pattern != strconv.Itoa(calls) {
		t.Errorf("stored experiences = %+v, want the newest run", exps)
	}
}
