package hookutil_test

import (
	"context"
	"errors"
	"testing"

	"github.com/lookatitude/beluga-ai/internal/hookutil"
)

// testHook is a simple hook struct used across tests.
type testHook struct {
	fn0   func(context.Context) error
	fn1   func(context.Context, string) error
	fn2   func(context.Context, string, int) error
	fn3   func(context.Context, string, int, bool) error
	onErr func(context.Context, error) error
	v0    func(context.Context)
	v1    func(context.Context, string)
	v2    func(context.Context, string, int)
	v3    func(context.Context, string, int, bool)
	v4    func(context.Context, string, int, bool, float64)
}

var ctx = context.Background()

// ---- ComposeErrorPassthrough ----

func TestComposeErrorPassthrough_NoHooks(t *testing.T) {
	fn := hookutil.ComposeErrorPassthrough([]testHook{}, func(h testHook) func(context.Context, error) error { return h.onErr })
	sentinel := errors.New("original")
	if got := fn(ctx, sentinel); got != sentinel {
		t.Fatalf("expected original error, got %v", got)
	}
}

func TestComposeErrorPassthrough_NilField(t *testing.T) {
	hooks := []testHook{{onErr: nil}}
	fn := hookutil.ComposeErrorPassthrough(hooks, func(h testHook) func(context.Context, error) error { return h.onErr })
	sentinel := errors.New("original")
	if got := fn(ctx, sentinel); got != sentinel {
		t.Fatalf("expected original error, got %v", got)
	}
}

func TestComposeErrorPassthrough_HookReturnsNil(t *testing.T) {
	hooks := []testHook{{onErr: func(_ context.Context, _ error) error { return nil }}}
	fn := hookutil.ComposeErrorPassthrough(hooks, func(h testHook) func(context.Context, error) error { return h.onErr })
	sentinel := errors.New("original")
	if got := fn(ctx, sentinel); got != sentinel {
		t.Fatalf("expected original error passthrough, got %v", got)
	}
}

func TestComposeErrorPassthrough_HookReplacesError(t *testing.T) {
	replacement := errors.New("replaced")
	hooks := []testHook{{onErr: func(_ context.Context, _ error) error { return replacement }}}
	fn := hookutil.ComposeErrorPassthrough(hooks, func(h testHook) func(context.Context, error) error { return h.onErr })
	if got := fn(ctx, errors.New("original")); got != replacement {
		t.Fatalf("expected replaced error, got %v", got)
	}
}

func TestComposeErrorPassthrough_ShortCircuitsOnFirstNonNil(t *testing.T) {
	calls := 0
	first := errors.New("first")
	hooks := []testHook{
		{onErr: func(_ context.Context, _ error) error { calls++; return first }},
		{onErr: func(_ context.Context, _ error) error { calls++; return errors.New("second") }},
	}
	fn := hookutil.ComposeErrorPassthrough(hooks, func(h testHook) func(context.Context, error) error { return h.onErr })
	if got := fn(ctx, errors.New("original")); got != first {
		t.Fatalf("expected first error, got %v", got)
	}
	if calls != 1 {
		t.Fatalf("expected 1 hook call, got %d", calls)
	}
}

// ---- ComposeErrorPassthrough1 ----

func TestComposeErrorPassthrough1_Passthrough(t *testing.T) {
	hooks := []testHook{{}}
	fn := hookutil.ComposeErrorPassthrough1(hooks, func(h testHook) func(context.Context, string, error) error { return nil })
	sentinel := errors.New("original")
	if got := fn(ctx, "key", sentinel); got != sentinel {
		t.Fatalf("expected original error, got %v", got)
	}
}

func TestComposeErrorPassthrough1_Replaces(t *testing.T) {
	replacement := errors.New("replaced")
	hooks := []testHook{{}}
	// use a closure to capture replacement
	var called string
	hooks[0].fn1 = nil // not used here; wire directly
	type h2 struct{ fn func(context.Context, string, error) error }
	h2hooks := []h2{{fn: func(_ context.Context, s string, _ error) error { called = s; return replacement }}}
	fn := hookutil.ComposeErrorPassthrough1(h2hooks, func(h h2) func(context.Context, string, error) error { return h.fn })
	if got := fn(ctx, "name", errors.New("orig")); got != replacement {
		t.Fatalf("expected replacement, got %v", got)
	}
	if called != "name" {
		t.Fatalf("expected called=name, got %q", called)
	}
}

// ---- ComposeError0 ----

func TestComposeError0_AllNil(t *testing.T) {
	hooks := []testHook{{fn0: nil}, {fn0: nil}}
	fn := hookutil.ComposeError0(hooks, func(h testHook) func(context.Context) error { return h.fn0 })
	if err := fn(ctx); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestComposeError0_ShortCircuits(t *testing.T) {
	sentinel := errors.New("stop")
	calls := 0
	hooks := []testHook{
		{fn0: func(_ context.Context) error { calls++; return sentinel }},
		{fn0: func(_ context.Context) error { calls++; return nil }},
	}
	fn := hookutil.ComposeError0(hooks, func(h testHook) func(context.Context) error { return h.fn0 })
	if err := fn(ctx); err != sentinel {
		t.Fatalf("expected sentinel, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

// ---- ComposeError1 ----

func TestComposeError1_CallsAll(t *testing.T) {
	var got []string
	hooks := []testHook{
		{fn1: func(_ context.Context, s string) error { got = append(got, "a:"+s); return nil }},
		{fn1: func(_ context.Context, s string) error { got = append(got, "b:"+s); return nil }},
	}
	fn := hookutil.ComposeError1(hooks, func(h testHook) func(context.Context, string) error { return h.fn1 })
	if err := fn(ctx, "x"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 || got[0] != "a:x" || got[1] != "b:x" {
		t.Fatalf("unexpected calls: %v", got)
	}
}

// ---- ComposeError2 ----

func TestComposeError2_ShortCircuits(t *testing.T) {
	sentinel := errors.New("stop")
	calls := 0
	hooks := []testHook{
		{fn2: func(_ context.Context, _ string, _ int) error { calls++; return sentinel }},
		{fn2: func(_ context.Context, _ string, _ int) error { calls++; return nil }},
	}
	fn := hookutil.ComposeError2(hooks, func(h testHook) func(context.Context, string, int) error { return h.fn2 })
	if err := fn(ctx, "s", 1); err != sentinel {
		t.Fatalf("expected sentinel, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

// ---- ComposeError3 ----

func TestComposeError3_CallsAll(t *testing.T) {
	calls := 0
	hooks := []testHook{
		{fn3: func(_ context.Context, _ string, _ int, _ bool) error { calls++; return nil }},
		{fn3: func(_ context.Context, _ string, _ int, _ bool) error { calls++; return nil }},
	}
	fn := hookutil.ComposeError3(hooks, func(h testHook) func(context.Context, string, int, bool) error { return h.fn3 })
	if err := fn(ctx, "s", 1, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
}

// ---- ComposeVoid0 ----

func TestComposeVoid0_SkipsNil(t *testing.T) {
	calls := 0
	hooks := []testHook{
		{v0: nil},
		{v0: func(_ context.Context) { calls++ }},
	}
	fn := hookutil.ComposeVoid0(hooks, func(h testHook) func(context.Context) { return h.v0 })
	fn(ctx)
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

// ---- ComposeVoid1 ----

func TestComposeVoid1_CallsAll(t *testing.T) {
	var got []string
	hooks := []testHook{
		{v1: func(_ context.Context, s string) { got = append(got, "a:"+s) }},
		{v1: func(_ context.Context, s string) { got = append(got, "b:"+s) }},
	}
	fn := hookutil.ComposeVoid1(hooks, func(h testHook) func(context.Context, string) { return h.v1 })
	fn(ctx, "q")
	if len(got) != 2 || got[0] != "a:q" || got[1] != "b:q" {
		t.Fatalf("unexpected calls: %v", got)
	}
}

// ---- ComposeVoid2 ----

func TestComposeVoid2_CallsAll(t *testing.T) {
	calls := 0
	hooks := []testHook{
		{v2: func(_ context.Context, _ string, _ int) { calls++ }},
		{v2: func(_ context.Context, _ string, _ int) { calls++ }},
	}
	fn := hookutil.ComposeVoid2(hooks, func(h testHook) func(context.Context, string, int) { return h.v2 })
	fn(ctx, "s", 1)
	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
}

// ---- ComposeVoid3 ----

func TestComposeVoid3_SkipsNil(t *testing.T) {
	calls := 0
	hooks := []testHook{
		{v3: nil},
		{v3: func(_ context.Context, _ string, _ int, _ bool) { calls++ }},
	}
	fn := hookutil.ComposeVoid3(hooks, func(h testHook) func(context.Context, string, int, bool) { return h.v3 })
	fn(ctx, "s", 1, true)
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

// ---- ComposeVoid4 ----

func TestComposeVoid4_CallsAll(t *testing.T) {
	calls := 0
	hooks := []testHook{
		{v4: func(_ context.Context, _ string, _ int, _ bool, _ float64) { calls++ }},
		{v4: func(_ context.Context, _ string, _ int, _ bool, _ float64) { calls++ }},
	}
	fn := hookutil.ComposeVoid4(hooks, func(h testHook) func(context.Context, string, int, bool, float64) { return h.v4 })
	fn(ctx, "s", 1, true, 3.14)
	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
}
