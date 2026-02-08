package core

import (
	"errors"
	"fmt"
	"testing"
)

func TestNewError(t *testing.T) {
	cause := fmt.Errorf("connection refused")
	e := NewError("llm.generate", ErrProviderDown, "provider unreachable", cause)

	if e.Op != "llm.generate" {
		t.Errorf("Op = %q, want %q", e.Op, "llm.generate")
	}
	if e.Code != ErrProviderDown {
		t.Errorf("Code = %q, want %q", e.Code, ErrProviderDown)
	}
	if e.Message != "provider unreachable" {
		t.Errorf("Message = %q, want %q", e.Message, "provider unreachable")
	}
	if e.Err != cause {
		t.Errorf("Err = %v, want %v", e.Err, cause)
	}
}

func TestNewError_NilCause(t *testing.T) {
	e := NewError("tool.execute", ErrToolFailed, "tool error", nil)
	if e.Err != nil {
		t.Errorf("Err = %v, want nil", e.Err)
	}
}

func TestError_Error(t *testing.T) {
	tests := []struct {
		name  string
		err   *Error
		want  string
	}{
		{
			name: "with_cause",
			err:  NewError("llm.generate", ErrRateLimit, "too many requests", fmt.Errorf("429")),
			want: "llm.generate [rate_limit]: too many requests: 429",
		},
		{
			name: "without_cause",
			err:  NewError("tool.execute", ErrToolFailed, "tool crashed", nil),
			want: "tool.execute [tool_failed]: tool crashed",
		},
		{
			name: "empty_fields",
			err:  NewError("", "", "", nil),
			want: " []: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestError_Unwrap(t *testing.T) {
	tests := []struct {
		name string
		err  *Error
		want error
	}{
		{
			name: "with_cause",
			err:  NewError("op", ErrAuth, "msg", fmt.Errorf("underlying")),
			want: fmt.Errorf("underlying"),
		},
		{
			name: "nil_cause",
			err:  NewError("op", ErrAuth, "msg", nil),
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Unwrap()
			if tt.want == nil && got != nil {
				t.Errorf("Unwrap() = %v, want nil", got)
			}
			if tt.want != nil && (got == nil || got.Error() != tt.want.Error()) {
				t.Errorf("Unwrap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestError_Is(t *testing.T) {
	tests := []struct {
		name   string
		err    *Error
		target error
		want   bool
	}{
		{
			name:   "same_code",
			err:    NewError("op1", ErrRateLimit, "msg1", nil),
			target: NewError("op2", ErrRateLimit, "msg2", nil),
			want:   true,
		},
		{
			name:   "different_code",
			err:    NewError("op", ErrRateLimit, "msg", nil),
			target: NewError("op", ErrAuth, "msg", nil),
			want:   false,
		},
		{
			name:   "non_beluga_error",
			err:    NewError("op", ErrRateLimit, "msg", nil),
			target: fmt.Errorf("plain error"),
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Is(tt.target)
			if got != tt.want {
				t.Errorf("Is() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestError_ErrorsIs(t *testing.T) {
	cause := NewError("inner", ErrRateLimit, "rate limited", nil)
	wrapped := fmt.Errorf("outer: %w", cause)

	if !errors.Is(wrapped, NewError("", ErrRateLimit, "", nil)) {
		t.Error("errors.Is should match wrapped Error by code")
	}
}

func TestError_ErrorsAs(t *testing.T) {
	cause := NewError("inner", ErrAuth, "unauthorized", nil)
	wrapped := fmt.Errorf("outer: %w", cause)

	var target *Error
	if !errors.As(wrapped, &target) {
		t.Fatal("errors.As should find *Error in chain")
	}
	if target.Code != ErrAuth {
		t.Errorf("Code = %q, want %q", target.Code, ErrAuth)
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "rate_limit",
			err:  NewError("op", ErrRateLimit, "msg", nil),
			want: true,
		},
		{
			name: "timeout",
			err:  NewError("op", ErrTimeout, "msg", nil),
			want: true,
		},
		{
			name: "provider_down",
			err:  NewError("op", ErrProviderDown, "msg", nil),
			want: true,
		},
		{
			name: "auth_error",
			err:  NewError("op", ErrAuth, "msg", nil),
			want: false,
		},
		{
			name: "invalid_input",
			err:  NewError("op", ErrInvalidInput, "msg", nil),
			want: false,
		},
		{
			name: "tool_failed",
			err:  NewError("op", ErrToolFailed, "msg", nil),
			want: false,
		},
		{
			name: "guard_blocked",
			err:  NewError("op", ErrGuardBlocked, "msg", nil),
			want: false,
		},
		{
			name: "budget_exhausted",
			err:  NewError("op", ErrBudgetExhausted, "msg", nil),
			want: false,
		},
		{
			name: "plain_error",
			err:  fmt.Errorf("not a beluga error"),
			want: false,
		},
		{
			name: "nil_error",
			err:  nil,
			want: false,
		},
		{
			name: "wrapped_retryable",
			err:  fmt.Errorf("wrap: %w", NewError("op", ErrRateLimit, "msg", nil)),
			want: true,
		},
		{
			name: "wrapped_non_retryable",
			err:  fmt.Errorf("wrap: %w", NewError("op", ErrAuth, "msg", nil)),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsRetryable(tt.err)
			if got != tt.want {
				t.Errorf("IsRetryable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestErrorCodes_Values(t *testing.T) {
	// Ensure error codes have expected string values.
	codes := map[ErrorCode]string{
		ErrRateLimit:       "rate_limit",
		ErrAuth:            "auth_error",
		ErrTimeout:         "timeout",
		ErrInvalidInput:    "invalid_input",
		ErrToolFailed:      "tool_failed",
		ErrProviderDown:    "provider_unavailable",
		ErrGuardBlocked:    "guard_blocked",
		ErrBudgetExhausted: "budget_exhausted",
	}

	for code, want := range codes {
		if string(code) != want {
			t.Errorf("ErrorCode %v = %q, want %q", code, string(code), want)
		}
	}
}
