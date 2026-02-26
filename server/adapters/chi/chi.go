package chi

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/internal/httputil"
	"github.com/lookatitude/beluga-ai/server"
)

// Adapter implements server.ServerAdapter using the Chi router.
type Adapter struct {
	router chi.Router
	lc     httputil.ServerLifecycle
	cfg    server.Config
}

// Compile-time interface check.
var _ server.ServerAdapter = (*Adapter)(nil)

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
	a.router.Handle(path+"/*", stripped)
	return nil
}

// RegisterHandler registers a raw http.Handler at the given path.
func (a *Adapter) RegisterHandler(path string, handler http.Handler) error {
	if handler == nil {
		return fmt.Errorf("server/chi: handler must not be nil")
	}
	a.router.Handle(path, handler)
	return nil
}

// Serve starts the HTTP server on the given address. It blocks until the
// server exits or the context is canceled.
func (a *Adapter) Serve(ctx context.Context, addr string) error {
	return a.lc.Serve(ctx, addr, a.router,
		a.cfg.ReadTimeout, a.cfg.WriteTimeout, a.cfg.IdleTimeout,
		"server/chi")
}

// Shutdown gracefully shuts down the server.
func (a *Adapter) Shutdown(ctx context.Context) error {
	return a.lc.Shutdown(ctx, "server/chi")
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
