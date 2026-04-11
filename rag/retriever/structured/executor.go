package structured

import (
	"context"
	"strings"

	"github.com/lookatitude/beluga-ai/core"
)

// QueryExecutor runs a database query and returns its results.
type QueryExecutor interface {
	// Execute runs the given query string against the target database and
	// returns the result rows as a slice of maps.
	Execute(ctx context.Context, query string) ([]map[string]any, error)
}

// writeKeywords lists SQL/Cypher keywords that indicate a write operation.
// All entries must be uppercase for case-insensitive comparison.
var writeKeywords = []string{
	"DROP",
	"DELETE",
	"INSERT",
	"UPDATE",
	"ALTER",
	"TRUNCATE",
	"CREATE",
	"MERGE",
	"SET",
	"REMOVE",
	"DETACH",
	// Additional write / DCL keywords that mutate data or access control.
	"REPLACE", // MySQL REPLACE INTO = DELETE + INSERT
	"UPSERT",  // CockroachDB, SQLite
	"GRANT",   // DCL: mutates permissions
	"REVOKE",  // DCL: mutates permissions
}

// ReadOnlyExecutor wraps a QueryExecutor and rejects any query containing
// write keywords. It provides a safety layer to prevent accidental mutations
// from LLM-generated queries.
type ReadOnlyExecutor struct {
	inner QueryExecutor
}

// Compile-time interface check.
var _ QueryExecutor = (*ReadOnlyExecutor)(nil)

// NewReadOnlyExecutor wraps the given executor with write-keyword rejection.
func NewReadOnlyExecutor(inner QueryExecutor) *ReadOnlyExecutor {
	return &ReadOnlyExecutor{inner: inner}
}

// Execute validates that the query contains no write keywords and then
// delegates to the inner executor.
func (e *ReadOnlyExecutor) Execute(ctx context.Context, query string) ([]map[string]any, error) {
	if err := validateReadOnly(query); err != nil {
		return nil, err
	}
	return e.inner.Execute(ctx, query)
}

// validateReadOnly checks whether the query contains any write keywords.
func validateReadOnly(query string) error {
	upper := strings.ToUpper(query)
	// Tokenize by splitting on whitespace and common delimiters to reduce
	// false positives from substring matching in identifiers.
	tokens := tokenize(upper)
	for _, kw := range writeKeywords {
		for _, tok := range tokens {
			if tok == kw {
				return core.Errorf(core.ErrInvalidInput, "structured.execute: write operation %q is not allowed in read-only mode", kw)
			}
		}
	}
	return nil
}

// tokenize splits an uppercased query string into tokens by whitespace and
// common SQL/Cypher punctuation, allowing keyword matching without substring
// false positives.
func tokenize(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool {
		switch r {
		case ' ', '\t', '\n', '\r', '(', ')', ',', ';', '{', '}', '[', ']':
			return true
		}
		return false
	})
}
