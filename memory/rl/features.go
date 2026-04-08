package rl

// PolicyFeatures holds the scalar and vector features extracted from the
// current memory state and incoming query. These serve as the observation
// for the RL policy's decision.
type PolicyFeatures struct {
	// StoreSize is the current number of entries in the memory store.
	StoreSize float64

	// MaxSimilarity is the highest cosine similarity between the query and
	// any existing entry. Range [0, 1].
	MaxSimilarity float64

	// MeanSimilarity is the average cosine similarity between the query and
	// the top-k existing entries. Range [0, 1].
	MeanSimilarity float64

	// HasMatchingEntry is true if an existing entry exceeds the similarity
	// threshold and could be updated or considered redundant.
	HasMatchingEntry bool

	// TurnIndex is the conversation turn number (0-indexed).
	TurnIndex int

	// QueryTokenCount is the approximate token count of the incoming content.
	QueryTokenCount int

	// EntryAge is the normalized age of the most similar existing entry.
	// Range [0, 1] where 0 is newest and 1 is oldest.
	EntryAge float64

	// RetrievalFrequency is how many times the most similar entry has been
	// retrieved in recent turns.
	RetrievalFrequency int

	// QueryEmbedding is the projected query embedding, typically 32-64 dims.
	// This is the dense feature used by neural policies. May be nil for
	// heuristic policies.
	QueryEmbedding []float32
}

// ScalarFeatureCount is the number of scalar features in PolicyFeatures.
// This does not include QueryEmbedding.
const ScalarFeatureCount = 8

// ToTensor converts the features into a flat float32 slice suitable for
// model inference. Scalar features come first, followed by QueryEmbedding.
func (f PolicyFeatures) ToTensor() []float32 {
	boolToFloat := func(b bool) float32 {
		if b {
			return 1.0
		}
		return 0.0
	}

	scalars := []float32{
		float32(f.StoreSize),
		float32(f.MaxSimilarity),
		float32(f.MeanSimilarity),
		boolToFloat(f.HasMatchingEntry),
		float32(f.TurnIndex),
		float32(f.QueryTokenCount),
		float32(f.EntryAge),
		float32(f.RetrievalFrequency),
	}

	if len(f.QueryEmbedding) == 0 {
		return scalars
	}

	tensor := make([]float32, 0, len(scalars)+len(f.QueryEmbedding))
	tensor = append(tensor, scalars...)
	tensor = append(tensor, f.QueryEmbedding...)
	return tensor
}

// TensorSize returns the total length of the tensor produced by ToTensor.
func (f PolicyFeatures) TensorSize() int {
	return ScalarFeatureCount + len(f.QueryEmbedding)
}
