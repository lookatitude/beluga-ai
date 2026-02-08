package retriever

import (
	"sort"

	"github.com/lookatitude/beluga-ai/schema"
)

// sortByScore sorts documents by Score in descending order.
func sortByScore(docs []schema.Document) {
	sort.Slice(docs, func(i, j int) bool {
		return docs[i].Score > docs[j].Score
	})
}

// dedup removes duplicate documents by ID, keeping the one with the highest score.
func dedup(docs []schema.Document) []schema.Document {
	seen := make(map[string]int, len(docs)) // id -> index in result
	result := make([]schema.Document, 0, len(docs))
	for _, doc := range docs {
		if idx, ok := seen[doc.ID]; ok {
			if doc.Score > result[idx].Score {
				result[idx] = doc
			}
		} else {
			seen[doc.ID] = len(result)
			result = append(result, doc)
		}
	}
	return result
}
