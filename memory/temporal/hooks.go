package temporal

import (
	"context"

	"github.com/lookatitude/beluga-ai/memory"
)

// Hooks provides optional callback functions for temporal memory operations.
// All fields are optional -- nil hooks are skipped.
type Hooks struct {
	// OnConflictResolved is called after conflict resolution invalidates existing relations.
	// The invalidated slice contains relations that were marked as no longer valid,
	// and newRelation is the relation that triggered the invalidation.
	OnConflictResolved func(ctx context.Context, invalidated []memory.Relation, newRelation memory.Relation)

	// OnEntityMerged is called when an existing entity is updated with new information.
	// The existing parameter holds the entity before the merge, and merged holds the
	// entity after the merge.
	OnEntityMerged func(ctx context.Context, existing, merged memory.Entity)
}
