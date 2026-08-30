// root.go builds ladderbench's own cobra command tree, following this repo's existing
// internal/cli/cli.go pattern: one Command() constructor for the whole tree, one <verb>Command()
// constructor per subcommand, unexported flag variables closed over each subcommand's own RunE.

// Package main is ladderbench: the CLI surface a live session runs before and around a dispatch. It
// exists because scripts/run_ladder.py's own entry point dispatched the claude subprocess itself, which
// this port's Shared Decision retires -- the harness never dispatches a model call; a live session does,
// one supervised, killable run at a time. Every subcommand below is therefore drawn at a session
// boundary: it either prepares what a session needs before it launches, or records what a finished
// session (or one of its attempts) produced, never anything in between -- the boundary between "before a
// dispatch" and "after one" is the one place this suite can still intervene without becoming the
// unwatchable subprocess the task exists to remove.
//
// The eleven-subcommand surface, in the order a matrix repetition actually visits them:
//
//   - prepare-session materialises a session's scratch directory -- a main-matrix run, the single shared
//     scoring session, or one of the two permission probes -- and prints the launch command the operator
//     runs. Its --release mode instead clears the cross-session lock and does nothing else.
//   - warm pre-warms the target worktree's daemon immediately before a run session's dispatch; never
//     called for the cold config.
//   - next-run reports the next pending run repetition (or, under --scoring, the next run pending
//     scoring), so the operator knows what to dispatch next.
//   - restore-worktree resets, cleans, and re-neutralises a warm config's task worktree after an attempt,
//     whatever that attempt's outcome; never called for the cold config, which discards its own
//     per-repetition worktree outright instead of restoring it in place.
//   - probe-record, record-score, redact, ingest, and invalidate (cli-run-commands, this batch's
//     follow-up) record what a finished run or probe attempt produced.
//   - cold-cell and summarize (also cli-run-commands) aggregate the matrix's own disposition once enough
//     of it has run.
//
// This batch (cli-session-commands) lands prepare-session, warm, next-run, and restore-worktree;
// cli-run-commands lands the remaining seven.
package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/Knatte18/quarry/bench/loomyard-eval/ladder/internal/ladder"
)

// defaultLadderPath is the --ladder flag's default value: the committed ladder.yaml's own repo-relative
// path, matching this suite's own convention (Python and Go alike) of being invoked from the repository
// root.
const defaultLadderPath = "bench/loomyard-eval/ladder/ladder.yaml"

// ladderFlagName and resultsRootFlagName are the persistent flag names resolveLadder/resolveResultsRoot
// read, so every subcommand shares one derivation site rather than each re-parsing its own flag.
const (
	ladderFlagName      = "ladder"
	resultsRootFlagName = "results-root"
)

// Command returns ladderbench's own cobra command tree.
func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ladderbench",
		Short: "the quarry-mcp capability ladder benchmark's session-boundary CLI",
	}

	cmd.PersistentFlags().String(ladderFlagName, defaultLadderPath, "path to ladder.yaml")
	cmd.PersistentFlags().String(resultsRootFlagName, "", "the results directory this invocation reads and writes under (required by every subcommand except prepare-session --release)")

	cmd.AddCommand(prepareSessionCommand())
	cmd.AddCommand(nextRunCommand())
	cmd.AddCommand(warmCommand())
	cmd.AddCommand(restoreWorktreeCommand())
	cmd.AddCommand(ingestCommand())
	cmd.AddCommand(invalidateCommand())
	cmd.AddCommand(redactCommand())
	cmd.AddCommand(recordScoreCommand())

	return cmd
}

// resolveLadder loads and validates the --ladder flag's file through ladder.LoadLadder, so a load
// failure -- a malformed ladder.yaml, or one that fails LoadLadder's own validation -- surfaces to the
// caller rather than being swallowed; no subcommand re-derives this itself.
func resolveLadder(cmd *cobra.Command) (*ladder.Ladder, error) {
	path, err := cmd.Flags().GetString(ladderFlagName)
	if err != nil {
		return nil, fmt.Errorf("ladderbench: read --%s flag: %w", ladderFlagName, err)
	}
	l, err := ladder.LoadLadder(path)
	if err != nil {
		return nil, fmt.Errorf("ladderbench: load ladder file %s: %w", path, err)
	}
	return l, nil
}

// resolveResultsRoot reads the --results-root flag, erroring when it is empty -- every subcommand needs
// a results root to key its session lock, run directories, or ingest/scoring markers against.
func resolveResultsRoot(cmd *cobra.Command) (string, error) {
	root, err := cmd.Flags().GetString(resultsRootFlagName)
	if err != nil {
		return "", fmt.Errorf("ladderbench: read --%s flag: %w", resultsRootFlagName, err)
	}
	if root == "" {
		return "", fmt.Errorf("ladderbench: --%s is required", resultsRootFlagName)
	}
	return root, nil
}

// resolveRepoRoot resolves the repository root every ladder-declared path (a task file, the benchmark
// README, the tracked orchestration skill) is resolved relative to. It is the process's current working
// directory: every tool in this suite, Python and Go alike, is documented as run from the repository
// root, matching scripts/run_ladder.py's own Path.cwd() default.
func resolveRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("ladderbench: resolve repo root: %w", err)
	}
	return dir, nil
}

// realBuild is ladder.Builder's real implementation, run through the standard library: it runs args in
// dir with env as the child's full environment and returns its combined stdout+stderr, mirroring
// ladder.RunGit's own combined-output convention. Shared by every subcommand in this batch that calls
// ladder.BuildServer, so the real spawn has one implementation.
func realBuild(dir string, env []string, args ...string) (string, error) {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	cmd.Env = env
	output, err := cmd.CombinedOutput()
	return string(output), err
}
