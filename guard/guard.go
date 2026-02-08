// Package guard provides a three-stage safety pipeline for the Beluga AI
// framework. It validates content at three points: input (user messages),
// output (model responses), and tool (tool call arguments). Each stage runs
// a configurable set of Guard implementations that can block, modify, or
// allow content to pass through.
//
// Built-in guards include prompt injection detection, PII redaction, content
// moderation, and spotlighting (untrusted input isolation). Custom guards
// are registered via the Registry pattern and composed into a Pipeline.
//
// Usage:
//
//	p := guard.NewPipeline(
//	    guard.Input(guard.NewPromptInjectionDetector()),
//	    guard.Output(guard.NewPIIRedactor(guard.DefaultPIIPatterns...)),
//	    guard.Tool(guard.NewContentFilter()),
//	)
//	result, err := p.ValidateInput(ctx, userMessage)
//	if !result.Allowed {
//	    // handle blocked content
//	}
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
