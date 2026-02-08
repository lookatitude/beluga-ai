package prompt

import (
	"github.com/lookatitude/beluga-ai/schema"
)

// Builder constructs a prompt message sequence in cache-optimal order.
// LLM prompt caching works best when the most static content appears first,
// so Builder enforces the following slot ordering:
//
//  1. System prompt (most static, rarely changes)
//  2. Tool definitions (semi-static, change when tools are added/removed)
//  3. Static context documents (semi-static, change per deployment)
//  4. Cache breakpoint marker (explicit cache boundary)
//  5. Dynamic context messages (change per session)
//  6. User input (always changes)
//
// Use NewBuilder with functional options to configure each slot, then call
// Build to produce the ordered message list.
type Builder struct {
	systemPrompt    string
	toolDefs        []schema.ToolDefinition
	staticContext   []string
	cacheBreakpoint bool
	dynamicContext  []schema.Message
	userInput       schema.Message
}

// BuilderOption configures a Builder slot.
type BuilderOption func(*Builder)

// NewBuilder creates a new Builder with the given options applied.
func NewBuilder(opts ...BuilderOption) *Builder {
	b := &Builder{}
	for _, opt := range opts {
		opt(b)
	}
	return b
}

// WithSystemPrompt sets slot 1: the system prompt. This is the most static
// content and appears first for maximum cache hit potential.
func WithSystemPrompt(prompt string) BuilderOption {
	return func(b *Builder) {
		b.systemPrompt = prompt
	}
}

// WithToolDefinitions sets slot 2: tool definitions. These change infrequently
// and appear after the system prompt.
func WithToolDefinitions(tools []schema.ToolDefinition) BuilderOption {
	return func(b *Builder) {
		b.toolDefs = tools
	}
}

// WithStaticContext sets slot 3: static context documents. These are
// semi-static content like retrieved documentation or reference material.
func WithStaticContext(docs []string) BuilderOption {
	return func(b *Builder) {
		b.staticContext = docs
	}
}

// WithCacheBreakpoint sets an explicit cache boundary marker between static
// and dynamic content. Providers that support cache control points can use
// this to indicate where the cached prefix ends.
func WithCacheBreakpoint() BuilderOption {
	return func(b *Builder) {
		b.cacheBreakpoint = true
	}
}

// WithDynamicContext sets slot 4: dynamic context messages. These change per
// session (e.g., conversation history) and appear after static content.
func WithDynamicContext(msgs []schema.Message) BuilderOption {
	return func(b *Builder) {
		b.dynamicContext = msgs
	}
}

// WithUserInput sets slot 5: the user's current input message. This always
// changes and appears last.
func WithUserInput(msg schema.Message) BuilderOption {
	return func(b *Builder) {
		b.userInput = msg
	}
}

// Build produces the ordered message list. Messages are arranged in
// cache-optimal order: system prompt → tool definitions → static context →
// cache breakpoint → dynamic context → user input. Nil/empty slots are skipped.
func (b *Builder) Build() []schema.Message {
	var msgs []schema.Message

	// Slot 1: System prompt
	if b.systemPrompt != "" {
		msgs = append(msgs, schema.NewSystemMessage(b.systemPrompt))
	}

	// Slot 2: Tool definitions as a system message describing available tools
	if len(b.toolDefs) > 0 {
		toolText := formatToolDefinitions(b.toolDefs)
		msgs = append(msgs, schema.NewSystemMessage(toolText))
	}

	// Slot 3: Static context documents
	for _, doc := range b.staticContext {
		if doc != "" {
			msgs = append(msgs, schema.NewSystemMessage(doc))
		}
	}

	// Slot 4: Cache breakpoint marker (metadata-only system message)
	if b.cacheBreakpoint {
		msg := &schema.SystemMessage{
			Parts:    []schema.ContentPart{schema.TextPart{Text: ""}},
			Metadata: map[string]any{"cache_breakpoint": true},
		}
		msgs = append(msgs, msg)
	}

	// Slot 5: Dynamic context messages
	msgs = append(msgs, b.dynamicContext...)

	// Slot 6: User input
	if b.userInput != nil {
		msgs = append(msgs, b.userInput)
	}

	return msgs
}

// formatToolDefinitions renders tool definitions as a text description.
func formatToolDefinitions(tools []schema.ToolDefinition) string {
	var buf []byte
	buf = append(buf, "Available tools:\n"...)
	for i, t := range tools {
		if i > 0 {
			buf = append(buf, '\n')
		}
		buf = append(buf, "- "...)
		buf = append(buf, t.Name...)
		if t.Description != "" {
			buf = append(buf, ": "...)
			buf = append(buf, t.Description...)
		}
	}
	return string(buf)
}
