package guard

import (
	"context"
	"strings"
)

// ContentFilter is a Guard that performs keyword-based content moderation.
// It checks content against a set of blocked keywords and blocks the content
// when the number of matches meets or exceeds the configured threshold.
type ContentFilter struct {
	keywords  []string
	threshold int
}

// ContentOption configures a ContentFilter.
type ContentOption func(*ContentFilter)

// WithKeywords sets the list of blocked keywords. Keywords are matched
// case-insensitively against the content.
func WithKeywords(keywords ...string) ContentOption {
	return func(f *ContentFilter) {
		f.keywords = keywords
	}
}

// WithThreshold sets the minimum number of keyword matches required to
// block content. The default threshold is 1.
func WithThreshold(n int) ContentOption {
	return func(f *ContentFilter) {
		if n > 0 {
			f.threshold = n
		}
	}
}

// NewContentFilter creates a ContentFilter with the given options. By
// default, the filter has an empty keyword list and a threshold of 1.
func NewContentFilter(opts ...ContentOption) *ContentFilter {
	f := &ContentFilter{
		threshold: 1,
	}
	for _, opt := range opts {
		opt(f)
	}
	return f
}

// Name returns "content_filter".
func (f *ContentFilter) Name() string {
	return "content_filter"
}

// Validate checks the input content for blocked keywords. If the number of
// distinct keyword matches meets or exceeds the threshold, the content is
// blocked and the matching keywords are listed in the reason.
func (f *ContentFilter) Validate(_ context.Context, input GuardInput) (GuardResult, error) {
	if len(f.keywords) == 0 {
		return GuardResult{Allowed: true}, nil
	}

	lower := strings.ToLower(input.Content)
	var matched []string

	for _, kw := range f.keywords {
		if strings.Contains(lower, strings.ToLower(kw)) {
			matched = append(matched, kw)
		}
	}

	if len(matched) >= f.threshold {
		return GuardResult{
			Allowed:   false,
			Reason:    "content blocked: matched keywords [" + strings.Join(matched, ", ") + "]",
			GuardName: f.Name(),
		}, nil
	}

	return GuardResult{Allowed: true}, nil
}

func init() {
	Register("content_filter", func(cfg map[string]any) (Guard, error) {
		return NewContentFilter(), nil
	})
}
