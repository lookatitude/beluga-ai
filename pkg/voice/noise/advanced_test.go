package noise

import (
	"context"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoiseCancellation_Process(t *testing.T) {
	tests := []struct {
		name           string
		cancellation   iface.NoiseCancellation
		audio          []byte
		expectedError  bool
		expectedOutput bool
	}{
		{
			name: "successful processing",
			cancellation: NewAdvancedMockNoiseCancellation("test",
				WithProcessedAudio([]byte{5, 4, 3, 2, 1})),
			audio:          []byte{1, 2, 3, 4, 5},
			expectedError:  false,
			expectedOutput: true,
		},
		{
			name: "error on processing",
			cancellation: NewAdvancedMockNoiseCancellation("test",
				WithError(NewNoiseCancellationError("Process", ErrCodeInternalError, nil))),
			audio:          []byte{1, 2, 3, 4, 5},
			expectedError:  true,
			expectedOutput: false,
		},
		{
			name:           "empty audio",
			cancellation:   NewAdvancedMockNoiseCancellation("test"),
			audio:          []byte{},
			expectedError:  false,
			expectedOutput: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			processed, err := tt.cancellation.Process(ctx, tt.audio)

			if tt.expectedError {
				require.Error(t, err)
				assert.Nil(t, processed)
			} else {
				require.NoError(t, err)
				if tt.expectedOutput {
					assert.NotNil(t, processed)
				}
			}
		})
	}
}

func TestNoiseCancellation_ProcessStream(t *testing.T) {
	tests := []struct {
		name         string
		cancellation iface.NoiseCancellation
		audio        [][]byte
		expectedErr  bool
	}{
		{
			name: "successful streaming",
			cancellation: NewAdvancedMockNoiseCancellation("test",
				WithProcessedAudio([]byte{5, 4, 3}, []byte{2, 1})),
			audio:       [][]byte{{1, 2, 3}, {4, 5}},
			expectedErr: false,
		},
		{
			name: "error on streaming",
			cancellation: NewAdvancedMockNoiseCancellation("test",
				WithError(NewNoiseCancellationError("ProcessStream", ErrCodeInternalError, nil))),
			audio:       [][]byte{{1, 2, 3}},
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			audioCh := make(chan []byte, 10)
			for _, a := range tt.audio {
				audioCh <- a
			}
			close(audioCh)

			processedCh, err := tt.cancellation.ProcessStream(ctx, audioCh)

			if tt.expectedErr {
				require.Error(t, err)
				// When there's an error, the channel is still returned but closed
				if processedCh != nil {
					// Verify channel is closed
					_, ok := <-processedCh
					assert.False(t, ok, "channel should be closed on error")
				}
			} else {
				require.NoError(t, err)
				assert.NotNil(t, processedCh)

				// Receive processed audio
				timeout := time.After(2 * time.Second)
				received := 0
				for {
					select {
					case processed, ok := <-processedCh:
						if !ok {
							if received == 0 {
								t.Fatal("expected data but received none")
							}
							return
						}
						assert.NotNil(t, processed)
						received++
					case <-timeout:
						if received == 0 {
							t.Fatal("timeout waiting for processed audio")
						}
						return
					}
				}
			}
		})
	}
}

func TestNoiseCancellation_InterfaceCompliance(t *testing.T) {
	cancellation := NewAdvancedMockNoiseCancellation("test")
	AssertNoiseCancellationInterface(t, cancellation)
}
