package github

import (
	"context"
	"encoding/base64"
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
		if n == "github" {
			found = true
			break
		}
	}
	if !found {
		t.Error("github loader not registered")
	}
}

func TestNew(t *testing.T) {
	t.Run("success without key", func(t *testing.T) {
		l, err := New(config.ProviderConfig{})
		require.NoError(t, err)
		assert.NotNil(t, l)
	})

	t.Run("with api key", func(t *testing.T) {
		l, err := New(config.ProviderConfig{APIKey: "ghp-test"})
		require.NoError(t, err)
		assert.NotNil(t, l)
	})
}

func TestLoad(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		encoded := base64.StdEncoding.EncodeToString([]byte("package main\n\nfunc main() {}"))
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			assert.Equal(t, "/repos/owner/repo/contents/cmd/main.go", r.URL.Path)

			resp := contentResponse{
				Name:     "main.go",
				Path:     "cmd/main.go",
				SHA:      "abc123",
				Size:     28,
				Type:     "file",
				Content:  encoded,
				Encoding: "base64",
				HTMLURL:  "https://github.com/owner/repo/blob/main/cmd/main.go",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer srv.Close()

		l, err := New(config.ProviderConfig{BaseURL: srv.URL})
		require.NoError(t, err)

		docs, err := l.Load(context.Background(), "owner/repo/cmd/main.go")
		require.NoError(t, err)
		require.Len(t, docs, 1)
		assert.Contains(t, docs[0].Content, "package main")
		assert.Equal(t, "github", docs[0].Metadata["loader"])
		assert.Equal(t, "abc123", docs[0].Metadata["sha"])
	})

	t.Run("with ref", func(t *testing.T) {
		encoded := base64.StdEncoding.EncodeToString([]byte("content"))
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "v1.0", r.URL.Query().Get("ref"))
			resp := contentResponse{Name: "f", Path: "f", Type: "file", Content: encoded, Encoding: "base64"}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer srv.Close()

		l, err := New(config.ProviderConfig{
			BaseURL: srv.URL,
			Options: map[string]any{"ref": "v1.0"},
		})
		require.NoError(t, err)

		docs, err := l.Load(context.Background(), "o/r/f")
		require.NoError(t, err)
		require.Len(t, docs, 1)
	})

	t.Run("empty source", func(t *testing.T) {
		l, err := New(config.ProviderConfig{})
		require.NoError(t, err)

		_, err = l.Load(context.Background(), "")
		assert.Error(t, err)
	})

	t.Run("invalid source format", func(t *testing.T) {
		l, err := New(config.ProviderConfig{})
		require.NoError(t, err)

		_, err = l.Load(context.Background(), "owner/repo")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "format")
	})

	t.Run("directory not file", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := contentResponse{Name: "dir", Path: "dir", Type: "dir"}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer srv.Close()

		l, err := New(config.ProviderConfig{BaseURL: srv.URL})
		require.NoError(t, err)

		_, err = l.Load(context.Background(), "o/r/dir")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "dir")
	})

	t.Run("server error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"message":"Not Found"}`))
		}))
		defer srv.Close()

		l, err := New(config.ProviderConfig{BaseURL: srv.URL})
		require.NoError(t, err)

		_, err = l.Load(context.Background(), "o/r/missing.go")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "github")
	})

	t.Run("auth header", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "Bearer ghp-test", r.Header.Get("Authorization"))
			encoded := base64.StdEncoding.EncodeToString([]byte("x"))
			resp := contentResponse{Name: "f", Path: "f", Type: "file", Content: encoded, Encoding: "base64"}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer srv.Close()

		l, err := New(config.ProviderConfig{BaseURL: srv.URL, APIKey: "ghp-test"})
		require.NoError(t, err)

		docs, err := l.Load(context.Background(), "o/r/f")
		require.NoError(t, err)
		assert.Len(t, docs, 1)
	})
}

func TestRegistryNew(t *testing.T) {
	l, err := loader.New("github", config.ProviderConfig{})
	require.NoError(t, err)
	assert.NotNil(t, l)
}
