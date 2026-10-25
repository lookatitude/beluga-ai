package codeact

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/lookatitude/beluga-ai/v2/core"
)

// Static import checking for WithAllowedCodeImports.
//
// The check is a best-effort, line-based scan of Python and JavaScript
// source. It rejects imports of modules outside the allowlist and, while an
// allowlist is set, the common dynamic-import escape hatches (__import__,
// importlib, eval/exec, require/import() with a non-literal argument). It is
// defence in depth, NOT a security boundary: determined code can still reach
// other modules (for example through already-imported objects), so untrusted
// code must still run in an isolating executor.

var (
	pyImportRe     = regexp.MustCompile(`^\s*import\s+(.+)$`)
	pyFromImportRe = regexp.MustCompile(`^\s*from\s+([.\w]+)\s+import\s`)
	pyDynamicRe    = regexp.MustCompile(`\b(__import__|importlib|exec|eval|compile)\s*\(|\bimportlib\b`)

	jsImportFromRe  = regexp.MustCompile(`\bfrom\s+['"]([^'"]+)['"]`)
	jsBareImportRe  = regexp.MustCompile(`^\s*import\s+['"]([^'"]+)['"]`)
	jsRequireLitRe  = regexp.MustCompile(`\brequire\s*\(\s*['"]([^'"]+)['"]\s*\)`)
	jsDynImportRe   = regexp.MustCompile(`\bimport\s*\(\s*['"]([^'"]+)['"]\s*\)`)
	jsRequireAnyRe  = regexp.MustCompile(`\brequire\s*\(`)
	jsImportCallRe  = regexp.MustCompile(`\bimport\s*\(`)
	jsEvalRe        = regexp.MustCompile(`\b(eval|Function)\s*\(`)
	jsNodePrefixRe  = regexp.MustCompile(`^node:`)
	pyAliasStripper = regexp.MustCompile(`\s+as\s+\w+`)
)

// checkImports enforces the allowlist for language. A nil or empty allowlist
// allows everything. Languages other than python and javascript are not
// checked, because their import syntax is not parsed.
func checkImports(language, code string, allowed []string) error {
	if len(allowed) == 0 {
		return nil
	}
	allow := make(map[string]bool, len(allowed))
	for _, a := range allowed {
		allow[strings.TrimSpace(a)] = true
	}
	var mods []string
	var dynamic bool
	switch language {
	case "python":
		mods, dynamic = pythonImports(code)
	case "javascript":
		mods, dynamic = javascriptImports(code)
	default:
		return nil
	}
	if dynamic {
		return core.NewError(opExecute, core.ErrGuardBlocked,
			"code uses dynamic imports or eval, which WithAllowedCodeImports does not permit", nil)
	}
	for _, m := range mods {
		if !allow[m] {
			return core.NewError(opExecute, core.ErrGuardBlocked,
				fmt.Sprintf("import %q is not in the allowed imports %v", m, allowed), nil)
		}
	}
	return nil
}

// pythonImports returns the top-level module names imported by code and
// whether dynamic-import constructs were found.
func pythonImports(code string) ([]string, bool) {
	var mods []string
	dynamic := false
	for _, line := range strings.Split(code, "\n") {
		if i := strings.IndexByte(line, '#'); i >= 0 {
			line = line[:i]
		}
		if pyDynamicRe.MatchString(line) {
			dynamic = true
		}
		for _, stmt := range strings.Split(line, ";") {
			if m := pyFromImportRe.FindStringSubmatch(stmt); m != nil {
				mods = append(mods, topLevel(m[1]))
				continue
			}
			if m := pyImportRe.FindStringSubmatch(stmt); m != nil {
				for _, part := range strings.Split(pyAliasStripper.ReplaceAllString(m[1], ""), ",") {
					if name := strings.TrimSpace(part); name != "" {
						mods = append(mods, topLevel(name))
					}
				}
			}
		}
	}
	return mods, dynamic
}

// javascriptImports returns the module specifiers imported by code and
// whether non-literal require/import() or eval was found.
func javascriptImports(code string) ([]string, bool) {
	var mods []string
	dynamic := false
	for _, line := range strings.Split(code, "\n") {
		if i := strings.Index(line, "//"); i >= 0 {
			line = line[:i]
		}
		for _, re := range []*regexp.Regexp{jsImportFromRe, jsBareImportRe, jsRequireLitRe, jsDynImportRe} {
			for _, m := range re.FindAllStringSubmatch(line, -1) {
				mods = append(mods, jsNodePrefixRe.ReplaceAllString(m[1], ""))
			}
		}
		if len(jsRequireAnyRe.FindAllString(line, -1)) > len(jsRequireLitRe.FindAllString(line, -1)) ||
			len(jsImportCallRe.FindAllString(line, -1)) > len(jsDynImportRe.FindAllString(line, -1)) ||
			jsEvalRe.MatchString(line) {
			dynamic = true
		}
	}
	return mods, dynamic
}

// topLevel returns the first dotted component of a module path.
func topLevel(mod string) string {
	mod = strings.TrimLeft(mod, ".")
	if i := strings.IndexByte(mod, '.'); i >= 0 {
		return mod[:i]
	}
	return mod
}
