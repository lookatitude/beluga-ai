package core

import "context"

// contextKey is an unexported type used for context keys in this package to
// prevent collisions with keys defined in other packages.
type contextKey int

const (
	sessionIDKey contextKey = iota
	requestIDKey
)

// WithSessionID returns a copy of ctx carrying the given session ID.
func WithSessionID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, sessionIDKey, id)
}

// GetSessionID extracts the session ID from ctx. It returns an empty string
// if no session ID is present.
func GetSessionID(ctx context.Context) string {
	id, _ := ctx.Value(sessionIDKey).(string)
	return id
}

// WithRequestID returns a copy of ctx carrying the given request ID.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// GetRequestID extracts the request ID from ctx. It returns an empty string
// if no request ID is present.
func GetRequestID(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}
