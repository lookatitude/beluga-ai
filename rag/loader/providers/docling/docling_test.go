package docling

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/rag/loader"
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
