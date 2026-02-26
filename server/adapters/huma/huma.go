package huma

import (
	"context"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/internal/httputil"
	"github.com/lookatitude/beluga-ai/server"
)

// Adapter implements server.ServerAdapter using Huma with the stdlib net/http mux.
type Adapter struct {
	mux *http.ServeMux
	api huma.API
	lc  httputil.ServerLifecycle
	cfg server.Config
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
	a.mux.Handle(path+"/", stripped)
	return nil
}

// RegisterHandler registers a raw http.Handler at the given path.
func (a *Adapter) RegisterHandler(path string, handler http.Handler) error {
	if handler == nil {
		return fmt.Errorf("server/huma: handler must not be nil")
	}
	a.mux.Handle(path, handler)
	return nil
}

// Serve starts the HTTP server on the given address. It blocks until the
// server exits or the context is canceled.
func (a *Adapter) Serve(ctx context.Context, addr string) error {
	return a.lc.Serve(ctx, addr, a.mux,
		a.cfg.ReadTimeout, a.cfg.WriteTimeout, a.cfg.IdleTimeout,
		"server/huma")
}

// Shutdown gracefully shuts down the server.
func (a *Adapter) Shutdown(ctx context.Context) error {
	return a.lc.Shutdown(ctx, "server/huma")
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
