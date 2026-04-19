package learning

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"

	"github.com/lookatitude/beluga-ai/v2/core"
)

// CodeExecutor defines the interface for executing dynamically generated tool code.
// Implementations control the sandboxing and safety guarantees of code execution.
type CodeExecutor interface {
	// Execute runs the given code with the provided JSON input string and returns
	// the output as a string. The code is expected to be a valid Go function body.
	Execute(ctx context.Context, code string, input string) (string, error)
}

// ASTValidator performs static analysis on Go source code using go/ast to ensure
// only allowed imports and constructs are used. It acts as a first safety gate
// before code execution.
type ASTValidator struct {
	// allowedImports is the set of import paths that are permitted in generated code.
	allowedImports map[string]bool
}

// NewASTValidator creates a new ASTValidator with the given set of allowed import paths.
// If allowedImports is nil, a default safe set is used: encoding/json, fmt, math,
// strings, strconv, sort, unicode.
func NewASTValidator(allowedImports []string) *ASTValidator {
	allowed := make(map[string]bool, len(allowedImports))
	if len(allowedImports) == 0 {
		// Default safe imports.
		for _, imp := range []string{
			"encoding/json", "fmt", "math", "strings",
			"strconv", "sort", "unicode",
		} {
			allowed[imp] = true
		}
	} else {
		for _, imp := range allowedImports {
			allowed[imp] = true
		}
	}
	return &ASTValidator{allowedImports: allowed}
}

// Validate parses the given Go source code and checks that all imports are in the
// allowed set and that no disallowed constructs (go statements, unsafe operations)
// are used. Returns nil if the code passes validation.
func (v *ASTValidator) Validate(code string) error {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "tool.go", code, parser.AllErrors)
	if err != nil {
		return core.Errorf(core.ErrInvalidInput, "ast validation: parse error: %w", err)
	}

	// Check imports.
	for _, imp := range f.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		if !v.allowedImports[path] {
			return core.Errorf(core.ErrGuardBlocked, "ast validation: disallowed import %q", path)
		}
	}

	// Walk the AST looking for disallowed constructs.
	var walkErr error
	ast.Inspect(f, func(n ast.Node) bool {
		if walkErr != nil {
			return false
		}
		switch n.(type) {
		case *ast.GoStmt:
			walkErr = core.Errorf(core.ErrGuardBlocked, "ast validation: goroutine spawning (go statement) is not allowed")
			return false
		}
		// Check for unsafe package usage in selector expressions.
		if sel, ok := n.(*ast.SelectorExpr); ok {
			if ident, ok := sel.X.(*ast.Ident); ok {
				if ident.Name == "unsafe" {
					walkErr = core.Errorf(core.ErrGuardBlocked, "ast validation: unsafe package usage is not allowed")
					return false
				}
			}
		}
		return true
	})

	return walkErr
}

// NoopExecutor is a CodeExecutor that returns a configurable fixed response.
// It is intended for testing and development purposes.
type NoopExecutor struct {
	// Response is the fixed string returned by Execute.
	Response string
	// Err is the error returned by Execute, if non-nil.
	Err error
}

// Execute returns the configured Response and Err without executing any code.
func (n *NoopExecutor) Execute(_ context.Context, _ string, _ string) (string, error) {
	return n.Response, n.Err
}

// Compile-time check.
var _ CodeExecutor = (*NoopExecutor)(nil)
