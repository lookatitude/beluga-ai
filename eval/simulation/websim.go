package simulation

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/lookatitude/beluga-ai/v2/core"
)

// Page represents a web page in the simulated environment.
type Page struct {
	// Path is the URL path (e.g., "/login", "/dashboard").
	Path string

	// Title is the page title.
	Title string

	// Content is the textual content of the page.
	Content string

	// Forms maps form names to their field definitions.
	Forms map[string][]FormField

	// Links lists paths this page links to.
	Links []string
}

// FormField defines a single form field.
type FormField struct {
	// Name is the field name.
	Name string

	// Type is the field type (e.g., "text", "password", "submit").
	Type string

	// Required indicates whether the field must be filled.
	Required bool
}

// Compile-time interface check.
var _ SimEnvironment = (*WebSimulator)(nil)

// WebSimulator is a mock web environment that simulates page navigation and
// form submission. Pages are defined statically and actions are interpreted
// as navigation or form submission commands.
type WebSimulator struct {
	mu          sync.Mutex
	pages       map[string]*Page
	currentPath string
	formData    map[string]string
	submissions []map[string]string
	closed      bool
}

// WebSimOption configures a WebSimulator.
type WebSimOption func(*WebSimulator)

// WithPages sets the available pages in the web simulator.
func WithPages(pages ...*Page) WebSimOption {
	return func(w *WebSimulator) {
		for _, p := range pages {
			w.pages[p.Path] = p
		}
	}
}

// WithStartPage sets the initial page path.
func WithStartPage(path string) WebSimOption {
	return func(w *WebSimulator) {
		w.currentPath = path
	}
}

// NewWebSimulator creates a new WebSimulator with the given options.
func NewWebSimulator(opts ...WebSimOption) *WebSimulator {
	w := &WebSimulator{
		pages:       make(map[string]*Page),
		currentPath: "/",
		formData:    make(map[string]string),
	}
	for _, opt := range opts {
		opt(w)
	}
	return w
}

// Reset reinitializes the simulator to its starting state.
func (w *WebSimulator) Reset(_ context.Context) (*Observation, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.formData = make(map[string]string)
	w.submissions = nil
	return w.observeLocked()
}

// Step processes an action string. Supported actions:
//   - "navigate <path>": Navigate to a page
//   - "fill <field> <value>": Fill a form field
//   - "submit <form>": Submit a form
func (w *WebSimulator) Step(_ context.Context, action string) (*Observation, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return nil, core.Errorf(core.ErrProviderDown, "websim: environment is closed")
	}

	parts := strings.Fields(action)
	if len(parts) == 0 {
		return nil, core.Errorf(core.ErrInvalidInput, "websim: empty action")
	}

	switch parts[0] {
	case "navigate":
		if len(parts) < 2 {
			return nil, core.Errorf(core.ErrInvalidInput, "websim: navigate requires a path")
		}
		path := parts[1]
		if _, ok := w.pages[path]; !ok {
			return &Observation{
				Text: fmt.Sprintf("Page not found: %s", path),
				Data: map[string]any{"error": "not_found", "path": path},
			}, nil
		}
		w.currentPath = path
		w.formData = make(map[string]string)

	case "fill":
		if len(parts) < 3 {
			return nil, core.Errorf(core.ErrInvalidInput, "websim: fill requires field and value")
		}
		field := parts[1]
		value := strings.Join(parts[2:], " ")
		w.formData[field] = value

	case "submit":
		if len(parts) < 2 {
			return nil, core.Errorf(core.ErrInvalidInput, "websim: submit requires a form name")
		}
		formName := parts[1]
		page := w.pages[w.currentPath]
		if page == nil {
			return nil, core.Errorf(core.ErrNotFound, "websim: no current page")
		}
		fields, ok := page.Forms[formName]
		if !ok {
			return &Observation{
				Text: fmt.Sprintf("Form not found: %s", formName),
				Data: map[string]any{"error": "form_not_found", "form": formName},
			}, nil
		}

		// Validate required fields.
		for _, f := range fields {
			if f.Required {
				if _, filled := w.formData[f.Name]; !filled {
					return &Observation{
						Text: fmt.Sprintf("Required field missing: %s", f.Name),
						Data: map[string]any{"error": "validation", "field": f.Name},
					}, nil
				}
			}
		}

		submission := make(map[string]string, len(w.formData))
		for k, v := range w.formData {
			submission[k] = v
		}
		w.submissions = append(w.submissions, submission)
		w.formData = make(map[string]string)

	default:
		return nil, core.Errorf(core.ErrInvalidInput, "websim: unknown action %q", parts[0])
	}

	return w.observeLocked()
}

// Observe returns the current page observation.
func (w *WebSimulator) Observe(_ context.Context) (*Observation, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.observeLocked()
}

// Close releases simulator resources.
func (w *WebSimulator) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.closed = true
	return nil
}

// Submissions returns all form submissions recorded so far.
func (w *WebSimulator) Submissions() []map[string]string {
	w.mu.Lock()
	defer w.mu.Unlock()
	result := make([]map[string]string, len(w.submissions))
	copy(result, w.submissions)
	return result
}

// observeLocked returns an observation for the current page. Must be called
// with w.mu held.
func (w *WebSimulator) observeLocked() (*Observation, error) {
	page, ok := w.pages[w.currentPath]
	if !ok {
		return &Observation{
			Text: fmt.Sprintf("Current page not found: %s", w.currentPath),
			Data: map[string]any{"path": w.currentPath, "error": "not_found"},
		}, nil
	}

	data := map[string]any{
		"path":    page.Path,
		"title":   page.Title,
		"content": page.Content,
		"links":   page.Links,
	}
	if len(page.Forms) > 0 {
		formNames := make([]string, 0, len(page.Forms))
		for name := range page.Forms {
			formNames = append(formNames, name)
		}
		data["forms"] = formNames
	}

	return &Observation{
		Text: fmt.Sprintf("[%s] %s\n%s", page.Path, page.Title, page.Content),
		Data: data,
	}, nil
}
