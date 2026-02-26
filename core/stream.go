package core

import (
	"context"
	"iter"
	"sync"
)

// EventType identifies the kind of event flowing through a stream.
type EventType string

const (
	// EventData carries a payload chunk (e.g. text token, audio frame).
	EventData EventType = "data"

	// EventToolCall signals that the LLM is requesting a tool invocation.
	EventToolCall EventType = "tool_call"

	// EventToolResult carries the result of a tool invocation.
	EventToolResult EventType = "tool_result"

	// EventHandoff signals an agent-to-agent transfer.
	EventHandoff EventType = "handoff"

	// EventDone signals the end of the stream.
	EventDone EventType = "done"

	// EventError signals an error within the stream.
	EventError EventType = "error"
)

// Event is the unit of data flowing through the system. It carries a typed
// payload, an optional error, and arbitrary metadata such as trace IDs,
// latency measurements, or token counts.
type Event[T any] struct {
	// Type identifies the kind of event.
	Type EventType

	// Payload is the event data. Its concrete type depends on Type.
	Payload T

	// Err carries an error for EventError events.
	Err error

	// Meta holds supplementary key-value pairs (trace ID, latency, etc.).
	Meta map[string]any
}

// Stream is a pull-based event iterator built on Go 1.23+ iter.Seq2.
// Consumers use range to iterate:
//
//	for event, err := range stream {
//	    if err != nil { break }
//	    // handle event
//	}
type Stream[T any] = iter.Seq2[Event[T], error]

// CollectStream drains a Stream into a slice, returning all events and the
// first error encountered (if any).
func CollectStream[T any](stream Stream[T]) ([]Event[T], error) {
	var events []Event[T]
	for event, err := range stream {
		if err != nil {
			return events, err
		}
		events = append(events, event)
	}
	return events, nil
}

// MapStream transforms each event in src by applying fn. If fn returns an
// error the mapped stream yields that error and stops.
func MapStream[T, U any](src Stream[T], fn func(Event[T]) (Event[U], error)) Stream[U] {
	return func(yield func(Event[U], error) bool) {
		for event, err := range src {
			if err != nil {
				yield(Event[U]{}, err)
				return
			}
			mapped, mapErr := fn(event)
			if mapErr != nil {
				yield(Event[U]{}, mapErr)
				return
			}
			if !yield(mapped, nil) {
				return
			}
		}
	}
}

// FilterStream returns a Stream that only yields events for which keep
// returns true.
func FilterStream[T any](src Stream[T], keep func(Event[T]) bool) Stream[T] {
	return func(yield func(Event[T], error) bool) {
		for event, err := range src {
			if err != nil {
				yield(Event[T]{}, err)
				return
			}
			if keep(event) {
				if !yield(event, nil) {
					return
				}
			}
		}
	}
}

// MergeStreams merges multiple streams into a single stream. Events from all
// input streams are interleaved in arrival order. The merged stream completes
// when all input streams are exhausted.
func MergeStreams[T any](ctx context.Context, streams ...Stream[T]) Stream[T] {
	return func(yield func(Event[T], error) bool) {
		ch := make(chan eventOrErr[T], len(streams))
		var wg sync.WaitGroup
		wg.Add(len(streams))

		for _, s := range streams {
			go mergeStreamProducer(ctx, s, ch, &wg)
		}

		go func() {
			wg.Wait()
			close(ch)
		}()

		drainChannel(ch, yield)
	}
}

// mergeStreamProducer reads events from a single stream and sends them to ch.
func mergeStreamProducer[T any](ctx context.Context, s Stream[T], ch chan<- eventOrErr[T], wg *sync.WaitGroup) {
	defer wg.Done()
	for event, err := range s {
		select {
		case <-ctx.Done():
			return
		case ch <- eventOrErr[T]{event: event, err: err}:
			if err != nil {
				return
			}
		}
	}
}

// drainChannel yields items from ch until the channel closes or yield returns false.
func drainChannel[T any](ch <-chan eventOrErr[T], yield func(Event[T], error) bool) {
	for item := range ch {
		if !yield(item.event, item.err) {
			return
		}
		if item.err != nil {
			return
		}
	}
}

// eventOrErr bundles an event and its associated error for channel transport.
type eventOrErr[T any] struct {
	event Event[T]
	err   error
}

// FanOut copies a single stream to n consumers. Each consumer receives all
// events independently. The returned slice has n streams.
func FanOut[T any](ctx context.Context, src Stream[T], n int) []Stream[T] {
	chs := make([]chan eventOrErr[T], n)
	for i := range chs {
		chs[i] = make(chan eventOrErr[T], 16)
	}

	go fanOutProducer(ctx, src, chs)

	streams := make([]Stream[T], n)
	for i := range chs {
		streams[i] = channelToStream(chs[i])
	}
	return streams
}

// fanOutProducer reads events from src and broadcasts them to all channels.
func fanOutProducer[T any](ctx context.Context, src Stream[T], chs []chan eventOrErr[T]) {
	defer func() {
		for _, ch := range chs {
			close(ch)
		}
	}()
	for event, err := range src {
		item := eventOrErr[T]{event: event, err: err}
		for _, ch := range chs {
			select {
			case <-ctx.Done():
				return
			case ch <- item:
			}
		}
		if err != nil {
			return
		}
	}
}

// channelToStream converts a channel of eventOrErr into a Stream.
func channelToStream[T any](ch <-chan eventOrErr[T]) Stream[T] {
	return func(yield func(Event[T], error) bool) {
		for item := range ch {
			if !yield(item.event, item.err) {
				return
			}
			if item.err != nil {
				return
			}
		}
	}
}

// BufferedStream wraps a producer stream with an internal channel buffer to
// absorb bursts between a fast producer and a slow consumer. The buffer size
// controls the backpressure threshold.
type BufferedStream[T any] struct {
	ch   chan eventOrErr[T]
	done chan struct{}
	once sync.Once
}

// NewBufferedStream starts consuming src into an internal buffer of the given
// size and returns a BufferedStream that can be iterated. Cancel ctx to stop
// the background goroutine.
func NewBufferedStream[T any](ctx context.Context, src Stream[T], bufSize int) *BufferedStream[T] {
	if bufSize < 1 {
		bufSize = 1
	}
	bs := &BufferedStream[T]{
		ch:   make(chan eventOrErr[T], bufSize),
		done: make(chan struct{}),
	}

	go func() {
		defer close(bs.ch)
		defer close(bs.done)
		for event, err := range src {
			select {
			case <-ctx.Done():
				return
			case bs.ch <- eventOrErr[T]{event: event, err: err}:
			}
			if err != nil {
				return
			}
		}
	}()

	return bs
}

// Iter returns an iter.Seq2 that drains the buffered stream. It is safe to
// call Iter only once.
func (bs *BufferedStream[T]) Iter() Stream[T] {
	return func(yield func(Event[T], error) bool) {
		for item := range bs.ch {
			if !yield(item.event, item.err) {
				return
			}
			if item.err != nil {
				return
			}
		}
	}
}

// Len returns the current number of buffered events.
func (bs *BufferedStream[T]) Len() int {
	return len(bs.ch)
}

// Cap returns the buffer capacity.
func (bs *BufferedStream[T]) Cap() int {
	return cap(bs.ch)
}

// FlowController provides backpressure control for streams. It allows
// producers to pause/resume based on consumer readiness and can be queried
// for current buffer state.
type FlowController interface {
	// Acquire blocks until capacity is available or ctx is cancelled.
	Acquire(ctx context.Context) error

	// Release signals that capacity has been freed.
	Release()

	// TryAcquire attempts to acquire without blocking. Returns true if
	// successful, false if no capacity is currently available.
	TryAcquire() bool
}

// semaphoreFlowController implements FlowController using a semaphore pattern.
type semaphoreFlowController struct {
	sem chan struct{}
}

// NewFlowController creates a FlowController with the given maximum
// concurrency. maxConcurrency must be at least 1.
func NewFlowController(maxConcurrency int) FlowController {
	if maxConcurrency < 1 {
		maxConcurrency = 1
	}
	return &semaphoreFlowController{
		sem: make(chan struct{}, maxConcurrency),
	}
}

// Acquire blocks until capacity is available or ctx is cancelled.
func (s *semaphoreFlowController) Acquire(ctx context.Context) error {
	select {
	case s.sem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Release signals that capacity has been freed.
func (s *semaphoreFlowController) Release() {
	select {
	case <-s.sem:
	default:
		// Channel is empty; this is a programming error but we don't panic.
	}
}

// TryAcquire attempts to acquire without blocking.
func (s *semaphoreFlowController) TryAcquire() bool {
	select {
	case s.sem <- struct{}{}:
		return true
	default:
		return false
	}
}
