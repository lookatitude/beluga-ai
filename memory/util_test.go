package memory

import (
	"testing"

	"github.com/lookatitude/beluga-ai/schema"
	"github.com/stretchr/testify/assert"
)

func TestExtractMessageText(t *testing.T) {
	tests := []struct {
		name string
		msg  schema.Message
		want string
	}{
		{
			name: "single text part",
			msg:  schema.NewHumanMessage("hello world"),
			want: "hello world",
		},
		{
			name: "multiple text parts",
			msg: &schema.HumanMessage{
				Parts: []schema.ContentPart{
					schema.TextPart{Text: "hello"},
					schema.TextPart{Text: "world"},
				},
			},
			want: "hello\nworld",
		},
		{
			name: "empty message",
			msg:  &schema.HumanMessage{Parts: []schema.ContentPart{}},
			want: "",
		},
		{
			name: "no text parts",
			msg: &schema.HumanMessage{
				Parts: []schema.ContentPart{
					// Non-text parts would be other ContentPart implementations
				},
			},
			want: "",
		},
		{
			name: "mixed parts (only text extracted)",
			msg: &schema.AIMessage{
				Parts: []schema.ContentPart{
					schema.TextPart{Text: "first"},
					schema.TextPart{Text: "second"},
				},
			},
			want: "first\nsecond",
		},
		{
			name: "system message",
			msg:  schema.NewSystemMessage("system instruction"),
			want: "system instruction",
		},
		{
			name: "ai message",
			msg:  schema.NewAIMessage("ai response"),
			want: "ai response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractMessageText(tt.msg)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMatchesQuery(t *testing.T) {
	tests := []struct {
		name  string
		msg   schema.Message
		query string
		want  bool
	}{
		{
			name:  "exact match",
			msg:   schema.NewHumanMessage("hello world"),
			query: "hello world",
			want:  true,
		},
		{
			name:  "substring match",
			msg:   schema.NewHumanMessage("hello world"),
			query: "world",
			want:  true,
		},
		{
			name:  "case insensitive match",
			msg:   schema.NewHumanMessage("Hello World"),
			query: "hello",
			want:  true,
		},
		{
			name:  "case insensitive query",
			msg:   schema.NewHumanMessage("hello world"),
			query: "WORLD",
			want:  true,
		},
		{
			name:  "no match",
			msg:   schema.NewHumanMessage("hello world"),
			query: "goodbye",
			want:  false,
		},
		{
			name:  "empty query matches",
			msg:   schema.NewHumanMessage("hello"),
			query: "",
			want:  true,
		},
		{
			name:  "empty message no match",
			msg:   &schema.HumanMessage{Parts: []schema.ContentPart{}},
			query: "hello",
			want:  false,
		},
		{
			name: "multiple parts match",
			msg: &schema.HumanMessage{
				Parts: []schema.ContentPart{
					schema.TextPart{Text: "first part"},
					schema.TextPart{Text: "second part"},
				},
			},
			query: "second",
			want:  true,
		},
		{
			name: "multiple parts no match",
			msg: &schema.HumanMessage{
				Parts: []schema.ContentPart{
					schema.TextPart{Text: "first part"},
					schema.TextPart{Text: "second part"},
				},
			},
			query: "third",
			want:  false,
		},
		{
			name:  "ai message match",
			msg:   schema.NewAIMessage("I am an assistant"),
			query: "assistant",
			want:  true,
		},
		{
			name:  "system message match",
			msg:   schema.NewSystemMessage("You are helpful"),
			query: "helpful",
			want:  true,
		},
		{
			name:  "partial word match",
			msg:   schema.NewHumanMessage("development"),
			query: "dev",
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesQuery(tt.msg, tt.query)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMatchesQueryWithSpecialCharacters(t *testing.T) {
	tests := []struct {
		name  string
		msg   schema.Message
		query string
		want  bool
	}{
		{
			name:  "punctuation",
			msg:   schema.NewHumanMessage("Hello, world!"),
			query: "hello,",
			want:  true,
		},
		{
			name:  "numbers",
			msg:   schema.NewHumanMessage("The answer is 42"),
			query: "42",
			want:  true,
		},
		{
			name:  "unicode",
			msg:   schema.NewHumanMessage("Hello 世界"),
			query: "世界",
			want:  true,
		},
		{
			name:  "newlines in text",
			msg:   schema.NewHumanMessage("Line 1\nLine 2"),
			query: "line 2",
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesQuery(tt.msg, tt.query)
			assert.Equal(t, tt.want, got)
		})
	}
}
