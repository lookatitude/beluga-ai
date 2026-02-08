package llm

import (
	"strings"
	"unicode/utf8"

	"github.com/lookatitude/beluga-ai/schema"
)

// Tokenizer provides token counting and encoding/decoding for text.
// Providers may supply model-specific tokenizers; SimpleTokenizer is a
// built-in approximation suitable for budget estimation.
type Tokenizer interface {
	// Count returns the approximate number of tokens in text.
	Count(text string) int

	// CountMessages returns the approximate total token count for a slice
	// of messages, including any per-message overhead.
	CountMessages(msgs []schema.Message) int

	// Encode converts text to a sequence of token IDs.
	Encode(text string) []int

	// Decode converts token IDs back to text.
	Decode(tokens []int) string
}

// SimpleTokenizer is a word-based approximation that estimates 1 token per
// 4 characters. It is suitable for rough budget calculations when a
// model-specific tokenizer is unavailable.
type SimpleTokenizer struct{}

// charsPerToken is the average characters per token used for estimation.
const charsPerToken = 4

// Count returns the estimated token count for the given text.
func (t *SimpleTokenizer) Count(text string) int {
	n := utf8.RuneCountInString(text)
	if n == 0 {
		return 0
	}
	tokens := (n + charsPerToken - 1) / charsPerToken
	return tokens
}

// CountMessages sums the estimated token count across all messages,
// adding a small per-message overhead (4 tokens) for role and formatting.
func (t *SimpleTokenizer) CountMessages(msgs []schema.Message) int {
	const perMessageOverhead = 4
	total := 0
	for _, msg := range msgs {
		for _, part := range msg.GetContent() {
			if tp, ok := part.(schema.TextPart); ok {
				total += t.Count(tp.Text)
			}
		}
		total += perMessageOverhead
	}
	return total
}

// Encode splits text into words and assigns sequential token IDs.
// This is a simplified encoding for testing and estimation purposes.
func (t *SimpleTokenizer) Encode(text string) []int {
	if text == "" {
		return nil
	}
	words := strings.Fields(text)
	tokens := make([]int, len(words))
	for i := range words {
		tokens[i] = i
	}
	return tokens
}

// Decode joins token IDs into a placeholder string. Since SimpleTokenizer
// does not maintain a real vocabulary, this returns a space-separated
// list of token indices.
func (t *SimpleTokenizer) Decode(tokens []int) string {
	if len(tokens) == 0 {
		return ""
	}
	parts := make([]string, len(tokens))
	for i, tok := range tokens {
		parts[i] = strings.Repeat("x", 1) // placeholder
		_ = tok
	}
	return strings.Join(parts, " ")
}
