package associative

import (
	"context"

	"github.com/lookatitude/beluga-ai/schema"
)

// LinkManager finds the top-k most similar notes for a given note and creates
// bidirectional links between them. It operates on a NoteStore and uses the
// store's Search method for cosine similarity ranking.
type LinkManager struct {
	store      NoteStore
	candidates int
}

// NewLinkManager creates a LinkManager that links each note to up to
// candidates similar notes.
func NewLinkManager(store NoteStore, candidates int) *LinkManager {
	if candidates <= 0 {
		candidates = 10
	}
	return &LinkManager{
		store:      store,
		candidates: candidates,
	}
}

// Link finds the top-k most similar notes to the given note and creates
// bidirectional links. The note itself is excluded from candidates. Both
// the target note and each linked neighbor are updated in the store.
// Returns the IDs of notes that were linked.
func (lm *LinkManager) Link(ctx context.Context, note *schema.Note) ([]string, error) {
	if note == nil || len(note.Embedding) == 0 {
		return nil, nil
	}

	// Search for candidates (+1 because the note itself may be in results).
	candidates, err := lm.store.Search(ctx, note.Embedding, lm.candidates+1)
	if err != nil {
		return nil, err
	}

	var linkedIDs []string
	for _, candidate := range candidates {
		if candidate.ID == note.ID {
			continue
		}
		if len(linkedIDs) >= lm.candidates {
			break
		}

		// Add bidirectional link.
		if !containsString(note.Links, candidate.ID) {
			note.Links = append(note.Links, candidate.ID)
		}
		if !containsString(candidate.Links, note.ID) {
			candidate.Links = append(candidate.Links, note.ID)
			if err := lm.store.Update(ctx, candidate); err != nil {
				return linkedIDs, err
			}
		}

		linkedIDs = append(linkedIDs, candidate.ID)
	}

	// Update the source note with its new links.
	if len(linkedIDs) > 0 {
		if err := lm.store.Update(ctx, note); err != nil {
			return linkedIDs, err
		}
	}

	return linkedIDs, nil
}

// containsString checks if a slice contains a given string.
func containsString(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}
