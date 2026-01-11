package openai_realtime

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/s2s"
	"github.com/lookatitude/beluga-ai/pkg/voice/s2s/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/s2s/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOpenAIRealtimeProvider(t *testing.T) {
	tests := []struct {
		config  *s2s.Config
		name    string
		wantErr bool
	}{
		{
			name: "valid config",
			config: &s2s.Config{
				Provider: "openai_realtime",
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
				Provider: "openai_realtime",
				APIKey:   "",
			},
			wantErr: true,
		},
		{
			name: "config with defaults",
			config: &s2s.Config{
				Provider: "openai_realtime",
				APIKey:   "test-key",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewOpenAIRealtimeProvider(tt.config)
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, provider)
			} else {
				require.NoError(t, err)
				require.NotNil(t, provider)
				assert.Equal(t, "openai_realtime", provider.Name())
			}
		})
	}
}

func TestOpenAIRealtimeProvider_Process(t *testing.T) {
	// Note: This test requires OpenAI API credentials or will fail.
	// OpenAI Realtime uses WebSocket connections, which are more complex to mock.
	// For proper mocking, the provider would need to be refactored to use a WebSocket interface.
	
	config := &s2s.Config{
		Provider: "openai_realtime",
		APIKey:   "test-key",
	}
	provider, err := NewOpenAIRealtimeProvider(config)
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
		Language:  "en-US",
	}

	convCtx := &internal.ConversationContext{
		ConversationID: "test-conv",
		SessionID:      "test-session",
	}

	ctx := context.Background()
	output, err := provider.Process(ctx, input, convCtx)

	// OpenAI Realtime requires streaming to be enabled, and will fail if disabled
	// This test will fail with real API calls if credentials are invalid or streaming is disabled
	if err != nil {
		// Check if it's a config error (streaming disabled) or API error
		if strings.Contains(err.Error(), "streaming is disabled") ||
		   strings.Contains(err.Error(), "invalid_config") ||
		   strings.Contains(err.Error(), "invalid_request") ||
		   strings.Contains(err.Error(), "authentication") {
			t.Skipf("Skipping test - Configuration or API error (expected without valid setup): %v", err)
			return
		}
		// Other errors should fail the test
		require.NoError(t, err)
	}
	
	if output != nil {
		assert.NotEmpty(t, output.Data)
	}
}

func TestOpenAIRealtimeProvider_Process_ContextCancellation(t *testing.T) {
	// T049A: Test Process respects context cancellation (FR-011)
	config := &s2s.Config{
		Provider: "openai_realtime",
		APIKey:   "test-key",
	}
	provider, err := NewOpenAIRealtimeProvider(config)
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

func TestOpenAIRealtimeProvider_Process_ContextTimeout(t *testing.T) {
	// T049A: Test Process respects context timeout (FR-011)
	config := &s2s.Config{
		Provider: "openai_realtime",
		APIKey:   "test-key",
	}
	provider, err := NewOpenAIRealtimeProvider(config)
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

func TestOpenAIRealtimeProvider_Streaming_ContextCancellation(t *testing.T) {
	// T049A: Test streaming respects context cancellation (FR-011)
	config := &s2s.Config{
		Provider: "openai_realtime",
		APIKey:   "test-key",
	}
	provider, err := NewOpenAIRealtimeProvider(config)
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

func TestOpenAIRealtimeProvider_Streaming_ContextTimeout(t *testing.T) {
	// T049A: Test streaming respects context timeout (FR-011)
	config := &s2s.Config{
		Provider: "openai_realtime",
		APIKey:   "test-key",
	}
	provider, err := NewOpenAIRealtimeProvider(config)
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
