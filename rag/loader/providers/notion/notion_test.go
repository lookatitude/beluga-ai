package notion

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
		if n == "notion" {
			found = true
			break
		}
	}
	if !found {
		t.Error("notion loader not registered")
	}
}

func TestNew(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		l, err := New(config.ProviderConfig{APIKey: "ntn-test"})
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
	t.Run("success", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "2022-06-28", r.Header.Get("Notion-Version"))
			assert.Contains(t, r.Header.Get("Authorization"), "Bearer ")

			w.Header().Set("Content-Type", "application/json")

			if r.URL.Path == "/v1/pages/abc123" {
				page := pageResponse{
					ID: "abc123",
					Properties: map[string]property{
						"Name": {Type: "title", Title: []richText{{PlainText: "Test Page"}}},
					},
				}
				json.NewEncoder(w).Encode(page)
				return
			}

			if r.URL.Path == "/v1/blocks/abc123/children" {
				blocks := blockChildren{
					Results: []block{
						{
							ID:   "b1",
							Type: "heading_1",
							Heading1: &richTextBlock{
								RichText: []richText{{PlainText: "Title"}},
							},
						},
						{
							ID:   "b2",
							Type: "paragraph",
							Paragraph: &richTextBlock{
								RichText: []richText{{PlainText: "Hello World"}},
							},
						},
					},
				}
				json.NewEncoder(w).Encode(blocks)
				return
			}

			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()

		l, err := New(config.ProviderConfig{
			APIKey:  "ntn-test",
			BaseURL: srv.URL,
		})
		require.NoError(t, err)

		docs, err := l.Load(context.Background(), "abc123")
		require.NoError(t, err)
		require.Len(t, docs, 1)
		assert.Equal(t, "abc123", docs[0].ID)
		assert.Contains(t, docs[0].Content, "Title")
		assert.Contains(t, docs[0].Content, "Hello World")
		assert.Equal(t, "notion", docs[0].Metadata["loader"])
		assert.Equal(t, "Test Page", docs[0].Metadata["title"])
	})

	t.Run("empty source", func(t *testing.T) {
		l, err := New(config.ProviderConfig{APIKey: "test"})
		require.NoError(t, err)

		_, err = l.Load(context.Background(), "")
		assert.Error(t, err)
	})

	t.Run("empty blocks", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if r.URL.Path == "/v1/pages/empty" {
				json.NewEncoder(w).Encode(pageResponse{ID: "empty"})
				return
			}
			json.NewEncoder(w).Encode(blockChildren{})
		}))
		defer srv.Close()

		l, err := New(config.ProviderConfig{APIKey: "test", BaseURL: srv.URL})
		require.NoError(t, err)

		docs, err := l.Load(context.Background(), "empty")
		require.NoError(t, err)
		assert.Nil(t, docs)
	})

	t.Run("server error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"failed"}`))
		}))
		defer srv.Close()

		l, err := New(config.ProviderConfig{APIKey: "test", BaseURL: srv.URL})
		require.NoError(t, err)

		_, err = l.Load(context.Background(), "abc")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "notion")
	})

	t.Run("code block", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if r.URL.Path == "/v1/pages/code1" {
				json.NewEncoder(w).Encode(pageResponse{ID: "code1"})
				return
			}
			blocks := blockChildren{
				Results: []block{
					{
						ID:   "c1",
						Type: "code",
						Code: &codeBlock{
							RichText: []richText{{PlainText: "fmt.Println(\"hello\")"}},
							Language: "go",
						},
					},
				},
			}
			json.NewEncoder(w).Encode(blocks)
		}))
		defer srv.Close()

		l, err := New(config.ProviderConfig{APIKey: "test", BaseURL: srv.URL})
		require.NoError(t, err)

		docs, err := l.Load(context.Background(), "code1")
		require.NoError(t, err)
		require.Len(t, docs, 1)
		assert.Contains(t, docs[0].Content, "fmt.Println")
	})
}

func TestRegistryNew(t *testing.T) {
	l, err := loader.New("notion", config.ProviderConfig{APIKey: "test"})
	require.NoError(t, err)
	assert.NotNil(t, l)
}
