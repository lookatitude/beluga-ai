package scaffold

import (
	"fmt"
	"io/fs"
)

// init registers the single built-in template ("basic") with the default
// registry. Registration failure is a programming error (the embed tree
// ships with the binary and cannot vary at runtime), so we panic — the
// same pattern used by the llm/providers/*/init() blocks.
func init() {
	sub, err := fs.Sub(builtinTemplatesFS, "templates/basic")
	if err != nil {
		panic(fmt.Sprintf("beluga: scaffold: fs.Sub(templates/basic): %v", err))
	}
	if err := defaultRegistry.Register("basic", sub); err != nil {
		panic(fmt.Sprintf("beluga: scaffold: register basic: %v", err))
	}
}
