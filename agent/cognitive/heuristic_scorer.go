package cognitive

import (
	"context"
	"regexp"
	"strings"
	"unicode/utf8"
)

// HeuristicScorer classifies input complexity using zero-cost heuristics
// including token count estimation, keyword detection, and structural
// analysis. It requires no LLM calls and is suitable as a default scorer.
type HeuristicScorer struct {
	// complexThreshold is the token count above which input is considered complex.
	complexThreshold int
	// moderateThreshold is the token count above which input is considered moderate.
	moderateThreshold int
}

// Compile-time interface check.
var _ ComplexityScorer = (*HeuristicScorer)(nil)

// HeuristicOption configures a HeuristicScorer.
type HeuristicOption func(*HeuristicScorer)

// WithComplexThreshold sets the token count threshold for complex classification.
// Inputs with estimated token counts above this value are classified as Complex.
// Defaults to 100.
func WithComplexThreshold(n int) HeuristicOption {
	return func(s *HeuristicScorer) {
		if n > 0 {
			s.complexThreshold = n
		}
	}
}

// WithModerateThreshold sets the token count threshold for moderate classification.
// Inputs with estimated token counts above this value (but below complex threshold)
// are classified as Moderate. Defaults to 30.
func WithModerateThreshold(n int) HeuristicOption {
	return func(s *HeuristicScorer) {
		if n > 0 {
			s.moderateThreshold = n
		}
	}
}

// NewHeuristicScorer creates a new HeuristicScorer with the given options.
func NewHeuristicScorer(opts ...HeuristicOption) *HeuristicScorer {
	s := &HeuristicScorer{
		complexThreshold:  100,
		moderateThreshold: 30,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// complexKeywords are phrases that indicate analytical or multi-step reasoning.
var complexKeywords = []string{
	"step by step",
	"compare and contrast",
	"analyze",
	"evaluate",
	"synthesize",
	"explain why",
	"what are the implications",
	"trade-offs",
	"tradeoffs",
	"pros and cons",
	"critically",
	"in depth",
	"comprehensive",
	"multi-step",
	"chain of thought",
}

// mathPattern detects mathematical expressions in input.
var mathPattern = regexp.MustCompile(`[\d]+\s*[+\-*/^=<>]+\s*[\d]+|∫|∑|∏|lim|d[xy]/d[xy]|√|log|ln\(`)

// multiQuestionPattern detects multiple questions in the input.
var multiQuestionPattern = regexp.MustCompile(`\?[^?]*\?`)

// Score evaluates input complexity using heuristics. It considers token count,
// complex keywords, mathematical expressions, and multi-question patterns.
func (s *HeuristicScorer) Score(_ context.Context, input string) (ComplexityScore, error) {
	lower := strings.ToLower(input)
	tokenCount := estimateTokens(input)

	// Track complexity signals
	signals := 0
	reasons := make([]string, 0, 4)

	// Check token count
	if tokenCount >= s.complexThreshold {
		signals += 2
		reasons = append(reasons, "high token count")
	} else if tokenCount >= s.moderateThreshold {
		signals++
		reasons = append(reasons, "moderate token count")
	}

	// Check complex keywords
	for _, kw := range complexKeywords {
		if strings.Contains(lower, kw) {
			signals += 2
			reasons = append(reasons, "complex keyword: "+kw)
			break // one keyword match is enough
		}
	}

	// Check math expressions
	if mathPattern.MatchString(input) {
		signals += 2
		reasons = append(reasons, "mathematical expression detected")
	}

	// Check multiple questions
	if multiQuestionPattern.MatchString(input) {
		signals++
		reasons = append(reasons, "multiple questions detected")
	}

	// Classify based on accumulated signals
	var level ComplexityLevel
	var confidence float64

	switch {
	case signals >= 3:
		level = Complex
		confidence = clamp(0.6+float64(signals)*0.05, 0.0, 1.0)
	case signals >= 1:
		level = Moderate
		confidence = clamp(0.5+float64(signals)*0.1, 0.0, 1.0)
	default:
		level = Simple
		confidence = 0.8 // high confidence for simple inputs
	}

	reason := "heuristic analysis"
	if len(reasons) > 0 {
		reason = strings.Join(reasons, "; ")
	}

	return ComplexityScore{
		Level:      level,
		Confidence: confidence,
		Reason:     reason,
	}, nil
}

// estimateTokens estimates the token count using the ~4 chars per token rule.
func estimateTokens(text string) int {
	n := utf8.RuneCountInString(text)
	if n == 0 {
		return 0
	}
	return (n + 3) / 4
}

// clamp constrains v to the range [min, max].
func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
