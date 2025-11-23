package patterns

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
)

// IterationsDetector detects large iteration counts.
type IterationsDetector interface {
	Detect(ctx context.Context, function *TestFunction) ([]PerformanceIssue, error)
}

// iterationsDetector implements IterationsDetector.
type iterationsDetector struct {
	simpleThreshold  int
	complexThreshold int
}

// NewIterationsDetector creates a new IterationsDetector.
func NewIterationsDetector() IterationsDetector {
	return &iterationsDetector{
		simpleThreshold:  100,
		complexThreshold: 20,
	}
}

// Detect implements IterationsDetector.Detect.
func (d *iterationsDetector) Detect(ctx context.Context, function *TestFunction) ([]PerformanceIssue, error) {
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

		// Extract iteration count
		iterCount := d.extractIterationCount(loop)
		if iterCount <= 0 {
			continue
		}

		// Determine if operations in loop are simple or complex
		isComplex := d.isComplexLoop(loop)

		threshold := d.simpleThreshold
		severity := "Medium"
		if isComplex {
			threshold = d.complexThreshold
			severity = "High"
		}

		if iterCount > int64(threshold) {
			issue := PerformanceIssue{
				Type:     "LargeIteration",
				Severity: severity,
				Location: getLocation(function, loop),
				Description: fmt.Sprintf("Loop in %s has %d iterations (threshold: %d)",
					function.Name, iterCount, threshold),
				Context: map[string]interface{}{
					"iterations": iterCount,
					"threshold":  threshold,
					"is_complex": isComplex,
					"function":   function.Name,
				},
				Fixable: true,
			}

			issues = append(issues, issue)
		}
	}

	return issues, nil
}

// extractIterationCount attempts to extract the iteration count from a for loop.
func (d *iterationsDetector) extractIterationCount(forStmt *ast.ForStmt) int64 {
	if forStmt.Cond == nil {
		return -1
	}

	// Check for `for i := 0; i < N; i++` pattern
	binaryExpr, ok := forStmt.Cond.(*ast.BinaryExpr)
	if !ok {
		return -1
	}

	// Look for < or <= operator
	if binaryExpr.Op != token.LSS && binaryExpr.Op != token.LEQ {
		return -1
	}

	// Check if right side is a constant
	if basicLit, ok := binaryExpr.Y.(*ast.BasicLit); ok && basicLit.Kind == token.INT {
		if val, err := strconv.ParseInt(basicLit.Value, 10, 64); err == nil {
			return val
		}
	}

	// Check for range loops: for i := range slice
	if rangeStmt, ok := forStmt.Init.(*ast.RangeStmt); ok {
		// For range loops, we can't determine count statically
		// But we can check if it's a slice/array literal
		if compLit, ok := rangeStmt.X.(*ast.CompositeLit); ok {
			// Estimate based on number of elements in literal
			return int64(len(compLit.Elts))
		}
	}

	return -1
}

// isComplexLoop determines if a loop contains complex operations.
func (d *iterationsDetector) isComplexLoop(forStmt *ast.ForStmt) bool {
	if forStmt.Body == nil {
		return false
	}

	// Count function calls, channel operations, etc.
	complexity := 0

	ast.Inspect(forStmt.Body, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.CallExpr:
			complexity++
		case *ast.SendStmt, *ast.UnaryExpr: // Channel operations
			complexity++
		case *ast.SelectStmt:
			complexity += 2 // Select is complex
		case *ast.GoStmt: // Goroutines
			complexity += 2
		}
		return true
	})

	// Consider complex if it has multiple function calls or channel operations
	return complexity > 1
}
