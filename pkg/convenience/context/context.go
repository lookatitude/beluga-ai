// Package context provides utilities for building RAG context.
// It offers a fluent builder pattern for assembling context from
// retrieved documents, conversation history, and templates.
//
// Example usage:
//
//	ctx := context.NewBuilder().
//	    AddDocuments(docs, scores).
//	    AddHistory(history).
//	    WithSystemPrompt("You are helpful").
//	    WithTemplate("...").
//	    Build()
package context

import (
	"fmt"
	"sort"
	"strings"
)

// Document represents a document in the context.
type Document struct {
	Content  string
	Metadata map[string]any
	ID       string
}

// Message represents a conversation message.
type Message struct {
	Role    string
	Content string
}

// Builder provides a fluent interface for building RAG context.
type Builder struct {
	documents      []DocumentWithScore
	history        []Message
	systemPrompt   string
	template       string
	maxDocLength   int
	maxHistorySize int
	metadata       map[string]any
}

// DocumentWithScore pairs a document with its relevance score.
type DocumentWithScore struct {
	Document Document
	Score    float64
}

// NewBuilder creates a new context builder.
func NewBuilder() *Builder {
	return &Builder{
		maxDocLength:   10000,
		maxHistorySize: 50,
		metadata:       make(map[string]any),
	}
}

// AddDocuments adds documents with their relevance scores.
func (b *Builder) AddDocuments(docs []Document, scores []float64) *Builder {
	for i, doc := range docs {
		score := 0.0
		if i < len(scores) {
			score = scores[i]
		}
		b.documents = append(b.documents, DocumentWithScore{
			Document: doc,
			Score:    score,
		})
	}
	return b
}

// AddDocument adds a single document with score.
func (b *Builder) AddDocument(doc Document, score float64) *Builder {
	b.documents = append(b.documents, DocumentWithScore{
		Document: doc,
		Score:    score,
	})
	return b
}

// AddHistory adds conversation history.
func (b *Builder) AddHistory(messages []Message) *Builder {
	b.history = append(b.history, messages...)
	return b
}

// AddMessage adds a single message to history.
func (b *Builder) AddMessage(role, content string) *Builder {
	b.history = append(b.history, Message{
		Role:    role,
		Content: content,
	})
	return b
}

// WithSystemPrompt sets the system prompt.
func (b *Builder) WithSystemPrompt(prompt string) *Builder {
	b.systemPrompt = prompt
	return b
}

// WithTemplate sets a custom template for formatting.
// Available placeholders: {{system}}, {{documents}}, {{history}}, {{question}}.
func (b *Builder) WithTemplate(template string) *Builder {
	b.template = template
	return b
}

// WithMaxDocumentLength sets the maximum length for document content.
func (b *Builder) WithMaxDocumentLength(length int) *Builder {
	b.maxDocLength = length
	return b
}

// WithMaxHistorySize sets the maximum number of history messages.
func (b *Builder) WithMaxHistorySize(size int) *Builder {
	b.maxHistorySize = size
	return b
}

// WithMetadata adds metadata that can be used in templates.
func (b *Builder) WithMetadata(key string, value any) *Builder {
	b.metadata[key] = value
	return b
}

// SortByScore sorts documents by relevance score (highest first).
func (b *Builder) SortByScore() *Builder {
	sort.Slice(b.documents, func(i, j int) bool {
		return b.documents[i].Score > b.documents[j].Score
	})
	return b
}

// Build creates the final context string.
func (b *Builder) Build() string {
	if b.template != "" {
		return b.buildWithTemplate()
	}
	return b.buildDefault()
}

// BuildForQuestion builds context for a specific question.
func (b *Builder) BuildForQuestion(question string) string {
	if b.template != "" {
		return b.buildWithTemplateAndQuestion(question)
	}
	return b.buildDefaultWithQuestion(question)
}

func (b *Builder) buildDefault() string {
	var parts []string

	if b.systemPrompt != "" {
		parts = append(parts, b.systemPrompt)
	}

	if len(b.documents) > 0 {
		parts = append(parts, b.formatDocuments())
	}

	if len(b.history) > 0 {
		parts = append(parts, b.formatHistory())
	}

	return strings.Join(parts, "\n\n")
}

func (b *Builder) buildDefaultWithQuestion(question string) string {
	var parts []string

	if b.systemPrompt != "" {
		parts = append(parts, b.systemPrompt)
	}

	if len(b.documents) > 0 {
		parts = append(parts, "Context:\n"+b.formatDocuments())
	}

	if len(b.history) > 0 {
		parts = append(parts, "Conversation History:\n"+b.formatHistory())
	}

	parts = append(parts, "Question: "+question)

	return strings.Join(parts, "\n\n")
}

func (b *Builder) buildWithTemplate() string {
	result := b.template

	result = strings.ReplaceAll(result, "{{system}}", b.systemPrompt)
	result = strings.ReplaceAll(result, "{{documents}}", b.formatDocuments())
	result = strings.ReplaceAll(result, "{{history}}", b.formatHistory())

	for key, value := range b.metadata {
		placeholder := fmt.Sprintf("{{%s}}", key)
		result = strings.ReplaceAll(result, placeholder, fmt.Sprint(value))
	}

	return result
}

func (b *Builder) buildWithTemplateAndQuestion(question string) string {
	result := b.buildWithTemplate()
	result = strings.ReplaceAll(result, "{{question}}", question)
	return result
}

func (b *Builder) formatDocuments() string {
	var parts []string
	for i, docScore := range b.documents {
		content := docScore.Document.Content
		if len(content) > b.maxDocLength {
			content = content[:b.maxDocLength] + "..."
		}
		parts = append(parts, fmt.Sprintf("[Document %d (score: %.2f)]\n%s", i+1, docScore.Score, content))
	}
	return strings.Join(parts, "\n\n")
}

func (b *Builder) formatHistory() string {
	msgs := b.history
	if len(msgs) > b.maxHistorySize {
		msgs = msgs[len(msgs)-b.maxHistorySize:]
	}

	var parts []string
	for _, msg := range msgs {
		parts = append(parts, fmt.Sprintf("%s: %s", msg.Role, msg.Content))
	}
	return strings.Join(parts, "\n")
}

// GetDocuments returns the documents added to the builder.
func (b *Builder) GetDocuments() []DocumentWithScore {
	return b.documents
}

// GetHistory returns the conversation history.
func (b *Builder) GetHistory() []Message {
	return b.history
}

// Context represents a built RAG context.
type Context struct {
	Content    string
	Documents  []DocumentWithScore
	History    []Message
	Metadata   map[string]any
	TokenCount int
}

// BuildContext creates a Context struct with all information.
func (b *Builder) BuildContext(question string) *Context {
	return &Context{
		Content:    b.BuildForQuestion(question),
		Documents:  b.documents,
		History:    b.history,
		Metadata:   b.metadata,
		TokenCount: b.estimateTokens(),
	}
}

func (b *Builder) estimateTokens() int {
	// Simple token estimation: ~4 characters per token
	content := b.Build()
	return len(content) / 4
}
