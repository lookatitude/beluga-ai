package ast

import (
	"context"
	"fmt"
	"go/ast"
	goparser "go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// TestFile represents a parsed Go test file.
type TestFile struct {
	Path                string
	Package             string
	Functions           []*TestFunction
	HasIntegrationSuffix bool
	AST                 *ast.File
}

// TestFunction represents a test function.
type TestFunction struct {
	Name                   string
	Type                   string // Will be converted from main.TestType
	File                   *TestFile
	LineStart              int
	LineEnd                int
	HasTimeout             bool
	TimeoutDuration        int64 // nanoseconds
	ExecutionTime          int64 // nanoseconds
	UsesActualImplementation bool
	UsesMocks              bool
	MixedUsage             bool
}

// Parser is the interface for parsing Go test files.
type Parser interface {
	// ParseFile parses a Go test file and returns a TestFile.
	ParseFile(ctx context.Context, filePath string) (*TestFile, error)

	// ExtractTestFunctions extracts all test functions from a parsed file.
	ExtractTestFunctions(ctx context.Context, file *TestFile) ([]*TestFunction, error)
}

// parser implements the Parser interface.
type parser struct {
	fset *token.FileSet
}

// NewParser creates a new Parser instance.
func NewParser() Parser {
	return &parser{
		fset: token.NewFileSet(),
	}
}

// ParseFile implements Parser.ParseFile.
func (p *parser) ParseFile(ctx context.Context, filePath string) (*TestFile, error) {
	// Check if context is cancelled
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	// Parse the file
	astFile, err := goparser.ParseFile(p.fset, filePath, content, goparser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parsing file: %w", err)
	}

	// Determine if it's an integration test file
	hasIntegrationSuffix := strings.HasSuffix(filePath, "_integration_test.go")

	// Extract package name
	packageName := ""
	if astFile.Name != nil {
		packageName = astFile.Name.Name
	}

	// Get absolute path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		absPath = filePath
	}

	return &TestFile{
		Path:                absPath,
		Package:             packageName,
		HasIntegrationSuffix: hasIntegrationSuffix,
		AST:                 astFile,
	}, nil
}

// ExtractTestFunctions implements Parser.ExtractTestFunctions.
func (p *parser) ExtractTestFunctions(ctx context.Context, file *TestFile) ([]*TestFunction, error) {
	if file.AST == nil {
		return nil, fmt.Errorf("AST is nil")
	}

	var functions []*TestFunction
	walker := NewWalker(p.fset)

	// Walk the AST to find test functions
	ast.Walk(walker, file.AST)

	// Extract functions from walker
	for _, fn := range walker.Functions() {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		testFunc := &TestFunction{
			Name:     fn.Name.Name,
			File:     file,
			LineStart: p.fset.Position(fn.Pos()).Line,
			LineEnd:   p.fset.Position(fn.End()).Line,
		}

		// Determine function type based on name
		testFunc.Type = determineTestType(fn.Name.Name, file.HasIntegrationSuffix)

		functions = append(functions, testFunc)
	}

	return functions, nil
}

// determineTestType determines the test type based on function name and file suffix.
func determineTestType(funcName string, hasIntegrationSuffix bool) string {
	if strings.HasPrefix(funcName, "Benchmark") {
		return "Load"
	}
	if strings.HasPrefix(funcName, "Fuzz") {
		return "Load"
	}
	if hasIntegrationSuffix {
		return "Integration"
	}
	if strings.HasPrefix(funcName, "Test") {
		return "Unit"
	}
	return "Unit" // Default
}
