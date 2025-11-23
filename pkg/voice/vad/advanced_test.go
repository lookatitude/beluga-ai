package vad

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/voice/iface"
	"github.com/stretchr/testify/assert"
)

func TestVADProvider_Process(t *testing.T) {
	tests := []struct {
		name           string
		provider       iface.VADProvider
		audio          []byte
		expectedError  bool
		expectedSpeech bool
	}{
		{
			name: "speech detected",
			provider: NewAdvancedMockVADProvider("test",
				WithSpeechResults(true)),
			audio:          []byte{1, 2, 3, 4, 5},
			expectedError:  false,
			expectedSpeech: true,
		},
		{
			name: "silence detected",
			provider: NewAdvancedMockVADProvider("test",
				WithSpeechResults(false)),
			audio:          []byte{1, 2, 3, 4, 5},
			expectedError:  false,
			expectedSpeech: false,
		},
		{
			name: "error on processing",
			provider: NewAdvancedMockVADProvider("test",
				WithError(NewVADError("Process", ErrCodeInternalError, nil))),
			audio:          []byte{1, 2, 3, 4, 5},
			expectedError:  true,
			expectedSpeech: false,
		},
		{
			name: "empty audio",
			provider: NewAdvancedMockVADProvider("test",
				WithSpeechResults(false)),
			audio:          []byte{},
			expectedError:  false,
			expectedSpeech: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			speech, err := tt.provider.Process(ctx, tt.audio)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedSpeech, speech)
			}
		})
	}
}

func TestVADProvider_InterfaceCompliance(t *testing.T) {
	provider := NewAdvancedMockVADProvider("test")
	AssertVADProviderInterface(t, provider)
}
