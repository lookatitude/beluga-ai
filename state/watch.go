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
//
// If WithBufferSize(n) is provided with n > 0, events are relayed through
// an internal n-slot buffer so that brief stalls in the consumer do not
// back-pressure the store's watch channel. If n == 0 the store channel is
// consumed directly.
func WatchSeq(ctx context.Context, store Store, key string, opts ...WatchOption) iter.Seq2[StateChange, error] {
	o := defaultWatchOptions()
	for _, opt := range opts {
		opt(&o)
	}

	return func(yield func(StateChange, error) bool) {
		ch, err := store.Watch(ctx, key)
		if err != nil {
			yield(StateChange{}, err)
			return
		}

		// Zero buffer: consume the store channel directly.
		if o.bufSize <= 0 {
			for {
				select {
				case <-ctx.Done():
					yield(StateChange{}, ctx.Err())
					return
				case change, ok := <-ch:
					if !ok {
						return
					}
					if !yield(change, nil) {
						return
					}
				}
			}
		}

		// Relay events through an internal buffer to absorb brief consumer stalls.
		relayCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		relay := make(chan StateChange, o.bufSize)
		go func() {
			defer close(relay)
			for {
				select {
				case <-relayCtx.Done():
					return
				case change, ok := <-ch:
					if !ok {
						return
					}
					select {
					case relay <- change:
					case <-relayCtx.Done():
						return
					}
				}
			}
		}()

		for {
			select {
			case <-ctx.Done():
				yield(StateChange{}, ctx.Err())
				return
			case change, ok := <-relay:
				if !ok {
					return
				}
				if !yield(change, nil) {
					return
				}
			}
		}
	}
}
