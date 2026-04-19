package main

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/lookatitude/beluga-ai/v2/llm"
	"github.com/lookatitude/beluga-ai/v2/memory"
	"github.com/lookatitude/beluga-ai/v2/rag/embedding"
	"github.com/lookatitude/beluga-ai/v2/rag/vectorstore"
)

// providerCategory is the JSON element emitted by `beluga providers --output json`.
// The stable wire shape is:
//
//	[{"category": "<name>", "providers": ["<p1>", "<p2>", ...]}, ...]
//
// Top-level is a JSON array so jq consumers can filter by category, e.g.
//
//	beluga providers --output json | jq '.[] | select(.category=="llm") | .providers'
//
// Each category's Providers slice is alphabetically sorted (inherited from
// each registry's List() — see llm/registry.go, rag/embedding/registry.go,
// rag/vectorstore/registry.go, memory/memory.go).
type providerCategory struct {
	Category  string   `json:"category"`
	Providers []string `json:"providers"`
}

// newProvidersCmd returns the cobra subcommand for `beluga providers`.
//
// Formats:
//   - default / "text" → `<category>\t<provider>` rows via tabwriter.
//   - "json"           → indented JSON array of providerCategory.
//   - anything else    → error (exit 1), message "unsupported output format".
//
// The --output value is sourced from the root persistent flag (registered
// in newRootCmd() as --output/-o). Category order is fixed and deliberate:
// llm, embedding, vectorstore, memory — matches the brief's schema sample
// and gives CI consumers a stable index to look up.
func newProvidersCmd() *cobra.Command {
	return &cobra.Command{
		Use:           "providers",
		Short:         "List providers compiled into this binary",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cats := []providerCategory{
				{Category: "llm", Providers: llm.List()},
				{Category: "embedding", Providers: embedding.List()},
				{Category: "vectorstore", Providers: vectorstore.List()},
				{Category: "memory", Providers: memory.List()},
			}

			format, _ := cmd.Flags().GetString("output")
			w := cmd.OutOrStdout()

			switch format {
			case "json":
				enc := json.NewEncoder(w)
				enc.SetIndent("", "  ")
				return enc.Encode(cats)
			case "", "text":
				tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
				for _, c := range cats {
					for _, p := range c.Providers {
						_, _ = fmt.Fprintf(tw, "%s\t%s\n", c.Category, p)
					}
				}
				return tw.Flush()
			default:
				return fmt.Errorf("unsupported output format: %q (supported: text, json)", format)
			}
		},
	}
}
