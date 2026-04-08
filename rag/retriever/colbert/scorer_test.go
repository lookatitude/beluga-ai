package colbert

import (
	"math"
	"testing"
)

func TestMaxSim(t *testing.T) {
	tests := []struct {
		name      string
		queryVecs [][]float32
		docVecs   [][]float32
		want      float64
		tolerance float64
	}{
		{
			name:      "empty query vectors",
			queryVecs: nil,
			docVecs:   [][]float32{{1, 0, 0}},
			want:      0,
		},
		{
			name:      "empty doc vectors",
			queryVecs: [][]float32{{1, 0, 0}},
			docVecs:   nil,
			want:      0,
		},
		{
			name:      "both empty",
			queryVecs: nil,
			docVecs:   nil,
			want:      0,
		},
		{
			name:      "identical single token",
			queryVecs: [][]float32{{1, 0, 0}},
			docVecs:   [][]float32{{1, 0, 0}},
			want:      1.0,
			tolerance: 1e-9,
		},
		{
			name:      "orthogonal single token",
			queryVecs: [][]float32{{1, 0, 0}},
			docVecs:   [][]float32{{0, 1, 0}},
			want:      0.0,
			tolerance: 1e-9,
		},
		{
			name:      "opposite single token",
			queryVecs: [][]float32{{1, 0, 0}},
			docVecs:   [][]float32{{-1, 0, 0}},
			want:      -1.0,
			tolerance: 1e-9,
		},
		{
			name: "two query tokens pick best doc token each",
			queryVecs: [][]float32{
				{1, 0, 0}, // best match: doc token {1, 0, 0} -> sim=1.0
				{0, 1, 0}, // best match: doc token {0, 1, 0} -> sim=1.0
			},
			docVecs: [][]float32{
				{1, 0, 0},
				{0, 1, 0},
				{0, 0, 1},
			},
			want:      2.0, // 1.0 + 1.0
			tolerance: 1e-9,
		},
		{
			name: "query token matches partial doc token",
			queryVecs: [][]float32{
				{1, 1, 0}, // should match {1, 0, 0} or {0, 1, 0} partially
			},
			docVecs: [][]float32{
				{1, 0, 0},
				{0, 1, 0},
			},
			// cos({1,1,0}, {1,0,0}) = 1/sqrt(2) ~ 0.7071
			// cos({1,1,0}, {0,1,0}) = 1/sqrt(2) ~ 0.7071
			// max = 0.7071
			want:      1.0 / math.Sqrt(2),
			tolerance: 1e-6,
		},
		{
			name: "scaled vectors same direction",
			queryVecs: [][]float32{
				{2, 0, 0},
			},
			docVecs: [][]float32{
				{5, 0, 0},
			},
			want:      1.0, // cosine similarity is scale-invariant
			tolerance: 1e-9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MaxSim(tt.queryVecs, tt.docVecs)
			if tt.tolerance == 0 {
				if got != tt.want {
					t.Errorf("MaxSim() = %v, want %v", got, tt.want)
				}
			} else {
				if math.Abs(got-tt.want) > tt.tolerance {
					t.Errorf("MaxSim() = %v, want %v (tolerance %v)", got, tt.want, tt.tolerance)
				}
			}
		})
	}
}

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name      string
		a, b      []float32
		want      float64
		tolerance float64
	}{
		{
			name: "different lengths",
			a:    []float32{1, 0},
			b:    []float32{1, 0, 0},
			want: 0,
		},
		{
			name: "empty vectors",
			a:    nil,
			b:    nil,
			want: 0,
		},
		{
			name: "zero vector a",
			a:    []float32{0, 0, 0},
			b:    []float32{1, 0, 0},
			want: 0,
		},
		{
			name:      "unit vectors identical",
			a:         []float32{0, 1, 0},
			b:         []float32{0, 1, 0},
			want:      1.0,
			tolerance: 1e-9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cosineSimilarity(tt.a, tt.b)
			if tt.tolerance == 0 {
				if got != tt.want {
					t.Errorf("cosineSimilarity() = %v, want %v", got, tt.want)
				}
			} else {
				if math.Abs(got-tt.want) > tt.tolerance {
					t.Errorf("cosineSimilarity() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func BenchmarkMaxSim(b *testing.B) {
	// Simulate 32 query tokens x 128 doc tokens x 128 dimensions.
	queryVecs := makeVecs(32, 128)
	docVecs := makeVecs(128, 128)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		MaxSim(queryVecs, docVecs)
	}
}

func makeVecs(n, dim int) [][]float32 {
	vecs := make([][]float32, n)
	for i := range vecs {
		v := make([]float32, dim)
		for j := range v {
			v[j] = float32(i+j+1) * 0.01
		}
		vecs[i] = v
	}
	return vecs
}
