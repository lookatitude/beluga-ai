package codeact

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/v2/core"
)

func TestCheckImports(t *testing.T) {
	allowed := []string{"math", "json", "lodash"}
	tests := []struct {
		name    string
		lang    string
		code    string
		allowed []string
		wantErr bool
	}{
		{"no allowlist allows all", "python", "import os", nil, false},
		{"py allowed", "python", "import math\nfrom json import dumps\nprint(math.pi)", allowed, false},
		{"py alias and list", "python", "import math as m, json", allowed, false},
		{"py submodule", "python", "from math.sub import x", allowed, false},
		{"py blocked", "python", "import os\nprint(os.environ)", allowed, true},
		{"py blocked from", "python", "from subprocess import run", allowed, true},
		{"py blocked after semicolon", "python", "import math; import os", allowed, true},
		{"py dunder import", "python", "m = __import__('os')", allowed, true},
		{"py importlib", "python", "import importlib", []string{"importlib"}, true},
		{"py eval", "python", "eval('1+1')", allowed, true},
		{"py comment ignored", "python", "import math # import os", allowed, false},
		{"js allowed", "javascript", "const _ = require('lodash')\nimport x from 'json'", allowed, false},
		{"js node prefix", "javascript", "const m = require('node:math')", allowed, false},
		{"js blocked", "javascript", "const fs = require('fs')", allowed, true},
		{"js bare import blocked", "javascript", "import 'fs'", allowed, true},
		{"js dynamic require", "javascript", "const n = 'fs'; require(n)", allowed, true},
		{"js dynamic import()", "javascript", "import(name)", allowed, true},
		{"js literal import()", "javascript", "await import('fs')", allowed, true},
		{"js eval", "javascript", "eval('1')", allowed, true},
		{"other language unchecked", "ruby", "require 'fileutils'", allowed, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkImports(tt.lang, tt.code, tt.allowed)
			if (err != nil) != tt.wantErr {
				t.Fatalf("checkImports err = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && core.CodeOf(err, "") != core.ErrGuardBlocked {
				t.Errorf("code = %q, want guard_blocked", core.CodeOf(err, ""))
			}
		})
	}
}

// TestCodeActAgent_EnforcesAllowedImports is a regression test for FWC-17:
// WithAllowedCodeImports was only a prompt instruction and never enforced.
func TestCodeActAgent_EnforcesAllowedImports(t *testing.T) {
	var ran bool
	exec := &recordingExecutor{fn: func() { ran = true }}
	a := NewCodeActAgent("c", WithExecutor(exec), WithAllowedCodeImports([]string{"math"}))
	_, err := a.ExecuteCode(context.Background(), CodeAction{Language: "python", Code: "import os"})
	if core.CodeOf(err, "") != core.ErrGuardBlocked {
		t.Fatalf("err = %v, want guard_blocked", err)
	}
	if ran {
		t.Error("executor ran code with a disallowed import")
	}
	if _, err := a.ExecuteCode(context.Background(), CodeAction{Language: "python", Code: "import math"}); err != nil {
		t.Fatalf("allowed import rejected: %v", err)
	}
	if !ran {
		t.Error("executor did not run allowed code")
	}
}

type recordingExecutor struct{ fn func() }

func (r *recordingExecutor) Execute(_ context.Context, a CodeAction) (CodeResult, error) {
	r.fn()
	return CodeResult{Output: a.Code}, nil
}

// TestProcessExecutor_MinimalEnvAndTempDir is a regression test for FWC-17:
// ProcessExecutor gave LLM-generated code the full parent environment and
// working directory.
func TestProcessExecutor_MinimalEnvAndTempDir(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available")
	}
	t.Setenv("BELUGA_FAKE_API_KEY", "sk-should-not-leak")
	cwd, _ := os.Getwd()
	e := NewProcessExecutor()
	res, err := e.Execute(context.Background(), CodeAction{
		Language: "python",
		Code:     "import os\nprint(os.environ.get('BELUGA_FAKE_API_KEY', 'unset'))\nprint(os.getcwd())",
		Timeout:  10 * time.Second,
	})
	if err != nil {
		t.Fatalf("Execute: %v (%s)", err, res.Error)
	}
	lines := strings.Split(strings.TrimSpace(res.Output), "\n")
	if len(lines) != 2 || lines[0] != "unset" {
		t.Fatalf("env leaked or bad output: %q", res.Output)
	}
	if lines[1] == cwd {
		t.Errorf("code ran in the parent working directory %q", cwd)
	}
}

func TestProcessExecutor_SupportedLanguagesSorted(t *testing.T) {
	e := NewProcessExecutor(WithInterpreter("ruby", "ruby"), WithInterpreter("bash", "bash"))
	got := strings.Join(e.supportedLanguages(), ",")
	if got != "bash,javascript,python,ruby" {
		t.Errorf("supportedLanguages = %s", got)
	}
}
