// coldcell.go implements ladderbench's cold-cell subcommand: the CLI-level caller that either tears down
// one cold repetition's disposable worktree after its attempt, or, once the cold config's own repetitions
// have run, finalises and writes the cold cell's disposition record.

package main

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/Knatte18/quarry/bench/loomyard-eval/ladder/internal/ladder"
)

func coldCellCommand() *cobra.Command {
	var teardown bool
	var rep int

	cmd := &cobra.Command{
		Use:   "cold-cell",
		Short: "tear down one cold repetition's worktree, or finalise the cold cell's disposition",
		Long: `cold-cell has two forms.

With --teardown --rep, it removes that repetition's disposable cold worktree unconditionally, whatever
the run's outcome, and waits for that worktree's daemon to exit -- a cold repetition's worktree is never
restored in place the way a warm config's is (see restore-worktree --help); it is discarded outright once
its attempt ends.

With no flags, it finalises the cold cell: it reads whatever a prior run or cold-session-preparation
abort already left on disk for the cold config's own repetitions (ladder.ColdCellDisposition) and writes
<results-root>/cold_cell.json.

cold-cell enforces the full pin set.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			l, err := resolveLadder(cmd)
			if err != nil {
				return err
			}
			if err := ladder.RequirePins(l); err != nil {
				return err
			}

			if teardown {
				if !cmd.Flags().Changed("rep") {
					return fmt.Errorf("cold-cell: --rep is required with --teardown")
				}
				return runColdCellTeardown(cmd.OutOrStdout(), l, rep, ladder.RunGit)
			}

			resultsRoot, err := resolveResultsRoot(cmd)
			if err != nil {
				return err
			}
			return runColdCellFinalize(cmd.OutOrStdout(), l, resultsRoot)
		},
	}

	cmd.Flags().BoolVar(&teardown, "teardown", false, "tear down --rep's disposable cold worktree instead of finalising the cold cell")
	cmd.Flags().IntVar(&rep, "rep", 0, "1-based repetition index whose disposable worktree to tear down (required with --teardown)")

	return cmd
}

// runColdCellTeardown removes rep's disposable cold worktree unconditionally and waits for its daemon to
// exit before returning.
func runColdCellTeardown(out io.Writer, l *ladder.Ladder, rep int, git ladder.GitRunner) error {
	targetDir := coldWorktreeDir(l, rep)
	if err := ladder.RemoveWorktree(l.SourceRepo, targetDir, git); err != nil {
		return err
	}

	cacheDir, err := ladder.UserCacheDir()
	if err != nil {
		return err
	}
	env := ladder.ScrubbedEnv()
	if err := ladder.WaitForDaemonExit(targetDir, cacheDir, env, ladder.DaemonExitTimeout, coldDaemonLang); err != nil {
		return err
	}

	fmt.Fprintf(out, "cold-cell: tore down rep %d's worktree at %s\n", rep, targetDir)
	return nil
}

// runColdCellFinalize builds and writes the cold cell's disposition record.
func runColdCellFinalize(out io.Writer, l *ladder.Ladder, resultsRoot string) error {
	record, err := ladder.ColdCellDisposition(l, resultsRoot)
	if err != nil {
		return err
	}
	if err := ladder.WriteColdCellRecord(resultsRoot, record); err != nil {
		return err
	}
	fmt.Fprintf(out, "cold-cell: disposition %s (%s)\n", record.Disposition, record.Reason)
	return nil
}
