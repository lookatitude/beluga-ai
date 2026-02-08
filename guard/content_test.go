package guard

import (
	"context"
	"strings"
	"testing"
)

func TestContentFilter_Name(t *testing.T) {
	f := NewContentFilter()
	if got := f.Name(); got != "content_filter" {
		t.Errorf("Name() = %q, want %q", got, "content_filter")
	}
}

func TestContentFilter_NoKeywords_AllowsAll(t *testing.T) {
	f := NewContentFilter()

	tests := []struct {
		name  string
		input string
	}{
		{"empty_content", ""},
		{"normal_content", "Hello, how are you?"},
		{"any_content", "violence hate bad words"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := f.Validate(context.Background(), GuardInput{Content: tt.input})
			if err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
			if !result.Allowed {
				t.Errorf("Allowed = false, want true (no keywords configured)")
			}
		})
	}
}

func TestContentFilter_KeywordMatching(t *testing.T) {
	f := NewContentFilter(
		WithKeywords("violence", "hate", "spam"),
	)

	tests := []struct {
		name    string
		input   string
		blocked bool
		keyword string
	}{
		{
			name:    "matches_violence",
			input:   "This contains violence and is bad.",
			blocked: true,
			keyword: "violence",
		},
		{
			name:    "matches_hate",
			input:   "Spreading hate speech is wrong.",
			blocked: true,
			keyword: "hate",
		},
		{
			name:    "matches_spam",
			input:   "This is spam content.",
			blocked: true,
			keyword: "spam",
		},
		{
			name:    "no_match",
			input:   "This is a perfectly fine message.",
			blocked: false,
		},
		{
			name:    "empty_content",
			input:   "",
			blocked: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := f.Validate(context.Background(), GuardInput{Content: tt.input})
			if err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
			if result.Allowed == tt.blocked {
				t.Errorf("Allowed = %v, want %v", result.Allowed, !tt.blocked)
			}
			if tt.blocked && tt.keyword != "" {
				if !strings.Contains(result.Reason, tt.keyword) {
					t.Errorf("Reason = %q, want to contain %q", result.Reason, tt.keyword)
				}
			}
		})
	}
}

func TestContentFilter_CaseInsensitive(t *testing.T) {
	f := NewContentFilter(WithKeywords("BadWord"))

	inputs := []string{
		"this has badword in it",
		"this has BADWORD in it",
		"this has BadWord in it",
		"this has bAdWoRd in it",
	}

	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			result, err := f.Validate(context.Background(), GuardInput{Content: input})
			if err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
			if result.Allowed {
				t.Errorf("should block %q (case insensitive)", input)
			}
		})
	}
}

func TestContentFilter_Threshold(t *testing.T) {
	f := NewContentFilter(
		WithKeywords("bad", "evil", "toxic"),
		WithThreshold(2),
	)

	tests := []struct {
		name    string
		input   string
		blocked bool
	}{
		{
			name:    "one_match_below_threshold",
			input:   "This is bad content.",
			blocked: false,
		},
		{
			name:    "two_matches_at_threshold",
			input:   "This is bad and evil.",
			blocked: true,
		},
		{
			name:    "three_matches_above_threshold",
			input:   "This is bad, evil, and toxic.",
			blocked: true,
		},
		{
			name:    "no_matches",
			input:   "This is perfectly fine.",
			blocked: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := f.Validate(context.Background(), GuardInput{Content: tt.input})
			if err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
			if result.Allowed == tt.blocked {
				t.Errorf("Allowed = %v, want %v", result.Allowed, !tt.blocked)
			}
		})
	}
}

func TestContentFilter_ThresholdZeroOrNegative(t *testing.T) {
	// WithThreshold ignores non-positive values, so threshold stays at default (1).
	f := NewContentFilter(
		WithKeywords("bad"),
		WithThreshold(0),
	)

	result, err := f.Validate(context.Background(), GuardInput{Content: "this is bad"})
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if result.Allowed {
		t.Error("threshold 0 should keep default (1), content should be blocked")
	}

	f2 := NewContentFilter(
		WithKeywords("bad"),
		WithThreshold(-1),
	)

	result, err = f2.Validate(context.Background(), GuardInput{Content: "this is bad"})
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if result.Allowed {
		t.Error("negative threshold should keep default (1), content should be blocked")
	}
}

func TestContentFilter_GuardName_InResult(t *testing.T) {
	f := NewContentFilter(WithKeywords("blocked"))

	result, err := f.Validate(context.Background(), GuardInput{Content: "this is blocked"})
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if result.GuardName != "content_filter" {
		t.Errorf("GuardName = %q, want %q", result.GuardName, "content_filter")
	}
}

func TestContentFilter_ReasonFormat(t *testing.T) {
	f := NewContentFilter(WithKeywords("spam", "scam"))

	result, err := f.Validate(context.Background(), GuardInput{Content: "this is spam and a scam"})
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if result.Allowed {
		t.Fatal("expected blocked result")
	}
	if !strings.Contains(result.Reason, "content blocked") {
		t.Errorf("Reason = %q, want to contain %q", result.Reason, "content blocked")
	}
	if !strings.Contains(result.Reason, "spam") {
		t.Errorf("Reason = %q, want to contain %q", result.Reason, "spam")
	}
	if !strings.Contains(result.Reason, "scam") {
		t.Errorf("Reason = %q, want to contain %q", result.Reason, "scam")
	}
}

func TestContentFilter_PartialKeywordMatch(t *testing.T) {
	f := NewContentFilter(WithKeywords("bad"))

	// "bad" appears as substring in "badly".
	result, err := f.Validate(context.Background(), GuardInput{Content: "This went badly"})
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if result.Allowed {
		t.Error("should block substring match 'bad' in 'badly'")
	}
}

func TestContentFilter_MultipleOptions(t *testing.T) {
	f := NewContentFilter(
		WithKeywords("a", "b", "c"),
		WithThreshold(3),
	)

	// All three keywords present.
	result, err := f.Validate(context.Background(), GuardInput{Content: "a b c"})
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if result.Allowed {
		t.Error("all 3 keywords present, threshold is 3, should block")
	}

	// Only two keywords.
	result, err = f.Validate(context.Background(), GuardInput{Content: "a b"})
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if !result.Allowed {
		t.Error("only 2 keywords present, threshold is 3, should allow")
	}
}

func TestContentFilter_WithRole(t *testing.T) {
	f := NewContentFilter(WithKeywords("forbidden"))

	// The filter should work the same regardless of role metadata.
	roles := []string{"input", "output", "tool"}
	for _, role := range roles {
		t.Run(role, func(t *testing.T) {
			result, err := f.Validate(context.Background(), GuardInput{
				Content: "this is forbidden",
				Role:    role,
			})
			if err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
			if result.Allowed {
				t.Errorf("should block forbidden content for role %q", role)
			}
		})
	}
}

func TestContentFilter_AllowedResult_NoGuardName(t *testing.T) {
	f := NewContentFilter(WithKeywords("bad"))

	result, err := f.Validate(context.Background(), GuardInput{Content: "good content"})
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if !result.Allowed {
		t.Error("should allow clean content")
	}
	// When allowed, GuardName is not set.
	if result.GuardName != "" {
		t.Errorf("GuardName = %q, want empty for allowed result", result.GuardName)
	}
}
