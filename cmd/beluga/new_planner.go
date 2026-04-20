package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// newNewPlannerCmd returns the `beluga new planner <Name>` subcommand.
// The generated planner stub implements agent.Planner (2 methods: Plan,
// Replan), returns core.Errorf(core.ErrNotFound) instead of panicking per
// Decision #18, and includes a commented init() block showing how to wire
// the planner into agent.RegisterPlanner once implemented.
func newNewPlannerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "planner <Name>",
		Short:         "Create a new planner stub (PascalCase name)",
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runNewComponent(cmd, "planner", args[0], renderPlannerStub)
		},
	}
	return cmd
}

// renderPlannerStub builds the two files for `beluga new planner <Name>`.
// The registry name (used with agent.RegisterPlanner) is the snake-case
// name with underscores replaced by hyphens — matches the convention
// followed by framework built-ins ("react", "reflexion", ...).
//
// Both generated files declare `package main` to match the flat Layer-7
// layout of the `basic` init template. The packageName argument is retained
// for future non-flat templates but is not used for the `package` line —
// mixing package declarations in the same directory is a compile-time
// error.
func renderPlannerStub(packageName, pascalName, snakeName string) (string, string) {
	_ = packageName // reserved for future non-flat templates
	registryName := strings.ReplaceAll(snakeName, "_", "-")

	source := fmt.Sprintf(`// Generated stub for %s planner.
// Replace the TODO bodies below with your implementation, then
// remove the t.Skip guards in %s_test.go to enable the tests.
package main

import (
	"context"

	"github.com/lookatitude/beluga-ai/v2/agent"
	"github.com/lookatitude/beluga-ai/v2/core"
)

// Compile-time check that %s implements agent.Planner.
var _ agent.Planner = (*%s)(nil)

// %s is a user-defined planner stub. See docs/architecture/06-reasoning-
// strategies.md for design notes on existing strategies (react, reflexion,
// etc.) before rolling your own.
type %s struct {
	// TODO: add fields the planner needs (LLM handle, scratchpad, ...).
}

// New%s constructs a %s with zero-value fields. Replace this with an
// options-based constructor when you add configuration.
func New%s() *%s {
	return &%s{}
}

// Plan generates a list of actions given the current planner state.
// TODO: implement.
func (p *%s) Plan(ctx context.Context, state agent.PlannerState) ([]agent.Action, error) {
	return nil, core.Errorf(core.ErrNotFound, "%s.Plan not implemented")
}

// Replan generates updated actions based on new observations.
// Called after actions have been executed and observations collected.
// TODO: implement.
func (p *%s) Replan(ctx context.Context, state agent.PlannerState) ([]agent.Action, error) {
	return nil, core.Errorf(core.ErrNotFound, "%s.Replan not implemented")
}

// Uncomment once Plan / Replan are implemented to make the planner
// discoverable via agent.WithPlannerName(%q):
//
//	func init() {
//		agent.RegisterPlanner(%q, func(cfg agent.PlannerConfig) (agent.Planner, error) {
//			return New%s(), nil
//		})
//	}
`, pascalName, snakeName,
		pascalName, pascalName,
		pascalName,
		pascalName,
		pascalName, pascalName, pascalName, pascalName, pascalName,
		pascalName, pascalName,
		pascalName, pascalName,
		registryName, registryName, pascalName)

	test := fmt.Sprintf(`package main

import (
	"testing"
)

// Test_%s_Plan exercises the planner's Plan contract. The t.Skip guard
// keeps the suite green until the stub is implemented.
func Test_%s_Plan(t *testing.T) {
	t.Skip("remove when %s is implemented")

	// Compile-time reference so go vet is happy and the symbol flips to a
	// real call site once the user removes t.Skip.
	_ = New%s()
}

// Test_%s_Replan exercises the planner's Replan contract.
func Test_%s_Replan(t *testing.T) {
	t.Skip("remove when %s is implemented")

	_ = New%s()
}
`, pascalName,
		pascalName,
		pascalName,
		pascalName,
		pascalName,
		pascalName,
		pascalName,
		pascalName)

	return source, test
}
