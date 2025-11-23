package main

import (
	"go/ast"
	"testing"
	"time"
)

func TestTestType_String(t *testing.T) {
	tests := []struct {
		name     string
		testType TestType
		want     string
	}{
		{"Unit", TestTypeUnit, "Unit"},
		{"Integration", TestTypeIntegration, "Integration"},
		{"Load", TestTypeLoad, "Load"},
		{"Unknown", TestType(999), "Unknown"},
		{"Negative", TestType(-1), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.testType.String(); got != tt.want {
				t.Errorf("TestType.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIssueType_String(t *testing.T) {
	tests := []struct {
		name      string
		issueType IssueType
		want      string
	}{
		{"InfiniteLoop", IssueTypeInfiniteLoop, "InfiniteLoop"},
		{"MissingTimeout", IssueTypeMissingTimeout, "MissingTimeout"},
		{"LargeIteration", IssueTypeLargeIteration, "LargeIteration"},
		{"HighConcurrency", IssueTypeHighConcurrency, "HighConcurrency"},
		{"SleepDelay", IssueTypeSleepDelay, "SleepDelay"},
		{"ActualImplementationUsage", IssueTypeActualImplementationUsage, "ActualImplementationUsage"},
		{"MixedMockRealUsage", IssueTypeMixedMockRealUsage, "MixedMockRealUsage"},
		{"MissingMock", IssueTypeMissingMock, "MissingMock"},
		{"BenchmarkHelperUsage", IssueTypeBenchmarkHelperUsage, "BenchmarkHelperUsage"},
		{"Other", IssueTypeOther, "Other"},
		{"Unknown", IssueType(999), "Unknown"},
		{"Negative", IssueType(-1), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.issueType.String(); got != tt.want {
				t.Errorf("IssueType.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSeverity_String(t *testing.T) {
	tests := []struct {
		name     string
		severity Severity
		want     string
	}{
		{"Low", SeverityLow, "Low"},
		{"Medium", SeverityMedium, "Medium"},
		{"High", SeverityHigh, "High"},
		{"Critical", SeverityCritical, "Critical"},
		{"Unknown", Severity(999), "Unknown"},
		{"Negative", Severity(-1), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.severity.String(); got != tt.want {
				t.Errorf("Severity.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFixType_String(t *testing.T) {
	tests := []struct {
		name    string
		fixType FixType
		want    string
	}{
		{"Unknown", FixTypeUnknown, "Unknown"},
		{"AddTimeout", FixTypeAddTimeout, "AddTimeout"},
		{"ReduceIterations", FixTypeReduceIterations, "ReduceIterations"},
		{"OptimizeSleep", FixTypeOptimizeSleep, "OptimizeSleep"},
		{"AddLoopExit", FixTypeAddLoopExit, "AddLoopExit"},
		{"ReplaceWithMock", FixTypeReplaceWithMock, "ReplaceWithMock"},
		{"CreateMock", FixTypeCreateMock, "CreateMock"},
		{"UpdateTestFile", FixTypeUpdateTestFile, "UpdateTestFile"},
		{"Invalid", FixType(999), "Unknown"},
		{"Negative", FixType(-1), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.fixType.String(); got != tt.want {
				t.Errorf("FixType.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFixStatus_String(t *testing.T) {
	tests := []struct {
		name   string
		status FixStatus
		want   string
	}{
		{"Proposed", FixStatusProposed, "Proposed"},
		{"Applied", FixStatusApplied, "Applied"},
		{"Validated", FixStatusValidated, "Validated"},
		{"Failed", FixStatusFailed, "Failed"},
		{"RolledBack", FixStatusRolledBack, "RolledBack"},
		{"Unknown", FixStatus(999), "Unknown"},
		{"Negative", FixStatus(-1), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.String(); got != tt.want {
				t.Errorf("FixStatus.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMockStatus_String(t *testing.T) {
	tests := []struct {
		name   string
		status MockStatus
		want   string
	}{
		{"Template", MockStatusTemplate, "Template"},
		{"Complete", MockStatusComplete, "Complete"},
		{"Validated", MockStatusValidated, "Validated"},
		{"Unknown", MockStatus(999), "Unknown"},
		{"Negative", MockStatus(-1), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.String(); got != tt.want {
				t.Errorf("MockStatus.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTestFile(t *testing.T) {
	t.Run("EmptyTestFile", func(t *testing.T) {
		tf := &TestFile{
			Path:    "test.go",
			Package: "test",
		}

		if tf.Path != "test.go" {
			t.Errorf("Expected Path to be 'test.go', got %v", tf.Path)
		}
		if tf.Package != "test" {
			t.Errorf("Expected Package to be 'test', got %v", tf.Package)
		}
		// Functions is nil by default, which is valid in Go
		if tf.Functions != nil && len(tf.Functions) != 0 {
			t.Errorf("Expected Functions to be nil or empty, got %d", len(tf.Functions))
		}
	})

	t.Run("TestFileWithFunctions", func(t *testing.T) {
		tf := &TestFile{
			Path:    "test_test.go",
			Package: "test",
			Functions: []*TestFunction{
				{Name: "TestOne"},
				{Name: "TestTwo"},
			},
		}

		if len(tf.Functions) != 2 {
			t.Errorf("Expected 2 functions, got %d", len(tf.Functions))
		}
		if tf.Functions[0].Name != "TestOne" {
			t.Errorf("Expected first function to be 'TestOne', got %v", tf.Functions[0].Name)
		}
	})

	t.Run("TestFileWithAST", func(t *testing.T) {
		fileAST := &ast.File{
			Name: &ast.Ident{Name: "test"},
		}

		tf := &TestFile{
			Path:    "test.go",
			Package: "test",
			AST:     fileAST,
		}

		if tf.AST == nil {
			t.Error("Expected AST to be set, got nil")
		}
		if tf.AST.Name.Name != "test" {
			t.Errorf("Expected AST package name to be 'test', got %v", tf.AST.Name.Name)
		}
	})
}

func TestTestFunction(t *testing.T) {
	t.Run("EmptyTestFunction", func(t *testing.T) {
		tf := &TestFunction{
			Name: "TestExample",
			Type: TestTypeUnit,
		}

		if tf.Name != "TestExample" {
			t.Errorf("Expected Name to be 'TestExample', got %v", tf.Name)
		}
		if tf.Type != TestTypeUnit {
			t.Errorf("Expected Type to be TestTypeUnit, got %v", tf.Type)
		}
		// Issues is nil by default, which is valid in Go
		if tf.Issues != nil && len(tf.Issues) != 0 {
			t.Errorf("Expected Issues to be nil or empty, got %d", len(tf.Issues))
		}
	})

	t.Run("TestFunctionWithTimeout", func(t *testing.T) {
		tf := &TestFunction{
			Name:            "TestWithTimeout",
			Type:            TestTypeIntegration,
			HasTimeout:      true,
			TimeoutDuration: 5 * time.Second,
		}

		if !tf.HasTimeout {
			t.Error("Expected HasTimeout to be true")
		}
		if tf.TimeoutDuration != 5*time.Second {
			t.Errorf("Expected TimeoutDuration to be 5s, got %v", tf.TimeoutDuration)
		}
	})

	t.Run("TestFunctionWithIssues", func(t *testing.T) {
		issue := PerformanceIssue{
			Type:        IssueTypeMissingTimeout,
			Severity:    SeverityHigh,
			Description: "Missing timeout",
		}

		tf := &TestFunction{
			Name:   "TestWithIssues",
			Type:   TestTypeUnit,
			Issues: []PerformanceIssue{issue},
		}

		if len(tf.Issues) != 1 {
			t.Errorf("Expected 1 issue, got %d", len(tf.Issues))
		}
		if tf.Issues[0].Type != IssueTypeMissingTimeout {
			t.Errorf("Expected issue type to be MissingTimeout, got %v", tf.Issues[0].Type)
		}
	})

	t.Run("TestFunctionWithFileReference", func(t *testing.T) {
		testFile := &TestFile{
			Path:    "test_test.go",
			Package: "test",
		}

		tf := &TestFunction{
			Name: "TestWithFile",
			Type: TestTypeUnit,
			File: testFile,
		}

		if tf.File == nil {
			t.Error("Expected File to be set, got nil")
		}
		if tf.File.Path != "test_test.go" {
			t.Errorf("Expected File.Path to be 'test_test.go', got %v", tf.File.Path)
		}
	})
}

func TestPerformanceIssue(t *testing.T) {
	t.Run("EmptyPerformanceIssue", func(t *testing.T) {
		issue := PerformanceIssue{
			Type:     IssueTypeMissingTimeout,
			Severity: SeverityHigh,
		}

		if issue.Type != IssueTypeMissingTimeout {
			t.Errorf("Expected Type to be MissingTimeout, got %v", issue.Type)
		}
		if issue.Severity != SeverityHigh {
			t.Errorf("Expected Severity to be High, got %v", issue.Severity)
		}
		// Context is nil by default, which is valid in Go
		// It will be initialized when used
	})

	t.Run("PerformanceIssueWithLocation", func(t *testing.T) {
		location := Location{
			File:      "test_test.go",
			Function:  "TestExample",
			LineStart: 10,
			LineEnd:   20,
		}

		issue := PerformanceIssue{
			Type:        IssueTypeInfiniteLoop,
			Severity:    SeverityCritical,
			Location:    location,
			Description: "Infinite loop detected",
		}

		if issue.Location.File != "test_test.go" {
			t.Errorf("Expected Location.File to be 'test_test.go', got %v", issue.Location.File)
		}
		if issue.Location.Function != "TestExample" {
			t.Errorf("Expected Location.Function to be 'TestExample', got %v", issue.Location.Function)
		}
		if issue.Location.LineStart != 10 {
			t.Errorf("Expected Location.LineStart to be 10, got %v", issue.Location.LineStart)
		}
	})

	t.Run("PerformanceIssueWithContext", func(t *testing.T) {
		issue := PerformanceIssue{
			Type:     IssueTypeSleepDelay,
			Severity: SeverityMedium,
			Context: map[string]interface{}{
				"total_sleep": "120ms",
				"threshold":   "100ms",
			},
		}

		if issue.Context["total_sleep"] != "120ms" {
			t.Errorf("Expected total_sleep to be '120ms', got %v", issue.Context["total_sleep"])
		}
		if issue.Context["threshold"] != "100ms" {
			t.Errorf("Expected threshold to be '100ms', got %v", issue.Context["threshold"])
		}
	})

	t.Run("PerformanceIssueWithFix", func(t *testing.T) {
		fix := &Fix{
			Type:   FixTypeAddTimeout,
			Status: FixStatusProposed,
		}

		issue := PerformanceIssue{
			Type:     IssueTypeMissingTimeout,
			Severity: SeverityHigh,
			Fixable:  true,
			Fix:      fix,
		}

		if !issue.Fixable {
			t.Error("Expected Fixable to be true")
		}
		if issue.Fix == nil {
			t.Error("Expected Fix to be set, got nil")
		}
		if issue.Fix.Type != FixTypeAddTimeout {
			t.Errorf("Expected Fix.Type to be AddTimeout, got %v", issue.Fix.Type)
		}
	})
}

func TestLocation(t *testing.T) {
	t.Run("EmptyLocation", func(t *testing.T) {
		loc := Location{}

		if loc.File != "" {
			t.Errorf("Expected File to be empty, got %v", loc.File)
		}
		if loc.Function != "" {
			t.Errorf("Expected Function to be empty, got %v", loc.Function)
		}
		if loc.LineStart != 0 {
			t.Errorf("Expected LineStart to be 0, got %v", loc.LineStart)
		}
		if loc.LineEnd != 0 {
			t.Errorf("Expected LineEnd to be 0, got %v", loc.LineEnd)
		}
	})

	t.Run("LocationWithAllFields", func(t *testing.T) {
		loc := Location{
			File:      "test_test.go",
			Function:  "TestExample",
			LineStart: 15,
			LineEnd:   25,
		}

		if loc.File != "test_test.go" {
			t.Errorf("Expected File to be 'test_test.go', got %v", loc.File)
		}
		if loc.Function != "TestExample" {
			t.Errorf("Expected Function to be 'TestExample', got %v", loc.Function)
		}
		if loc.LineStart != 15 {
			t.Errorf("Expected LineStart to be 15, got %v", loc.LineStart)
		}
		if loc.LineEnd != 25 {
			t.Errorf("Expected LineEnd to be 25, got %v", loc.LineEnd)
		}
	})
}

func TestFix(t *testing.T) {
	t.Run("EmptyFix", func(t *testing.T) {
		fix := &Fix{
			Type:   FixTypeAddTimeout,
			Status: FixStatusProposed,
		}

		if fix.Type != FixTypeAddTimeout {
			t.Errorf("Expected Type to be AddTimeout, got %v", fix.Type)
		}
		if fix.Status != FixStatusProposed {
			t.Errorf("Expected Status to be Proposed, got %v", fix.Status)
		}
		// Changes is nil by default, which is valid in Go
		if fix.Changes != nil && len(fix.Changes) != 0 {
			t.Errorf("Expected Changes to be nil or empty, got %d", len(fix.Changes))
		}
	})

	t.Run("FixWithChanges", func(t *testing.T) {
		change := CodeChange{
			File:        "test_test.go",
			LineStart:   10,
			LineEnd:     15,
			OldCode:     "time.Sleep(5 * time.Second)",
			NewCode:     "ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)",
			Description: "Add timeout context",
		}

		fix := &Fix{
			Type:    FixTypeAddTimeout,
			Status:  FixStatusProposed,
			Changes: []CodeChange{change},
		}

		if len(fix.Changes) != 1 {
			t.Errorf("Expected 1 change, got %d", len(fix.Changes))
		}
		if fix.Changes[0].File != "test_test.go" {
			t.Errorf("Expected change file to be 'test_test.go', got %v", fix.Changes[0].File)
		}
	})

	t.Run("FixWithBackup", func(t *testing.T) {
		fix := &Fix{
			Type:       FixTypeReplaceWithMock,
			Status:     FixStatusApplied,
			BackupPath: "/tmp/test_test.go.backup",
			AppliedAt:  time.Now(),
		}

		if fix.BackupPath != "/tmp/test_test.go.backup" {
			t.Errorf("Expected BackupPath to be '/tmp/test_test.go.backup', got %v", fix.BackupPath)
		}
		if fix.AppliedAt.IsZero() {
			t.Error("Expected AppliedAt to be set, got zero time")
		}
	})
}

func TestCodeChange(t *testing.T) {
	t.Run("EmptyCodeChange", func(t *testing.T) {
		change := CodeChange{}

		if change.File != "" {
			t.Errorf("Expected File to be empty, got %v", change.File)
		}
		if change.LineStart != 0 {
			t.Errorf("Expected LineStart to be 0, got %v", change.LineStart)
		}
		if change.OldCode != "" {
			t.Errorf("Expected OldCode to be empty, got %v", change.OldCode)
		}
		if change.NewCode != "" {
			t.Errorf("Expected NewCode to be empty, got %v", change.NewCode)
		}
	})

	t.Run("CodeChangeWithAllFields", func(t *testing.T) {
		change := CodeChange{
			File:        "test_test.go",
			LineStart:   20,
			LineEnd:     25,
			OldCode:     "client := http.Client{}",
			NewCode:     "client := mockHTTPClient{}",
			Description: "Replace with mock",
		}

		if change.File != "test_test.go" {
			t.Errorf("Expected File to be 'test_test.go', got %v", change.File)
		}
		if change.LineStart != 20 {
			t.Errorf("Expected LineStart to be 20, got %v", change.LineStart)
		}
		if change.LineEnd != 25 {
			t.Errorf("Expected LineEnd to be 25, got %v", change.LineEnd)
		}
		if change.OldCode != "client := http.Client{}" {
			t.Errorf("Expected OldCode to match, got %v", change.OldCode)
		}
		if change.NewCode != "client := mockHTTPClient{}" {
			t.Errorf("Expected NewCode to match, got %v", change.NewCode)
		}
		if change.Description != "Replace with mock" {
			t.Errorf("Expected Description to be 'Replace with mock', got %v", change.Description)
		}
	})
}

func TestValidationResult(t *testing.T) {
	t.Run("EmptyValidationResult", func(t *testing.T) {
		result := ValidationResult{}

		// Errors is nil by default, which is valid in Go
		if result.Errors != nil && len(result.Errors) != 0 {
			t.Errorf("Expected Errors to be nil or empty, got %d", len(result.Errors))
		}
	})

	t.Run("ValidationResultWithSuccess", func(t *testing.T) {
		fix := &Fix{
			Type:   FixTypeAddTimeout,
			Status: FixStatusApplied,
		}

		result := ValidationResult{
			Fix:                   fix,
			InterfaceCompatible:   true,
			TestsPass:             true,
			ExecutionTimeImproved: true,
			OriginalExecutionTime: 10 * time.Second,
			NewExecutionTime:      5 * time.Second,
			ValidatedAt:           time.Now(),
		}

		if !result.InterfaceCompatible {
			t.Error("Expected InterfaceCompatible to be true")
		}
		if !result.TestsPass {
			t.Error("Expected TestsPass to be true")
		}
		if !result.ExecutionTimeImproved {
			t.Error("Expected ExecutionTimeImproved to be true")
		}
		if result.OriginalExecutionTime != 10*time.Second {
			t.Errorf("Expected OriginalExecutionTime to be 10s, got %v", result.OriginalExecutionTime)
		}
		if result.NewExecutionTime != 5*time.Second {
			t.Errorf("Expected NewExecutionTime to be 5s, got %v", result.NewExecutionTime)
		}
		if result.ValidatedAt.IsZero() {
			t.Error("Expected ValidatedAt to be set, got zero time")
		}
	})

	t.Run("ValidationResultWithErrors", func(t *testing.T) {
		result := ValidationResult{
			InterfaceCompatible: false,
			TestsPass:           false,
			Errors: []error{
				NewAnalyzerError("ValidateFix", ErrCodeInterfaceNotFound, nil),
			},
		}

		if result.InterfaceCompatible {
			t.Error("Expected InterfaceCompatible to be false")
		}
		if result.TestsPass {
			t.Error("Expected TestsPass to be false")
		}
		if len(result.Errors) != 1 {
			t.Errorf("Expected 1 error, got %d", len(result.Errors))
		}
	})
}

func TestMockImplementation(t *testing.T) {
	t.Run("EmptyMockImplementation", func(t *testing.T) {
		mock := MockImplementation{
			Status: MockStatusTemplate,
		}

		if mock.ComponentName != "" {
			t.Errorf("Expected ComponentName to be empty, got %v", mock.ComponentName)
		}
		if mock.InterfaceName != "" {
			t.Errorf("Expected InterfaceName to be empty, got %v", mock.InterfaceName)
		}
		if mock.Status != MockStatusTemplate {
			t.Errorf("Expected Status to be Template, got %v", mock.Status)
		}
		// InterfaceMethods is nil by default, which is valid in Go
		if mock.InterfaceMethods != nil && len(mock.InterfaceMethods) != 0 {
			t.Errorf("Expected InterfaceMethods to be nil or empty, got %d", len(mock.InterfaceMethods))
		}
	})

	t.Run("MockImplementationWithFields", func(t *testing.T) {
		method := MethodSignature{
			Name: "DoSomething",
			Parameters: []Parameter{
				{Name: "ctx", Type: "context.Context"},
			},
			Returns: []Return{
				{Name: "", Type: "error"},
			},
		}

		mock := MockImplementation{
			ComponentName:    "HTTPClient",
			InterfaceName:    "Client",
			Package:          "mocks",
			FilePath:         "mocks/http_client.go",
			Code:             "type mockHTTPClient struct {}",
			InterfaceMethods: []MethodSignature{method},
			Status:           MockStatusComplete,
			GeneratedAt:      time.Now(),
		}

		if mock.ComponentName != "HTTPClient" {
			t.Errorf("Expected ComponentName to be 'HTTPClient', got %v", mock.ComponentName)
		}
		if mock.InterfaceName != "Client" {
			t.Errorf("Expected InterfaceName to be 'Client', got %v", mock.InterfaceName)
		}
		if len(mock.InterfaceMethods) != 1 {
			t.Errorf("Expected 1 method, got %d", len(mock.InterfaceMethods))
		}
		if mock.InterfaceMethods[0].Name != "DoSomething" {
			t.Errorf("Expected method name to be 'DoSomething', got %v", mock.InterfaceMethods[0].Name)
		}
		if mock.GeneratedAt.IsZero() {
			t.Error("Expected GeneratedAt to be set, got zero time")
		}
	})
}

func TestMethodSignature(t *testing.T) {
	t.Run("EmptyMethodSignature", func(t *testing.T) {
		method := MethodSignature{}

		if method.Name != "" {
			t.Errorf("Expected Name to be empty, got %v", method.Name)
		}
		// Parameters and Returns are nil by default, which is valid in Go
		if method.Parameters != nil && len(method.Parameters) != 0 {
			t.Errorf("Expected Parameters to be nil or empty, got %d", len(method.Parameters))
		}
		if method.Returns != nil && len(method.Returns) != 0 {
			t.Errorf("Expected Returns to be nil or empty, got %d", len(method.Returns))
		}
	})

	t.Run("MethodSignatureWithParameters", func(t *testing.T) {
		method := MethodSignature{
			Name: "Process",
			Parameters: []Parameter{
				{Name: "ctx", Type: "context.Context"},
				{Name: "data", Type: "[]byte"},
			},
			Returns: []Return{
				{Name: "result", Type: "string"},
				{Name: "err", Type: "error"},
			},
		}

		if len(method.Parameters) != 2 {
			t.Errorf("Expected 2 parameters, got %d", len(method.Parameters))
		}
		if method.Parameters[0].Name != "ctx" {
			t.Errorf("Expected first parameter name to be 'ctx', got %v", method.Parameters[0].Name)
		}
		if len(method.Returns) != 2 {
			t.Errorf("Expected 2 returns, got %d", len(method.Returns))
		}
		if method.Returns[1].Type != "error" {
			t.Errorf("Expected second return type to be 'error', got %v", method.Returns[1].Type)
		}
	})
}

func TestParameter(t *testing.T) {
	t.Run("EmptyParameter", func(t *testing.T) {
		param := Parameter{}

		if param.Name != "" {
			t.Errorf("Expected Name to be empty, got %v", param.Name)
		}
		if param.Type != "" {
			t.Errorf("Expected Type to be empty, got %v", param.Type)
		}
	})

	t.Run("ParameterWithFields", func(t *testing.T) {
		param := Parameter{
			Name: "ctx",
			Type: "context.Context",
		}

		if param.Name != "ctx" {
			t.Errorf("Expected Name to be 'ctx', got %v", param.Name)
		}
		if param.Type != "context.Context" {
			t.Errorf("Expected Type to be 'context.Context', got %v", param.Type)
		}
	})
}

func TestReturn(t *testing.T) {
	t.Run("EmptyReturn", func(t *testing.T) {
		ret := Return{}

		if ret.Name != "" {
			t.Errorf("Expected Name to be empty, got %v", ret.Name)
		}
		if ret.Type != "" {
			t.Errorf("Expected Type to be empty, got %v", ret.Type)
		}
	})

	t.Run("ReturnWithFields", func(t *testing.T) {
		ret := Return{
			Name: "err",
			Type: "error",
		}

		if ret.Name != "err" {
			t.Errorf("Expected Name to be 'err', got %v", ret.Name)
		}
		if ret.Type != "error" {
			t.Errorf("Expected Type to be 'error', got %v", ret.Type)
		}
	})

	t.Run("ReturnWithoutName", func(t *testing.T) {
		ret := Return{
			Type: "string",
		}

		if ret.Name != "" {
			t.Errorf("Expected Name to be empty, got %v", ret.Name)
		}
		if ret.Type != "string" {
			t.Errorf("Expected Type to be 'string', got %v", ret.Type)
		}
	})
}

func TestAnalysisReport(t *testing.T) {
	t.Run("EmptyAnalysisReport", func(t *testing.T) {
		report := AnalysisReport{}

		if report.PackagesAnalyzed != 0 {
			t.Errorf("Expected PackagesAnalyzed to be 0, got %d", report.PackagesAnalyzed)
		}
		if report.FilesAnalyzed != 0 {
			t.Errorf("Expected FilesAnalyzed to be 0, got %d", report.FilesAnalyzed)
		}
		// Maps and slices are nil by default in Go, which is valid
		// They will be initialized when used
	})

	t.Run("AnalysisReportWithData", func(t *testing.T) {
		report := AnalysisReport{
			PackagesAnalyzed:  4,
			FilesAnalyzed:     35,
			FunctionsAnalyzed: 513,
			IssuesFound:       901,
			IssuesByType: map[IssueType]int{
				IssueTypeMissingTimeout:            492,
				IssueTypeActualImplementationUsage: 186,
			},
			IssuesBySeverity: map[Severity]int{
				SeverityHigh:   500,
				SeverityMedium: 300,
			},
			IssuesByPackage: map[string]int{
				"pkg/llms":   63,
				"pkg/memory": 321,
			},
			FixesApplied:  0,
			FixesFailed:   0,
			ExecutionTime: 100 * time.Millisecond,
			GeneratedAt:   time.Now(),
		}

		if report.PackagesAnalyzed != 4 {
			t.Errorf("Expected PackagesAnalyzed to be 4, got %d", report.PackagesAnalyzed)
		}
		if report.IssuesFound != 901 {
			t.Errorf("Expected IssuesFound to be 901, got %d", report.IssuesFound)
		}
		if report.IssuesByType[IssueTypeMissingTimeout] != 492 {
			t.Errorf("Expected MissingTimeout issues to be 492, got %d", report.IssuesByType[IssueTypeMissingTimeout])
		}
		if report.IssuesBySeverity[SeverityHigh] != 500 {
			t.Errorf("Expected High severity issues to be 500, got %d", report.IssuesBySeverity[SeverityHigh])
		}
		if report.IssuesByPackage["pkg/llms"] != 63 {
			t.Errorf("Expected pkg/llms issues to be 63, got %d", report.IssuesByPackage["pkg/llms"])
		}
		if report.GeneratedAt.IsZero() {
			t.Error("Expected GeneratedAt to be set, got zero time")
		}
	})
}
