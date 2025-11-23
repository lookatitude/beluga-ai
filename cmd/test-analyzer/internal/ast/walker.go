package ast

import (
	"go/ast"
	"go/token"
	"strings"
)

// Walker traverses AST nodes and extracts test functions.
type Walker interface {
	Functions() []*ast.FuncDecl
	ast.Visitor // Embed ast.Visitor interface
}

// walker implements the Walker interface.
type walker struct {
	fset      *token.FileSet
	functions []*ast.FuncDecl
}

// NewWalker creates a new ASTWalker instance.
func NewWalker(fset *token.FileSet) Walker {
	return &walker{
		fset:      fset,
		functions: make([]*ast.FuncDecl, 0),
	}
}

// Functions returns all collected test functions.
func (w *walker) Functions() []*ast.FuncDecl {
	return w.functions
}

// Visit implements ast.Visitor interface.
func (w *walker) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return nil
	}

	// Look for function declarations
	switch n := node.(type) {
	case *ast.FuncDecl:
		if w.isTestFunction(n) {
			w.functions = append(w.functions, n)
		}
		return w
	}

	return w
}

// isTestFunction checks if a function is a test function (Test*, Benchmark*, Fuzz*).
func (w *walker) isTestFunction(fn *ast.FuncDecl) bool {
	if fn.Name == nil {
		return false
	}

	name := fn.Name.Name
	return strings.HasPrefix(name, "Test") ||
		strings.HasPrefix(name, "Benchmark") ||
		strings.HasPrefix(name, "Fuzz")
}
