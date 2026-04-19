package shared

import (
	"context"
	"fmt"
	"sync"

	"github.com/lookatitude/beluga-ai/v2/core"
)

// SharedStore is the storage backend for shared memory fragments.
// Implementations must be safe for concurrent use.
type SharedStore interface {
	// WriteFragment persists a fragment. If the fragment's key already exists,
	// the store updates it in place and increments the version.
	WriteFragment(ctx context.Context, frag *Fragment) error

	// ReadFragment retrieves a fragment by key. Returns a core.ErrNotFound
	// error if the key does not exist.
	ReadFragment(ctx context.Context, key string) (*Fragment, error)

	// ListFragments returns all fragments matching the given scope. If scope
	// is empty, all fragments are returned.
	ListFragments(ctx context.Context, scope Scope) ([]*Fragment, error)

	// DeleteFragment removes the fragment with the given key. Deleting a
	// non-existent key is a no-op.
	DeleteFragment(ctx context.Context, key string) error

	// UpdateProvenance updates only the provenance field of the fragment
	// identified by key, without bumping the version or applying conflict
	// policy. This is used internally after a write to attach a provenance
	// hash that reflects the final stored content (for AppendOnly the
	// post-write content is the accumulated string, not the delta).
	// Returns core.ErrNotFound if the key does not exist.
	UpdateProvenance(ctx context.Context, key string, prov *Provenance) error
}

// Compile-time check.
var _ SharedStore = (*InMemorySharedStore)(nil)

// InMemorySharedStore is a thread-safe, in-memory implementation of SharedStore.
// It is suitable for development and testing.
type InMemorySharedStore struct {
	mu        sync.RWMutex
	fragments map[string]*Fragment
}

// NewInMemorySharedStore creates a new empty in-memory store.
func NewInMemorySharedStore() *InMemorySharedStore {
	return &InMemorySharedStore{
		fragments: make(map[string]*Fragment),
	}
}

// WriteFragment stores the fragment, incrementing its version if it already
// exists. For RejectOnConflict policy, the write fails if the provided
// fragment's version does not match the stored version. For AppendOnly,
// the new content is appended to the existing content.
func (s *InMemorySharedStore) WriteFragment(ctx context.Context, frag *Fragment) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	existing, ok := s.fragments[frag.Key]
	if !ok {
		// New fragment.
		frag.Version = 1
		stored := cloneFragment(frag)
		s.fragments[frag.Key] = stored
		frag.Version = stored.Version
		return nil
	}

	// Apply conflict policy.
	switch frag.ConflictPolicy {
	case RejectOnConflict:
		if frag.Version != existing.Version {
			return core.NewError(
				"shared.store.write",
				core.ErrInvalidInput,
				fmt.Sprintf("version conflict: expected %d, got %d", existing.Version, frag.Version),
				nil,
			)
		}
		existing.Content = frag.Content
	case AppendOnly:
		existing.Content = existing.Content + frag.Content
	default: // LastWriteWins or unset
		existing.Content = frag.Content
	}

	existing.Version++
	existing.AuthorID = frag.AuthorID
	existing.UpdatedAt = frag.UpdatedAt
	existing.Provenance = frag.Provenance
	existing.Metadata = frag.Metadata
	existing.Scope = frag.Scope
	existing.Readers = frag.Readers
	existing.Writers = frag.Writers
	existing.ConflictPolicy = frag.ConflictPolicy

	// Reflect the new version back to the caller.
	frag.Version = existing.Version
	frag.Content = existing.Content

	return nil
}

// ReadFragment returns the fragment for the given key, or a core.ErrNotFound
// error if it does not exist.
func (s *InMemorySharedStore) ReadFragment(ctx context.Context, key string) (*Fragment, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	frag, ok := s.fragments[key]
	if !ok {
		return nil, core.NewError(
			"shared.store.read",
			core.ErrNotFound,
			fmt.Sprintf("fragment %q not found", key),
			nil,
		)
	}
	return cloneFragment(frag), nil
}

// ListFragments returns all fragments matching the given scope. If scope is
// empty, all fragments are returned.
func (s *InMemorySharedStore) ListFragments(ctx context.Context, scope Scope) ([]*Fragment, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Fragment, 0, len(s.fragments))
	for _, frag := range s.fragments {
		if scope == "" || frag.Scope == scope {
			result = append(result, cloneFragment(frag))
		}
	}
	return result, nil
}

// UpdateProvenance updates only the provenance field of an existing
// fragment without touching version, content, or conflict policy.
func (s *InMemorySharedStore) UpdateProvenance(ctx context.Context, key string, prov *Provenance) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.fragments[key]
	if !ok {
		return core.NewError(
			"shared.store.update_provenance",
			core.ErrNotFound,
			fmt.Sprintf("fragment %q not found", key),
			nil,
		)
	}
	if prov != nil {
		p := *prov
		existing.Provenance = &p
	} else {
		existing.Provenance = nil
	}
	return nil
}

// DeleteFragment removes the fragment with the given key. If the key does
// not exist, this is a no-op.
func (s *InMemorySharedStore) DeleteFragment(ctx context.Context, key string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.fragments, key)
	return nil
}

// cloneFragment returns a deep copy of a fragment to prevent callers from
// mutating the store's internal state.
func cloneFragment(f *Fragment) *Fragment {
	c := *f
	if f.Readers != nil {
		c.Readers = make([]string, len(f.Readers))
		copy(c.Readers, f.Readers)
	}
	if f.Writers != nil {
		c.Writers = make([]string, len(f.Writers))
		copy(c.Writers, f.Writers)
	}
	if f.Metadata != nil {
		c.Metadata = make(map[string]string, len(f.Metadata))
		for k, v := range f.Metadata {
			c.Metadata[k] = v
		}
	}
	if f.Provenance != nil {
		p := *f.Provenance
		c.Provenance = &p
	}
	return &c
}
