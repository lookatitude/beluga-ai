package temporal

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/memory"
)

const (
	opAddRelation         = "temporal.add_relation"
	opAddTemporalRelation = "temporal.add_temporal_relation"
	opInvalidateRelation  = "temporal.invalidate_relation"

	errFromToEmpty   = "from and to entity IDs must not be empty"
	errRelTypeEmpty  = "relation type must not be empty"
)

// InMemoryStore is a thread-safe in-memory implementation of memory.TemporalGraphStore.
// It stores entities and relations in maps protected by a sync.RWMutex and supports
// all temporal query operations.
type InMemoryStore struct {
	mu        sync.RWMutex
	entities  map[string]memory.Entity
	relations []memory.Relation
	nextRelID int
}

// NewInMemoryStore creates a new empty InMemoryStore.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		entities: make(map[string]memory.Entity),
	}
}

// AddEntity adds or updates an entity in the in-memory graph. If an entity with
// the same ID already exists, its properties, type, and summary are updated.
// CreatedAt is only set on first creation.
func (s *InMemoryStore) AddEntity(ctx context.Context, entity memory.Entity) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if entity.ID == "" {
		return core.NewError("temporal.add_entity", core.ErrInvalidInput, "entity ID must not be empty", nil)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	entity = s.mergeEntity(entity)
	s.entities[entity.ID] = entity
	return nil
}

// mergeEntity merges the given entity with any existing entity of the same ID.
// It preserves the original creation time and merges properties maps.
// The caller must hold s.mu.Lock().
func (s *InMemoryStore) mergeEntity(entity memory.Entity) memory.Entity {
	existing, ok := s.entities[entity.ID]
	if !ok {
		if entity.CreatedAt.IsZero() {
			entity.CreatedAt = time.Now()
		}
		return entity
	}
	if entity.CreatedAt.IsZero() {
		entity.CreatedAt = existing.CreatedAt
	}
	entity.Properties = mergeProperties(existing.Properties, entity.Properties)
	return entity
}

// mergeProperties merges src into dst, returning a new combined map.
// If incoming is nil, existing is returned unchanged.
func mergeProperties(existing, incoming map[string]any) map[string]any {
	if existing == nil || incoming == nil {
		if incoming == nil {
			return existing
		}
		return incoming
	}
	merged := make(map[string]any, len(existing)+len(incoming))
	for k, v := range existing {
		merged[k] = v
	}
	for k, v := range incoming {
		merged[k] = v
	}
	return merged
}

// AddRelation creates a directed relationship between two entities. Both entities
// must already exist. The relation is assigned a unique ID stored in Properties["id"].
func (s *InMemoryStore) AddRelation(ctx context.Context, from, to, relation string, props map[string]any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if from == "" || to == "" {
		return core.NewError(opAddRelation, core.ErrInvalidInput, errFromToEmpty, nil)
	}
	if relation == "" {
		return core.NewError(opAddRelation, core.ErrInvalidInput, errRelTypeEmpty, nil)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.entities[from]; !ok {
		return core.NewError(opAddRelation, core.ErrNotFound, fmt.Sprintf("source entity %q not found", from), nil)
	}
	if _, ok := s.entities[to]; !ok {
		return core.NewError(opAddRelation, core.ErrNotFound, fmt.Sprintf("target entity %q not found", to), nil)
	}

	if props == nil {
		props = make(map[string]any)
	}

	s.nextRelID++
	relID := fmt.Sprintf("rel-%d", s.nextRelID)
	props["id"] = relID

	now := time.Now()
	rel := memory.Relation{
		From:       from,
		To:         to,
		Type:       relation,
		Properties: props,
		CreatedAt:  now,
		ValidAt:    now,
	}
	s.relations = append(s.relations, rel)
	return nil
}

// AddTemporalRelation creates a directed relationship with explicit temporal fields.
// Both entities must already exist. This is the preferred method when the caller knows
// the valid time and episodes for the relation.
func (s *InMemoryStore) AddTemporalRelation(ctx context.Context, rel memory.Relation) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if rel.From == "" || rel.To == "" {
		return core.NewError(opAddTemporalRelation, core.ErrInvalidInput, errFromToEmpty, nil)
	}
	if rel.Type == "" {
		return core.NewError(opAddTemporalRelation, core.ErrInvalidInput, errRelTypeEmpty, nil)
	}
	if rel.ValidAt.IsZero() {
		return core.NewError(opAddTemporalRelation, core.ErrInvalidInput, "ValidAt must not be zero", nil)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.entities[rel.From]; !ok {
		return core.NewError(opAddTemporalRelation, core.ErrNotFound, fmt.Sprintf("source entity %q not found", rel.From), nil)
	}
	if _, ok := s.entities[rel.To]; !ok {
		return core.NewError(opAddTemporalRelation, core.ErrNotFound, fmt.Sprintf("target entity %q not found", rel.To), nil)
	}

	if rel.Properties == nil {
		rel.Properties = make(map[string]any)
	}

	if _, ok := rel.Properties["id"]; !ok {
		s.nextRelID++
		rel.Properties["id"] = fmt.Sprintf("rel-%d", s.nextRelID)
	}

	if rel.CreatedAt.IsZero() {
		rel.CreatedAt = time.Now()
	}

	s.relations = append(s.relations, rel)
	return nil
}

// Query executes a simple query against the in-memory graph. The query string is
// interpreted as a case-insensitive substring match against entity types and relation types.
func (s *InMemoryStore) Query(ctx context.Context, query string) ([]memory.GraphResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	q := strings.ToLower(query)
	var entities []memory.Entity
	var relations []memory.Relation

	for _, e := range s.entities {
		if strings.Contains(strings.ToLower(e.Type), q) ||
			strings.Contains(strings.ToLower(e.ID), q) ||
			strings.Contains(strings.ToLower(e.Summary), q) {
			entities = append(entities, e)
		}
	}

	for _, r := range s.relations {
		if r.ExpiredAt != nil {
			continue // skip system-expired relations
		}
		if strings.Contains(strings.ToLower(r.Type), q) {
			relations = append(relations, r)
		}
	}

	return []memory.GraphResult{{
		Entities:  entities,
		Relations: relations,
	}}, nil
}

// Neighbors returns all entities and relations within the given depth from the
// specified entity, using breadth-first traversal. Only active relations (not
// system-expired) are traversed.
func (s *InMemoryStore) Neighbors(ctx context.Context, entityID string, depth int) ([]memory.Entity, []memory.Relation, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	if depth <= 0 {
		depth = 1
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.entities[entityID]; !ok {
		return nil, nil, core.NewError("temporal.neighbors", core.ErrNotFound, fmt.Sprintf("entity %q not found", entityID), nil)
	}

	visited := map[string]bool{entityID: true}
	relSeen := map[string]bool{}
	frontier := []string{entityID}
	var resultEntities []memory.Entity
	var resultRelations []memory.Relation

	for d := 0; d < depth && len(frontier) > 0; d++ {
		frontier, resultEntities, resultRelations = s.expandFrontier(
			frontier, visited, relSeen, resultEntities, resultRelations,
		)
	}

	return resultEntities, resultRelations, nil
}

// expandFrontier performs one BFS level expansion, returning the next frontier
// and updated result slices.
func (s *InMemoryStore) expandFrontier(
	frontier []string,
	visited map[string]bool,
	relSeen map[string]bool,
	resultEntities []memory.Entity,
	resultRelations []memory.Relation,
) ([]string, []memory.Entity, []memory.Relation) {
	var nextFrontier []string
	for _, nodeID := range frontier {
		for _, rel := range s.relations {
			if rel.ExpiredAt != nil {
				continue
			}
			neighborID := neighborOf(rel, nodeID)
			if neighborID == "" {
				continue
			}
			resultRelations = deduplicateRelation(rel, relSeen, resultRelations)
			if !visited[neighborID] {
				visited[neighborID] = true
				if e, ok := s.entities[neighborID]; ok {
					resultEntities = append(resultEntities, e)
					nextFrontier = append(nextFrontier, neighborID)
				}
			}
		}
	}
	return nextFrontier, resultEntities, resultRelations
}

// neighborOf returns the neighbor entity ID from a relation given a focal node.
// Returns an empty string if the node is not part of the relation.
func neighborOf(rel memory.Relation, nodeID string) string {
	if rel.From == nodeID {
		return rel.To
	}
	if rel.To == nodeID {
		return rel.From
	}
	return ""
}

// deduplicateRelation appends rel to results only if it has not been seen before.
func deduplicateRelation(rel memory.Relation, seen map[string]bool, results []memory.Relation) []memory.Relation {
	if relID, ok := rel.Properties["id"].(string); ok && relID != "" {
		if seen[relID] {
			return results
		}
		seen[relID] = true
	}
	return append(results, rel)
}

// QueryAsOf returns entities and relations that were valid at the specified time.
// A relation is considered valid at a point in time if:
//   - ValidAt <= validTime, AND
//   - InvalidAt is nil OR InvalidAt > validTime, AND
//   - ExpiredAt is nil (not system-expired)
//
// The query string filters by type/ID substring match (same as Query).
func (s *InMemoryStore) QueryAsOf(ctx context.Context, query string, validTime time.Time, opts ...memory.QueryOption) ([]memory.Entity, []memory.Relation, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	if validTime.IsZero() {
		return nil, nil, core.NewError("temporal.query_as_of", core.ErrInvalidInput, "validTime must not be zero", nil)
	}

	qopts := memory.ApplyQueryOptions(opts)

	s.mu.RLock()
	defer s.mu.RUnlock()

	q := strings.ToLower(query)
	entities := s.filterEntitiesAsOf(q, validTime)
	relations := s.filterRelationsAsOf(q, validTime, qopts.Limit)

	return entities, relations, nil
}

// filterEntitiesAsOf returns entities that existed at validTime matching the query.
func (s *InMemoryStore) filterEntitiesAsOf(q string, validTime time.Time) []memory.Entity {
	var entities []memory.Entity
	for _, e := range s.entities {
		if !e.CreatedAt.IsZero() && e.CreatedAt.After(validTime) {
			continue
		}
		if q == "" || strings.Contains(strings.ToLower(e.Type), q) ||
			strings.Contains(strings.ToLower(e.ID), q) ||
			strings.Contains(strings.ToLower(e.Summary), q) {
			entities = append(entities, e)
		}
	}
	return entities
}

// filterRelationsAsOf returns relations valid at validTime matching the query, up to limit.
func (s *InMemoryStore) filterRelationsAsOf(q string, validTime time.Time, limit int) []memory.Relation {
	var relations []memory.Relation
	for _, r := range s.relations {
		if r.ExpiredAt != nil {
			continue
		}
		if !r.ValidAt.After(validTime) && (r.InvalidAt == nil || r.InvalidAt.After(validTime)) {
			if q == "" || strings.Contains(strings.ToLower(r.Type), q) {
				relations = append(relations, r)
				if len(relations) >= limit {
					break
				}
			}
		}
	}
	return relations
}

// InvalidateRelation marks a relation as no longer valid. The relation is identified
// by its ID stored in Properties["id"]. InvalidAt is set to the provided time and
// ExpiredAt is set to the current system time.
func (s *InMemoryStore) InvalidateRelation(ctx context.Context, relationID string, invalidAt time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if relationID == "" {
		return core.NewError(opInvalidateRelation, core.ErrInvalidInput, "relation ID must not be empty", nil)
	}
	if invalidAt.IsZero() {
		return core.NewError(opInvalidateRelation, core.ErrInvalidInput, "invalidAt must not be zero", nil)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.relations {
		if id, ok := s.relations[i].Properties["id"]; ok && id == relationID {
			// Idempotent: if the relation is already invalidated, preserve the
			// original InvalidAt/ExpiredAt timestamps rather than overwriting.
			if s.relations[i].ExpiredAt != nil {
				return nil
			}
			now := time.Now()
			s.relations[i].InvalidAt = &invalidAt
			s.relations[i].ExpiredAt = &now
			return nil
		}
	}

	return core.NewError(opInvalidateRelation, core.ErrNotFound, fmt.Sprintf("relation %q not found", relationID), nil)
}

// History returns all versions of relations between two entities, including
// invalidated ones, ordered by ValidAt ascending.
func (s *InMemoryStore) History(ctx context.Context, fromID, toID string) ([]memory.Relation, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if fromID == "" || toID == "" {
		return nil, core.NewError("temporal.history", core.ErrInvalidInput, "fromID and toID must not be empty", nil)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []memory.Relation
	for _, r := range s.relations {
		if r.From == fromID && r.To == toID {
			result = append(result, r)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].ValidAt.Before(result[j].ValidAt)
	})

	return result, nil
}

// Clear removes all entities and relations from the in-memory store.
func (s *InMemoryStore) Clear(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entities = make(map[string]memory.Entity)
	s.relations = nil
	s.nextRelID = 0
	return nil
}

// Compile-time check that InMemoryStore implements TemporalGraphStore.
var _ memory.TemporalGraphStore = (*InMemoryStore)(nil)
