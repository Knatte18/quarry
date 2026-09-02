// nextrun.go implements ladderbench's next-run subcommand: the CLI-level caller that tells the operator
// what to dispatch next, for either a run session (the config's next pending repetition) or the scoring
// session (the next run directory ingested but not yet scored).

package main

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/Knatte18/quarry/bench/loomyard-eval/ladder/internal/ladder"
)

func nextRunCommand() *cobra.Command {
	var configID string
	var scoring bool

	cmd := &cobra.Command{
		Use:   "next-run",
		Short: "report the next pending run repetition, or (under --scoring) the next run pending scoring",
		Long: `next-run reports what a session should dispatch next.

For a run session (--config-id), it prints that config's next pending repetition -- the first
repetition ladder.PendingRuns still has work for -- alongside its current attempt index (derived from
ladder.NextAttempt, never from a session-held counter, so the index has one derivation site on disk),
its full assembled prompt, and its agent-definition name. It reports nothing pending when every
repetition already carries an ingest marker.

Under --scoring, it instead prints the next run directory that has been ingested but not yet scored
(ladder.PendingScoring), across the whole matrix rather than one config.

next-run enforces the full pin set (ladder.RequirePins), since the assembled prompt it prints depends on
every pin the matrix depends on.`,
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
			repoRoot, err := resolveRepoRoot()
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			if scoring {
				return printNextScoring(out, l, resultsRoot)
			}
			if configID == "" {
				return fmt.Errorf("next-run: --config-id is required unless --scoring is given")
			}
			return printNextRun(out, l, repoRoot, resultsRoot, configID)
		},
	}

	cmd.Flags().StringVar(&configID, "config-id", "", "the LadderConfig id to find the next pending repetition for (required unless --scoring)")
	cmd.Flags().BoolVar(&scoring, "scoring", false, "report the next run directory ingested but not yet scored, across the whole matrix, instead of a run session's next repetition")

	return cmd
}

// pairsForConfig builds the ordered list of configID's own Reps repetitions as ladder.RunPair values, so
// PendingRuns can filter them the same way it filters any other pair slice.
func pairsForConfig(l *ladder.Ladder, configID string) ([]ladder.RunPair, error) {
	config, err := ladder.ConfigByID(l, configID)
	if err != nil {
		return nil, err
	}
	pairs := make([]ladder.RunPair, 0, l.Reps)
	for n := 1; n <= l.Reps; n++ {
		pairs = append(pairs, ladder.RunPair{Config: config, N: n})
	}
	return pairs, nil
}

// targetDirFor returns the target directory config's runs prompt names: the cold worktree template
// (substituted for rep) for the cold config, or the task's fixed pinned worktree for every warm config.
func targetDirFor(l *ladder.Ladder, config ladder.LadderConfig, rep int) string {
	if config.Cold {
		return coldWorktreeDir(l, rep)
	}
	return l.Tasks[config.Task].Worktree
}

// printNextRun prints configID's next pending repetition to out, or a "nothing pending" line when every
// repetition already carries an ingest marker.
func printNextRun(out io.Writer, l *ladder.Ladder, repoRoot, resultsRoot, configID string) error {
	pairs, err := pairsForConfig(l, configID)
	if err != nil {
		return err
	}
	pending := ladder.PendingRuns(resultsRoot, pairs)
	if len(pending) == 0 {
		fmt.Fprintf(out, "next-run: nothing pending for config %q\n", configID)
		return nil
	}
	pair := pending[0]

	attempt, err := ladder.NextAttempt(resultsRoot, pair.Config.ID, pair.N)
	if err != nil {
		return err
	}

	taskText, err := ladder.TaskTextFor(l, repoRoot, pair.Config.Task)
	if err != nil {
		return err
	}
	schemaJSON, err := ladder.SchemaFor(l, repoRoot, pair.Config.Task)
	if err != nil {
		return err
	}
	targetDir := targetDirFor(l, pair.Config, pair.N)
	annexText := ""
	if pair.Config.Annex != "" {
		scratchDir, err := ladder.SessionDir(l, repoRoot, pair.Config.ID, pair.N)
		if err != nil {
			return err
		}
		annex, err := ladder.ReadAnnex(scratchDir)
		if err != nil {
			return err
		}
		annexText = ladder.AnnexBlock(targetDir, annex)
	}
	prompt := ladder.PreambleFor(l, pair.Config, targetDir, taskText, schemaJSON, annexText)

	fmt.Fprintf(out, "rep: %d\n", pair.N)
	fmt.Fprintf(out, "attempt: %d\n", attempt)
	fmt.Fprintf(out, "agent_definition: %s\n", pair.Config.ID)
	fmt.Fprintf(out, "prompt:\n%s\n", prompt)
	return nil
}

// printNextScoring prints the next run directory pending scoring across the whole matrix to out, or a
// "nothing pending" line when none remain.
func printNextScoring(out io.Writer, l *ladder.Ladder, resultsRoot string) error {
	pending := ladder.PendingScoring(resultsRoot, ladder.PlanRuns(l))
	if len(pending) == 0 {
		fmt.Fprintln(out, "next-run: nothing pending for scoring")
		return nil
	}
	pair := pending[0]
	fmt.Fprintln(out, ladder.RunDirPath(resultsRoot, pair.Config.ID, pair.N))
	return nil
}
