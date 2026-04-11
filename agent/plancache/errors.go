package plancache

import "github.com/lookatitude/beluga-ai/core"

const (
	// ErrCacheOp is the error code for plan cache operation failures.
	ErrCacheOp core.ErrorCode = "plancache_error"

	// ErrTemplateNotFound is the error code when a template is not found.
	ErrTemplateNotFound core.ErrorCode = "template_not_found"

	// ErrMatcherNotRegistered is the error code when a matcher name is unknown.
	ErrMatcherNotRegistered core.ErrorCode = "matcher_not_registered"

	// ErrStoreFull is the error code when the store has reached capacity.
	ErrStoreFull core.ErrorCode = "store_full"
)

// newCacheError creates a new cache operation error.
func newCacheError(op, msg string, cause error) *core.Error {
	return core.NewError(op, ErrCacheOp, msg, cause)
}

// newNotFoundError creates a template-not-found error.
func newNotFoundError(op, id string) *core.Error {
	return core.NewError(op, ErrTemplateNotFound, "template not found: "+id, nil)
}

// newMatcherNotRegisteredError creates a matcher-not-registered error.
func newMatcherNotRegisteredError(name string) *core.Error {
	return core.NewError("plancache.NewMatcher", ErrMatcherNotRegistered, "matcher not registered: "+name, nil)
}
