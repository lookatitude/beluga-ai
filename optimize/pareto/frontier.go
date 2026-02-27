// Package pareto provides Pareto frontier data structures and operations
// for multi-objective optimization.
package pareto

import (
	"sort"
)

// Point represents a point in multi-objective space.
type Point struct {
	ID       string
	Objectives []float64 // One value per objective (higher is better)
	Payload  interface{} // Associated data (e.g., prompt candidate)
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

// distance calculates Euclidean distance between two points.
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
