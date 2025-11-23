package patterns

import (
	"context"
	"fmt"
	"go/ast"
	"strings"
)

// MocksDetector detects missing mock implementations.
type MocksDetector interface {
	Detect(ctx context.Context, function *TestFunction) ([]PerformanceIssue, error)
}

// mocksDetector implements MocksDetector.
type mocksDetector struct{}

// NewMocksDetector creates a new MocksDetector.
func NewMocksDetector() MocksDetector {
	return &mocksDetector{}
}

// Detect implements MocksDetector.Detect.
func (d *mocksDetector) Detect(ctx context.Context, function *TestFunction) ([]PerformanceIssue, error) {
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

	// Find interface usage in the test
	interfaceUsages := d.findInterfaceUsages(funcDecl.Body)

	// For each interface usage, check if a mock exists
	// This is a simplified check - full implementation would scan test_utils.go, internal/mock/, etc.
	for _, iface := range interfaceUsages {
		// Check if mock is already used
		if !d.hasMockUsage(funcDecl.Body, iface) {
			// Check if mock exists (simplified - would check actual files)
			// For now, flag if interface is used but no mock constructor is found
			issue := PerformanceIssue{
				Type:     "MissingMock",
				Severity: "Medium",
				Location: Location{
					File:      getFilePath(function),
					Function:  function.Name,
					LineStart: function.LineStart,
					LineEnd:   function.LineEnd,
				},
				Description: fmt.Sprintf("Unit test %s uses interface %s but no mock implementation found",
					function.Name, iface),
				Context: map[string]interface{}{
					"interface": iface,
					"test_type": function.Type,
				},
				Fixable: true,
			}

			issues = append(issues, issue)
		}
	}

	return issues, nil
}

// findInterfaceUsages finds interface types used in the function.
func (d *mocksDetector) findInterfaceUsages(body *ast.BlockStmt) []string {
	var interfaces []string

	if body == nil {
		return interfaces
	}

	// Look for interface type assertions, interface method calls, etc.
	ast.Inspect(body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.TypeAssertExpr:
			// Check if asserting to an interface
			if ident, ok := node.Type.(*ast.Ident); ok {
				// Heuristic: interfaces often start with 'I' or end with 'er'
				if strings.HasPrefix(ident.Name, "I") || strings.HasSuffix(ident.Name, "er") {
					interfaces = append(interfaces, ident.Name)
				}
			}
		case *ast.CallExpr:
			// Check if calling methods on interfaces
			if sel, ok := node.Fun.(*ast.SelectorExpr); ok {
				// If X is an interface type, record it
				if ident, ok := sel.X.(*ast.Ident); ok {
					// Heuristic check
					if strings.HasPrefix(ident.Name, "I") || strings.HasSuffix(ident.Name, "er") {
						interfaces = append(interfaces, ident.Name)
					}
				}
			}
		}
		return true
	})

	// Remove duplicates
	seen := make(map[string]bool)
	var unique []string
	for _, iface := range interfaces {
		if !seen[iface] {
			seen[iface] = true
			unique = append(unique, iface)
		}
	}

	return unique
}

// hasMockUsage checks if a mock is used for the given interface.
func (d *mocksDetector) hasMockUsage(body *ast.BlockStmt, iface string) bool {
	if body == nil {
		return false
	}

	// Look for mock constructor calls
	calls := findCallExprs(body)

	for _, call := range calls {
		// Check for NewMock*, NewAdvancedMock* patterns
		ident, ok := call.Fun.(*ast.Ident)
		if !ok {
			continue
		}

		name := ident.Name
		// Check if it's a mock constructor for this interface
		if strings.Contains(strings.ToLower(name), "mock") {
			// Extract component name from mock name
			// e.g., NewAdvancedMockLLM -> LLM
			component := d.extractComponentFromMockName(name)
			if component != "" && strings.Contains(iface, component) {
				return true
			}
		}
	}

	return false
}

// extractComponentFromMockName extracts component name from mock constructor name.
func (d *mocksDetector) extractComponentFromMockName(mockName string) string {
	// Patterns: NewMockComponent, NewAdvancedMockComponent
	mockName = strings.TrimPrefix(mockName, "New")
	mockName = strings.TrimPrefix(mockName, "Mock")
	mockName = strings.TrimPrefix(mockName, "AdvancedMock")
	return mockName
}
