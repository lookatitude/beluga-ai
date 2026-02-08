package voice

import (
	"context"
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
