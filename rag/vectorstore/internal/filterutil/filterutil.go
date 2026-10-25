// Package filterutil validates metadata filters before vector-store
// providers translate them into a backend query language.
//
// Every provider treats a [vectorstore.SearchConfig] filter as a set of
// key = value equality predicates. Keys end up as field names or
// identifiers in a query language (GraphQL, YQL, Milvus boolean
// expressions, RediSearch, MongoDB documents), so they are restricted to a
// safe identifier pattern; values are restricted to scalars so a value can
// never smuggle an operator object (for example a MongoDB {"$ne": ...}).
package filterutil

import (
	"fmt"
	"regexp"
	"sort"

	"github.com/lookatitude/beluga-ai/v2/core"
)

// keyPattern is the identifier pattern every filter key must match.
var keyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// ValidKey reports whether key is a safe filter identifier.
func ValidKey(key string) bool {
	return keyPattern.MatchString(key)
}

// Validate checks every key and value of filter and returns the keys in
// sorted order, so callers build deterministic queries. op names the caller
// in the returned core.Error (code ErrInvalidInput).
func Validate(op string, filter map[string]any) ([]string, error) {
	keys := make([]string, 0, len(filter))
	for k, v := range filter {
		if !ValidKey(k) {
			return nil, core.NewError(op, core.ErrInvalidInput,
				fmt.Sprintf("invalid filter key %q: must match %s", k, keyPattern.String()), nil)
		}
		if !IsScalar(v) {
			return nil, core.NewError(op, core.ErrInvalidInput,
				fmt.Sprintf("invalid filter value for key %q: type %T is not a string, bool or number", k, v), nil)
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys, nil
}

// IsScalar reports whether v is a string, bool or numeric value.
func IsScalar(v any) bool {
	switch v.(type) {
	case string, bool,
		int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return true
	default:
		return false
	}
}
