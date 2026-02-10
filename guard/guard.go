package guard

import "context"

// Guard validates content at any stage of the safety pipeline. Implementations
// must be safe for concurrent use.
type Guard interface {
	// Name returns a unique identifier for this guard, used in logging and
	// error reporting.
	Name() string

	// Validate checks the given input and returns a result indicating whether
	// the content is allowed, along with an optional modified version.
	Validate(ctx context.Context, input GuardInput) (GuardResult, error)
}

// GuardInput carries the content to be validated along with metadata about
// the validation stage and arbitrary key-value pairs for guard-specific
// configuration.
type GuardInput struct {
	// Content is the text to validate.
	Content string

	// Role identifies the pipeline stage: "input", "output", or "tool".
	Role string

	// Metadata carries guard-specific key-value pairs, such as a tool name
	// for tool-stage validation.
	Metadata map[string]any
}

// GuardResult conveys the outcome of a Guard.Validate call.
type GuardResult struct {
	// Allowed is true when the content passes validation.
	Allowed bool

	// Reason explains why the content was blocked or modified. Empty when
	// Allowed is true and no modification occurred.
	Reason string

	// Modified holds an optional sanitized version of the content. When
	// non-empty, downstream consumers should use this instead of the original.
	Modified string

	// GuardName identifies which guard produced this result.
	GuardName string
}

// GuardFactory creates a Guard from an arbitrary configuration map. Factories
// are stored in the package-level registry and invoked by New.
type GuardFactory func(cfg map[string]any) (Guard, error)
