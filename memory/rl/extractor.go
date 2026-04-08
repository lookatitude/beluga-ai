package rl

import (
	"context"
	"math"
	"strings"

	"github.com/lookatitude/beluga-ai/memory"
	"github.com/lookatitude/beluga-ai/schema"
)

// DefaultFeatureExtractor computes PolicyFeatures by querying the underlying
// memory for similar documents and deriving scalar statistics.
type DefaultFeatureExtractor struct {
	// TopK is the number of similar entries to retrieve for computing
	// similarity statistics. Default: 5.
	TopK int

	// SimilarityThreshold is the cosine similarity above which an entry
	// is considered a match. Default: 0.7.
	SimilarityThreshold float64
}

// Extract implements FeatureExtractor by searching the memory for similar
// entries and computing statistics from the results.
func (e *DefaultFeatureExtractor) Extract(ctx context.Context, mem memory.Memory, _, output schema.Message) (PolicyFeatures, error) {
	topK := e.TopK
	if topK <= 0 {
		topK = 5
	}
	threshold := e.SimilarityThreshold
	if threshold <= 0 {
		threshold = 0.7
	}

	// Build query text from output message content.
	query := messageText(output)

	// Search for similar entries.
	docs, err := mem.Search(ctx, query, topK)
	if err != nil {
		// If search is not supported, return minimal features.
		return PolicyFeatures{
			QueryTokenCount: approximateTokenCount(query),
		}, nil
	}

	features := PolicyFeatures{
		StoreSize:       float64(len(docs)),
		QueryTokenCount: approximateTokenCount(query),
	}

	if len(docs) > 0 {
		features.MaxSimilarity = docs[0].Score
		features.MeanSimilarity = meanScore(docs)
		features.HasMatchingEntry = docs[0].Score >= threshold

		// Extract age and retrieval frequency from metadata if available.
		if age, ok := docs[0].Metadata["entry_age"].(float64); ok {
			features.EntryAge = age
		}
		if freq, ok := docs[0].Metadata["retrieval_frequency"].(int); ok {
			features.RetrievalFrequency = freq
		}
		if turn, ok := docs[0].Metadata["turn_index"].(int); ok {
			features.TurnIndex = turn
		}
	}

	return features, nil
}

// messageText extracts the plain text from a message's content parts.
func messageText(msg schema.Message) string {
	parts := msg.GetContent()
	if len(parts) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, part := range parts {
		if tp, ok := part.(schema.TextPart); ok {
			if sb.Len() > 0 {
				sb.WriteByte(' ')
			}
			sb.WriteString(tp.Text)
		}
	}
	return sb.String()
}

// approximateTokenCount provides a rough word-based token estimate.
func approximateTokenCount(s string) int {
	if s == "" {
		return 0
	}
	// Rough heuristic: ~0.75 words per token for English text.
	words := len(strings.Fields(s))
	tokens := int(math.Ceil(float64(words) * 1.33))
	if tokens == 0 && len(s) > 0 {
		tokens = 1
	}
	return tokens
}

// meanScore computes the average score across documents.
func meanScore(docs []schema.Document) float64 {
	if len(docs) == 0 {
		return 0
	}
	var sum float64
	for _, d := range docs {
		sum += d.Score
	}
	return sum / float64(len(docs))
}

// Compile-time interface check.
var _ FeatureExtractor = (*DefaultFeatureExtractor)(nil)
