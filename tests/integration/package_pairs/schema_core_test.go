// Package package_pairs provides integration tests between Schema and Core packages.
// This test suite verifies that Core components can work with Schema types (Document, Message)
// and that Schema factory functions produce types compatible with Core interfaces.
package package_pairs

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/tests/integration/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegrationSchemaCore tests the integration between Schema and Core packages.
func TestIntegrationSchemaCore(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	t.Run("core_runnable_with_schema_messages", func(t *testing.T) {
		// Create a simple runnable that processes schema messages
		runnable := &testRunnable{
			invokeFunc: func(ctx context.Context, input any, options ...core.Option) (any, error) {
				// Verify input is a schema.Message
				msg, ok := input.(schema.Message)
				require.True(t, ok, "Input should be a schema.Message")
				assert.NotEmpty(t, msg.GetContent())
				assert.NotEmpty(t, msg.GetType())
				return msg.GetContent(), nil
			},
		}

		// Test with different schema message types
		humanMsg := schema.NewHumanMessage("Hello, world!")
		result, err := runnable.Invoke(context.Background(), humanMsg)
		require.NoError(t, err)
		assert.Equal(t, "Hello, world!", result)

		aiMsg := schema.NewAIMessage("AI response")
		result, err = runnable.Invoke(context.Background(), aiMsg)
		require.NoError(t, err)
		assert.Equal(t, "AI response", result)

		systemMsg := schema.NewSystemMessage("System prompt")
		result, err = runnable.Invoke(context.Background(), systemMsg)
		require.NoError(t, err)
		assert.Equal(t, "System prompt", result)
	})

	t.Run("core_runnable_with_schema_documents", func(t *testing.T) {
		// Create a runnable that processes schema documents
		runnable := &testRunnable{
			invokeFunc: func(ctx context.Context, input any, options ...core.Option) (any, error) {
				// Verify input is a schema.Document
				doc, ok := input.(schema.Document)
				require.True(t, ok, "Input should be a schema.Document")
				assert.NotEmpty(t, doc.GetContent())
				return doc.GetContent(), nil
			},
		}

		// Test with different document types
		doc := schema.NewDocument("Document content", map[string]string{"key": "value"})
		result, err := runnable.Invoke(context.Background(), doc)
		require.NoError(t, err)
		assert.Equal(t, "Document content", result)

		docWithID := schema.NewDocumentWithID("doc_123", "Content with ID", map[string]string{"id": "doc_123"})
		result, err = runnable.Invoke(context.Background(), docWithID)
		require.NoError(t, err)
		assert.Equal(t, "Content with ID", result)

		embedding := []float32{0.1, 0.2, 0.3}
		docWithEmbedding := schema.NewDocumentWithEmbedding("Content with embedding", map[string]string{"key": "value"}, embedding)
		result, err = runnable.Invoke(context.Background(), docWithEmbedding)
		require.NoError(t, err)
		assert.Equal(t, "Content with embedding", result)
	})

	t.Run("core_runnable_batch_with_schema_messages", func(t *testing.T) {
		// Create a runnable that processes batches of schema messages
		runnable := &testRunnable{
			batchFunc: func(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
				results := make([]any, len(inputs))
				for i, input := range inputs {
					msg, ok := input.(schema.Message)
					require.True(t, ok, "Input %d should be a schema.Message", i)
					results[i] = msg.GetContent()
				}
				return results, nil
			},
		}

		// Test batch processing with schema messages
		messages := []any{
			schema.NewHumanMessage("Message 1"),
			schema.NewAIMessage("Message 2"),
			schema.NewSystemMessage("Message 3"),
		}

		results, err := runnable.Batch(context.Background(), messages)
		require.NoError(t, err)
		require.Len(t, results, 3)
		assert.Equal(t, "Message 1", results[0])
		assert.Equal(t, "Message 2", results[1])
		assert.Equal(t, "Message 3", results[2])
	})

	t.Run("core_runnable_stream_with_schema_messages", func(t *testing.T) {
		// Create a runnable that streams schema messages
		runnable := &testRunnable{
			streamFunc: func(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
				msg, ok := input.(schema.Message)
				require.True(t, ok, "Input should be a schema.Message")

				ch := make(chan any, 1)
				go func() {
					defer close(ch)
					// Stream the message content as chunks
					content := msg.GetContent()
					if len(content) > 0 {
						ch <- content[:min(len(content), 5)] // First 5 chars
						if len(content) > 5 {
							ch <- content[5:] // Rest
						}
					}
				}()
				return ch, nil
			},
		}

		// Test streaming with schema message
		msg := schema.NewHumanMessage("Hello, world!")
		ch, err := runnable.Stream(context.Background(), msg)
		require.NoError(t, err)

		var results []any
		for result := range ch {
			results = append(results, result)
		}
		assert.Greater(t, len(results), 0)
	})

	t.Run("schema_document_as_core_document_loader_output", func(t *testing.T) {
		// Test that schema.Document can be used as output from core.DocumentLoader
		loader := &testDocumentLoader{
			loadFunc: func(ctx context.Context) ([]schema.Document, error) {
				return []schema.Document{
					schema.NewDocument("Document 1", map[string]string{"source": "test1"}),
					schema.NewDocument("Document 2", map[string]string{"source": "test2"}),
					schema.NewDocumentWithID("doc_3", "Document 3", map[string]string{"source": "test3"}),
				}, nil
			},
		}

		docs, err := loader.Load(context.Background())
		require.NoError(t, err)
		require.Len(t, docs, 3)
		assert.Equal(t, "Document 1", docs[0].GetContent())
		assert.Equal(t, "Document 2", docs[1].GetContent())
		assert.Equal(t, "Document 3", docs[2].GetContent())
	})

	t.Run("schema_message_validation_with_core_runnable", func(t *testing.T) {
		// Test that schema message validation works with core runnable
		runnable := &testRunnable{
			invokeFunc: func(ctx context.Context, input any, options ...core.Option) (any, error) {
				msg, ok := input.(schema.Message)
				require.True(t, ok)

				// Validate message using schema validation
				err := schema.ValidateMessage(msg)
				if err != nil {
					return nil, err
				}
				return msg.GetContent(), nil
			},
		}

		// Valid message should work
		validMsg := schema.NewHumanMessage("Valid content")
		result, err := runnable.Invoke(context.Background(), validMsg)
		require.NoError(t, err)
		assert.Equal(t, "Valid content", result)

		// Empty content message should fail validation
		emptyMsg := schema.NewHumanMessage("")
		_, err = runnable.Invoke(context.Background(), emptyMsg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
	})

	t.Run("schema_document_validation_with_core_runnable", func(t *testing.T) {
		// Test that schema document validation works with core runnable
		runnable := &testRunnable{
			invokeFunc: func(ctx context.Context, input any, options ...core.Option) (any, error) {
				doc, ok := input.(schema.Document)
				require.True(t, ok)

				// Validate document using schema validation
				err := schema.ValidateDocument(doc)
				if err != nil {
					return nil, err
				}
				return doc.GetContent(), nil
			},
		}

		// Valid document should work
		validDoc := schema.NewDocument("Valid content", map[string]string{"key": "value"})
		result, err := runnable.Invoke(context.Background(), validDoc)
		require.NoError(t, err)
		assert.Equal(t, "Valid content", result)

		// Empty content document should fail validation
		emptyDoc := schema.NewDocument("", map[string]string{"key": "value"})
		_, err = runnable.Invoke(context.Background(), emptyDoc)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
	})
}

// testRunnable is a simple implementation of core.Runnable for testing.
type testRunnable struct {
	invokeFunc func(ctx context.Context, input any, options ...core.Option) (any, error)
	batchFunc  func(ctx context.Context, inputs []any, options ...core.Option) ([]any, error)
	streamFunc func(ctx context.Context, input any, options ...core.Option) (<-chan any, error)
}

func (r *testRunnable) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	if r.invokeFunc != nil {
		return r.invokeFunc(ctx, input, options...)
	}
	return input, nil
}

func (r *testRunnable) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
	if r.batchFunc != nil {
		return r.batchFunc(ctx, inputs, options...)
	}
	results := make([]any, len(inputs))
	for i, input := range inputs {
		results[i] = input
	}
	return results, nil
}

func (r *testRunnable) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
	if r.streamFunc != nil {
		return r.streamFunc(ctx, input, options...)
	}
	ch := make(chan any, 1)
	ch <- input
	close(ch)
	return ch, nil
}

// testDocumentLoader is a simple implementation of core.DocumentLoader for testing.
type testDocumentLoader struct {
	loadFunc func(ctx context.Context) ([]schema.Document, error)
}

func (l *testDocumentLoader) Load(ctx context.Context) ([]schema.Document, error) {
	if l.loadFunc != nil {
		return l.loadFunc(ctx)
	}
	return []schema.Document{}, nil
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
