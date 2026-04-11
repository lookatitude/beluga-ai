package plancache

import "context"

// Store persists and retrieves plan templates. Implementations must be safe
// for concurrent use.
type Store interface {
	// Save persists a template. If a template with the same ID already exists,
	// it is updated.
	Save(ctx context.Context, tmpl *Template) error

	// Get retrieves a template by ID. Returns an error with code
	// ErrTemplateNotFound if the template does not exist.
	Get(ctx context.Context, id string) (*Template, error)

	// List returns all templates for the given agent ID. Returns an empty
	// slice if no templates are found.
	List(ctx context.Context, agentID string) ([]*Template, error)

	// Delete removes a template by ID. Returns nil if the template does not
	// exist.
	Delete(ctx context.Context, id string) error
}
