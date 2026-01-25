package s2s

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/s2s/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/s2s/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestS2SProvider_Process(t *testing.T) {
	tests := []struct {
		provider      iface.S2SProvider
		input         *internal.AudioInput
		name          string
		expectedError bool
	}{
		{
			name: "successful process",
			provider: NewAdvancedMockS2SProvider("test",
				WithAudioOutputs(&internal.AudioOutput{
					Data: []byte{1, 2, 3, 4, 5},
					Format: internal.AudioFormat{
						SampleRate: 24000,
						Channels:   1,
						BitDepth:   16,
						Encoding:   "PCM",
					},
					Timestamp: time.Now(),
					Provider:  "test",
					Latency:   100 * time.Millisecond,
				})),
			input: &internal.AudioInput{
				Data: []byte{1, 2, 3, 4, 5},
				Format: internal.AudioFormat{
					SampleRate: 24000,
					Channels:   1,
					BitDepth:   16,
					Encoding:   "PCM",
				},
				Timestamp: time.Now(),
			},
			expectedError: false,
		},
		{
			name: "error on process",
			provider: NewAdvancedMockS2SProvider("test",
				WithError(NewS2SError("Process", ErrCodeNetworkError, nil))),
			input: &internal.AudioInput{
				Data: []byte{1, 2, 3, 4, 5},
				Format: internal.AudioFormat{
					SampleRate: 24000,
					Channels:   1,
					BitDepth:   16,
					Encoding:   "PCM",
				},
				Timestamp: time.Now(),
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			convCtx := &internal.ConversationContext{
				ConversationID: "test-conv",
				SessionID:      "test-session",
			}
			output, err := tt.provider.Process(ctx, tt.input, convCtx)

			if tt.expectedError {
				require.Error(t, err)
				assert.Nil(t, output)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, output)
				assert.NotEmpty(t, output.Data)
			}
		})
	}
}

func TestS2SProvider_Process_Concurrent(t *testing.T) {
	mockProvider := NewAdvancedMockS2SProvider("test",
		WithAudioOutputs(&internal.AudioOutput{
			Data: []byte{1, 2, 3, 4, 5},
			Format: internal.AudioFormat{
				SampleRate: 24000,
				Channels:   1,
				BitDepth:   16,
				Encoding:   "PCM",
			},
			Timestamp: time.Now(),
			Provider:  "test",
			Latency:   100 * time.Millisecond,
		}))

	var provider iface.S2SProvider = mockProvider

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

	const numGoroutines = 10
	const numCallsPerGoroutine = 5

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			ctx := context.Background()
			for j := 0; j < numCallsPerGoroutine; j++ {
				output, err := provider.Process(ctx, input, convCtx)
				require.NoError(t, err)
				assert.NotNil(t, output)
			}
		}()
	}

	wg.Wait()

	assert.Equal(t, numGoroutines*numCallsPerGoroutine, mockProvider.GetCallCount())
}

func TestS2SProvider_InterfaceCompliance(t *testing.T) {
	provider := NewAdvancedMockS2SProvider("test")
	AssertS2SProviderInterface(t, provider)
}

func TestStreamingS2SProvider_StartStreaming(t *testing.T) {
	tests := []struct {
		provider      iface.S2SProvider
		name          string
		expectedError bool
	}{
		{
			name: "successful streaming start",
			provider: NewAdvancedMockS2SProvider("test",
				WithAudioOutputs(&internal.AudioOutput{
					Data: []byte{1, 2, 3, 4, 5},
					Format: internal.AudioFormat{
						SampleRate: 24000,
						Channels:   1,
						BitDepth:   16,
						Encoding:   "PCM",
					},
					Timestamp: time.Now(),
					Provider:  "test",
					Latency:   100 * time.Millisecond,
				})),
			expectedError: false,
		},
		{
			name: "error on streaming start",
			provider: NewAdvancedMockS2SProvider("test",
				WithError(NewS2SError("StartStreaming", ErrCodeNetworkError, nil))),
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			convCtx := &internal.ConversationContext{
				ConversationID: "test-conv",
				SessionID:      "test-session",
			}
			streamingProvider, ok := tt.provider.(iface.StreamingS2SProvider)
			if !ok {
				t.Skip("provider does not implement StreamingS2SProvider")
				return
			}

			session, err := streamingProvider.StartStreaming(ctx, convCtx)

			if tt.expectedError {
				require.Error(t, err)
				assert.Nil(t, session)
			} else {
				require.NoError(t, err)
				require.NotNil(t, session)
				defer func() { _ = session.Close() }()

				// Test receiving audio
				timeout := time.After(2 * time.Second)
				select {
				case chunk := <-session.ReceiveAudio():
					assert.NotEmpty(t, chunk.Audio)
					require.NoError(t, chunk.Error)
				case <-timeout:
					t.Fatal("timeout waiting for audio chunk")
				}
			}
		})
	}
}

func TestStreamingSession_SendAudio(t *testing.T) {
	ctx := context.Background()
	mockProvider := NewAdvancedMockS2SProvider("test",
		WithAudioOutputs(&internal.AudioOutput{
			Data: []byte{1, 2, 3, 4, 5},
			Format: internal.AudioFormat{
				SampleRate: 24000,
				Channels:   1,
				BitDepth:   16,
				Encoding:   "PCM",
			},
			Timestamp: time.Now(),
			Provider:  "test",
			Latency:   100 * time.Millisecond,
		}))

	var provider iface.S2SProvider = mockProvider
	streamingProvider, ok := provider.(iface.StreamingS2SProvider)
	if !ok {
		t.Skip("provider does not implement StreamingS2SProvider")
		return
	}

	convCtx := &internal.ConversationContext{
		ConversationID: "test-conv",
		SessionID:      "test-session",
	}
	session, err := streamingProvider.StartStreaming(ctx, convCtx)
	require.NoError(t, err)
	require.NotNil(t, session)
	defer func() { _ = session.Close() }()

	err = session.SendAudio(ctx, []byte{1, 2, 3, 4, 5})
	require.NoError(t, err)
}

func TestStreamingSession_Close(t *testing.T) {
	ctx := context.Background()
	mockProvider := NewAdvancedMockS2SProvider("test",
		WithAudioOutputs(&internal.AudioOutput{
			Data: []byte{1, 2, 3, 4, 5},
			Format: internal.AudioFormat{
				SampleRate: 24000,
				Channels:   1,
				BitDepth:   16,
				Encoding:   "PCM",
			},
			Timestamp: time.Now(),
			Provider:  "test",
			Latency:   100 * time.Millisecond,
		}))

	var provider iface.S2SProvider = mockProvider
	streamingProvider, ok := provider.(iface.StreamingS2SProvider)
	if !ok {
		t.Skip("provider does not implement StreamingS2SProvider")
		return
	}

	convCtx := &internal.ConversationContext{
		ConversationID: "test-conv",
		SessionID:      "test-session",
	}
	session, err := streamingProvider.StartStreaming(ctx, convCtx)
	require.NoError(t, err)
	require.NotNil(t, session)

	err = session.Close()
	require.NoError(t, err)

	// Closing again should not error
	err = session.Close()
	require.NoError(t, err)
}

func TestS2SProvider_Name(t *testing.T) {
	provider := NewAdvancedMockS2SProvider("test-provider")
	assert.Equal(t, "test-provider", provider.Name())
}
