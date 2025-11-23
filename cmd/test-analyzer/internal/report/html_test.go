package report

import (
	"context"
	"testing"
	"time"
)

func TestNewHTMLReportGenerator(t *testing.T) {
	generator := NewHTMLReportGenerator()
	if generator == nil {
		t.Fatal("NewHTMLReportGenerator() returned nil")
	}

	if _, ok := generator.(HTMLReportGenerator); !ok {
		t.Error("NewHTMLReportGenerator() does not implement HTMLReportGenerator interface")
	}
}

func TestHTMLReportGenerator_GenerateHTMLReport(t *testing.T) {
	ctx := context.Background()
	generator := NewHTMLReportGenerator()

	t.Run("GenerateBasicHTML", func(t *testing.T) {
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

		_, err := generator.GenerateHTMLReport(ctx, report)
		if err != nil {
			t.Fatalf("GenerateHTMLReport() error = %v", err)
		}
	})
}
