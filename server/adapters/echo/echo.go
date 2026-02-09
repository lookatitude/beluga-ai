// Package echo provides an Echo-based ServerAdapter for the Beluga AI server package.
//
// Usage:
//
//	import _ "github.com/lookatitude/beluga-ai/server/adapters/echo"
//
//	adapter, err := server.New("echo", server.Config{})
//	adapter.RegisterAgent("/chat", myAgent)
//	adapter.Serve(ctx, ":8080")
package echo

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/server"
)

// Adapter implements server.ServerAdapter using the Echo HTTP framework.
type Adapter struct {
	echo *echo.Echo
	srv  *http.Server
	cfg  server.Config
	mu   sync.RWMutex
}

// Compile-time interface check.
var _ server.ServerAdapter = (*Adapter)(nil)

// New creates a new Echo adapter with the given configuration.
func New(cfg server.Config) *Adapter {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	return &Adapter{
		echo: e,
		cfg:  cfg,
	}
}

// RegisterAgent registers an agent at the given path prefix. It creates
// sub-routes for invoke and stream endpoints using the standard agent handler.
func (a *Adapter) RegisterAgent(path string, ag agent.Agent) error {
	if ag == nil {
		return fmt.Errorf("server/echo: agent must not be nil")
	}
	handler := server.NewAgentHandler(ag)
	stripped := http.StripPrefix(path, handler)
	a.mu.Lock()
	defer a.mu.Unlock()
	a.echo.Any(path+"/*", echo.WrapHandler(stripped))
	return nil
}

// RegisterHandler registers a raw http.Handler at the given path.
func (a *Adapter) RegisterHandler(path string, handler http.Handler) error {
	if handler == nil {
		return fmt.Errorf("server/echo: handler must not be nil")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.echo.Any(path, echo.WrapHandler(handler))
	return nil
}

// Serve starts the HTTP server on the given address. It blocks until the
// server exits or the context is canceled.
func (a *Adapter) Serve(ctx context.Context, addr string) error {
	a.mu.Lock()
	a.srv = &http.Server{
		Addr:         addr,
		Handler:      a.echo,
		ReadTimeout:  a.cfg.ReadTimeout,
		WriteTimeout: a.cfg.WriteTimeout,
		IdleTimeout:  a.cfg.IdleTimeout,
	}
	a.mu.Unlock()

	errCh := make(chan error, 1)
	go func() {
		errCh <- a.srv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := a.srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("server/echo: shutdown error: %w", err)
		}
		return ctx.Err()
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return fmt.Errorf("server/echo: %w", err)
	}
}

// Shutdown gracefully shuts down the server.
func (a *Adapter) Shutdown(ctx context.Context) error {
	a.mu.RLock()
	srv := a.srv
	a.mu.RUnlock()
	if srv == nil {
		return nil
	}
	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("server/echo: shutdown error: %w", err)
	}
	return nil
}

// Echo returns the underlying echo.Echo instance for advanced configuration.
func (a *Adapter) Echo() *echo.Echo {
	return a.echo
}

func init() {
	server.Register("echo", func(cfg server.Config) (server.ServerAdapter, error) {
		return New(cfg), nil
	})
}
