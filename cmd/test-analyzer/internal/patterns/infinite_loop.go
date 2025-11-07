package patterns

import (
	"context"
	"fmt"
	"go/ast"
)

// InfiniteLoopDetector detects infinite loop patterns.
type InfiniteLoopDetector interface {
	Detect(ctx context.Context, function *TestFunction) ([]PerformanceIssue, error)
}

// infiniteLoopDetector implements InfiniteLoopDetector.
type infiniteLoopDetector struct{}

// NewInfiniteLoopDetector creates a new InfiniteLoopDetector.
func NewInfiniteLoopDetector() InfiniteLoopDetector {
	return &infiniteLoopDetector{}
}

// Detect implements InfiniteLoopDetector.Detect.
func (d *infiniteLoopDetector) Detect(ctx context.Context, function *TestFunction) ([]PerformanceIssue, error) {
	var issues []PerformanceIssue
	
	// Get function AST
	funcDecl := getFuncDecl(function)
	if funcDecl == nil || funcDecl.Body == nil {
		return issues, nil
	}
	
	// Find all for loops in the function
	loops := findForLoops(funcDecl.Body)
	
	for _, loop := range loops {
		// Check if it's an infinite loop
		if hasInfiniteLoop(loop) {
			// Check if it has an exit condition in the body
			if !hasExitCondition(loop) {
				// This is a problematic infinite loop
				issue := PerformanceIssue{
					Type:        "InfiniteLoop",
					Severity:    "Critical",
					Location:    getLocation(function, loop),
					Description: fmt.Sprintf("Infinite loop detected in %s without exit condition", function.Name),
					Context: map[string]interface{}{
						"function": function.Name,
						"line":     function.LineStart,
					},
					Fixable: true,
				}
				issues = append(issues, issue)
			} else {
				// Has exit condition but still an infinite loop - might be intentional
				// Check if it's a ConcurrentTestRunner pattern
				if isConcurrentTestRunnerPattern(loop) {
					issue := PerformanceIssue{
						Type:        "InfiniteLoop",
						Severity:    "High",
						Location:    getLocation(function, loop),
						Description: fmt.Sprintf("Timer-based infinite loop in %s (ConcurrentTestRunner pattern)", function.Name),
						Context: map[string]interface{}{
							"function": function.Name,
							"pattern":  "ConcurrentTestRunner",
						},
						Fixable: true,
					}
					issues = append(issues, issue)
				}
			}
		}
	}
	
	return issues, nil
}

// isConcurrentTestRunnerPattern detects timer-based infinite loops.
func isConcurrentTestRunnerPattern(forStmt *ast.ForStmt) bool {
	if forStmt.Body == nil {
		return false
	}
	
	// Look for select statement with timer channel
	selects := findSelectStmts(forStmt.Body)
	for _, sel := range selects {
		for _, caseClause := range sel.Body.List {
			if comm, ok := caseClause.(*ast.CommClause); ok && comm.Comm != nil {
				// Check if it's a receive operation (ExprStmt with UnaryExpr)
				if exprStmt, ok := comm.Comm.(*ast.ExprStmt); ok {
					if unary, ok := exprStmt.X.(*ast.UnaryExpr); ok && unary.Op.String() == "<-" {
						// This is a receive operation - could be from timer
						return true
					}
				}
				// Also check for assignment receive: case val := <-ch
				if assign, ok := comm.Comm.(*ast.AssignStmt); ok && len(assign.Rhs) > 0 {
					if unary, ok := assign.Rhs[0].(*ast.UnaryExpr); ok && unary.Op.String() == "<-" {
						return true
					}
				}
			}
		}
	}
	
	return false
}

// getLocation creates a Location from function and AST node.
func getLocation(function *TestFunction, node ast.Node) Location {
	// Use function location as base
	loc := Location{
		Package:  getPackageName(function),
		File:      getFilePath(function),
		Function: function.Name,
		LineStart: function.LineStart,
		LineEnd:   function.LineEnd,
	}
	
	// Try to get more specific location from node if possible
	// (would need FileSet for precise positions)
	
	return loc
}

// getPackageName extracts package name from function.
func getPackageName(function *TestFunction) string {
	if function.File != nil {
		// Would need to extract from file path or AST
		return ""
	}
	return ""
}

