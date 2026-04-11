package voice

import (
	"context"
	"errors"
	"iter"
	"testing"
)

// framesFromSlice returns an iter.Seq2 that yields the given frames with nil errors.
func framesFromSlice(frames ...Frame) iter.Seq2[Frame, error] {
	return func(yield func(Frame, error) bool) {
		for _, f := range frames {
			if !yield(f, nil) {
				return
			}
		}
	}
}

// collectFrames drains an iter.Seq2[Frame, error], returning collected frames and
// the first error encountered (if any).
func collectFrames(stream iter.Seq2[Frame, error]) ([]Frame, error) {
	var frames []Frame
	for f, err := range stream {
		if err != nil {
			return frames, err
		}
		frames = append(frames, f)
	}
	return frames, nil
}

func TestFrameProcessorFunc(t *testing.T) {
	called := false
	f := FrameProcessorFunc(func(_ context.Context, in iter.Seq2[Frame, error]) iter.Seq2[Frame, error] {
		called = true
		return in
	})

	out := f.Process(context.Background(), framesFromSlice(NewTextFrame("test")))
	frames, err := collectFrames(out)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if !called {
		t.Error("FrameProcessorFunc was not called")
	}
	if len(frames) != 1 || frames[0].Text() != "test" {
		t.Errorf("frames = %v, want one 'test' frame", frames)
	}
}

func TestChainEmpty(t *testing.T) {
	chain := Chain()
	out := chain.Process(context.Background(), framesFromSlice(NewTextFrame("passthrough")))
	frames, err := collectFrames(out)
	if err != nil {
		t.Fatalf("Chain() error = %v", err)
	}
	if len(frames) != 1 || frames[0].Text() != "passthrough" {
		t.Errorf("frames = %v, want one 'passthrough' frame", frames)
	}
}

// suffixProcessor appends a suffix to each text frame; non-text frames pass through.
func suffixProcessor(suffix string) FrameProcessor {
	return FrameLoop(func(_ context.Context, f Frame) ([]Frame, error) {
		if f.Type == FrameText {
			return []Frame{NewTextFrame(f.Text() + suffix)}, nil
		}
		return []Frame{f}, nil
	})
}

func TestChainSingle(t *testing.T) {
	chain := Chain(suffixProcessor("!"))
	out := chain.Process(context.Background(), framesFromSlice(NewTextFrame("hello")))
	frames, err := collectFrames(out)
	if err != nil {
		t.Fatalf("Chain() error = %v", err)
	}
	if len(frames) != 1 || frames[0].Text() != "hello!" {
		t.Errorf("frames = %v, want one 'hello!' frame", frames)
	}
}

func TestChainMultiple(t *testing.T) {
	chain := Chain(suffixProcessor("!"), suffixProcessor("?"))
	out := chain.Process(context.Background(), framesFromSlice(NewTextFrame("hello")))
	frames, err := collectFrames(out)
	if err != nil {
		t.Fatalf("Chain() error = %v", err)
	}
	if len(frames) != 1 || frames[0].Text() != "hello!?" {
		t.Errorf("frames = %v, want one 'hello!?' frame", frames)
	}
}

func TestChainCancelContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// A source that blocks forever unless ctx cancels.
	blockingSource := iter.Seq2[Frame, error](func(yield func(Frame, error) bool) {
		<-ctx.Done()
		yield(Frame{}, ctx.Err())
	})

	// Identity processor — will propagate the source's error.
	chain := Chain(FrameProcessorFunc(func(_ context.Context, in iter.Seq2[Frame, error]) iter.Seq2[Frame, error] {
		return in
	}))

	done := make(chan error, 1)
	go func() {
		_, err := collectFrames(chain.Process(ctx, blockingSource))
		done <- err
	}()

	cancel()

	err := <-done
	if err != context.Canceled {
		t.Errorf("error = %v, want context.Canceled", err)
	}
}

func TestChainErrorInMiddleProcessor(t *testing.T) {
	// A middle processor that errors after consuming its input.
	errProc := FrameProcessorFunc(func(_ context.Context, in iter.Seq2[Frame, error]) iter.Seq2[Frame, error] {
		return func(yield func(Frame, error) bool) {
			for _, err := range in {
				if err != nil {
					yield(Frame{}, err)
					return
				}
				// drain
			}
			yield(Frame{}, errors.New("middle processor failed"))
		}
	})

	// Passthroughs on either side.
	pass := FrameProcessorFunc(func(_ context.Context, in iter.Seq2[Frame, error]) iter.Seq2[Frame, error] {
		return in
	})

	chain := Chain(pass, errProc, pass)
	_, err := collectFrames(chain.Process(context.Background(), framesFromSlice(NewTextFrame("test"))))
	if err == nil {
		t.Error("Chain should return error from failing processor")
	}
	if err != nil && err.Error() != "middle processor failed" {
		t.Errorf("error = %q, want %q", err, "middle processor failed")
	}
}

func TestChainThreeProcessors(t *testing.T) {
	chain := Chain(suffixProcessor("A"), suffixProcessor("B"), suffixProcessor("C"))
	out := chain.Process(context.Background(), framesFromSlice(NewTextFrame("x")))
	frames, err := collectFrames(out)
	if err != nil {
		t.Fatalf("Chain() error = %v", err)
	}
	if len(frames) != 1 || frames[0].Text() != "xABC" {
		t.Errorf("frames = %v, want one 'xABC' frame", frames)
	}
}

// framesWithError yields the given frames and then terminates with err.
func framesWithError(err error, frames ...Frame) iter.Seq2[Frame, error] {
	return func(yield func(Frame, error) bool) {
		for _, f := range frames {
			if !yield(f, nil) {
				return
			}
		}
		yield(Frame{}, err)
	}
}

func TestFrameLoop_PropagatesInputError(t *testing.T) {
	wantErr := errors.New("upstream boom")
	loop := FrameLoop(func(_ context.Context, f Frame) ([]Frame, error) {
		return []Frame{f}, nil
	})
	out := loop.Process(context.Background(), framesWithError(wantErr, NewTextFrame("a")))
	frames, err := collectFrames(out)
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want %v", err, wantErr)
	}
	if len(frames) != 1 {
		t.Errorf("got %d frames before error, want 1", len(frames))
	}
}

func TestFrameLoop_PropagatesHandlerError(t *testing.T) {
	wantErr := errors.New("handler boom")
	loop := FrameLoop(func(_ context.Context, _ Frame) ([]Frame, error) {
		return nil, wantErr
	})
	out := loop.Process(context.Background(), framesFromSlice(NewTextFrame("a")))
	_, err := collectFrames(out)
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want %v", err, wantErr)
	}
}

func TestFrameLoop_RespectsCtxCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	loop := FrameLoop(func(_ context.Context, f Frame) ([]Frame, error) {
		return []Frame{f}, nil
	})
	out := loop.Process(ctx, framesFromSlice(NewTextFrame("a")))
	_, err := collectFrames(out)
	if err == nil {
		t.Error("expected ctx.Err() to be yielded, got nil")
	}
}
