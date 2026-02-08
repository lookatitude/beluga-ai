package core

import (
	"context"
	"testing"
)

func TestWithSessionID_GetSessionID(t *testing.T) {
	tests := []struct {
		name string
		id   string
	}{
		{name: "normal", id: "sess-abc-123"},
		{name: "empty", id: ""},
		{name: "uuid", id: "550e8400-e29b-41d4-a716-446655440000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := WithSessionID(context.Background(), tt.id)
			got := GetSessionID(ctx)
			if got != tt.id {
				t.Errorf("GetSessionID() = %q, want %q", got, tt.id)
			}
		})
	}
}

func TestGetSessionID_NotSet(t *testing.T) {
	got := GetSessionID(context.Background())
	if got != "" {
		t.Errorf("GetSessionID() = %q, want empty", got)
	}
}

func TestWithRequestID_GetRequestID(t *testing.T) {
	tests := []struct {
		name string
		id   string
	}{
		{name: "normal", id: "req-xyz-789"},
		{name: "empty", id: ""},
		{name: "uuid", id: "6ba7b810-9dad-11d1-80b4-00c04fd430c8"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := WithRequestID(context.Background(), tt.id)
			got := GetRequestID(ctx)
			if got != tt.id {
				t.Errorf("GetRequestID() = %q, want %q", got, tt.id)
			}
		})
	}
}

func TestGetRequestID_NotSet(t *testing.T) {
	got := GetRequestID(context.Background())
	if got != "" {
		t.Errorf("GetRequestID() = %q, want empty", got)
	}
}

func TestContextValues_Independent(t *testing.T) {
	ctx := context.Background()
	ctx = WithSessionID(ctx, "session-1")
	ctx = WithRequestID(ctx, "request-1")

	if got := GetSessionID(ctx); got != "session-1" {
		t.Errorf("GetSessionID() = %q, want %q", got, "session-1")
	}
	if got := GetRequestID(ctx); got != "request-1" {
		t.Errorf("GetRequestID() = %q, want %q", got, "request-1")
	}
}

func TestSessionID_Overwrite(t *testing.T) {
	ctx := WithSessionID(context.Background(), "first")
	ctx = WithSessionID(ctx, "second")

	got := GetSessionID(ctx)
	if got != "second" {
		t.Errorf("GetSessionID() = %q, want %q", got, "second")
	}
}

func TestRequestID_Overwrite(t *testing.T) {
	ctx := WithRequestID(context.Background(), "first")
	ctx = WithRequestID(ctx, "second")

	got := GetRequestID(ctx)
	if got != "second" {
		t.Errorf("GetRequestID() = %q, want %q", got, "second")
	}
}

func TestContextValues_DoNotAffectParent(t *testing.T) {
	parent := context.Background()
	_ = WithSessionID(parent, "child-session")
	_ = WithRequestID(parent, "child-request")

	if got := GetSessionID(parent); got != "" {
		t.Errorf("parent GetSessionID() = %q, want empty", got)
	}
	if got := GetRequestID(parent); got != "" {
		t.Errorf("parent GetRequestID() = %q, want empty", got)
	}
}

func TestAllContextHelpers_Together(t *testing.T) {
	ctx := context.Background()
	ctx = WithSessionID(ctx, "s1")
	ctx = WithRequestID(ctx, "r1")
	ctx = WithTenant(ctx, TenantID("t1"))

	if got := GetSessionID(ctx); got != "s1" {
		t.Errorf("GetSessionID() = %q, want %q", got, "s1")
	}
	if got := GetRequestID(ctx); got != "r1" {
		t.Errorf("GetRequestID() = %q, want %q", got, "r1")
	}
	if got := GetTenant(ctx); got != TenantID("t1") {
		t.Errorf("GetTenant() = %q, want %q", got, "t1")
	}
}
