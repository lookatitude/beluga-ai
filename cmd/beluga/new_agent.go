package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newNewAgentCmd returns the `beluga new agent <Name>` subcommand. The
// generated `<snake>.go` defines a stub agent whose `Invoke` method returns
// core.Errorf(core.ErrNotFound, "<Name>.Invoke not implemented"); the
// companion `<snake>_test.go` uses t.Skip so `go test ./...` is green out
// of the box until the user implements the agent. See
// docs/consultations/2026-04-20-loo-149-architect-plan.md §T9.
func newNewAgentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "agent <Name>",
		Short:         "Create a new agent stub (PascalCase name)",
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runNewComponent(cmd, "agent", args[0], renderAgentStub)
		},
	}
	return cmd
}

// renderAgentStub builds the two file bodies for `beluga new agent <Name>`.
// The source uses core.Errorf(core.ErrNotFound) per Decision #18 — panic
// would violate the go-packages rule forbidding panic for recoverable
// errors. The test case uses t.Skip so `go test ./...` succeeds until
// the user implements the agent.
//
// Both generated files declare `package main` to match the flat Layer-7
// layout of the `basic` init template (main.go also declares package main).
// The packageName argument is retained for future templates whose stubs live
// in a subpackage, but is not used for the `package` line here — mixing
// package declarations in the same directory is a compile-time error.
func renderAgentStub(packageName, pascalName, snakeName string) (string, string) {
	_ = packageName // reserved for future non-flat templates
	source := fmt.Sprintf(`// Generated stub for %s.
// Replace the TODO bodies below with your implementation, then
// remove the t.Skip guard in %s_test.go to enable the tests.
package main

import (
	"context"

	"github.com/lookatitude/beluga-ai/v2/core"
)

// %s is a user-defined agent stub. Extend it with the fields your agent
// needs (LLM handle, tools, memory adapter, etc.) and implement Invoke.
type %s struct {
	// TODO: add fields — llm.ChatModel, []tool.Tool, memory.Store, ...
}

// New%s constructs a %s with zero-value fields. Replace this with an
// options-based constructor when you add configuration.
func New%s() *%s {
	return &%s{}
}

// Invoke runs the agent for a single request. TODO: implement.
//
// Return core.Errorf(core.ErrNotFound, ...) while the stub is in place so
// accidental wiring surfaces clearly in tests instead of panicking.
func (a *%s) Invoke(ctx context.Context, input string) (string, error) {
	return "", core.Errorf(core.ErrNotFound, "%s.Invoke not implemented")
}
`, pascalName, snakeName,
		pascalName, pascalName, pascalName, pascalName,
		pascalName, pascalName, pascalName, pascalName, pascalName)

	test := fmt.Sprintf(`package main

import (
	"testing"
)

// Test_%s_Invoke is a placeholder test. It exercises the stub's symbol so
// the compile-time wiring is covered without actually running the
// not-yet-implemented Invoke body. Remove the t.Skip call below once
// %s.Invoke is implemented.
func Test_%s_Invoke(t *testing.T) {
	t.Skip("remove when %s is implemented")

	// Compile-time reference to the generated stub; keeps go vet happy and
	// flips to a real call site when the user removes t.Skip.
	_ = &%s{}
}
`, pascalName,
		pascalName,
		pascalName,
		pascalName,
		pascalName)

	return source, test
}
