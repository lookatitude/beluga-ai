// Package multimodal provides unit tests for multimodal operations.
package multimodal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewContentBlock(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		data        []byte
		wantErr     bool
		errContains string
	}{
		{
			name:        "valid text block",
			contentType: "text",
			data:        []byte("Hello, world!"),
			wantErr:     false,
		},
		{
			name:        "valid image block",
			contentType: "image",
			data:        []byte{0x89, 0x50, 0x4E, 0x47}, // PNG header
			wantErr:     false,
		},
		{
			name:        "empty content type",
			contentType: "",
			data:        []byte("test"),
			wantErr:     true,
			errContains: "content type cannot be empty",
		},
		{
			name:        "invalid content type",
			contentType: "invalid",
			data:        []byte("test"),
			wantErr:     true,
			errContains: "invalid content type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			block, err := NewContentBlock(tt.contentType, tt.data)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.Nil(t, block)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, block)
				assert.Equal(t, tt.contentType, block.Type)
				assert.Equal(t, tt.data, block.Data)
				assert.Equal(t, int64(len(tt.data)), block.Size)
			}
		})
	}
}

func TestContentBlock_Validate(t *testing.T) {
	tests := []struct {
		name        string
		block       *ContentBlock
		wantErr     bool
		errContains string
	}{
		{
			name: "valid text block with data",
			block: &ContentBlock{
				Type: "text",
				Data: []byte("Hello"),
				Size: 5,
			},
			wantErr: false,
		},
		{
			name: "valid image block with URL",
			block: &ContentBlock{
				Type: "image",
				URL:  "https://example.com/image.png",
				Size: 100,
			},
			wantErr: false,
		},
		{
			name: "valid audio block with file path",
			block: &ContentBlock{
				Type:     "audio",
				FilePath: "/path/to/audio.mp3",
				Size:     1000,
			},
			wantErr: false,
		},
		{
			name: "invalid content type",
			block: &ContentBlock{
				Type: "invalid",
				Data: []byte("test"),
			},
			wantErr:     true,
			errContains: "invalid content type",
		},
		{
			name: "no data source",
			block: &ContentBlock{
				Type: "text",
			},
			wantErr:     true,
			errContains: "must have at least one of",
		},
		{
			name: "negative size",
			block: &ContentBlock{
				Type: "text",
				Data: []byte("test"),
				Size: -1,
			},
			wantErr:     true,
			errContains: "size must be >= 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.block.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestNewMultimodalInput(t *testing.T) {
	tests := []struct {
		name        string
		blocks      []*ContentBlock
		opts        []MultimodalInputOption
		wantErr     bool
		errContains string
	}{
		{
			name: "valid input with one block",
			blocks: []*ContentBlock{
				{
					Type: "text",
					Data: []byte("Hello"),
					Size: 5,
				},
			},
			wantErr: false,
		},
		{
			name: "valid input with multiple blocks",
			blocks: []*ContentBlock{
				{
					Type: "text",
					Data: []byte("Hello"),
					Size: 5,
				},
				{
					Type: "image",
					URL:  "https://example.com/image.png",
					Size: 100,
				},
			},
			wantErr: false,
		},
		{
			name:        "empty blocks",
			blocks:      []*ContentBlock{},
			wantErr:     true,
			errContains: "must have at least one content block",
		},
		{
			name: "invalid block",
			blocks: []*ContentBlock{
				{
					Type: "invalid",
					Data: []byte("test"),
				},
			},
			wantErr:     true,
			errContains: "validation failed",
		},
		{
			name: "with routing option",
			blocks: []*ContentBlock{
				{
					Type: "text",
					Data: []byte("Hello"),
					Size: 5,
				},
			},
			opts: []MultimodalInputOption{
				WithRouting(&RoutingConfig{
					Strategy:       "auto",
					FallbackToText: true,
				}),
			},
			wantErr: false,
		},
		{
			name: "with metadata option",
			blocks: []*ContentBlock{
				{
					Type: "text",
					Data: []byte("Hello"),
					Size: 5,
				},
			},
			opts: []MultimodalInputOption{
				WithMetadata(map[string]any{
					"key": "value",
				}),
			},
			wantErr: false,
		},
		{
			name: "with format option",
			blocks: []*ContentBlock{
				{
					Type: "text",
					Data: []byte("Hello"),
					Size: 5,
				},
			},
			opts: []MultimodalInputOption{
				WithFormat("url"),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input, err := NewMultimodalInput(tt.blocks, tt.opts...)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.Nil(t, input)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, input)
				assert.NotEmpty(t, input.ID)
				assert.Equal(t, len(tt.blocks), len(input.ContentBlocks))
				assert.NotNil(t, input.Metadata)
			}
		})
	}
}

func TestMultimodalInput_Validate(t *testing.T) {
	tests := []struct {
		name        string
		input       *MultimodalInput
		wantErr     bool
		errContains string
	}{
		{
			name: "valid input",
			input: &MultimodalInput{
				ContentBlocks: []*ContentBlock{
					{
						Type: "text",
						Data: []byte("Hello"),
						Size: 5,
					},
				},
				Format: "base64",
			},
			wantErr: false,
		},
		{
			name: "empty content blocks",
			input: &MultimodalInput{
				ContentBlocks: []*ContentBlock{},
			},
			wantErr:     true,
			errContains: "must have at least one content block",
		},
		{
			name: "invalid format",
			input: &MultimodalInput{
				ContentBlocks: []*ContentBlock{
					{
						Type: "text",
						Data: []byte("Hello"),
						Size: 5,
					},
				},
				Format: "invalid",
			},
			wantErr:     true,
			errContains: "invalid format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMultimodalOutput_Validate(t *testing.T) {
	tests := []struct {
		name        string
		output      *MultimodalOutput
		wantErr     bool
		errContains string
	}{
		{
			name: "valid output",
			output: &MultimodalOutput{
				ContentBlocks: []*ContentBlock{
					{
						Type: "text",
						Data: []byte("Hello"),
						Size: 5,
					},
				},
				Confidence: 0.95,
				Provider:   "openai",
				Model:      "gpt-4o",
			},
			wantErr: false,
		},
		{
			name: "empty content blocks",
			output: &MultimodalOutput{
				ContentBlocks: []*ContentBlock{},
			},
			wantErr:     true,
			errContains: "must have at least one content block",
		},
		{
			name: "invalid confidence too low",
			output: &MultimodalOutput{
				ContentBlocks: []*ContentBlock{
					{
						Type: "text",
						Data: []byte("Hello"),
						Size: 5,
					},
				},
				Confidence: -0.1,
				Provider:   "openai",
				Model:      "gpt-4o",
			},
			wantErr:     true,
			errContains: "confidence must be between",
		},
		{
			name: "invalid confidence too high",
			output: &MultimodalOutput{
				ContentBlocks: []*ContentBlock{
					{
						Type: "text",
						Data: []byte("Hello"),
						Size: 5,
					},
				},
				Confidence: 1.1,
				Provider:   "openai",
				Model:      "gpt-4o",
			},
			wantErr:     true,
			errContains: "confidence must be between",
		},
		{
			name: "empty provider",
			output: &MultimodalOutput{
				ContentBlocks: []*ContentBlock{
					{
						Type: "text",
						Data: []byte("Hello"),
						Size: 5,
					},
				},
				Confidence: 0.95,
				Provider:   "",
				Model:      "gpt-4o",
			},
			wantErr:     true,
			errContains: "provider cannot be empty",
		},
		{
			name: "empty model",
			output: &MultimodalOutput{
				ContentBlocks: []*ContentBlock{
					{
						Type: "text",
						Data: []byte("Hello"),
						Size: 5,
					},
				},
				Confidence: 0.95,
				Provider:   "openai",
				Model:      "",
			},
			wantErr:     true,
			errContains: "model cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.output.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestFunctionalOptions(t *testing.T) {
	blocks := []*ContentBlock{
		{
			Type: "text",
			Data: []byte("Hello"),
			Size: 5,
		},
	}

	// Test WithRouting
	input, err := NewMultimodalInput(blocks, WithRouting(&RoutingConfig{
		Strategy:       "auto",
		FallbackToText: true,
	}))
	require.NoError(t, err)
	assert.NotNil(t, input.Routing)
	assert.Equal(t, "auto", input.Routing.Strategy)

	// Test WithMetadata
	input, err = NewMultimodalInput(blocks, WithMetadata(map[string]any{
		"key1": "value1",
		"key2": 42,
	}))
	require.NoError(t, err)
	assert.Equal(t, "value1", input.Metadata["key1"])
	assert.Equal(t, 42, input.Metadata["key2"])

	// Test WithFormat
	input, err = NewMultimodalInput(blocks, WithFormat("url"))
	require.NoError(t, err)
	assert.Equal(t, "url", input.Format)
}
