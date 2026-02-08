// Package fiber provides a Fiber v3-based ServerAdapter for the Beluga AI server package.
//
// Usage:
//
//	import _ "github.com/lookatitude/beluga-ai/server/adapters/fiber"
//
//	adapter, err := server.New("fiber", server.Config{})
//	adapter.RegisterAgent("/chat", myAgent)
//	adapter.Serve(ctx, ":8080")
package fiber

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/server"
)

// Adapter implements server.ServerAdapter using the Fiber v3 HTTP framework.
type Adapter struct {
	app *fiber.App
	cfg server.Config
	mu  sync.RWMutex
}

// New creates a new Fiber adapter with the given configuration.
func New(cfg server.Config) *Adapter {
	app := fiber.New(fiber.Config{
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	})
	return &Adapter{
		app: app,
		cfg: cfg,
	}
}

// RegisterAgent registers an agent at the given path prefix. It creates
// sub-routes for invoke and stream endpoints using the standard agent handler.
func (a *Adapter) RegisterAgent(path string, ag agent.Agent) error {
	if ag == nil {
		return fmt.Errorf("server/fiber: agent must not be nil")
	}
	handler := server.NewAgentHandler(ag)
	stripped := http.StripPrefix(path, handler)
	a.mu.Lock()
	defer a.mu.Unlock()
	a.app.All(path+"/*", stripped)
	return nil
}

// RegisterHandler registers a raw http.Handler at the given path.
func (a *Adapter) RegisterHandler(path string, handler http.Handler) error {
	if handler == nil {
		return fmt.Errorf("server/fiber: handler must not be nil")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.app.All(path, handler)
	return nil
}

// Serve starts the HTTP server on the given address. It blocks until the
// server exits or the context is canceled.
func (a *Adapter) Serve(ctx context.Context, addr string) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- a.app.Listen(addr)
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = shutdownCtx // Fiber's ShutdownWithContext accepts context
		if err := a.app.ShutdownWithContext(shutdownCtx); err != nil {
			return fmt.Errorf("server/fiber: shutdown error: %w", err)
		}
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

// Shutdown gracefully shuts down the server.
func (a *Adapter) Shutdown(ctx context.Context) error {
	if err := a.app.ShutdownWithContext(ctx); err != nil {
		return fmt.Errorf("server/fiber: shutdown error: %w", err)
	}
	return nil
}

// App returns the underlying fiber.App for advanced configuration.
func (a *Adapter) App() *fiber.App {
	return a.app
}

func init() {
	server.Register("fiber", func(cfg server.Config) (server.ServerAdapter, error) {
		return New(cfg), nil
	})
}
