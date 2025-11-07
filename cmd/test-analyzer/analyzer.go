package main

import (
	"context"
	"fmt"
	"go/ast"
	"os"
	"path/filepath"
	"strings"
	"time"

	astparser "github.com/lookatitude/beluga-ai/cmd/test-analyzer/internal/ast"
	"github.com/lookatitude/beluga-ai/cmd/test-analyzer/internal/patterns"
)

// Import patterns package for type conversion

// Analyzer is the core interface for analyzing test files and detecting performance issues.
type Analyzer interface {
	// AnalyzePackage analyzes all test files in a package and returns package analysis results.
	AnalyzePackage(ctx context.Context, packagePath string) (*PackageAnalysis, error)

	// AnalyzeFile analyzes a single test file and returns file analysis results.
	AnalyzeFile(ctx context.Context, filePath string) (*FileAnalysis, error)

	// DetectIssues analyzes a test function and returns all detected performance issues.
	DetectIssues(ctx context.Context, function *TestFunction) ([]PerformanceIssue, error)
}

// analyzer implements the Analyzer interface.
type analyzer struct {
	detector patterns.PatternDetector
	parser   astparser.Parser
}

// NewAnalyzer creates a new Analyzer instance with the given dependencies.
func NewAnalyzer(detector patterns.PatternDetector, parser astparser.Parser) Analyzer {
	return &analyzer{
		detector: detector,
		parser:   parser,
	}
}

// AnalyzePackage implements Analyzer.AnalyzePackage.
func (a *analyzer) AnalyzePackage(ctx context.Context, packagePath string) (*PackageAnalysis, error) {
	// Find all test files in the package
	testFiles, err := a.findTestFiles(ctx, packagePath)
	if err != nil {
		return nil, fmt.Errorf("finding test files: %w", err)
	}

	var files []*FileAnalysis
	var allIssues []PerformanceIssue
	totalFiles := 0
	totalFunctions := 0

	// Analyze each test file
	for _, testFile := range testFiles {
		fileAnalysis, err := a.AnalyzeFile(ctx, testFile)
		if err != nil {
			// Log error but continue with other files
			continue
		}

		files = append(files, fileAnalysis)
		allIssues = append(allIssues, fileAnalysis.Issues...)
		totalFiles++
		totalFunctions += len(fileAnalysis.Functions)
	}

	// Build summary
	summary := &AnalysisSummary{
		TotalFiles:     totalFiles,
		TotalFunctions: totalFunctions,
		TotalIssues:    len(allIssues),
		IssuesByType:    make(map[IssueType]int),
		IssuesBySeverity: make(map[Severity]int),
	}

	for _, issue := range allIssues {
		summary.IssuesByType[issue.Type]++
		summary.IssuesBySeverity[issue.Severity]++
	}

	return &PackageAnalysis{
		Package:    packagePath,
		Files:      files,
		Issues:     allIssues,
		Summary:    summary,
		AnalyzedAt: time.Now(),
	}, nil
}

// AnalyzeFile implements Analyzer.AnalyzeFile.
func (a *analyzer) AnalyzeFile(ctx context.Context, filePath string) (*FileAnalysis, error) {
	// Parse the test file
	astFile, err := a.parser.ParseFile(ctx, filePath)
	if err != nil {
		return nil, fmt.Errorf("parsing file: %w", err)
	}

	// Convert internal TestFile to main TestFile
	testFile := &TestFile{
		Path:                astFile.Path,
		Package:             astFile.Package,
		HasIntegrationSuffix: astFile.HasIntegrationSuffix,
		AST:                 astFile.AST,
	}

	// Extract test functions
	astFunctions, err := a.parser.ExtractTestFunctions(ctx, astFile)
	if err != nil {
		return nil, fmt.Errorf("extracting test functions: %w", err)
	}

	// Convert internal TestFunction to main TestFunction
	var functions []*TestFunction
	for _, astFunc := range astFunctions {
		functions = append(functions, &TestFunction{
			Name:                   astFunc.Name,
			Type:                   convertTestType(astFunc.Type),
			File:                   testFile,
			LineStart:              astFunc.LineStart,
			LineEnd:                astFunc.LineEnd,
			HasTimeout:             astFunc.HasTimeout,
			TimeoutDuration:        time.Duration(astFunc.TimeoutDuration),
			ExecutionTime:          time.Duration(astFunc.ExecutionTime),
			UsesActualImplementation: astFunc.UsesActualImplementation,
			UsesMocks:              astFunc.UsesMocks,
			MixedUsage:             astFunc.MixedUsage,
		})
	}

	var allIssues []PerformanceIssue

	// Analyze each function
	for _, function := range functions {
		issues, err := a.DetectIssues(ctx, function)
		if err != nil {
			// Log error but continue with other functions
			continue
		}

		function.Issues = issues
		allIssues = append(allIssues, issues...)
	}

	testFile.Functions = functions

	return &FileAnalysis{
		File:       testFile,
		Functions:  functions,
		Issues:     allIssues,
		AnalyzedAt: time.Now(),
	}, nil
}

// DetectIssues implements Analyzer.DetectIssues.
func (a *analyzer) DetectIssues(ctx context.Context, function *TestFunction) ([]PerformanceIssue, error) {
	// Extract the function AST node from the file
	var funcAST interface{} // *ast.FuncDecl
	if function.File != nil && function.File.AST != nil {
		// Find the function declaration in the file AST
		funcAST = a.findFunctionAST(function.File.AST, function.Name)
	}

	// Convert main TestFunction to internal TestFunction
	internalFunc := &patterns.TestFunction{
		Name:                   function.Name,
		Type:                   function.Type.String(),
		LineStart:              function.LineStart,
		LineEnd:                function.LineEnd,
		HasTimeout:             function.HasTimeout,
		TimeoutDuration:       int64(function.TimeoutDuration),
		ExecutionTime:          int64(function.ExecutionTime),
		UsesActualImplementation: function.UsesActualImplementation,
		UsesMocks:              function.UsesMocks,
		MixedUsage:             function.MixedUsage,
		AST:                    funcAST,
	}
	// Set File field if available
	if function.File != nil {
		internalFunc.File = &patterns.TestFileInfo{
			Path: function.File.Path,
		}
	}

	// Use pattern detector to find all issues
	internalIssues, err := a.detector.DetectAll(ctx, internalFunc)
	if err != nil {
		return nil, fmt.Errorf("detecting patterns: %w", err)
	}

	// Convert internal PerformanceIssue to main PerformanceIssue
	var allIssues []PerformanceIssue
	for _, internalIssue := range internalIssues {
		allIssues = append(allIssues, PerformanceIssue{
			Type:        convertIssueType(internalIssue.Type),
			Severity:    convertSeverity(internalIssue.Severity),
			Location:    convertLocation(internalIssue.Location),
			Description: internalIssue.Description,
			Context:     internalIssue.Context,
			Fixable:     internalIssue.Fixable,
		})
	}

	return allIssues, nil
}

// Helper conversion functions
func convertTestType(s string) TestType {
	switch s {
	case "Unit":
		return TestTypeUnit
	case "Integration":
		return TestTypeIntegration
	case "Load":
		return TestTypeLoad
	default:
		return TestTypeUnit
	}
}

func convertIssueType(s string) IssueType {
	switch s {
	case "InfiniteLoop":
		return IssueTypeInfiniteLoop
	case "MissingTimeout":
		return IssueTypeMissingTimeout
	case "LargeIteration":
		return IssueTypeLargeIteration
	case "HighConcurrency":
		return IssueTypeHighConcurrency
	case "SleepDelay":
		return IssueTypeSleepDelay
	case "ActualImplementationUsage":
		return IssueTypeActualImplementationUsage
	case "MixedMockRealUsage":
		return IssueTypeMixedMockRealUsage
	case "MissingMock":
		return IssueTypeMissingMock
	case "BenchmarkHelperUsage":
		return IssueTypeBenchmarkHelperUsage
	default:
		return IssueTypeOther
	}
}

func convertSeverity(s string) Severity {
	switch s {
	case "Low":
		return SeverityLow
	case "Medium":
		return SeverityMedium
	case "High":
		return SeverityHigh
	case "Critical":
		return SeverityCritical
	default:
		return SeverityMedium
	}
}

func convertLocation(l patterns.Location) Location {
	return Location{
		Package:     l.Package,
		File:        l.File,
		Function:    l.Function,
		LineStart:   l.LineStart,
		LineEnd:     l.LineEnd,
		ColumnStart: l.ColumnStart,
		ColumnEnd:   l.ColumnEnd,
	}
}

// findTestFiles finds all test files in a package directory.
func (a *analyzer) findTestFiles(ctx context.Context, packagePath string) ([]string, error) {
	// This is a placeholder - will be implemented with filepath.Walk
	// For now, return empty slice
	var testFiles []string

	err := filepath.Walk(packagePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(path, "_test.go") {
			testFiles = append(testFiles, path)
		}

		return nil
	})

	return testFiles, err
}

// findFunctionAST finds the AST node for a function by name in a file.
func (a *analyzer) findFunctionAST(fileAST *ast.File, funcName string) *ast.FuncDecl {
	for _, decl := range fileAST.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			if fn.Name != nil && fn.Name.Name == funcName {
				return fn
			}
		}
	}
	return nil
}

