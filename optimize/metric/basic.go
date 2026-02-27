package metric

import (
	"strings"

	"github.com/lookatitude/beluga-ai/optimize"
)

// ExactMatch returns 1.0 if predictions match exactly, 0.0 otherwise.
func ExactMatch(example optimize.Example, pred optimize.Prediction, trace *optimize.Trace) float64 {
	for key, expected := range example.Outputs {
		actual, ok := pred.Outputs[key]
		if !ok {
			return 0.0
		}
		if expected != actual {
			return 0.0
		}
	}
	return 1.0
}

// Contains checks if the prediction contains the expected answer.
func Contains(example optimize.Example, pred optimize.Prediction, trace *optimize.Trace) float64 {
	for key, expected := range example.Outputs {
		expectedStr, ok1 := expected.(string)
		actual, ok2 := pred.Outputs[key]
		if !ok1 || !ok2 {
			continue
		}
		actualStr, ok := actual.(string)
		if !ok {
			continue
		}
		if strings.Contains(strings.ToLower(actualStr), strings.ToLower(expectedStr)) {
			return 1.0
		}
	}
	return 0.0
}

// Tokenizer splits text into tokens.
type Tokenizer interface {
	Tokenize(text string) []string
}

// WordTokenizer splits on whitespace.
type WordTokenizer struct{}

// Tokenize implements Tokenizer.
func (t WordTokenizer) Tokenize(text string) []string {
	return strings.Fields(strings.ToLower(text))
}

// F1Metric computes F1 score based on token overlap.
type F1Metric struct {
	Tokenizer Tokenizer
	Field     string // Field to compare (empty = compare all)
}

// Evaluate implements optimize.Metric.
func (m *F1Metric) Evaluate(example optimize.Example, pred optimize.Prediction, trace *optimize.Trace) (float64, error) {
	if m.Tokenizer == nil {
		m.Tokenizer = WordTokenizer{}
	}

	var totalF1 float64
	var count int

	for key, expected := range example.Outputs {
		if m.Field != "" && key != m.Field {
			continue
		}

		expectedStr, ok1 := expected.(string)
		actual, ok2 := pred.Outputs[key]
		if !ok1 || !ok2 {
			continue
		}
		actualStr, ok := actual.(string)
		if !ok {
			continue
		}

		goldTokens := m.Tokenizer.Tokenize(expectedStr)
		predTokens := m.Tokenizer.Tokenize(actualStr)

		f1 := computeF1(goldTokens, predTokens)
		totalF1 += f1
		count++
	}

	if count == 0 {
		return 0.0, nil
	}
	return totalF1 / float64(count), nil
}

func computeF1(gold, pred []string) float64 {
	if len(gold) == 0 && len(pred) == 0 {
		return 1.0
	}
	if len(gold) == 0 || len(pred) == 0 {
		return 0.0
	}

	// Count overlaps
	goldSet := make(map[string]int)
	for _, t := range gold {
		goldSet[t]++
	}

	overlap := 0
	for _, t := range pred {
		if goldSet[t] > 0 {
			overlap++
			goldSet[t]--
		}
	}

	precision := float64(overlap) / float64(len(pred))
	recall := float64(overlap) / float64(len(gold))

	if precision+recall == 0 {
		return 0.0
	}
	return 2 * (precision * recall) / (precision + recall)
}

// MultiMetric combines multiple metrics with weights.
type MultiMetric struct {
	Metrics []optimize.Metric
	Weights []float64
}

// Evaluate implements optimize.Metric.
func (m *MultiMetric) Evaluate(example optimize.Example, pred optimize.Prediction, trace *optimize.Trace) (float64, error) {
	if len(m.Metrics) == 0 {
		return 0.0, nil
	}

	var totalWeight float64
	for _, w := range m.Weights {
		totalWeight += w
	}
	if totalWeight == 0 {
		totalWeight = float64(len(m.Metrics))
		weights := make([]float64, len(m.Metrics))
		for i := range weights {
			weights[i] = 1.0
		}
		m.Weights = weights
	}

	var weightedScore float64
	for i, metric := range m.Metrics {
		score, err := metric.Evaluate(example, pred, trace)
		if err != nil {
			return 0.0, err
		}
		weightedScore += score * m.Weights[i] / totalWeight
	}

	return weightedScore, nil
}

// PassAtK checks if at least one of k predictions passes.
type PassAtK struct {
	Metric optimize.Metric
	K      int
}

// Evaluate implements optimize.Metric.
func (p *PassAtK) Evaluate(example optimize.Example, pred optimize.Prediction, trace *optimize.Trace) (float64, error) {
	// For PassAtK, we need multiple predictions which isn't supported in the basic interface.
	// This is a placeholder for when we have multi-prediction support.
	score, err := p.Metric.Evaluate(example, pred, trace)
	if err != nil {
		return 0.0, err
	}
	if score > 0 {
		return 1.0, nil
	}
	return 0.0, nil
}
