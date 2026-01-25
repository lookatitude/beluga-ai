package gemini

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

func TestNewGeminiNativeProvider(t *testing.T) {
	tests := []struct {
		config  *s2s.Config
		name    string
		wantErr bool
	}{
		{
			name: "valid config",
			config: &s2s.Config{
				Provider: "gemini",
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
				Provider: "gemini",
				APIKey:   "",
			},
			wantErr: true,
		},
		{
			name: "config with defaults",
			config: &s2s.Config{
				Provider: "gemini",
				APIKey:   "test-key",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewGeminiNativeProvider(tt.config)
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, provider)
			} else {
				require.NoError(t, err)
				require.NotNil(t, provider)
				assert.Equal(t, "gemini", provider.Name())
			}
		})
	}
}

func TestGeminiNativeProvider_Process(t *testing.T) {
	// Create mock HTTP server
	mockAudioData := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	mockResponse := mock.CreateGeminiMockResponse(mockAudioData, internal.AudioFormat{
		SampleRate: 24000,
		Channels:   1,
		BitDepth:   16,
		Encoding:   "PCM",
	})
	server := mock.CreateHTTPServer(mockResponse, http.StatusOK)
	defer server.Close()

	// Create config with mock server endpoint
	// We need to manually construct the provider with the mock endpoint
	// Since the provider doesn't expose config modification, we'll use a workaround
	config := &s2s.Config{
		Provider: "gemini",
		APIKey:   "test-key",
	}

	// Create provider normally, then we'll need to modify it
	// For now, let's create a test that uses the actual provider but with a mock server
	// by setting the endpoint via environment or config override
	// Since we can't easily inject the endpoint, let's create a provider that uses NewGeminiNativeProvider
	// and then modify the URL in the request

	// Use test helper to create provider with mock server endpoint
	// The endpoint is set via config, so we need to update the config
	provider, err := NewGeminiNativeProviderWithEndpoint(config, server.URL)
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

func TestGeminiNativeProvider_Process_ContextCancellation(t *testing.T) {
	// T044A: Test Process respects context cancellation (FR-011)
	config := &s2s.Config{
		Provider: "gemini",
		APIKey:   "test-key",
	}
	provider, err := NewGeminiNativeProvider(config)
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

	// Test with canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	output, err := provider.Process(ctx, input, convCtx)
	require.Error(t, err)
	assert.Nil(t, output)
	assert.Contains(t, err.Error(), "canceled")
}

func TestGeminiNativeProvider_Process_ContextTimeout(t *testing.T) {
	// T044A: Test Process respects context timeout (FR-011)
	config := &s2s.Config{
		Provider: "gemini",
		APIKey:   "test-key",
	}
	provider, err := NewGeminiNativeProvider(config)
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
	assert.Contains(t, err.Error(), "canceled")
}

func TestGeminiNativeProvider_Streaming_ContextCancellation(t *testing.T) {
	// T044A: Test streaming respects context cancellation (FR-011)
	config := &s2s.Config{
		Provider: "gemini",
		APIKey:   "test-key",
	}
	provider, err := NewGeminiNativeProvider(config)
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

	// Test with canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	session, err := streamingProvider.StartStreaming(ctx, convCtx)
	if err != nil {
		// Expected to fail with canceled context
		assert.Contains(t, err.Error(), "canceled")
		return
	}

	// If session was created, test sending audio with canceled context
	if session != nil {
		ctx2, cancel2 := context.WithCancel(context.Background())
		cancel2()
		err = session.SendAudio(ctx2, []byte{1, 2, 3})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "canceled")
		_ = session.Close()
	}
}

func TestGeminiNativeProvider_Streaming_ContextTimeout(t *testing.T) {
	// T044A: Test streaming respects context timeout (FR-011)
	config := &s2s.Config{
		Provider: "gemini",
		APIKey:   "test-key",
	}
	provider, err := NewGeminiNativeProvider(config)
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
		assert.Contains(t, err.Error(), "canceled")
		return
	}

	// If session was created, test sending audio with timeout context
	if session != nil {
		ctx2, cancel2 := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel2()
		time.Sleep(10 * time.Millisecond)
		err = session.SendAudio(ctx2, []byte{1, 2, 3})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "canceled")
		_ = session.Close()
	}
}
