// Package hookutil provides generic helpers for composing hook functions.
//
// It is an internal utility package used by extensible packages throughout
// the Beluga framework to implement the ComposeHooks pattern. Each helper
// takes a slice of hook structs and a field-extractor function, then returns
// a composed function that calls every non-nil hook in order.
//
// This package is internal and must not be imported outside of this module.
package hookutil
