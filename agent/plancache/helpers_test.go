package plancache

import "fmt"

// mustNewMatcher creates a Matcher or panics. Used only in tests.
func mustNewMatcher(name string) Matcher {
	m, err := NewMatcher(name)
	if err != nil {
		panic(fmt.Sprintf("plancache test: %v", err))
	}
	return m
}
