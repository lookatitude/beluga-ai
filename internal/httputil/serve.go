// Package httputil provides shared HTTP server lifecycle helpers used by the
// server adapter implementations. It is an internal package and must not be
// imported outside of this module.
package httputil

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// defaultShutdownTimeout is the grace period given to an http.Server to finish
// in-flight requests when the serve context is canceled.
const defaultShutdownTimeout = 5 * time.Second

// ServerLifecycle manages the lifecycle of a single *http.Server. It is
// designed to be embedded in server adapter structs so they can delegate their
// Serve and Shutdown logic without duplicating the select/goroutine pattern.
//
// The zero value is ready to use.
type ServerLifecycle struct {
	mu  sync.RWMutex
	srv *http.Server
}

// Serve constructs an *http.Server from the supplied handler and timeout
// values, starts it in a goroutine, and blocks until either the context is
// canceled or the server exits on its own.
//
//   - addr is the TCP address to listen on (e.g. ":8080").
//   - handler is the root http.Handler for the server.
//   - readTimeout, writeTimeout, idleTimeout are forwarded verbatim to
//     http.Server; zero values disable the corresponding timeout.
//   - errPrefix is included in any error message returned, e.g. "server/chi".
//
// When the context is canceled Serve performs a graceful shutdown with a
// 5-second deadline and returns ctx.Err(). When the server closes on its own
// (http.ErrServerClosed) Serve returns nil.
func (l *ServerLifecycle) Serve(
	ctx context.Context,
	addr string,
	handler http.Handler,
	readTimeout, writeTimeout, idleTimeout time.Duration,
	errPrefix string,
) error {
	srv := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}

	l.mu.Lock()
	l.srv = srv
	l.mu.Unlock()

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), defaultShutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("%s: shutdown error: %w", errPrefix, err)
		}
		return ctx.Err()
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return fmt.Errorf("%s: %w", errPrefix, err)
	}
}

// Shutdown gracefully shuts down the server that was started by the most
// recent call to Serve. If Serve has not been called yet, Shutdown is a no-op.
// errPrefix is included in any error message returned.
func (l *ServerLifecycle) Shutdown(ctx context.Context, errPrefix string) error {
	l.mu.RLock()
	srv := l.srv
	l.mu.RUnlock()
	if srv == nil {
		return nil
	}
	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("%s: shutdown error: %w", errPrefix, err)
	}
	return nil
}
