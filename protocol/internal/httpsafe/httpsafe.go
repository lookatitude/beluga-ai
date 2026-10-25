// Package httpsafe holds small HTTP hardening helpers shared by the protocol
// servers (rest, a2a, mcp): request-body limits, default server timeouts,
// correlation IDs for sanitized error responses, and log/SSE field
// sanitization. It is internal to the protocol layer.
package httpsafe

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// DefaultMaxBodyBytes is the default request-body limit (1 MiB), matching
// server/playground and protocol/agui.
const DefaultMaxBodyBytes int64 = 1 << 20

// Default http.Server timeouts used by the protocol servers' Serve methods.
// WriteTimeout is deliberately not defined here: servers that stream (SSE)
// must leave it at zero, and each server documents its own choice.
const (
	DefaultReadHeaderTimeout = 10 * time.Second
	DefaultReadTimeout       = 30 * time.Second
	DefaultIdleTimeout       = 120 * time.Second
)

// GenericInternalError is the message returned to clients in place of
// internal error text.
const GenericInternalError = "internal error"

// RequestIDHeader is the response header carrying the correlation ID.
const RequestIDHeader = "X-Request-Id"

// NewRequestID returns a random 128-bit hex correlation ID. If the system
// random source fails it falls back to a timestamp-derived ID so that error
// handling never fails.
func NewRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "t" + strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return hex.EncodeToString(b[:])
}

// IsBodyTooLarge reports whether err was produced by an http.MaxBytesReader
// whose limit was exceeded.
func IsBodyTooLarge(err error) bool {
	var mbe *http.MaxBytesError
	return errors.As(err, &mbe)
}

// maxLogValueLen bounds sanitized log values.
const maxLogValueLen = 512

// LogValue sanitizes s for inclusion in a structured log field: it truncates
// to a bounded length, replaces CR/LF/TAB with spaces and other control
// characters with '?', preventing log injection.
func LogValue(s string) string {
	if len(s) > maxLogValueLen {
		s = s[:maxLogValueLen]
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r == '\n' || r == '\r' || r == '\t':
			b.WriteByte(' ')
		case r < 0x20 || r == 0x7f:
			b.WriteByte('?')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// SSEField strips CR and LF from a value written into a single-line SSE
// field (event:, id:), preventing SSE field/event injection.
func SSEField(s string) string {
	if !strings.ContainsAny(s, "\r\n") {
		return s
	}
	return strings.NewReplacer("\r", "", "\n", "").Replace(s)
}
