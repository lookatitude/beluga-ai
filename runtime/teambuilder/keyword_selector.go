package teambuilder

import (
	"context"
	"sort"
	"strings"
)

// Compile-time check that KeywordSelector implements Selector.
var _ Selector = (*KeywordSelector)(nil)

// KeywordSelector selects agents by computing word overlap between the task
// description and each candidate's capabilities and persona goal. It is the
// fastest selector (no external calls) and suitable for most use cases.
type KeywordSelector struct {
	// minScore is the minimum overlap score required for an agent to be
	// included. Defaults to 1 (at least one keyword match).
	minScore int
}

// KeywordOption configures a KeywordSelector.
type KeywordOption func(*KeywordSelector)

// WithMinScore sets the minimum keyword overlap score. Agents scoring below
// this threshold are excluded from results.
func WithMinScore(n int) KeywordOption {
	return func(ks *KeywordSelector) {
		if n > 0 {
			ks.minScore = n
		}
	}
}

// NewKeywordSelector creates a KeywordSelector with the given options.
func NewKeywordSelector(opts ...KeywordOption) *KeywordSelector {
	ks := &KeywordSelector{
		minScore: 1,
	}
	for _, opt := range opts {
		opt(ks)
	}
	return ks
}

// scoredEntry pairs a PoolEntry with its keyword overlap score.
type scoredEntry struct {
	entry PoolEntry
	score int
}

// Select returns candidates ranked by keyword overlap with the task.
// It matches task words against each candidate's capabilities and persona goal.
// Candidates scoring below minScore are excluded.
func (ks *KeywordSelector) Select(_ context.Context, task string, candidates []PoolEntry) ([]PoolEntry, error) {
	taskWords := tokenize(task)
	if len(taskWords) == 0 {
		return nil, nil
	}

	var scored []scoredEntry
	for _, c := range candidates {
		score := computeKeywordScore(taskWords, c)
		if score >= ks.minScore {
			scored = append(scored, scoredEntry{entry: c, score: score})
		}
	}

	// Sort descending by score, stable to preserve input order for ties.
	sort.SliceStable(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	result := make([]PoolEntry, len(scored))
	for i, s := range scored {
		result[i] = s.entry
	}
	return result, nil
}

// computeKeywordScore calculates the keyword overlap score for a candidate.
// It checks task words against capabilities (case-insensitive) and the
// agent's persona goal.
func computeKeywordScore(taskWords []string, candidate PoolEntry) int {
	score := 0

	// Build a searchable text from capabilities.
	capText := strings.ToLower(strings.Join(candidate.Capabilities, " "))

	// Also include persona goal for broader matching.
	goalText := strings.ToLower(candidate.Agent.Persona().Goal)

	for _, word := range taskWords {
		if len(word) <= 2 {
			continue // Skip very short words.
		}
		if strings.Contains(capText, word) {
			score += 2 // Capability match scores higher.
		}
		if strings.Contains(goalText, word) {
			score++
		}
	}
	return score
}

// tokenize splits text into lowercase words, filtering out very short tokens.
func tokenize(text string) []string {
	words := strings.Fields(strings.ToLower(text))
	result := make([]string, 0, len(words))
	for _, w := range words {
		// Strip common punctuation.
		w = strings.Trim(w, ".,;:!?\"'()[]{}")
		if len(w) > 2 {
			result = append(result, w)
		}
	}
	return result
}

func init() {
	RegisterSelector("keyword", func(cfg map[string]any) (Selector, error) {
		var opts []KeywordOption
		if minScore, ok := cfg["min_score"]; ok {
			if n, ok := minScore.(int); ok {
				opts = append(opts, WithMinScore(n))
			}
		}
		return NewKeywordSelector(opts...), nil
	})
}
