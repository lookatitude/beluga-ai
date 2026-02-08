package unstructured

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
