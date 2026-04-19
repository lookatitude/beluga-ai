package agentic

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/lookatitude/beluga-ai/v2/guard"
)

// Compile-time check.
var _ guard.Guard = (*DataExfiltrationGuard)(nil)

// piiPattern pairs a human-readable category with a compiled regexp for PII
// detection.
type piiPattern struct {
	category string
	pattern  *regexp.Regexp
}

// defaultPIIPatterns detect common PII categories. Patterns are intentionally
// broad to catch encoded and obfuscated forms.
var defaultPIIPatterns = []piiPattern{
	{"ssn", regexp.MustCompile(`\b\d{3}[-.\s]?\d{2}[-.\s]?\d{4}\b`)},
	{"credit_card", regexp.MustCompile(`\b(?:\d[ -]*?){13,19}\b`)},
	{"email", regexp.MustCompile(`\b[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}\b`)},
	{"phone", regexp.MustCompile(`\b(?:\+?1[-.\s]?)?\(?\d{3}\)?[-.\s]?\d{3}[-.\s]?\d{4}\b`)},
	{"ip_address", regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)},
	{"api_key", regexp.MustCompile(`(?i)(api[_-]?key|token|secret|password|bearer)\s*[:=]\s*\S+`)},
}

// suspiciousURLPattern matches URLs that may indicate data exfiltration
// through outbound requests. Covers http and https schemes.
var suspiciousURLPattern = regexp.MustCompile(`(?i)https?://[^\s"'\x60]+`)

// DataExfiltrationGuard scans tool arguments and content for PII patterns,
// URL-encoded data, and suspicious outbound payloads. It addresses OWASP AG06
// (Data Exfiltration).
type DataExfiltrationGuard struct {
	piiPatterns      []piiPattern
	blockURLs        bool
	allowedDomains   map[string]bool
	scanURLEncoding  bool
	maxContentLength int // 0 = no limit
}

// ExfiltrationOption configures a DataExfiltrationGuard.
type ExfiltrationOption func(*DataExfiltrationGuard)

// WithPIIPattern adds a custom PII detection pattern. If the provided regex
// fails to compile, the option is silently a no-op rather than panicking;
// callers should validate patterns before passing them.
func WithPIIPattern(category, pattern string) ExfiltrationOption {
	compiled, err := regexp.Compile(pattern)
	if err != nil {
		return func(*DataExfiltrationGuard) {}
	}
	return func(g *DataExfiltrationGuard) {
		g.piiPatterns = append(g.piiPatterns, piiPattern{
			category: category,
			pattern:  compiled,
		})
	}
}

// WithoutDefaultPII removes the default PII patterns so only custom patterns
// are used.
func WithoutDefaultPII() ExfiltrationOption {
	return func(g *DataExfiltrationGuard) {
		g.piiPatterns = nil
	}
}

// WithBlockURLs enables blocking when outbound URLs are detected in content.
// When allowedDomains is provided, only URLs pointing to unlisted domains are
// blocked.
func WithBlockURLs(block bool) ExfiltrationOption {
	return func(g *DataExfiltrationGuard) {
		g.blockURLs = block
	}
}

// WithAllowedDomains sets the domains that are permitted in outbound URLs.
// Only effective when WithBlockURLs(true) is set.
func WithAllowedDomains(domains ...string) ExfiltrationOption {
	return func(g *DataExfiltrationGuard) {
		for _, d := range domains {
			g.allowedDomains[strings.ToLower(d)] = true
		}
	}
}

// WithScanURLEncoding enables detection of URL-encoded content that may hide
// sensitive data.
func WithScanURLEncoding(enabled bool) ExfiltrationOption {
	return func(g *DataExfiltrationGuard) {
		g.scanURLEncoding = enabled
	}
}

// WithMaxContentLength sets the maximum content length that will be scanned.
// Content exceeding this length is automatically blocked. Zero means no limit.
func WithMaxContentLength(n int) ExfiltrationOption {
	return func(g *DataExfiltrationGuard) {
		g.maxContentLength = n
	}
}

// NewDataExfiltrationGuard creates a DataExfiltrationGuard with the given
// options. By default, it scans for PII and URL-encoded data but does not
// block URLs.
func NewDataExfiltrationGuard(opts ...ExfiltrationOption) *DataExfiltrationGuard {
	g := &DataExfiltrationGuard{
		piiPatterns:     make([]piiPattern, len(defaultPIIPatterns)),
		allowedDomains:  make(map[string]bool),
		scanURLEncoding: true,
	}
	copy(g.piiPatterns, defaultPIIPatterns)
	for _, opt := range opts {
		opt(g)
	}
	return g
}

// Name returns "data_exfiltration_guard".
func (g *DataExfiltrationGuard) Name() string {
	return "data_exfiltration_guard"
}

// Validate scans the input content for PII patterns, suspicious URLs, and
// URL-encoded data.
func (g *DataExfiltrationGuard) Validate(ctx context.Context, input guard.GuardInput) (guard.GuardResult, error) {
	select {
	case <-ctx.Done():
		return guard.GuardResult{}, ctx.Err()
	default:
	}

	content := input.Content

	// Content length check.
	if g.maxContentLength > 0 && len(content) > g.maxContentLength {
		return guard.GuardResult{
			Allowed:   false,
			Reason:    fmt.Sprintf("content length %d exceeds maximum %d", len(content), g.maxContentLength),
			GuardName: g.Name(),
		}, nil
	}

	// Decode URL-encoded content for deeper inspection.
	decoded := content
	if g.scanURLEncoding {
		if d, err := url.QueryUnescape(content); err == nil && d != content {
			decoded = d
		}
	}

	// Also try to extract string values from JSON arguments.
	textToScan := []string{content}
	if decoded != content {
		textToScan = append(textToScan, decoded)
	}
	textToScan = append(textToScan, extractJSONStrings(content)...)

	// PII scan.
	for _, text := range textToScan {
		for _, p := range g.piiPatterns {
			if p.pattern.MatchString(text) {
				return guard.GuardResult{
					Allowed:   false,
					Reason:    fmt.Sprintf("potential %s detected in content", p.category),
					GuardName: g.Name(),
				}, nil
			}
		}
	}

	// URL exfiltration check.
	if g.blockURLs {
		if blocked, reason := g.checkURLs(content); blocked {
			return guard.GuardResult{
				Allowed:   false,
				Reason:    reason,
				GuardName: g.Name(),
			}, nil
		}
	}

	// URL-encoding detection: flag content that looks percent-encoded and
	// differs when decoded (potential data hiding).
	if g.scanURLEncoding && decoded != content && containsPercentEncoding(content) {
		return guard.GuardResult{
			Allowed:   false,
			Reason:    "content contains suspicious URL-encoded data",
			GuardName: g.Name(),
		}, nil
	}

	return guard.GuardResult{Allowed: true}, nil
}

// checkURLs scans content for URLs and blocks those pointing to domains not
// in the allowed set.
func (g *DataExfiltrationGuard) checkURLs(content string) (bool, string) {
	matches := suspiciousURLPattern.FindAllString(content, -1)
	for _, m := range matches {
		parsed, err := url.Parse(m)
		if err != nil {
			continue
		}
		host := strings.ToLower(parsed.Hostname())
		if host == "" {
			continue
		}
		if len(g.allowedDomains) > 0 && !g.isDomainAllowed(host) {
			return true, fmt.Sprintf("outbound URL to disallowed domain %q detected", host)
		}
		if len(g.allowedDomains) == 0 {
			return true, fmt.Sprintf("outbound URL detected: %s", m)
		}
	}
	return false, ""
}

// isDomainAllowed checks if host or any of its parent domains are in the
// allowed set.
func (g *DataExfiltrationGuard) isDomainAllowed(host string) bool {
	if g.allowedDomains[host] {
		return true
	}
	// Check parent domains: e.g. "api.example.com" matches "example.com".
	parts := strings.Split(host, ".")
	for i := 1; i < len(parts)-1; i++ {
		parent := strings.Join(parts[i:], ".")
		if g.allowedDomains[parent] {
			return true
		}
	}
	return false
}

// extractJSONStrings parses content as JSON and returns all string values
// found recursively. This enables PII detection inside structured arguments.
func extractJSONStrings(content string) []string {
	content = strings.TrimSpace(content)
	if content == "" || (content[0] != '{' && content[0] != '[') {
		return nil
	}

	var raw any
	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		return nil
	}

	var result []string
	var extract func(v any)
	extract = func(v any) {
		switch val := v.(type) {
		case string:
			result = append(result, val)
		case map[string]any:
			for _, child := range val {
				extract(child)
			}
		case []any:
			for _, child := range val {
				extract(child)
			}
		}
	}
	extract(raw)
	return result
}

// containsPercentEncoding returns true if the string contains percent-encoded
// sequences like %20, %3A, etc.
func containsPercentEncoding(s string) bool {
	for i := 0; i < len(s)-2; i++ {
		if s[i] == '%' && isHexDigit(s[i+1]) && isHexDigit(s[i+2]) {
			return true
		}
	}
	return false
}

// isHexDigit returns true if b is a hexadecimal digit.
func isHexDigit(b byte) bool {
	return (b >= '0' && b <= '9') || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F')
}

func init() {
	guard.Register("data_exfiltration_guard", func(cfg map[string]any) (guard.Guard, error) {
		return NewDataExfiltrationGuard(), nil
	})
}
