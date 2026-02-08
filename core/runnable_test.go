package core

import (
	"context"
	"fmt"
	"iter"
	"testing"
)

// mockRunnable is a configurable Runnable for testing.
type mockRunnable struct {
	invokeFunc func(ctx context.Context, input any, opts ...Option) (any, error)
	streamFunc func(ctx context.Context, input any, opts ...Option) iter.Seq2[any, error]
}

func (m *mockRunnable) Invoke(ctx context.Context, input any, opts ...Option) (any, error) {
	if m.invokeFunc != nil {
		return m.invokeFunc(ctx, input, opts...)
	}
	return input, nil
}

func (m *mockRunnable) Stream(ctx context.Context, input any, opts ...Option) iter.Seq2[any, error] {
	if m.streamFunc != nil {
		return m.streamFunc(ctx, input, opts...)
	}
	return func(yield func(any, error) bool) {
		yield(input, nil)
	}
}

// transformRunnable returns a Runnable that transforms input by appending a suffix.
func transformRunnable(suffix string) *mockRunnable {
	return &mockRunnable{
		invokeFunc: func(_ context.Context, input any, _ ...Option) (any, error) {
			return fmt.Sprintf("%v%s", input, suffix), nil
		},
		streamFunc: func(_ context.Context, input any, _ ...Option) iter.Seq2[any, error] {
			return func(yield func(any, error) bool) {
				yield(fmt.Sprintf("%v%s", input, suffix), nil)
			}
		},
	}
}

// errorRunnable returns a Runnable that always returns an error.
func errorRunnable(err error) *mockRunnable {
	return &mockRunnable{
		invokeFunc: func(_ context.Context, _ any, _ ...Option) (any, error) {
			return nil, err
		},
		streamFunc: func(_ context.Context, _ any, _ ...Option) iter.Seq2[any, error] {
			return func(yield func(any, error) bool) {
				yield(nil, err)
			}
		},
	}
}

func TestPipe_Invoke(t *testing.T) {
	tests := []struct {
		name    string
		a       Runnable
		b       Runnable
		input   any
		want    any
		wantErr bool
		errMsg  string
	}{
		{
			name:  "chain_two_transforms",
			a:     transformRunnable("-A"),
			b:     transformRunnable("-B"),
			input: "start",
			want:  "start-A-B",
		},
		{
			name:    "first_errors",
			a:       errorRunnable(fmt.Errorf("first failed")),
			b:       transformRunnable("-B"),
			input:   "x",
			wantErr: true,
			errMsg:  "first failed",
		},
		{
			name:    "second_errors",
			a:       transformRunnable("-A"),
			b:       errorRunnable(fmt.Errorf("second failed")),
			input:   "x",
			wantErr: true,
			errMsg:  "second failed",
		},
		{
			name:  "nil_input",
			a:     &mockRunnable{},
			b:     &mockRunnable{},
			input: nil,
			want:  nil,
		},
		{
			name:  "passthrough",
			a:     &mockRunnable{},
			b:     &mockRunnable{},
			input: "hello",
			want:  "hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Pipe(tt.a, tt.b)
			got, err := p.Invoke(context.Background(), tt.input)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("Invoke() error = nil, want error containing %q", tt.errMsg)
				}
				if err.Error() != tt.errMsg {
					t.Errorf("Invoke() error = %q, want %q", err.Error(), tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Fatalf("Invoke() unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("Invoke() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPipe_Invoke_OptionsPassthrough(t *testing.T) {
	var aOpts, bOpts []Option

	a := &mockRunnable{
		invokeFunc: func(_ context.Context, input any, opts ...Option) (any, error) {
			aOpts = opts
			return input, nil
		},
	}
	b := &mockRunnable{
		invokeFunc: func(_ context.Context, input any, opts ...Option) (any, error) {
			bOpts = opts
			return input, nil
		},
	}

	opt1 := OptionFunc(func(_ any) {})
	opt2 := OptionFunc(func(_ any) {})

	p := Pipe(a, b)
	_, err := p.Invoke(context.Background(), "test", opt1, opt2)
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}

	if len(aOpts) != 2 {
		t.Errorf("a received %d opts, want 2", len(aOpts))
	}
	if len(bOpts) != 2 {
		t.Errorf("b received %d opts, want 2", len(bOpts))
	}
}

func TestPipe_Invoke_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	a := &mockRunnable{
		invokeFunc: func(ctx context.Context, _ any, _ ...Option) (any, error) {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			return "ok", nil
		},
	}
	b := transformRunnable("-B")

	p := Pipe(a, b)
	_, err := p.Invoke(ctx, "input")
	if err == nil {
		t.Fatal("Invoke() expected context cancellation error, got nil")
	}
}

func TestPipe_Stream(t *testing.T) {
	t.Run("streams_from_second", func(t *testing.T) {
		a := transformRunnable("-A")
		b := &mockRunnable{
			streamFunc: func(_ context.Context, input any, _ ...Option) iter.Seq2[any, error] {
				return func(yield func(any, error) bool) {
					yield(fmt.Sprintf("%v-1", input), nil)
					yield(fmt.Sprintf("%v-2", input), nil)
				}
			},
		}

		p := Pipe(a, b)
		var results []any
		for val, err := range p.Stream(context.Background(), "start") {
			if err != nil {
				t.Fatalf("Stream() unexpected error: %v", err)
			}
			results = append(results, val)
		}

		if len(results) != 2 {
			t.Fatalf("len(results) = %d, want 2", len(results))
		}
		if results[0] != "start-A-1" {
			t.Errorf("results[0] = %v, want %q", results[0], "start-A-1")
		}
		if results[1] != "start-A-2" {
			t.Errorf("results[1] = %v, want %q", results[1], "start-A-2")
		}
	})

	t.Run("first_invoke_error_yields_error", func(t *testing.T) {
		a := errorRunnable(fmt.Errorf("a failed"))
		b := transformRunnable("-B")

		p := Pipe(a, b)
		var gotErr error
		for _, err := range p.Stream(context.Background(), "x") {
			if err != nil {
				gotErr = err
				break
			}
		}
		if gotErr == nil {
			t.Fatal("Stream() expected error from first runnable, got nil")
		}
		if gotErr.Error() != "a failed" {
			t.Errorf("error = %q, want %q", gotErr.Error(), "a failed")
		}
	})

	t.Run("second_stream_error", func(t *testing.T) {
		a := &mockRunnable{}
		b := &mockRunnable{
			streamFunc: func(_ context.Context, _ any, _ ...Option) iter.Seq2[any, error] {
				return func(yield func(any, error) bool) {
					yield("ok", nil)
					yield(nil, fmt.Errorf("stream err"))
				}
			},
		}

		p := Pipe(a, b)
		var vals []any
		var gotErr error
		for val, err := range p.Stream(context.Background(), "in") {
			if err != nil {
				gotErr = err
				break
			}
			vals = append(vals, val)
		}

		if len(vals) != 1 {
			t.Errorf("received %d values before error, want 1", len(vals))
		}
		if gotErr == nil || gotErr.Error() != "stream err" {
			t.Errorf("error = %v, want %q", gotErr, "stream err")
		}
	})

	t.Run("early_break_stops_stream", func(t *testing.T) {
		count := 0
		a := &mockRunnable{}
		b := &mockRunnable{
			streamFunc: func(_ context.Context, _ any, _ ...Option) iter.Seq2[any, error] {
				return func(yield func(any, error) bool) {
					for i := 0; i < 100; i++ {
						count++
						if !yield(i, nil) {
							return
						}
					}
				}
			},
		}

		p := Pipe(a, b)
		for _, err := range p.Stream(context.Background(), "in") {
			if err != nil {
				break
			}
			break // Stop after first value.
		}

		// The stream should have stopped early.
		if count >= 100 {
			t.Errorf("stream produced %d values, expected early stop", count)
		}
	})
}

func TestPipe_Chaining(t *testing.T) {
	// Pipe(Pipe(a, b), c) should work.
	a := transformRunnable("-A")
	b := transformRunnable("-B")
	c := transformRunnable("-C")

	chain := Pipe(Pipe(a, b), c)
	got, err := chain.Invoke(context.Background(), "x")
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
	if got != "x-A-B-C" {
		t.Errorf("Invoke() = %v, want %q", got, "x-A-B-C")
	}
}

func TestParallel_Invoke(t *testing.T) {
	t.Run("all_succeed", func(t *testing.T) {
		r1 := transformRunnable("-1")
		r2 := transformRunnable("-2")
		r3 := transformRunnable("-3")

		p := Parallel(r1, r2, r3)
		got, err := p.Invoke(context.Background(), "in")
		if err != nil {
			t.Fatalf("Invoke() error = %v", err)
		}

		results, ok := got.([]any)
		if !ok {
			t.Fatalf("Invoke() result type = %T, want []any", got)
		}
		if len(results) != 3 {
			t.Fatalf("len(results) = %d, want 3", len(results))
		}
		if results[0] != "in-1" {
			t.Errorf("results[0] = %v, want %q", results[0], "in-1")
		}
		if results[1] != "in-2" {
			t.Errorf("results[1] = %v, want %q", results[1], "in-2")
		}
		if results[2] != "in-3" {
			t.Errorf("results[2] = %v, want %q", results[2], "in-3")
		}
	})

	t.Run("one_error_returns_error", func(t *testing.T) {
		r1 := transformRunnable("-ok")
		r2 := errorRunnable(fmt.Errorf("r2 failed"))
		r3 := transformRunnable("-ok")

		p := Parallel(r1, r2, r3)
		_, err := p.Invoke(context.Background(), "in")
		if err == nil {
			t.Fatal("Invoke() expected error, got nil")
		}
		if err.Error() != "r2 failed" {
			t.Errorf("error = %q, want %q", err.Error(), "r2 failed")
		}
	})

	t.Run("all_errors_returns_first", func(t *testing.T) {
		r1 := errorRunnable(fmt.Errorf("err1"))
		r2 := errorRunnable(fmt.Errorf("err2"))

		p := Parallel(r1, r2)
		_, err := p.Invoke(context.Background(), "in")
		if err == nil {
			t.Fatal("Invoke() expected error, got nil")
		}
		// Should return the first error in order.
		if err.Error() != "err1" {
			t.Errorf("error = %q, want %q", err.Error(), "err1")
		}
	})

	t.Run("no_runnables", func(t *testing.T) {
		p := Parallel()
		got, err := p.Invoke(context.Background(), "in")
		if err != nil {
			t.Fatalf("Invoke() error = %v", err)
		}
		results, ok := got.([]any)
		if !ok {
			t.Fatalf("result type = %T, want []any", got)
		}
		if len(results) != 0 {
			t.Errorf("len(results) = %d, want 0", len(results))
		}
	})

	t.Run("single_runnable", func(t *testing.T) {
		r := transformRunnable("-solo")
		p := Parallel(r)
		got, err := p.Invoke(context.Background(), "in")
		if err != nil {
			t.Fatalf("Invoke() error = %v", err)
		}
		results := got.([]any)
		if len(results) != 1 {
			t.Fatalf("len(results) = %d, want 1", len(results))
		}
		if results[0] != "in-solo" {
			t.Errorf("results[0] = %v, want %q", results[0], "in-solo")
		}
	})

	t.Run("nil_input", func(t *testing.T) {
		r := &mockRunnable{}
		p := Parallel(r)
		got, err := p.Invoke(context.Background(), nil)
		if err != nil {
			t.Fatalf("Invoke() error = %v", err)
		}
		results := got.([]any)
		if results[0] != nil {
			t.Errorf("results[0] = %v, want nil", results[0])
		}
	})

	t.Run("options_passthrough", func(t *testing.T) {
		var received []Option
		r := &mockRunnable{
			invokeFunc: func(_ context.Context, input any, opts ...Option) (any, error) {
				received = opts
				return input, nil
			},
		}

		opt := OptionFunc(func(_ any) {})
		p := Parallel(r)
		_, err := p.Invoke(context.Background(), "in", opt)
		if err != nil {
			t.Fatalf("Invoke() error = %v", err)
		}
		if len(received) != 1 {
			t.Errorf("runnable received %d opts, want 1", len(received))
		}
	})
}

func TestParallel_Invoke_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	r := &mockRunnable{
		invokeFunc: func(ctx context.Context, _ any, _ ...Option) (any, error) {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			return "ok", nil
		},
	}

	p := Parallel(r)
	_, err := p.Invoke(ctx, "in")
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
}

func TestParallel_Stream(t *testing.T) {
	t.Run("yields_result_slice", func(t *testing.T) {
		r1 := transformRunnable("-1")
		r2 := transformRunnable("-2")

		p := Parallel(r1, r2)
		var results []any
		var gotErr error
		for val, err := range p.Stream(context.Background(), "in") {
			if err != nil {
				gotErr = err
				break
			}
			results = append(results, val)
		}

		if gotErr != nil {
			t.Fatalf("Stream() error = %v", gotErr)
		}
		if len(results) != 1 {
			t.Fatalf("Stream yielded %d values, want 1", len(results))
		}

		slice, ok := results[0].([]any)
		if !ok {
			t.Fatalf("result type = %T, want []any", results[0])
		}
		if len(slice) != 2 {
			t.Fatalf("len(slice) = %d, want 2", len(slice))
		}
		if slice[0] != "in-1" {
			t.Errorf("slice[0] = %v, want %q", slice[0], "in-1")
		}
		if slice[1] != "in-2" {
			t.Errorf("slice[1] = %v, want %q", slice[1], "in-2")
		}
	})

	t.Run("error_propagates", func(t *testing.T) {
		r1 := transformRunnable("-ok")
		r2 := errorRunnable(fmt.Errorf("parallel err"))

		p := Parallel(r1, r2)
		var gotErr error
		for _, err := range p.Stream(context.Background(), "in") {
			if err != nil {
				gotErr = err
				break
			}
		}
		if gotErr == nil || gotErr.Error() != "parallel err" {
			t.Errorf("error = %v, want %q", gotErr, "parallel err")
		}
	})
}

func TestParallel_Concurrent_Execution(t *testing.T) {
	// Verify that runnables actually execute concurrently by using channels.
	started := make(chan struct{}, 2)
	proceed := make(chan struct{})

	r1 := &mockRunnable{
		invokeFunc: func(_ context.Context, _ any, _ ...Option) (any, error) {
			started <- struct{}{}
			<-proceed
			return "r1", nil
		},
	}
	r2 := &mockRunnable{
		invokeFunc: func(_ context.Context, _ any, _ ...Option) (any, error) {
			started <- struct{}{}
			<-proceed
			return "r2", nil
		},
	}

	p := Parallel(r1, r2)

	done := make(chan struct{})
	go func() {
		defer close(done)
		p.Invoke(context.Background(), "in")
	}()

	// Both should start before either proceeds.
	<-started
	<-started
	close(proceed)
	<-done
}

// Verify Runnable interface compliance at compile time.
func TestRunnable_InterfaceCompliance(t *testing.T) {
	var _ Runnable = &mockRunnable{}
	var _ Runnable = Pipe(&mockRunnable{}, &mockRunnable{})
	var _ Runnable = Parallel(&mockRunnable{})
}
