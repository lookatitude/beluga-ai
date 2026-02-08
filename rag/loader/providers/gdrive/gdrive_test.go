package gdrive

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/rag/loader"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistration(t *testing.T) {
	names := loader.List()
	found := false
	for _, n := range names {
		if n == "gdrive" {
			found = true
			break
		}
	}
	if !found {
		t.Error("gdrive loader not registered")
	}
}

func TestNew(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		l, err := New(config.ProviderConfig{APIKey: "test-token"})
		require.NoError(t, err)
		assert.NotNil(t, l)
	})

	t.Run("missing api key", func(t *testing.T) {
		_, err := New(config.ProviderConfig{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "API key")
	})
}

func TestLoad(t *testing.T) {
	t.Run("regular file", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/drive/v3/files/file123" {
				if r.URL.Query().Get("alt") == "media" {
					w.Write([]byte("file content here"))
					return
				}
				meta := fileMetadata{
					ID:       "file123",
					Name:     "readme.txt",
					MimeType: "text/plain",
					Size:     "17",
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(meta)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()

		l, err := New(config.ProviderConfig{APIKey: "test", BaseURL: srv.URL})
		require.NoError(t, err)

		docs, err := l.Load(context.Background(), "file123")
		require.NoError(t, err)
		require.Len(t, docs, 1)
		assert.Equal(t, "file content here", docs[0].Content)
		assert.Equal(t, "gdrive", docs[0].Metadata["loader"])
		assert.Equal(t, "readme.txt", docs[0].Metadata["file_name"])
	})

	t.Run("google doc export", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/drive/v3/files/doc1" {
				meta := fileMetadata{
					ID:       "doc1",
					Name:     "My Document",
					MimeType: "application/vnd.google-apps.document",
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(meta)
				return
			}
			if r.URL.Path == "/drive/v3/files/doc1/export" {
				assert.Equal(t, "text/plain", r.URL.Query().Get("mimeType"))
				w.Write([]byte("exported text content"))
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()

		l, err := New(config.ProviderConfig{APIKey: "test", BaseURL: srv.URL})
		require.NoError(t, err)

		docs, err := l.Load(context.Background(), "doc1")
		require.NoError(t, err)
		require.Len(t, docs, 1)
		assert.Equal(t, "exported text content", docs[0].Content)
	})

	t.Run("empty source", func(t *testing.T) {
		l, err := New(config.ProviderConfig{APIKey: "test"})
		require.NoError(t, err)

		_, err = l.Load(context.Background(), "")
		assert.Error(t, err)
	})

	t.Run("server error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":{"message":"not found"}}`))
		}))
		defer srv.Close()

		l, err := New(config.ProviderConfig{APIKey: "test", BaseURL: srv.URL})
		require.NoError(t, err)

		_, err = l.Load(context.Background(), "missing")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "gdrive")
	})
}

func TestIsGoogleDoc(t *testing.T) {
	assert.True(t, isGoogleDoc("application/vnd.google-apps.document"))
	assert.True(t, isGoogleDoc("application/vnd.google-apps.spreadsheet"))
	assert.False(t, isGoogleDoc("text/plain"))
	assert.False(t, isGoogleDoc("application/pdf"))
}

func TestRegistryNew(t *testing.T) {
	l, err := loader.New("gdrive", config.ProviderConfig{APIKey: "test"})
	require.NoError(t, err)
	assert.NotNil(t, l)
}
