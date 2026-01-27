package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewBuffering(t *testing.T) {
	b := NewBuffering(BufferingStrategyFixed, 1000)
	assert.NotNil(t, b)
	assert.Equal(t, BufferingStrategyFixed, b.strategy)
	assert.Equal(t, 1000, b.maxSize)
	assert.Equal(t, 0, b.GetSize())
}

func TestBuffering_Add_FixedStrategy(t *testing.T) {
	b := NewBuffering(BufferingStrategyFixed, 100)

	// Add data within limit
	result := b.Add([]byte{1, 2, 3, 4, 5})
	assert.True(t, result)
	assert.Equal(t, 5, b.GetSize())

	// Add more data
	result = b.Add([]byte{6, 7, 8, 9, 10})
	assert.True(t, result)
	assert.Equal(t, 10, b.GetSize())

	// Try to add beyond limit
	result = b.Add(make([]byte, 100))
	assert.False(t, result)
	assert.Equal(t, 10, b.GetSize()) // Size unchanged
}

func TestBuffering_Add_NoneStrategy(t *testing.T) {
	b := NewBuffering(BufferingStrategyNone, 100)
	result := b.Add([]byte{1, 2, 3})
	assert.False(t, result)
	assert.Equal(t, 0, b.GetSize())
}

func TestBuffering_Add_AdaptiveStrategy(t *testing.T) {
	b := NewBuffering(BufferingStrategyAdaptive, 100)

	// Add data within limit
	result := b.Add([]byte{1, 2, 3, 4, 5})
	assert.True(t, result)
	assert.Equal(t, 5, b.GetSize())

	// Try to add beyond available space
	result = b.Add(make([]byte, 100))
	assert.False(t, result)
}

func TestBuffering_Flush(t *testing.T) {
	b := NewBuffering(BufferingStrategyFixed, 100)
	data := []byte{1, 2, 3, 4, 5}
	b.Add(data)

	flushed := b.Flush()
	assert.Equal(t, data, flushed)
	assert.Equal(t, 0, b.GetSize())
}

func TestBuffering_GetSize(t *testing.T) {
	b := NewBuffering(BufferingStrategyFixed, 100)
	assert.Equal(t, 0, b.GetSize())

	b.Add([]byte{1, 2, 3})
	assert.Equal(t, 3, b.GetSize())

	b.Add([]byte{4, 5})
	assert.Equal(t, 5, b.GetSize())
}

func TestBuffering_IsFull(t *testing.T) {
	b := NewBuffering(BufferingStrategyFixed, 10)
	assert.False(t, b.IsFull())

	b.Add([]byte{1, 2, 3, 4, 5})
	assert.False(t, b.IsFull())

	b.Add([]byte{6, 7, 8, 9, 10})
	assert.True(t, b.IsFull())
}

func TestBuffering_Clear(t *testing.T) {
	b := NewBuffering(BufferingStrategyFixed, 100)
	b.Add([]byte{1, 2, 3, 4, 5})
	assert.Equal(t, 5, b.GetSize())

	b.Clear()
	assert.Equal(t, 0, b.GetSize())
	assert.False(t, b.IsFull())
}
