package main

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/lookatitude/beluga-ai/cmd/beluga/internal/version"
	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/memory"
	"github.com/lookatitude/beluga-ai/rag/embedding"
	"github.com/lookatitude/beluga-ai/rag/vectorstore"
)

// newVersionCmd returns the cobra subcommand for `beluga version`. The
// output contains three lines:
//
//	beluga <framework-version>
//	<go-runtime-version>            (e.g. "go1.25.9")
//	providers: llm=N embedding=N vectorstore=N memory=N
//
// AC2 requires the literal substrings "beluga ", "go1.", and "providers:" to
// appear in the stdout; printing runtime.Version() unmodified preserves the
// "go1." prefix unambiguously (see architect decision on the second line).
func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:           "version",
		Short:         "Print framework version, Go runtime, and provider counts",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			w := cmd.OutOrStdout()
			_, _ = fmt.Fprintf(w, "beluga %s\n", version.Get())
			_, _ = fmt.Fprintf(w, "%s\n", runtime.Version())
			_, _ = fmt.Fprintf(w, "providers: llm=%d embedding=%d vectorstore=%d memory=%d\n",
				len(llm.List()), len(embedding.List()),
				len(vectorstore.List()), len(memory.List()))
			return nil
		},
	}
}
