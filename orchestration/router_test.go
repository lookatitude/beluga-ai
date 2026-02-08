package orchestration

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

func TestRouter_RouteSelection(t *testing.T) {
	classifier := func(_ context.Context, input any) (string, error) {
		return fmt.Sprintf("%v", input), nil
	}

	r := NewRouter(classifier).
		AddRoute("math", newStep(func(_ any) (any, error) { return "math-result", nil })).
		AddRoute("code", newStep(func(_ any) (any, error) { return "code-result", nil }))

	result, err := r.Invoke(context.Background(), "math")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "math-result" {
		t.Fatalf("expected math-result, got %v", result)
	}

	result, err = r.Invoke(context.Background(), "code")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "code-result" {
		t.Fatalf("expected code-result, got %v", result)
	}
}

func TestRouter_Fallback(t *testing.T) {
	classifier := func(_ context.Context, _ any) (string, error) {
		return "unknown-route", nil
	}

	r := NewRouter(classifier).
		AddRoute("known", newStep(func(_ any) (any, error) { return "known", nil })).
		SetFallback(newStep(func(_ any) (any, error) { return "fallback-result", nil }))

	result, err := r.Invoke(context.Background(), "x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "fallback-result" {
		t.Fatalf("expected fallback-result, got %v", result)
	}
}

func TestRouter_UnknownRoute_NoFallback(t *testing.T) {
	classifier := func(_ context.Context, _ any) (string, error) {
		return "unknown", nil
	}

	r := NewRouter(classifier).
		AddRoute("known", newStep(func(_ any) (any, error) { return "known", nil }))

	_, err := r.Invoke(context.Background(), "x")
	if err == nil {
		t.Fatal("expected error for unknown route without fallback")
	}
}

func TestRouter_ClassifierError(t *testing.T) {
	errClassify := errors.New("classify failed")
	classifier := func(_ context.Context, _ any) (string, error) {
		return "", errClassify
	}

	r := NewRouter(classifier).
		AddRoute("a", newStep(func(_ any) (any, error) { return "a", nil }))

	_, err := r.Invoke(context.Background(), "x")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, errClassify) {
		t.Fatalf("expected classify error, got %v", err)
	}
}

func TestRouter_RouteHandlerError(t *testing.T) {
	errHandler := errors.New("handler error")
	classifier := func(_ context.Context, _ any) (string, error) {
		return "broken", nil
	}

	r := NewRouter(classifier).
		AddRoute("broken", newStep(func(_ any) (any, error) { return nil, errHandler }))

	_, err := r.Invoke(context.Background(), "x")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, errHandler) {
		t.Fatalf("expected handler error, got %v", err)
	}
}

func TestRouter_Stream(t *testing.T) {
	classifier := func(_ context.Context, _ any) (string, error) {
		return "route1", nil
	}

	r := NewRouter(classifier).
		AddRoute("route1", newStep(func(_ any) (any, error) { return "streamed", nil }))

	var results []any
	for val, err := range r.Stream(context.Background(), "x") {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		results = append(results, val)
	}
	if len(results) != 1 || results[0] != "streamed" {
		t.Fatalf("expected [streamed], got %v", results)
	}
}

func TestRouter_Stream_ClassifierError(t *testing.T) {
	errClassify := errors.New("classify failed")
	classifier := func(_ context.Context, _ any) (string, error) {
		return "", errClassify
	}

	r := NewRouter(classifier)

	for _, err := range r.Stream(context.Background(), "x") {
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, errClassify) {
			t.Fatalf("expected classify error, got %v", err)
		}
		return
	}
	t.Fatal("expected at least one stream result")
}
