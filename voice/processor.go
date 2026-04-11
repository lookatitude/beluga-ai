package voice

import (
	"context"
	"iter"
)

// FrameProcessor processes a stream of frames. Each processor is a pure
// transformer over an iter.Seq2[Frame, error] stream: it consumes input frames
// lazily and yields output frames to its caller. A fatal error is delivered by
// yielding a zero Frame paired with the error and then ending the iterator.
//
// Implementations must respect ctx cancellation by ending the iterator when
// ctx.Done() fires (typically by propagating cancellation from the input
// iterator or by checking ctx at yield points).
type FrameProcessor interface {
	// Process returns an output stream derived from the input stream.
	// The returned iterator is single-use and safe for exactly one consumer.
	Process(ctx context.Context, in iter.Seq2[Frame, error]) iter.Seq2[Frame, error]
}

// FrameProcessorFunc is an adapter to allow the use of ordinary functions as
// FrameProcessors. If f is a function with the appropriate signature,
// FrameProcessorFunc(f) is a FrameProcessor that calls f.
type FrameProcessorFunc func(ctx context.Context, in iter.Seq2[Frame, error]) iter.Seq2[Frame, error]

// Process calls f(ctx, in).
func (f FrameProcessorFunc) Process(ctx context.Context, in iter.Seq2[Frame, error]) iter.Seq2[Frame, error] {
	return f(ctx, in)
}

// passthroughProcessor returns a FrameProcessor that forwards all frames unchanged.
func passthroughProcessor() FrameProcessor {
	return FrameProcessorFunc(func(_ context.Context, in iter.Seq2[Frame, error]) iter.Seq2[Frame, error] {
		return in
	})
}

// FrameHandler processes a single input frame and returns zero or more output
// frames. A non-nil error terminates the surrounding FrameLoop as a fatal
// error. This shape matches the existing STT / TTS / VAD handler bodies which
// emit at most a small, fixed number of frames per input.
type FrameHandler func(ctx context.Context, frame Frame) ([]Frame, error)

// FrameLoop creates a FrameProcessor that reads frames from the input iterator,
// calls handler for each one, and yields any resulting frames to the output
// iterator. This is the standard read-process-write loop used by STT, TTS,
// and similar processors.
//
// The loop terminates when:
//   - the input iterator ends,
//   - ctx is cancelled,
//   - the input iterator yields a non-nil error (propagated and then end),
//   - handler returns a non-nil error (yielded and then end), or
//   - the downstream consumer stops pulling.
func FrameLoop(handler FrameHandler) FrameProcessor {
	return FrameProcessorFunc(func(ctx context.Context, in iter.Seq2[Frame, error]) iter.Seq2[Frame, error] {
		return func(yield func(Frame, error) bool) {
			for frame, err := range in {
				if !handleOneFrame(ctx, handler, frame, err, yield) {
					return
				}
			}
		}
	})
}

// handleOneFrame processes a single input frame and returns true if the loop
// should continue, false if it should terminate. It encapsulates error
// propagation, context checking, handler invocation, and output yielding so
// that FrameLoop itself remains trivially simple.
func handleOneFrame(
	ctx context.Context,
	handler FrameHandler,
	frame Frame,
	inErr error,
	yield func(Frame, error) bool,
) bool {
	if inErr != nil {
		yield(Frame{}, inErr)
		return false
	}
	if ctx.Err() != nil {
		yield(Frame{}, ctx.Err())
		return false
	}
	outFrames, herr := handler(ctx, frame)
	if herr != nil {
		yield(Frame{}, herr)
		return false
	}
	for _, f := range outFrames {
		if !yield(f, nil) {
			return false
		}
	}
	return true
}

// Chain connects multiple FrameProcessors in series. Frames flow from the
// first processor to the last. Returns a single FrameProcessor representing
// the full pipeline.
//
// Composition is lazy: each stage's output iterator becomes the next stage's
// input iterator, so no intermediate buffered channels or goroutines are
// created. A fatal error at any stage propagates to the final output.
func Chain(processors ...FrameProcessor) FrameProcessor {
	if len(processors) == 0 {
		return passthroughProcessor()
	}
	if len(processors) == 1 {
		return processors[0]
	}
	return FrameProcessorFunc(func(ctx context.Context, in iter.Seq2[Frame, error]) iter.Seq2[Frame, error] {
		stream := in
		for _, p := range processors {
			stream = p.Process(ctx, stream)
		}
		return stream
	})
}
