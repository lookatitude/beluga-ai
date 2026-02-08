// Package connect provides a Connect-Go based ServerAdapter for the Beluga AI
// server package. Connect-Go enables HTTP/1.1 + protobuf communication that is
// compatible with gRPC, gRPC-Web, and Connect protocol clients.
//
// Usage:
//
//	import _ "github.com/lookatitude/beluga-ai/server/adapters/connect"
//
//	adapter, err := server.New("connect", server.Config{})
//	adapter.RegisterAgent("/chat", myAgent)
//	adapter.Serve(ctx, ":8080")
package connect

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/server"
)

// Adapter implements server.ServerAdapter using Connect-Go over net/http.
// It provides HTTP/1.1 + HTTP/2 support compatible with gRPC, gRPC-Web,
// and Connect protocol clients.
type Adapter struct {
	mux *http.ServeMux
	srv *http.Server
	cfg server.Config
	mu  sync.RWMutex
}

// New creates a new Connect-Go adapter with the given configuration.
func New(cfg server.Config) *Adapter {
	return &Adapter{
		mux: http.NewServeMux(),
		cfg: cfg,
	}
}

// RegisterAgent registers an agent at the given path prefix. It creates
// sub-routes for invoke and stream endpoints using the standard agent handler.
func (a *Adapter) RegisterAgent(path string, ag agent.Agent) error {
	if ag == nil {
		return fmt.Errorf("server/connect: agent must not be nil")
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
		return fmt.Errorf("server/connect: handler must not be nil")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.mux.Handle(path, handler)
	return nil
}

// Serve starts the HTTP server on the given address. It supports both HTTP/1.1
// and HTTP/2, making it compatible with Connect, gRPC, and gRPC-Web clients.
// It blocks until the server exits or the context is canceled.
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
			return fmt.Errorf("server/connect: shutdown error: %w", err)
		}
		return ctx.Err()
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return fmt.Errorf("server/connect: %w", err)
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
		return fmt.Errorf("server/connect: shutdown error: %w", err)
	}
	return nil
}

// Mux returns the underlying http.ServeMux for advanced configuration,
// such as registering Connect-Go service handlers directly.
func (a *Adapter) Mux() *http.ServeMux {
	return a.mux
}

func init() {
	server.Register("connect", func(cfg server.Config) (server.ServerAdapter, error) {
		return New(cfg), nil
	})
}
