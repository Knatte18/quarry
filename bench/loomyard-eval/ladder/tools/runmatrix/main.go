// Command runmatrix drives every warm-matrix run session (14 configs x 3 reps = 42) to completion,
// automatically, one live claude session at a time.
//
// This does not run headless. Each session is launched inside a detached tmux session
// ("tmux new-session -d -s ladder-run", never claude -p/--print) with "/ladder-run" pre-submitted as its
// first message via claude's own positional prompt argument, so the operator never types it. The operator
// is expected to `tmux attach -t ladder-run` to watch, exactly as they would watch any other session in
// this suite -- free to intervene, answer a permission prompt, or kill the session if something looks
// wrong. Detecting "this session is done, and did it succeed" is not scraped from the pane's own text: a
// claude session never exits on its own once it finishes responding (it waits for the next human message
// indefinitely, same as any other interactive session), so ladder-run/SKILL.md's run-session loop writes
// a completion marker (<scratch-dir>/.ladder-run-outcome, naming "ingested", "truncated", or "exhausted")
// as its own actual final step -- this command polls for that file and kills the tmux session once it
// sees it, after a short grace period so an operator who is watching still gets to read the session's own
// final summary message before the pane disappears.
//
// A "truncated" or "exhausted" outcome halts this command outright rather than continuing to the next
// config -- matching ladder-run/SKILL.md's own "a truncated outcome halts the whole matrix, never
// retried" rule. So does the tmux session disappearing without ever writing a marker: that means the
// operator closed it themselves, and there is then no reliable way to tell whether the attempt actually
// finished, so this command stops and leaves it to the operator to sort out rather than guessing.
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
	"path/filepath"
	"strings"
	"time"
)

const (
	repoRoot    = "/home/knatte/Code/quarry/wts/quarry"
	ladderDir   = repoRoot + "/bench/loomyard-eval/ladder"
	binPath     = "/tmp/ladderbench"
	resultsRoot = ladderDir + "/results/2026-08-30"
	tmuxSession = "ladder-run"

	// outcomeMarkerName is ladder-run/SKILL.md's own fixed filename for its run-session loop's completion
	// signal -- see that file's own doc comment for why it exists and what writes it.
	outcomeMarkerName = ".ladder-run-outcome"
	pollInterval      = 2 * time.Second
	// completionGraceDelay is how long this command waits, once it sees the outcome marker, before
	// killing the tmux session -- purely so an operator who is attached and watching still gets a moment
	// to read the session's own final summary message before the pane disappears.
	completionGraceDelay = 5 * time.Second
)

// warmConfigIDs lists the 14 warm-matrix config IDs in the exact order ladder.yaml's own configs: list
// declares them. Kept as a literal here rather than derived from ladder.yaml, since no ladderbench
// subcommand currently lists config IDs; if ladder.yaml's configs: list ever changes, this list must be
// updated to match.
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

			fmt.Printf("== %s rep %s: launching in tmux session %q -- `tmux attach -t %s` to watch ==\n",
				configID, rep, tmuxSession, tmuxSession)
			outcome, err := launchAndWait(scratchDir)
			if err != nil {
				return fmt.Errorf("session for %s rep %s: %w", configID, rep, err)
			}
			fmt.Printf("== %s rep %s: outcome = %s ==\n", configID, rep, outcome)

			// /ladder-run's own last step already releases the lock before it writes the outcome marker;
			// this is a defensive backstop for the operator-killed-it path, where that never happened.
			_ = runVisible(binPath, "prepare-session", "--release", "--results-root", resultsRoot)

			if outcome != "ingested" {
				return fmt.Errorf("%s rep %s outcome was %q, not \"ingested\" -- halting rather than continuing past it, per ladder-run/SKILL.md's own halt-on-truncated rule", configID, rep, outcome)
			}
		}
	}

	fmt.Println("== warm matrix complete (42 runs) ==")
	fmt.Println("Still separate: the cold cell (3 reps) and the scoring session.")
	return nil
}

// mcpTrustDialogSignature is the fixed text Claude Code's own "New MCP server found in this project"
// startup dialog prints, confirmed empirically against a live pane. No settings.json field, ~/.claude.json
// project entry, or --strict-mcp-config flag this suite tried actually suppresses it (all three tested
// live and ruled out) -- it appears to require an actual keystroke from whoever is attached, which this
// command deliberately does not send itself (scripting a keypress into a permission dialog is exactly the
// shape of thing Claude Code's own auto-mode classifier already refuses, for good reason). So this only
// detects the stall and says so loudly instead of silently polling for a marker that will never appear
// until a human answers it.
const mcpTrustDialogSignature = "New MCP server found in this project"

// stuckAlertInterval is how often launchAndWait re-prints its "attach and answer the dialog" alert while
// the pane appears stuck on the MCP trust dialog -- repeated, not printed once, since a single line is
// easy to scroll past in a long-running terminal.
const stuckAlertInterval = 30 * time.Second

// launchAndWait starts scratchDir's session detached inside tmuxSession and polls for either its
// completion marker (outcomeMarkerName) to appear or the tmux session itself to disappear.
//
// On a marker appearing, it waits completionGraceDelay, kills the tmux session, and returns the marker's
// trimmed contents as the outcome.
//
// On the tmux session disappearing first, it returns an error: that only happens if the operator closed
// the pane themselves (tmux's own default is to keep a session alive until its pane's command exits, and
// nothing this suite launches inside it ever does that on its own before writing the marker), and there
// is then no reliable way to tell whether the attempt actually finished -- so this stops rather than
// guessing.
//
// While waiting, it also captures the pane's own visible content on every poll and watches for
// mcpTrustDialogSignature -- Claude Code's MCP server trust prompt requires an actual human keystroke and
// nothing this command does can answer it, so a session stuck on it would otherwise sit silently until
// someone happened to check. Detecting it prints a repeated, hard-to-miss alert naming exactly what to do
// instead.
func launchAndWait(scratchDir string) (string, error) {
	if err := runVisible("tmux", "new-session", "-d", "-s", tmuxSession, ladderDir+"/launch-session.sh", scratchDir); err != nil {
		return "", fmt.Errorf("start tmux session: %w", err)
	}

	markerPath := filepath.Join(scratchDir, outcomeMarkerName)
	var lastStuckAlert time.Time
	for {
		data, err := os.ReadFile(markerPath)
		switch {
		case err == nil:
			time.Sleep(completionGraceDelay)
			_ = runVisible("tmux", "kill-session", "-t", tmuxSession)
			return strings.TrimSpace(string(data)), nil
		case !os.IsNotExist(err):
			return "", fmt.Errorf("read %s: %w", markerPath, err)
		}

		if !tmuxSessionExists() {
			return "", fmt.Errorf("tmux session %q ended without ever writing %s -- assuming the operator closed it themselves; not guessing whether the attempt finished", tmuxSession, markerPath)
		}

		if paneShowsMCPTrustDialog() && time.Since(lastStuckAlert) >= stuckAlertInterval {
			fmt.Printf("\a!! ATTENTION: %s is waiting on Claude Code's MCP trust dialog and nothing but a human keystroke can answer it.\n!! Run `tmux attach -t %s`, press Enter (or 1) to approve, then detach (ctrl-b d) -- this command will pick back up on its own.\n",
				scratchDir, tmuxSession)
			lastStuckAlert = time.Now()
		}

		time.Sleep(pollInterval)
	}
}

// tmuxSessionExists reports whether tmuxSession is still a live tmux session.
func tmuxSessionExists() bool {
	return exec.Command("tmux", "has-session", "-t", tmuxSession).Run() == nil
}

// paneShowsMCPTrustDialog reports whether tmuxSession's current visible pane content contains
// mcpTrustDialogSignature. A capture failure (session gone between the has-session check above and this
// call, a genuine race under concurrent polling) is treated as "not showing it", never as an error --
// tmuxSessionExists is the authority on whether the session itself is still alive, this is only a
// best-effort peek at its content.
func paneShowsMCPTrustDialog() bool {
	out, err := exec.Command("tmux", "capture-pane", "-t", tmuxSession, "-p").Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), mcpTrustDialogSignature)
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
// error) -- next-run's/prepare-session's own stdout is this tool's only source for the rep and
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
