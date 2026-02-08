// Package gin provides a Gin-based ServerAdapter for the Beluga AI server package.
//
// Usage:
//
//	import _ "github.com/lookatitude/beluga-ai/server/adapters/gin"
//
//	adapter, err := server.New("gin", server.Config{})
//	adapter.RegisterAgent("/chat", myAgent)
//	adapter.Serve(ctx, ":8080")
package gin

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/server"
)

// Adapter implements server.ServerAdapter using the Gin HTTP framework.
type Adapter struct {
	engine *gin.Engine
	srv    *http.Server
	cfg    server.Config
	mu     sync.RWMutex
}

// New creates a new Gin adapter with the given configuration.
func New(cfg server.Config) *Adapter {
	gin.SetMode(gin.ReleaseMode)
	return &Adapter{
		engine: gin.New(),
		cfg:    cfg,
	}
}

// RegisterAgent registers an agent at the given path prefix. It creates
// sub-routes for invoke and stream endpoints using the standard agent handler.
func (a *Adapter) RegisterAgent(path string, ag agent.Agent) error {
	if ag == nil {
		return fmt.Errorf("server/gin: agent must not be nil")
	}
	handler := server.NewAgentHandler(ag)
	stripped := http.StripPrefix(path, handler)
	a.mu.Lock()
	defer a.mu.Unlock()
	a.engine.Any(path+"/*action", gin.WrapH(stripped))
	return nil
}

// RegisterHandler registers a raw http.Handler at the given path.
func (a *Adapter) RegisterHandler(path string, handler http.Handler) error {
	if handler == nil {
		return fmt.Errorf("server/gin: handler must not be nil")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.engine.Any(path, gin.WrapH(handler))
	return nil
}

// Serve starts the HTTP server on the given address. It blocks until the
// server exits or the context is canceled.
func (a *Adapter) Serve(ctx context.Context, addr string) error {
	a.mu.Lock()
	a.srv = &http.Server{
		Addr:         addr,
		Handler:      a.engine,
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
			return fmt.Errorf("server/gin: shutdown error: %w", err)
		}
		return ctx.Err()
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return fmt.Errorf("server/gin: %w", err)
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
		return fmt.Errorf("server/gin: shutdown error: %w", err)
	}
	return nil
}

// Engine returns the underlying gin.Engine for advanced configuration.
func (a *Adapter) Engine() *gin.Engine {
	return a.engine
}

func init() {
	server.Register("gin", func(cfg server.Config) (server.ServerAdapter, error) {
		return New(cfg), nil
	})
}
