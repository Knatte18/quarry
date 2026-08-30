// restoreworktree.go implements ladderbench's restore-worktree subcommand: the CLI-level caller that
// resets, cleans, and re-neutralises a warm config's task worktree after an attempt.

package main

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/Knatte18/quarry/bench/loomyard-eval/ladder/internal/ladder"
)

func restoreWorktreeCommand() *cobra.Command {
	var configID string

	cmd := &cobra.Command{
		Use:   "restore-worktree",
		Short: "reset, clean, and re-neutralise this config's task worktree after an attempt",
		Long: `restore-worktree runs git reset --hard, then git clean -fdx, then re-applies the
neutralisation that removes CLAUDE.md/CONSTRAINTS.md/.claude from the worktree -- clean -fdx restores
exactly the ambient-context files neutralisation removed, so ladder.RestoreWorktree always performs all
three steps as one unconditional unit.

It runs after every attempt, whatever that attempt's outcome -- complete, failed, or truncated -- so a
dirtied worktree never survives into the next attempt's own worktree_dirtied observation.

restore-worktree is deliberately never used by cold sessions: it resets and re-neutralises a persistent
worktree, which is the opposite of a cold repetition's disposable, per-attempt one -- a cold worktree is
removed outright once its attempt ends, never restored in place. Calling restore-worktree against the
cold config is refused as an error naming the config.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			l, err := resolveLadder(cmd)
			if err != nil {
				return err
			}
			if configID == "" {
				return fmt.Errorf("restore-worktree: --config-id is required")
			}
			return runRestoreWorktree(cmd.OutOrStdout(), l, configID, ladder.RunGit)
		},
	}

	cmd.Flags().StringVar(&configID, "config-id", "", "the LadderConfig id whose task worktree to restore (required)")

	return cmd
}

// runRestoreWorktree resolves configID's task worktree and restores it through git, refusing to run for
// the cold config before touching git at all.
func runRestoreWorktree(out io.Writer, l *ladder.Ladder, configID string, git ladder.GitRunner) error {
	config, err := ladder.ConfigByID(l, configID)
	if err != nil {
		return err
	}
	if config.Cold {
		return fmt.Errorf("restore-worktree: config %q is the cold config; restore-worktree is never called for it -- see its --help", configID)
	}
	task, ok := l.Tasks[config.Task]
	if !ok {
		return fmt.Errorf("restore-worktree: config %q references unknown task %q", config.ID, config.Task)
	}

	if err := ladder.RestoreWorktree(task.Worktree, git); err != nil {
		return err
	}
	fmt.Fprintf(out, "restore-worktree: restored %s\n", task.Worktree)
	return nil
}
