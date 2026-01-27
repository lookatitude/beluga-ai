package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewMemoryIntegration(t *testing.T) {
	mi := NewMemoryIntegration("test-session-123")
	assert.NotNil(t, mi)
	assert.Equal(t, "test-session-123", mi.sessionID)
	assert.NotNil(t, mi.context)
}

func TestMemoryIntegration_Store(t *testing.T) {
	mi := NewMemoryIntegration("test-session")

	mi.Store("key1", "value1")
	mi.Store("key2", 42)
	mi.Store("key3", map[string]any{"nested": "value"})

	// Verify stored values
	value, exists := mi.Retrieve("key1")
	assert.True(t, exists)
	assert.Equal(t, "value1", value)

	value, exists = mi.Retrieve("key2")
	assert.True(t, exists)
	assert.Equal(t, 42, value)
}

func TestMemoryIntegration_Retrieve(t *testing.T) {
	mi := NewMemoryIntegration("test-session")

	// Retrieve non-existent key
	_, exists := mi.Retrieve("nonexistent")
	assert.False(t, exists)

	// Store and retrieve
	mi.Store("test-key", "test-value")
	value, exists := mi.Retrieve("test-key")
	assert.True(t, exists)
	assert.Equal(t, "test-value", value)
}

func TestMemoryIntegration_Clear(t *testing.T) {
	mi := NewMemoryIntegration("test-session")

	// Store some values
	mi.Store("key1", "value1")
	mi.Store("key2", "value2")

	// Verify they exist
	_, exists := mi.Retrieve("key1")
	assert.True(t, exists)

	// Clear
	mi.Clear()

	// Verify they're gone
	_, exists = mi.Retrieve("key1")
	assert.False(t, exists)
	_, exists = mi.Retrieve("key2")
	assert.False(t, exists)
}

func TestMemoryIntegration_GetContext(t *testing.T) {
	mi := NewMemoryIntegration("test-session")

	// Store multiple values
	mi.Store("key1", "value1")
	mi.Store("key2", 42)
	mi.Store("key3", true)

	context := mi.GetContext()
	assert.NotNil(t, context)
	assert.Len(t, context, 3)
	assert.Equal(t, "value1", context["key1"])
	assert.Equal(t, 42, context["key2"])
	assert.Equal(t, true, context["key3"])
}

func TestMemoryIntegration_GetContext_Empty(t *testing.T) {
	mi := NewMemoryIntegration("test-session")

	context := mi.GetContext()
	assert.NotNil(t, context)
	assert.Empty(t, context)
}

func TestMemoryIntegration_ConcurrentAccess(t *testing.T) {
	mi := NewMemoryIntegration("test-session")

	// Test concurrent stores
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			mi.Store("key", idx)
			done <- true
		}(i)
	}

	// Wait for all stores
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have a value (last one written)
	value, exists := mi.Retrieve("key")
	assert.True(t, exists)
	assert.NotNil(t, value)
}

func TestMemoryIntegration_Overwrite(t *testing.T) {
	mi := NewMemoryIntegration("test-session")

	mi.Store("key", "value1")
	value, _ := mi.Retrieve("key")
	assert.Equal(t, "value1", value)

	mi.Store("key", "value2")
	value, _ = mi.Retrieve("key")
	assert.Equal(t, "value2", value)
}
