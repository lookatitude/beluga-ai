package llm

import (
	"testing"

	"github.com/lookatitude/beluga-ai/schema"
)

func TestSimpleTokenizer_Count(t *testing.T) {
	tok := &SimpleTokenizer{}

	tests := []struct {
		name string
		text string
		want int
	}{
		{name: "empty", text: "", want: 0},
		{name: "short", text: "hi", want: 1},   // 2 chars / 4 = 0.5, rounds up to 1
		{name: "four chars", text: "test", want: 1}, // 4 / 4 = 1
		{name: "five chars", text: "hello", want: 2}, // 5 / 4 = 1.25, rounds up to 2
		{name: "long text", text: "hello world this is a longer text for testing", want: 12}, // 45 chars, (45+3)/4=12
		{name: "unicode", text: "héllo wörld", want: 3}, // 11 runes, (11+3)/4=3 (rounded up)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tok.Count(tt.text)
			if got != tt.want {
				t.Errorf("Count(%q) = %d, want %d", tt.text, got, tt.want)
			}
		})
	}
}

func TestSimpleTokenizer_CountMessages(t *testing.T) {
	tok := &SimpleTokenizer{}

	tests := []struct {
		name string
		msgs []schema.Message
		want int
	}{
		{
			name: "nil messages",
			msgs: nil,
			want: 0,
		},
		{
			name: "empty messages",
			msgs: []schema.Message{},
			want: 0,
		},
		{
			name: "single message",
			msgs: []schema.Message{
				schema.NewHumanMessage("test"), // 4 chars = 1 token + 4 overhead = 5
			},
			want: 5,
		},
		{
			name: "multiple messages",
			msgs: []schema.Message{
				schema.NewHumanMessage("test"),       // 1 + 4 = 5
				schema.NewAIMessage("hello world"),   // 3 + 4 = 7 (11 chars = 3 tokens)
			},
			want: 12,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tok.CountMessages(tt.msgs)
			if got != tt.want {
				t.Errorf("CountMessages() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestSimpleTokenizer_Encode(t *testing.T) {
	tok := &SimpleTokenizer{}

	tests := []struct {
		name  string
		text  string
		count int
	}{
		{name: "empty", text: "", count: 0},
		{name: "one word", text: "hello", count: 1},
		{name: "three words", text: "hello world foo", count: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := tok.Encode(tt.text)
			if len(tokens) != tt.count {
				t.Errorf("Encode(%q) len = %d, want %d", tt.text, len(tokens), tt.count)
			}
			// Token IDs should be sequential: 0, 1, 2, ...
			for i, tok := range tokens {
				if tok != i {
					t.Errorf("token[%d] = %d, want %d", i, tok, i)
				}
			}
		})
	}
}

func TestSimpleTokenizer_Decode(t *testing.T) {
	tok := &SimpleTokenizer{}

	tests := []struct {
		name   string
		tokens []int
		want   string
	}{
		{name: "empty", tokens: nil, want: ""},
		{name: "one token", tokens: []int{0}, want: "x"},
		{name: "three tokens", tokens: []int{0, 1, 2}, want: "x x x"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tok.Decode(tt.tokens)
			if got != tt.want {
				t.Errorf("Decode(%v) = %q, want %q", tt.tokens, got, tt.want)
			}
		})
	}
}

func TestSimpleTokenizer_RoundTrip(t *testing.T) {
	tok := &SimpleTokenizer{}
	text := "hello world test"
	tokens := tok.Encode(text)
	decoded := tok.Decode(tokens)

	// Not a true round-trip since SimpleTokenizer uses placeholders,
	// but the token count should match.
	if len(tokens) != 3 {
		t.Errorf("expected 3 tokens, got %d", len(tokens))
	}
	if decoded != "x x x" {
		t.Errorf("Decode = %q, want %q", decoded, "x x x")
	}
}
