package report

import (
	"context"
	"testing"
	"time"
)

func TestNewPlainReportGenerator(t *testing.T) {
	generator := NewPlainReportGenerator()
	if generator == nil {
		t.Fatal("NewPlainReportGenerator() returned nil")
	}

	if _, ok := generator.(PlainReportGenerator); !ok {
		t.Error("NewPlainReportGenerator() does not implement PlainReportGenerator interface")
	}
}

func TestPlainReportGenerator_GeneratePlainReport(t *testing.T) {
	ctx := context.Background()
	generator := NewPlainReportGenerator()

	t.Run("GenerateBasicPlain", func(t *testing.T) {
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

		_, err := generator.GeneratePlainReport(ctx, report)
		if err != nil {
			t.Fatalf("GeneratePlainReport() error = %v", err)
		}
	})
}

