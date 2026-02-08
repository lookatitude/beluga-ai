package memory

import (
	"context"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/schema"
)

// CompositeOption configures a CompositeMemory.
type CompositeOption func(*CompositeMemory)

// WithCore sets the core memory tier.
func WithCore(c *Core) CompositeOption {
	return func(m *CompositeMemory) {
		m.core = c
	}
}

// WithRecall sets the recall memory tier.
func WithRecall(r *Recall) CompositeOption {
	return func(m *CompositeMemory) {
		m.recall = r
	}
}

// WithArchival sets the archival memory tier.
func WithArchival(a *Archival) CompositeOption {
	return func(m *CompositeMemory) {
		m.archival = a
	}
}

// WithGraph sets the graph memory store.
func WithGraph(g GraphStore) CompositeOption {
	return func(m *CompositeMemory) {
		m.graph = g
	}
}

// CompositeMemory combines all memory tiers into a unified Memory
// implementation. Each tier is optional — only configured tiers participate
// in Save/Load/Search operations.
type CompositeMemory struct {
	core     *Core
	recall   *Recall
	archival *Archival
	graph    GraphStore
}

// NewComposite creates a CompositeMemory with the given tier options.
func NewComposite(opts ...CompositeOption) *CompositeMemory {
	m := &CompositeMemory{}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// Core returns the core memory tier, or nil if not configured.
func (m *CompositeMemory) Core() *Core { return m.core }

// Recall returns the recall memory tier, or nil if not configured.
func (m *CompositeMemory) Recall() *Recall { return m.recall }

// Archival returns the archival memory tier, or nil if not configured.
func (m *CompositeMemory) Archival() *Archival { return m.archival }

// Graph returns the graph memory store, or nil if not configured.
func (m *CompositeMemory) Graph() GraphStore { return m.graph }

// Save implements Memory. Delegates to all configured tiers. Core memory is
// skipped (it uses explicit edits). Recall and archival both receive the
// input/output pair.
func (m *CompositeMemory) Save(ctx context.Context, input, output schema.Message) error {
	if m.recall != nil {
		if err := m.recall.Save(ctx, input, output); err != nil {
			return err
		}
	}
	if m.archival != nil {
		if err := m.archival.Save(ctx, input, output); err != nil {
			return err
		}
	}
	return nil
}

// Load implements Memory. Returns core memory messages (always in context)
// followed by recall memory results for the query.
func (m *CompositeMemory) Load(ctx context.Context, query string) ([]schema.Message, error) {
	var msgs []schema.Message
	if m.core != nil {
		coreMsgs, err := m.core.Load(ctx, query)
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, coreMsgs...)
	}
	if m.recall != nil {
		recallMsgs, err := m.recall.Load(ctx, query)
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, recallMsgs...)
	}
	return msgs, nil
}

// Search implements Memory. Delegates to archival memory for vector search.
// Returns nil if archival is not configured.
func (m *CompositeMemory) Search(ctx context.Context, query string, k int) ([]schema.Document, error) {
	if m.archival != nil {
		return m.archival.Search(ctx, query, k)
	}
	return nil, nil
}

// Clear implements Memory. Clears all configured tiers.
func (m *CompositeMemory) Clear(ctx context.Context) error {
	if m.core != nil {
		if err := m.core.Clear(ctx); err != nil {
			return err
		}
	}
	if m.recall != nil {
		if err := m.recall.Clear(ctx); err != nil {
			return err
		}
	}
	if m.archival != nil {
		if err := m.archival.Clear(ctx); err != nil {
			return err
		}
	}
	return nil
}

func init() {
	Register("composite", func(cfg config.ProviderConfig) (Memory, error) {
		// The composite registry entry creates a basic composite with default
		// core and recall tiers. For full configuration, use NewComposite directly.
		core := NewCore(CoreConfig{SelfEditable: true})
		recall := NewRecall(&inlineMessageStore{})
		return NewComposite(WithCore(core), WithRecall(recall)), nil
	})
}
