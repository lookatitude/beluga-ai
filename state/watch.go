package state

import (
	"context"
	"iter"
)

// WatchOption configures the behavior of WatchSeq.
type WatchOption func(*watchOptions)

type watchOptions struct {
	bufSize int
}

func defaultWatchOptions() watchOptions {
	return watchOptions{bufSize: 0}
}

// WithBufferSize sets the internal buffer size for the watch adapter.
// A value of 0 (the default) passes events through directly. A value > 0
// relays events through an internal buffered channel so that brief stalls
// in the consumer do not back-pressure the store's watch stream.
func WithBufferSize(size int) WatchOption {
	return func(o *watchOptions) {
		if size >= 0 {
			o.bufSize = size
		}
	}
}

// WatchSeq is an adapter around Store.Watch that optionally relays events
// through an internal buffer. With the default (bufSize == 0) it is a thin
// passthrough to store.Watch(ctx, key). This function is retained for
// backward compatibility; new code should call store.Watch directly.
func WatchSeq(ctx context.Context, store Store, key string, opts ...WatchOption) iter.Seq2[StateChange, error] {
	o := defaultWatchOptions()
	for _, opt := range opts {
		opt(&o)
	}

	inner := store.Watch(ctx, key)
	if o.bufSize <= 0 {
		return inner
	}

	// Relay events through an internal buffer to absorb brief consumer stalls.
	return func(yield func(StateChange, error) bool) {
		relayCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		type item struct {
			change StateChange
			err    error
		}
		relay := make(chan item, o.bufSize)

		go func() {
			defer close(relay)
			for change, err := range inner {
				select {
				case relay <- item{change: change, err: err}:
				case <-relayCtx.Done():
					return
				}
				if err != nil {
					return
				}
			}
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case it, ok := <-relay:
				if !ok {
					return
				}
				if !yield(it.change, it.err) {
					return
				}
			}
		}
	}
}
