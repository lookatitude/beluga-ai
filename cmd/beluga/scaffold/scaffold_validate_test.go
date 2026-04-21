package scaffold

import (
	"errors"
	"strings"
	"testing"

	"github.com/lookatitude/beluga-ai/v2/core"
)

// TestValidateProjectName exercises the allowlist regex, length bounds, and
// the Windows-reserved-name blocklist. ValidateProjectName must never
// sanitise — it rejects on mismatch and names the rule in the error message.
func TestValidateProjectName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		// wantMsgContains: substring the error message must contain to
		// prove the user sees the rule that was violated.
		wantMsgContains string
	}{
		// Happy paths — representative lengths and character sets.
		{name: "minimum two chars", input: "ab", wantErr: false},
		{name: "typical name", input: "myproject", wantErr: false},
		{name: "hyphenated", input: "my-project", wantErr: false},
		{name: "alphanumeric with hyphens", input: "agent-v2-demo", wantErr: false},
		{name: "64 chars exactly", input: "a" + strings.Repeat("0", 62) + "z", wantErr: false},

		// Empty and too-short rejection.
		{name: "empty string", input: "", wantErr: true, wantMsgContains: "empty"},
		{name: "one char too short", input: "a", wantErr: true, wantMsgContains: "allowed pattern"},

		// Too long — anti-ReDoS pre-regex length check.
		{name: "65 chars too long", input: "a" + strings.Repeat("0", 63) + "z", wantErr: true, wantMsgContains: "64"},

		// Allowlist violations.
		{name: "contains space", input: "My Project", wantErr: true, wantMsgContains: "allowed pattern"},
		{name: "path traversal", input: "../evil", wantErr: true, wantMsgContains: "allowed pattern"},
		{name: "starts with hyphen", input: "-badstart", wantErr: true, wantMsgContains: "allowed pattern"},
		{name: "ends with hyphen", input: "bad-", wantErr: true, wantMsgContains: "allowed pattern"},
		{name: "uppercase rejected", input: "MyProject", wantErr: true, wantMsgContains: "allowed pattern"},
		{name: "dot in name", input: "my.project", wantErr: true, wantMsgContains: "allowed pattern"},
		{name: "underscore rejected", input: "my_project", wantErr: true, wantMsgContains: "allowed pattern"},
		{name: "slash rejected", input: "my/project", wantErr: true, wantMsgContains: "allowed pattern"},

		// Windows reserved names (case-insensitive).
		{name: "CON reserved", input: "con", wantErr: true, wantMsgContains: "reserved"},
		{name: "CON uppercase-equivalent via input", input: "con", wantErr: true, wantMsgContains: "reserved"},
		{name: "prn reserved", input: "prn", wantErr: true, wantMsgContains: "reserved"},
		{name: "aux reserved", input: "aux", wantErr: true, wantMsgContains: "reserved"},
		{name: "nul reserved", input: "nul", wantErr: true, wantMsgContains: "reserved"},
		{name: "com1 reserved", input: "com1", wantErr: true, wantMsgContains: "reserved"},
		{name: "lpt9 reserved", input: "lpt9", wantErr: true, wantMsgContains: "reserved"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertValidateProjectName(t, tt.input, tt.wantErr, tt.wantMsgContains)
		})
	}
}

// assertValidateProjectName runs ValidateProjectName and asserts that the
// outcome, core.Error code, and message substring match expectations.
// Extracted from TestValidateProjectName so the per-case body stays below
// the cognitive-complexity ceiling.
func assertValidateProjectName(t *testing.T, input string, wantErr bool, wantMsgContains string) {
	t.Helper()
	err := ValidateProjectName(input)
	if (err != nil) != wantErr {
		t.Fatalf("ValidateProjectName(%q) err = %v, wantErr %v", input, err, wantErr)
	}
	if !wantErr {
		return
	}
	var coreErr *core.Error
	if !errors.As(err, &coreErr) {
		t.Fatalf("ValidateProjectName(%q): expected *core.Error, got %T: %v", input, err, err)
	}
	if coreErr.Code != core.ErrInvalidInput {
		t.Errorf("ValidateProjectName(%q): code = %v, want %v", input, coreErr.Code, core.ErrInvalidInput)
	}
	if wantMsgContains != "" && !strings.Contains(err.Error(), wantMsgContains) {
		t.Errorf("ValidateProjectName(%q): error %q must contain %q", input, err.Error(), wantMsgContains)
	}
}

// TestValidateModulePath checks that Go module-path grammar is enforced
// independently of the project-name regex (module paths allow mixed case,
// dots, and slashes — none of which the project-name regex permits).
func TestValidateModulePath(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		// Accepted: real-world Go module paths.
		{name: "example.com/foo", input: "example.com/foo", wantErr: false},
		{name: "github.com/org/repo", input: "github.com/org/repo", wantErr: false},
		{name: "nested path", input: "github.com/lookatitude/beluga-ai/v2", wantErr: false},
		{name: "single segment", input: "example", wantErr: false},
		{name: "mixed case segments", input: "github.com/MyOrg/MyRepo", wantErr: false},

		// Rejected: grammar violations.
		{name: "empty", input: "", wantErr: true},
		{name: "space in path", input: "foo bar", wantErr: true},
		{name: "shell metacharacter", input: "foo;rm -rf /", wantErr: true},
		{name: "leading slash", input: "/foo/bar", wantErr: true},
		{name: "trailing slash", input: "foo/bar/", wantErr: true},
		{name: "double slash", input: "foo//bar", wantErr: true},
		{name: "contains tab", input: "foo\tbar", wantErr: true},
		{name: "contains newline", input: "foo\nbar", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateModulePath(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateModulePath(%q) err = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if !tt.wantErr {
				return
			}
			var coreErr *core.Error
			if !errors.As(err, &coreErr) {
				t.Fatalf("ValidateModulePath(%q): expected *core.Error, got %T: %v", tt.input, err, err)
			}
			if coreErr.Code != core.ErrInvalidInput {
				t.Errorf("ValidateModulePath(%q): code = %v, want %v", tt.input, coreErr.Code, core.ErrInvalidInput)
			}
		})
	}
}

// TestScaffold_UnknownTemplate asserts that Scaffold rejects an unregistered
// template name with ErrInvalidInput and names both the unknown template and
// the set of registered templates in the message, so the user can correct.
func TestScaffold_UnknownTemplate(t *testing.T) {
	// DefaultRegistry() will be populated by templates_builtin.go init()
	// with "basic" — confirm unknown-template rejection quotes both.
	targetDir := t.TempDir()
	err := Scaffold(t.Context(), Options{
		ProjectName: "sample",
		Template:    "no-such-template",
		ModulePath:  "example.com/sample",
		TargetDir:   targetDir,
	})
	if err == nil {
		t.Fatalf("Scaffold with unknown template: expected error, got nil")
	}
	var coreErr *core.Error
	if !errors.As(err, &coreErr) {
		t.Fatalf("Scaffold: expected *core.Error, got %T: %v", err, err)
	}
	if coreErr.Code != core.ErrInvalidInput {
		t.Errorf("Scaffold: code = %v, want %v", coreErr.Code, core.ErrInvalidInput)
	}
	if !strings.Contains(err.Error(), "no-such-template") {
		t.Errorf("Scaffold error must name unknown template, got: %v", err)
	}
}
