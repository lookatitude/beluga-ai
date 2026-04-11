package plancache

import (
	"sort"
	"strings"
	"unicode"
)

// stopWords is a set of common English stop words that are filtered out
// during keyword extraction.
var stopWords = map[string]bool{
	"a": true, "an": true, "the": true, "and": true, "or": true,
	"but": true, "in": true, "on": true, "at": true, "to": true,
	"for": true, "of": true, "with": true, "by": true, "from": true,
	"is": true, "are": true, "was": true, "were": true, "be": true,
	"been": true, "being": true, "have": true, "has": true, "had": true,
	"do": true, "does": true, "did": true, "will": true, "would": true,
	"could": true, "should": true, "may": true, "might": true, "shall": true,
	"can": true, "this": true, "that": true, "these": true, "those": true,
	"i": true, "you": true, "he": true, "she": true, "it": true,
	"we": true, "they": true, "me": true, "him": true, "her": true,
	"us": true, "them": true, "my": true, "your": true, "his": true,
	"its": true, "our": true, "their": true, "what": true, "which": true,
	"who": true, "whom": true, "how": true, "when": true, "where": true,
	"why": true, "if": true, "then": true, "else": true, "so": true,
	"not": true, "no": true, "nor": true, "as": true, "up": true,
	"out": true, "about": true, "into": true, "over": true, "after": true,
}

// maxKeywords is the maximum number of keywords returned by ExtractKeywords.
const maxKeywords = 20

// ExtractKeywords extracts meaningful keywords from an input string. It
// lowercases the input, removes stop words, and returns keywords sorted by
// term frequency (highest first). Keywords are capped at maxKeywords.
func ExtractKeywords(input string) []string {
	if input == "" {
		return nil
	}

	// Tokenize: split on non-letter/non-digit boundaries.
	tokens := tokenize(input)

	// Count term frequencies, filtering stop words and short tokens.
	freq := make(map[string]int)
	for _, tok := range tokens {
		lower := strings.ToLower(tok)
		if len(lower) < 2 {
			continue
		}
		if stopWords[lower] {
			continue
		}
		freq[lower]++
	}

	if len(freq) == 0 {
		return nil
	}

	// Sort by frequency (descending), then alphabetically for stability.
	type kv struct {
		word string
		freq int
	}
	pairs := make([]kv, 0, len(freq))
	for w, f := range freq {
		pairs = append(pairs, kv{w, f})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].freq != pairs[j].freq {
			return pairs[i].freq > pairs[j].freq
		}
		return pairs[i].word < pairs[j].word
	})

	limit := len(pairs)
	if limit > maxKeywords {
		limit = maxKeywords
	}

	result := make([]string, limit)
	for i := 0; i < limit; i++ {
		result[i] = pairs[i].word
	}
	return result
}

// tokenize splits input into tokens on non-letter/non-digit boundaries.
func tokenize(input string) []string {
	return strings.FieldsFunc(input, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}
