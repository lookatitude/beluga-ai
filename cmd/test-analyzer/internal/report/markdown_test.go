package report

import (
	"context"
	"testing"
	"time"
)

func TestNewMarkdownReportGenerator(t *testing.T) {
	generator := NewMarkdownReportGenerator()
	if generator == nil {
		t.Fatal("NewMarkdownReportGenerator() returned nil")
	}

	if _, ok := generator.(MarkdownReportGenerator); !ok {
		t.Error("NewMarkdownReportGenerator() does not implement MarkdownReportGenerator interface")
	}
}

func TestMarkdownReportGenerator_GenerateMarkdownReport(t *testing.T) {
	ctx := context.Background()
	generator := NewMarkdownReportGenerator()

	t.Run("GenerateBasicMarkdown", func(t *testing.T) {
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

		_, err := generator.GenerateMarkdownReport(ctx, report)
		if err != nil {
			t.Fatalf("GenerateMarkdownReport() error = %v", err)
		}
	})
}
