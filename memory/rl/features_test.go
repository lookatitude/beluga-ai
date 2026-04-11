package rl

import "testing"

func TestPolicyFeatures_ToTensor(t *testing.T) {
	tests := []struct {
		name     string
		features PolicyFeatures
		wantLen  int
	}{
		{
			name: "scalars only",
			features: PolicyFeatures{
				StoreSize:          10,
				MaxSimilarity:      0.8,
				MeanSimilarity:     0.5,
				HasMatchingEntry:   true,
				TurnIndex:          3,
				QueryTokenCount:    20,
				EntryAge:           0.4,
				RetrievalFrequency: 2,
			},
			wantLen: ScalarFeatureCount,
		},
		{
			name: "with embedding",
			features: PolicyFeatures{
				StoreSize:      5,
				QueryEmbedding: []float32{0.1, 0.2, 0.3, 0.4},
			},
			wantLen: ScalarFeatureCount + 4,
		},
		{
			name:     "zero values",
			features: PolicyFeatures{},
			wantLen:  ScalarFeatureCount,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tensor := tt.features.ToTensor()
			if len(tensor) != tt.wantLen {
				t.Errorf("ToTensor() len = %d, want %d", len(tensor), tt.wantLen)
			}
		})
	}
}

func TestPolicyFeatures_ToTensor_Values(t *testing.T) {
	f := PolicyFeatures{
		StoreSize:          10,
		MaxSimilarity:      0.8,
		MeanSimilarity:     0.5,
		HasMatchingEntry:   true,
		TurnIndex:          3,
		QueryTokenCount:    20,
		EntryAge:           0.4,
		RetrievalFrequency: 2,
	}

	tensor := f.ToTensor()

	// Check scalar values.
	expected := []float32{10, 0.8, 0.5, 1.0, 3, 20, 0.4, 2}
	for i, want := range expected {
		if tensor[i] != want {
			t.Errorf("tensor[%d] = %v, want %v", i, tensor[i], want)
		}
	}
}

func TestPolicyFeatures_ToTensor_HasMatchingEntry_False(t *testing.T) {
	f := PolicyFeatures{HasMatchingEntry: false}
	tensor := f.ToTensor()
	if tensor[3] != 0.0 {
		t.Errorf("HasMatchingEntry=false should produce 0.0, got %v", tensor[3])
	}
}

func TestPolicyFeatures_TensorSize(t *testing.T) {
	f := PolicyFeatures{QueryEmbedding: make([]float32, 32)}
	if got := f.TensorSize(); got != ScalarFeatureCount+32 {
		t.Errorf("TensorSize() = %d, want %d", got, ScalarFeatureCount+32)
	}

	f2 := PolicyFeatures{}
	if got := f2.TensorSize(); got != ScalarFeatureCount {
		t.Errorf("TensorSize() = %d, want %d", got, ScalarFeatureCount)
	}
}
