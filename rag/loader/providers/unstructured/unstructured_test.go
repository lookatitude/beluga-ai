package unstructured

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/rag/loader"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistration(t *testing.T) {
	names := loader.List()
	found := false
	for _, n := range names {
		if n == "unstructured" {
			found = true
			break
		}
	}
	if !found {
		t.Error("unstructured loader not registered")
	}
}

func TestNew(t *testing.T) {
	l, err := New(config.ProviderConfig{
		APIKey: "test-key",
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if l == nil {
		t.Fatal("expected non-nil loader")
	}
}

func TestLoad_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/general/v0/general" {
			t.Errorf("expected /general/v0/general, got %s", r.URL.Path)
		}
		if r.Header.Get("unstructured-api-key") != "test-key" {
			t.Errorf("missing or wrong API key header")
		}

		ct := r.Header.Get("Content-Type")
		if ct == "" {
			t.Error("missing Content-Type")
		}

		elements := []element{
			{Type: "Title", ElementID: "1", Text: "Document Title"},
			{Type: "NarrativeText", ElementID: "2", Text: "This is the body of the document."},
			{Type: "NarrativeText", ElementID: "3", Text: "Second paragraph."},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(elements)
	}))
	defer ts.Close()

	dir := t.TempDir()
	path := filepath.Join(dir, "test.pdf")
	if err := os.WriteFile(path, []byte("fake pdf content"), 0644); err != nil {
		t.Fatal(err)
	}

	l, err := New(config.ProviderConfig{
		APIKey:  "test-key",
		BaseURL: ts.URL,
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	docs, err := l.Load(context.Background(), path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 doc, got %d", len(docs))
	}
	if docs[0].ID != path {
		t.Errorf("ID = %q, want %q", docs[0].ID, path)
	}
	if docs[0].Metadata["loader"] != "unstructured" {
		t.Errorf("metadata loader = %v, want unstructured", docs[0].Metadata["loader"])
	}
	expected := "Document Title\n\nThis is the body of the document.\n\nSecond paragraph."
	if docs[0].Content != expected {
		t.Errorf("content = %q, want %q", docs[0].Content, expected)
	}
}

func TestLoad_EmptySource(t *testing.T) {
	l, err := New(config.ProviderConfig{APIKey: "test"})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	_, err = l.Load(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty source")
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	l, err := New(config.ProviderConfig{APIKey: "test"})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	_, err = l.Load(context.Background(), "/nonexistent/file.pdf")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoad_APIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"detail":"internal error"}`))
	}))
	defer ts.Close()

	dir := t.TempDir()
	path := filepath.Join(dir, "test.pdf")
	if err := os.WriteFile(path, []byte("fake"), 0644); err != nil {
		t.Fatal(err)
	}

	l, err := New(config.ProviderConfig{
		APIKey:  "test",
		BaseURL: ts.URL,
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	_, err = l.Load(context.Background(), path)
	if err == nil {
		t.Fatal("expected error for API failure")
	}
}

func TestLoad_EmptyElements(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]element{})
	}))
	defer ts.Close()

	dir := t.TempDir()
	path := filepath.Join(dir, "test.pdf")
	if err := os.WriteFile(path, []byte("fake"), 0644); err != nil {
		t.Fatal(err)
	}

	l, err := New(config.ProviderConfig{
		APIKey:  "test",
		BaseURL: ts.URL,
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	docs, err := l.Load(context.Background(), path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if docs != nil {
		t.Errorf("expected nil docs for empty elements, got %d", len(docs))
	}
}

func TestLoad_RegistryNew(t *testing.T) {
	l, err := loader.New("unstructured", config.ProviderConfig{
		APIKey: "test",
	})
	if err != nil {
		t.Fatalf("loader.New() error: %v", err)
	}
	if l == nil {
		t.Fatal("expected non-nil loader")
	}
}

func TestNew_DefaultBaseURL(t *testing.T) {
	l, err := New(config.ProviderConfig{APIKey: "test"})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if l.baseURL != defaultBaseURL {
		t.Errorf("baseURL = %q, want %q", l.baseURL, defaultBaseURL)
	}
}

func TestNew_BaseURLTrailingSlash(t *testing.T) {
	l, err := New(config.ProviderConfig{
		APIKey:  "test",
		BaseURL: "https://example.com/",
	})
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", l.baseURL, "trailing slash should be removed")
}

func TestNew_CustomTimeout(t *testing.T) {
	timeout := 30 * time.Second
	l, err := New(config.ProviderConfig{
		APIKey:  "test",
		Timeout: timeout,
	})
	require.NoError(t, err)
	assert.NotNil(t, l.client)
	assert.Equal(t, timeout, l.client.Timeout, "timeout should be set")
}

func TestLoad_NoAPIKey(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify that no API key header is sent
		if r.Header.Get("unstructured-api-key") != "" {
			t.Error("expected no API key header")
		}
		elements := []element{{Type: "Text", ElementID: "1", Text: "Content"}}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(elements)
	}))
	defer ts.Close()

	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	require.NoError(t, os.WriteFile(path, []byte("test"), 0644))

	l, err := New(config.ProviderConfig{BaseURL: ts.URL})
	require.NoError(t, err)

	docs, err := l.Load(context.Background(), path)
	require.NoError(t, err)
	assert.Len(t, docs, 1)
}

func TestLoad_ContextCancellation(t *testing.T) {
	// Create a slow server that will be interrupted
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	require.NoError(t, os.WriteFile(path, []byte("test"), 0644))

	l, err := New(config.ProviderConfig{BaseURL: ts.URL})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err = l.Load(ctx, path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unstructured: request")
}

func TestLoad_InvalidJSONResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{invalid json`))
	}))
	defer ts.Close()

	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	require.NoError(t, os.WriteFile(path, []byte("test"), 0644))

	l, err := New(config.ProviderConfig{BaseURL: ts.URL})
	require.NoError(t, err)

	_, err = l.Load(context.Background(), path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unstructured: decode response")
}

func TestLoad_ElementsWithEmptyText(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		elements := []element{
			{Type: "Title", ElementID: "1", Text: "First"},
			{Type: "Empty", ElementID: "2", Text: ""}, // empty text should be skipped
			{Type: "Text", ElementID: "3", Text: "Second"},
			{Type: "Empty2", ElementID: "4", Text: ""}, // empty text should be skipped
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(elements)
	}))
	defer ts.Close()

	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	require.NoError(t, os.WriteFile(path, []byte("test"), 0644))

	l, err := New(config.ProviderConfig{BaseURL: ts.URL})
	require.NoError(t, err)

	docs, err := l.Load(context.Background(), path)
	require.NoError(t, err)
	require.Len(t, docs, 1)
	assert.Equal(t, "First\n\nSecond", docs[0].Content, "empty text elements should be skipped")
}

func TestLoad_AllElementsEmptyText(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		elements := []element{
			{Type: "Empty", ElementID: "1", Text: ""},
			{Type: "Empty", ElementID: "2", Text: ""},
			{Type: "Empty", ElementID: "3", Text: ""},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(elements)
	}))
	defer ts.Close()

	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	require.NoError(t, os.WriteFile(path, []byte("test"), 0644))

	l, err := New(config.ProviderConfig{BaseURL: ts.URL})
	require.NoError(t, err)

	docs, err := l.Load(context.Background(), path)
	require.NoError(t, err)
	assert.Nil(t, docs, "should return nil when all elements have empty text")
}

func TestLoad_SingleElementNoSeparator(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		elements := []element{
			{Type: "Text", ElementID: "1", Text: "Only one element"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(elements)
	}))
	defer ts.Close()

	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	require.NoError(t, os.WriteFile(path, []byte("test"), 0644))

	l, err := New(config.ProviderConfig{BaseURL: ts.URL})
	require.NoError(t, err)

	docs, err := l.Load(context.Background(), path)
	require.NoError(t, err)
	require.Len(t, docs, 1)
	assert.Equal(t, "Only one element", docs[0].Content)
	assert.NotContains(t, docs[0].Content, "\n\n", "single element should not have separator")
}

func TestLoad_MetadataFields(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		elements := []element{
			{Type: "Text", ElementID: "1", Text: "Content"},
			{Type: "Text", ElementID: "2", Text: "More"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(elements)
	}))
	defer ts.Close()

	dir := t.TempDir()
	path := filepath.Join(dir, "document.pdf")
	require.NoError(t, os.WriteFile(path, []byte("test"), 0644))

	l, err := New(config.ProviderConfig{BaseURL: ts.URL})
	require.NoError(t, err)

	docs, err := l.Load(context.Background(), path)
	require.NoError(t, err)
	require.Len(t, docs, 1)

	doc := docs[0]
	assert.Equal(t, path, doc.ID)
	assert.Equal(t, "Content\n\nMore", doc.Content)
	assert.Equal(t, path, doc.Metadata["source"])
	assert.Equal(t, "unstructured", doc.Metadata["format"])
	assert.Equal(t, "unstructured", doc.Metadata["loader"])
	assert.Equal(t, "document.pdf", doc.Metadata["filename"])
	assert.Equal(t, 2, doc.Metadata["elements"])
}

func TestLoad_HTTPClientError(t *testing.T) {
	// Use an invalid URL to trigger HTTP client error
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	require.NoError(t, os.WriteFile(path, []byte("test"), 0644))

	l, err := New(config.ProviderConfig{
		BaseURL: "http://invalid-hostname-that-does-not-exist-12345.local",
	})
	require.NoError(t, err)

	// Set a short timeout to fail faster
	l.client.Timeout = 100 * time.Millisecond

	_, err = l.Load(context.Background(), path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unstructured: request")
}

func TestLoad_APIErrorMessageFormat(t *testing.T) {
	errorMsg := "Invalid API key"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(errorMsg))
	}))
	defer ts.Close()

	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	require.NoError(t, os.WriteFile(path, []byte("test"), 0644))

	l, err := New(config.ProviderConfig{BaseURL: ts.URL})
	require.NoError(t, err)

	_, err = l.Load(context.Background(), path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unstructured: API error")
	assert.Contains(t, err.Error(), "status 401")
	assert.Contains(t, err.Error(), errorMsg)
}

func TestLoad_TableDriven(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		elements    []element
		statusCode  int
		wantErr     bool
		errContains string
		wantDocs    int
		checkDoc    func(t *testing.T, doc schema.Document)
	}{
		{
			name:   "empty source path",
			source: "",
			wantErr: true,
			errContains: "source file path is required",
		},
		{
			name:   "nonexistent file",
			source: "/tmp/nonexistent-file-xyz-12345.pdf",
			wantErr: true,
			errContains: "open file",
		},
		{
			name:       "multiple elements",
			elements:   []element{{Text: "A"}, {Text: "B"}, {Text: "C"}},
			statusCode: http.StatusOK,
			wantDocs:   1,
			checkDoc: func(t *testing.T, doc schema.Document) {
				assert.Equal(t, "A\n\nB\n\nC", doc.Content)
			},
		},
		{
			name:       "first element empty",
			elements:   []element{{Text: ""}, {Text: "B"}, {Text: "C"}},
			statusCode: http.StatusOK,
			wantDocs:   1,
			checkDoc: func(t *testing.T, doc schema.Document) {
				assert.Equal(t, "B\n\nC", doc.Content)
				assert.False(t, strings.HasPrefix(doc.Content, "\n\n"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var serverCalled bool
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				serverCalled = true
				if tt.statusCode != 0 {
					w.WriteHeader(tt.statusCode)
				}
				if tt.elements != nil {
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(tt.elements)
				}
			}))
			defer ts.Close()

			l, err := New(config.ProviderConfig{BaseURL: ts.URL})
			require.NoError(t, err)

			// Create temp file if source is not provided
			source := tt.source
			if source == "" && !tt.wantErr {
				dir := t.TempDir()
				source = filepath.Join(dir, "test.txt")
				require.NoError(t, os.WriteFile(source, []byte("content"), 0644))
			} else if source != "" && source != "/tmp/nonexistent-file-xyz-12345.pdf" {
				dir := t.TempDir()
				actualPath := filepath.Join(dir, filepath.Base(source))
				require.NoError(t, os.WriteFile(actualPath, []byte("content"), 0644))
				source = actualPath
			}

			docs, err := l.Load(context.Background(), source)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			if tt.wantDocs == 0 {
				assert.Nil(t, docs)
			} else {
				assert.Len(t, docs, tt.wantDocs)
				if tt.checkDoc != nil && len(docs) > 0 {
					tt.checkDoc(t, docs[0])
				}
			}
			if tt.elements != nil {
				assert.True(t, serverCalled, "server should have been called")
			}
		})
	}
}
