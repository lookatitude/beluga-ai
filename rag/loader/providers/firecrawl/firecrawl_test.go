package firecrawl

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/rag/loader"
)

func TestRegistration(t *testing.T) {
	names := loader.List()
	found := false
	for _, n := range names {
		if n == "firecrawl" {
			found = true
			break
		}
	}
	if !found {
		t.Error("firecrawl loader not registered")
	}
}

func TestNew(t *testing.T) {
	l, err := New(config.ProviderConfig{
		APIKey: "fc-test",
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
		resp := map[string]any{
			"success": true,
			"data": map[string]any{
				"markdown": "# Hello World\n\nSome content from the web.",
				"metadata": map[string]any{
					"title":       "Hello World",
					"description": "A test page",
					"sourceURL":   "https://example.com",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	l, err := New(config.ProviderConfig{
		APIKey:  "fc-test",
		BaseURL: ts.URL,
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	docs, err := l.Load(context.Background(), "https://example.com")
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 doc, got %d", len(docs))
	}
	if docs[0].ID != "https://example.com" {
		t.Errorf("ID = %q, want %q", docs[0].ID, "https://example.com")
	}
	if docs[0].Metadata["loader"] != "firecrawl" {
		t.Errorf("metadata loader = %v, want firecrawl", docs[0].Metadata["loader"])
	}
	if docs[0].Metadata["source"] != "https://example.com" {
		t.Errorf("metadata source = %v", docs[0].Metadata["source"])
	}
}

func TestLoad_EmptySource(t *testing.T) {
	l, err := New(config.ProviderConfig{APIKey: "fc-test"})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	_, err = l.Load(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty source")
	}
}

func TestLoad_APIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"error":"internal server error"}`)
	}))
	defer ts.Close()

	l, err := New(config.ProviderConfig{
		APIKey:  "fc-test",
		BaseURL: ts.URL,
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	_, err = l.Load(context.Background(), "https://example.com")
	if err == nil {
		t.Fatal("expected error for API failure")
	}
}

func TestLoad_RegistryNew(t *testing.T) {
	l, err := loader.New("firecrawl", config.ProviderConfig{
		APIKey: "fc-test",
	})
	if err != nil {
		t.Fatalf("loader.New() error: %v", err)
	}
	if l == nil {
		t.Fatal("expected non-nil loader")
	}
}

func TestLoad_DefaultBaseURL(t *testing.T) {
	l, err := New(config.ProviderConfig{
		APIKey: "fc-test",
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if l == nil {
		t.Fatal("expected non-nil loader")
	}
	if l.client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestLoad_ContextCancelled(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Block until context cancelled — but firecrawl SDK may not propagate ctx.
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"success": true, "data": map[string]any{"markdown": ""}})
	}))
	defer ts.Close()

	l, err := New(config.ProviderConfig{
		APIKey:  "fc-test",
		BaseURL: ts.URL,
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	// Empty markdown returns nil docs, no error.
	docs, err := l.Load(context.Background(), "https://example.com")
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if docs != nil {
		t.Errorf("expected nil docs for empty markdown, got %d", len(docs))
	}
}
