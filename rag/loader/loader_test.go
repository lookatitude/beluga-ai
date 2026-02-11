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

func TestPipeline_LoaderError(t *testing.T) {
	p := NewPipeline(WithLoader(NewTextLoader()))
	_, err := p.Load(context.Background(), "/nonexistent/file.txt")
	if err == nil {
		t.Fatal("expected error when loader fails")
	}
}

func TestPipeline_TransformerError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	errorTransformer := TransformerFunc(func(_ context.Context, doc schema.Document) (schema.Document, error) {
		return doc, os.ErrInvalid
	})

	p := NewPipeline(
		WithLoader(NewTextLoader()),
		WithTransformer(errorTransformer),
	)

	_, err := p.Load(context.Background(), path)
	if err == nil {
		t.Fatal("expected error when transformer fails")
	}
}

func TestJSONLoader_ContentKeyNotFound(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")
	data := []map[string]string{
		{"other": "value"},
	}
	b, _ := json.Marshal(data)
	if err := os.WriteFile(path, b, 0644); err != nil {
		t.Fatal(err)
	}

	l := NewJSONLoader(WithContentKey("missing"))
	_, err := l.Load(context.Background(), path)
	if err == nil {
		t.Fatal("expected error when content key not found")
	}
}

func TestJSONLoader_ContentKeyNonObject(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")
	data := []string{"string1", "string2"}
	b, _ := json.Marshal(data)
	if err := os.WriteFile(path, b, 0644); err != nil {
		t.Fatal(err)
	}

	l := NewJSONLoader(WithContentKey("text"))
	_, err := l.Load(context.Background(), path)
	if err == nil {
		t.Fatal("expected error when content_key used on non-object")
	}
}

func TestJSONLoader_ContentKeyNonString(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")
	data := []map[string]any{
		{"value": map[string]string{"nested": "data"}},
	}
	b, _ := json.Marshal(data)
	if err := os.WriteFile(path, b, 0644); err != nil {
		t.Fatal(err)
	}

	l := NewJSONLoader(WithContentKey("value"))
	docs, err := l.Load(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 doc, got %d", len(docs))
	}
	// Should marshal the nested object to JSON string
	if docs[0].Content == "" {
		t.Error("expected non-empty content from marshaled object")
	}
}

func TestJSONLoader_FileNotFound(t *testing.T) {
	l := NewJSONLoader()
	_, err := l.Load(context.Background(), "/nonexistent/file.json")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestJSONLoader_NavigatePathNonObject(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")
	data := map[string]any{
		"items": "not an object",
	}
	b, _ := json.Marshal(data)
	if err := os.WriteFile(path, b, 0644); err != nil {
		t.Fatal(err)
	}

	l := NewJSONLoader(WithJQPath("items.nested"))
	_, err := l.Load(context.Background(), path)
	if err == nil {
		t.Fatal("expected error navigating through non-object")
	}
}

func TestCSVLoader_ParseError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.csv")
	// Create a malformed CSV with unmatched quotes
	if err := os.WriteFile(path, []byte("header\n\"unclosed quote"), 0644); err != nil {
		t.Fatal(err)
	}

	l := NewCSVLoader()
	_, err := l.Load(context.Background(), path)
	if err == nil {
		t.Fatal("expected error for malformed CSV")
	}
}

func TestCSVLoader_FileNotFound(t *testing.T) {
	l := NewCSVLoader()
	_, err := l.Load(context.Background(), "/nonexistent/file.csv")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestCSVLoader_ContentColumnsNotFound(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.csv")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	w := csv.NewWriter(f)
	w.WriteAll([][]string{
		{"name", "age"},
		{"Alice", "30"},
	})
	f.Close()

	// Request columns that don't exist
	l := NewCSVLoader(WithContentColumns("nonexistent,missing"))
	docs, err := l.Load(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 doc, got %d", len(docs))
	}
	// Content should be empty since no matching columns
	if docs[0].Content != "" {
		t.Errorf("expected empty content, got %q", docs[0].Content)
	}
}

func TestCSVLoader_ShortRow(t *testing.T) {
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
		{"Bob", "25", ""}, // Empty city value
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
	// Check first doc has all values
	if docs[0].Metadata["name"] != "Alice" {
		t.Errorf("expected name=Alice, got %v", docs[0].Metadata["name"])
	}
	if docs[0].Metadata["city"] != "NYC" {
		t.Errorf("expected city=NYC, got %v", docs[0].Metadata["city"])
	}
	// Check second doc handles empty value
	if docs[1].Metadata["city"] != "" {
		t.Errorf("expected empty city, got %v", docs[1].Metadata["city"])
	}
}

func TestMarkdownLoader_FileNotFound(t *testing.T) {
	l := NewMarkdownLoader()
	_, err := l.Load(context.Background(), "/nonexistent/file.md")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestNew_CSV(t *testing.T) {
	cfg := config.ProviderConfig{
		Options: map[string]any{
			"content_columns": "name,description",
		},
	}
	l, err := New("csv", cfg)
	if err != nil {
		t.Fatal(err)
	}
	if l == nil {
		t.Fatal("expected non-nil loader")
	}
}

func TestNew_JSON(t *testing.T) {
	cfg := config.ProviderConfig{
		Options: map[string]any{
			"content_key": "text",
			"jq_path":     "data.items",
		},
	}
	l, err := New("json", cfg)
	if err != nil {
		t.Fatal(err)
	}
	if l == nil {
		t.Fatal("expected non-nil loader")
	}
}

func TestNew_Markdown(t *testing.T) {
	l, err := New("markdown", config.ProviderConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if l == nil {
		t.Fatal("expected non-nil loader")
	}
}

func TestJSONLoader_ExtractContentError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")

	// Create a JSON with a value that would normally be marshaled
	// Then test the contentKey path with non-string marshaling
	data := []map[string]any{
		{"value": []int{1, 2, 3}},
	}
	b, _ := json.Marshal(data)
	if err := os.WriteFile(path, b, 0644); err != nil {
		t.Fatal(err)
	}

	l := NewJSONLoader(WithContentKey("value"))
	docs, err := l.Load(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 doc, got %d", len(docs))
	}
	// Should marshal array to JSON string
	if docs[0].Content != "[1,2,3]" {
		t.Errorf("expected '[1,2,3]', got %q", docs[0].Content)
	}
}
