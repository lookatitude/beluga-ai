// Package pareto provides Pareto frontier data structures and multi-objective
// optimization utilities for the GEPA optimizer and other evolutionary strategies.
//
// Core concepts:
//   - Dominance: point A dominates point B when A is at least as good as B on all
//     objectives and strictly better on at least one.
//   - Frontier (Pareto front): the set of all mutually non-dominated points.
//   - Archive: a size-bounded, generation-aware store that maintains the frontier
//     across multiple rounds of an evolutionary algorithm.
//
// The package supports an arbitrary number of objectives. All objectives are
// maximisation objectives; callers should negate minimisation objectives before
// inserting points.
package pareto

import (
	"math"
	"sort"
)

// Point represents a point in multi-objective space.
type Point struct {
	ID         string
	Objectives []float64   // One value per objective (higher is better)
	Payload    interface{} // Associated data (e.g., prompt candidate)
}

// Frontier represents a Pareto frontier — a set of non-dominated points.
type Frontier struct {
	points []Point
}

// NewFrontier creates an empty Pareto frontier.
func NewFrontier() *Frontier {
	return &Frontier{
		points: make([]Point, 0),
	}
}

// Add adds a point to the frontier, removing any points it dominates.
// Returns true if the point was added (i.e., it's non-dominated).
func (f *Frontier) Add(p Point) bool {
	// Check if p is dominated by any existing point
	for _, existing := range f.points {
		if dominates(existing, p) {
			// p is dominated, don't add
			return false
		}
	}

	// Remove points that are dominated by p
	newPoints := make([]Point, 0, len(f.points)+1)
	for _, existing := range f.points {
		if !dominates(p, existing) {
			newPoints = append(newPoints, existing)
		}
	}
	newPoints = append(newPoints, p)
	f.points = newPoints

	return true
}

// Get returns all points on the frontier.
func (f *Frontier) Get() []Point {
	result := make([]Point, len(f.points))
	copy(result, f.points)
	return result
}

// Len returns the number of points on the frontier.
func (f *Frontier) Len() int {
	return len(f.points)
}

// dominates returns true if a dominates b (a is better in all objectives).
// Assumes higher values are better.
func dominates(a, b Point) bool {
	if len(a.Objectives) != len(b.Objectives) {
		return false
	}

	// a dominates b if a is >= b in all objectives and > in at least one
	strictlyBetter := false
	for i := range a.Objectives {
		if a.Objectives[i] < b.Objectives[i] {
			return false
		}
		if a.Objectives[i] > b.Objectives[i] {
			strictlyBetter = true
		}
	}
	return strictlyBetter
}

// Coverage calculates how much of the objective space is covered by the frontier.
// Returns a value between 0 and 1, where 1 means complete coverage.
func (f *Frontier) Coverage(reference *Frontier) float64 {
	if len(f.points) == 0 || len(reference.points) == 0 {
		return 0.0
	}

	// Calculate hypervolume indicator (simplified version)
	// For each reference point, count how many frontier points dominate it
	covered := 0
	for _, ref := range reference.points {
		for _, p := range f.points {
			if dominates(p, ref) {
				covered++
				break
			}
		}
	}

	return float64(covered) / float64(len(reference.points))
}

// Spacing calculates the spacing metric (uniformity of points).
// Lower values indicate more uniform distribution.
func (f *Frontier) Spacing() float64 {
	if len(f.points) < 2 {
		return 0.0
	}

	// Calculate distances between consecutive points
	distances := make([]float64, 0, len(f.points)-1)
	for i := 0; i < len(f.points)-1; i++ {
		d := distance(f.points[i], f.points[i+1])
		distances = append(distances, d)
	}

	// Calculate mean distance
	mean := 0.0
	for _, d := range distances {
		mean += d
	}
	mean /= float64(len(distances))

	// Calculate standard deviation
	variance := 0.0
	for _, d := range distances {
		diff := d - mean
		variance += diff * diff
	}
	variance /= float64(len(distances))

	return variance
}

// distance calculates squared Euclidean distance between two points.
// Squared distance is used internally to avoid unnecessary sqrt calls when only
// relative ordering matters.
func distance(a, b Point) float64 {
	if len(a.Objectives) != len(b.Objectives) {
		return 0.0
	}

	sum := 0.0
	for i := range a.Objectives {
		diff := a.Objectives[i] - b.Objectives[i]
		sum += diff * diff
	}
	return sum // No sqrt needed for comparison
}

// EuclideanDistance returns the true Euclidean distance between two points.
func EuclideanDistance(a, b Point) float64 {
	return math.Sqrt(distance(a, b))
}

// CrowdingDistance computes the NSGA-II crowding distance for each point in the
// frontier. Points with a higher crowding distance are in less crowded regions
// of the objective space, which is useful for maintaining diversity.
//
// The returned slice has the same length as f.Get(). Boundary points receive
// a distance of +Inf.
func (f *Frontier) CrowdingDistance() []float64 {
	pts := f.Get()
	n := len(pts)
	if n == 0 {
		return nil
	}

	dist := make([]float64, n)
	if n <= 2 {
		for i := range dist {
			dist[i] = math.Inf(1)
		}
		return dist
	}

	if len(pts[0].Objectives) == 0 {
		return dist
	}
	numObj := len(pts[0].Objectives)

	for obj := 0; obj < numObj; obj++ {
		// Sort indices by this objective (ascending).
		indices := make([]int, n)
		for i := range indices {
			indices[i] = i
		}
		sort.Slice(indices, func(a, b int) bool {
			return pts[indices[a]].Objectives[obj] < pts[indices[b]].Objectives[obj]
		})

		// Boundary points get infinite distance.
		dist[indices[0]] = math.Inf(1)
		dist[indices[n-1]] = math.Inf(1)

		objRange := pts[indices[n-1]].Objectives[obj] - pts[indices[0]].Objectives[obj]
		if objRange == 0 {
			continue
		}

		for i := 1; i < n-1; i++ {
			dist[indices[i]] += (pts[indices[i+1]].Objectives[obj] - pts[indices[i-1]].Objectives[obj]) / objRange
		}
	}

	return dist
}

// HypervolumeIndicator estimates the hypervolume dominated by the Pareto frontier
// relative to a reference point (nadir). All objectives in the reference point
// should be worse than any point on the frontier (i.e. the worst possible values).
//
// This implementation uses a sweep-line algorithm accurate for 2-objective problems
// and an approximate decomposition for higher dimensions.
func (f *Frontier) HypervolumeIndicator(reference []float64) float64 {
	pts := f.Get()
	if len(pts) == 0 || len(reference) == 0 {
		return 0.0
	}

	numObj := len(reference)
	if numObj != len(pts[0].Objectives) {
		return 0.0
	}

	if numObj == 1 {
		// 1-D: just the maximum span.
		maxVal := pts[0].Objectives[0]
		for _, p := range pts[1:] {
			if p.Objectives[0] > maxVal {
				maxVal = p.Objectives[0]
			}
		}
		hv := maxVal - reference[0]
		if hv < 0 {
			return 0
		}
		return hv
	}

	if numObj == 2 {
		return hypervolume2D(pts, reference)
	}

	// ≥3 objectives: Monte Carlo approximation.
	return hypervolumeApprox(pts, reference, 10000)
}

// hypervolume2D computes exact hypervolume for 2-objective problems via sweep line.
func hypervolume2D(pts []Point, ref []float64) float64 {
	// Sort by objective 0 descending.
	sorted := make([]Point, len(pts))
	copy(sorted, pts)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Objectives[0] > sorted[j].Objectives[0]
	})

	hv := 0.0
	prevX := ref[0]
	for _, p := range sorted {
		width := p.Objectives[0] - prevX
		if width <= 0 {
			continue
		}
		height := p.Objectives[1] - ref[1]
		if height > 0 {
			hv += width * height
		}
		prevX = p.Objectives[0]
	}
	return hv
}

// hypervolumeApprox uses Monte Carlo sampling to approximate the hypervolume.
func hypervolumeApprox(pts []Point, ref []float64, samples int) float64 {
	numObj := len(ref)

	// Determine the bounding box.
	upper := make([]float64, numObj)
	for i := range upper {
		upper[i] = ref[i]
		for _, p := range pts {
			if p.Objectives[i] > upper[i] {
				upper[i] = p.Objectives[i]
			}
		}
	}

	// Volume of bounding box.
	boxVol := 1.0
	for i := range upper {
		side := upper[i] - ref[i]
		if side <= 0 {
			return 0
		}
		boxVol *= side
	}

	// Use a deterministic sequence (Van der Corput) for low-discrepancy sampling.
	dominated := 0
	for s := 1; s <= samples; s++ {
		point := make([]float64, numObj)
		for d := 0; d < numObj; d++ {
			point[d] = ref[d] + vanDerCorput(s, d+2)*(upper[d]-ref[d])
		}
		for _, p := range pts {
			isDom := true
			for i := range point {
				if p.Objectives[i] < point[i] {
					isDom = false
					break
				}
			}
			if isDom {
				dominated++
				break
			}
		}
	}

	return boxVol * float64(dominated) / float64(samples)
}

// vanDerCorput generates the s-th term of the Van der Corput sequence in base b.
func vanDerCorput(s, b int) float64 {
	q := 0.0
	bk := 1.0 / float64(b)
	for s > 0 {
		q += float64(s%b) * bk
		s /= b
		bk /= float64(b)
	}
	return q
}

// RankNonDominated partitions points into non-domination ranks (NSGA-II style).
// Rank 0 is the Pareto-optimal front; rank 1 is optimal after removing rank 0, etc.
// Returns a slice where result[i] is the rank of pts[i].
func RankNonDominated(pts []Point) []int {
	n := len(pts)
	ranks := make([]int, n)
	dominated := make([]int, n)   // number of points that dominate i
	dominates := make([][]int, n) // indices of points that i dominates

	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if i == j {
				continue
			}
			if dominates_(pts[i], pts[j]) {
				dominates[i] = append(dominates[i], j)
			} else if dominates_(pts[j], pts[i]) {
				dominated[i]++
			}
		}
	}

	// BFS over fronts.
	current := make([]int, 0)
	for i := 0; i < n; i++ {
		if dominated[i] == 0 {
			current = append(current, i)
		}
	}

	rank := 0
	for len(current) > 0 {
		next := make([]int, 0)
		for _, i := range current {
			ranks[i] = rank
			for _, j := range dominates[i] {
				dominated[j]--
				if dominated[j] == 0 {
					next = append(next, j)
				}
			}
		}
		current = next
		rank++
	}

	return ranks
}

// dominates_ is an alias for the package-level dominates function for use in
// exported helpers that need to reference it without shadowing.
func dominates_(a, b Point) bool {
	return dominates(a, b)
}

// SelectByCoverage selects points from the frontier proportional to their coverage.
// Returns indices of selected points.
func (f *Frontier) SelectByCoverage(n int) []int {
	if n >= len(f.points) {
		indices := make([]int, len(f.points))
		for i := 0; i < len(f.points); i++ {
			indices[i] = i
		}
		return indices
	}

	// Calculate coverage contribution for each point
	contributions := make([]float64, len(f.points))
	for i, p := range f.points {
		// Contribution is inverse of how many other points dominate similar area
		contrib := 1.0
		for j, other := range f.points {
			if i != j {
				// If other is close to p, reduce contribution
				d := distance(p, other)
				if d < 0.01 {
					contrib *= 0.5
				}
			}
		}
		contributions[i] = contrib
	}

	// Sort by contribution descending
	type indexedContrib struct {
		index int
		value float64
	}
	indexed := make([]indexedContrib, len(contributions))
	for i, c := range contributions {
		indexed[i] = indexedContrib{i, c}
	}
	sort.Slice(indexed, func(i, j int) bool {
		return indexed[i].value > indexed[j].value
	})

	// Return top n
	result := make([]int, n)
	for i := 0; i < n; i++ {
		result[i] = indexed[i].index
	}
	return result
}

// Archive maintains an archive of candidates for evolutionary algorithms.
type Archive struct {
	frontier    *Frontier
	maxSize     int
	generations map[string]int // Track generation for each point
}

// NewArchive creates a new archive with the given maximum size.
func NewArchive(maxSize int) *Archive {
	return &Archive{
		frontier:    NewFrontier(),
		maxSize:     maxSize,
		generations: make(map[string]int),
	}
}

// Add adds a point to the archive.
func (a *Archive) Add(p Point, generation int) bool {
	if a.frontier.Add(p) {
		a.generations[p.ID] = generation
		a.maintainSize()
		return true
	}
	return false
}

// maintainSize keeps the archive size within limits by removing oldest points.
func (a *Archive) maintainSize() {
	if a.maxSize <= 0 || a.frontier.Len() <= a.maxSize {
		return
	}

	// Sort by generation (oldest first)
	points := a.frontier.Get()
	sort.Slice(points, func(i, j int) bool {
		return a.generations[points[i].ID] < a.generations[points[j].ID]
	})

	// Keep only the newest maxSize points
	newFrontier := NewFrontier()
	for i := len(points) - a.maxSize; i < len(points); i++ {
		newFrontier.Add(points[i])
	}
	a.frontier = newFrontier
}

// Get returns all points in the archive.
func (a *Archive) Get() []Point {
	return a.frontier.Get()
}

// Len returns the number of points in the archive.
func (a *Archive) Len() int {
	return a.frontier.Len()
}
