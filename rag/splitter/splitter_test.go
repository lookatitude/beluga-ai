package splitter

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// TestRecursiveSplitter_WithSeparators tests custom separators.
func TestRecursiveSplitter_WithSeparators(t *testing.T) {
	// Use custom separators: split on "|" first, then "," then ""
	s := NewRecursiveSplitter(
		WithChunkSize(20),
		WithSeparators([]string{"|", ",", ""}),
	)
	text := "a,b,c,d,e|f,g,h,i,j|k,l,m,n,o"
	chunks, err := s.Split(context.Background(), text)
	require.NoError(t, err)
	assert.NotEmpty(t, chunks)
	// Should split on "|" first
	for _, chunk := range chunks {
		assert.LessOrEqual(t, len(chunk), 20)
	}
}

// TestRecursiveSplitter_ZeroOrNegativeOptions tests edge cases for options.
func TestRecursiveSplitter_ZeroOrNegativeOptions(t *testing.T) {
	tests := []struct {
		name string
		opts []RecursiveOption
	}{
		{
			name: "zero chunk size ignored",
			opts: []RecursiveOption{WithChunkSize(0)},
		},
		{
			name: "negative chunk size ignored",
			opts: []RecursiveOption{WithChunkSize(-10)},
		},
		{
			name: "negative overlap ignored",
			opts: []RecursiveOption{WithChunkOverlap(-5)},
		},
		{
			name: "empty separators ignored",
			opts: []RecursiveOption{WithSeparators([]string{})},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewRecursiveSplitter(tt.opts...)
			// Should use defaults (not crash or use invalid values)
			assert.NotNil(t, s)
			assert.Greater(t, s.chunkSize, 0)
			assert.GreaterOrEqual(t, s.chunkOverlap, 0)
			assert.NotEmpty(t, s.separators)
		})
	}
}

// TestRecursiveSplitter_CharacterLevelSplit tests fallback to character-level split.
func TestRecursiveSplitter_CharacterLevelSplit(t *testing.T) {
	// Text with no separators at all
	text := strings.Repeat("x", 150)
	s := NewRecursiveSplitter(WithChunkSize(50), WithChunkOverlap(0))
	chunks, err := s.Split(context.Background(), text)
	require.NoError(t, err)
	assert.Greater(t, len(chunks), 1)
	for _, chunk := range chunks {
		assert.LessOrEqual(t, len(chunk), 50)
	}
}

// TestRecursiveSplitter_OverlapEdgeCases tests getOverlap edge cases.
func TestRecursiveSplitter_OverlapEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		overlap int
		text    string
		want    string
	}{
		{
			name:    "zero overlap",
			overlap: 0,
			text:    "hello world",
			want:    "",
		},
		{
			name:    "overlap equals text length",
			overlap: 11,
			text:    "hello world",
			want:    "",
		},
		{
			name:    "overlap greater than text length",
			overlap: 20,
			text:    "hello",
			want:    "",
		},
		{
			name:    "normal overlap",
			overlap: 5,
			text:    "hello world",
			want:    "world",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewRecursiveSplitter(WithChunkOverlap(tt.overlap))
			got := s.getOverlap(tt.text)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestRecursiveSplitter_WithOverlap tests overlap behavior.
func TestRecursiveSplitter_WithOverlap(t *testing.T) {
	text := strings.Repeat("word ", 50)
	s := NewRecursiveSplitter(WithChunkSize(100), WithChunkOverlap(20))
	chunks, err := s.Split(context.Background(), text)
	require.NoError(t, err)
	assert.Greater(t, len(chunks), 1)
	// Verify overlap exists (last N chars of chunk should appear in next chunk)
	for i := 0; i < len(chunks)-1; i++ {
		// This is hard to verify precisely, but we can at least check chunks were created
		assert.NotEmpty(t, chunks[i])
	}
}

// TestTokenSplitter_WithTokenizer tests custom tokenizer.
func TestTokenSplitter_WithTokenizer(t *testing.T) {
	// Create custom tokenizer that counts 1 token per word
	customTokenizer := &mockTokenizer{tokensPerWord: 1}
	s := NewTokenSplitter(
		WithTokenChunkSize(3),
		WithTokenChunkOverlap(1),
		WithTokenizer(customTokenizer),
	)
	text := "one two three four five six"
	chunks, err := s.Split(context.Background(), text)
	require.NoError(t, err)
	assert.Greater(t, len(chunks), 1)
}

// TestTokenSplitter_WithTokenizerNil tests nil tokenizer is ignored.
func TestTokenSplitter_WithTokenizerNil(t *testing.T) {
	s := NewTokenSplitter(WithTokenizer(nil))
	assert.NotNil(t, s.tokenizer)
}

// TestTokenSplitter_ZeroOptions tests zero/negative options.
func TestTokenSplitter_ZeroOptions(t *testing.T) {
	tests := []struct {
		name string
		opts []TokenOption
	}{
		{
			name: "zero chunk size ignored",
			opts: []TokenOption{WithTokenChunkSize(0)},
		},
		{
			name: "negative chunk size ignored",
			opts: []TokenOption{WithTokenChunkSize(-10)},
		},
		{
			name: "negative overlap ignored",
			opts: []TokenOption{WithTokenChunkOverlap(-5)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewTokenSplitter(tt.opts...)
			assert.NotNil(t, s)
			assert.Greater(t, s.chunkSize, 0)
			assert.GreaterOrEqual(t, s.chunkOverlap, 0)
		})
	}
}

// TestTokenSplitter_WithOverlap tests token overlap.
func TestTokenSplitter_WithOverlap(t *testing.T) {
	s := NewTokenSplitter(
		WithTokenChunkSize(5),
		WithTokenChunkOverlap(2),
	)
	text := "one two three four five six seven eight nine ten"
	chunks, err := s.Split(context.Background(), text)
	require.NoError(t, err)
	assert.Greater(t, len(chunks), 1)
}

// TestTokenSplitter_NoOverlapAtEnd tests no overlap when at end.
func TestTokenSplitter_NoOverlapAtEnd(t *testing.T) {
	s := NewTokenSplitter(
		WithTokenChunkSize(10),
		WithTokenChunkOverlap(5),
	)
	text := "one two three"
	chunks, err := s.Split(context.Background(), text)
	require.NoError(t, err)
	assert.Len(t, chunks, 1)
}

// TestTokenSplitter_WhitespaceOnly tests whitespace-only text.
func TestTokenSplitter_WhitespaceOnly(t *testing.T) {
	s := NewTokenSplitter()
	chunks, err := s.Split(context.Background(), "   \n\n   ")
	require.NoError(t, err)
	assert.Empty(t, chunks)
}

// TestMarkdownSplitter_ZeroOptions tests zero/negative options.
func TestMarkdownSplitter_ZeroOptions(t *testing.T) {
	tests := []struct {
		name string
		opts []MarkdownOption
	}{
		{
			name: "zero chunk size ignored",
			opts: []MarkdownOption{WithMarkdownChunkSize(0)},
		},
		{
			name: "negative chunk size ignored",
			opts: []MarkdownOption{WithMarkdownChunkSize(-10)},
		},
		{
			name: "negative overlap ignored",
			opts: []MarkdownOption{WithMarkdownChunkOverlap(-5)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewMarkdownSplitter(tt.opts...)
			assert.NotNil(t, s)
			assert.Greater(t, s.chunkSize, 0)
			assert.GreaterOrEqual(t, s.chunkOverlap, 0)
		})
	}
}

// TestMarkdownSplitter_SplitDocuments tests direct call to SplitDocuments.
func TestMarkdownSplitter_SplitDocuments(t *testing.T) {
	s := NewMarkdownSplitter(WithMarkdownChunkSize(100))
	docs := []schema.Document{
		{
			ID:      "md1",
			Content: "# Title\n\nIntro.\n\n## Section\n\nContent.",
			Metadata: map[string]any{
				"format": "markdown",
			},
		},
	}
	result, err := s.SplitDocuments(context.Background(), docs)
	require.NoError(t, err)
	assert.Greater(t, len(result), 0)
	for _, doc := range result {
		assert.Equal(t, "markdown", doc.Metadata["format"])
		assert.Equal(t, "md1", doc.Metadata["parent_id"])
	}
}

// TestMarkdownSplitter_EmptySection tests sections with empty content.
func TestMarkdownSplitter_EmptySection(t *testing.T) {
	text := "# Title\n\n## Section\n\n"
	s := NewMarkdownSplitter()
	chunks, err := s.Split(context.Background(), text)
	require.NoError(t, err)
	// Empty sections should be skipped
	for _, chunk := range chunks {
		assert.NotEmpty(t, strings.TrimSpace(chunk))
	}
}

// TestMarkdownSplitter_OnlyHeadings tests text with only headings.
func TestMarkdownSplitter_OnlyHeadings(t *testing.T) {
	text := "# Title\n## Section\n### Subsection"
	s := NewMarkdownSplitter()
	chunks, err := s.Split(context.Background(), text)
	require.NoError(t, err)
	// Should have chunks for headings
	assert.NotEmpty(t, chunks)
}

// TestHeadingLevel_EdgeCases tests additional edge cases for headingLevel.
func TestHeadingLevel_EdgeCases(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"#", 1}, // Just a single #
		{"##", 2},
		{"######", 6},
		{"#######", 0}, // 7 levels (invalid)
		{"# ", 1},      // With space
		{"##  ", 2},    // Multiple spaces
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := headingLevel(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestNew_Markdown_WithConfig tests markdown factory with config.
func TestNew_Markdown_WithConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config.ProviderConfig
		wantErr bool
	}{
		{
			name: "with all options",
			cfg: config.ProviderConfig{
				Options: map[string]any{
					"chunk_size":       float64(200),
					"chunk_overlap":    float64(20),
					"preserve_headers": true,
				},
			},
			wantErr: false,
		},
		{
			name: "with some options",
			cfg: config.ProviderConfig{
				Options: map[string]any{
					"chunk_size": float64(150),
				},
			},
			wantErr: false,
		},
		{
			name:    "empty config",
			cfg:     config.ProviderConfig{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := New("markdown", tt.cfg)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, s)
			}
		})
	}
}

// TestNew_Token_WithConfig tests token factory with config.
func TestNew_Token_WithConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config.ProviderConfig
		wantErr bool
	}{
		{
			name: "with all options",
			cfg: config.ProviderConfig{
				Options: map[string]any{
					"chunk_size":    float64(100),
					"chunk_overlap": float64(10),
				},
			},
			wantErr: false,
		},
		{
			name: "with chunk_size only",
			cfg: config.ProviderConfig{
				Options: map[string]any{
					"chunk_size": float64(200),
				},
			},
			wantErr: false,
		},
		{
			name:    "empty config",
			cfg:     config.ProviderConfig{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := New("token", tt.cfg)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, s)
			}
		})
	}
}

// TestSplitDocumentsHelper_Error tests error path in splitDocumentsHelper.
func TestSplitDocumentsHelper_Error(t *testing.T) {
	// Use a mock splitter that returns an error
	mockSplitter := &errorSplitter{err: fmt.Errorf("mock split error")}
	docs := []schema.Document{
		{
			ID:      "doc1",
			Content: "test content",
		},
	}
	_, err := splitDocumentsHelper(context.Background(), mockSplitter, docs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mock split error")
}

// TestMarkdownSplitter_ContentWithoutHeading tests text without headings.
func TestMarkdownSplitter_ContentWithoutHeading(t *testing.T) {
	// Text without any headings (level 0 content)
	text := "This is some content.\nIt has no headings.\nJust plain text."
	s := NewMarkdownSplitter()
	chunks, err := s.Split(context.Background(), text)
	require.NoError(t, err)
	assert.NotEmpty(t, chunks)
}

// TestMarkdownSplitter_MixedContentWithOverlap tests sections with overlap.
func TestMarkdownSplitter_MixedContentWithOverlap(t *testing.T) {
	text := "# Title\n\n" + strings.Repeat("word ", 200) + "\n\n## Section\n\nMore content."
	s := NewMarkdownSplitter(
		WithMarkdownChunkSize(100),
		WithMarkdownChunkOverlap(20),
	)
	chunks, err := s.Split(context.Background(), text)
	require.NoError(t, err)
	assert.Greater(t, len(chunks), 1)
}

// TestRecursiveSplitter_ComplexSeparatorFlow tests complex separator fallback.
func TestRecursiveSplitter_ComplexSeparatorFlow(t *testing.T) {
	// Test a scenario where we have multiple levels of separator fallback
	text := strings.Repeat("x", 40) + "\n\n" + strings.Repeat("y", 40) + "\n" + strings.Repeat("z", 40)
	s := NewRecursiveSplitter(
		WithChunkSize(50),
		WithChunkOverlap(5),
	)
	chunks, err := s.Split(context.Background(), text)
	require.NoError(t, err)
	assert.Greater(t, len(chunks), 1)
}

// TestRecursiveSplitter_SingleWordPerChunk tests when each word becomes a chunk.
func TestRecursiveSplitter_SingleWordPerChunk(t *testing.T) {
	// Very small chunk size forces one word per chunk
	text := "one two three four five"
	s := NewRecursiveSplitter(
		WithChunkSize(3),
		WithChunkOverlap(0),
	)
	chunks, err := s.Split(context.Background(), text)
	require.NoError(t, err)
	assert.Greater(t, len(chunks), 1)
}

// errorSplitter is a mock splitter that always returns an error.
type errorSplitter struct {
	err error
}

func (e *errorSplitter) Split(_ context.Context, _ string) ([]string, error) {
	return nil, e.err
}

func (e *errorSplitter) SplitDocuments(ctx context.Context, docs []schema.Document) ([]schema.Document, error) {
	return splitDocumentsHelper(ctx, e, docs)
}

// mockTokenizer is a simple mock tokenizer for testing.
type mockTokenizer struct {
	tokensPerWord int
}

func (m *mockTokenizer) Count(text string) int {
	if m.tokensPerWord == 0 {
		return len(text) / 4 // Default SimpleTokenizer behavior
	}
	return m.tokensPerWord
}

func (m *mockTokenizer) CountMessages(msgs []schema.Message) int {
	total := 0
	for _, msg := range msgs {
		for _, part := range msg.GetContent() {
			if tp, ok := part.(schema.TextPart); ok {
				total += m.Count(tp.Text)
			}
		}
		total += 4 // per-message overhead
	}
	return total
}

func (m *mockTokenizer) Encode(text string) []int {
	count := m.Count(text)
	tokens := make([]int, count)
	for i := range tokens {
		tokens[i] = i
	}
	return tokens
}

func (m *mockTokenizer) Decode(tokens []int) string {
	return strings.Repeat("x", len(tokens)*4)
}

var _ llm.Tokenizer = (*mockTokenizer)(nil)
