// warm.go implements ladderbench's warm subcommand: the CLI-level caller that pre-warms a run session's
// daemon immediately before its dispatch, using the session's own server binary and target worktree with
// the scrubbed environment.

package main

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/Knatte18/quarry/bench/loomyard-eval/ladder/internal/ladder"
)

// warmFunc mirrors ladder.Warm's own signature. runWarm calls the actual pre-warm through this seam so a
// test exercises the post-condition-failure path with a fake rather than a real server binary or daemon.
type warmFunc func(serverPath, targetDir string, env []string, cacheDir string) error

func warmCommand() *cobra.Command {
	var configID string

	cmd := &cobra.Command{
		Use:   "warm",
		Short: "pre-warm this config's daemon for one dispatch attempt",
		Long: `warm calls the daemon warm-up against this config's session server binary and target
worktree, using the scrubbed environment every daemon-state resolution in this suite relies on.

warm is skipped entirely for the cold config: a cold repetition's entire premise is that its worktree
carries no resident daemon, and a warm-up call is precisely what would start one. Calling warm against
the cold config is refused as an error naming the config.

warm runs once per attempt, not once per repetition: the daemon self-expires after its own idle timeout,
so a retried attempt that skipped this call could dispatch against a cold daemon and silently
contaminate the warm arm's timings against the cold cell it is being compared to.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			l, err := resolveLadder(cmd)
			if err != nil {
				return err
			}
			if err := ladder.RequirePins(l); err != nil {
				return err
			}
			if configID == "" {
				return fmt.Errorf("warm: --config-id is required")
			}

			repoRoot, err := resolveRepoRoot()
			if err != nil {
				return err
			}
			return runWarm(cmd.OutOrStdout(), l, configID, repoRoot, realBuild, ladder.Warm)
		},
	}

	cmd.Flags().StringVar(&configID, "config-id", "", "the LadderConfig id to warm the daemon for (required)")

	return cmd
}

// runWarm resolves configID's target worktree, builds the server binary through build, and calls warmFn
// against it with the scrubbed environment. It refuses to run for the cold config before touching build
// or warmFn at all, and it returns warmFn's own error unchanged on a post-condition failure -- cobra
// turns a non-nil RunE error into a non-zero process exit.
func runWarm(out io.Writer, l *ladder.Ladder, configID, repoRoot string, build ladder.Builder, warmFn warmFunc) error {
	config, err := ladder.ConfigByID(l, configID)
	if err != nil {
		return err
	}
	if config.Cold {
		return fmt.Errorf("warm: config %q is the cold config; warm is never called for it -- see its --help", configID)
	}
	task, ok := l.Tasks[config.Task]
	if !ok {
		return fmt.Errorf("warm: config %q references unknown task %q", config.ID, config.Task)
	}

	serverPath, err := ladder.BuildServer(repoRoot, build)
	if err != nil {
		return err
	}

	cacheDir, err := ladder.UserCacheDir()
	if err != nil {
		return err
	}
	env := ladder.ScrubbedEnv()

	if err := warmFn(serverPath, task.Worktree, env, cacheDir); err != nil {
		return err
	}
	fmt.Fprintf(out, "warm: daemon warmed for config %q at %s\n", config.ID, task.Worktree)
	return nil
}
