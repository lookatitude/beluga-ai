package optimizer

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Metric evaluates agent performance during optimization.
// Higher scores indicate better performance.
// Implementations should be safe for concurrent use.
type Metric interface {
	// Score evaluates a prediction against an expected example.
	// Returns a score where higher is better (typically 0.0 to 1.0).
	Score(ctx context.Context, example Example, prediction Prediction) (float64, error)
}

// MetricFunc adapts a function to the Metric interface.
type MetricFunc func(ctx context.Context, example Example, prediction Prediction) (float64, error)

// Score implements Metric.
func (f MetricFunc) Score(ctx context.Context, example Example, prediction Prediction) (float64, error) {
	return f(ctx, example, prediction)
}

// ExactMatchMetric scores 1.0 if the prediction text exactly matches
// the expected output field, 0.0 otherwise.
type ExactMatchMetric struct {
	// Field is the output field to compare. Defaults to "answer".
	Field string
	// CaseInsensitive enables case-insensitive comparison.
	CaseInsensitive bool
}

// Score implements Metric.
func (m *ExactMatchMetric) Score(_ context.Context, example Example, prediction Prediction) (float64, error) {
	field := m.Field
	if field == "" {
		field = "answer"
	}

	expected, ok := example.Outputs[field]
	if !ok {
		return 0.0, fmt.Errorf("expected output field %q not found in example", field)
	}

	expectedStr := fmt.Sprintf("%v", expected)
	predStr := prediction.Text
	if v, ok := prediction.Outputs[field]; ok {
		predStr = fmt.Sprintf("%v", v)
	}

	if m.CaseInsensitive {
		expectedStr = strings.ToLower(expectedStr)
		predStr = strings.ToLower(predStr)
	}

	if expectedStr == predStr {
		return 1.0, nil
	}
	return 0.0, nil
}

// ContainsMetric scores 1.0 if the prediction contains the expected value.
type ContainsMetric struct {
	// Field is the output field to check. Defaults to "answer".
	Field string
	// CaseInsensitive enables case-insensitive comparison.
	CaseInsensitive bool
}

// Score implements Metric.
func (m *ContainsMetric) Score(_ context.Context, example Example, prediction Prediction) (float64, error) {
	field := m.Field
	if field == "" {
		field = "answer"
	}

	expected, ok := example.Outputs[field]
	if !ok {
		return 0.0, fmt.Errorf("expected output field %q not found in example", field)
	}

	expectedStr := fmt.Sprintf("%v", expected)
	predStr := prediction.Text
	if v, ok := prediction.Outputs[field]; ok {
		predStr = fmt.Sprintf("%v", v)
	}

	if m.CaseInsensitive {
		expectedStr = strings.ToLower(expectedStr)
		predStr = strings.ToLower(predStr)
	}

	if strings.Contains(predStr, expectedStr) {
		return 1.0, nil
	}
	return 0.0, nil
}

// MetricFactory creates a Metric from configuration.
type MetricFactory func(cfg MetricConfig) (Metric, error)

var (
	metricMu       sync.RWMutex
	metricRegistry = make(map[string]MetricFactory)
)

// RegisterMetric registers a metric factory under the given name.
// This is typically called from init() in metric implementation files.
func RegisterMetric(name string, factory MetricFactory) {
	metricMu.Lock()
	defer metricMu.Unlock()
	metricRegistry[name] = factory
}

// NewMetric creates a new metric by name from the registry.
func NewMetric(name string, cfg MetricConfig) (Metric, error) {
	metricMu.RLock()
	factory, ok := metricRegistry[name]
	metricMu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("metric %q not registered (available: %v)", name, ListMetrics())
	}
	return factory(cfg)
}

// ListMetrics returns the sorted names of all registered metrics.
func ListMetrics() []string {
	metricMu.RLock()
	defer metricMu.RUnlock()

	names := make([]string, 0, len(metricRegistry))
	for name := range metricRegistry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Register built-in metrics.
func init() {
	RegisterMetric("exact_match", func(cfg MetricConfig) (Metric, error) {
		m := &ExactMatchMetric{}
		if field, ok := cfg.Extra["field"].(string); ok {
			m.Field = field
		}
		if ci, ok := cfg.Extra["case_insensitive"].(bool); ok {
			m.CaseInsensitive = ci
		}
		return m, nil
	})

	RegisterMetric("contains", func(cfg MetricConfig) (Metric, error) {
		m := &ContainsMetric{}
		if field, ok := cfg.Extra["field"].(string); ok {
			m.Field = field
		}
		if ci, ok := cfg.Extra["case_insensitive"].(bool); ok {
			m.CaseInsensitive = ci
		}
		return m, nil
	})
}
