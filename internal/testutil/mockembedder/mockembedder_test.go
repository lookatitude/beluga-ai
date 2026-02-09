package mockembedder

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name       string
		opts       []Option
		wantDim    int
		wantEmbs   int
		wantErr    error
	}{
		{
			name:    "default configuration",
			wantDim: 384,
		},
		{
			name:    "custom dimensions",
			opts:    []Option{WithDimensions(768)},
			wantDim: 768,
		},
		{
			name:     "with embeddings",
			opts:     []Option{WithEmbeddings([][]float32{{1, 2}, {3, 4}})},
			wantDim:  384,
			wantEmbs: 2,
		},
		{
			name:    "with error",
			opts:    []Option{WithError(errors.New("test error"))},
			wantDim: 384,
			wantErr: errors.New("test error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New(tt.opts...)
			require.NotNil(t, m)
			assert.Equal(t, tt.wantDim, m.Dimensions())
			if tt.wantErr != nil {
				assert.NotNil(t, m.err)
			}
		})
	}
}

func TestMockEmbedder_Embed(t *testing.T) {
	tests := []struct {
		name      string
		opts      []Option
		texts     []string
		wantLen   int
		wantDim   int
		wantErr   bool
		wantCalls int
	}{
		{
			name:      "default zero vectors",
			texts:     []string{"hello", "world"},
			wantLen:   2,
			wantDim:   384,
			wantCalls: 1,
		},
		{
			name:      "custom dimensions",
			opts:      []Option{WithDimensions(128)},
			texts:     []string{"test"},
			wantLen:   1,
			wantDim:   128,
			wantCalls: 1,
		},
		{
			name: "preset embeddings single",
			opts: []Option{WithEmbeddings([][]float32{{1.0, 2.0, 3.0}})},
			texts: []string{"hello"},
			wantLen: 1,
			wantDim: 3,
			wantCalls: 1,
		},
		{
			name: "preset embeddings repeat",
			opts: []Option{WithEmbeddings([][]float32{{1.0, 2.0}, {3.0, 4.0}})},
			texts: []string{"a", "b", "c"},
			wantLen: 3,
			wantDim: 2,
			wantCalls: 1,
		},
		{
			name:      "error path",
			opts:      []Option{WithError(errors.New("embed failed"))},
			texts:     []string{"test"},
			wantErr:   true,
			wantCalls: 1,
		},
		{
			name:      "empty texts",
			texts:     []string{},
			wantLen:   0,
			wantCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New(tt.opts...)
			ctx := context.Background()

			result, err := m.Embed(ctx, tt.texts)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, result, tt.wantLen)
			if tt.wantLen > 0 {
				assert.Len(t, result[0], tt.wantDim)
			}
			assert.Equal(t, tt.wantCalls, m.EmbedCalls())
			assert.Equal(t, tt.texts, m.LastTexts())
		})
	}
}

func TestMockEmbedder_EmbedSingle(t *testing.T) {
	tests := []struct {
		name    string
		opts    []Option
		text    string
		wantDim int
		wantErr bool
	}{
		{
			name:    "default zero vector",
			text:    "hello",
			wantDim: 384,
		},
		{
			name:    "custom dimensions",
			opts:    []Option{WithDimensions(512)},
			text:    "test",
			wantDim: 512,
		},
		{
			name: "preset embedding",
			opts: []Option{WithEmbeddings([][]float32{{1.0, 2.0, 3.0}})},
			text: "hello",
			wantDim: 3,
		},
		{
			name:    "error path",
			opts:    []Option{WithError(errors.New("embed failed"))},
			text:    "test",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New(tt.opts...)
			ctx := context.Background()

			result, err := m.EmbedSingle(ctx, tt.text)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, result, tt.wantDim)
			assert.Equal(t, 1, m.EmbedCalls())
			assert.Equal(t, []string{tt.text}, m.LastTexts())
		})
	}
}

func TestMockEmbedder_WithEmbedFunc(t *testing.T) {
	called := false
	customFn := func(ctx context.Context, texts []string) ([][]float32, error) {
		called = true
		return [][]float32{{9.9, 8.8}}, nil
	}

	m := New(WithEmbedFunc(customFn))
	result, err := m.Embed(context.Background(), []string{"test"})

	require.NoError(t, err)
	assert.True(t, called)
	assert.Equal(t, [][]float32{{9.9, 8.8}}, result)
}

func TestMockEmbedder_SetEmbeddings(t *testing.T) {
	m := New()
	ctx := context.Background()

	// Initial call returns zero vectors
	result1, err := m.Embed(ctx, []string{"test"})
	require.NoError(t, err)
	assert.Equal(t, []float32{0, 0, 0, 0}, result1[0][:4])

	// Update embeddings
	m.SetEmbeddings([][]float32{{1.0, 2.0}})
	result2, err := m.Embed(ctx, []string{"test"})
	require.NoError(t, err)
	assert.Equal(t, []float32{1.0, 2.0}, result2[0])
}

func TestMockEmbedder_SetError(t *testing.T) {
	m := New()
	ctx := context.Background()

	// Initial call succeeds
	_, err := m.Embed(ctx, []string{"test"})
	require.NoError(t, err)

	// Set error
	testErr := errors.New("new error")
	m.SetError(testErr)
	_, err = m.Embed(ctx, []string{"test"})
	require.Error(t, err)
	assert.Equal(t, testErr, err)
}

func TestMockEmbedder_Reset(t *testing.T) {
	m := New(
		WithDimensions(512),
		WithEmbeddings([][]float32{{1, 2}}),
		WithError(errors.New("test")),
	)

	// Make some calls
	_, _ = m.Embed(context.Background(), []string{"a", "b"})
	assert.Equal(t, 1, m.EmbedCalls())
	assert.Equal(t, []string{"a", "b"}, m.LastTexts())

	// Reset
	m.Reset()
	assert.Equal(t, 0, m.EmbedCalls())
	assert.Empty(t, m.LastTexts())
	assert.Equal(t, 384, m.Dimensions()) // back to default
	assert.Nil(t, m.err)
	assert.Nil(t, m.embeddings)
}

func TestMockEmbedder_Concurrency(t *testing.T) {
	m := New(WithDimensions(128))
	ctx := context.Background()

	// Multiple goroutines calling Embed concurrently
	const goroutines = 10
	done := make(chan bool, goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			_, err := m.Embed(ctx, []string{"concurrent"})
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < goroutines; i++ {
		<-done
	}

	assert.Equal(t, goroutines, m.EmbedCalls())
}

func TestMockEmbedder_ContextCancellation(t *testing.T) {
	m := New()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// The mock doesn't check context by default, but it should still work
	_, err := m.Embed(ctx, []string{"test"})
	require.NoError(t, err) // Mock doesn't enforce context cancellation
}

func TestMockEmbedder_PresetEmbeddingsRepeat(t *testing.T) {
	// Test that preset embeddings repeat when there are more texts than embeddings
	m := New(WithEmbeddings([][]float32{
		{1.0, 2.0},
		{3.0, 4.0},
	}))

	result, err := m.Embed(context.Background(), []string{"a", "b", "c", "d", "e"})
	require.NoError(t, err)
	assert.Len(t, result, 5)

	// Check pattern repeats
	assert.Equal(t, []float32{1.0, 2.0}, result[0])
	assert.Equal(t, []float32{3.0, 4.0}, result[1])
	assert.Equal(t, []float32{1.0, 2.0}, result[2])
	assert.Equal(t, []float32{3.0, 4.0}, result[3])
	assert.Equal(t, []float32{1.0, 2.0}, result[4])
}
