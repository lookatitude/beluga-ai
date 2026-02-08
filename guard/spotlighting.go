package guard

import "context"

// defaultDelimiter is the default spotlighting delimiter used when none is
// specified.
const defaultDelimiter = "^^^"

// Spotlighting is a Guard that wraps untrusted content in delimiters to
// isolate it from trusted instructions. This technique reduces the
// effectiveness of prompt injection by making the boundary between trusted
// and untrusted content explicit to the model.
//
// For example, with delimiter "^^^":
//
//	^^^
//	untrusted user content here
//	^^^
type Spotlighting struct {
	delimiter string
}

// NewSpotlighting creates a Spotlighting guard with the given delimiter. If
// delimiter is empty, the default delimiter "^^^" is used.
func NewSpotlighting(delimiter string) *Spotlighting {
	if delimiter == "" {
		delimiter = defaultDelimiter
	}
	return &Spotlighting{delimiter: delimiter}
}

// Name returns "spotlighting".
func (s *Spotlighting) Name() string {
	return "spotlighting"
}

// Validate wraps the input content in delimiter markers and returns the
// wrapped version as modified content. The result is always Allowed because
// spotlighting transforms rather than blocks.
func (s *Spotlighting) Validate(_ context.Context, input GuardInput) (GuardResult, error) {
	wrapped := s.delimiter + "\n" + input.Content + "\n" + s.delimiter
	return GuardResult{
		Allowed:   true,
		Modified:  wrapped,
		Reason:    "content wrapped with spotlighting delimiters",
		GuardName: s.Name(),
	}, nil
}

func init() {
	Register("spotlighting", func(cfg map[string]any) (Guard, error) {
		delimiter, _ := cfg["delimiter"].(string)
		return NewSpotlighting(delimiter), nil
	})
}
