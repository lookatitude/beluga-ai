package transport

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTransport_Connect(t *testing.T) {
	tests := []struct {
		name          string
		transport     *AdvancedMockTransport
		expectedError bool
	}{
		{
			name: "successful connection",
			transport: NewAdvancedMockTransport("test",
				WithConnected(false)),
			expectedError: false,
		},
		{
			name: "error on connection",
			transport: NewAdvancedMockTransport("test",
				WithError(NewTransportError("Connect", ErrCodeConnectionFailed, nil))),
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := tt.transport.Connect(ctx)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.True(t, tt.transport.IsConnected())
			}
		})
	}
}

func TestTransport_Disconnect(t *testing.T) {
	tests := []struct {
		name          string
		transport     *AdvancedMockTransport
		expectedError bool
	}{
		{
			name: "successful disconnection",
			transport: NewAdvancedMockTransport("test",
				WithConnected(true)),
			expectedError: false,
		},
		{
			name: "error on disconnection",
			transport: NewAdvancedMockTransport("test",
				WithError(NewTransportError("Disconnect", ErrCodeInternalError, nil))),
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := tt.transport.Disconnect(ctx)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.False(t, tt.transport.IsConnected())
			}
		})
	}
}

func TestTransport_SendAudio(t *testing.T) {
	tests := []struct {
		name          string
		transport     *AdvancedMockTransport
		audio         []byte
		expectedError bool
	}{
		{
			name: "successful send",
			transport: NewAdvancedMockTransport("test",
				WithConnected(true)),
			audio:         []byte{1, 2, 3, 4, 5},
			expectedError: false,
		},
		{
			name: "error when not connected",
			transport: NewAdvancedMockTransport("test",
				WithConnected(false)),
			audio:         []byte{1, 2, 3, 4, 5},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := tt.transport.SendAudio(ctx, tt.audio)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTransport_ReceiveAudio(t *testing.T) {
	tests := []struct {
		name      string
		transport *AdvancedMockTransport
		wantData  bool
	}{
		{
			name: "successful receive",
			transport: NewAdvancedMockTransport("test",
				WithConnected(true),
				WithAudioData(
					[]byte{1, 2, 3},
					[]byte{4, 5, 6},
				)),
			wantData: true,
		},
		{
			name: "no data when not connected",
			transport: NewAdvancedMockTransport("test",
				WithConnected(false)),
			wantData: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			audioCh := tt.transport.ReceiveAudio()
			assert.NotNil(t, audioCh)

			// Receive audio data
			timeout := time.After(2 * time.Second)
			received := 0
			for {
				select {
				case audio, ok := <-audioCh:
					if !ok {
						if tt.wantData && received == 0 {
							t.Fatal("expected data but received none")
						}
						return
					}
					assert.NotNil(t, audio)
					received++
				case <-timeout:
					if tt.wantData && received == 0 {
						t.Fatal("timeout waiting for audio data")
					}
					return
				}
			}
		})
	}
}

func TestTransport_OnAudioReceived(t *testing.T) {
	transport := NewAdvancedMockTransport("test",
		WithConnected(true),
		WithAudioData([]byte{1, 2, 3, 4, 5}))

	received := false
	transport.OnAudioReceived(func(audio []byte) {
		received = true
		assert.NotNil(t, audio)
	})

	// Trigger callback if implemented
	_ = received
}

func TestTransport_Close(t *testing.T) {
	transport := NewAdvancedMockTransport("test",
		WithConnected(true))

	err := transport.Close()
	assert.NoError(t, err)
	assert.False(t, transport.IsConnected())
}

func TestTransport_InterfaceCompliance(t *testing.T) {
	transport := NewAdvancedMockTransport("test")
	AssertTransportInterface(t, transport)
}
