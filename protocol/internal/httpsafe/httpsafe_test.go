package httpsafe

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewRequestID(t *testing.T) {
	a, b := NewRequestID(), NewRequestID()
	if len(a) != 32 || a == b {
		t.Fatalf("unexpected ids %q %q", a, b)
	}
}

func TestIsBodyTooLarge(t *testing.T) {
	rec := httptest.NewRecorder()
	r := http.MaxBytesReader(rec, io.NopCloser(strings.NewReader(strings.Repeat("x", 10))), 4)
	_, err := io.ReadAll(r)
	if !IsBodyTooLarge(err) {
		t.Fatalf("expected body too large, got %v", err)
	}
	if IsBodyTooLarge(fmt.Errorf("other")) || IsBodyTooLarge(nil) {
		t.Fatal("false positive")
	}
}

func TestLogValue(t *testing.T) {
	tests := []struct{ in, want string }{
		{"plain", "plain"},
		{"a\nb\rc\td", "a b c d"},
		{"x\x00y\x7f", "x?y?"},
	}
	for _, tt := range tests {
		if got := LogValue(tt.in); got != tt.want {
			t.Errorf("LogValue(%q)=%q want %q", tt.in, got, tt.want)
		}
	}
	if got := LogValue(strings.Repeat("a", 2000)); len(got) != maxLogValueLen {
		t.Errorf("expected truncation to %d, got %d", maxLogValueLen, len(got))
	}
}

func TestSSEField(t *testing.T) {
	tests := []struct{ in, want string }{
		{"text", "text"},
		{"evil\r\ndata: injected", "evildata: injected"},
		{"a\nb", "ab"},
	}
	for _, tt := range tests {
		if got := SSEField(tt.in); got != tt.want {
			t.Errorf("SSEField(%q)=%q want %q", tt.in, got, tt.want)
		}
	}
}
