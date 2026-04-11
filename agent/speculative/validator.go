package speculative

import (
	"context"
	"math"
	"strings"
)

// Validator checks whether a prediction matches the ground truth.
type Validator interface {
	// Validate returns true if the prediction is considered equivalent to
	// the ground truth.
	Validate(ctx context.Context, prediction, groundTruth string) (valid bool, err error)
}

// ExactValidator checks for exact string equality after trimming whitespace
// and normalizing case.
type ExactValidator struct{}

// compile-time check
var _ Validator = (*ExactValidator)(nil)

// NewExactValidator creates a Validator that requires exact string match.
func NewExactValidator() *ExactValidator {
	return &ExactValidator{}
}

// Validate returns true if prediction equals ground truth after normalization.
func (v *ExactValidator) Validate(_ context.Context, prediction, groundTruth string) (bool, error) {
	p := strings.TrimSpace(strings.ToLower(prediction))
	g := strings.TrimSpace(strings.ToLower(groundTruth))
	return p == g, nil
}

// SemanticValidator checks similarity using cosine similarity of term frequency
// vectors. If the similarity exceeds the configured threshold, the prediction
// is considered valid.
type SemanticValidator struct {
	threshold float64
}

// compile-time check
var _ Validator = (*SemanticValidator)(nil)

// NewSemanticValidator creates a Validator that checks cosine similarity
// of term-frequency vectors against the given threshold (0.0-1.0).
func NewSemanticValidator(threshold float64) *SemanticValidator {
	if threshold < 0 {
		threshold = 0
	}
	if threshold > 1 {
		threshold = 1
	}
	return &SemanticValidator{threshold: threshold}
}

// Validate returns true if the cosine similarity between prediction and
// ground truth term-frequency vectors exceeds the configured threshold.
func (v *SemanticValidator) Validate(_ context.Context, prediction, groundTruth string) (bool, error) {
	sim := cosineSimilarity(prediction, groundTruth)
	return sim >= v.threshold, nil
}

// tokenize splits text into lowercase word tokens.
func tokenize(text string) []string {
	return strings.Fields(strings.ToLower(strings.TrimSpace(text)))
}

// termFrequency builds a term frequency map from tokens.
func termFrequency(tokens []string) map[string]float64 {
	tf := make(map[string]float64, len(tokens))
	for _, t := range tokens {
		tf[t]++
	}
	return tf
}

// cosineSimilarity computes the cosine similarity between two text strings
// using their term frequency vectors.
func cosineSimilarity(a, b string) float64 {
	tokA := tokenize(a)
	tokB := tokenize(b)

	if len(tokA) == 0 || len(tokB) == 0 {
		if len(tokA) == 0 && len(tokB) == 0 {
			return 1.0 // both empty = identical
		}
		return 0.0
	}

	tfA := termFrequency(tokA)
	tfB := termFrequency(tokB)

	// Compute dot product and magnitudes.
	var dot, magA, magB float64
	for term, freqA := range tfA {
		magA += freqA * freqA
		if freqB, ok := tfB[term]; ok {
			dot += freqA * freqB
		}
	}
	for _, freqB := range tfB {
		magB += freqB * freqB
	}

	denom := math.Sqrt(magA) * math.Sqrt(magB)
	if denom == 0 {
		return 0.0
	}

	return dot / denom
}
