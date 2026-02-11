package docling

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/rag/loader"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistration(t *testing.T) {
	names := loader.List()
	found := false
	for _, n := range names {
		if n == "docling" {
			found = true
			break
		}
	}
	if !found {
		t.Error("docling loader not registered")
	}
}

func TestNew(t *testing.T) {
	l, err := New(config.ProviderConfig{})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if l == nil {
		t.Fatal("expected non-nil loader")
	}
}

func TestLoad_FromFile(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/convert" {
			t.Errorf("expected /v1/convert, got %s", r.URL.Path)
		}

		resp := convertResponse{}
		resp.Document.Markdown = "# Converted Document\n\nExtracted content from PDF."
		resp.Status = "success"
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	dir := t.TempDir()
	path := filepath.Join(dir, "test.pdf")
	if err := os.WriteFile(path, []byte("fake pdf"), 0644); err != nil {
		t.Fatal(err)
	}

	l, err := New(config.ProviderConfig{BaseURL: ts.URL})
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
	if docs[0].Metadata["loader"] != "docling" {
		t.Errorf("metadata loader = %v, want docling", docs[0].Metadata["loader"])
	}
	expected := "# Converted Document\n\nExtracted content from PDF."
	if docs[0].Content != expected {
		t.Errorf("content = %q, want %q", docs[0].Content, expected)
	}
}

func TestLoad_FromURL(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json content type for URL source")
		}
		resp := convertResponse{}
		resp.Document.Markdown = "# Web Document"
		resp.Status = "success"
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	l, err := New(config.ProviderConfig{BaseURL: ts.URL})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	docs, err := l.Load(context.Background(), "https://example.com/doc.pdf")
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 doc, got %d", len(docs))
	}
	if docs[0].Content != "# Web Document" {
		t.Errorf("content = %q", docs[0].Content)
	}
}

func TestLoad_EmptySource(t *testing.T) {
	l, err := New(config.ProviderConfig{})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	_, err = l.Load(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty source")
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	l, err := New(config.ProviderConfig{})
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
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"bad request"}`))
	}))
	defer ts.Close()

	dir := t.TempDir()
	path := filepath.Join(dir, "test.pdf")
	if err := os.WriteFile(path, []byte("fake"), 0644); err != nil {
		t.Fatal(err)
	}

	l, err := New(config.ProviderConfig{BaseURL: ts.URL})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	_, err = l.Load(context.Background(), path)
	if err == nil {
		t.Fatal("expected error for API failure")
	}
}

func TestLoad_EmptyContent(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := convertResponse{}
		resp.Status = "success"
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	dir := t.TempDir()
	path := filepath.Join(dir, "test.pdf")
	if err := os.WriteFile(path, []byte("fake"), 0644); err != nil {
		t.Fatal(err)
	}

	l, err := New(config.ProviderConfig{BaseURL: ts.URL})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	docs, err := l.Load(context.Background(), path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if docs != nil {
		t.Errorf("expected nil docs for empty content, got %d", len(docs))
	}
}

func TestLoad_RegistryNew(t *testing.T) {
	l, err := loader.New("docling", config.ProviderConfig{})
	if err != nil {
		t.Fatalf("loader.New() error: %v", err)
	}
	if l == nil {
		t.Fatal("expected non-nil loader")
	}
}

func TestNew_DefaultBaseURL(t *testing.T) {
	l, err := New(config.ProviderConfig{})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if l.baseURL != defaultBaseURL {
		t.Errorf("baseURL = %q, want %q", l.baseURL, defaultBaseURL)
	}
}

func TestNew_WithTimeout(t *testing.T) {
	timeout := 30 * time.Second
	l, err := New(config.ProviderConfig{Timeout: timeout})
	require.NoError(t, err)
	require.NotNil(t, l)
	assert.Equal(t, timeout, l.client.Timeout)
}

func TestNew_BaseURLTrimming(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		expected string
	}{
		{
			name:     "with trailing slash",
			baseURL:  "http://localhost:5001/",
			expected: "http://localhost:5001",
		},
		{
			name:     "with multiple trailing slashes",
			baseURL:  "http://localhost:5001///",
			expected: "http://localhost:5001",
		},
		{
			name:     "without trailing slash",
			baseURL:  "http://localhost:5001",
			expected: "http://localhost:5001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l, err := New(config.ProviderConfig{BaseURL: tt.baseURL})
			require.NoError(t, err)
			assert.Equal(t, tt.expected, l.baseURL)
		})
	}
}

func TestLoad_WithAPIKey(t *testing.T) {
	apiKey := "test-api-key-12345"
	var receivedAuth string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		resp := convertResponse{}
		resp.Document.Markdown = "# Test Content"
		resp.Status = "success"
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	t.Run("URL source with API key", func(t *testing.T) {
		l, err := New(config.ProviderConfig{BaseURL: ts.URL, APIKey: apiKey})
		require.NoError(t, err)

		docs, err := l.Load(context.Background(), "https://example.com/doc.pdf")
		require.NoError(t, err)
		require.Len(t, docs, 1)
		assert.Equal(t, "Bearer "+apiKey, receivedAuth)
	})

	t.Run("file source with API key", func(t *testing.T) {
		receivedAuth = "" // reset
		dir := t.TempDir()
		path := filepath.Join(dir, "test.pdf")
		require.NoError(t, os.WriteFile(path, []byte("fake pdf"), 0644))

		l, err := New(config.ProviderConfig{BaseURL: ts.URL, APIKey: apiKey})
		require.NoError(t, err)

		docs, err := l.Load(context.Background(), path)
		require.NoError(t, err)
		require.Len(t, docs, 1)
		assert.Equal(t, "Bearer "+apiKey, receivedAuth)
	})
}

func TestLoad_FallbackToTextContent(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := convertResponse{}
		// No markdown, only text content
		resp.Document.Text = "Plain text content"
		resp.Status = "success"
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	l, err := New(config.ProviderConfig{BaseURL: ts.URL})
	require.NoError(t, err)

	docs, err := l.Load(context.Background(), "https://example.com/doc.pdf")
	require.NoError(t, err)
	require.Len(t, docs, 1)
	assert.Equal(t, "Plain text content", docs[0].Content)
}

func TestLoad_InvalidJSONResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{invalid json`))
	}))
	defer ts.Close()

	dir := t.TempDir()
	path := filepath.Join(dir, "test.pdf")
	require.NoError(t, os.WriteFile(path, []byte("fake pdf"), 0644))

	l, err := New(config.ProviderConfig{BaseURL: ts.URL})
	require.NoError(t, err)

	_, err = l.Load(context.Background(), path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode response")
}

func TestLoad_NetworkError(t *testing.T) {
	// Use a non-routable IP to simulate network error
	l, err := New(config.ProviderConfig{
		BaseURL: "http://192.0.2.1:1234", // TEST-NET-1, non-routable
		Timeout: 100 * time.Millisecond,
	})
	require.NoError(t, err)

	dir := t.TempDir()
	path := filepath.Join(dir, "test.pdf")
	require.NoError(t, os.WriteFile(path, []byte("fake pdf"), 0644))

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_, err = l.Load(ctx, path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "docling: request")
}

func TestLoad_ContextCancellation(t *testing.T) {
	// Server that delays to allow context cancellation
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done() // wait for context cancellation
	}))
	defer ts.Close()

	dir := t.TempDir()
	path := filepath.Join(dir, "test.pdf")
	require.NoError(t, os.WriteFile(path, []byte("fake pdf"), 0644))

	l, err := New(config.ProviderConfig{BaseURL: ts.URL})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err = l.Load(ctx, path)
	require.Error(t, err)
}

func TestLoad_MetadataFields(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := convertResponse{}
		resp.Document.Markdown = "# Test"
		resp.Status = "success"
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	l, err := New(config.ProviderConfig{BaseURL: ts.URL})
	require.NoError(t, err)

	source := "https://example.com/test.pdf"
	docs, err := l.Load(context.Background(), source)
	require.NoError(t, err)
	require.Len(t, docs, 1)

	meta := docs[0].Metadata
	assert.Equal(t, source, meta["source"])
	assert.Equal(t, "docling", meta["format"])
	assert.Equal(t, "docling", meta["loader"])
	assert.Equal(t, source, docs[0].ID)
}

func TestLoadFromURL_HTTPPrefix(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify it's using JSON body for http:// URLs
		contentType := r.Header.Get("Content-Type")
		assert.Equal(t, "application/json", contentType)

		var req convertRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(req.Source, "http://"))

		resp := convertResponse{}
		resp.Document.Markdown = "# HTTP Document"
		resp.Status = "success"
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	l, err := New(config.ProviderConfig{BaseURL: ts.URL})
	require.NoError(t, err)

	docs, err := l.Load(context.Background(), "http://example.com/doc.pdf")
	require.NoError(t, err)
	require.Len(t, docs, 1)
	assert.Equal(t, "# HTTP Document", docs[0].Content)
}

func TestLoadFromFile_MultipartUpload(t *testing.T) {
	var receivedFilename string
	var receivedContentType string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedContentType = r.Header.Get("Content-Type")
		assert.True(t, strings.HasPrefix(receivedContentType, "multipart/form-data"))

		// Parse multipart form
		err := r.ParseMultipartForm(10 << 20) // 10 MB
		require.NoError(t, err)
		defer r.MultipartForm.RemoveAll()

		file, header, err := r.FormFile("file")
		require.NoError(t, err)
		defer file.Close()

		receivedFilename = header.Filename

		// Read file content
		content, err := io.ReadAll(file)
		require.NoError(t, err)
		assert.Equal(t, []byte("fake pdf content"), content)

		resp := convertResponse{}
		resp.Document.Markdown = "# Uploaded Document"
		resp.Status = "success"
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	dir := t.TempDir()
	path := filepath.Join(dir, "test-doc.pdf")
	require.NoError(t, os.WriteFile(path, []byte("fake pdf content"), 0644))

	l, err := New(config.ProviderConfig{BaseURL: ts.URL})
	require.NoError(t, err)

	docs, err := l.Load(context.Background(), path)
	require.NoError(t, err)
	require.Len(t, docs, 1)
	assert.Equal(t, "test-doc.pdf", receivedFilename)
	assert.Equal(t, "# Uploaded Document", docs[0].Content)
}

func TestLoad_VariousHTTPStatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    bool
		errContain string
	}{
		{
			name:       "400 Bad Request",
			statusCode: http.StatusBadRequest,
			body:       `{"error":"invalid file format"}`,
			wantErr:    true,
			errContain: "API error (status 400)",
		},
		{
			name:       "401 Unauthorized",
			statusCode: http.StatusUnauthorized,
			body:       `{"error":"invalid API key"}`,
			wantErr:    true,
			errContain: "API error (status 401)",
		},
		{
			name:       "500 Internal Server Error",
			statusCode: http.StatusInternalServerError,
			body:       `{"error":"server error"}`,
			wantErr:    true,
			errContain: "API error (status 500)",
		},
		{
			name:       "503 Service Unavailable",
			statusCode: http.StatusServiceUnavailable,
			body:       `{"error":"service temporarily unavailable"}`,
			wantErr:    true,
			errContain: "API error (status 503)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.body))
			}))
			defer ts.Close()

			dir := t.TempDir()
			path := filepath.Join(dir, "test.pdf")
			require.NoError(t, os.WriteFile(path, []byte("fake"), 0644))

			l, err := New(config.ProviderConfig{BaseURL: ts.URL})
			require.NoError(t, err)

			_, err = l.Load(context.Background(), path)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContain)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
