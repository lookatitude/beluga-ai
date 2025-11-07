package ast

import (
	"context"
	"go/ast"
	"go/token"
)

// Analyzer analyzes function AST for patterns.
type Analyzer interface {
	// AnalyzeFunction analyzes a function AST for patterns.
	AnalyzeFunction(ctx context.Context, fn *ast.FuncDecl, fset *token.FileSet) (*FunctionAnalysis, error)
}

// FunctionAnalysis contains analysis results for a function.
type FunctionAnalysis struct {
	HasTimeout             bool
	TimeoutDuration        int64 // nanoseconds
	UsesActualImplementation bool
	UsesMocks              bool
	MixedUsage             bool
	HasInfiniteLoop        bool
	HasLargeIteration      bool
	HasSleepDelays         bool
	TotalSleepDuration     int64 // nanoseconds
}

// analyzer implements the Analyzer interface.
type analyzer struct{}

// NewAnalyzer creates a new Analyzer instance.
func NewAnalyzer() Analyzer {
	return &analyzer{}
}

// AnalyzeFunction implements Analyzer.AnalyzeFunction.
func (a *analyzer) AnalyzeFunction(ctx context.Context, fn *ast.FuncDecl, fset *token.FileSet) (*FunctionAnalysis, error) {
	if fn == nil {
		return nil, nil
	}

	analysis := &FunctionAnalysis{}

	// Walk the function body to analyze patterns
	if fn.Body != nil {
		visitor := newFunctionVisitor(analysis, fset)
		ast.Walk(visitor, fn.Body)
	}

	return analysis, nil
}

// functionVisitor visits AST nodes within a function body.
type functionVisitor struct {
	analysis *FunctionAnalysis
	fset     *token.FileSet
}

// newFunctionVisitor creates a new function visitor.
func newFunctionVisitor(analysis *FunctionAnalysis, fset *token.FileSet) *functionVisitor {
	return &functionVisitor{
		analysis: analysis,
		fset:     fset,
	}
}

// Visit implements ast.Visitor interface.
func (v *functionVisitor) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return nil
	}

	switch n := node.(type) {
	case *ast.CallExpr:
		v.analyzeCall(n)
	case *ast.ForStmt:
		v.analyzeForLoop(n)
	case *ast.RangeStmt:
		v.analyzeRangeLoop(n)
	}

	return v
}

// analyzeCall analyzes a function call.
func (v *functionVisitor) analyzeCall(call *ast.CallExpr) {
	// Check for context.WithTimeout
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		if x, ok := sel.X.(*ast.Ident); ok && x.Name == "context" {
			if sel.Sel.Name == "WithTimeout" {
				v.analysis.HasTimeout = true
			}
		}
	}

	// Check for time.Sleep
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		if x, ok := sel.X.(*ast.Ident); ok && x.Name == "time" {
			if sel.Sel.Name == "Sleep" {
				v.analysis.HasSleepDelays = true
				// Try to extract duration (simplified)
				if len(call.Args) > 0 {
					// This is a simplified check - full implementation would need type checking
					v.analysis.TotalSleepDuration += 100000000 // Assume 100ms default
				}
			}
		}
	}
}

// analyzeForLoop analyzes a for loop.
func (v *functionVisitor) analyzeForLoop(loop *ast.ForStmt) {
	// Check for infinite loop (no condition, no init, no post)
	if loop.Init == nil && loop.Cond == nil && loop.Post == nil {
		v.analysis.HasInfiniteLoop = true
	}
}

// analyzeRangeLoop analyzes a range loop.
func (v *functionVisitor) analyzeRangeLoop(loop *ast.RangeStmt) {
	// Range loops are generally safe, but we could check for large collections
	// This would require more sophisticated analysis
}

