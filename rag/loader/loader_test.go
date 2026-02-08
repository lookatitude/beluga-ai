package loader

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/schema"
)

func TestRegistry(t *testing.T) {
	names := List()
	expected := []string{"csv", "json", "markdown", "text"}
	if len(names) != len(expected) {
		t.Fatalf("expected %d loaders, got %d: %v", len(expected), len(names), names)
	}
	for i, name := range expected {
		if names[i] != name {
			t.Errorf("expected %q at index %d, got %q", name, i, names[i])
		}
	}
}

func TestNew_Unknown(t *testing.T) {
	_, err := New("nonexistent", config.ProviderConfig{})
	if err == nil {
		t.Fatal("expected error for unknown loader")
	}
}

func TestNew_Text(t *testing.T) {
	l, err := New("text", config.ProviderConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if l == nil {
		t.Fatal("expected non-nil loader")
	}
}

func TestTextLoader(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	content := "Hello, World!\nLine 2."
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	l := NewTextLoader()
	docs, err := l.Load(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 doc, got %d", len(docs))
	}
	if docs[0].Content != content {
		t.Errorf("content mismatch: got %q", docs[0].Content)
	}
	if docs[0].ID != path {
		t.Errorf("expected ID %q, got %q", path, docs[0].ID)
	}
	if docs[0].Metadata["format"] != "text" {
		t.Errorf("expected format=text, got %v", docs[0].Metadata["format"])
	}
	if docs[0].Metadata["name"] != "test.txt" {
		t.Errorf("expected name=test.txt, got %v", docs[0].Metadata["name"])
	}
}

func TestTextLoader_FileNotFound(t *testing.T) {
	l := NewTextLoader()
	_, err := l.Load(context.Background(), "/nonexistent/file.txt")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestJSONLoader_Array(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")
	data := []map[string]string{
		{"text": "first", "id": "1"},
		{"text": "second", "id": "2"},
	}
	b, _ := json.Marshal(data)
	if err := os.WriteFile(path, b, 0644); err != nil {
		t.Fatal(err)
	}

	l := NewJSONLoader(WithContentKey("text"))
	docs, err := l.Load(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 2 {
		t.Fatalf("expected 2 docs, got %d", len(docs))
	}
	if docs[0].Content != "first" {
		t.Errorf("expected 'first', got %q", docs[0].Content)
	}
	if docs[1].Content != "second" {
		t.Errorf("expected 'second', got %q", docs[1].Content)
	}
}

func TestJSONLoader_Object(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")
	data := map[string]string{"key": "value"}
	b, _ := json.Marshal(data)
	if err := os.WriteFile(path, b, 0644); err != nil {
		t.Fatal(err)
	}

	l := NewJSONLoader()
	docs, err := l.Load(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 doc, got %d", len(docs))
	}
}

func TestJSONLoader_JQPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")
	data := map[string]any{
		"data": map[string]any{
			"items": []map[string]string{
				{"content": "item1"},
				{"content": "item2"},
			},
		},
	}
	b, _ := json.Marshal(data)
	if err := os.WriteFile(path, b, 0644); err != nil {
		t.Fatal(err)
	}

	l := NewJSONLoader(WithJQPath("data.items"), WithContentKey("content"))
	docs, err := l.Load(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 2 {
		t.Fatalf("expected 2 docs, got %d", len(docs))
	}
	if docs[0].Content != "item1" {
		t.Errorf("expected 'item1', got %q", docs[0].Content)
	}
}

func TestJSONLoader_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte("not json"), 0644); err != nil {
		t.Fatal(err)
	}

	l := NewJSONLoader()
	_, err := l.Load(context.Background(), path)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestJSONLoader_BadPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")
	data := map[string]string{"key": "value"}
	b, _ := json.Marshal(data)
	if err := os.WriteFile(path, b, 0644); err != nil {
		t.Fatal(err)
	}

	l := NewJSONLoader(WithJQPath("nonexistent.path"))
	_, err := l.Load(context.Background(), path)
	if err == nil {
		t.Fatal("expected error for bad path")
	}
}

func TestCSVLoader(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.csv")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	w := csv.NewWriter(f)
	w.WriteAll([][]string{
		{"name", "age", "city"},
		{"Alice", "30", "NYC"},
		{"Bob", "25", "LA"},
	})
	f.Close()

	l := NewCSVLoader()
	docs, err := l.Load(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 2 {
		t.Fatalf("expected 2 docs, got %d", len(docs))
	}
	// All columns in content.
	if docs[0].Metadata["name"] != "Alice" {
		t.Errorf("expected name=Alice, got %v", docs[0].Metadata["name"])
	}
	if docs[0].Metadata["row"] != 0 {
		t.Errorf("expected row=0, got %v", docs[0].Metadata["row"])
	}
}

func TestCSVLoader_ContentColumns(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.csv")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	w := csv.NewWriter(f)
	w.WriteAll([][]string{
		{"name", "description", "category"},
		{"Widget", "A useful widget", "tools"},
		{"Gadget", "A cool gadget", "tech"},
	})
	f.Close()

	l := NewCSVLoader(WithContentColumns("name,description"))
	docs, err := l.Load(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 2 {
		t.Fatalf("expected 2 docs, got %d", len(docs))
	}
	expected := "name: Widget\ndescription: A useful widget"
	if docs[0].Content != expected {
		t.Errorf("expected %q, got %q", expected, docs[0].Content)
	}
}

func TestCSVLoader_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.csv")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	w := csv.NewWriter(f)
	w.WriteAll([][]string{{"header"}})
	f.Close()

	l := NewCSVLoader()
	docs, err := l.Load(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 0 {
		t.Errorf("expected 0 docs for headers-only CSV, got %d", len(docs))
	}
}

func TestMarkdownLoader(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	content := "# Title\n\nSome content.\n\n## Section\n\nMore content."
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	l := NewMarkdownLoader()
	docs, err := l.Load(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 doc, got %d", len(docs))
	}
	if docs[0].Content != content {
		t.Errorf("content mismatch")
	}
	if docs[0].Metadata["format"] != "markdown" {
		t.Errorf("expected format=markdown, got %v", docs[0].Metadata["format"])
	}
}

func TestPipeline(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, []byte("hello world"), 0644); err != nil {
		t.Fatal(err)
	}

	uppercase := TransformerFunc(func(_ context.Context, doc schema.Document) (schema.Document, error) {
		doc.Content = "TRANSFORMED: " + doc.Content
		return doc, nil
	})

	p := NewPipeline(
		WithLoader(NewTextLoader()),
		WithTransformer(uppercase),
	)

	docs, err := p.Load(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 doc, got %d", len(docs))
	}
	if docs[0].Content != "TRANSFORMED: hello world" {
		t.Errorf("expected transformed content, got %q", docs[0].Content)
	}
}

func TestPipeline_NoLoaders(t *testing.T) {
	p := NewPipeline()
	_, err := p.Load(context.Background(), "anything")
	if err == nil {
		t.Fatal("expected error for empty pipeline")
	}
}

func TestPipeline_MultipleLoaders(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	p := NewPipeline(
		WithLoader(NewTextLoader()),
		WithLoader(NewMarkdownLoader()),
	)

	docs, err := p.Load(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 2 {
		t.Fatalf("expected 2 docs (one per loader), got %d", len(docs))
	}
}
