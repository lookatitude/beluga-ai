package prompt

import (
	"github.com/lookatitude/beluga-ai/schema"
)

// TemplateInfo holds summary information about a registered template.
type TemplateInfo struct {
	// Name is the unique name of the template.
	Name string
	// Version is the template's version string.
	Version string
	// Metadata holds arbitrary key-value pairs from the template.
	Metadata map[string]any
}

// PromptManager provides versioned access to prompt templates.
// Implementations load templates from various backends (filesystem, database, etc.)
// and support rendering templates into message sequences.
type PromptManager interface {
	// Get retrieves a template by name and version. If version is empty,
	// the latest version is returned.
	Get(name string, version string) (*Template, error)

	// Render retrieves a template by name (latest version), renders it with
	// the given variables, and returns the result as a slice of schema.Message.
	// The rendered text is returned as a single SystemMessage.
	Render(name string, vars map[string]any) ([]schema.Message, error)

	// List returns summary information for all available templates.
	List() []TemplateInfo
}
