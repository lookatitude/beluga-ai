package plancache

import "context"

// KeywordMatcher scores input-template similarity using Jaccard-style keyword
// overlap with term-frequency weighting. Registered as "keyword" in init().
type KeywordMatcher struct{}

var _ Matcher = (*KeywordMatcher)(nil)

func init() {
	RegisterMatcher("keyword", func() (Matcher, error) {
		return &KeywordMatcher{}, nil
	})
}

// Score computes a weighted Jaccard similarity between the input keywords and
// the template keywords. Returns a value in [0.0, 1.0].
func (m *KeywordMatcher) Score(ctx context.Context, input string, tmpl *Template) (float64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	if tmpl == nil {
		return 0, nil
	}

	inputKW := ExtractKeywords(input)
	if len(inputKW) == 0 || len(tmpl.Keywords) == 0 {
		return 0, nil
	}

	// Build frequency maps.
	inputFreq := keywordFreq(inputKW)
	tmplFreq := keywordFreq(tmpl.Keywords)

	// Compute weighted Jaccard: sum(min(freqA, freqB)) / sum(max(freqA, freqB))
	var intersection, union float64

	// All keys from both sets.
	allKeys := make(map[string]struct{})
	for k := range inputFreq {
		allKeys[k] = struct{}{}
	}
	for k := range tmplFreq {
		allKeys[k] = struct{}{}
	}

	for k := range allKeys {
		a := float64(inputFreq[k])
		b := float64(tmplFreq[k])
		if a < b {
			intersection += a
			union += b
		} else {
			intersection += b
			union += a
		}
	}

	if union == 0 {
		return 0, nil
	}

	return intersection / union, nil
}

// keywordFreq builds a frequency map from a keyword slice.
func keywordFreq(keywords []string) map[string]int {
	freq := make(map[string]int, len(keywords))
	for _, kw := range keywords {
		freq[kw]++
	}
	return freq
}
