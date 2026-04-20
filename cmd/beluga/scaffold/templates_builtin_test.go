package scaffold

import (
	"io/fs"
	"testing"
)

// TestBuiltinTemplates_BasicRegistered asserts the default registry has a
// "basic" entry populated at package init — the scaffolder is not usable
// without it.
func TestBuiltinTemplates_BasicRegistered(t *testing.T) {
	names := DefaultRegistry().Names()
	found := false
	for _, n := range names {
		if n == "basic" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("DefaultRegistry must contain 'basic'; got %v", names)
	}

	fsys, ok := DefaultRegistry().Get("basic")
	if !ok {
		t.Fatalf("Get('basic'): ok=false")
	}
	var count int
	if err := fs.WalkDir(fsys, ".", func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		count++
		return nil
	}); err != nil {
		t.Fatalf("walk basic template: %v", err)
	}
	// 8 expected files: main.go.tmpl, go.mod.tmpl, .env.example.tmpl,
	// .gitignore.tmpl, .beluga/project.yaml.tmpl, Dockerfile.tmpl,
	// Makefile.tmpl, .github/workflows/ci.yml.tmpl.
	if count < 8 {
		t.Errorf("basic template should have at least 8 files, got %d", count)
	}
}
