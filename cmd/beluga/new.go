package main

import (
	"context"
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/lookatitude/beluga-ai/v2/cmd/beluga/internal/version"
	"github.com/lookatitude/beluga-ai/v2/cmd/beluga/scaffold"
	"github.com/lookatitude/beluga-ai/v2/core"
)

// newNewCmd returns the `beluga new` parent cobra command. It dispatches to
// three kind-specific subcommands (agent, tool, planner), each of which
// generates exactly two files (`<snake>.go` + `<snake>_test.go`) inside a
// detected Beluga project root. See docs/consultations/2026-04-20-loo-149-
// architect-plan.md §T9 for the canonical shape of each stub.
func newNewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "new",
		Short:         "Scaffold a new component (agent, tool, planner) inside a Beluga project",
		Long:          "Create a stub for a new agent, tool, or planner in the current Beluga project.\n\nRun from within a directory scaffolded by `beluga init`.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.AddCommand(newNewAgentCmd(), newNewToolCmd(), newNewPlannerCmd())
	return cmd
}

// componentNameRegex validates `<Name>` arguments passed to `beluga new <kind>`.
// PascalCase: start with uppercase ASCII letter, then letters/digits only.
// Rejects hyphens, underscores, and anything non-ASCII alphanumeric so the
// generated Go identifier is legal and idiomatic.
var componentNameRegex = regexp.MustCompile(`^[A-Z][A-Za-z0-9]*$`)

// validateComponentName enforces the PascalCase allowlist and a 64-byte
// length cap. Returns *core.Error with core.ErrInvalidInput on rejection.
func validateComponentName(name string) error {
	if name == "" {
		return core.Errorf(core.ErrInvalidInput,
			"beluga: component name is empty; a PascalCase name is required")
	}
	if len(name) > 64 {
		return core.Errorf(core.ErrInvalidInput,
			"beluga: component name %q exceeds 64 bytes", name)
	}
	if !componentNameRegex.MatchString(name) {
		return core.Errorf(core.ErrInvalidInput,
			"beluga: component name %q must match PascalCase pattern %s",
			name, componentNameRegex.String())
	}
	return nil
}

// toSnakeCase converts a PascalCase identifier to snake_case for file names.
// "MyAgent" -> "my_agent"; "HTTPServer" -> "http_server"; "URL" -> "url".
// Runs of uppercase letters are treated as a single word; a following
// lowercase letter starts the next word unless the previous run had one
// uppercase letter (canonical ascii PascalCase, no unicode handling).
func toSnakeCase(name string) string {
	var b strings.Builder
	for i, r := range name {
		if i > 0 && r >= 'A' && r <= 'Z' {
			// Lowercase prev was lower → new word. Also, uppercase run
			// followed by a lowercase char starts a new word at the
			// boundary uppercase.
			prev := rune(name[i-1])
			next := rune(0)
			if i+1 < len(name) {
				next = rune(name[i+1])
			}
			prevLower := prev >= 'a' && prev <= 'z'
			nextLower := next >= 'a' && next <= 'z'
			if prevLower || (nextLower && prev >= 'A' && prev <= 'Z') {
				b.WriteByte('_')
			}
		}
		if r >= 'A' && r <= 'Z' {
			b.WriteRune(r + ('a' - 'A'))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// detectProjectPackage reads <projectRoot>/go.mod and returns the Go package
// name the stubs should carry. The simple heuristic: use the last segment of
// the module path. When go.mod is missing or unreadable, fall back to
// "main" — the generated stubs still parse; `go build` reports the mismatch.
func detectProjectPackage(projectRoot string) string {
	goMod := filepath.Join(projectRoot, "go.mod")
	data, err := os.ReadFile(goMod) // #nosec G304 -- path is inside detected project root
	if err != nil {
		return "main"
	}
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "module ") {
			modulePath := strings.TrimSpace(strings.TrimPrefix(trimmed, "module "))
			// Strip optional comment suffix.
			if idx := strings.Index(modulePath, "//"); idx >= 0 {
				modulePath = strings.TrimSpace(modulePath[:idx])
			}
			last := modulePath
			if idx := strings.LastIndex(modulePath, "/"); idx >= 0 {
				last = modulePath[idx+1:]
			}
			// Strip a trailing /vN version suffix ("myproj/v2" → "myproj").
			if matched, _ := regexp.MatchString(`^v\d+$`, last); matched {
				trimmedMod := strings.TrimSuffix(modulePath, "/"+last)
				if idx := strings.LastIndex(trimmedMod, "/"); idx >= 0 {
					last = trimmedMod[idx+1:]
				} else {
					last = trimmedMod
				}
			}
			pkg := sanitizePackageName(last)
			if pkg == "" {
				return "main"
			}
			return pkg
		}
	}
	return "main"
}

// sanitizePackageName reduces an arbitrary module-path final segment to a
// valid Go package identifier: lowercase, ASCII letters/digits only. Any
// disallowed character is dropped. Returns the sanitized string or "" when
// nothing survives.
func sanitizePackageName(s string) string {
	var b strings.Builder
	for i, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + ('a' - 'A'))
		case r >= '0' && r <= '9':
			if i == 0 {
				continue // identifiers cannot start with a digit
			}
			b.WriteRune(r)
		default:
			continue
		}
	}
	return b.String()
}

// writeComponentFiles formats and writes the two generated files to disk.
// It refuses to overwrite an existing file (safer default — `beluga new` is
// additive; users who want to regenerate should delete first). The formatted
// bytes are re-checked with go/parser.ParseFile to surface substitution bugs
// early (a malformed stub ships as a scaffolder bug, same policy as
// scaffold.renderFS — Risk #9).
func writeComponentFiles(projectRoot, snake, sourceBody, testBody string) error {
	srcPath := filepath.Join(projectRoot, snake+".go")
	testPath := filepath.Join(projectRoot, snake+"_test.go")

	for _, p := range []string{srcPath, testPath} {
		if _, err := os.Stat(p); err == nil {
			return core.Errorf(core.ErrInvalidInput,
				"beluga: %s already exists; delete it first or pick a different name", p)
		}
	}

	// Ordered pairs (not a map) so writes happen deterministically: if the
	// second write fails we roll the first back to avoid leaving a
	// half-scaffolded project that would then refuse both names on re-run.
	type filePair struct{ path, body string }
	pairs := []filePair{{srcPath, sourceBody}, {testPath, testBody}}
	var written []string
	for _, fp := range pairs {
		formatted, fmtErr := format.Source([]byte(fp.body))
		if fmtErr != nil {
			rollbackWrites(written)
			return fmt.Errorf(
				"beluga: generated source has a syntax error — this is a bug in the scaffolder, please report it at github.com/lookatitude/beluga-ai/issues (details: %w)",
				fmtErr)
		}
		// Parse the formatted bytes to catch any structural problem the
		// format pass lets through silently.
		fset := token.NewFileSet()
		if _, parseErr := parser.ParseFile(fset, filepath.Base(fp.path), formatted, parser.AllErrors); parseErr != nil {
			rollbackWrites(written)
			return fmt.Errorf(
				"beluga: generated source failed to parse — this is a bug in the scaffolder (details: %w)",
				parseErr)
		}
		// 0o644 is intentional: generated stubs are user-editable Go
		// source, not secret-bearing. path is constructed inside the
		// detected project root under a validated PascalCase identifier.
		if err := os.WriteFile(fp.path, formatted, 0o644); err != nil { // #nosec G306 -- scaffolded user-editable source file
			rollbackWrites(written)
			return fmt.Errorf("beluga: write %q: %w", fp.path, err)
		}
		written = append(written, fp.path)
	}
	return nil
}

// rollbackWrites best-effort deletes any files written during a partial
// component scaffold so a subsequent re-run isn't blocked by the
// "<path> already exists" guard.
func rollbackWrites(paths []string) {
	for _, p := range paths {
		_ = os.Remove(p)
	}
}

// runNewComponent is the shared body of all three `beluga new <kind>`
// commands. It validates the name, detects the project root, computes the
// package/snake names, renders the two files, and prints a two-line summary.
// The caller supplies a renderer that builds the two file bodies given
// (packageName, pascalName, snakeName).
func runNewComponent(
	cmd *cobra.Command,
	kind, name string,
	render func(packageName, pascalName, snakeName string) (source, test string),
) error {
	if err := validateComponentName(name); err != nil {
		return err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("resolve working directory: %w", err)
	}
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	projectRoot, err := scaffold.DetectProjectRoot(ctx, cwd)
	if err != nil {
		return err
	}

	// Version-skew warning: advisory, does not block. Surface on stderr.
	pin, pinErr := scaffold.ReadFrameworkPin(ctx, filepath.Join(projectRoot, "go.mod"))
	if pinErr != nil {
		return pinErr
	}
	if pin != "" && pin != version.Get() {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(),
			"warning: CLI version %s differs from project's go.mod pin %s; regenerated stubs may not match the project's API surface\n",
			version.Get(), pin)
	}

	pkg := detectProjectPackage(projectRoot)
	snake := toSnakeCase(name)
	source, test := render(pkg, name, snake)
	if err := writeComponentFiles(projectRoot, snake, source, test); err != nil {
		return err
	}
	out := cmd.OutOrStdout()
	_, _ = fmt.Fprintf(out, "Created %s %q in %s\n", kind, name, projectRoot)
	_, _ = fmt.Fprintf(out, "Next: open %s/%s.go and replace the stub with your implementation\n", projectRoot, snake)
	return nil
}
