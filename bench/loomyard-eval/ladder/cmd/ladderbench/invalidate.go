// invalidate.go implements ladderbench's invalidate subcommand: the CLI-level caller that renames a
// failed run directory aside and reports the next attempt index, erroring once the attempt ceiling is
// already exhausted.

package main

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/Knatte18/quarry/bench/loomyard-eval/ladder/internal/ladder"
)

func invalidateCommand() *cobra.Command {
	var configID string
	var rep int

	cmd := &cobra.Command{
		Use:   "invalidate",
		Short: "rename a failed run directory aside and report the next attempt index",
		Long: `invalidate renames --config-id's --rep-th run directory aside to its lowest unused
<n>.invalid-<k> sibling (ladder.Invalidate) and prints the next attempt index. Once the attempt ceiling
(ladder.MaxAttempts) is already exhausted, it errors instead -- the matrix halt this command is the sole
trigger for, never a silently retried attempt past the cap.

invalidate is deliberately separate from ingest: ingest must be able to report a failed or truncated
outcome without destroying the evidence of it, so nothing in ingest itself ever renames or removes a run
directory. Call invalidate only once ingest (or, on the cold path, a live-daemon session-preparation
abort) has already recorded why an attempt failed.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			l, err := resolveLadder(cmd)
			if err != nil {
				return err
			}
			resultsRoot, err := resolveResultsRoot(cmd)
			if err != nil {
				return err
			}
			if configID == "" {
				return fmt.Errorf("invalidate: --config-id is required")
			}
			if !cmd.Flags().Changed("rep") {
				return fmt.Errorf("invalidate: --rep is required")
			}
			return runInvalidate(cmd.OutOrStdout(), l, resultsRoot, configID, rep)
		},
	}

	cmd.Flags().StringVar(&configID, "config-id", "", "the LadderConfig id whose run directory to invalidate (required)")
	cmd.Flags().IntVar(&rep, "rep", 0, "1-based repetition index (required)")

	return cmd
}

// runInvalidate resolves configID's rep-th run directory and invalidates it, printing the next attempt
// index on success.
func runInvalidate(out io.Writer, l *ladder.Ladder, resultsRoot, configID string, rep int) error {
	if _, err := ladder.ConfigByID(l, configID); err != nil {
		return err
	}

	runDir := ladder.RunDirPath(resultsRoot, configID, rep)
	next, err := ladder.Invalidate(runDir)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "invalidate: %s invalidated; next attempt is %d\n", runDir, next)
	return nil
}
