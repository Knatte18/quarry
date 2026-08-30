// summarize.go implements ladderbench's summarize subcommand: the CLI-level caller that builds and
// writes summary.json and reports non-zero when the matrix is not yet fully complete.

package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Knatte18/quarry/bench/loomyard-eval/ladder/internal/ladder"
)

func summarizeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "summarize",
		Short: "build and write summary.json, exiting non-zero when any cell is incomplete",
		Long: `summarize builds and writes <results-root>/summary.json (ladder.WriteSummary, built on
ladder.BuildSummary) and exits non-zero (ladder.SummaryExitCode) when any cell is incomplete, naming the
incomplete cells on standard error -- a summary of a partial matrix is still written, but it must never
be mistaken for a finished one.

summarize enforces the full pin set.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			l, err := resolveLadder(cmd)
			if err != nil {
				return err
			}
			if err := ladder.RequirePins(l); err != nil {
				return err
			}
			resultsRoot, err := resolveResultsRoot(cmd)
			if err != nil {
				return err
			}
			return runSummarize(cmd.OutOrStdout(), l, resultsRoot)
		},
	}

	return cmd
}

// runSummarize builds and writes summary.json at resultsRoot, returning a non-nil error naming the
// incomplete cells when the matrix is not yet fully complete -- cobra's default error handling prints a
// returned RunE error to standard error and exits the process non-zero, which is what carries this
// naming onto standard error and the non-zero exit onto the process alike.
func runSummarize(out io.Writer, l *ladder.Ladder, resultsRoot string) error {
	summary, err := ladder.WriteSummary(l, resultsRoot)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "summarize: wrote summary.json to %s\n", resultsRoot)

	if ladder.SummaryExitCode(summary) != 0 {
		return fmt.Errorf("summarize: incomplete cells: %s", strings.Join(summary.Incomplete, ", "))
	}
	return nil
}
