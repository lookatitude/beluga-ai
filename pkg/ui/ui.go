// Package ui provides user interface components for the Beluga AI framework.
// This package contains web-based interfaces and components for interacting with agents and chains.
package ui

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
)

// UI represents the user interface for the Beluga AI framework.
type UI struct {
	server      *http.Server
	templates   *template.Template
	staticFiles http.FileSystem
	agents      []AgentUI
	chains      []ChainUI
}

// AgentUI represents the UI configuration for an agent.
type AgentUI struct {
	Name        string
	Description string
	Inputs      []InputField
	Outputs     []OutputField
}

// ChainUI represents the UI configuration for a chain.
type ChainUI struct {
	Name        string
	Description string
	Steps       []StepUI
}

// InputField represents an input field in the UI.
type InputField struct {
	Name        string
	Type        string // "text", "textarea", "select", etc.
	Label       string
	Required    bool
	Placeholder string
	Options     []string // for select fields
}

// OutputField represents an output field in the UI.
type OutputField struct {
	Name  string
	Type  string // "text", "json", "markdown", etc.
	Label string
}

// StepUI represents a step in a chain's UI.
type StepUI struct {
	Name        string
	Description string
	Inputs      []InputField
	Outputs     []OutputField
}

// NewUI creates a new UI instance.
func NewUI(port int) *UI {
	ui := &UI{
		templates:   template.Must(template.ParseGlob("templates/*.html")),
		staticFiles: http.Dir("static"),
		agents:      []AgentUI{},
		chains:      []ChainUI{},
	}

	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(ui.handleIndex))
	mux.Handle("/agents/", http.HandlerFunc(ui.handleAgents))
	mux.Handle("/chains/", http.HandlerFunc(ui.handleChains))
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(ui.staticFiles)))

	ui.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	return ui
}

// RegisterAgent registers an agent with the UI.
func (ui *UI) RegisterAgent(agent AgentUI) {
	ui.agents = append(ui.agents, agent)
}

// RegisterChain registers a chain with the UI.
func (ui *UI) RegisterChain(chain ChainUI) {
	ui.chains = append(ui.chains, chain)
}

// Start starts the UI server.
func (ui *UI) Start(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		ui.server.Shutdown(context.Background())
	}()

	return ui.server.ListenAndServe()
}

// Stop stops the UI server.
func (ui *UI) Stop(ctx context.Context) error {
	return ui.server.Shutdown(ctx)
}

// Handler methods (placeholders)
func (ui *UI) handleIndex(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Agents []AgentUI
		Chains []ChainUI
	}{
		Agents: ui.agents,
		Chains: ui.chains,
	}

	ui.templates.ExecuteTemplate(w, "index.html", data)
}

func (ui *UI) handleAgents(w http.ResponseWriter, r *http.Request) {
	// Extract agent name from URL and render agent interface
	w.WriteHeader(http.StatusNotImplemented)
}

func (ui *UI) handleChains(w http.ResponseWriter, r *http.Request) {
	// Extract chain name from URL and render chain interface
	w.WriteHeader(http.StatusNotImplemented)
}
