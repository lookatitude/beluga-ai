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
// A value of 0 (the default) uses the channel buffer provided by the store.
func WithBufferSize(size int) WatchOption {
	return func(o *watchOptions) {
		if size >= 0 {
			o.bufSize = size
		}
	}
}

// WatchSeq wraps Store.Watch as an iter.Seq2[StateChange, error] stream.
// The returned iterator yields state changes until the context is cancelled,
// the store is closed, or the consumer breaks out of the range loop.
func WatchSeq(ctx context.Context, store Store, key string, opts ...WatchOption) iter.Seq2[StateChange, error] {
	o := defaultWatchOptions()
	for _, opt := range opts {
		opt(&o)
	}
	_ = o // bufSize is informational; the store controls the channel buffer

	return func(yield func(StateChange, error) bool) {
		ch, err := store.Watch(ctx, key)
		if err != nil {
			yield(StateChange{}, err)
			return
		}

		for {
			select {
			case <-ctx.Done():
				yield(StateChange{}, ctx.Err())
				return
			case change, ok := <-ch:
				if !ok {
					// Channel closed — store was closed or context cancelled.
					return
				}
				if !yield(change, nil) {
					return
				}
			}
		}
	}
}
