package internal

import (
	"context"
	"sync"
)

// FinalHandler manages handling of final transcripts.
type FinalHandler struct {
	handler      func(transcript string)
	lastFinal    string
	finalCount   int
	mu           sync.RWMutex
	useIfSimilar bool
	alwaysUse    bool
}

// NewFinalHandler creates a new final handler.
func NewFinalHandler(handler func(transcript string), useIfSimilar, alwaysUse bool) *FinalHandler {
	return &FinalHandler{
		handler:      handler,
		useIfSimilar: useIfSimilar,
		alwaysUse:    alwaysUse,
		finalCount:   0,
	}
}

// Handle processes a final transcript.
func (fh *FinalHandler) Handle(ctx context.Context, transcript, preemptiveResponse string) {
	fh.mu.Lock()
	fh.lastFinal = transcript
	fh.finalCount++
	handler := fh.handler
	useIfSimilar := fh.useIfSimilar
	alwaysUse := fh.alwaysUse
	fh.mu.Unlock()

	// Determine if we should use preemptive response
	shouldUsePreemptive := false
	if alwaysUse && preemptiveResponse != "" {
		shouldUsePreemptive = true
	} else if useIfSimilar && preemptiveResponse != "" {
		// Check similarity (simplified - would use proper similarity metric in production)
		similarity := calculateStringSimilarity(transcript, preemptiveResponse)
		shouldUsePreemptive = similarity > 0.8 // 80% similarity threshold
	}

	if handler != nil {
		if shouldUsePreemptive {
			handler(preemptiveResponse)
		} else {
			handler(transcript)
		}
	}
}

// calculateStringSimilarity calculates a simple similarity score between two strings.
func calculateStringSimilarity(s1, s2 string) float64 {
	// Simplified similarity calculation
	// In production, use proper string similarity algorithms (Levenshtein, Jaro-Winkler, etc.)
	if s1 == s2 {
		return 1.0
	}

	// Simple word overlap calculation
	words1 := splitWords(s1)
	words2 := splitWords(s2)

	if len(words1) == 0 || len(words2) == 0 {
		return 0.0
	}

	overlap := 0
	for _, w1 := range words1 {
		for _, w2 := range words2 {
			if w1 == w2 {
				overlap++
				break
			}
		}
	}

	return float64(overlap) / float64(max(len(words1), len(words2)))
}

// splitWords splits a string into words (simplified).
func splitWords(s string) []string {
	// Simplified word splitting - in production, use proper tokenization
	words := []string{}
	current := ""
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' {
			if current != "" {
				words = append(words, current)
				current = ""
			}
		} else {
			current += string(r)
		}
	}
	if current != "" {
		words = append(words, current)
	}
	return words
}

// max returns the maximum of two integers.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// GetLastFinal returns the last final transcript.
func (fh *FinalHandler) GetLastFinal() string {
	fh.mu.RLock()
	defer fh.mu.RUnlock()
	return fh.lastFinal
}

// GetFinalCount returns the number of final transcripts received.
func (fh *FinalHandler) GetFinalCount() int {
	fh.mu.RLock()
	defer fh.mu.RUnlock()
	return fh.finalCount
}
