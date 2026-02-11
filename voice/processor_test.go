package voice

import (
	"context"
	"errors"
	"testing"
)

func TestFrameProcessorFunc(t *testing.T) {
	called := false
	f := FrameProcessorFunc(func(_ context.Context, in <-chan Frame, out chan<- Frame) error {
		defer close(out)
		called = true
		for frame := range in {
			out <- frame
		}
		return nil
	})

	in := make(chan Frame, 1)
	out := make(chan Frame, 1)
	in <- NewTextFrame("test")
	close(in)

	err := f.Process(context.Background(), in, out)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if !called {
		t.Error("FrameProcessorFunc was not called")
	}

	frame := <-out
	if frame.Text() != "test" {
		t.Errorf("frame.Text() = %q, want %q", frame.Text(), "test")
	}
}

func TestChainEmpty(t *testing.T) {
	chain := Chain()

	in := make(chan Frame, 1)
	out := make(chan Frame, 1)
	in <- NewTextFrame("passthrough")
	close(in)

	err := chain.Process(context.Background(), in, out)
	if err != nil {
		t.Fatalf("Chain() error = %v", err)
	}

	frame := <-out
	if frame.Text() != "passthrough" {
		t.Errorf("frame.Text() = %q, want %q", frame.Text(), "passthrough")
	}
}

func TestChainSingle(t *testing.T) {
	upper := FrameProcessorFunc(func(_ context.Context, in <-chan Frame, out chan<- Frame) error {
		defer close(out)
		for f := range in {
			if f.Type == FrameText {
				out <- NewTextFrame(f.Text() + "!")
			} else {
				out <- f
			}
		}
		return nil
	})

	chain := Chain(upper)

	in := make(chan Frame, 1)
	out := make(chan Frame, 1)
	in <- NewTextFrame("hello")
	close(in)

	err := chain.Process(context.Background(), in, out)
	if err != nil {
		t.Fatalf("Chain() error = %v", err)
	}

	frame := <-out
	if frame.Text() != "hello!" {
		t.Errorf("frame.Text() = %q, want %q", frame.Text(), "hello!")
	}
}

func TestChainMultiple(t *testing.T) {
	addExclaim := FrameProcessorFunc(func(_ context.Context, in <-chan Frame, out chan<- Frame) error {
		defer close(out)
		for f := range in {
			if f.Type == FrameText {
				out <- NewTextFrame(f.Text() + "!")
			} else {
				out <- f
			}
		}
		return nil
	})

	addQuestion := FrameProcessorFunc(func(_ context.Context, in <-chan Frame, out chan<- Frame) error {
		defer close(out)
		for f := range in {
			if f.Type == FrameText {
				out <- NewTextFrame(f.Text() + "?")
			} else {
				out <- f
			}
		}
		return nil
	})

	chain := Chain(addExclaim, addQuestion)

	in := make(chan Frame, 1)
	out := make(chan Frame, 1)
	in <- NewTextFrame("hello")
	close(in)

	err := chain.Process(context.Background(), in, out)
	if err != nil {
		t.Fatalf("Chain() error = %v", err)
	}

	frame := <-out
	if frame.Text() != "hello!?" {
		t.Errorf("frame.Text() = %q, want %q", frame.Text(), "hello!?")
	}
}

func TestChainCancelContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	blocker := FrameProcessorFunc(func(ctx context.Context, in <-chan Frame, out chan<- Frame) error {
		defer close(out)
		<-ctx.Done()
		return ctx.Err()
	})

	chain := Chain(blocker)
	in := make(chan Frame)
	out := make(chan Frame, 1)

	done := make(chan error, 1)
	go func() {
		done <- chain.Process(ctx, in, out)
	}()

	cancel()

	err := <-done
	if err != context.Canceled {
		t.Errorf("error = %v, want context.Canceled", err)
	}
}

func TestChainErrorInMiddleProcessor(t *testing.T) {
	// When a middle processor fails, Chain should return the error.
	proc1 := FrameProcessorFunc(func(_ context.Context, in <-chan Frame, out chan<- Frame) error {
		defer close(out)
		for f := range in {
			out <- f
		}
		return nil
	})

	errProc := FrameProcessorFunc(func(_ context.Context, in <-chan Frame, out chan<- Frame) error {
		defer close(out)
		for range in {
			// drain input
		}
		return errors.New("middle processor failed")
	})

	proc3 := FrameProcessorFunc(func(_ context.Context, in <-chan Frame, out chan<- Frame) error {
		defer close(out)
		for f := range in {
			out <- f
		}
		return nil
	})

	chain := Chain(proc1, errProc, proc3)
	in := make(chan Frame, 1)
	out := make(chan Frame, 1)
	in <- NewTextFrame("test")
	close(in)

	err := chain.Process(context.Background(), in, out)
	if err == nil {
		t.Error("Chain should return error from failing processor")
	}
	if err.Error() != "middle processor failed" {
		t.Errorf("error = %q, want %q", err, "middle processor failed")
	}
}

func TestChainThreeProcessors(t *testing.T) {
	// Three-processor chain exercises the intermediate channel path (i > 0 && i < len-1).
	addA := FrameProcessorFunc(func(_ context.Context, in <-chan Frame, out chan<- Frame) error {
		defer close(out)
		for f := range in {
			if f.Type == FrameText {
				out <- NewTextFrame(f.Text() + "A")
			} else {
				out <- f
			}
		}
		return nil
	})

	addB := FrameProcessorFunc(func(_ context.Context, in <-chan Frame, out chan<- Frame) error {
		defer close(out)
		for f := range in {
			if f.Type == FrameText {
				out <- NewTextFrame(f.Text() + "B")
			} else {
				out <- f
			}
		}
		return nil
	})

	addC := FrameProcessorFunc(func(_ context.Context, in <-chan Frame, out chan<- Frame) error {
		defer close(out)
		for f := range in {
			if f.Type == FrameText {
				out <- NewTextFrame(f.Text() + "C")
			} else {
				out <- f
			}
		}
		return nil
	})

	chain := Chain(addA, addB, addC)
	in := make(chan Frame, 1)
	out := make(chan Frame, 1)
	in <- NewTextFrame("x")
	close(in)

	err := chain.Process(context.Background(), in, out)
	if err != nil {
		t.Fatalf("Chain() error = %v", err)
	}

	frame := <-out
	if frame.Text() != "xABC" {
		t.Errorf("frame.Text() = %q, want %q", frame.Text(), "xABC")
	}
}
