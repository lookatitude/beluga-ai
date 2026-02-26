package echo

import (
	"context"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/internal/httputil"
	"github.com/lookatitude/beluga-ai/server"
)

// Adapter implements server.ServerAdapter using the Echo HTTP framework.
type Adapter struct {
	echo *echo.Echo
	lc   httputil.ServerLifecycle
	cfg  server.Config
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
	a.echo.Any(path+"/*", echo.WrapHandler(stripped))
	return nil
}

// RegisterHandler registers a raw http.Handler at the given path.
func (a *Adapter) RegisterHandler(path string, handler http.Handler) error {
	if handler == nil {
		return fmt.Errorf("server/echo: handler must not be nil")
	}
	a.echo.Any(path, echo.WrapHandler(handler))
	return nil
}

// Serve starts the HTTP server on the given address. It blocks until the
// server exits or the context is canceled.
func (a *Adapter) Serve(ctx context.Context, addr string) error {
	return a.lc.Serve(ctx, addr, a.echo,
		a.cfg.ReadTimeout, a.cfg.WriteTimeout, a.cfg.IdleTimeout,
		"server/echo")
}

// Shutdown gracefully shuts down the server.
func (a *Adapter) Shutdown(ctx context.Context) error {
	return a.lc.Shutdown(ctx, "server/echo")
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
