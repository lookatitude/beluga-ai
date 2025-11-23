package silero

import (
	"context"
	"os"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/voice/vad"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSileroProvider(t *testing.T) {
	tests := []struct {
		name    string
		config  *vad.Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &vad.Config{
				Provider: "silero",
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
			config: &vad.Config{
				Provider: "silero",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewSileroProvider(tt.config)
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

func TestSileroProvider_Process(t *testing.T) {
	// Create a temporary model file for testing
	tmpFile, err := os.CreateTemp("", "test_model.onnx")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	config := &vad.Config{
		Provider: "silero",
	}
	sileroConfig := &SileroConfig{
		Config:     config,
		ModelPath:  tmpFile.Name(),
		Threshold:  0.5,
		SampleRate: 16000,
		FrameSize:  512,
	}

	provider, err := NewSileroProvider(config)
	require.NoError(t, err)
	require.NotNil(t, provider)

	// Update provider config for test
	sileroProvider := provider.(*SileroProvider)
	sileroProvider.config = sileroConfig

	ctx := context.Background()
	audio := make([]byte, 1024) // 512 samples * 2 bytes

	// Test processing
	speech, err := sileroProvider.Process(ctx, audio)
	assert.NoError(t, err)
	// Result depends on audio content, but should not error
	_ = speech
}

func TestSileroProvider_ProcessStream(t *testing.T) {
	// Create a temporary model file for testing
	tmpFile, err := os.CreateTemp("", "test_model.onnx")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	config := &vad.Config{
		Provider: "silero",
	}
	sileroConfig := &SileroConfig{
		Config:     config,
		ModelPath:  tmpFile.Name(),
		Threshold:  0.5,
		SampleRate: 16000,
		FrameSize:  512,
	}

	provider, err := NewSileroProvider(config)
	require.NoError(t, err)

	sileroProvider := provider.(*SileroProvider)
	sileroProvider.config = sileroConfig

	ctx := context.Background()
	audioCh := make(chan []byte, 1)

	audio := make([]byte, 1024)
	audioCh <- audio
	close(audioCh)

	resultCh, err := sileroProvider.ProcessStream(ctx, audioCh)
	require.NoError(t, err)

	// Read results
	result := <-resultCh
	assert.NotNil(t, result)
}

func TestSileroProvider_ProcessStream_ContextCancellation(t *testing.T) {
	config := &vad.Config{
		Provider: "silero",
	}

	provider, err := NewSileroProvider(config)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	audioCh := make(chan []byte, 1)

	cancel()

	resultCh, err := provider.ProcessStream(ctx, audioCh)
	// May error due to initialization, but if it succeeds, channel should be closed
	if err == nil {
		_, ok := <-resultCh
		assert.False(t, ok)
	}
}

func TestDefaultSileroConfig(t *testing.T) {
	config := DefaultSileroConfig()
	assert.NotNil(t, config)
	assert.Equal(t, "silero_vad.onnx", config.ModelPath)
	assert.Equal(t, 0.5, config.Threshold)
	assert.Equal(t, 16000, config.SampleRate)
	assert.Equal(t, 512, config.FrameSize)
}
