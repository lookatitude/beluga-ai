// Package chi provides a Chi-based ServerAdapter for the Beluga AI server package.
//
// Usage:
//
//	import _ "github.com/lookatitude/beluga-ai/server/adapters/chi"
//
//	adapter, err := server.New("chi", server.Config{})
//	adapter.RegisterAgent("/chat", myAgent)
//	adapter.Serve(ctx, ":8080")
package chi

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/server"
)

// Adapter implements server.ServerAdapter using the Chi router.
type Adapter struct {
	router chi.Router
	srv    *http.Server
	cfg    server.Config
	mu     sync.RWMutex
}

// New creates a new Chi adapter with the given configuration.
func New(cfg server.Config) *Adapter {
	return &Adapter{
		router: chi.NewRouter(),
		cfg:    cfg,
	}
}

// RegisterAgent registers an agent at the given path prefix. It creates
// sub-routes for invoke and stream endpoints using the standard agent handler.
func (a *Adapter) RegisterAgent(path string, ag agent.Agent) error {
	if ag == nil {
		return fmt.Errorf("server/chi: agent must not be nil")
	}
	handler := server.NewAgentHandler(ag)
	stripped := http.StripPrefix(path, handler)
	a.mu.Lock()
	defer a.mu.Unlock()
	a.router.Handle(path+"/*", stripped)
	return nil
}

// RegisterHandler registers a raw http.Handler at the given path.
func (a *Adapter) RegisterHandler(path string, handler http.Handler) error {
	if handler == nil {
		return fmt.Errorf("server/chi: handler must not be nil")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.router.Handle(path, handler)
	return nil
}

// Serve starts the HTTP server on the given address. It blocks until the
// server exits or the context is canceled.
func (a *Adapter) Serve(ctx context.Context, addr string) error {
	a.mu.Lock()
	a.srv = &http.Server{
		Addr:         addr,
		Handler:      a.router,
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
			return fmt.Errorf("server/chi: shutdown error: %w", err)
		}
		return ctx.Err()
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return fmt.Errorf("server/chi: %w", err)
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
		return fmt.Errorf("server/chi: shutdown error: %w", err)
	}
	return nil
}

// Router returns the underlying chi.Router for advanced configuration.
func (a *Adapter) Router() chi.Router {
	return a.router
}

func init() {
	server.Register("chi", func(cfg server.Config) (server.ServerAdapter, error) {
		return New(cfg), nil
	})
}
