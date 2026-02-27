package pareto

import (
	"testing"
)

func TestNewFrontier(t *testing.T) {
	f := NewFrontier()
	if f.Len() != 0 {
		t.Errorf("expected empty frontier, got %d points", f.Len())
	}
}

func TestDominates(t *testing.T) {
	tests := []struct {
		name     string
		a        Point
		b        Point
		expected bool
	}{
		{
			name:     "a dominates b",
			a:        Point{Objectives: []float64{5.0, 5.0}},
			b:        Point{Objectives: []float64{3.0, 3.0}},
			expected: true,
		},
		{
			name:     "b dominates a",
			a:        Point{Objectives: []float64{3.0, 3.0}},
			b:        Point{Objectives: []float64{5.0, 5.0}},
			expected: false,
		},
		{
			name:     "equal points",
			a:        Point{Objectives: []float64{3.0, 3.0}},
			b:        Point{Objectives: []float64{3.0, 3.0}},
			expected: false,
		},
		{
			name:     "a better in one, worse in other",
			a:        Point{Objectives: []float64{5.0, 3.0}},
			b:        Point{Objectives: []float64{3.0, 5.0}},
			expected: false,
		},
		{
			name:     "a better in all",
			a:        Point{Objectives: []float64{5.0, 5.0}},
			b:        Point{Objectives: []float64{4.0, 4.0}},
			expected: true,
		},
		{
			name:     "different lengths",
			a:        Point{Objectives: []float64{5.0, 5.0, 5.0}},
			b:        Point{Objectives: []float64{3.0, 3.0}},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dominates(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("dominates() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFrontier_Add(t *testing.T) {
	f := NewFrontier()

	// Add first point
	p1 := Point{ID: "1", Objectives: []float64{1.0, 1.0}}
	if !f.Add(p1) {
		t.Error("expected first point to be added")
	}
	if f.Len() != 1 {
		t.Errorf("expected 1 point, got %d", f.Len())
	}

	// Add dominated point
	p2 := Point{ID: "2", Objectives: []float64{0.5, 0.5}}
	if f.Add(p2) {
		t.Error("expected dominated point to not be added")
	}
	if f.Len() != 1 {
		t.Errorf("expected 1 point, got %d", f.Len())
	}

	// Add non-dominated point
	p3 := Point{ID: "3", Objectives: []float64{2.0, 0.5}}
	if !f.Add(p3) {
		t.Error("expected non-dominated point to be added")
	}
	if f.Len() != 2 {
		t.Errorf("expected 2 points, got %d", f.Len())
	}

	// Add point that dominates existing
	p4 := Point{ID: "4", Objectives: []float64{3.0, 3.0}}
	if !f.Add(p4) {
		t.Error("expected dominating point to be added")
	}
	// Should remove dominated points
	if f.Len() != 1 {
		t.Errorf("expected 1 point (dominating one), got %d", f.Len())
	}
}

func TestFrontier_Get(t *testing.T) {
	f := NewFrontier()
	f.Add(Point{ID: "1", Objectives: []float64{1.0, 2.0}})
	f.Add(Point{ID: "2", Objectives: []float64{2.0, 1.0}})

	points := f.Get()
	if len(points) != 2 {
		t.Errorf("expected 2 points, got %d", len(points))
	}

	// Verify it's a copy
	points[0].ID = "modified"
	original := f.Get()
	if original[0].ID == "modified" {
		t.Error("Get() should return a copy")
	}
}

func TestDistance(t *testing.T) {
	tests := []struct {
		name     string
		a        Point
		b        Point
		expected float64
	}{
		{
			name:     "same point",
			a:        Point{Objectives: []float64{1.0, 2.0}},
			b:        Point{Objectives: []float64{1.0, 2.0}},
			expected: 0.0,
		},
		{
			name:     "different points",
			a:        Point{Objectives: []float64{0.0, 0.0}},
			b:        Point{Objectives: []float64{3.0, 4.0}},
			expected: 25.0, // 3^2 + 4^2 = 25
		},
		{
			name:     "different lengths",
			a:        Point{Objectives: []float64{1.0, 2.0}},
			b:        Point{Objectives: []float64{1.0}},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := distance(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("distance() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFrontier_Spacing(t *testing.T) {
	// Empty frontier
	f := NewFrontier()
	if f.Spacing() != 0.0 {
		t.Errorf("expected 0 spacing for empty frontier, got %f", f.Spacing())
	}

	// Single point
	f.Add(Point{Objectives: []float64{1.0, 1.0}})
	if f.Spacing() != 0.0 {
		t.Errorf("expected 0 spacing for single point, got %f", f.Spacing())
	}

	// Multiple points
	f.Add(Point{Objectives: []float64{2.0, 2.0}})
	f.Add(Point{Objectives: []float64{3.0, 3.0}})
	spacing := f.Spacing()
	if spacing < 0 {
		t.Errorf("expected non-negative spacing, got %f", spacing)
	}
}

func TestFrontier_SelectByCoverage(t *testing.T) {
	f := NewFrontier()
	// Use non-dominated points (trade-offs between objectives)
	f.Add(Point{ID: "1", Objectives: []float64{1.0, 3.0}}) // Low on obj1, high on obj2
	f.Add(Point{ID: "2", Objectives: []float64{2.0, 2.0}}) // Balanced
	f.Add(Point{ID: "3", Objectives: []float64{3.0, 1.0}}) // High on obj1, low on obj2

	// Select all
	indices := f.SelectByCoverage(5)
	if len(indices) != 3 {
		t.Errorf("expected 3 indices when requesting more than available, got %d", len(indices))
	}

	// Select subset
	indices = f.SelectByCoverage(2)
	if len(indices) != 2 {
		t.Errorf("expected 2 indices, got %d", len(indices))
	}

	// Check indices are valid
	for _, idx := range indices {
		if idx < 0 || idx >= f.Len() {
			t.Errorf("invalid index %d", idx)
		}
	}
}

func TestNewArchive(t *testing.T) {
	a := NewArchive(10)
	if a.Len() != 0 {
		t.Errorf("expected empty archive, got %d points", a.Len())
	}
	if a.maxSize != 10 {
		t.Errorf("expected maxSize=10, got %d", a.maxSize)
	}
}

func TestArchive_Add(t *testing.T) {
	a := NewArchive(3)

	// Add non-dominated points from different generations (multi-objective trade-offs)
	if !a.Add(Point{ID: "1", Objectives: []float64{1.0, 3.0}}, 1) { // Low obj1, high obj2
		t.Error("expected first point to be added")
	}
	if !a.Add(Point{ID: "2", Objectives: []float64{2.0, 2.0}}, 2) { // Balanced
		t.Error("expected second point to be added")
	}
	if !a.Add(Point{ID: "3", Objectives: []float64{3.0, 1.0}}, 3) { // High obj1, low obj2
		t.Error("expected third point to be added")
	}

	if a.Len() != 3 {
		t.Errorf("expected 3 points, got %d", a.Len())
	}

	// Add fourth point - should evict oldest (generation 1)
	if !a.Add(Point{ID: "4", Objectives: []float64{2.5, 1.5}}, 4) { // New trade-off
		t.Error("expected fourth point to be added")
	}

	if a.Len() != 3 {
		t.Errorf("expected 3 points after eviction, got %d", a.Len())
	}

	// Check that oldest was evicted
	points := a.Get()
	foundOld := false
	for _, p := range points {
		if p.ID == "1" {
			foundOld = true
			break
		}
	}
	if foundOld {
		t.Error("expected oldest point (ID=1) to be evicted")
	}
}

func TestArchive_Get(t *testing.T) {
	a := NewArchive(10)
	a.Add(Point{ID: "1", Objectives: []float64{1.0}}, 1)

	points := a.Get()
	if len(points) != 1 {
		t.Errorf("expected 1 point, got %d", len(points))
	}
}
