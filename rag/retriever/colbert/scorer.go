package colbert

import "math"

// MaxSim computes the MaxSim score between query token vectors and document
// token vectors, as defined in the ColBERT paper. For each query token vector,
// the maximum cosine similarity across all document token vectors is computed.
// These per-query-token maximums are then summed to produce the final score.
//
// Returns 0 if either queryVecs or docVecs is empty.
func MaxSim(queryVecs, docVecs [][]float32) float64 {
	if len(queryVecs) == 0 || len(docVecs) == 0 {
		return 0
	}
	var total float64
	for _, qv := range queryVecs {
		var maxSim float64 = -math.MaxFloat64
		for _, dv := range docVecs {
			sim := cosineSimilarity(qv, dv)
			if sim > maxSim {
				maxSim = sim
			}
		}
		total += maxSim
	}
	return total
}

// cosineSimilarity computes the cosine similarity between two vectors.
// Returns 0 if either vector has zero magnitude or the vectors have different
// lengths.
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		ai, bi := float64(a[i]), float64(b[i])
		dot += ai * bi
		normA += ai * ai
		normB += bi * bi
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
