package elevenlabs

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lookatitude/beluga-ai/v2/voice/tts"
)

// FI-44: a voice ID must not be able to change the request path.
func TestSynthesize_EscapesVoiceID(t *testing.T) {
	var rawPath, query string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawPath = r.URL.EscapedPath()
		query = r.URL.RawQuery
		_, _ = w.Write([]byte("mp3"))
	}))
	defer srv.Close()

	e, err := New(tts.Config{Voice: "../user/subscription?x=1", Extra: map[string]any{"api_key": "k", "base_url": srv.URL}})
	require.NoError(t, err)
	_, err = e.Synthesize(context.Background(), "hi")
	require.NoError(t, err)
	assert.Equal(t, "/text-to-speech/..%2Fuser%2Fsubscription%3Fx=1", rawPath)
	assert.Empty(t, query)
}

// FI-32: the deprecated eleven_monolingual_v1 model is no longer the default.
func TestDefaultModelNotDeprecated(t *testing.T) {
	assert.Equal(t, "eleven_multilingual_v2", defaultModel)
}
