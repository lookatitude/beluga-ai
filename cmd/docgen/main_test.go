package main

import (
	"strings"
	"testing"
)

// TestRenderPage_Frontmatter asserts the YAML frontmatter is emitted with
// quotes escaped.
func TestRenderPage_Frontmatter(t *testing.T) {
	out := renderPage(page{
		Title:       `Core "runtime"`,
		Description: "Streams and errors",
		Packages:    []packageGroup{{ImportPath: "github.com/lookatitude/beluga-ai/v2/core", PkgName: "core"}},
	})

	wantPrefix := "---\ntitle: \"Core \\\"runtime\\\"\"\ndescription: \"Streams and errors\"\n---\n\n"
	if !strings.HasPrefix(out, wantPrefix) {
		t.Fatalf("frontmatter mismatch:\n got: %q\nwant prefix: %q", out[:min(len(out), len(wantPrefix)+20)], wantPrefix)
	}
	if !strings.Contains(out, "import \"github.com/lookatitude/beluga-ai/v2/core\"") {
		t.Fatalf("import block missing:\n%s", out)
	}
	if strings.Contains(out, "## core") {
		t.Fatalf("single-package page must not get a per-package heading:\n%s", out)
	}
}

// TestRenderPage_MultiPackageHeadings asserts multi-package pages get a
// heading per package (provider short name or package name) and a
// separator between sections.
func TestRenderPage_MultiPackageHeadings(t *testing.T) {
	out := renderPage(page{
		Title: "LLM",
		Packages: []packageGroup{
			{ImportPath: "github.com/lookatitude/beluga-ai/v2/llm", PkgName: "llm"},
			{ImportPath: "github.com/lookatitude/beluga-ai/v2/llm/providers/anthropic", PkgName: "anthropic", IsProvider: true},
		},
	})

	for _, want := range []string{"## llm\n", "## anthropic\n", "\n---\n\n"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	if strings.Index(out, "## llm") > strings.Index(out, "## anthropic") {
		t.Errorf("package sections out of order:\n%s", out)
	}
}

// TestWriteIndexSection asserts the index table lists only the pages named
// in the section, linked by slug.
func TestWriteIndexSection(t *testing.T) {
	pages := []page{
		{Filename: "core.md", Title: "core", Description: "Foundation"},
		{Filename: "llm.md", Title: "llm", Description: "Models"},
	}
	var sb strings.Builder
	writeIndexSection(&sb, "Foundation", []string{"core.md"}, pages)

	out := sb.String()
	if !strings.HasPrefix(out, "## Foundation\n\n| Package | Description |\n|---------|-------------|\n") {
		t.Fatalf("missing heading/table header:\n%s", out)
	}
	if !strings.Contains(out, "| [core](./core/) | Foundation |\n") {
		t.Fatalf("core row missing:\n%s", out)
	}
	if strings.Contains(out, "llm") {
		t.Fatalf("page not in the section was listed:\n%s", out)
	}
}
