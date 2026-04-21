package scaffold

import (
	"bufio"
	"bytes"
	"context"
	"embed"
	"fmt"
	"go/format"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/lookatitude/beluga-ai/v2/core"
)

// builtinTemplatesFS holds the embedded template tree. The //go:embed
// directive lives here (not in a cobra command file) so the cobra layer
// stays free of embed concerns and the templates subtree travels with the
// scaffolder package. templates_builtin.go in the same package accesses
// this variable directly; there is no accessor.
//
//go:embed all:templates
var builtinTemplatesFS embed.FS

// dotBelugaDir is the name of the Beluga project metadata directory
// (contains project.yaml and related config) in both embedded templates
// and scaffolded output.
const dotBelugaDir = ".beluga"

// goModFilename is the canonical Go module file. Referenced when
// post-processing generated go.mod and when detecting project/workspace
// roots from go.mod content.
const goModFilename = "go.mod"

// applyTemplate substitutes every __BELUGA_<FIELD>__ sentinel in src with
// the corresponding field of vars. The substitution order is fixed (fields
// are listed alphabetically) so output is deterministic across runs —
// required for golden-file tests and reproducible CI.
//
// Unknown sentinels are left in place; callers (tests) assert no leftover
// __BELUGA_ substring exists in the final tree. That failure mode surfaces
// as a test failure, not a runtime error — by design, so adding a new
// sentinel in a template forces a matching ScaffoldVars field addition.
func applyTemplate(src string, vars ScaffoldVars) string {
	// Alphabetical by sentinel name. Keep this list in sync with
	// ScaffoldVars field doc comments in template.go.
	out := src
	out = strings.ReplaceAll(out, "__BELUGA_AGENT_NAME__", vars.AgentName)
	out = strings.ReplaceAll(out, "__BELUGA_VERSION__", vars.BelugaVersion)
	out = strings.ReplaceAll(out, "__BELUGA_MODEL_NAME__", vars.ModelName)
	out = strings.ReplaceAll(out, "__BELUGA_MODULE_PATH__", vars.ModulePath)
	out = strings.ReplaceAll(out, "__BELUGA_PROJECT_NAME__", vars.ProjectName)
	out = strings.ReplaceAll(out, "__BELUGA_PROVIDER_IMPORT__", vars.ProviderImport)
	out = strings.ReplaceAll(out, "__BELUGA_PROVIDER_NAME__", vars.ProviderName)
	out = strings.ReplaceAll(out, "__BELUGA_SCAFFOLDED_AT__", vars.ScaffoldedAt)
	return out
}

// renderFS walks fsys and writes every file under it to targetDir:
//
//   - Each file's bytes pass through applyTemplate.
//   - Files whose rendered path ends in .go additionally pass through
//     go/format.Source; a failure returns a wrapped error with the
//     "this is a bug in the scaffolder" preamble (brief Risk #9).
//   - Files named "<base>.tmpl" have their .tmpl suffix stripped on write.
//   - Paths under ".beluga/" are created with 0o750 perms; other dirs 0o755;
//     files 0o644.
//   - The file named "go.mod" receives post-processing when
//     vars.BelugaVersion == "(devel)": the require line's pseudo-version is
//     removed and a replace directive pointing at the detected workspace
//     root is appended (brief Risk #12).
//   - ctx.Err() is checked before each file so large templates cancel.
//
// When force is false and targetDir already exists with contents, renderFS
// returns *core.Error with ErrInvalidInput before writing any file. The
// error message names --force so CLI tests can match Success Criterion 7.
func renderFS(ctx context.Context, fsys fs.FS, targetDir string, vars ScaffoldVars, force bool) error {
	if err := validateTargetDir(targetDir, force); err != nil {
		return err
	}

	// Ensure root exists before walking. 0o755 is intentional: this is a
	// user-editable project directory scaffolded for the invoking user,
	// not a secret-bearing path. The .beluga/ subdirectory drops to 0o750
	// as a defence in depth; see renderWalkEntry.
	if err := os.MkdirAll(targetDir, 0o755); err != nil { // #nosec G301 -- user-editable scaffolded project root
		return fmt.Errorf("beluga: mkdir target %q: %w", targetDir, err)
	}

	return fs.WalkDir(fsys, ".", func(relPath string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}
		return renderWalkEntry(fsys, relPath, d, targetDir, vars)
	})
}

// validateTargetDir enforces the three-state target-directory policy:
// non-existent → proceed; exists as dir + empty → proceed; exists as dir +
// non-empty → require --force; exists as non-directory → reject.
func validateTargetDir(targetDir string, force bool) error {
	info, statErr := os.Stat(targetDir)
	switch {
	case os.IsNotExist(statErr):
		return nil
	case statErr != nil:
		return fmt.Errorf("beluga: stat target directory %q: %w", targetDir, statErr)
	case !info.IsDir():
		return core.Errorf(core.ErrInvalidInput,
			"beluga: target %q exists but is not a directory", targetDir)
	}
	entries, err := os.ReadDir(targetDir)
	if err != nil {
		return fmt.Errorf("beluga: read target directory %q: %w", targetDir, err)
	}
	if len(entries) > 0 && !force {
		return core.Errorf(core.ErrInvalidInput,
			"beluga: target directory %q is not empty; use --force to overwrite individual files",
			targetDir)
	}
	return nil
}

// renderWalkEntry renders a single embedded-FS entry into targetDir,
// applying sentinel substitution, .tmpl stripping, go.mod postprocessing,
// and the .go format gate. Returns nil for "." (caller skips root).
func renderWalkEntry(fsys fs.FS, relPath string, d fs.DirEntry, targetDir string, vars ScaffoldVars) error {
	// Substitute sentinels in the PATH as well as the file content so
	// templates can rename files per-project if ever needed. None of
	// the S2 templates rely on this, but it's cheap and future-proof.
	renderedRel := applyTemplate(relPath, vars)
	// Strip .tmpl suffix from every segment so nested dirs like
	// ".beluga/project.yaml.tmpl" write correctly.
	renderedRel = stripTmplSuffixes(renderedRel)
	outPath := filepath.Join(targetDir, renderedRel)

	if d.IsDir() {
		return mkdirWithBelugaPerm(outPath, renderedRel)
	}
	return writeTemplatedFile(fsys, relPath, outPath, renderedRel, vars)
}

// mkdirWithBelugaPerm creates a scaffolded directory, using 0o750 for any
// path under the .beluga/ metadata tree and 0o755 elsewhere.
func mkdirWithBelugaPerm(outPath, renderedRel string) error {
	perm := os.FileMode(0o755)
	if strings.HasPrefix(renderedRel, dotBelugaDir) {
		perm = 0o750
	}
	if err := os.MkdirAll(outPath, perm); err != nil {
		return fmt.Errorf("beluga: mkdir %q: %w", outPath, err)
	}
	return nil
}

// writeTemplatedFile reads one template, renders it, applies format/gate
// postprocessing, ensures the parent directory exists, and writes the
// final bytes to outPath. Split out of renderFS to keep its walker
// closure readable and its cognitive complexity bounded.
func writeTemplatedFile(fsys fs.FS, relPath, outPath, renderedRel string, vars ScaffoldVars) error {
	raw, err := fs.ReadFile(fsys, relPath)
	if err != nil {
		return fmt.Errorf("beluga: read template %q: %w", relPath, err)
	}
	rendered := applyTemplate(string(raw), vars)

	// go.mod receives (devel)-version post-processing before any format
	// gate because go.mod is not Go source.
	if filepath.Base(outPath) == goModFilename {
		var postErr error
		rendered, postErr = postProcessGoMod(rendered, vars)
		if postErr != nil {
			return postErr
		}
	}

	// go/format.Source gate on any .go file to catch substitution bugs
	// before invalid Go reaches disk.
	if strings.HasSuffix(outPath, ".go") {
		formatted, fmtErr := format.Source([]byte(rendered))
		if fmtErr != nil {
			return fmt.Errorf(
				"beluga: generated source has a syntax error — this is a bug in the scaffolder, please report it at github.com/lookatitude/beluga-ai/issues (details: %w)",
				fmtErr)
		}
		rendered = string(formatted)
	}

	// Ensure parent directory exists (needed for nested template entries
	// when fs.WalkDir yields files before their parent dir).
	parentPerm := os.FileMode(0o755)
	if strings.Contains(renderedRel, string(filepath.Separator)+dotBelugaDir) ||
		strings.HasPrefix(renderedRel, dotBelugaDir) {
		parentPerm = 0o750
	}
	if err := os.MkdirAll(filepath.Dir(outPath), parentPerm); err != nil {
		return fmt.Errorf("beluga: mkdir %q: %w", filepath.Dir(outPath), err)
	}

	// 0o644 is intentional: scaffolded source files must be readable
	// and editable by the invoking user; they are not secret-bearing.
	// outPath is built from a validated project name and template
	// contents embedded via //go:embed, so G304 does not apply.
	f, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644) // #nosec G302,G304 -- scaffolded user-readable file from embedded template tree
	if err != nil {
		return fmt.Errorf("beluga: open %q: %w", outPath, err)
	}
	if _, writeErr := f.Write([]byte(rendered)); writeErr != nil {
		_ = f.Close()
		return fmt.Errorf("beluga: write %q: %w", outPath, writeErr)
	}
	if closeErr := f.Close(); closeErr != nil {
		return fmt.Errorf("beluga: close %q: %w", outPath, closeErr)
	}
	return nil
}

// stripTmplSuffixes removes the ".tmpl" suffix from every path segment so
// a template named "nested/sub.yaml.tmpl" writes as "nested/sub.yaml".
func stripTmplSuffixes(p string) string {
	parts := strings.Split(p, "/")
	for i, seg := range parts {
		parts[i] = strings.TrimSuffix(seg, ".tmpl")
	}
	return strings.Join(parts, "/")
}

// goModRequireRegex matches the framework's require line in a rendered
// go.mod so we can replace it when the version pin is "(devel)".
var goModRequireRegex = regexp.MustCompile(`(?m)^require github\.com/lookatitude/beluga-ai/v2 .*$`)

// postProcessGoMod rewrites the rendered go.mod when vars.BelugaVersion is
// "(devel)": the bogus pseudo-version require line is stripped and a
// replace directive is appended pointing at the detected workspace root.
// When workspace root cannot be found, the function falls back to pinning
// "v2.0.0-unknown" and writes a comment warning to the generated go.mod —
// the caller surfaces a stderr warning.
func postProcessGoMod(rendered string, vars ScaffoldVars) (string, error) {
	if vars.BelugaVersion != "(devel)" {
		return rendered, nil
	}
	// Find workspace root by walking ancestors of the current working
	// directory; this is the framework repo's root when the CLI is
	// invoked from within a checkout, which is exactly the CI and
	// developer-local case.
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("beluga: getwd: %w", err)
	}
	wsRoot, wsErr := detectWorkspaceRoot(cwd)

	// Replace the require line with a stable "v2.0.0-unknown" pin — the
	// framework module is major-version 2, so Go's minimum-version
	// selector rejects any non-v2 pseudo-version with "invalid: should be
	// v2, not v0". The string literal is the placeholder pinned by the
	// accompanying replace directive when wsRoot is found; it must still
	// parse as a valid semver for the /v2 module path so `go mod tidy`
	// does not fail before the replace directive takes effect.
	rewritten := goModRequireRegex.ReplaceAllString(rendered,
		"require github.com/lookatitude/beluga-ai/v2 v2.0.0-unknown")

	if wsErr != nil {
		// No workspace root found. Append a comment so future `go mod
		// tidy` failures explain themselves.
		rewritten += "\n// NOTE: CLI version was (devel) and no Beluga workspace root was detected.\n"
		rewritten += "// Adjust the require line above or add a replace directive before `go mod tidy`.\n"
		return rewritten, nil
	}
	rewritten += fmt.Sprintf("\nreplace github.com/lookatitude/beluga-ai/v2 => %s\n", wsRoot)
	return rewritten, nil
}

// detectProjectRoot ancestor-walks from startDir upward looking for a
// directory where BOTH .beluga/project.yaml exists AND go.mod contains a
// require line for github.com/lookatitude/beluga-ai/v2. Returns the
// absolute path of that directory. The walk stops at the filesystem root.
//
// Returns *core.Error with ErrNotFound when no ancestor qualifies. Used
// by the beluga new <kind> subcommands.
func detectProjectRoot(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("beluga: resolve startDir: %w", err)
	}
	for {
		if hasBelugaProject(dir) && goModRequiresBeluga(filepath.Join(dir, goModFilename)) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", core.Errorf(core.ErrNotFound,
		"not inside a Beluga project (no .beluga/ directory found — run beluga init first)")
}

// detectWorkspaceRoot ancestor-walks from startDir upward looking for a
// go.mod whose module directive is "github.com/lookatitude/beluga-ai/v2".
// Used only when Options.BelugaVersion == "(devel)" to emit a replace
// directive for a standalone-compilable generated project.
func detectWorkspaceRoot(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("beluga: resolve startDir: %w", err)
	}
	for {
		if goModDeclaresFramework(filepath.Join(dir, goModFilename)) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", core.Errorf(core.ErrNotFound,
		"beluga: workspace root not found (no go.mod with module github.com/lookatitude/beluga-ai/v2)")
}

// hasBelugaProject reports whether .beluga/project.yaml exists in dir.
func hasBelugaProject(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, dotBelugaDir, "project.yaml"))
	return err == nil && !info.IsDir()
}

// goModRequiresBeluga reports whether the given go.mod contains a require
// line for github.com/lookatitude/beluga-ai/v2. The scanner stops at the
// first match; it does not parse the full go.mod grammar (keeping the
// scaffolder free of golang.org/x/mod per brief Decision #9).
func goModRequiresBeluga(goModPath string) bool {
	return scanGoModFor(goModPath, func(line string) bool {
		line = strings.TrimSpace(line)
		return strings.HasPrefix(line, "require github.com/lookatitude/beluga-ai/v2 ") ||
			line == "require github.com/lookatitude/beluga-ai/v2" ||
			strings.HasPrefix(line, "github.com/lookatitude/beluga-ai/v2 ")
	})
}

// goModDeclaresFramework reports whether the given go.mod's module
// directive is exactly "github.com/lookatitude/beluga-ai/v2".
func goModDeclaresFramework(goModPath string) bool {
	return scanGoModFor(goModPath, func(line string) bool {
		line = strings.TrimSpace(line)
		return line == "module github.com/lookatitude/beluga-ai/v2"
	})
}

// scanGoModFor opens goModPath and returns true when match returns true
// for any line. Silently returns false on open/read failures — the
// callers (detectProjectRoot / detectWorkspaceRoot) treat that as
// "ancestor does not qualify" and keep walking.
func scanGoModFor(goModPath string, match func(line string) bool) bool {
	// filepath.Clean guards against any caller passing a relative segment;
	// the scaffolder only invokes this from its own code so this is belt-
	// and-braces for gosec G304.
	cleaned := filepath.Clean(goModPath)
	f, err := os.Open(cleaned) //nolint:gosec // path comes from ancestor walk, not external input
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	// Accept moderately long go.mod lines (replace directives with long paths).
	scanner.Buffer(make([]byte, 0, 4096), 1<<20)
	for scanner.Scan() {
		if match(scanner.Text()) {
			return true
		}
	}
	return false
}

// ensure bytes is used (required by buffered scanner setup on some tool
// versions; kept as a compile-time no-op import guard).
var _ = bytes.MinRead
