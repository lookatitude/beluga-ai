package plancache

import "context"

// Matcher scores how well an input matches a cached template. Implementations
// return a score in [0.0, 1.0] where 1.0 is a perfect match and 0.0 is no
// match at all.
type Matcher interface {
	// Score returns a similarity score between the input and the template.
	// The score must be in the range [0.0, 1.0].
	Score(ctx context.Context, input string, tmpl *Template) (float64, error)
}
