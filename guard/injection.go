package guard

import (
	"context"
	"regexp"
	"strings"
)

// injectionPattern pairs a human-readable description with a compiled regexp
// used to detect prompt injection attempts.
type injectionPattern struct {
	name    string
	pattern *regexp.Regexp
}

// defaultInjectionPatterns are the built-in patterns that detect common prompt
// injection techniques. Each pattern is case-insensitive.
var defaultInjectionPatterns = []injectionPattern{
	{"ignore_instructions", regexp.MustCompile(`(?i)ignore\s+(all\s+)?(previous|prior|above)\s+(instructions?|prompts?|directions?)`)},
	{"system_prompt_leak", regexp.MustCompile(`(?i)(reveal|show|print|output|display|give\s+me)\s+(\w+\s+)*(system\s+prompt|instructions?|rules?)`)},
	{"role_override", regexp.MustCompile(`(?i)(you\s+are\s+now|act\s+as\s+if|pretend\s+(you\s+are|to\s+be)|new\s+role|new\s+persona)`)},
	{"delimiter_escape", regexp.MustCompile("(?i)(" + "``" + "`" + `\s*system|<\|?(system|im_start)\|?>|\[INST\]|\[SYS\])`)},
	{"jailbreak", regexp.MustCompile(`(?i)(DAN\s+mode|do\s+anything\s+now|jailbreak|bypass\s+(safety|filter|restriction))`)},
	{"instruction_inject", regexp.MustCompile(`(?i)(system\s*prompt\s*:|assistant\s*:\s*\n|human\s*:\s*\n)`)},
}

// PromptInjectionDetector is a Guard that detects common prompt injection
// patterns in content using configurable regular expressions. It blocks
// content that matches any registered pattern.
type PromptInjectionDetector struct {
	patterns []injectionPattern
}

// InjectionOption configures a PromptInjectionDetector.
type InjectionOption func(*PromptInjectionDetector)

// WithPattern adds a custom injection detection pattern.
func WithPattern(name, pattern string) InjectionOption {
	return func(d *PromptInjectionDetector) {
		d.patterns = append(d.patterns, injectionPattern{
			name:    name,
			pattern: regexp.MustCompile(pattern),
		})
	}
}

// WithoutDefaults removes the default injection patterns so only custom
// patterns added via WithPattern are used.
func WithoutDefaults() InjectionOption {
	return func(d *PromptInjectionDetector) {
		d.patterns = nil
	}
}

// NewPromptInjectionDetector creates a PromptInjectionDetector with the
// default patterns, optionally modified by the given options.
func NewPromptInjectionDetector(opts ...InjectionOption) *PromptInjectionDetector {
	d := &PromptInjectionDetector{
		patterns: make([]injectionPattern, len(defaultInjectionPatterns)),
	}
	copy(d.patterns, defaultInjectionPatterns)

	for _, opt := range opts {
		opt(d)
	}
	return d
}

// Name returns "prompt_injection_detector".
func (d *PromptInjectionDetector) Name() string {
	return "prompt_injection_detector"
}

// Validate checks the input content against all configured injection patterns.
// If any pattern matches, the content is blocked with a reason identifying the
// matched pattern.
func (d *PromptInjectionDetector) Validate(_ context.Context, input GuardInput) (GuardResult, error) {
	lower := strings.ToLower(input.Content)
	for _, p := range d.patterns {
		if p.pattern.MatchString(lower) {
			return GuardResult{
				Allowed:   false,
				Reason:    "prompt injection detected: " + p.name,
				GuardName: d.Name(),
			}, nil
		}
	}
	return GuardResult{Allowed: true}, nil
}

func init() {
	Register("prompt_injection_detector", func(cfg map[string]any) (Guard, error) {
		return NewPromptInjectionDetector(), nil
	})
}
