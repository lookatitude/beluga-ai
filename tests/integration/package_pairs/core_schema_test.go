// Package package_pairs provides integration tests between Core and Schema packages.
// This test suite verifies that Core Runnable components can work with Schema types
// (Document, Message) and that Schema factory functions produce types compatible
// with Core interfaces.
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

// TestIntegrationCoreSchema tests the integration between Core and Schema packages.
func TestIntegrationCoreSchema(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	t.Run("core_runnable_with_schema_messages", func(t *testing.T) {
		// Create a runnable that processes schema messages
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
	})

	t.Run("core_traced_runnable_with_schema_messages", func(t *testing.T) {
		// Test that TracedRunnable works with schema messages
		msg := schema.NewHumanMessage("test message")
		runnable := &testRunnable{
			invokeFunc: func(ctx context.Context, input any, options ...core.Option) (any, error) {
				assert.Equal(t, msg, input)
				return "result", nil
			},
		}

		traced := core.NewTracedRunnable(
			runnable,
			nil,
			core.NoOpMetrics(),
			"test-component",
			"",
		)

		result, err := traced.Invoke(context.Background(), msg, core.WithOption("key", "value"))
		require.NoError(t, err)
		assert.Equal(t, "result", result)
	})

	t.Run("core_runnable_batch_with_schema_messages", func(t *testing.T) {
		// Test batch processing with schema messages
		messages := []any{
			schema.NewHumanMessage("Message 1"),
			schema.NewAIMessage("Message 2"),
		}
		runnable := &testRunnable{
			batchFunc: func(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
				assert.Equal(t, messages, inputs)
				return []any{"result1", "result2"}, nil
			},
		}

		traced := core.NewTracedRunnable(
			runnable,
			nil,
			core.NoOpMetrics(),
			"test-component",
			"",
		)

		results, err := traced.Batch(context.Background(), messages, core.WithOption("key", "value"))
		require.NoError(t, err)
		assert.Len(t, results, 2)
	})

	t.Run("core_di_container_with_schema_types", func(t *testing.T) {
		// Test that DI container can resolve schema types
		container := core.NewContainer()
		err := container.Register(func() schema.Message {
			return schema.NewHumanMessage("factory message")
		})
		require.NoError(t, err)

		var msg schema.Message
		err = container.Resolve(&msg)
		require.NoError(t, err)
		assert.Equal(t, "factory message", msg.GetContent())
	})
}
