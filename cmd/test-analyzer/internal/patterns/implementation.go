package patterns

import (
	"context"
	"fmt"
	"go/ast"
	"strings"
)

// ImplementationDetector detects actual implementation usage instead of mocks.
type ImplementationDetector interface {
	Detect(ctx context.Context, function *TestFunction) ([]PerformanceIssue, error)
}

// implementationDetector implements ImplementationDetector.
type implementationDetector struct{}

// NewImplementationDetector creates a new ImplementationDetector.
func NewImplementationDetector() ImplementationDetector {
	return &implementationDetector{}
}

// Detect implements ImplementationDetector.Detect.
func (d *implementationDetector) Detect(ctx context.Context, function *TestFunction) ([]PerformanceIssue, error) {
	var issues []PerformanceIssue

	// Only check unit tests
	if function.Type != "Unit" {
		return issues, nil
	}

	// Get function AST
	funcDecl := getFuncDecl(function)
	if funcDecl == nil || funcDecl.Body == nil {
		return issues, nil
	}

	// Analyze AST for actual implementation usage
	actualImpls := d.findActualImplementations(funcDecl.Body)
	mockImpls := d.findMockImplementations(funcDecl.Body)

	// Check if using actual implementations
	if len(actualImpls) > 0 && len(mockImpls) == 0 {
		issue := PerformanceIssue{
			Type:     "ActualImplementationUsage",
			Severity: "High",
			Location: Location{
				File:      getFilePath(function),
				Function:  function.Name,
				LineStart: function.LineStart,
				LineEnd:   function.LineEnd,
			},
			Description: fmt.Sprintf("Unit test %s uses actual implementations instead of mocks: %s", 
				function.Name, strings.Join(actualImpls, ", ")),
			Context: map[string]interface{}{
				"test_type":        function.Type,
				"implementations":   actualImpls,
				"implementation_count": len(actualImpls),
			},
			Fixable: true,
		}

		issues = append(issues, issue)
	}

	// Check for mixed usage
	if len(actualImpls) > 0 && len(mockImpls) > 0 {
		issue := PerformanceIssue{
			Type:     "MixedMockRealUsage",
			Severity: "Medium",
			Location: Location{
				File:      getFilePath(function),
				Function:  function.Name,
				LineStart: function.LineStart,
				LineEnd:   function.LineEnd,
			},
			Description: fmt.Sprintf("Unit test %s mixes mocks and real implementations", function.Name),
			Context: map[string]interface{}{
				"test_type":        function.Type,
				"actual_impls":     actualImpls,
				"mock_impls":       mockImpls,
			},
			Fixable: true,
		}

		issues = append(issues, issue)
	}

	return issues, nil
}

// findActualImplementations finds actual implementation constructor calls and struct literals.
func (d *implementationDetector) findActualImplementations(body *ast.BlockStmt) []string {
	var impls []string

	if body == nil {
		return impls
	}

	// Find all function calls (constructors)
	calls := findCallExprs(body)
	for _, call := range calls {
		if impl := d.isActualImplementation(call); impl != "" {
			impls = append(impls, impl)
		}
	}

	// Find struct literals and composite literals
	ast.Inspect(body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.CompositeLit:
			if impl := d.isActualImplementationLiteral(node); impl != "" {
				impls = append(impls, impl)
			}
		case *ast.UnaryExpr:
			// Check for &Type{} patterns
			if node.Op.String() == "&" {
				if compLit, ok := node.X.(*ast.CompositeLit); ok {
					if impl := d.isActualImplementationLiteral(compLit); impl != "" {
						impls = append(impls, impl)
					}
				}
			}
		}
		return true
	})

	// Remove duplicates
	seen := make(map[string]bool)
	var unique []string
	for _, impl := range impls {
		if !seen[impl] {
			seen[impl] = true
			unique = append(unique, impl)
		}
	}

	return unique
}

// findMockImplementations finds mock implementation constructor calls.
func (d *implementationDetector) findMockImplementations(body *ast.BlockStmt) []string {
	var mocks []string

	if body == nil {
		return mocks
	}

	// Find all function calls
	calls := findCallExprs(body)

	for _, call := range calls {
		// Check for mock patterns: NewMockComponent(), NewAdvancedMockComponent(), etc.
		if mock := d.isMockImplementation(call); mock != "" {
			mocks = append(mocks, mock)
		}
	}

	return mocks
}

// isActualImplementation checks if a call is an actual implementation constructor.
func (d *implementationDetector) isActualImplementation(call *ast.CallExpr) string {
	// Check for New*() patterns that are NOT mocks
	ident, ok := call.Fun.(*ast.Ident)
	if !ok {
		return ""
	}

	name := ident.Name

	// Skip if it's a mock
	if strings.Contains(strings.ToLower(name), "mock") {
		return ""
	}

	// Check for constructor patterns: NewComponent, NewClient, NewService, etc.
	if strings.HasPrefix(name, "New") && len(name) > 3 {
		// Check if it's a common constructor pattern
		// This is a heuristic - actual implementations typically don't have "Mock" in name
		return name
	}

	return ""
}

// isMockImplementation checks if a call is a mock implementation constructor.
func (d *implementationDetector) isMockImplementation(call *ast.CallExpr) string {
	// Check for mock patterns
	ident, ok := call.Fun.(*ast.Ident)
	if !ok {
		// Also check selector expressions: mock.NewComponent()
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
			if ident, ok := sel.X.(*ast.Ident); ok {
				if strings.Contains(strings.ToLower(ident.Name), "mock") {
					return sel.Sel.Name
				}
			}
		}
		return ""
	}

	name := ident.Name

	// Check for mock patterns: NewMock*, NewAdvancedMock*, etc.
	if strings.Contains(strings.ToLower(name), "mock") {
		return name
	}

	return ""
}

// isActualImplementationLiteral checks if a composite literal is an actual implementation.
func (d *implementationDetector) isActualImplementationLiteral(compLit *ast.CompositeLit) string {
	if compLit.Type == nil {
		return ""
	}

	var typeName string

	// Handle different type expressions
	switch typ := compLit.Type.(type) {
	case *ast.Ident:
		// Simple type: http.Client{}
		typeName = typ.Name
	case *ast.SelectorExpr:
		// Qualified type: http.Client{}
		if pkg, ok := typ.X.(*ast.Ident); ok {
			typeName = pkg.Name + "." + typ.Sel.Name
		}
	case *ast.ArrayType:
		// Array/slice literals - check element type
		if ident, ok := typ.Elt.(*ast.Ident); ok {
			typeName = "[]" + ident.Name
		}
	default:
		return ""
	}

	if typeName == "" {
		return ""
	}

	// Skip if it's clearly a mock type
	if strings.Contains(strings.ToLower(typeName), "mock") {
		return ""
	}

	// Check if it's a common standard library type that should be mocked in unit tests
	if d.isStandardLibraryType(typeName) {
		return typeName
	}

	// Check if it's a custom type (not from standard library)
	// Custom types in unit tests should typically use mocks
	if !d.isStandardLibraryPackage(typeName) {
		// Extract package if qualified
		parts := strings.Split(typeName, ".")
		if len(parts) > 1 {
			// Qualified type from a package
			return typeName
		}
		// Unqualified type - might be from same package, check if it's a common implementation pattern
		if d.looksLikeImplementationType(typeName) {
			return typeName
		}
	}

	return ""
}

// isStandardLibraryType checks if a type is from Go standard library.
func (d *implementationDetector) isStandardLibraryType(typeName string) bool {
	// Common standard library packages that should be mocked in unit tests
	stdLibPackages := []string{
		"http", "net", "os", "io", "database/sql", "sql",
		"context", "time", "encoding/json", "encoding/xml",
		"crypto", "tls", "grpc", "rpc",
	}

	// Check if type is qualified with standard library package
	for _, pkg := range stdLibPackages {
		if strings.HasPrefix(typeName, pkg+".") {
			return true
		}
	}

	// Check unqualified common types that are often from stdlib
	commonStdTypes := []string{
		"Client", "Server", "Request", "Response", "Conn", "Listener",
		"File", "Reader", "Writer", "Buffer",
		"DB", "Tx", "Stmt", "Row",
		"Context", "CancelFunc",
	}

	for _, stdType := range commonStdTypes {
		if typeName == stdType || strings.HasSuffix(typeName, "."+stdType) {
			return true
		}
	}

	return false
}

// isStandardLibraryPackage checks if a package name is from standard library.
func (d *implementationDetector) isStandardLibraryPackage(typeName string) bool {
	// Extract package name if qualified
	parts := strings.Split(typeName, ".")
	if len(parts) < 2 {
		return false
	}

	pkg := parts[0]
	stdLibPackages := []string{
		"http", "net", "os", "io", "database", "sql", "context", "time",
		"encoding", "crypto", "tls", "grpc", "rpc", "fmt", "strings",
		"bytes", "bufio", "filepath", "path", "url", "json", "xml",
	}

	for _, stdPkg := range stdLibPackages {
		if pkg == stdPkg {
			return true
		}
	}

	return false
}

// looksLikeImplementationType checks if a type name looks like an implementation type.
func (d *implementationDetector) looksLikeImplementationType(typeName string) bool {
	// Common patterns for implementation types
	implPatterns := []string{
		"Client", "Service", "Manager", "Handler", "Processor",
		"Repository", "Store", "Database", "Cache", "Queue",
		"Provider", "Factory", "Builder", "Executor",
	}

	for _, pattern := range implPatterns {
		if strings.Contains(typeName, pattern) {
			return true
		}
	}

	return false
}

