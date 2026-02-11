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

	t.Run("all block types", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if r.URL.Path == "/v1/pages/alltypes" {
				json.NewEncoder(w).Encode(pageResponse{ID: "alltypes"})
				return
			}
			blocks := blockChildren{
				Results: []block{
					{ID: "h2", Type: "heading_2", Heading2: &richTextBlock{RichText: []richText{{PlainText: "H2 Title"}}}},
					{ID: "h3", Type: "heading_3", Heading3: &richTextBlock{RichText: []richText{{PlainText: "H3 Title"}}}},
					{ID: "bullet", Type: "bulleted_list_item", BulletedList: &richTextBlock{RichText: []richText{{PlainText: "Bullet point"}}}},
					{ID: "number", Type: "numbered_list_item", NumberedList: &richTextBlock{RichText: []richText{{PlainText: "Numbered item"}}}},
					{ID: "toggle", Type: "toggle", Toggle: &richTextBlock{RichText: []richText{{PlainText: "Toggle content"}}}},
					{ID: "quote", Type: "quote", Quote: &richTextBlock{RichText: []richText{{PlainText: "Quote text"}}}},
					{ID: "callout", Type: "callout", Callout: &richTextBlock{RichText: []richText{{PlainText: "Callout info"}}}},
					{ID: "unknown", Type: "unsupported_type"},
				},
			}
			json.NewEncoder(w).Encode(blocks)
		}))
		defer srv.Close()

		l, err := New(config.ProviderConfig{APIKey: "test", BaseURL: srv.URL})
		require.NoError(t, err)

		docs, err := l.Load(context.Background(), "alltypes")
		require.NoError(t, err)
		require.Len(t, docs, 1)
		assert.Contains(t, docs[0].Content, "H2 Title")
		assert.Contains(t, docs[0].Content, "H3 Title")
		assert.Contains(t, docs[0].Content, "Bullet point")
		assert.Contains(t, docs[0].Content, "Numbered item")
		assert.Contains(t, docs[0].Content, "Toggle content")
		assert.Contains(t, docs[0].Content, "Quote text")
		assert.Contains(t, docs[0].Content, "Callout info")
		assert.NotContains(t, docs[0].Content, "unsupported_type")
	})

	t.Run("page id with dashes", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			// Verify dashes are stripped
			if r.URL.Path == "/v1/pages/abc123def456" {
				json.NewEncoder(w).Encode(pageResponse{ID: "abc123def456"})
				return
			}
			if r.URL.Path == "/v1/blocks/abc123def456/children" {
				json.NewEncoder(w).Encode(blockChildren{
					Results: []block{
						{ID: "p1", Type: "paragraph", Paragraph: &richTextBlock{RichText: []richText{{PlainText: "Test"}}}},
					},
				})
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()

		l, err := New(config.ProviderConfig{APIKey: "test", BaseURL: srv.URL})
		require.NoError(t, err)

		// Load with dashed page ID
		docs, err := l.Load(context.Background(), "abc123-def456")
		require.NoError(t, err)
		require.Len(t, docs, 1)
		assert.Equal(t, "abc123def456", docs[0].ID)
	})

	t.Run("blocks fetch error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if r.URL.Path == "/v1/pages/test123" {
				json.NewEncoder(w).Encode(pageResponse{ID: "test123"})
				return
			}
			// Fail on blocks fetch
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"failed to fetch blocks"}`))
		}))
		defer srv.Close()

		l, err := New(config.ProviderConfig{APIKey: "test", BaseURL: srv.URL})
		require.NoError(t, err)

		_, err = l.Load(context.Background(), "test123")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "fetch blocks")
	})
}

func TestRegistryNew(t *testing.T) {
	l, err := loader.New("notion", config.ProviderConfig{APIKey: "test"})
	require.NoError(t, err)
	assert.NotNil(t, l)
}

func TestExtractTitle(t *testing.T) {
	tests := []struct {
		name string
		page pageResponse
		want string
	}{
		{
			name: "has title",
			page: pageResponse{
				Properties: map[string]property{
					"Name": {Type: "title", Title: []richText{{PlainText: "My Title"}}},
				},
			},
			want: "My Title",
		},
		{
			name: "no title property",
			page: pageResponse{
				Properties: map[string]property{
					"Other": {Type: "text"},
				},
			},
			want: "",
		},
		{
			name: "empty title array",
			page: pageResponse{
				Properties: map[string]property{
					"Name": {Type: "title", Title: []richText{}},
				},
			},
			want: "",
		},
		{
			name: "empty properties",
			page: pageResponse{Properties: map[string]property{}},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTitle(tt.page)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtractContent(t *testing.T) {
	tests := []struct {
		name   string
		blocks []block
		want   string
	}{
		{
			name: "multiple blocks",
			blocks: []block{
				{ID: "1", Type: "paragraph", Paragraph: &richTextBlock{RichText: []richText{{PlainText: "First"}}}},
				{ID: "2", Type: "paragraph", Paragraph: &richTextBlock{RichText: []richText{{PlainText: "Second"}}}},
			},
			want: "First\n\nSecond",
		},
		{
			name:   "empty blocks",
			blocks: []block{},
			want:   "",
		},
		{
			name: "blocks with empty text",
			blocks: []block{
				{ID: "1", Type: "paragraph", Paragraph: &richTextBlock{RichText: []richText{}}},
				{ID: "2", Type: "unsupported"},
			},
			want: "",
		},
		{
			name: "mixed blocks",
			blocks: []block{
				{ID: "1", Type: "paragraph", Paragraph: &richTextBlock{RichText: []richText{{PlainText: "Content"}}}},
				{ID: "2", Type: "unsupported"},
				{ID: "3", Type: "paragraph", Paragraph: &richTextBlock{RichText: []richText{{PlainText: "More"}}}},
			},
			want: "Content\n\nMore",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractContent(tt.blocks)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBlockText(t *testing.T) {
	tests := []struct {
		name  string
		block block
		want  string
	}{
		{
			name:  "paragraph",
			block: block{Type: "paragraph", Paragraph: &richTextBlock{RichText: []richText{{PlainText: "Para text"}}}},
			want:  "Para text",
		},
		{
			name:  "heading_1",
			block: block{Type: "heading_1", Heading1: &richTextBlock{RichText: []richText{{PlainText: "H1"}}}},
			want:  "H1",
		},
		{
			name:  "heading_2",
			block: block{Type: "heading_2", Heading2: &richTextBlock{RichText: []richText{{PlainText: "H2"}}}},
			want:  "H2",
		},
		{
			name:  "heading_3",
			block: block{Type: "heading_3", Heading3: &richTextBlock{RichText: []richText{{PlainText: "H3"}}}},
			want:  "H3",
		},
		{
			name:  "bulleted_list_item",
			block: block{Type: "bulleted_list_item", BulletedList: &richTextBlock{RichText: []richText{{PlainText: "Bullet"}}}},
			want:  "Bullet",
		},
		{
			name:  "numbered_list_item",
			block: block{Type: "numbered_list_item", NumberedList: &richTextBlock{RichText: []richText{{PlainText: "Number"}}}},
			want:  "Number",
		},
		{
			name:  "toggle",
			block: block{Type: "toggle", Toggle: &richTextBlock{RichText: []richText{{PlainText: "Toggle"}}}},
			want:  "Toggle",
		},
		{
			name:  "quote",
			block: block{Type: "quote", Quote: &richTextBlock{RichText: []richText{{PlainText: "Quote"}}}},
			want:  "Quote",
		},
		{
			name:  "callout",
			block: block{Type: "callout", Callout: &richTextBlock{RichText: []richText{{PlainText: "Callout"}}}},
			want:  "Callout",
		},
		{
			name:  "code",
			block: block{Type: "code", Code: &codeBlock{RichText: []richText{{PlainText: "code"}}, Language: "go"}},
			want:  "code",
		},
		{
			name:  "code with nil block",
			block: block{Type: "code", Code: nil},
			want:  "",
		},
		{
			name:  "unsupported type",
			block: block{Type: "unsupported"},
			want:  "",
		},
		{
			name:  "nil rich text block",
			block: block{Type: "paragraph", Paragraph: nil},
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := blockText(tt.block)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRichTextToPlain(t *testing.T) {
	tests := []struct {
		name string
		rts  []richText
		want string
	}{
		{
			name: "single element",
			rts:  []richText{{PlainText: "hello"}},
			want: "hello",
		},
		{
			name: "multiple elements",
			rts:  []richText{{PlainText: "hello "}, {PlainText: "world"}},
			want: "hello world",
		},
		{
			name: "empty array",
			rts:  []richText{},
			want: "",
		},
		{
			name: "empty text elements",
			rts:  []richText{{PlainText: ""}, {PlainText: ""}},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := richTextToPlain(tt.rts)
			assert.Equal(t, tt.want, got)
		})
	}
}
