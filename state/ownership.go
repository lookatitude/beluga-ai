package state

import (
	"context"
	"errors"
	"sync"

	"github.com/lookatitude/beluga-ai/core"
)

// ErrOwnershipDenied is returned when a non-owner attempts to write to a
// claimed key.
var ErrOwnershipDenied = errors.New("state: ownership denied")

// ErrAlreadyClaimed is returned when a key is already claimed by another owner.
var ErrAlreadyClaimed = errors.New("state: key already claimed")

// OwnershipManager tracks which agent owns which keys and enforces write
// access control.
type OwnershipManager struct {
	mu     sync.RWMutex
	claims map[string]string // key -> ownerID
}

// NewOwnershipManager creates a new OwnershipManager.
func NewOwnershipManager() *OwnershipManager {
	return &OwnershipManager{
		claims: make(map[string]string),
	}
}

// Claim registers ownerID as the owner of key. Returns ErrAlreadyClaimed if
// the key is already claimed by a different owner.
func (om *OwnershipManager) Claim(key, ownerID string) error {
	om.mu.Lock()
	defer om.mu.Unlock()

	if existing, ok := om.claims[key]; ok && existing != ownerID {
		return core.Errorf(core.ErrInvalidInput, "%w: key %q owned by %q", ErrAlreadyClaimed, key, existing)
	}
	om.claims[key] = ownerID
	return nil
}

// Release removes the ownership claim on key. Only the current owner can
// release. Returns ErrOwnershipDenied if the caller is not the owner.
func (om *OwnershipManager) Release(key, ownerID string) error {
	om.mu.Lock()
	defer om.mu.Unlock()

	existing, ok := om.claims[key]
	if !ok {
		return nil // no claim to release
	}
	if existing != ownerID {
		return core.Errorf(core.ErrInvalidInput, "%w: key %q owned by %q, not %q", ErrOwnershipDenied, key, existing, ownerID)
	}
	delete(om.claims, key)
	return nil
}

// Owner returns the owner of key, or empty string if unclaimed.
func (om *OwnershipManager) Owner(key string) string {
	om.mu.RLock()
	defer om.mu.RUnlock()
	return om.claims[key]
}

// CheckWrite returns nil if ownerID is allowed to write to key.
// Unclaimed keys are writable by anyone. Claimed keys are only writable
// by the owner.
func (om *OwnershipManager) CheckWrite(key, ownerID string) error {
	om.mu.RLock()
	defer om.mu.RUnlock()

	existing, ok := om.claims[key]
	if !ok {
		return nil // unclaimed, anyone can write
	}
	if existing != ownerID {
		return core.Errorf(core.ErrInvalidInput, "%w: key %q owned by %q, writer %q", ErrOwnershipDenied, key, existing, ownerID)
	}
	return nil
}

// WithOwnership returns middleware that enforces ownership on Set and Delete
// operations. The ownerID is extracted from the context using OwnerIDFromContext.
// Keys without ownership claims are accessible to all writers.
func WithOwnership(om *OwnershipManager) Middleware {
	return func(next Store) Store {
		return &ownedStore{next: next, om: om}
	}
}

type ownerIDKey struct{}

// WithOwnerID returns a context with the given owner ID attached.
func WithOwnerID(ctx context.Context, ownerID string) context.Context {
	return context.WithValue(ctx, ownerIDKey{}, ownerID)
}

// OwnerIDFromContext extracts the owner ID from the context.
// Returns an empty string if no owner ID is set.
func OwnerIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ownerIDKey{}).(string)
	return v
}

// ownedStore wraps a Store with ownership enforcement.
type ownedStore struct {
	next Store
	om   *OwnershipManager
}

func (s *ownedStore) Get(ctx context.Context, key string) (any, error) {
	return s.next.Get(ctx, key)
}

func (s *ownedStore) Set(ctx context.Context, key string, value any) error {
	ownerID := OwnerIDFromContext(ctx)
	if ownerID != "" {
		if err := s.om.CheckWrite(key, ownerID); err != nil {
			return err
		}
	}
	return s.next.Set(ctx, key, value)
}

func (s *ownedStore) Delete(ctx context.Context, key string) error {
	ownerID := OwnerIDFromContext(ctx)
	if ownerID != "" {
		if err := s.om.CheckWrite(key, ownerID); err != nil {
			return err
		}
	}
	return s.next.Delete(ctx, key)
}

func (s *ownedStore) Watch(ctx context.Context, key string) (<-chan StateChange, error) {
	return s.next.Watch(ctx, key)
}

func (s *ownedStore) Close() error {
	return s.next.Close()
}

var _ Store = (*ownedStore)(nil)
