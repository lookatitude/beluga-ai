// Package huma provides a Huma-based ServerAdapter for the Beluga AI server package.
// Huma is an OpenAPI-first framework that wraps standard net/http with automatic
// documentation generation.
//
// Usage:
//
//	import _ "github.com/lookatitude/beluga-ai/server/adapters/huma"
//
//	adapter, err := server.New("huma", server.Config{})
//	adapter.RegisterAgent("/chat", myAgent)
//	adapter.Serve(ctx, ":8080")
package huma

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/server"
)

// Adapter implements server.ServerAdapter using Huma with the stdlib net/http mux.
type Adapter struct {
	mux *http.ServeMux
	api huma.API
	srv *http.Server
	cfg server.Config
	mu  sync.RWMutex
}

// Compile-time interface check.
var _ server.ServerAdapter = (*Adapter)(nil)

// New creates a new Huma adapter with the given configuration.
func New(cfg server.Config) *Adapter {
	mux := http.NewServeMux()
	title := "Beluga AI"
	version := "1.0.0"
	if t, ok := cfg.Extra["title"].(string); ok && t != "" {
		title = t
	}
	if v, ok := cfg.Extra["version"].(string); ok && v != "" {
		version = v
	}
	api := humago.New(mux, huma.DefaultConfig(title, version))
	return &Adapter{
		mux: mux,
		api: api,
		cfg: cfg,
	}
}

// RegisterAgent registers an agent at the given path prefix. It creates
// sub-routes for invoke and stream endpoints using the standard agent handler.
func (a *Adapter) RegisterAgent(path string, ag agent.Agent) error {
	if ag == nil {
		return fmt.Errorf("server/huma: agent must not be nil")
	}
	handler := server.NewAgentHandler(ag)
	stripped := http.StripPrefix(path, handler)
	a.mu.Lock()
	defer a.mu.Unlock()
	a.mux.Handle(path+"/", stripped)
	return nil
}

// RegisterHandler registers a raw http.Handler at the given path.
func (a *Adapter) RegisterHandler(path string, handler http.Handler) error {
	if handler == nil {
		return fmt.Errorf("server/huma: handler must not be nil")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.mux.Handle(path, handler)
	return nil
}

// Serve starts the HTTP server on the given address. It blocks until the
// server exits or the context is canceled.
func (a *Adapter) Serve(ctx context.Context, addr string) error {
	a.mu.Lock()
	a.srv = &http.Server{
		Addr:         addr,
		Handler:      a.mux,
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
			return fmt.Errorf("server/huma: shutdown error: %w", err)
		}
		return ctx.Err()
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return fmt.Errorf("server/huma: %w", err)
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
		return fmt.Errorf("server/huma: shutdown error: %w", err)
	}
	return nil
}

// API returns the underlying huma.API for advanced configuration.
func (a *Adapter) API() huma.API {
	return a.api
}

// Mux returns the underlying http.ServeMux.
func (a *Adapter) Mux() *http.ServeMux {
	return a.mux
}

func init() {
	server.Register("huma", func(cfg server.Config) (server.ServerAdapter, error) {
		return New(cfg), nil
	})
}
