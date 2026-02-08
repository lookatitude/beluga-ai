package voice

import "context"

// FrameProcessor processes a stream of frames. Each processor is a goroutine
// connected by channels, reading frames from in and writing results to out.
// The processor must return when ctx is cancelled or in is closed.
type FrameProcessor interface {
	// Process reads frames from in, processes them, and writes results to out.
	// It must close out when done (either because in was closed or ctx cancelled).
	// Returning a non-nil error indicates a fatal processing failure.
	Process(ctx context.Context, in <-chan Frame, out chan<- Frame) error
}

// FrameProcessorFunc is an adapter to allow the use of ordinary functions as
// FrameProcessors. If f is a function with the appropriate signature,
// FrameProcessorFunc(f) is a FrameProcessor that calls f.
type FrameProcessorFunc func(ctx context.Context, in <-chan Frame, out chan<- Frame) error

// Process calls f(ctx, in, out).
func (f FrameProcessorFunc) Process(ctx context.Context, in <-chan Frame, out chan<- Frame) error {
	return f(ctx, in, out)
}

// Chain connects multiple FrameProcessors in series. Frames flow from the
// first processor to the last. Returns a single FrameProcessor representing
// the full pipeline.
func Chain(processors ...FrameProcessor) FrameProcessor {
	if len(processors) == 0 {
		return FrameProcessorFunc(func(_ context.Context, in <-chan Frame, out chan<- Frame) error {
			defer close(out)
			for f := range in {
				out <- f
			}
			return nil
		})
	}
	if len(processors) == 1 {
		return processors[0]
	}
	return FrameProcessorFunc(func(ctx context.Context, in <-chan Frame, out chan<- Frame) error {
		// Create intermediate channels between processors.
		channels := make([]chan Frame, len(processors)-1)
		for i := range channels {
			channels[i] = make(chan Frame, 64)
		}

		errc := make(chan error, len(processors))

		// Start all processors.
		for i, p := range processors {
			var pIn <-chan Frame
			var pOut chan<- Frame

			if i == 0 {
				pIn = in
				pOut = channels[0]
			} else if i == len(processors)-1 {
				pIn = channels[i-1]
				pOut = out
			} else {
				pIn = channels[i-1]
				pOut = channels[i]
			}

			go func(proc FrameProcessor, procIn <-chan Frame, procOut chan<- Frame) {
				errc <- proc.Process(ctx, procIn, procOut)
			}(p, pIn, pOut)
		}

		// Wait for all processors to finish, return first error.
		var firstErr error
		for range processors {
			if err := <-errc; err != nil && firstErr == nil {
				firstErr = err
			}
		}
		return firstErr
	})
}
