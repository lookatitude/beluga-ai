package main

import (
	"context"
	"testing"
	"time"
)

func TestNewReporter(t *testing.T) {
	reporter := NewReporter()
	if reporter == nil {
		t.Fatal("NewReporter() returned nil")
	}

	if _, ok := reporter.(Reporter); !ok {
		t.Error("NewReporter() does not implement Reporter interface")
	}
}

func TestReporter_GenerateReport(t *testing.T) {
	ctx := context.Background()
	reporter := NewReporter()

	t.Run("GenerateJSONReport", func(t *testing.T) {
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

		_, err := reporter.GenerateReport(ctx, report, FormatJSON)
		if err != nil {
			t.Fatalf("GenerateReport() error = %v", err)
		}
	})

	t.Run("GenerateMarkdownReport", func(t *testing.T) {
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

		_, err := reporter.GenerateReport(ctx, report, FormatMarkdown)
		if err != nil {
			t.Fatalf("GenerateReport() error = %v", err)
		}
	})

	t.Run("GenerateHTMLReport", func(t *testing.T) {
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

		_, err := reporter.GenerateReport(ctx, report, FormatHTML)
		if err != nil {
			t.Fatalf("GenerateReport() error = %v", err)
		}
	})

	t.Run("GeneratePlainReport", func(t *testing.T) {
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

		_, err := reporter.GenerateReport(ctx, report, FormatPlain)
		if err != nil {
			t.Fatalf("GenerateReport() error = %v", err)
		}
	})
}
