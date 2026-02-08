package memory

import (
	"context"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/schema"
)

// MessageStore is the interface for storing and retrieving conversation
// messages. Implementations provide the persistence backend for recall memory.
// All methods must be safe for concurrent use.
type MessageStore interface {
	// Append adds a message to the store.
	Append(ctx context.Context, msg schema.Message) error

	// Search finds messages matching the query, returning at most k results.
	// The matching strategy (substring, semantic, etc.) depends on the
	// implementation.
	Search(ctx context.Context, query string, k int) ([]schema.Message, error)

	// All returns all stored messages in chronological order.
	All(ctx context.Context) ([]schema.Message, error)

	// Clear removes all messages from the store.
	Clear(ctx context.Context) error
}

// Recall implements the MemGPT recall memory tier. It provides searchable
// conversation history, storing every message exchanged during agent
// interactions. Recall memory supports both full retrieval and query-based
// search over past messages.
type Recall struct {
	store MessageStore
}

// NewRecall creates a new Recall memory backed by the given MessageStore.
func NewRecall(store MessageStore) *Recall {
	return &Recall{store: store}
}

// Save implements Memory. Appends both the input and output messages to the
// underlying store.
func (r *Recall) Save(ctx context.Context, input, output schema.Message) error {
	if err := r.store.Append(ctx, input); err != nil {
		return err
	}
	return r.store.Append(ctx, output)
}

// Load implements Memory. Searches recall memory for messages relevant to the
// query. If query is empty, returns all messages.
func (r *Recall) Load(ctx context.Context, query string) ([]schema.Message, error) {
	if query == "" {
		return r.store.All(ctx)
	}
	return r.store.Search(ctx, query, 20)
}

// Search implements Memory. Recall memory does not store documents, so this
// always returns nil.
func (r *Recall) Search(ctx context.Context, query string, k int) ([]schema.Document, error) {
	return nil, nil
}

// Clear implements Memory. Removes all messages from the underlying store.
func (r *Recall) Clear(ctx context.Context) error {
	return r.store.Clear(ctx)
}

func init() {
	Register("recall", func(cfg config.ProviderConfig) (Memory, error) {
		// Default recall uses an in-memory message store.
		// Production usage should pass a store via composite memory.
		return NewRecall(&inlineMessageStore{}), nil
	})
}

// inlineMessageStore is a minimal in-memory MessageStore used as the default
// for the "recall" registry entry. For production use, prefer the stores in
// memory/stores/inmemory or a database-backed store.
type inlineMessageStore struct {
	msgs []schema.Message
}

func (s *inlineMessageStore) Append(_ context.Context, msg schema.Message) error {
	s.msgs = append(s.msgs, msg)
	return nil
}

func (s *inlineMessageStore) Search(_ context.Context, query string, k int) ([]schema.Message, error) {
	var results []schema.Message
	for _, msg := range s.msgs {
		if matchesQuery(msg, query) {
			results = append(results, msg)
			if len(results) >= k {
				break
			}
		}
	}
	return results, nil
}

func (s *inlineMessageStore) All(_ context.Context) ([]schema.Message, error) {
	cp := make([]schema.Message, len(s.msgs))
	copy(cp, s.msgs)
	return cp, nil
}

func (s *inlineMessageStore) Clear(_ context.Context) error {
	s.msgs = nil
	return nil
}
