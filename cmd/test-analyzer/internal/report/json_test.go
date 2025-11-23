package report

import (
	"context"
	"testing"
	"time"
)

func TestNewJSONReportGenerator(t *testing.T) {
	generator := NewJSONReportGenerator()
	if generator == nil {
		t.Fatal("NewJSONReportGenerator() returned nil")
	}

	if _, ok := generator.(JSONReportGenerator); !ok {
		t.Error("NewJSONReportGenerator() does not implement JSONReportGenerator interface")
	}
}

func TestJSONReportGenerator_GenerateJSONReport(t *testing.T) {
	ctx := context.Background()
	generator := NewJSONReportGenerator()

	t.Run("GenerateBasicJSON", func(t *testing.T) {
		report := &AnalysisReport{
			PackagesAnalyzed:  1,
			FilesAnalyzed:     1,
			FunctionsAnalyzed: 1,
			IssuesFound:       0,
			IssuesByType:      make(map[IssueType]int),
			IssuesBySeverity:  make(map[Severity]int),
			IssuesByPackage:   make(map[string]int),
			GeneratedAt:       time.Now(),
		}

		_, err := generator.GenerateJSONReport(ctx, report)
		if err != nil {
			t.Fatalf("GenerateJSONReport() error = %v", err)
		}
	})
}
