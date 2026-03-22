package optimizer

import (
	"context"
	"testing"
)

func TestExactMatchMetric_Score(t *testing.T) {
	tests := []struct {
		name       string
		metric     ExactMatchMetric
		example    Example
		prediction Prediction
		want       float64
		wantErr    bool
	}{
		{
			name:       "exact match on default field",
			metric:     ExactMatchMetric{},
			example:    Example{Outputs: map[string]any{"answer": "Paris"}},
			prediction: Prediction{Text: "Paris"},
			want:       1.0,
		},
		{
			name:       "no match on default field",
			metric:     ExactMatchMetric{},
			example:    Example{Outputs: map[string]any{"answer": "Paris"}},
			prediction: Prediction{Text: "London"},
			want:       0.0,
		},
		{
			name:       "custom field match",
			metric:     ExactMatchMetric{Field: "city"},
			example:    Example{Outputs: map[string]any{"city": "Tokyo"}},
			prediction: Prediction{Outputs: map[string]any{"city": "Tokyo"}},
			want:       1.0,
		},
		{
			name:       "case insensitive match",
			metric:     ExactMatchMetric{CaseInsensitive: true},
			example:    Example{Outputs: map[string]any{"answer": "Paris"}},
			prediction: Prediction{Text: "paris"},
			want:       1.0,
		},
		{
			name:       "case sensitive mismatch",
			metric:     ExactMatchMetric{},
			example:    Example{Outputs: map[string]any{"answer": "Paris"}},
			prediction: Prediction{Text: "paris"},
			want:       0.0,
		},
		{
			name:       "missing expected field",
			metric:     ExactMatchMetric{},
			example:    Example{Outputs: map[string]any{"city": "Paris"}},
			prediction: Prediction{Text: "Paris"},
			wantErr:    true,
		},
		{
			name:       "prediction output field preferred over text",
			metric:     ExactMatchMetric{},
			example:    Example{Outputs: map[string]any{"answer": "42"}},
			prediction: Prediction{Text: "wrong", Outputs: map[string]any{"answer": "42"}},
			want:       1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.metric.Score(context.Background(), tt.example, tt.prediction)
			if (err != nil) != tt.wantErr {
				t.Errorf("error: got %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("score: got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContainsMetric_Score(t *testing.T) {
	tests := []struct {
		name       string
		metric     ContainsMetric
		example    Example
		prediction Prediction
		want       float64
		wantErr    bool
	}{
		{
			name:       "contains match",
			metric:     ContainsMetric{},
			example:    Example{Outputs: map[string]any{"answer": "Paris"}},
			prediction: Prediction{Text: "The capital of France is Paris."},
			want:       1.0,
		},
		{
			name:       "no contains match",
			metric:     ContainsMetric{},
			example:    Example{Outputs: map[string]any{"answer": "Paris"}},
			prediction: Prediction{Text: "The capital is London"},
			want:       0.0,
		},
		{
			name:       "case insensitive contains",
			metric:     ContainsMetric{CaseInsensitive: true},
			example:    Example{Outputs: map[string]any{"answer": "Paris"}},
			prediction: Prediction{Text: "the answer is paris"},
			want:       1.0,
		},
		{
			name:       "custom field contains",
			metric:     ContainsMetric{Field: "result"},
			example:    Example{Outputs: map[string]any{"result": "42"}},
			prediction: Prediction{Outputs: map[string]any{"result": "The answer is 42."}},
			want:       1.0,
		},
		{
			name:       "missing expected field",
			metric:     ContainsMetric{},
			example:    Example{Outputs: map[string]any{"city": "Paris"}},
			prediction: Prediction{Text: "Paris"},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.metric.Score(context.Background(), tt.example, tt.prediction)
			if (err != nil) != tt.wantErr {
				t.Errorf("error: got %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("score: got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMetricFunc(t *testing.T) {
	fn := MetricFunc(func(_ context.Context, _ Example, pred Prediction) (float64, error) {
		if pred.Text == "correct" {
			return 1.0, nil
		}
		return 0.0, nil
	})

	got, err := fn.Score(context.Background(), Example{}, Prediction{Text: "correct"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 1.0 {
		t.Errorf("got %v, want 1.0", got)
	}

	got, err = fn.Score(context.Background(), Example{}, Prediction{Text: "wrong"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 0.0 {
		t.Errorf("got %v, want 0.0", got)
	}
}

func TestMetricRegistry(t *testing.T) {
	// Built-in metrics should be registered via init().
	metrics := ListMetrics()
	if len(metrics) < 2 {
		t.Fatalf("expected at least 2 built-in metrics, got %d: %v", len(metrics), metrics)
	}

	// Check that exact_match and contains are registered.
	found := map[string]bool{}
	for _, name := range metrics {
		found[name] = true
	}
	if !found["exact_match"] {
		t.Error("exact_match metric not registered")
	}
	if !found["contains"] {
		t.Error("contains metric not registered")
	}
}

func TestNewMetric_ExactMatch(t *testing.T) {
	m, err := NewMetric("exact_match", MetricConfig{
		Extra: map[string]any{
			"field":            "answer",
			"case_insensitive": true,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	score, err := m.Score(context.Background(),
		Example{Outputs: map[string]any{"answer": "Hello"}},
		Prediction{Text: "hello"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if score != 1.0 {
		t.Errorf("case insensitive exact match: got %v, want 1.0", score)
	}
}

func TestNewMetric_Contains(t *testing.T) {
	m, err := NewMetric("contains", MetricConfig{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	score, err := m.Score(context.Background(),
		Example{Outputs: map[string]any{"answer": "42"}},
		Prediction{Text: "The answer is 42"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if score != 1.0 {
		t.Errorf("contains: got %v, want 1.0", score)
	}
}

func TestNewMetric_NotRegistered(t *testing.T) {
	_, err := NewMetric("nonexistent", MetricConfig{})
	if err == nil {
		t.Fatal("expected error for unregistered metric")
	}
}
