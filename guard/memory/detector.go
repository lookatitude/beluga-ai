package memory

import (
	"context"
	"math"
	"regexp"
	"sync"
	"time"
)

// AnomalyDetector inspects content destined for memory and reports whether it
// appears anomalous. Implementations must be safe for concurrent use.
type AnomalyDetector interface {
	// Name returns a unique identifier for this detector.
	Name() string

	// Detect analyzes content and returns an AnomalyResult describing
	// whether the content is suspicious.
	Detect(ctx context.Context, content string) (AnomalyResult, error)
}

// AnomalyResult describes the outcome of an anomaly detection check.
type AnomalyResult struct {
	// Detected is true when anomalous content was found.
	Detected bool

	// Score ranges from 0.0 (benign) to 1.0 (certainly malicious).
	Score float64

	// Reason describes why the content was flagged.
	Reason string

	// DetectorName identifies which detector produced the result.
	DetectorName string
}

// --- EntropyDetector ---

// EntropyDetector flags content with unusually high Shannon entropy, which
// may indicate injection payloads or encoded malicious content.
type EntropyDetector struct {
	// Threshold is the minimum entropy (bits per byte) to flag. Default is 4.5.
	Threshold float64
}

// Compile-time check.
var _ AnomalyDetector = (*EntropyDetector)(nil)

// Name returns "entropy".
func (d *EntropyDetector) Name() string { return "entropy" }

// Detect calculates Shannon entropy of content and flags it if above threshold.
func (d *EntropyDetector) Detect(_ context.Context, content string) (AnomalyResult, error) {
	if len(content) == 0 {
		return AnomalyResult{DetectorName: d.Name()}, nil
	}

	threshold := d.Threshold
	if threshold <= 0 {
		threshold = 4.5
	}

	entropy := shannonEntropy(content)
	// Normalize to 0-1 scale (max entropy for byte is 8 bits).
	score := entropy / 8.0
	detected := entropy >= threshold

	var reason string
	if detected {
		reason = "high entropy content detected"
	}

	return AnomalyResult{
		Detected:     detected,
		Score:        score,
		Reason:       reason,
		DetectorName: d.Name(),
	}, nil
}

// shannonEntropy calculates the Shannon entropy of a string in bits per byte.
func shannonEntropy(s string) float64 {
	if len(s) == 0 {
		return 0
	}

	freq := make(map[byte]int)
	for i := 0; i < len(s); i++ {
		freq[s[i]]++
	}

	length := float64(len(s))
	var entropy float64
	for _, count := range freq {
		p := float64(count) / length
		if p > 0 {
			entropy -= p * math.Log2(p)
		}
	}
	return entropy
}

// --- PatternDetector ---

// PatternDetector matches content against known injection marker patterns
// such as prompt injection attempts, system prompt overrides, and encoded
// payloads.
type PatternDetector struct {
	// Patterns is a list of compiled regular expressions to match against.
	// If nil, a default set of injection markers is used.
	Patterns []*regexp.Regexp
}

// Compile-time check.
var _ AnomalyDetector = (*PatternDetector)(nil)

// defaultPatterns contains common prompt injection and memory poisoning markers.
var defaultPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)ignore\s+(previous|above|all)\s+(instructions?|prompts?)`),
	regexp.MustCompile(`(?i)system\s*:\s*you\s+are`),
	regexp.MustCompile(`(?i)new\s+instructions?\s*:`),
	regexp.MustCompile(`(?i)<\s*system\s*>`),
	regexp.MustCompile(`(?i)\[\s*INST\s*\]`),
	regexp.MustCompile(`(?i)forget\s+(everything|all|previous)`),
	regexp.MustCompile(`(?i)override\s+(system|safety|guard)`),
	regexp.MustCompile(`(?i)act\s+as\s+(if\s+)?you\s+(are|were)\s+a`),
}

// Name returns "pattern".
func (d *PatternDetector) Name() string { return "pattern" }

// Detect checks content against known injection patterns.
func (d *PatternDetector) Detect(_ context.Context, content string) (AnomalyResult, error) {
	patterns := d.Patterns
	if len(patterns) == 0 {
		patterns = defaultPatterns
	}

	for _, p := range patterns {
		if p.MatchString(content) {
			return AnomalyResult{
				Detected:     true,
				Score:        1.0,
				Reason:       "injection pattern matched: " + p.String(),
				DetectorName: d.Name(),
			}, nil
		}
	}

	return AnomalyResult{DetectorName: d.Name()}, nil
}

// --- RateDetector ---

// RateDetector detects burst write patterns that may indicate an automated
// memory poisoning attack. It tracks write counts within a sliding window.
type RateDetector struct {
	// MaxWrites is the maximum number of writes allowed within Window.
	// Default is 10.
	MaxWrites int

	// Window is the time window for counting writes. Default is 1 minute.
	Window time.Duration

	mu      sync.Mutex
	writers map[string][]time.Time
}

// Compile-time check.
var _ AnomalyDetector = (*RateDetector)(nil)

// Name returns "rate".
func (d *RateDetector) Name() string { return "rate" }

// Detect records a write event and checks if the rate exceeds the threshold.
// The writer identity is extracted from the context via WriterFromContext.
func (d *RateDetector) Detect(ctx context.Context, _ string) (AnomalyResult, error) {
	maxWrites := d.MaxWrites
	if maxWrites <= 0 {
		maxWrites = 10
	}
	window := d.Window
	if window <= 0 {
		window = time.Minute
	}

	writer := WriterFromContext(ctx)
	if writer == "" {
		writer = "_unknown"
	}

	now := time.Now()
	cutoff := now.Add(-window)

	d.mu.Lock()
	if d.writers == nil {
		d.writers = make(map[string][]time.Time)
	}

	// Prune old entries and add current.
	times := d.writers[writer]
	pruned := times[:0]
	for _, t := range times {
		if t.After(cutoff) {
			pruned = append(pruned, t)
		}
	}
	pruned = append(pruned, now)
	d.writers[writer] = pruned
	count := len(pruned)

	// Opportunistically sweep stale writers to prevent unbounded growth from
	// transient agent identities. Only touches a small bounded number of keys
	// per call to keep worst-case latency stable.
	swept := 0
	for k, ts := range d.writers {
		if swept >= 8 {
			break
		}
		swept++
		if k == writer {
			continue
		}
		alive := ts[:0]
		for _, t := range ts {
			if t.After(cutoff) {
				alive = append(alive, t)
			}
		}
		if len(alive) == 0 {
			delete(d.writers, k)
		} else {
			d.writers[k] = alive
		}
	}
	d.mu.Unlock()

	if count > maxWrites {
		score := math.Min(float64(count)/float64(maxWrites*2), 1.0)
		return AnomalyResult{
			Detected:     true,
			Score:        score,
			Reason:       "burst write rate exceeded threshold",
			DetectorName: d.Name(),
		}, nil
	}

	return AnomalyResult{DetectorName: d.Name()}, nil
}

// --- SizeDetector ---

// SizeDetector flags content that exceeds a maximum byte length, which may
// indicate data exfiltration or injection of large payloads.
type SizeDetector struct {
	// MaxSize is the maximum content length in bytes. Default is 10000.
	MaxSize int
}

// Compile-time check.
var _ AnomalyDetector = (*SizeDetector)(nil)

// Name returns "size".
func (d *SizeDetector) Name() string { return "size" }

// Detect checks whether content exceeds the configured size limit.
func (d *SizeDetector) Detect(_ context.Context, content string) (AnomalyResult, error) {
	maxSize := d.MaxSize
	if maxSize <= 0 {
		maxSize = 10000
	}

	if len(content) > maxSize {
		score := math.Min(float64(len(content))/float64(maxSize*2), 1.0)
		return AnomalyResult{
			Detected:     true,
			Score:        score,
			Reason:       "content size exceeds limit",
			DetectorName: d.Name(),
		}, nil
	}

	return AnomalyResult{DetectorName: d.Name()}, nil
}

// --- Context keys ---

type contextKey string

const writerKey contextKey = "memory_guard_writer"

// WithWriter returns a new context carrying the writer agent identity.
func WithWriter(ctx context.Context, writer string) context.Context {
	return context.WithValue(ctx, writerKey, writer)
}

// WriterFromContext extracts the writer agent identity from the context.
// Returns an empty string if not set.
func WriterFromContext(ctx context.Context) string {
	v, _ := ctx.Value(writerKey).(string)
	return v
}
