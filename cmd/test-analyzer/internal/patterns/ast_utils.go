package patterns

import (
	"go/ast"
)

// getFuncDecl extracts the *ast.FuncDecl from TestFunction.AST.
func getFuncDecl(function *TestFunction) *ast.FuncDecl {
	if function.AST == nil {
		return nil
	}
	if fn, ok := function.AST.(*ast.FuncDecl); ok {
		return fn
	}
	return nil
}

// hasInfiniteLoop checks if a for loop has no exit condition.
func hasInfiniteLoop(stmt ast.Stmt) bool {
	forStmt, ok := stmt.(*ast.ForStmt)
	if !ok {
		return false
	}

	// Check if it's a for loop without init, condition, or post
	// Pattern: for { } or for ;; { }
	if forStmt.Init == nil && forStmt.Cond == nil && forStmt.Post == nil {
		return true
	}

	// Check for `for true { }` pattern
	if forStmt.Cond != nil {
		if ident, ok := forStmt.Cond.(*ast.Ident); ok && ident.Name == "true" {
			return true
		}
	}

	return false
}

// findForLoops finds all for loops in a function body.
func findForLoops(body *ast.BlockStmt) []*ast.ForStmt {
	var loops []*ast.ForStmt
	if body == nil {
		return loops
	}

	ast.Inspect(body, func(n ast.Node) bool {
		if forStmt, ok := n.(*ast.ForStmt); ok {
			loops = append(loops, forStmt)
		}
		return true
	})

	return loops
}

// findSelectStmts finds all select statements in a function body.
func findSelectStmts(body *ast.BlockStmt) []*ast.SelectStmt {
	var selects []*ast.SelectStmt
	if body == nil {
		return selects
	}

	ast.Inspect(body, func(n ast.Node) bool {
		if selectStmt, ok := n.(*ast.SelectStmt); ok {
			selects = append(selects, selectStmt)
		}
		return true
	})

	return selects
}

// findCallExprs finds all function call expressions in a function body.
func findCallExprs(body *ast.BlockStmt) []*ast.CallExpr {
	var calls []*ast.CallExpr
	if body == nil {
		return calls
	}

	ast.Inspect(body, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			calls = append(calls, call)
		}
		return true
	})

	return calls
}

// isTimeSleepCall checks if a call expression is time.Sleep.
func isTimeSleepCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}

	return ident.Name == "time" && sel.Sel.Name == "Sleep"
}

// isContextWithTimeout checks if a call expression is context.WithTimeout.
func isContextWithTimeout(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}

	return ident.Name == "context" && sel.Sel.Name == "WithTimeout"
}

// isTimeAfter checks if a call expression is time.After.
func isTimeAfter(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}

	return ident.Name == "time" && sel.Sel.Name == "After"
}

// getLoopIterationCount attempts to extract the iteration count from a for loop.
// Returns -1 if unable to determine.
func getLoopIterationCount(forStmt *ast.ForStmt) int64 {
	if forStmt.Cond == nil {
		return -1 // Infinite loop
	}

	// Check for `for i := 0; i < N; i++` pattern
	binaryExpr, ok := forStmt.Cond.(*ast.BinaryExpr)
	if !ok {
		return -1
	}

	if binaryExpr.Op.String() == "<" {
		if right, ok := binaryExpr.Y.(*ast.BasicLit); ok {
			// Try to parse the number
			if right.Kind.String() == "INT" {
				// This is a simplified check - would need proper parsing
				return -1 // Placeholder - would parse the actual number
			}
		}
	}

	return -1
}

// hasExitCondition checks if a for loop has an exit condition in its body.
func hasExitCondition(forStmt *ast.ForStmt) bool {
	if forStmt.Body == nil {
		return false
	}

	// Check for break, return, or context cancellation
	hasExit := false
	ast.Inspect(forStmt.Body, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.BranchStmt: // break, continue
			hasExit = true
			return false
		case *ast.ReturnStmt:
			hasExit = true
			return false
		case *ast.SelectStmt:
			// Select with context.Done() is an exit condition
			hasExit = true
			return false
		}
		return true
	})

	return hasExit
}
