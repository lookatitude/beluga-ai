package mocks

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// PatternExtractor extracts mock patterns from existing test files.
type PatternExtractor interface {
	ExtractMockPattern(ctx context.Context, packagePath string) (*MockPattern, error)
}

// patternExtractor implements PatternExtractor.
type patternExtractor struct{}

// NewPatternExtractor creates a new PatternExtractor.
func NewPatternExtractor() PatternExtractor {
	return &patternExtractor{}
}

// ExtractMockPattern implements PatternExtractor.ExtractMockPattern.
func (e *patternExtractor) ExtractMockPattern(ctx context.Context, packagePath string) (*MockPattern, error) {
	// Look for test_utils.go file
	testUtilsPath := filepath.Join(packagePath, "test_utils.go")
	if _, err := os.Stat(testUtilsPath); os.IsNotExist(err) {
		// Check internal/test_utils.go
		testUtilsPath = filepath.Join(packagePath, "internal", "test_utils.go")
		if _, err := os.Stat(testUtilsPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("test_utils.go not found")
		}
	}

	// Parse test_utils.go
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, testUtilsPath, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parsing test_utils.go: %w", err)
	}

	// Find AdvancedMock struct
	pattern := &MockPattern{}
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.TypeSpec:
			if strings.HasPrefix(node.Name.Name, "AdvancedMock") {
				pattern.StructName = node.Name.Name
				if st, ok := node.Type.(*ast.StructType); ok {
					// Check for embedded mock.Mock
					for _, field := range st.Fields.List {
						if sel, ok := field.Type.(*ast.SelectorExpr); ok {
							if x, ok := sel.X.(*ast.Ident); ok && x.Name == "mock" {
								if sel.Sel.Name == "Mock" {
									pattern.EmbeddedType = "mock.Mock"
								}
							}
						}
					}
				}
			}
			// Look for option type
			if strings.HasPrefix(node.Name.Name, "Mock") && strings.HasSuffix(node.Name.Name, "Option") {
				pattern.OptionsType = node.Name.Name
			}
		case *ast.FuncDecl:
			// Look for constructor
			if strings.HasPrefix(node.Name.Name, "NewAdvancedMock") {
				pattern.ConstructorName = node.Name.Name
			}
		}
		return true
	})

	if pattern.StructName == "" {
		return nil, fmt.Errorf("AdvancedMock pattern not found")
	}

	return pattern, nil
}

