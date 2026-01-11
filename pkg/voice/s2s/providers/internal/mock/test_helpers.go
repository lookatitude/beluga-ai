package mock

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/s2s/internal"
)

// CreateMockAudioResponse creates a mock audio response in the format expected by providers.
func CreateMockAudioResponse(audioData []byte, format internal.AudioFormat, voiceID, language string) map[string]interface{} {
	audioBase64 := base64.StdEncoding.EncodeToString(audioData)
	return map[string]interface{}{
		"audio": audioBase64,
		"format": map[string]interface{}{
			"sample_rate": format.SampleRate,
			"channels":    format.Channels,
			"encoding":    format.Encoding,
		},
		"voice": map[string]interface{}{
			"voice_id": voiceID,
			"language": language,
		},
	}
}

// CreateGeminiMockResponse creates a mock response in Gemini API format.
func CreateGeminiMockResponse(audioData []byte, format internal.AudioFormat) map[string]interface{} {
	audioBase64 := base64.StdEncoding.EncodeToString(audioData)
	return map[string]interface{}{
		"candidates": []map[string]interface{}{
			{
				"content": map[string]interface{}{
					"parts": []map[string]interface{}{
						{
							"inlineData": map[string]interface{}{
								"mimeType": "audio/pcm",
								"data":     audioBase64,
							},
						},
					},
				},
			},
		},
	}
}

// CreateGrokMockResponse creates a mock response in Grok API format.
func CreateGrokMockResponse(audioData []byte, format internal.AudioFormat, voiceID, language string) map[string]interface{} {
	audioBase64 := base64.StdEncoding.EncodeToString(audioData)
	return map[string]interface{}{
		"audio": audioBase64,
		"format": map[string]interface{}{
			"sample_rate": format.SampleRate,
			"channels":    format.Channels,
			"encoding":    format.Encoding,
		},
		"voice": map[string]interface{}{
			"voice_id": voiceID,
			"language": language,
		},
	}
}

// CreateAmazonNovaMockResponse creates a mock response in Amazon Nova API format.
func CreateAmazonNovaMockResponse(audioData []byte, format internal.AudioFormat, voiceID, language string) map[string]interface{} {
	audioBase64 := base64.StdEncoding.EncodeToString(audioData)
	return map[string]interface{}{
		"output": map[string]interface{}{
			"audio": audioBase64,
			"format": map[string]interface{}{
				"sample_rate": format.SampleRate,
				"channels":    format.Channels,
				"encoding":    format.Encoding,
			},
		},
		"voice": map[string]interface{}{
			"voice_id": voiceID,
			"language": language,
		},
		"metadata": map[string]interface{}{
			"latency_ms": 100.0,
		},
	}
}

// CreateHTTPServer creates an HTTP test server with a handler that returns the given response.
func CreateHTTPServer(response map[string]interface{}, statusCode int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		if response != nil {
			json.NewEncoder(w).Encode(response)
		}
	}))
}

// CreateHTTPServerWithHandler creates an HTTP test server with a custom handler.
func CreateHTTPServerWithHandler(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}

// CreateMockAudioData creates mock audio data for testing.
func CreateMockAudioData(size int) []byte {
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i % 256)
	}
	return data
}

// CreateMockAudioInput creates a mock audio input for testing.
func CreateMockAudioInput() *internal.AudioInput {
	return &internal.AudioInput{
		Data: CreateMockAudioData(1000),
		Format: internal.AudioFormat{
			SampleRate: 24000,
			Channels:   1,
			BitDepth:   16,
			Encoding:   "PCM",
		},
		Timestamp: time.Now(),
		Language:  "en-US",
	}
}

// CreateMockConversationContext creates a mock conversation context for testing.
func CreateMockConversationContext() *internal.ConversationContext {
	return &internal.ConversationContext{
		ConversationID: "test-conv-123",
		SessionID:      "test-session-456",
		History:        []internal.ConversationTurn{},
	}
}
