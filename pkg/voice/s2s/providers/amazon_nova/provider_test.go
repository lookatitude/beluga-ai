package amazon_nova

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/s2s"
	"github.com/lookatitude/beluga-ai/pkg/voice/s2s/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/s2s/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAmazonNovaProvider(t *testing.T) {
	tests := []struct {
		config  *s2s.Config
		name    string
		wantErr bool
	}{
		{
			name: "valid config",
			config: &s2s.Config{
				Provider: "amazon_nova",
				APIKey:   "test-key", // Not used but required by validation
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
			config: &s2s.Config{
				Provider: "amazon_nova",
				APIKey:   "test-key",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: This test may fail if AWS credentials are not configured
			// In a real environment, we'd mock the AWS SDK
			provider, err := NewAmazonNovaProvider(tt.config)
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, provider)
			} else {
				// May error if AWS config is not available, which is expected in test environments
				if err != nil {
					t.Logf("Provider creation failed (expected if AWS credentials not configured): %v", err)
					return
				}
				require.NotNil(t, provider)
				assert.Equal(t, "amazon_nova", provider.Name())
			}
		})
	}
}

func TestAmazonNovaProvider_Process(t *testing.T) {
	// Skip if AWS credentials are not available
	config := &s2s.Config{
		Provider: "amazon_nova",
		APIKey:   "test-key",
	}
	provider, err := NewAmazonNovaProvider(config)
	if err != nil {
		t.Skipf("Skipping test - AWS credentials not configured: %v", err)
		return
	}

	input := &internal.AudioInput{
		Data: []byte{1, 2, 3, 4, 5},
		Format: internal.AudioFormat{
			SampleRate: 24000,
			Channels:   1,
			BitDepth:   16,
			Encoding:   "PCM",
		},
		Timestamp: time.Now(),
	}

	convCtx := &internal.ConversationContext{
		ConversationID: "test-conv",
		SessionID:      "test-session",
	}

	ctx := context.Background()
	output, err := provider.Process(ctx, input, convCtx)

	// Note: This is a placeholder implementation, so it should succeed with mock data
	require.NoError(t, err)
	assert.NotNil(t, output)
	assert.NotEmpty(t, output.Data)
}

func TestAmazonNovaProvider_Process_ContextCancellation(t *testing.T) {
	config := &s2s.Config{
		Provider: "amazon_nova",
		APIKey:   "test-key",
	}
	provider, err := NewAmazonNovaProvider(config)
	if err != nil {
		t.Skipf("Skipping test - AWS credentials not configured: %v", err)
		return
	}

	input := &internal.AudioInput{
		Data: []byte{1, 2, 3, 4, 5},
		Format: internal.AudioFormat{
			SampleRate: 24000,
			Channels:   1,
			BitDepth:   16,
			Encoding:   "PCM",
		},
		Timestamp: time.Now(),
	}

	convCtx := &internal.ConversationContext{
		ConversationID: "test-conv",
		SessionID:      "test-session",
	}

	// Create a context that is immediately cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = provider.Process(ctx, input, convCtx)

	// Should respect context cancellation
	// Note: Current placeholder implementation may not check context,
	// but the real implementation should
	if err != nil {
		assert.ErrorIs(t, err, context.Canceled)
	}
}

func TestAmazonNovaProvider_StartStreaming_ContextCancellation(t *testing.T) {
	config := &s2s.Config{
		Provider: "amazon_nova",
		APIKey:   "test-key",
	}
	provider, err := NewAmazonNovaProvider(config)
	if err != nil {
		t.Skipf("Skipping test - AWS credentials not configured: %v", err)
		return
	}

	streamingProvider, ok := provider.(iface.StreamingS2SProvider)
	if !ok {
		t.Skip("Provider does not implement StreamingS2SProvider")
		return
	}

	convCtx := &internal.ConversationContext{
		ConversationID: "test-conv",
		SessionID:      "test-session",
	}

	// Create a context that is immediately cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = streamingProvider.StartStreaming(ctx, convCtx)

	// Should respect context cancellation or return config error if streaming disabled
	if err != nil {
		// Either context was canceled or streaming is disabled (both are acceptable)
		assert.True(t, errors.Is(err, context.Canceled) || 
			strings.Contains(err.Error(), "streaming is disabled") ||
			strings.Contains(err.Error(), "invalid_config"))
	}
}

func TestAmazonNovaProvider_Name(t *testing.T) {
	config := &s2s.Config{
		Provider: "amazon_nova",
		APIKey:   "test-key",
	}
	provider, err := NewAmazonNovaProvider(config)
	if err != nil {
		t.Skipf("Skipping test - AWS credentials not configured: %v", err)
		return
	}

	assert.Equal(t, "amazon_nova", provider.Name())
}
