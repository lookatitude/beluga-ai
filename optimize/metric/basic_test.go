package metric

import (
	"testing"

	"github.com/lookatitude/beluga-ai/optimize"
)

func TestExactMatch(t *testing.T) {
	tests := []struct {
		name     string
		example  optimize.Example
		pred     optimize.Prediction
		expected float64
	}{
		{
			name: "exact match",
			example: optimize.Example{
				Outputs: map[string]interface{}{"answer": "4"},
			},
			pred: optimize.Prediction{
				Outputs: map[string]interface{}{"answer": "4"},
			},
			expected: 1.0,
		},
		{
			name: "no match",
			example: optimize.Example{
				Outputs: map[string]interface{}{"answer": "4"},
			},
			pred: optimize.Prediction{
				Outputs: map[string]interface{}{"answer": "5"},
			},
			expected: 0.0,
		},
		{
			name: "missing key",
			example: optimize.Example{
				Outputs: map[string]interface{}{"answer": "4"},
			},
			pred: optimize.Prediction{
				Outputs: map[string]interface{}{"different": "4"},
			},
			expected: 0.0,
		},
		{
			name: "multiple fields all match",
			example: optimize.Example{
				Outputs: map[string]interface{}{"a": "1", "b": "2"},
			},
			pred: optimize.Prediction{
				Outputs: map[string]interface{}{"a": "1", "b": "2"},
			},
			expected: 1.0,
		},
		{
			name: "multiple fields one mismatch",
			example: optimize.Example{
				Outputs: map[string]interface{}{"a": "1", "b": "2"},
			},
			pred: optimize.Prediction{
				Outputs: map[string]interface{}{"a": "1", "b": "3"},
			},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExactMatch(tt.example, tt.pred, nil)
			if result != tt.expected {
				t.Errorf("Expected %f, got %f", tt.expected, result)
			}
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		example  optimize.Example
		pred     optimize.Prediction
		expected float64
	}{
		{
			name: "contains substring",
			example: optimize.Example{
				Outputs: map[string]interface{}{"answer": "Paris"},
			},
			pred: optimize.Prediction{
				Outputs: map[string]interface{}{"answer": "The capital of France is Paris."},
			},
			expected: 1.0,
		},
		{
			name: "does not contain",
			example: optimize.Example{
				Outputs: map[string]interface{}{"answer": "London"},
			},
			pred: optimize.Prediction{
				Outputs: map[string]interface{}{"answer": "The capital of France is Paris."},
			},
			expected: 0.0,
		},
		{
			name: "case insensitive",
			example: optimize.Example{
				Outputs: map[string]interface{}{"answer": "PARIS"},
			},
			pred: optimize.Prediction{
				Outputs: map[string]interface{}{"answer": "The capital is paris."},
			},
			expected: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Contains(tt.example, tt.pred, nil)
			if result != tt.expected {
				t.Errorf("Expected %f, got %f", tt.expected, result)
			}
		})
	}
}

func TestWordTokenizer(t *testing.T) {
	tokenizer := WordTokenizer{}

	tests := []struct {
		input    string
		expected []string
	}{
		{
			input:    "hello world",
			expected: []string{"hello", "world"},
		},
		{
			input:    "  multiple   spaces  ",
			expected: []string{"multiple", "spaces"},
		},
		{
			input:    "Mixed CASE Words",
			expected: []string{"mixed", "case", "words"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := tokenizer.Tokenize(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, result)
				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("Expected %v, got %v", tt.expected, result)
					return
				}
			}
		})
	}
}

func TestF1Metric(t *testing.T) {
	metric := &F1Metric{
		Tokenizer: WordTokenizer{},
		Field:     "answer",
	}

	tests := []struct {
		name     string
		example  optimize.Example
		pred     optimize.Prediction
		expected float64
	}{
		{
			name: "perfect match",
			example: optimize.Example{
				Outputs: map[string]interface{}{"answer": "hello world"},
			},
			pred: optimize.Prediction{
				Outputs: map[string]interface{}{"answer": "hello world"},
			},
			expected: 1.0,
		},
		{
			name: "partial match",
			example: optimize.Example{
				Outputs: map[string]interface{}{"answer": "hello world foo"},
			},
			pred: optimize.Prediction{
				Outputs: map[string]interface{}{"answer": "hello world bar"},
			},
			expected: 0.6666666666666666, // 2/3 precision, 2/3 recall, F1 = 0.666...
		},
		{
			name: "no match",
			example: optimize.Example{
				Outputs: map[string]interface{}{"answer": "foo bar"},
			},
			pred: optimize.Prediction{
				Outputs: map[string]interface{}{"answer": "hello world"},
			},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := metric.Evaluate(tt.example, tt.pred, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Allow small floating point differences
			if result < tt.expected-0.001 || result > tt.expected+0.001 {
				t.Errorf("Expected %f, got %f", tt.expected, result)
			}
		})
	}
}

func TestMultiMetric(t *testing.T) {
	mm := &MultiMetric{
		Metrics: []optimize.Metric{
			optimize.MetricFunc(ExactMatch),
			optimize.MetricFunc(Contains),
		},
		Weights: []float64{1.0, 1.0},
	}

	example := optimize.Example{
		Outputs: map[string]interface{}{"answer": "Paris"},
	}
	pred := optimize.Prediction{
		Outputs: map[string]interface{}{"answer": "The capital is Paris."},
	}

	result, err := mm.Evaluate(example, pred, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// ExactMatch = 0.0, Contains = 1.0, average = 0.5
	expected := 0.5
	if result != expected {
		t.Errorf("Expected %f, got %f", expected, result)
	}
}
