package trajectory

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockMetric is a simple mock for testing the registry.
type mockMetric struct {
	name string
}

func (m *mockMetric) Name() string { return m.name }
func (m *mockMetric) ScoreTrajectory(_ context.Context, _ Trajectory) (*TrajectoryScore, error) {
	return &TrajectoryScore{Overall: 1.0}, nil
}

func TestRegistry_RegisterAndNew(t *testing.T) {
	// Save and restore registry state.
	mu.Lock()
	origRegistry := registry
	registry = make(map[string]Factory)
	mu.Unlock()
	defer func() {
		mu.Lock()
		registry = origRegistry
		mu.Unlock()
	}()

	Register("test_metric", func(_ map[string]any) (TrajectoryMetric, error) {
		return &mockMetric{name: "test_metric"}, nil
	})

	m, err := New("test_metric", nil)
	require.NoError(t, err)
	assert.Equal(t, "test_metric", m.Name())
}

func TestRegistry_NewUnknown(t *testing.T) {
	_, err := New("nonexistent_metric_xyz", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown metric")
}

func TestRegistry_List(t *testing.T) {
	// Save and restore registry state.
	mu.Lock()
	origRegistry := registry
	registry = make(map[string]Factory)
	mu.Unlock()
	defer func() {
		mu.Lock()
		registry = origRegistry
		mu.Unlock()
	}()

	Register("beta", func(_ map[string]any) (TrajectoryMetric, error) { return nil, nil })
	Register("alpha", func(_ map[string]any) (TrajectoryMetric, error) { return nil, nil })

	names := List()
	assert.Equal(t, []string{"alpha", "beta"}, names)
}
