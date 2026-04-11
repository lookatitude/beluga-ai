package associative

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/rag/embedding"
	"github.com/lookatitude/beluga-ai/schema"
)

// RetroactiveUpdater evolves neighbor notes' keywords, tags, and descriptions
// when a new note is added. It uses an LLM to re-enrich neighbors in the
// context of the new note, then re-embeds and updates the store.
//
// Concurrency is bounded by the maxWorkers parameter. If maxWorkers is zero
// or negative, updates run sequentially (maxWorkers=1).
type RetroactiveUpdater struct {
	store      NoteStore
	enricher   *NoteEnricher
	embedder   embedding.Embedder
	maxWorkers int
}

// NewRetroactiveUpdater creates a RetroactiveUpdater with bounded concurrency.
func NewRetroactiveUpdater(store NoteStore, enricher *NoteEnricher, embedder embedding.Embedder, maxWorkers int) *RetroactiveUpdater {
	if maxWorkers <= 0 {
		maxWorkers = 1
	}
	return &RetroactiveUpdater{
		store:      store,
		enricher:   enricher,
		embedder:   embedder,
		maxWorkers: maxWorkers,
	}
}

// Update re-enriches the specified neighbor notes in the context of the new
// note's content. Each neighbor's keywords, tags, and description are updated
// by the LLM, then the note is re-embedded and stored. Returns the IDs of
// notes that were successfully updated.
func (u *RetroactiveUpdater) Update(ctx context.Context, newNote *schema.Note, neighborIDs []string) ([]string, error) {
	if u.enricher == nil || u.enricher.model == nil || len(neighborIDs) == 0 {
		return nil, nil
	}

	type result struct {
		id  string
		err error
	}

	results := make(chan result, len(neighborIDs))
	sem := make(chan struct{}, u.maxWorkers)

	var wg sync.WaitGroup
	for _, nid := range neighborIDs {
		wg.Add(1)
		go func(neighborID string) {
			defer wg.Done()

			// Acquire semaphore.
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				results <- result{id: neighborID, err: ctx.Err()}
				return
			}

			err := u.updateNeighbor(ctx, newNote, neighborID)
			results <- result{id: neighborID, err: err}
		}(nid)
	}

	// Close results channel once all goroutines complete.
	go func() {
		wg.Wait()
		close(results)
	}()

	var updated []string
	var firstErr error
	for r := range results {
		if r.err != nil {
			if firstErr == nil {
				firstErr = r.err
			}
			continue
		}
		updated = append(updated, r.id)
	}

	return updated, firstErr
}

// updateNeighbor re-enriches and re-embeds a single neighbor note.
func (u *RetroactiveUpdater) updateNeighbor(ctx context.Context, newNote *schema.Note, neighborID string) error {
	neighbor, err := u.store.Get(ctx, neighborID)
	if err != nil {
		return core.Errorf(core.ErrProviderDown, "associative.updater: get neighbor %q: %w", neighborID, err)
	}

	// Build enrichment prompt that includes context from the new note.
	combinedContent := fmt.Sprintf(
		"Existing note:\n%s\n\nNewly related note:\n%s\n\n"+
			"Re-analyze the existing note in light of this new related note.",
		neighbor.Content, newNote.Content,
	)

	enrichment, err := u.enricher.Enrich(ctx, combinedContent)
	if err != nil {
		return core.Errorf(core.ErrProviderDown, "associative.updater: enrich neighbor %q: %w", neighborID, err)
	}

	// Update the neighbor's metadata.
	neighbor.Keywords = enrichment.Keywords
	neighbor.Tags = enrichment.Tags
	neighbor.Description = enrichment.Description
	neighbor.UpdatedAt = time.Now().UTC()

	// Re-embed the neighbor's content.
	vec, err := u.embedder.EmbedSingle(ctx, neighbor.Content)
	if err != nil {
		return core.Errorf(core.ErrProviderDown, "associative.updater: embed neighbor %q: %w", neighborID, err)
	}
	neighbor.Embedding = vec

	if err := u.store.Update(ctx, neighbor); err != nil {
		return core.Errorf(core.ErrProviderDown, "associative.updater: update neighbor %q: %w", neighborID, err)
	}

	return nil
}
