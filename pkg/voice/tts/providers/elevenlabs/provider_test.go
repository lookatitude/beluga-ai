package elevenlabs

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/tts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewElevenLabsProvider(t *testing.T) {
	tests := []struct {
		name    string
		config  *tts.Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &tts.Config{
				Provider: "elevenlabs",
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
			name: "config with defaults",
			config: &tts.Config{
				Provider: "elevenlabs",
				APIKey:   "test-key",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewElevenLabsProvider(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, provider)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, provider)
			}
		})
	}
}

func TestElevenLabsProvider_GenerateSpeech_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.URL.Path, "/v1/text-to-speech/")
		assert.Equal(t, "test-key", r.Header.Get("xi-api-key"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Parse request body
		var requestBody map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&requestBody)
		require.NoError(t, err)
		assert.Equal(t, "Hello, world!", requestBody["text"])

		// Return mock audio data
		audioData := []byte{1, 2, 3, 4, 5, 6, 7, 8}
		w.Header().Set("Content-Type", "audio/mpeg")
		w.WriteHeader(http.StatusOK)
		w.Write(audioData)
	}))
	defer server.Close()

	config := &tts.Config{
		Provider: "elevenlabs",
		APIKey:   "test-key",
		Timeout:  30 * time.Second,
		BaseURL:  server.URL,
	}

	provider, err := NewElevenLabsProvider(config)
	require.NoError(t, err)
	require.NotNil(t, provider)

	// Set VoiceID (required for ElevenLabs)
	elConfig := provider.(*ElevenLabsProvider).config
	elConfig.VoiceID = "test-voice-id"

	ctx := context.Background()
	text := "Hello, world!"

	audio, err := provider.GenerateSpeech(ctx, text)
	assert.NoError(t, err)
	assert.NotEmpty(t, audio)
}

func TestElevenLabsProvider_GenerateSpeech_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	config := &tts.Config{
		Provider: "elevenlabs",
		APIKey:   "test-key",
		Timeout:  30 * time.Second,
		BaseURL:  server.URL,
	}

	provider, err := NewElevenLabsProvider(config)
	require.NoError(t, err)

	elConfig := provider.(*ElevenLabsProvider).config
	elConfig.VoiceID = "test-voice-id"

	ctx := context.Background()
	text := "Hello, world!"

	_, err = provider.GenerateSpeech(ctx, text)
	assert.Error(t, err)
}

func TestElevenLabsProvider_GenerateSpeech_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &tts.Config{
		Provider: "elevenlabs",
		APIKey:   "test-key",
		Timeout:  30 * time.Second,
		BaseURL:  server.URL,
	}

	provider, err := NewElevenLabsProvider(config)
	require.NoError(t, err)

	elConfig := provider.(*ElevenLabsProvider).config
	elConfig.VoiceID = "test-voice-id"

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(5 * time.Millisecond)
		cancel()
	}()

	_, err = provider.GenerateSpeech(ctx, "test")
	assert.Error(t, err)
}

func TestElevenLabsProvider_StreamGenerate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		audioData := []byte{1, 2, 3, 4, 5}
		w.Header().Set("Content-Type", "audio/mpeg")
		w.WriteHeader(http.StatusOK)
		w.Write(audioData)
	}))
	defer server.Close()

	config := &tts.Config{
		Provider: "elevenlabs",
		APIKey:   "test-key",
		Timeout:  30 * time.Second,
		BaseURL:  server.URL,
	}

	provider, err := NewElevenLabsProvider(config)
	require.NoError(t, err)

	elConfig := provider.(*ElevenLabsProvider).config
	elConfig.VoiceID = "test-voice-id"

	ctx := context.Background()
	reader, err := provider.StreamGenerate(ctx, "test")
	assert.NoError(t, err)
	assert.NotNil(t, reader)

	audio, err := io.ReadAll(reader)
	assert.NoError(t, err)
	assert.NotEmpty(t, audio)
}

func TestDefaultElevenLabsConfig(t *testing.T) {
	config := DefaultElevenLabsConfig()
	assert.NotNil(t, config)
	assert.Equal(t, "eleven_monolingual_v1", config.ModelID)
	assert.Equal(t, 0.5, config.Stability)
	assert.Equal(t, 0.5, config.SimilarityBoost)
	assert.Equal(t, "mp3_44100_128", config.OutputFormat)
	assert.Equal(t, "https://api.elevenlabs.io", config.BaseURL)
}
