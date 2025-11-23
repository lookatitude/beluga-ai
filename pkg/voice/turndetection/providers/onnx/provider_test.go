package onnx

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/turndetection"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewONNXProvider(t *testing.T) {
	tests := []struct {
		name    string
		config  *turndetection.Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &turndetection.Config{
				Provider: "onnx",
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "config with defaults",
			config: &turndetection.Config{
				Provider: "onnx",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewONNXProvider(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, provider)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, provider)
			}
		})
	}
}

func TestONNXProvider_DetectTurn(t *testing.T) {
	// Create a temporary model file for testing
	tmpFile, err := os.CreateTemp("", "test_model.onnx")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	config := &turndetection.Config{
		Provider: "onnx",
	}
	onnxConfig := &ONNXConfig{
		Config:     config,
		ModelPath:  tmpFile.Name(),
		Threshold:  0.5,
		SampleRate: 16000,
		FrameSize:  512,
	}

	provider, err := NewONNXProvider(config)
	require.NoError(t, err)
	require.NotNil(t, provider)

	// Update provider config for test
	onnxProvider := provider.(*ONNXProvider)
	onnxProvider.config = onnxConfig

	ctx := context.Background()
	audio := make([]byte, 1024) // 512 samples * 2 bytes

	// Test processing
	turn, err := onnxProvider.DetectTurn(ctx, audio)
	assert.NoError(t, err)
	// Result depends on audio content, but should not error
	_ = turn
}

func TestONNXProvider_DetectTurnWithSilence(t *testing.T) {
	config := &turndetection.Config{
		Provider: "onnx",
	}

	provider, err := NewONNXProvider(config)
	require.NoError(t, err)
	require.NotNil(t, provider)

	ctx := context.Background()
	audio := make([]byte, 1024)

	// Test with silence duration below threshold
	turn, err := provider.DetectTurnWithSilence(ctx, audio, 100*time.Millisecond)
	assert.NoError(t, err)
	// Result depends on implementation
	_ = turn

	// Test with silence duration above threshold
	turn, err = provider.DetectTurnWithSilence(ctx, audio, 600*time.Millisecond)
	assert.NoError(t, err)
	assert.True(t, turn)
}

func TestDefaultONNXConfig(t *testing.T) {
	config := DefaultONNXConfig()
	assert.NotNil(t, config)
	assert.Equal(t, "turn_detection.onnx", config.ModelPath)
	assert.Equal(t, 0.5, config.Threshold)
	assert.Equal(t, 16000, config.SampleRate)
	assert.Equal(t, 512, config.FrameSize)
}
