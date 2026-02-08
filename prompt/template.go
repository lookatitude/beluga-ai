// Package prompt provides prompt template management and cache-optimized prompt
// building for the Beluga AI framework. It supports template rendering with
// Go's text/template syntax, versioned template management via pluggable
// providers, and a builder that orders prompt content for optimal LLM cache hits.
package prompt

import (
	"bytes"
	"errors"
	"fmt"
	"text/template"
)

// Template represents a versioned prompt template with Go text/template syntax.
// Templates can define default variable values and carry arbitrary metadata.
type Template struct {
	// Name uniquely identifies this template.
	Name string `json:"name"`
	// Version is the semantic version of this template (e.g., "1.0.0").
	Version string `json:"version"`
	// Content is the template body using Go text/template syntax.
	Content string `json:"content"`
	// Variables holds default values for template variables.
	Variables map[string]string `json:"variables,omitempty"`
	// Metadata holds arbitrary key-value pairs for template organization.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// Validate checks that the template's content is parseable as a Go text/template.
// It returns an error if the name or content is empty, or if parsing fails.
func (t *Template) Validate() error {
	if t.Name == "" {
		return errors.New("prompt: template name is required")
	}
	if t.Content == "" {
		return errors.New("prompt: template content is required")
	}
	_, err := template.New(t.Name).Parse(t.Content)
	if err != nil {
		return fmt.Errorf("prompt: template %q parse error: %w", t.Name, err)
	}
	return nil
}

// Render executes the template with the provided variables. Default variable
// values from the template's Variables field are used for any keys not present
// in vars. The rendered output is returned as a string.
func (t *Template) Render(vars map[string]any) (string, error) {
	if err := t.Validate(); err != nil {
		return "", err
	}

	// Merge defaults with provided vars; provided vars take precedence.
	merged := make(map[string]any, len(t.Variables)+len(vars))
	for k, v := range t.Variables {
		merged[k] = v
	}
	for k, v := range vars {
		merged[k] = v
	}

	tmpl, err := template.New(t.Name).Parse(t.Content)
	if err != nil {
		return "", fmt.Errorf("prompt: template %q parse error: %w", t.Name, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, merged); err != nil {
		return "", fmt.Errorf("prompt: template %q execute error: %w", t.Name, err)
	}

	return buf.String(), nil
}
