package grok

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/s2s"
	"github.com/lookatitude/beluga-ai/pkg/voice/s2s/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/s2s/internal"
	mock "github.com/lookatitude/beluga-ai/pkg/voice/s2s/providers/internal/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGrokVoiceProvider(t *testing.T) {
	tests := []struct {
		config  *s2s.Config
		name    string
		wantErr bool
	}{
		{
			name: "valid config",
			config: &s2s.Config{
				Provider: "grok",
				APIKey:   "test-key",
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "config without API key",
			config: &s2s.Config{
				Provider: "grok",
				APIKey:   "",
			},
			wantErr: true,
		},
		{
			name: "config with defaults",
			config: &s2s.Config{
				Provider: "grok",
				APIKey:   "test-key",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewGrokVoiceProvider(tt.config)
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, provider)
			} else {
				require.NoError(t, err)
				require.NotNil(t, provider)
				assert.Equal(t, "grok", provider.Name())
			}
		})
	}
}

func TestGrokVoiceProvider_Process(t *testing.T) {
	// Create mock HTTP server
	mockAudioData := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	mockResponse := mock.CreateGrokMockResponse(mockAudioData, internal.AudioFormat{
		SampleRate: 24000,
		Channels:   1,
		BitDepth:   16,
		Encoding:   "PCM",
	}, "alloy", "en-US")
	server := mock.CreateHTTPServer(mockResponse, http.StatusOK)
	defer server.Close()

	config := &s2s.Config{
		Provider: "grok",
		APIKey:   "test-key",
	}
	
	// Use test helper to create provider with mock server endpoint
	provider, err := NewGrokVoiceProviderWithEndpoint(config, server.URL)
	require.NoError(t, err)
	require.NotNil(t, provider)

	input := &internal.AudioInput{
		Data: []byte{1, 2, 3, 4, 5},
		Format: internal.AudioFormat{
			SampleRate: 24000,
			Channels:   1,
			BitDepth:   16,
			Encoding:   "PCM",
		},
		Timestamp: time.Now(),
		Language:  "en-US",
	}

	convCtx := &internal.ConversationContext{
		ConversationID: "test-conv",
		SessionID:      "test-session",
	}

	ctx := context.Background()
	output, err := provider.Process(ctx, input, convCtx)

	require.NoError(t, err)
	assert.NotNil(t, output)
	assert.NotEmpty(t, output.Data)
}

func TestGrokVoiceProvider_Process_ContextCancellation(t *testing.T) {
	// T039A: Test Process respects context cancellation (FR-011)
	config := &s2s.Config{
		Provider: "grok",
		APIKey:   "test-key",
	}
	provider, err := NewGrokVoiceProvider(config)
	require.NoError(t, err)

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

	// Test with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	output, err := provider.Process(ctx, input, convCtx)
	require.Error(t, err)
	assert.Nil(t, output)
	assert.Contains(t, err.Error(), "cancelled")
}

func TestGrokVoiceProvider_Process_ContextTimeout(t *testing.T) {
	// T039A: Test Process respects context timeout (FR-011)
	config := &s2s.Config{
		Provider: "grok",
		APIKey:   "test-key",
	}
	provider, err := NewGrokVoiceProvider(config)
	require.NoError(t, err)

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

	// Test with timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Wait for timeout
	time.Sleep(10 * time.Millisecond)

	output, err := provider.Process(ctx, input, convCtx)
	require.Error(t, err)
	assert.Nil(t, output)
	assert.Contains(t, err.Error(), "cancelled")
}

func TestGrokVoiceProvider_Streaming_ContextCancellation(t *testing.T) {
	// T039A: Test streaming respects context cancellation (FR-011)
	config := &s2s.Config{
		Provider: "grok",
		APIKey:   "test-key",
	}
	provider, err := NewGrokVoiceProvider(config)
	require.NoError(t, err)

	streamingProvider, ok := provider.(iface.StreamingS2SProvider)
	if !ok {
		t.Skip("Provider does not implement StreamingS2SProvider")
		return
	}

	convCtx := &internal.ConversationContext{
		ConversationID: "test-conv",
		SessionID:      "test-session",
	}

	// Test with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	session, err := streamingProvider.StartStreaming(ctx, convCtx)
	if err != nil {
		// Expected to fail with cancelled context
		assert.Contains(t, err.Error(), "cancelled")
		return
	}

	// If session was created, test sending audio with cancelled context
	if session != nil {
		ctx2, cancel2 := context.WithCancel(context.Background())
		cancel2()
		err = session.SendAudio(ctx2, []byte{1, 2, 3})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cancelled")
		_ = session.Close()
	}
}

func TestGrokVoiceProvider_Streaming_ContextTimeout(t *testing.T) {
	// T039A: Test streaming respects context timeout (FR-011)
	config := &s2s.Config{
		Provider: "grok",
		APIKey:   "test-key",
	}
	provider, err := NewGrokVoiceProvider(config)
	require.NoError(t, err)

	streamingProvider, ok := provider.(iface.StreamingS2SProvider)
	if !ok {
		t.Skip("Provider does not implement StreamingS2SProvider")
		return
	}

	convCtx := &internal.ConversationContext{
		ConversationID: "test-conv",
		SessionID:      "test-session",
	}

	// Test with timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Wait for timeout
	time.Sleep(10 * time.Millisecond)

	session, err := streamingProvider.StartStreaming(ctx, convCtx)
	if err != nil {
		// Expected to fail with timeout context
		assert.Contains(t, err.Error(), "cancelled")
		return
	}

	// If session was created, test sending audio with timeout context
	if session != nil {
		ctx2, cancel2 := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel2()
		time.Sleep(10 * time.Millisecond)
		err = session.SendAudio(ctx2, []byte{1, 2, 3})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cancelled")
		_ = session.Close()
	}
}
