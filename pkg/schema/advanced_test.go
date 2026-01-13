// Package schema provides advanced test scenarios and comprehensive testing patterns.
// This file demonstrates improved testing practices including table-driven tests,
// concurrency testing, and integration test patterns.
package schema

import (
	"context"
	"sync"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/schema/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMessageCreationAdvanced provides advanced table-driven tests for message creation.
func TestMessageCreationAdvanced(t *testing.T) {
	tests := []struct {
		name        string
		description string
		createFunc  func() Message
		validate    func(t *testing.T, msg Message)
		wantErr     bool
	}{
		{
			name:        "human_message_basic",
			description: "Create basic human message",
			createFunc: func() Message {
				return NewHumanMessage("Hello, world!")
			},
			validate: func(t *testing.T, msg Message) {
				assert.Equal(t, iface.RoleHuman, msg.GetType())
				assert.Equal(t, "Hello, world!", msg.GetContent())
			},
		},
		{
			name:        "ai_message_basic",
			description: "Create basic AI message",
			createFunc: func() Message {
				return NewAIMessage("Hi there!")
			},
			validate: func(t *testing.T, msg Message) {
				assert.Equal(t, iface.RoleAssistant, msg.GetType())
				assert.Equal(t, "Hi there!", msg.GetContent())
			},
		},
		{
			name:        "system_message_basic",
			description: "Create basic system message",
			createFunc: func() Message {
				return NewSystemMessage("You are a helpful assistant.")
			},
			validate: func(t *testing.T, msg Message) {
				assert.Equal(t, iface.RoleSystem, msg.GetType())
				assert.Equal(t, "You are a helpful assistant.", msg.GetContent())
			},
		},
		{
			name:        "tool_message_basic",
			description: "Create basic tool message",
			createFunc: func() Message {
				return NewToolMessage("Tool result", "call_123")
			},
			validate: func(t *testing.T, msg Message) {
				assert.Equal(t, iface.RoleTool, msg.GetType())
				assert.Equal(t, "Tool result", msg.GetContent())
			},
		},
		{
			name:        "empty_content",
			description: "Create message with empty content",
			createFunc: func() Message {
				return NewHumanMessage("")
			},
			validate: func(t *testing.T, msg Message) {
				assert.Equal(t, iface.RoleHuman, msg.GetType())
				assert.Equal(t, "", msg.GetContent())
			},
		},
		{
			name:        "long_content",
			description: "Create message with long content",
			createFunc: func() Message {
				longContent := make([]byte, 10000)
				for i := range longContent {
					longContent[i] = 'a'
				}
				return NewHumanMessage(string(longContent))
			},
			validate: func(t *testing.T, msg Message) {
				assert.Equal(t, iface.RoleHuman, msg.GetType())
				assert.Len(t, msg.GetContent(), 10000)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)
			msg := tt.createFunc()
			tt.validate(t, msg)
		})
	}
}

// TestMessageWithContextAdvanced tests message creation with OTEL context.
func TestMessageWithContextAdvanced(t *testing.T) {
	ctx := context.Background()

	t.Run("human_message_with_context", func(t *testing.T) {
		msg := NewHumanMessageWithContext(ctx, "Test message")
		assert.Equal(t, iface.RoleHuman, msg.GetType())
		assert.Equal(t, "Test message", msg.GetContent())
	})

	t.Run("ai_message_with_context", func(t *testing.T) {
		msg := NewAIMessageWithContext(ctx, "AI response")
		assert.Equal(t, iface.RoleAssistant, msg.GetType())
		assert.Equal(t, "AI response", msg.GetContent())
	})
}

// TestDocumentCreationAdvanced provides advanced table-driven tests for document creation.
func TestDocumentCreationAdvanced(t *testing.T) {
	tests := []struct {
		name        string
		description string
		createFunc  func() Document
		validate    func(t *testing.T, doc Document)
	}{
		{
			name:        "basic_document",
			description: "Create basic document",
			createFunc: func() Document {
				return NewDocument("Test content", map[string]string{"key": "value"})
			},
			validate: func(t *testing.T, doc Document) {
				assert.Equal(t, "Test content", doc.GetContent())
				assert.Equal(t, "value", doc.Metadata["key"])
			},
		},
		{
			name:        "document_no_metadata",
			description: "Create document without metadata",
			createFunc: func() Document {
				return NewDocument("Content only", nil)
			},
			validate: func(t *testing.T, doc Document) {
				assert.Equal(t, "Content only", doc.GetContent())
				assert.Nil(t, doc.Metadata)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)
			doc := tt.createFunc()
			tt.validate(t, doc)
		})
	}
}

// TestConcurrentMessageCreation tests concurrent message creation.
func TestConcurrentMessageCreation(t *testing.T) {
	const numGoroutines = 100
	const numMessagesPerGoroutine = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	errors := make(chan error, numGoroutines*numMessagesPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numMessagesPerGoroutine; j++ {
				msg := NewHumanMessage("Message from goroutine")
				if msg.GetType() != iface.RoleHuman {
					errors <- assert.AnError
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		require.NoError(t, err)
	}
}

// TestConcurrentDocumentCreation tests concurrent document creation.
func TestConcurrentDocumentCreation(t *testing.T) {
	const numGoroutines = 50
	const numDocumentsPerGoroutine = 20

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	errors := make(chan error, numGoroutines*numDocumentsPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numDocumentsPerGoroutine; j++ {
				doc := NewDocument("Test content", map[string]string{"id": "test"})
				if doc.GetContent() != "Test content" {
					errors <- assert.AnError
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		require.NoError(t, err)
	}
}

// TestMessageValidationAdvanced tests message validation scenarios.
func TestMessageValidationAdvanced(t *testing.T) {
	tests := []struct {
		name        string
		description string
		message     Message
		validate    func(t *testing.T, msg Message)
	}{
		{
			name:        "valid_human_message",
			description: "Validate human message structure",
			message:     NewHumanMessage("Valid content"),
			validate: func(t *testing.T, msg Message) {
				assert.Equal(t, iface.RoleHuman, msg.GetType())
				assert.NotEmpty(t, msg.GetContent())
			},
		},
		{
			name:        "valid_ai_message",
			description: "Validate AI message structure",
			message:     NewAIMessage("Valid AI content"),
			validate: func(t *testing.T, msg Message) {
				assert.Equal(t, iface.RoleAssistant, msg.GetType())
				assert.NotEmpty(t, msg.GetContent())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)
			tt.validate(t, tt.message)
		})
	}
}

// BenchmarkMessageCreation benchmarks message creation performance.
func BenchmarkMessageCreation(b *testing.B) {
	b.Run("human_message", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = NewHumanMessage("Benchmark content")
		}
	})

	b.Run("ai_message", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = NewAIMessage("Benchmark content")
		}
	})

	b.Run("document", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = NewDocument("Benchmark content", map[string]string{"key": "value"})
		}
	})
}

// BenchmarkConcurrentMessageCreation benchmarks concurrent message creation.
func BenchmarkConcurrentMessageCreation(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = NewHumanMessage("Concurrent benchmark content")
		}
	})
}
