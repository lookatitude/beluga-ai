// Package prompt provides prompt template management and cache-optimized prompt
// building for the Beluga AI framework. It supports template rendering with
// Go's text/template syntax, versioned template management via pluggable
// providers, and a builder that orders prompt content for optimal LLM cache hits.
//
// # Template
//
// Template represents a versioned prompt template with Go text/template syntax.
// Templates define default variable values and carry arbitrary metadata.
// Use Validate to check that the template content is parseable, and Render
// to execute the template with provided variables.
//
// # PromptManager Interface
//
// The PromptManager interface provides versioned access to prompt templates:
//
//   - Get retrieves a template by name and version.
//   - Render retrieves, renders, and returns a template as schema.Message values.
//   - List returns summary information for all available templates.
//
// Implementations include the filesystem-based provider in prompt/providers/file.
//
// # Builder
//
// Builder constructs a prompt message sequence in cache-optimal order. LLM
// prompt caching works best when static content appears first, so Builder
// enforces this slot ordering:
//
//  1. System prompt (most static)
//  2. Tool definitions (semi-static)
//  3. Static context documents (semi-static)
//  4. Cache breakpoint marker (explicit cache boundary)
//  5. Dynamic context messages (per-session)
//  6. User input (always changes)
//
// # Usage
//
// Template rendering:
//
//	tmpl := &prompt.Template{
//	    Name:    "greeting",
//	    Version: "1.0.0",
//	    Content: "Hello, {{.name}}! Welcome to {{.system}}.",
//	    Variables: map[string]string{"system": "Beluga"},
//	}
//	result, err := tmpl.Render(map[string]any{"name": "Alice"})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Cache-optimized prompt building:
//
//	msgs := prompt.NewBuilder(
//	    prompt.WithSystemPrompt("You are a helpful assistant."),
//	    prompt.WithStaticContext([]string{"Reference: ..."}),
//	    prompt.WithCacheBreakpoint(),
//	    prompt.WithDynamicContext(history),
//	    prompt.WithUserInput(schema.NewHumanMessage("Hello")),
//	).Build()
package prompt
