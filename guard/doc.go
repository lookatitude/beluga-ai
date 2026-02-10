// Package guard provides a three-stage safety pipeline for the Beluga AI
// framework. It validates content at three points: input (user messages),
// output (model responses), and tool (tool call arguments). Each stage runs
// a configurable set of Guard implementations that can block, modify, or
// allow content to pass through.
//
// # Guard Interface
//
// The core Guard interface requires two methods:
//
//   - Name returns a unique identifier for the guard.
//   - Validate checks content and returns a GuardResult indicating whether
//     the content is allowed, along with an optional modified version.
//
// # Built-in Guards
//
// The package ships with four built-in guard implementations:
//
//   - PromptInjectionDetector detects common prompt injection patterns using
//     configurable regular expressions.
//   - PIIRedactor detects and redacts personally identifiable information
//     (email, phone, SSN, credit card, IP address) using regex-based patterns.
//   - ContentFilter performs keyword-based content moderation with a
//     configurable match threshold.
//   - Spotlighting wraps untrusted content in delimiters to isolate it
//     from trusted instructions, reducing prompt injection effectiveness.
//
// # Pipeline
//
// Guards are composed into a Pipeline using the Input, Output, and Tool
// stage options. The Pipeline runs guards sequentially within each stage;
// the first guard that blocks stops the pipeline for that stage. Modified
// content from one guard is passed to subsequent guards.
//
// # Registry
//
// The package follows the standard Beluga registry pattern with Register,
// New, and List functions. Built-in guards register themselves via init.
// External guard providers (Azure Content Safety, Lakera, NeMo, etc.) are
// available under guard/providers/.
//
// # Usage
//
// Create a pipeline with input, output, and tool guards:
//
//	p := guard.NewPipeline(
//	    guard.Input(guard.NewPromptInjectionDetector()),
//	    guard.Output(guard.NewPIIRedactor(guard.DefaultPIIPatterns...)),
//	    guard.Tool(guard.NewContentFilter(guard.WithKeywords("drop", "delete"))),
//	)
//	result, err := p.ValidateInput(ctx, userMessage)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	if !result.Allowed {
//	    fmt.Println("blocked:", result.Reason)
//	}
//
// Use the registry to create guards by name:
//
//	g, err := guard.New("prompt_injection_detector", nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	result, err := g.Validate(ctx, guard.GuardInput{Content: text, Role: "input"})
package guard
