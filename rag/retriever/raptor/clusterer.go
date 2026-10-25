package raptor

import (
	"context"
	"math"
	"math/rand/v2" //#nosec G404 -- non-crypto randomness for K-means++ initialization; seed is a reproducibility feature
	"sort"

	"github.com/lookatitude/beluga-ai/v2/core"
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
		return nil, core.Errorf(core.ErrInvalidInput, "raptor: cluster: no embeddings provided")
	}
	if n == 1 {
		return [][]int{{0}}, nil
	}

	k := c.resolveK(n)

	maxIter := c.MaxIterations
	if maxIter <= 0 {
		maxIter = 100
	}

	rng := c.newRNG()

	dim, err := commonDimension(embeddings)
	if err != nil {
		return nil, err
	}

	// K-means++ initialization.
	centroids, err := kmeansPlusPlusInit(ctx, embeddings, k, rng)
	if err != nil {
		return nil, err
	}

	// Lloyd's iterations.
	assignments := make([]int, n)
	for iter := 0; iter < maxIter; iter++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if !assignNearest(embeddings, centroids, assignments) {
			break
		}
		centroids = recomputeCentroids(embeddings, assignments, centroids, dim)
	}

	return groupByCluster(assignments, k), nil
}

// resolveK returns the configured K, or sqrt(n/2) when K <= 0, clamped to [2, n].
func (c *KMeansClusterer) resolveK(n int) int {
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
	return k
}

// newRNG returns a PRNG seeded from Seed, or from a random seed when Seed is zero.
func (c *KMeansClusterer) newRNG() *rand.Rand {
	if c.Seed != 0 {
		return rand.New(rand.NewPCG(c.Seed, 0)) //#nosec G404 -- non-crypto PRNG for K-means++ initialization
	}
	return rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64())) //#nosec G404 -- non-crypto PRNG for K-means++ initialization
}

// commonDimension returns the dimension shared by all embeddings, or an error
// naming the first embedding whose dimension differs from embeddings[0].
func commonDimension(embeddings [][]float32) (int, error) {
	dim := len(embeddings[0])
	for i, e := range embeddings[1:] {
		if len(e) != dim {
			return 0, core.Errorf(core.ErrInvalidInput, "raptor: cluster: embedding %d has dimension %d, want %d", i+1, len(e), dim)
		}
	}
	return dim, nil
}

// kmeansPlusPlusInit picks k initial centroids using K-means++: the first is
// uniform at random, each next one is sampled proportionally to its squared
// distance from the nearest centroid chosen so far.
func kmeansPlusPlusInit(ctx context.Context, embeddings [][]float32, k int, rng *rand.Rand) ([][]float32, error) {
	n := len(embeddings)
	centroids := make([][]float32, k)
	centroids[0] = copyVec(embeddings[rng.IntN(n)]) //#nosec G404 -- non-crypto PRNG

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

		r := rng.Float64() * totalDist //#nosec G404 -- non-crypto PRNG
		centroids[i] = copyVec(embeddings[sampleIndex(dist, r)])
	}
	return centroids, nil
}

// sampleIndex returns the first index whose cumulative weight reaches r, or
// the last index if none does.
func sampleIndex(weights []float64, r float64) int {
	var cumulative float64
	for j, w := range weights {
		cumulative += w
		if cumulative >= r {
			return j
		}
	}
	return len(weights) - 1
}

// assignNearest assigns each embedding to its nearest centroid (ties go to the
// lowest index), updating assignments in place. It reports whether any
// assignment changed.
func assignNearest(embeddings, centroids [][]float32, assignments []int) bool {
	changed := false
	for j, e := range embeddings {
		best := 0
		bestDist := euclideanDistSq(e, centroids[0])
		for ci := 1; ci < len(centroids); ci++ {
			d := euclideanDistSq(e, centroids[ci])
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
	return changed
}

// recomputeCentroids returns the mean of each cluster's members. Clusters with
// no members keep their previous centroid.
func recomputeCentroids(embeddings [][]float32, assignments []int, centroids [][]float32, dim int) [][]float32 {
	k := len(centroids)
	newCentroids := make([][]float32, k)
	counts := make([]int, k)
	for ci := 0; ci < k; ci++ {
		newCentroids[ci] = make([]float32, dim)
	}
	for j, e := range embeddings {
		ci := assignments[j]
		counts[ci]++
		for d := 0; d < dim; d++ {
			newCentroids[ci][d] += e[d]
		}
	}
	for ci := 0; ci < k; ci++ {
		if counts[ci] == 0 {
			// Keep old centroid for empty clusters.
			newCentroids[ci] = centroids[ci]
			continue
		}
		for d := 0; d < dim; d++ {
			newCentroids[ci][d] /= float32(counts[ci])
		}
	}
	return newCentroids
}

// groupByCluster converts per-embedding assignments into one index slice per
// non-empty cluster, ordered by cluster ID.
func groupByCluster(assignments []int, k int) [][]int {
	clusters := make(map[int][]int, k)
	for j, ci := range assignments {
		clusters[ci] = append(clusters[ci], j)
	}

	keys := make([]int, 0, len(clusters))
	for ci := range clusters {
		keys = append(keys, ci)
	}
	sort.Ints(keys)
	result := make([][]int, 0, len(clusters))
	for _, ci := range keys {
		result = append(result, clusters[ci])
	}
	return result
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
