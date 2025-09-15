package providers

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBaseChatMessageHistory tests the base chat message history implementation.
func TestBaseChatMessageHistory(t *testing.T) {
	ctx := context.Background()

	t.Run("NewBaseChatMessageHistory", func(t *testing.T) {
		history := NewBaseChatMessageHistory()
		assert.NotNil(t, history)
		assert.IsType(t, &BaseChatMessageHistory{}, history)
	})

	t.Run("NewBaseChatMessageHistoryWithOptions", func(t *testing.T) {
		history := NewBaseChatMessageHistory(WithMaxHistorySize(5))
		assert.NotNil(t, history)
		assert.Equal(t, 5, history.maxSize)
	})

	t.Run("AddAndGetMessages", func(t *testing.T) {
		history := NewBaseChatMessageHistory()

		// Add messages
		err := history.AddUserMessage(ctx, "Hello")
		assert.NoError(t, err)

		err = history.AddAIMessage(ctx, "Hi there!")
		assert.NoError(t, err)

		// Get messages
		messages, err := history.GetMessages(ctx)
		assert.NoError(t, err)
		assert.Len(t, messages, 2)
		assert.Equal(t, "Hello", messages[0].GetContent())
		assert.Equal(t, "Hi there!", messages[1].GetContent())
	})

	t.Run("MaxSizeLimit", func(t *testing.T) {
		history := NewBaseChatMessageHistory(WithMaxHistorySize(3))

		// Add more messages than the limit
		for i := 0; i < 5; i++ {
			err := history.AddUserMessage(ctx, "Message "+string(rune(i+'0')))
			assert.NoError(t, err)
		}

		// Should only keep the last 3 messages
		messages, err := history.GetMessages(ctx)
		assert.NoError(t, err)
		assert.Len(t, messages, 3)
		assert.Equal(t, "Message 2", messages[0].GetContent())
		assert.Equal(t, "Message 3", messages[1].GetContent())
		assert.Equal(t, "Message 4", messages[2].GetContent())
	})

	t.Run("Clear", func(t *testing.T) {
		history := NewBaseChatMessageHistory()

		// Add messages
		err := history.AddUserMessage(ctx, "Hello")
		assert.NoError(t, err)

		// Verify messages exist
		messages, err := history.GetMessages(ctx)
		assert.NoError(t, err)
		assert.Len(t, messages, 1)

		// Clear messages
		err = history.Clear(ctx)
		assert.NoError(t, err)

		// Verify messages are cleared
		messages, err = history.GetMessages(ctx)
		assert.NoError(t, err)
		assert.Len(t, messages, 0)
	})

	t.Run("ContextCancellation", func(t *testing.T) {
		history := NewBaseChatMessageHistory()

		// Create a cancelled context
		cancelledCtx, cancel := context.WithCancel(ctx)
		cancel()

		// Test AddMessage with cancelled context
		err := history.AddMessage(cancelledCtx, nil)
		assert.Error(t, err)

		// Test GetMessages with cancelled context
		_, err = history.GetMessages(cancelledCtx)
		assert.Error(t, err)

		// Test Clear with cancelled context
		err = history.Clear(cancelledCtx)
		assert.Error(t, err)
	})
}

// TestCompositeChatMessageHistory tests the composite chat message history implementation.
func TestCompositeChatMessageHistory(t *testing.T) {
	ctx := context.Background()

	t.Run("NewCompositeChatMessageHistory", func(t *testing.T) {
		primary := NewBaseChatMessageHistory()
		composite := NewCompositeChatMessageHistory(primary)
		assert.NotNil(t, composite)
		assert.IsType(t, &CompositeChatMessageHistory{}, composite)
	})

	t.Run("WithSecondaryHistory", func(t *testing.T) {
		primary := NewBaseChatMessageHistory()
		secondary := NewBaseChatMessageHistory()
		composite := NewCompositeChatMessageHistory(primary, WithSecondaryHistory(secondary))
		assert.NotNil(t, composite.secondary)
	})

	t.Run("WithMaxSize", func(t *testing.T) {
		primary := NewBaseChatMessageHistory()
		composite := NewCompositeChatMessageHistory(primary, WithMaxSize(10))
		assert.Equal(t, 10, composite.maxSize)
	})

	t.Run("AddAndGetMessages", func(t *testing.T) {
		primary := NewBaseChatMessageHistory()
		composite := NewCompositeChatMessageHistory(primary)

		// Add messages
		err := composite.AddUserMessage(ctx, "Hello")
		assert.NoError(t, err)

		err = composite.AddAIMessage(ctx, "Hi there!")
		assert.NoError(t, err)

		// Get messages
		messages, err := composite.GetMessages(ctx)
		assert.NoError(t, err)
		assert.Len(t, messages, 2)
	})

	t.Run("SecondaryHistory", func(t *testing.T) {
		primary := NewBaseChatMessageHistory()
		secondary := NewBaseChatMessageHistory()
		composite := NewCompositeChatMessageHistory(primary, WithSecondaryHistory(secondary))

		// Add message (should be added to both)
		err := composite.AddUserMessage(ctx, "Test message")
		assert.NoError(t, err)

		// Verify both histories have the message
		primaryMessages, err := primary.GetMessages(ctx)
		assert.NoError(t, err)
		assert.Len(t, primaryMessages, 1)

		secondaryMessages, err := secondary.GetMessages(ctx)
		assert.NoError(t, err)
		assert.Len(t, secondaryMessages, 1)
	})

	t.Run("Hooks", func(t *testing.T) {
		primary := NewBaseChatMessageHistory()
		var addHookCalled, getHookCalled bool

		composite := NewCompositeChatMessageHistory(primary,
			WithOnAddHook(func(ctx context.Context, msg schema.Message) error {
				addHookCalled = true
				return nil
			}),
			WithOnGetHook(func(ctx context.Context, msgs []schema.Message) ([]schema.Message, error) {
				getHookCalled = true
				return msgs, nil
			}),
		)

		// Add message (should trigger add hook)
		err := composite.AddUserMessage(ctx, "Test")
		assert.NoError(t, err)
		assert.True(t, addHookCalled)

		// Get messages (should trigger get hook)
		_, err = composite.GetMessages(ctx)
		assert.NoError(t, err)
		assert.True(t, getHookCalled)
	})

	t.Run("ClearComposite", func(t *testing.T) {
		primary := NewBaseChatMessageHistory()
		secondary := NewBaseChatMessageHistory()
		composite := NewCompositeChatMessageHistory(primary, WithSecondaryHistory(secondary))

		// Add messages to both
		err := composite.AddUserMessage(ctx, "Test")
		assert.NoError(t, err)

		// Clear composite (should clear both)
		err = composite.Clear(ctx)
		assert.NoError(t, err)

		// Verify both are cleared
		primaryMessages, err := primary.GetMessages(ctx)
		assert.NoError(t, err)
		assert.Len(t, primaryMessages, 0)

		secondaryMessages, err := secondary.GetMessages(ctx)
		assert.NoError(t, err)
		assert.Len(t, secondaryMessages, 0)
	})
}

// Benchmark tests for performance measurement

func BenchmarkBaseChatMessageHistory_AddMessage(b *testing.B) {
	history := NewBaseChatMessageHistory()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		history.AddUserMessage(ctx, "Benchmark message")
	}
}

func BenchmarkBaseChatMessageHistory_GetMessages(b *testing.B) {
	history := NewBaseChatMessageHistory()
	ctx := context.Background()

	// Add some messages first
	for i := 0; i < 100; i++ {
		history.AddUserMessage(ctx, "Message "+string(rune(i)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		history.GetMessages(ctx)
	}
}

func BenchmarkCompositeChatMessageHistory_AddMessage(b *testing.B) {
	primary := NewBaseChatMessageHistory()
	composite := NewCompositeChatMessageHistory(primary)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		composite.AddUserMessage(ctx, "Benchmark message")
	}
}

func BenchmarkCompositeChatMessageHistory_GetMessages(b *testing.B) {
	primary := NewBaseChatMessageHistory()
	composite := NewCompositeChatMessageHistory(primary)
	ctx := context.Background()

	// Add some messages first
	for i := 0; i < 100; i++ {
		composite.AddUserMessage(ctx, "Message "+string(rune(i)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		composite.GetMessages(ctx)
	}
}

// Table-driven tests for comprehensive coverage

func TestBaseChatMessageHistory_TableDriven(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name           string
		setup          func() *BaseChatMessageHistory
		action         func(*BaseChatMessageHistory) error
		expectedError  bool
		validateResult func(*BaseChatMessageHistory, *testing.T)
	}{
		{
			name: "AddUserMessage_Success",
			setup: func() *BaseChatMessageHistory {
				return NewBaseChatMessageHistory()
			},
			action: func(h *BaseChatMessageHistory) error {
				return h.AddUserMessage(ctx, "Test message")
			},
			expectedError: false,
			validateResult: func(h *BaseChatMessageHistory, t *testing.T) {
				messages, err := h.GetMessages(ctx)
				require.NoError(t, err)
				assert.Len(t, messages, 1)
				assert.Equal(t, "Test message", messages[0].GetContent())
			},
		},
		{
			name: "AddAIMessage_Success",
			setup: func() *BaseChatMessageHistory {
				return NewBaseChatMessageHistory()
			},
			action: func(h *BaseChatMessageHistory) error {
				return h.AddAIMessage(ctx, "AI response")
			},
			expectedError: false,
			validateResult: func(h *BaseChatMessageHistory, t *testing.T) {
				messages, err := h.GetMessages(ctx)
				require.NoError(t, err)
				assert.Len(t, messages, 1)
				assert.Equal(t, "AI response", messages[0].GetContent())
			},
		},
		{
			name: "Clear_EmptyHistory",
			setup: func() *BaseChatMessageHistory {
				return NewBaseChatMessageHistory()
			},
			action: func(h *BaseChatMessageHistory) error {
				return h.Clear(ctx)
			},
			expectedError: false,
			validateResult: func(h *BaseChatMessageHistory, t *testing.T) {
				messages, err := h.GetMessages(ctx)
				require.NoError(t, err)
				assert.Len(t, messages, 0)
			},
		},
		{
			name: "MaxSize_Enforced",
			setup: func() *BaseChatMessageHistory {
				return NewBaseChatMessageHistory(WithMaxHistorySize(2))
			},
			action: func(h *BaseChatMessageHistory) error {
				for i := 0; i < 3; i++ {
					if err := h.AddUserMessage(ctx, "Message "+string(rune(i+'0'))); err != nil {
						return err
					}
				}
				return nil
			},
			expectedError: false,
			validateResult: func(h *BaseChatMessageHistory, t *testing.T) {
				messages, err := h.GetMessages(ctx)
				require.NoError(t, err)
				assert.Len(t, messages, 2) // Should only keep last 2
				assert.Equal(t, "Message 1", messages[0].GetContent())
				assert.Equal(t, "Message 2", messages[1].GetContent())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			history := tc.setup()
			err := tc.action(history)

			if tc.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tc.validateResult != nil {
				tc.validateResult(history, t)
			}
		})
	}
}
