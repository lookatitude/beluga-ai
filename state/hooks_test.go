package state

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestComposeHooks_BeforeGet(t *testing.T) {
	var calls []string
	h1 := Hooks{BeforeGet: func(ctx context.Context, key string) error {
		calls = append(calls, "h1:"+key)
		return nil
	}}
	h2 := Hooks{BeforeGet: func(ctx context.Context, key string) error {
		calls = append(calls, "h2:"+key)
		return nil
	}}

	composed := ComposeHooks(h1, h2)
	err := composed.BeforeGet(context.Background(), "k")
	assert.NoError(t, err)
	assert.Equal(t, []string{"h1:k", "h2:k"}, calls)
}

func TestComposeHooks_BeforeGet_ShortCircuit(t *testing.T) {
	errAbort := errors.New("abort")
	var calls []string
	h1 := Hooks{BeforeGet: func(ctx context.Context, key string) error {
		calls = append(calls, "h1")
		return errAbort
	}}
	h2 := Hooks{BeforeGet: func(ctx context.Context, key string) error {
		calls = append(calls, "h2")
		return nil
	}}

	composed := ComposeHooks(h1, h2)
	err := composed.BeforeGet(context.Background(), "k")
	assert.ErrorIs(t, err, errAbort)
	assert.Equal(t, []string{"h1"}, calls)
}

func TestComposeHooks_AfterGet(t *testing.T) {
	var calls []string
	h1 := Hooks{AfterGet: func(ctx context.Context, key string, val any, err error) {
		calls = append(calls, "h1")
	}}
	h2 := Hooks{AfterGet: func(ctx context.Context, key string, val any, err error) {
		calls = append(calls, "h2")
	}}

	composed := ComposeHooks(h1, h2)
	composed.AfterGet(context.Background(), "k", 42, nil)
	assert.Equal(t, []string{"h1", "h2"}, calls)
}

func TestComposeHooks_BeforeSet(t *testing.T) {
	var calls []string
	h1 := Hooks{BeforeSet: func(ctx context.Context, key string, value any) error {
		calls = append(calls, "h1")
		return nil
	}}
	h2 := Hooks{BeforeSet: func(ctx context.Context, key string, value any) error {
		calls = append(calls, "h2")
		return nil
	}}

	composed := ComposeHooks(h1, h2)
	err := composed.BeforeSet(context.Background(), "k", "v")
	assert.NoError(t, err)
	assert.Equal(t, []string{"h1", "h2"}, calls)
}

func TestComposeHooks_BeforeSet_ShortCircuit(t *testing.T) {
	errAbort := errors.New("abort")
	h1 := Hooks{BeforeSet: func(ctx context.Context, key string, value any) error {
		return errAbort
	}}
	h2 := Hooks{BeforeSet: func(ctx context.Context, key string, value any) error {
		t.Fatal("should not be called")
		return nil
	}}

	composed := ComposeHooks(h1, h2)
	err := composed.BeforeSet(context.Background(), "k", "v")
	assert.ErrorIs(t, err, errAbort)
}

func TestComposeHooks_AfterSet(t *testing.T) {
	var calls []string
	h1 := Hooks{AfterSet: func(ctx context.Context, key string, value any, err error) {
		calls = append(calls, "h1")
	}}
	h2 := Hooks{AfterSet: func(ctx context.Context, key string, value any, err error) {
		calls = append(calls, "h2")
	}}

	composed := ComposeHooks(h1, h2)
	composed.AfterSet(context.Background(), "k", "v", nil)
	assert.Equal(t, []string{"h1", "h2"}, calls)
}

func TestComposeHooks_OnDelete(t *testing.T) {
	var calls []string
	h1 := Hooks{OnDelete: func(ctx context.Context, key string) error {
		calls = append(calls, "h1")
		return nil
	}}
	h2 := Hooks{OnDelete: func(ctx context.Context, key string) error {
		calls = append(calls, "h2")
		return nil
	}}

	composed := ComposeHooks(h1, h2)
	err := composed.OnDelete(context.Background(), "k")
	assert.NoError(t, err)
	assert.Equal(t, []string{"h1", "h2"}, calls)
}

func TestComposeHooks_OnDelete_ShortCircuit(t *testing.T) {
	errAbort := errors.New("abort")
	h1 := Hooks{OnDelete: func(ctx context.Context, key string) error {
		return errAbort
	}}
	h2 := Hooks{OnDelete: func(ctx context.Context, key string) error {
		t.Fatal("should not be called")
		return nil
	}}

	composed := ComposeHooks(h1, h2)
	err := composed.OnDelete(context.Background(), "k")
	assert.ErrorIs(t, err, errAbort)
}

func TestComposeHooks_OnWatch(t *testing.T) {
	var calls []string
	h1 := Hooks{OnWatch: func(ctx context.Context, key string) error {
		calls = append(calls, "h1")
		return nil
	}}
	h2 := Hooks{OnWatch: func(ctx context.Context, key string) error {
		calls = append(calls, "h2")
		return nil
	}}

	composed := ComposeHooks(h1, h2)
	err := composed.OnWatch(context.Background(), "k")
	assert.NoError(t, err)
	assert.Equal(t, []string{"h1", "h2"}, calls)
}

func TestComposeHooks_OnWatch_ShortCircuit(t *testing.T) {
	errAbort := errors.New("abort")
	h1 := Hooks{OnWatch: func(ctx context.Context, key string) error {
		return errAbort
	}}

	composed := ComposeHooks(h1)
	err := composed.OnWatch(context.Background(), "k")
	assert.ErrorIs(t, err, errAbort)
}

func TestComposeHooks_OnError_ReturnOriginal(t *testing.T) {
	origErr := errors.New("original")
	h1 := Hooks{OnError: func(ctx context.Context, err error) error {
		return nil // suppress — but ComposeHooks returns original if all return nil
	}}

	composed := ComposeHooks(h1)
	got := composed.OnError(context.Background(), origErr)
	assert.ErrorIs(t, got, origErr)
}

func TestComposeHooks_OnError_NonNilShortCircuits(t *testing.T) {
	origErr := errors.New("original")
	replacement := errors.New("replaced")
	var calls []string

	h1 := Hooks{OnError: func(ctx context.Context, err error) error {
		calls = append(calls, "h1")
		return replacement
	}}
	h2 := Hooks{OnError: func(ctx context.Context, err error) error {
		calls = append(calls, "h2")
		return nil
	}}

	composed := ComposeHooks(h1, h2)
	got := composed.OnError(context.Background(), origErr)
	assert.ErrorIs(t, got, replacement)
	assert.Equal(t, []string{"h1"}, calls)
}

func TestComposeHooks_NilHooksSkipped(t *testing.T) {
	// All nil hooks — should not panic.
	h1 := Hooks{} // all nil
	h2 := Hooks{AfterGet: func(ctx context.Context, key string, val any, err error) {}}

	composed := ComposeHooks(h1, h2)

	// BeforeGet with no hooks set on h1 or h2 should return nil.
	err := composed.BeforeGet(context.Background(), "k")
	assert.NoError(t, err)

	err = composed.BeforeSet(context.Background(), "k", "v")
	assert.NoError(t, err)

	err = composed.OnDelete(context.Background(), "k")
	assert.NoError(t, err)

	err = composed.OnWatch(context.Background(), "k")
	assert.NoError(t, err)
}
