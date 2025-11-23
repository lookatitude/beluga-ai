package ast

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"strings"
)

// Extractor extracts test functions from AST.
// Note: This interface is defined but the main extraction is done in parser.go
// This file provides additional extraction utilities if needed.
type Extractor interface {
	// ExtractTestFunctions extracts Test*, Benchmark*, Fuzz* functions from AST.
	ExtractTestFunctions(ctx context.Context, astFile *ast.File, fset *token.FileSet) ([]*TestFunction, error)
}

// extractor implements the Extractor interface.
type extractor struct{}

// NewExtractor creates a new Extractor instance.
func NewExtractor() Extractor {
	return &extractor{}
}

// ExtractTestFunctions implements Extractor.ExtractTestFunctions.
func (e *extractor) ExtractTestFunctions(ctx context.Context, astFile *ast.File, fset *token.FileSet) ([]*TestFunction, error) {
	if astFile == nil {
		return nil, fmt.Errorf("AST file is nil")
	}

	var functions []*TestFunction
	walker := NewWalker(fset).(*walker)

	// Walk the AST
	ast.Walk(walker, astFile)

	// Convert AST function declarations to TestFunction
	for _, fn := range walker.Functions() {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		testFunc := &TestFunction{
			Name:      fn.Name.Name,
			LineStart: fset.Position(fn.Pos()).Line,
			LineEnd:   fset.Position(fn.End()).Line,
		}

		// Determine function type
		testFunc.Type = determineFunctionType(fn.Name.Name)

		functions = append(functions, testFunc)
	}

	return functions, nil
}

// determineFunctionType determines the type of test function.
func determineFunctionType(name string) string {
	if strings.HasPrefix(name, "Benchmark") {
		return "Load"
	}
	if strings.HasPrefix(name, "Fuzz") {
		return "Load"
	}
	if strings.HasPrefix(name, "Test") {
		return "Unit" // Will be adjusted based on file suffix
	}
	return "Unit"
}
