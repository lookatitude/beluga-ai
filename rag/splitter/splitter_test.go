package splitter

import (
	"context"
	"strings"
	"testing"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/schema"
)

func TestRegistry(t *testing.T) {
	names := List()
	expected := []string{"markdown", "recursive", "token"}
	if len(names) != len(expected) {
		t.Fatalf("expected %d splitters, got %d: %v", len(expected), len(names), names)
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
		t.Fatal("expected error for unknown splitter")
	}
}

func TestNew_Recursive(t *testing.T) {
	s, err := New("recursive", config.ProviderConfig{
		Options: map[string]any{
			"chunk_size":    float64(100),
			"chunk_overlap": float64(10),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if s == nil {
		t.Fatal("expected non-nil splitter")
	}
}

func TestRecursiveSplitter_ShortText(t *testing.T) {
	s := NewRecursiveSplitter(WithChunkSize(100))
	chunks, err := s.Split(context.Background(), "short text")
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0] != "short text" {
		t.Errorf("expected 'short text', got %q", chunks[0])
	}
}

func TestRecursiveSplitter_ParagraphSplit(t *testing.T) {
	text := strings.Repeat("word ", 20) + "\n\n" + strings.Repeat("word ", 20)
	s := NewRecursiveSplitter(WithChunkSize(110), WithChunkOverlap(0))
	chunks, err := s.Split(context.Background(), text)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) < 2 {
		t.Fatalf("expected at least 2 chunks, got %d", len(chunks))
	}
}

func TestRecursiveSplitter_LineSplit(t *testing.T) {
	// No paragraph breaks, only line breaks.
	var lines []string
	for i := 0; i < 10; i++ {
		lines = append(lines, strings.Repeat("x", 30))
	}
	text := strings.Join(lines, "\n")

	s := NewRecursiveSplitter(WithChunkSize(100), WithChunkOverlap(0))
	chunks, err := s.Split(context.Background(), text)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}
	for _, c := range chunks {
		if len(c) > 100 {
			t.Errorf("chunk exceeds max size: len=%d", len(c))
		}
	}
}

func TestRecursiveSplitter_EmptyText(t *testing.T) {
	s := NewRecursiveSplitter()
	chunks, err := s.Split(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 0 {
		t.Errorf("expected 0 chunks for empty text, got %d", len(chunks))
	}
}

func TestRecursiveSplitter_WhitespaceOnly(t *testing.T) {
	s := NewRecursiveSplitter()
	chunks, err := s.Split(context.Background(), "   \n\n   ")
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 0 {
		t.Errorf("expected 0 chunks for whitespace-only text, got %d", len(chunks))
	}
}

func TestRecursiveSplitter_SplitDocuments(t *testing.T) {
	s := NewRecursiveSplitter(WithChunkSize(50), WithChunkOverlap(0))
	docs := []schema.Document{
		{
			ID:       "doc1",
			Content:  strings.Repeat("word ", 30),
			Metadata: map[string]any{"source": "test"},
		},
	}
	result, err := s.SplitDocuments(context.Background(), docs)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(result))
	}
	for _, doc := range result {
		if doc.Metadata["parent_id"] != "doc1" {
			t.Errorf("expected parent_id=doc1, got %v", doc.Metadata["parent_id"])
		}
		if doc.Metadata["source"] != "test" {
			t.Errorf("expected source=test preserved, got %v", doc.Metadata["source"])
		}
		if _, ok := doc.Metadata["chunk_index"]; !ok {
			t.Error("expected chunk_index in metadata")
		}
		if _, ok := doc.Metadata["chunk_total"]; !ok {
			t.Error("expected chunk_total in metadata")
		}
	}
}

func TestMarkdownSplitter_Basic(t *testing.T) {
	text := "# Introduction\n\nThis is the intro.\n\n## Methods\n\nHere are the methods.\n\n## Results\n\nHere are the results."
	s := NewMarkdownSplitter(WithMarkdownChunkSize(1000))
	chunks, err := s.Split(context.Background(), text)
	if err != nil {
		t.Fatal(err)
	}
	// Should split on headings: intro, methods, results.
	if len(chunks) < 3 {
		t.Fatalf("expected at least 3 chunks, got %d: %v", len(chunks), chunks)
	}
}

func TestMarkdownSplitter_PreserveHeaders(t *testing.T) {
	text := "# Title\n\nIntro text.\n\n## Section A\n\nSection A content."
	s := NewMarkdownSplitter(
		WithMarkdownChunkSize(1000),
		WithPreserveHeaders(true),
	)
	chunks, err := s.Split(context.Background(), text)
	if err != nil {
		t.Fatal(err)
	}
	// Section A chunk should contain "# Title" as parent header.
	found := false
	for _, c := range chunks {
		if strings.Contains(c, "# Title") && strings.Contains(c, "## Section A") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected a chunk with parent header preserved; chunks: %v", chunks)
	}
}

func TestMarkdownSplitter_NoPreserveHeaders(t *testing.T) {
	text := "# Title\n\nIntro.\n\n## Sub\n\nContent."
	s := NewMarkdownSplitter(
		WithMarkdownChunkSize(1000),
		WithPreserveHeaders(false),
	)
	chunks, err := s.Split(context.Background(), text)
	if err != nil {
		t.Fatal(err)
	}
	// Last chunk should NOT have "# Title" prepended.
	for _, c := range chunks {
		if strings.Contains(c, "## Sub") && strings.Contains(c, "# Title") {
			t.Errorf("expected no parent header when preserveHeaders=false, got %q", c)
		}
	}
}

func TestMarkdownSplitter_LargeSection(t *testing.T) {
	// A section that exceeds chunk size should be recursively split.
	text := "# Heading\n\n" + strings.Repeat("word ", 500)
	s := NewMarkdownSplitter(WithMarkdownChunkSize(100), WithMarkdownChunkOverlap(0))
	chunks, err := s.Split(context.Background(), text)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) < 2 {
		t.Fatalf("expected large section to be split, got %d chunks", len(chunks))
	}
}

func TestMarkdownSplitter_EmptyText(t *testing.T) {
	s := NewMarkdownSplitter()
	chunks, err := s.Split(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 0 {
		t.Errorf("expected 0 chunks for empty text, got %d", len(chunks))
	}
}

func TestTokenSplitter_Basic(t *testing.T) {
	// SimpleTokenizer: ~4 chars per token. "hello world" ≈ 3 tokens.
	s := NewTokenSplitter(WithTokenChunkSize(3), WithTokenChunkOverlap(0))
	text := "hello world foo bar baz qux"
	chunks, err := s.Split(context.Background(), text)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d: %v", len(chunks), chunks)
	}
}

func TestTokenSplitter_EmptyText(t *testing.T) {
	s := NewTokenSplitter()
	chunks, err := s.Split(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 0 {
		t.Errorf("expected 0 chunks for empty text, got %d", len(chunks))
	}
}

func TestTokenSplitter_ShortText(t *testing.T) {
	s := NewTokenSplitter(WithTokenChunkSize(1000))
	chunks, err := s.Split(context.Background(), "short")
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
}

func TestTokenSplitter_SplitDocuments(t *testing.T) {
	s := NewTokenSplitter(WithTokenChunkSize(5), WithTokenChunkOverlap(0))
	docs := []schema.Document{
		{
			ID:       "doc1",
			Content:  "one two three four five six seven eight nine ten",
			Metadata: map[string]any{"source": "test"},
		},
	}
	result, err := s.SplitDocuments(context.Background(), docs)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(result))
	}
	for _, doc := range result {
		if doc.Metadata["parent_id"] != "doc1" {
			t.Errorf("expected parent_id=doc1, got %v", doc.Metadata["parent_id"])
		}
	}
}

func TestHeadingLevel(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"# Title", 1},
		{"## Section", 2},
		{"### Sub", 3},
		{"#### Deep", 4},
		{"##### Deeper", 5},
		{"###### Deepest", 6},
		{"####### Too many", 0},
		{"Not a heading", 0},
		{"#NoSpace", 0},
		{"", 0},
	}
	for _, tt := range tests {
		got := headingLevel(tt.input)
		if got != tt.want {
			t.Errorf("headingLevel(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestSplitDocumentsHelper_Metadata(t *testing.T) {
	s := NewRecursiveSplitter(WithChunkSize(20), WithChunkOverlap(0))
	docs := []schema.Document{
		{
			ID:      "d1",
			Content: "a b c d e f g h i j k l m n o p",
			Metadata: map[string]any{
				"author": "test",
			},
		},
	}
	result, err := splitDocumentsHelper(context.Background(), s, docs)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) == 0 {
		t.Fatal("expected at least one result")
	}
	for i, doc := range result {
		if doc.Metadata["author"] != "test" {
			t.Errorf("chunk %d: author not preserved", i)
		}
		if doc.Metadata["chunk_index"] != i {
			t.Errorf("chunk %d: expected chunk_index=%d, got %v", i, i, doc.Metadata["chunk_index"])
		}
		if doc.Metadata["chunk_total"] != len(result) {
			t.Errorf("chunk %d: expected chunk_total=%d, got %v", i, len(result), doc.Metadata["chunk_total"])
		}
	}
}
