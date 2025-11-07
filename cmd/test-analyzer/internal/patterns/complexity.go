package patterns

import (
	"context"
	"fmt"
	"go/ast"
	"strings"
)

// ComplexityDetector detects operation complexity within loops.
type ComplexityDetector interface {
	Detect(ctx context.Context, function *TestFunction) ([]PerformanceIssue, error)
}

// complexityDetector implements ComplexityDetector.
type complexityDetector struct{}

// NewComplexityDetector creates a new ComplexityDetector.
func NewComplexityDetector() ComplexityDetector {
	return &complexityDetector{}
}

// Detect implements ComplexityDetector.Detect.
func (d *complexityDetector) Detect(ctx context.Context, function *TestFunction) ([]PerformanceIssue, error) {
	var issues []PerformanceIssue

	// Get function AST
	funcDecl := getFuncDecl(function)
	if funcDecl == nil || funcDecl.Body == nil {
		return issues, nil
	}

	// Find all for loops
	loops := findForLoops(funcDecl.Body)

	for _, loop := range loops {
		// Skip infinite loops (handled by InfiniteLoopDetector)
		if hasInfiniteLoop(loop) {
			continue
		}

		// Check for complex operations in the loop body
		complexOps := d.findComplexOperations(loop.Body)
		if len(complexOps) > 0 {
			issue := PerformanceIssue{
				Type:        "HighConcurrency",
				Severity:    "High",
				Location:    getLocation(function, loop),
				Description: fmt.Sprintf("Loop in %s contains %d complex operation(s): %s", 
					function.Name, len(complexOps), strings.Join(complexOps, ", ")),
				Context: map[string]interface{}{
					"function":      function.Name,
					"operations":    complexOps,
					"operation_count": len(complexOps),
				},
				Fixable: true,
			}

			issues = append(issues, issue)
		}
	}

	return issues, nil
}

// findComplexOperations finds complex operations in a block (network, I/O, DB, etc.).
func (d *complexityDetector) findComplexOperations(body *ast.BlockStmt) []string {
	var operations []string

	if body == nil {
		return operations
	}

	// Find all function calls
	calls := findCallExprs(body)

	for _, call := range calls {
		op := d.classifyOperation(call)
		if op != "" {
			operations = append(operations, op)
		}
	}

	return operations
}

// classifyOperation classifies a function call as a complex operation.
func (d *complexityDetector) classifyOperation(call *ast.CallExpr) string {
	// Get the function name/selector
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return ""
	}

	// Get the package/type name
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return ""
	}

	pkg := ident.Name
	method := sel.Sel.Name

	// Network operations
	if d.isNetworkOperation(pkg, method) {
		return fmt.Sprintf("%s.%s (network)", pkg, method)
	}

	// File I/O operations
	if d.isFileIOOperation(pkg, method) {
		return fmt.Sprintf("%s.%s (file I/O)", pkg, method)
	}

	// Database operations
	if d.isDatabaseOperation(pkg, method) {
		return fmt.Sprintf("%s.%s (database)", pkg, method)
	}

	// External service calls (common patterns)
	if d.isExternalServiceCall(pkg, method) {
		return fmt.Sprintf("%s.%s (external service)", pkg, method)
	}

	return ""
}

// isNetworkOperation checks if a call is a network operation.
func (d *complexityDetector) isNetworkOperation(pkg, method string) bool {
	networkPackages := []string{"http", "grpc", "net", "websocket", "rpc"}
	for _, npkg := range networkPackages {
		if pkg == npkg {
			return true
		}
	}
	return false
}

// isFileIOOperation checks if a call is a file I/O operation.
func (d *complexityDetector) isFileIOOperation(pkg, method string) bool {
	fileIOPackages := []string{"os", "io", "ioutil", "bufio", "filepath"}
	for _, fpkg := range fileIOPackages {
		if pkg == fpkg {
			// Check for read/write operations
			ioMethods := []string{"Read", "Write", "Open", "Create", "ReadFile", "WriteFile", "ReadDir"}
			for _, m := range ioMethods {
				if strings.HasPrefix(method, m) {
					return true
				}
			}
		}
	}
	return false
}

// isDatabaseOperation checks if a call is a database operation.
func (d *complexityDetector) isDatabaseOperation(pkg, method string) bool {
	dbPackages := []string{"sql", "database", "gorm", "pgx", "mongo"}
	for _, dpkg := range dbPackages {
		if pkg == dpkg || strings.Contains(pkg, dpkg) {
			return true
		}
	}
	return false
}

// isExternalServiceCall checks if a call is to an external service.
func (d *complexityDetector) isExternalServiceCall(pkg, method string) bool {
	// Common external service patterns
	externalPatterns := []string{"client", "service", "api", "sdk"}
	for _, pattern := range externalPatterns {
		if strings.Contains(strings.ToLower(pkg), pattern) {
			return true
		}
	}
	return false
}

