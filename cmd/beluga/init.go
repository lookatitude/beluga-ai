package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/lookatitude/beluga-ai/v2/cmd/beluga/internal/version"
	"github.com/lookatitude/beluga-ai/v2/cmd/beluga/scaffold"
)

// newInitCmd returns the cobra subcommand for `beluga init <project-name>`.
// S2 replaces the S1 stub wholesale — the positional argument + three flags
// (--template, --module, --force) drive the scaffolder directly. See
// docs/consultations/2026-04-20-loo-149-architect-plan.md §T7 for the
// rationale for a full replacement rather than an incremental migration.
func newInitCmd() *cobra.Command {
	var (
		template   string
		modulePath string
		force      bool
	)
	cmd := &cobra.Command{
		Use:   "init <project-name>",
		Short: "Initialize a new Beluga AI project",
		Long: "Initialize a new Beluga AI project from a named template.\n\n" +
			"The project name must be lowercase letters, digits, and hyphens (2-64 chars),\n" +
			"starting with a letter and ending with a letter or digit.",
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			projectName := args[0]

			if err := scaffold.ValidateProjectName(projectName); err != nil {
				return err
			}
			if modulePath != "" {
				if err := scaffold.ValidateModulePath(modulePath); err != nil {
					return err
				}
			} else {
				modulePath = "example.com/" + projectName
			}

			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("resolve working directory: %w", err)
			}
			targetDir := filepath.Join(cwd, projectName)

			opts := scaffold.Options{
				ProjectName:   projectName,
				Template:      template,
				ModulePath:    modulePath,
				TargetDir:     targetDir,
				Force:         force,
				BelugaVersion: version.Get(),
				ScaffoldedAt:  time.Now().UTC(),
			}

			if err := scaffold.Scaffold(cmd.Context(), opts); err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			_, _ = fmt.Fprintf(out, "Initialized Beluga AI project %q in %s\n", projectName, targetDir)
			_, _ = fmt.Fprintf(out, "Next: cd %s && export OPENAI_API_KEY=... && go run .\n", projectName)
			return nil
		},
	}
	names := scaffold.DefaultRegistry().Names()
	cmd.Flags().StringVar(&template, "template", "basic",
		"template name (available: "+strings.Join(names, ", ")+")")
	cmd.Flags().StringVar(&modulePath, "module", "",
		"Go module path (default: example.com/<project-name>)")
	cmd.Flags().BoolVar(&force, "force", false,
		"overwrite existing files in a non-empty target directory")
	return cmd
}
