package confluence

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
		if n == "confluence" {
			found = true
			break
		}
	}
	if !found {
		t.Error("confluence loader not registered")
	}
}

func TestNew(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		l, err := New(config.ProviderConfig{
			APIKey:  "test-token",
			BaseURL: "https://test.atlassian.net/wiki",
		})
		require.NoError(t, err)
		assert.NotNil(t, l)
	})

	t.Run("missing base url", func(t *testing.T) {
		_, err := New(config.ProviderConfig{APIKey: "test"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "base URL")
	})

	t.Run("missing api key", func(t *testing.T) {
		_, err := New(config.ProviderConfig{BaseURL: "https://test.atlassian.net/wiki"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "API key")
	})
}

func TestLoad(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			assert.Contains(t, r.URL.Path, "/rest/api/content/12345")
			assert.Contains(t, r.URL.RawQuery, "expand=body.storage,space")

			resp := pageResponse{
				ID:    "12345",
				Title: "Test Page",
			}
			resp.Body.Storage.Value = "<p>Hello <b>World</b></p>"
			resp.Space.Key = "TEAM"

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer srv.Close()

		l, err := New(config.ProviderConfig{
			APIKey:  "test-token",
			BaseURL: srv.URL,
		})
		require.NoError(t, err)

		docs, err := l.Load(context.Background(), "12345")
		require.NoError(t, err)
		require.Len(t, docs, 1)
		assert.Equal(t, "12345", docs[0].ID)
		assert.Equal(t, "Hello World", docs[0].Content)
		assert.Equal(t, "confluence", docs[0].Metadata["loader"])
		assert.Equal(t, "Test Page", docs[0].Metadata["title"])
		assert.Equal(t, "TEAM", docs[0].Metadata["space"])
	})

	t.Run("space/page-id format", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Contains(t, r.URL.Path, "/rest/api/content/67890")
			resp := pageResponse{ID: "67890", Title: "Page"}
			resp.Body.Storage.Value = "<p>content</p>"
			resp.Space.Key = "DEV"
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer srv.Close()

		l, err := New(config.ProviderConfig{APIKey: "test", BaseURL: srv.URL})
		require.NoError(t, err)

		docs, err := l.Load(context.Background(), "DEV/67890")
		require.NoError(t, err)
		require.Len(t, docs, 1)
	})

	t.Run("empty source", func(t *testing.T) {
		l, err := New(config.ProviderConfig{APIKey: "test", BaseURL: "http://localhost"})
		require.NoError(t, err)

		_, err = l.Load(context.Background(), "")
		assert.Error(t, err)
	})

	t.Run("empty content", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := pageResponse{ID: "1", Title: "Empty"}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer srv.Close()

		l, err := New(config.ProviderConfig{APIKey: "test", BaseURL: srv.URL})
		require.NoError(t, err)

		docs, err := l.Load(context.Background(), "1")
		require.NoError(t, err)
		assert.Nil(t, docs)
	})

	t.Run("server error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"internal error"}`))
		}))
		defer srv.Close()

		l, err := New(config.ProviderConfig{APIKey: "test", BaseURL: srv.URL})
		require.NoError(t, err)

		_, err = l.Load(context.Background(), "1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "confluence")
	})
}

func TestStripHTML(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"<p>Hello</p>", "Hello"},
		{"<b>Bold</b> and <i>italic</i>", "Bold and italic"},
		{"No tags here", "No tags here"},
		{"<div><p>Nested</p></div>", "Nested"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, stripHTML(tt.input))
	}
}
