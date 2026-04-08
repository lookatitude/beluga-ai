package raptor

import (
	"context"
	"fmt"
	"math"
	"math/rand/v2"
)

// Clusterer groups embedding vectors into clusters. Each returned cluster is a
// slice of indices into the original embeddings slice.
type Clusterer interface {
	// Cluster partitions embeddings into groups and returns the indices of
	// each group. The outer slice has one entry per cluster; the inner slice
	// holds indices into the embeddings parameter.
	Cluster(ctx context.Context, embeddings [][]float32) ([][]int, error)
}

// KMeansClusterer implements K-means clustering in pure Go with no external
// dependencies. It uses K-means++ initialization and Lloyd's algorithm.
type KMeansClusterer struct {
	// K is the number of clusters. If K <= 0 it is estimated from the input
	// size as sqrt(n/2), clamped to [2, n].
	K int
	// MaxIterations is the maximum number of Lloyd iterations. Default: 100.
	MaxIterations int
	// Seed controls the random number generator for reproducible results.
	// Zero means use a non-deterministic source.
	Seed uint64
}

// Compile-time interface check.
var _ Clusterer = (*KMeansClusterer)(nil)

// Cluster partitions embeddings into K clusters using K-means++ initialization
// followed by Lloyd's algorithm. It returns one slice of embedding indices per
// cluster. Clusters with zero members after convergence are omitted.
func (c *KMeansClusterer) Cluster(ctx context.Context, embeddings [][]float32) ([][]int, error) {
	n := len(embeddings)
	if n == 0 {
		return nil, fmt.Errorf("raptor: cluster: no embeddings provided")
	}
	if n == 1 {
		return [][]int{{0}}, nil
	}

	k := c.K
	if k <= 0 {
		k = int(math.Sqrt(float64(n) / 2.0))
	}
	if k < 2 {
		k = 2
	}
	if k > n {
		k = n
	}

	maxIter := c.MaxIterations
	if maxIter <= 0 {
		maxIter = 100
	}

	var rng *rand.Rand
	if c.Seed != 0 {
		rng = rand.New(rand.NewPCG(c.Seed, 0))
	} else {
		rng = rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
	}

	dim := len(embeddings[0])

	// K-means++ initialization.
	centroids := make([][]float32, k)
	centroids[0] = copyVec(embeddings[rng.IntN(n)])

	dist := make([]float64, n)
	for i := 1; i < k; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		var totalDist float64
		for j := 0; j < n; j++ {
			d := euclideanDistSq(embeddings[j], centroids[i-1])
			if i == 1 || d < dist[j] {
				dist[j] = d
			}
			totalDist += dist[j]
		}

		if totalDist == 0 {
			// All points are identical; duplicate centroids.
			centroids[i] = copyVec(centroids[0])
			continue
		}

		r := rng.Float64() * totalDist
		var cumulative float64
		chosen := n - 1
		for j := 0; j < n; j++ {
			cumulative += dist[j]
			if cumulative >= r {
				chosen = j
				break
			}
		}
		centroids[i] = copyVec(embeddings[chosen])
	}

	// Lloyd's iterations.
	assignments := make([]int, n)
	for iter := 0; iter < maxIter; iter++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		changed := false
		for j := 0; j < n; j++ {
			best := 0
			bestDist := euclideanDistSq(embeddings[j], centroids[0])
			for ci := 1; ci < k; ci++ {
				d := euclideanDistSq(embeddings[j], centroids[ci])
				if d < bestDist {
					bestDist = d
					best = ci
				}
			}
			if assignments[j] != best {
				assignments[j] = best
				changed = true
			}
		}

		if !changed {
			break
		}

		// Recompute centroids.
		newCentroids := make([][]float32, k)
		counts := make([]int, k)
		for ci := 0; ci < k; ci++ {
			newCentroids[ci] = make([]float32, dim)
		}
		for j := 0; j < n; j++ {
			ci := assignments[j]
			counts[ci]++
			for d := 0; d < dim; d++ {
				newCentroids[ci][d] += embeddings[j][d]
			}
		}
		for ci := 0; ci < k; ci++ {
			if counts[ci] > 0 {
				for d := 0; d < dim; d++ {
					newCentroids[ci][d] /= float32(counts[ci])
				}
			} else {
				// Keep old centroid for empty clusters.
				newCentroids[ci] = centroids[ci]
			}
		}
		centroids = newCentroids
	}

	// Build result, omitting empty clusters.
	clusters := make(map[int][]int, k)
	for j := 0; j < n; j++ {
		ci := assignments[j]
		clusters[ci] = append(clusters[ci], j)
	}

	result := make([][]int, 0, len(clusters))
	for _, indices := range clusters {
		result = append(result, indices)
	}

	return result, nil
}

// euclideanDistSq returns the squared Euclidean distance between two vectors.
func euclideanDistSq(a, b []float32) float64 {
	var sum float64
	for i := range a {
		d := float64(a[i]) - float64(b[i])
		sum += d * d
	}
	return sum
}

// copyVec returns a copy of the float32 slice.
func copyVec(v []float32) []float32 {
	c := make([]float32, len(v))
	copy(c, v)
	return c
}
