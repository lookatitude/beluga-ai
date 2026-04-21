package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// newNewToolCmd returns the `beluga new tool <Name>` subcommand. The
// generated `<snake>.go` defines a tool using tool.NewFuncTool (per
// framework/tool/functool.go) with a typed Input struct; the stub body
// returns core.Errorf(core.ErrNotFound, "<Name> not implemented") and the
// companion test is gated by t.Skip so `go test ./...` succeeds until
// implementation.
func newNewToolCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "tool <Name>",
		Short:         "Create a new tool stub (PascalCase name)",
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runNewComponent(cmd, "tool", args[0], renderToolStub)
		},
	}
	return cmd
}

// renderToolStub builds the two files for `beluga new tool <Name>`.
// The tool identifier (used at the LLM boundary) is the snake-case name
// with underscores replaced by hyphens — idiomatic for tool names in
// OpenAI / Anthropic / Gemini function-calling contracts.
//
// Both generated files declare `package main` to match the flat Layer-7
// layout of the `basic` init template. The packageName argument is retained
// for future non-flat templates but is not used for the `package` line —
// mixing package declarations in the same directory is a compile-time
// error.
func renderToolStub(packageName, pascalName, snakeName string) (string, string) {
	_ = packageName // reserved for future non-flat templates
	toolID := strings.ReplaceAll(snakeName, "_", "-")

	source := fmt.Sprintf(`// Generated stub for %s tool.
// Replace the TODO bodies below with your implementation, then
// remove the t.Skip guard in %s_test.go to enable the tests.
package main

import (
	"context"

	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/tool"
)

// %sInput is the typed input for the %s tool. Struct tags drive the
// generated JSON Schema so the LLM sees a well-formed tool contract.
//
// TODO: add fields here. Use tags like:
//
//	Name string `+"`"+`json:"name" description:"..." required:"true"`+"`"+`
type %sInput struct {
	// TODO: add request fields.
}

// New%s constructs the %s tool. Register it with an agent via
// agent.WithTools([]tool.Tool{New%s()}).
func New%s() tool.Tool {
	return tool.NewFuncTool(
		%q,
		"TODO: one-sentence description of what %s does.",
		func(ctx context.Context, in %sInput) (*tool.Result, error) {
			return nil, core.Errorf(core.ErrNotFound, "%s not implemented")
		},
	)
}
`, pascalName, snakeName,
		pascalName, pascalName,
		pascalName,
		pascalName, pascalName, pascalName,
		pascalName, toolID, pascalName, pascalName, pascalName)

	test := fmt.Sprintf(`package main

import (
	"testing"
)

// Test_%s exercises the tool's construction contract. The t.Skip call
// keeps the suite green until the stub is implemented; remove it when the
// tool body returns real results.
func Test_%s(t *testing.T) {
	t.Skip("remove when %s is implemented")

	// Compile-time reference to the constructor so go vet is satisfied and
	// the symbol flips to a real call site once the user removes t.Skip.
	_ = New%s()
}
`, pascalName,
		pascalName,
		pascalName,
		pascalName)

	return source, test
}
