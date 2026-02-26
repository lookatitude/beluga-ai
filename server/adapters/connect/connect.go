package connect

import (
	"context"
	"fmt"
	"net/http"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/internal/httputil"
	"github.com/lookatitude/beluga-ai/server"
)

// Adapter implements server.ServerAdapter using Connect-Go over net/http.
// It provides HTTP/1.1 + HTTP/2 support compatible with gRPC, gRPC-Web,
// and Connect protocol clients.
type Adapter struct {
	mux *http.ServeMux
	lc  httputil.ServerLifecycle
	cfg server.Config
}

// Compile-time interface check.
var _ server.ServerAdapter = (*Adapter)(nil)

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
	a.mux.Handle(path+"/", stripped)
	return nil
}

// RegisterHandler registers a raw http.Handler at the given path.
func (a *Adapter) RegisterHandler(path string, handler http.Handler) error {
	if handler == nil {
		return fmt.Errorf("server/connect: handler must not be nil")
	}
	a.mux.Handle(path, handler)
	return nil
}

// Serve starts the HTTP server on the given address. It supports both HTTP/1.1
// and HTTP/2, making it compatible with Connect, gRPC, and gRPC-Web clients.
// It blocks until the server exits or the context is canceled.
func (a *Adapter) Serve(ctx context.Context, addr string) error {
	return a.lc.Serve(ctx, addr, a.mux,
		a.cfg.ReadTimeout, a.cfg.WriteTimeout, a.cfg.IdleTimeout,
		"server/connect")
}

// Shutdown gracefully shuts down the server.
func (a *Adapter) Shutdown(ctx context.Context) error {
	return a.lc.Shutdown(ctx, "server/connect")
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
