// Command runmatrix drives every warm-matrix run session (14 configs x 3 reps = 42) to completion,
// automatically, one live claude session at a time -- the Go+tmux counterpart to ../run-matrix.sh, for an
// operator who wants a session survivable across a dropped terminal and attachable from elsewhere.
//
// This does not run headless. Each session is launched inside a named tmux session
// ("tmux new-session -s ladder-run", never claude -p/--print) with "/ladder-run" pre-submitted as its
// first message via claude's own positional prompt argument, so the operator never types it. This
// process's own stdin/stdout/stderr are wired straight through to that tmux session, so running this
// command in a terminal is exactly as interactive and killable as running claude directly there --
// nothing about it is unattended. The one thing tmux adds is that the session keeps running, reattachable
// with `tmux attach -t ladder-run`, if the terminal driving this command is itself lost.
//
// It shells out to the ladderbench binary for every state-touching step (next-run, prepare-session,
// warm) rather than calling internal/ladder directly, deliberately: those CLI paths already own the
// build/lock/worktree orchestration this tool has no business re-deriving, and reusing the exact binary
// the operator would otherwise run by hand keeps this tool and that binary provably in sync.
//
// Two things are deliberately not run by this command:
//   - The cold config (a5-bundle-cold). Its per-attempt lifecycle is a full session relaunch on failure,
//     never an in-session retry (see ladder-run/SKILL.md's "The cold config" section) -- structurally
//     different from a warm session's loop, and driven separately.
//   - The scoring session. It must run after every run (warm AND cold) has ingested, not partway through.
package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const (
	repoRoot    = "/home/knatte/Code/quarry/wts/quarry"
	ladderDir   = repoRoot + "/bench/loomyard-eval/ladder"
	binPath     = "/tmp/ladderbench"
	resultsRoot = ladderDir + "/results/2026-08-30"
	tmuxSession = "ladder-run"
)

// warmConfigIDs lists the 14 warm-matrix config IDs in the exact order ladder.yaml's own configs: list
// declares them. Kept as a literal here, mirroring run-matrix.sh's own literal, rather than derived from
// ladder.yaml, since no ladderbench subcommand currently lists config IDs; if ladder.yaml's configs: list
// ever changes, this list must be updated to match.
var warmConfigIDs = []string{
	"a0-none", "a1-toc-file", "a2-toc-dir", "a3-toc-pair", "a4-toc-pair-symbol", "a5-bundle",
	"b0-none", "b1-symbol", "b2-definition", "b3-references", "b4-lsp-trio", "b5-impact",
	"b6-assert-no-callers", "b7-bundle",
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "runmatrix:", err)
		os.Exit(1)
	}
}

func run() error {
	if err := runVisible("go", "build", "-o", binPath, ladderDir+"/cmd/ladderbench"); err != nil {
		return fmt.Errorf("build ladderbench: %w", err)
	}

	// Defensive: clears any lock left behind by an earlier, separately-run session (a probe, a manual
	// test, or a prior interrupted run of this same command) so the first prepare-session below never
	// refuses to acquire it. Releasing an absent or already-released lock is not an error, so its own
	// exit status is ignored here.
	_ = runVisible(binPath, "prepare-session", "--release", "--results-root", resultsRoot)

	for _, configID := range warmConfigIDs {
		for {
			nextOut, err := ladderbenchOutput("next-run", "--config-id", configID, "--results-root", resultsRoot)
			if err != nil {
				return fmt.Errorf("next-run --config-id %s: %w", configID, err)
			}
			if strings.Contains(nextOut, "nothing pending") {
				fmt.Printf("== %s: all repetitions already ingested ==\n", configID)
				break
			}
			rep := fieldValue(nextOut, "rep")
			if rep == "" {
				return fmt.Errorf("next-run --config-id %s printed no rep: line", configID)
			}

			fmt.Printf("== %s rep %s: preparing session ==\n", configID, rep)
			prepOut, err := ladderbenchOutput("prepare-session", "--config-id", configID, "--rep", rep, "--results-root", resultsRoot)
			if err != nil {
				return fmt.Errorf("prepare-session --config-id %s --rep %s: %w", configID, rep, err)
			}
			scratchDir := fieldValue(prepOut, "scratch_dir")
			if scratchDir == "" {
				return fmt.Errorf("prepare-session --config-id %s --rep %s printed no scratch_dir: line", configID, rep)
			}

			fmt.Printf("== %s rep %s: warming daemon ==\n", configID, rep)
			if err := runVisible(binPath, "warm", "--config-id", configID, "--results-root", resultsRoot); err != nil {
				return fmt.Errorf("warm --config-id %s: %w", configID, err)
			}

			fmt.Printf("== %s rep %s: launching in tmux session %q -- `tmux attach -t %s` from another terminal to watch; close the pane or `tmux kill-session -t %s` if anything looks wrong ==\n",
				configID, rep, tmuxSession, tmuxSession, tmuxSession)
			if err := launchInTmux(scratchDir); err != nil {
				return fmt.Errorf("launch tmux session for %s rep %s: %w", configID, rep, err)
			}

			// /ladder-run's own last step already releases the lock on a normal completion; this is a
			// defensive backstop for a session that was closed before it got that far.
			_ = runVisible(binPath, "prepare-session", "--release", "--results-root", resultsRoot)
		}
	}

	fmt.Println("== warm matrix complete (42 runs) ==")
	fmt.Println("Still separate: the cold cell (3 reps) and the scoring session.")
	return nil
}

// launchInTmux runs launch-session.sh inside a fresh tmux session named tmuxSession, with this process's
// own stdin/stdout/stderr wired straight through -- `tmux new-session` without -d attaches in the calling
// terminal exactly like a direct foreground exec would, blocking until the session's pane exits (tmux's
// default behaviour is to destroy a session once its last pane's command exits), while the session itself
// stays reattachable from elsewhere for as long as it runs.
func launchInTmux(scratchDir string) error {
	cmd := exec.Command("tmux", "new-session", "-s", tmuxSession, ladderDir+"/launch-session.sh", scratchDir)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// runVisible runs name with args, streaming its own stdout/stderr straight through so build/warm/release
// progress is visible as it happens rather than buffered until the call returns.
func runVisible(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// ladderbenchOutput runs binPath with args and returns its captured stdout, with stderr streamed through
// directly (so a failure's diagnostic still reaches the operator immediately, not just via the wrapped
// error) -- next-run and prepare-session's own stdout is this tool's only source for the rep and
// scratch_dir values fieldValue below extracts.
func ladderbenchOutput(args ...string) (string, error) {
	cmd := exec.Command(binPath, args...)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// fieldValue returns the value after "<field>: " on output's first line starting with that prefix, or ""
// if no such line is present, against next-run's/prepare-session's fixed "field: value" line shapes.
func fieldValue(output, field string) string {
	prefix := field + ": "
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if value, ok := strings.CutPrefix(line, prefix); ok {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
