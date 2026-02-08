package memory

import (
	"strings"

	"github.com/lookatitude/beluga-ai/schema"
)

// extractMessageText extracts concatenated text from a message's content parts.
func extractMessageText(msg schema.Message) string {
	var b strings.Builder
	for i, p := range msg.GetContent() {
		if tp, ok := p.(schema.TextPart); ok {
			if i > 0 && b.Len() > 0 {
				b.WriteByte('\n')
			}
			b.WriteString(tp.Text)
		}
	}
	return b.String()
}

// matchesQuery checks if a message's text content contains the query as a
// case-insensitive substring.
func matchesQuery(msg schema.Message, query string) bool {
	text := strings.ToLower(extractMessageText(msg))
	return strings.Contains(text, strings.ToLower(query))
}
