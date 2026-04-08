package plancache

import (
	"sort"
	"sync"
)

// MatcherFactory is a constructor function for creating a Matcher.
type MatcherFactory func() (Matcher, error)

var (
	matcherMu       sync.RWMutex
	matcherRegistry = make(map[string]MatcherFactory)
)

// RegisterMatcher registers a matcher factory under the given name. This is
// typically called from init() in matcher implementation files.
func RegisterMatcher(name string, factory MatcherFactory) {
	matcherMu.Lock()
	defer matcherMu.Unlock()
	matcherRegistry[name] = factory
}

// NewMatcher creates a new Matcher by looking up the registered factory.
// Returns an error with code ErrMatcherNotRegistered if the name is unknown.
func NewMatcher(name string) (Matcher, error) {
	matcherMu.RLock()
	factory, ok := matcherRegistry[name]
	matcherMu.RUnlock()

	if !ok {
		return nil, newMatcherNotRegisteredError(name)
	}
	return factory()
}

// ListMatchers returns the sorted names of all registered matchers.
func ListMatchers() []string {
	matcherMu.RLock()
	defer matcherMu.RUnlock()

	names := make([]string, 0, len(matcherRegistry))
	for name := range matcherRegistry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
