// Package scaffold generates a Beluga project from a named template. It is a
// Layer 7 subpackage of the beluga CLI — stdlib only, zero framework-domain
// imports, and the single entry point for the `beluga init` cobra command.
//
// The scaffolder is intentionally synchronous: one filesystem walk, one write
// per file. There is no streaming data to model, so the framework's
// iter.Seq2 invariant does not apply here (the call is bounded, deterministic,
// and fully driven by the embedded template tree in renderer.go).
package scaffold

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/lookatitude/beluga-ai/v2/core"
)

// Options controls a single Scaffold run. All fields except Force are
// required. The caller (the cobra RunE in cmd/beluga/init.go) resolves
// defaults — TargetDir defaults to filepath.Join(cwd, ProjectName),
// ModulePath defaults to "example.com/"+ProjectName, BelugaVersion comes
// from version.Get(), and ScaffoldedAt comes from time.Now() — so tests can
// drive every field with fixed values and keep golden output stable.
type Options struct {
	// ProjectName is the validated project identifier. Must satisfy
	// ValidateProjectName before Scaffold is called.
	ProjectName string

	// Template is the registered template name (e.g. "basic"). The
	// cobra layer defaults this to "basic" when the flag is empty.
	Template string

	// ModulePath is the Go module path written into go.mod. Must satisfy
	// ValidateModulePath before Scaffold is called when set via --module.
	ModulePath string

	// TargetDir is an absolute path to the project root that will be
	// created. The three-state overwrite policy applies (non-existent
	// → create; empty → warn on stderr and proceed; non-empty without
	// Force → reject with an error naming --force).
	TargetDir string

	// Force, when true, overwrites individual files in a non-empty
	// target directory. It never triggers os.RemoveAll — the scaffolder
	// only opens individual files with O_CREATE|O_TRUNC.
	Force bool

	// BelugaVersion is the framework version string to pin in go.mod.
	// When "(devel)" the renderer emits a replace directive instead of
	// a pseudo-version require line.
	BelugaVersion string

	// ScaffoldedAt is stamped into .beluga/project.yaml. Tests fix this
	// to a known UTC time so golden files stay deterministic.
	ScaffoldedAt time.Time
}

// projectNameRegex implements the allowlist from the brief Decision #7 /
// specialist-security-architect §Q1. The alternation permits names between
// 2 and 64 characters, all lowercase letters/digits/hyphens, starting with
// a letter and ending with an alphanumeric character. The second branch
// covers the minimum 2-character case (e.g. "ab") because the first branch
// requires a middle segment.
var projectNameRegex = regexp.MustCompile(`^[a-z][a-z0-9-]{0,62}[a-z0-9]$`)

// windowsReservedNames is the case-insensitive blocklist applied as a
// secondary check after the regex. Projects on Windows cannot be named
// these identifiers regardless of case.
var windowsReservedNames = map[string]struct{}{
	"con": {}, "prn": {}, "aux": {}, "nul": {},
	"com1": {}, "com2": {}, "com3": {}, "com4": {},
	"com5": {}, "com6": {}, "com7": {}, "com8": {}, "com9": {},
	"lpt1": {}, "lpt2": {}, "lpt3": {}, "lpt4": {},
	"lpt5": {}, "lpt6": {}, "lpt7": {}, "lpt8": {}, "lpt9": {},
}

// ValidateProjectName enforces the project-name allowlist:
//
//   - non-empty (rejected first with a clear message)
//   - length ≤ 64 bytes (rejected before the regex as an anti-ReDoS measure)
//   - matches ^[a-z][a-z0-9-]{0,62}[a-z0-9]$
//   - not a Windows reserved name (CON, PRN, AUX, NUL, COM1-9, LPT1-9)
//
// It returns *core.Error with code core.ErrInvalidInput. It never sanitises;
// callers must reject on error and not attempt a fallback.
func ValidateProjectName(name string) error {
	if name == "" {
		return core.Errorf(core.ErrInvalidInput,
			"beluga: project name is empty; a project name is required")
	}
	if len(name) > 64 {
		return core.Errorf(core.ErrInvalidInput,
			"beluga: project name %q exceeds 64 bytes; the maximum is 64", name)
	}
	if !projectNameRegex.MatchString(name) {
		return core.Errorf(core.ErrInvalidInput,
			"beluga: project name %q does not match the allowed pattern %s",
			name, projectNameRegex.String())
	}
	if _, reserved := windowsReservedNames[strings.ToLower(name)]; reserved {
		return core.Errorf(core.ErrInvalidInput,
			"beluga: project name %q is a Windows reserved name; pick a different name", name)
	}
	return nil
}

// modulePathSegmentRegex matches a single path element of a Go module path.
// Module-path grammar (golang.org/ref/mod#go-mod-file-ident) allows letters,
// digits, and a small set of punctuation (`. - _ ~`). We conservatively
// accept alphanumerics plus `. - _ ~` and require at least one character.
// Note: real `golang.org/x/mod` grammar is richer but this regex rejects
// everything dangerous (spaces, shell metacharacters, control bytes) while
// accepting all paths observed in the wild for Github/Gitlab/example.com.
var modulePathSegmentRegex = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._~-]*$`)

// ValidateModulePath validates a Go module path. The grammar is distinct
// from the project-name allowlist: module paths permit mixed case, dots,
// slashes, tildes, hyphens, and underscores. We reject anything that could
// produce a shell-injection or whitespace-bearing path.
func ValidateModulePath(path string) error {
	if path == "" {
		return core.Errorf(core.ErrInvalidInput,
			"beluga: module path is empty; provide --module <path> or omit for example.com/<name>")
	}
	if len(path) > 256 {
		return core.Errorf(core.ErrInvalidInput,
			"beluga: module path exceeds 256 bytes")
	}
	// Reject control characters, spaces, and tabs up front.
	for _, r := range path {
		if r < 0x21 || r == 0x7f {
			return core.Errorf(core.ErrInvalidInput,
				"beluga: module path %q contains whitespace or control characters", path)
		}
	}
	segments := strings.Split(path, "/")
	for _, seg := range segments {
		if seg == "" {
			return core.Errorf(core.ErrInvalidInput,
				"beluga: module path %q has empty segments (leading/trailing/double slash)", path)
		}
		if !modulePathSegmentRegex.MatchString(seg) {
			return core.Errorf(core.ErrInvalidInput,
				"beluga: module path %q contains segment %q with disallowed characters", path, seg)
		}
	}
	return nil
}

// Scaffold renders the template identified by opts.Template into
// opts.TargetDir. Returns *core.Error for user-correctable issues
// (unknown template, non-empty target without --force) and a plain
// fmt.Errorf when go/format.Source fails on a generated .go file
// (per the S1 CLI-local-error convention — those are scaffolder bugs,
// not user input errors).
func Scaffold(ctx context.Context, opts Options) error {
	// Resolve the template FS via the default registry so unknown template
	// names are reported clearly before any filesystem work begins.
	fsys, ok := DefaultRegistry().Get(opts.Template)
	if !ok {
		return core.Errorf(core.ErrInvalidInput,
			"beluga: unknown template %q; available templates: %s",
			opts.Template, strings.Join(DefaultRegistry().Names(), ", "))
	}

	vars := ScaffoldVars{
		AgentName:      opts.ProjectName + "-agent",
		BelugaVersion:  opts.BelugaVersion,
		ModelName:      "gpt-4o-mini",
		ModulePath:     opts.ModulePath,
		ProjectName:    opts.ProjectName,
		ProviderImport: "github.com/lookatitude/beluga-ai/v2/llm/providers/openai",
		ProviderName:   "openai",
		ScaffoldedAt:   opts.ScaffoldedAt.UTC().Format(time.RFC3339),
	}

	return renderFS(ctx, fsys, opts.TargetDir, vars, opts.Force)
}

// DetectProjectRoot ancestor-walks from startDir upward looking for a
// Beluga project root. A qualifying directory has BOTH .beluga/project.yaml
// AND a go.mod with a require line for github.com/lookatitude/beluga-ai/v2.
//
// Returns *core.Error with core.ErrNotFound when no ancestor qualifies; the
// error message names .beluga/ and points at `beluga init` so the CLI caller
// can surface it verbatim. Used by the `beluga new <kind>` subcommands to
// place generated stubs inside a scaffolded project. ctx is checked at entry
// so callers can cancel long-running pipelines before the ancestor walk.
func DetectProjectRoot(ctx context.Context, startDir string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return detectProjectRoot(startDir)
}

// ReadFrameworkPin reads the generated project's go.mod and returns the
// version pin on github.com/lookatitude/beluga-ai/v2. Returns empty string
// when the require line is absent; returns the raw pin token otherwise
// (e.g. "v2.10.1", "v0.0.0-unknown", or an arbitrary pseudo-version). Used
// by `beluga new <kind>` commands to warn when the CLI's own version differs
// from the project's pin — callers compare to version.Get(). ctx is checked
// at entry so callers can cancel long-running pipelines; returns "" and the
// ctx error via the second return value when ctx is already done.
func ReadFrameworkPin(ctx context.Context, goModPath string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	var pin string
	_ = scanGoModFor(goModPath, func(line string) bool {
		trimmed := strings.TrimSpace(line)
		const prefix = "require github.com/lookatitude/beluga-ai/v2 "
		if strings.HasPrefix(trimmed, prefix) {
			pin = strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))
			return true
		}
		return false
	})
	return pin, nil
}

// Assert at compile time that projectNameRegex compiles a useful pattern
// and that fmt is imported (used by diagnostic error strings the caller
// may wrap further).
var _ = fmt.Sprintf
