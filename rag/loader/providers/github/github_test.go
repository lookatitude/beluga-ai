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

func TestLoad_EmptyContent(t *testing.T) {
	// Test that empty content returns nil documents.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := contentResponse{
			Name:     "empty.txt",
			Path:     "empty.txt",
			Type:     "file",
			Content:  "", // Empty content
			Encoding: "base64",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	l, err := New(config.ProviderConfig{BaseURL: srv.URL})
	require.NoError(t, err)

	docs, err := l.Load(context.Background(), "o/r/empty.txt")
	require.NoError(t, err)
	assert.Nil(t, docs, "Empty content should return nil documents")
}

func TestDecodeContent(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		encoding string
		want     string
		wantErr  bool
	}{
		{
			name:     "base64 encoded",
			content:  base64.StdEncoding.EncodeToString([]byte("hello world")),
			encoding: "base64",
			want:     "hello world",
			wantErr:  false,
		},
		{
			name:     "base64 with newlines",
			content:  base64.StdEncoding.EncodeToString([]byte("hello")) + "\n" + base64.StdEncoding.EncodeToString([]byte(" world")),
			encoding: "base64",
			want:     "",
			wantErr:  true, // Invalid base64 after newline removal
		},
		{
			name:     "empty encoding defaults to base64",
			content:  base64.StdEncoding.EncodeToString([]byte("test")),
			encoding: "",
			want:     "test",
			wantErr:  false,
		},
		{
			name:     "non-base64 encoding returns as-is",
			content:  "plain text content",
			encoding: "plain",
			want:     "plain text content",
			wantErr:  false,
		},
		{
			name:     "invalid base64",
			content:  "not-valid-base64!@#",
			encoding: "base64",
			want:     "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeContent(tt.content, tt.encoding)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestLoad_DecodeError(t *testing.T) {
	// Test that decoding errors are propagated.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := contentResponse{
			Name:     "file.txt",
			Path:     "file.txt",
			Type:     "file",
			Content:  "invalid-base64!@#$%",
			Encoding: "base64",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	l, err := New(config.ProviderConfig{BaseURL: srv.URL})
	require.NoError(t, err)

	_, err = l.Load(context.Background(), "o/r/file.txt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decode")
}
