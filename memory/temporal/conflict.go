package temporal

import (
	"context"
	"time"

	"github.com/lookatitude/beluga-ai/memory"
)

// ConflictResolver determines how to handle temporal conflicts between relations.
// When a new relation is added that may contradict existing relations, the resolver
// decides which relations should be invalidated.
type ConflictResolver interface {
	// Resolve examines the newRelation against a set of candidate relations and
	// returns the list of candidates that should be invalidated. The returned
	// relations will have their InvalidAt and ExpiredAt fields set accordingly.
	Resolve(ctx context.Context, newRelation *memory.Relation, candidates []memory.Relation) ([]memory.Relation, error)
}

// TemporalResolver implements the Graphiti 2-condition invalidation algorithm.
// For each candidate relation, it checks:
//  1. If the candidate is already superseded (InvalidAt != nil && InvalidAt <= newRelation.ValidAt),
//     the candidate and new relation coexist without conflict.
//  2. If the new relation is already invalidated before the candidate became valid
//     (newRelation.InvalidAt != nil && newRelation.InvalidAt <= candidate.ValidAt),
//     they coexist without conflict (non-overlapping time ranges).
//  3. Otherwise, if the candidate's ValidAt < newRelation.ValidAt, the candidate
//     is invalidated: its InvalidAt is set to newRelation.ValidAt and its ExpiredAt
//     is set to the current system time.
type TemporalResolver struct{}

// NewTemporalResolver creates a new TemporalResolver.
func NewTemporalResolver() *TemporalResolver {
	return &TemporalResolver{}
}

// Resolve applies the Graphiti 2-condition algorithm to determine which candidate
// relations should be invalidated by the new relation.
func (r *TemporalResolver) Resolve(_ context.Context, newRelation *memory.Relation, candidates []memory.Relation) ([]memory.Relation, error) {
	if newRelation == nil {
		return nil, nil
	}

	var invalidated []memory.Relation
	now := time.Now()

	for _, candidate := range candidates {
		// Condition 1: candidate already superseded -- coexist.
		if candidate.InvalidAt != nil && !candidate.InvalidAt.After(newRelation.ValidAt) {
			continue
		}

		// Condition 2: new relation ends before candidate starts -- non-overlapping, coexist.
		if newRelation.InvalidAt != nil && !newRelation.InvalidAt.After(candidate.ValidAt) {
			continue
		}

		// Condition 3: candidate is older -- invalidate it.
		if candidate.ValidAt.Before(newRelation.ValidAt) {
			invalidAt := newRelation.ValidAt
			candidate.InvalidAt = &invalidAt
			expiredAt := now
			candidate.ExpiredAt = &expiredAt
			invalidated = append(invalidated, candidate)
		}
	}

	return invalidated, nil
}

// Compile-time check that TemporalResolver implements ConflictResolver.
var _ ConflictResolver = (*TemporalResolver)(nil)
