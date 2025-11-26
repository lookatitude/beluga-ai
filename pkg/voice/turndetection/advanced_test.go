package turndetection

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/voice/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTurnDetector_DetectTurn(t *testing.T) {
	tests := []struct {
		name          string
		detector      iface.TurnDetector
		audio         []byte
		expectedError bool
		expectedTurn  bool
	}{
		{
			name: "turn detected",
			detector: NewAdvancedMockTurnDetector("test",
				WithTurnResults(true)),
			audio:         []byte{1, 2, 3, 4, 5},
			expectedError: false,
			expectedTurn:  true,
		},
		{
			name: "no turn detected",
			detector: NewAdvancedMockTurnDetector("test",
				WithTurnResults(false)),
			audio:         []byte{1, 2, 3, 4, 5},
			expectedError: false,
			expectedTurn:  false,
		},
		{
			name: "error on detection",
			detector: NewAdvancedMockTurnDetector("test",
				WithError(NewTurnDetectionError("DetectTurn", ErrCodeInternalError, nil))),
			audio:         []byte{1, 2, 3, 4, 5},
			expectedError: true,
			expectedTurn:  false,
		},
		{
			name: "empty audio",
			detector: NewAdvancedMockTurnDetector("test",
				WithTurnResults(false)),
			audio:         []byte{},
			expectedError: false,
			expectedTurn:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			turn, err := tt.detector.DetectTurn(ctx, tt.audio)

			if tt.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedTurn, turn)
			}
		})
	}
}

func TestTurnDetector_InterfaceCompliance(t *testing.T) {
	detector := NewAdvancedMockTurnDetector("test")
	AssertTurnDetectorInterface(t, detector)
}
