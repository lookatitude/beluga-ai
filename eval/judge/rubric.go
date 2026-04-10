package judge

import (
	"fmt"
	"strings"
)

// ScoreLevel defines a named score level within a criterion, such as
// "excellent" = 1.0 or "poor" = 0.0.
type ScoreLevel struct {
	// Label is a human-readable name for this level (e.g., "excellent", "poor").
	Label string

	// Score is the numeric value for this level, in [0.0, 1.0].
	Score float64

	// Description explains what qualifies for this level.
	Description string
}

// Criterion defines a single evaluation dimension with weighted score levels.
type Criterion struct {
	// Name identifies this criterion (e.g., "accuracy", "clarity").
	Name string

	// Description explains what this criterion measures.
	Description string

	// Weight is the relative importance of this criterion. Weights are
	// normalized across all criteria in a rubric.
	Weight float64

	// Levels defines the possible score levels for this criterion,
	// ordered from lowest to highest.
	Levels []ScoreLevel
}

// Rubric defines a complete evaluation rubric with multiple weighted criteria.
// Each criterion has score levels that the LLM judge selects from.
type Rubric struct {
	// Name identifies this rubric.
	Name string

	// Description provides context for the LLM judge about what is being evaluated.
	Description string

	// Criteria contains the evaluation dimensions.
	Criteria []Criterion
}

// Validate checks that the rubric is well-formed: non-empty name, at least
// one criterion, each criterion has at least one level, and weights are positive.
func (r *Rubric) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("rubric name must not be empty")
	}
	if len(r.Criteria) == 0 {
		return fmt.Errorf("rubric %q must have at least one criterion", r.Name)
	}
	seen := make(map[string]bool, len(r.Criteria))
	for _, c := range r.Criteria {
		if c.Name == "" {
			return fmt.Errorf("rubric %q: criterion name must not be empty", r.Name)
		}
		if seen[c.Name] {
			return fmt.Errorf("rubric %q: duplicate criterion name %q", r.Name, c.Name)
		}
		seen[c.Name] = true
		if c.Weight <= 0 {
			return fmt.Errorf("rubric %q: criterion %q weight must be positive, got %f", r.Name, c.Name, c.Weight)
		}
		if len(c.Levels) == 0 {
			return fmt.Errorf("rubric %q: criterion %q must have at least one level", r.Name, c.Name)
		}
	}
	return nil
}

// ToPrompt renders the rubric as a text prompt suitable for an LLM judge.
func (r *Rubric) ToPrompt() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Evaluation Rubric: %s\n", r.Name)
	if r.Description != "" {
		fmt.Fprintf(&b, "%s\n", r.Description)
	}
	b.WriteString("\nCriteria:\n")
	for i, c := range r.Criteria {
		fmt.Fprintf(&b, "\n%d. %s (weight: %.1f)\n", i+1, c.Name, c.Weight)
		if c.Description != "" {
			fmt.Fprintf(&b, "   %s\n", c.Description)
		}
		b.WriteString("   Score levels:\n")
		for _, l := range c.Levels {
			fmt.Fprintf(&b, "   - %s (%.1f): %s\n", l.Label, l.Score, l.Description)
		}
	}
	return b.String()
}

// totalWeight returns the sum of all criterion weights.
func (r *Rubric) totalWeight() float64 {
	var total float64
	for _, c := range r.Criteria {
		total += c.Weight
	}
	return total
}
