// Command runmatrix drives every warm-matrix run session for one ladder file to completion,
// automatically, one live claude session at a time. Its defaults (no flags) reproduce its original
// purpose exactly: the main 14-config x 3-rep = 42-run matrix against ladder.yaml. --ladder,
// --results-root, and --configs override those three independently for a different matrix -- e.g. a
// distilled companion file's own smaller config set, run against its own results root -- without
// touching the main matrix's data or in-flight scratch dirs.
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
// One thing is deliberately still not run by this command:
//   - The scoring session. It must run after every run (warm AND cold) has ingested, not partway through.
//
// The cold config (a5-bundle-cold) IS driven by this command, via --cold-config, but through a separate
// loop (runCold) rather than run's warm-matrix loop: its per-attempt lifecycle is a full session relaunch
// on failure, never an in-session retry (see ladder-run/SKILL.md's "The cold config" section) --
// structurally different from a warm session's loop. runCold never calls warm (a cold repetition's whole
// premise is that its worktree starts with no resident daemon), and tears down that repetition's
// disposable worktree via `cold-cell --teardown --rep <n>` unconditionally after every attempt -- whatever
// the outcome -- since prepare-session's BuildWorktree refuses to reuse a stale path and the next attempt
// (or the next repetition) needs that path clear. Because prepare-session always rebuilds the worktree
// from scratch on every call, this loop is naturally self-healing against the worktree simply not
// existing when expected (e.g. a reboot clearing a /tmp-backed worktree between attempts) -- no special
// case is needed for that, only for the documented cold-before live-daemon self-abort (see runCold).
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	repoRoot  = "/home/hanf/Code/quarry/wts/quarry"
	ladderDir = repoRoot + "/bench/loomyard-eval/ladder"
	binPath   = "/tmp/ladderbench"

	defaultLadderPath  = ladderDir + "/ladder.yaml"
	defaultResultsRoot = ladderDir + "/results/2026-08-30"

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

// defaultConfigIDs lists the main matrix's 14 warm-matrix config IDs in the exact order ladder.yaml's
// own configs: list declares them. Kept as a literal here rather than derived from ladder.yaml, since no
// ladderbench subcommand currently lists config IDs; if ladder.yaml's configs: list ever changes, this
// list must be updated to match. --configs overrides this for a different ladder file's own config set.
var defaultConfigIDs = []string{
	"a0-none", "a1-toc-file", "a2-toc-dir", "a3-toc-pair", "a4-toc-pair-symbol", "a5-bundle",
	"b0-none", "b1-symbol", "b2-definition", "b3-references", "b4-lsp-trio", "b5-impact",
	"b6-assert-no-callers", "b7-bundle",
}

func main() {
	ladderPath := flag.String("ladder", defaultLadderPath, "path to the ladder file to drive")
	resultsRoot := flag.String("results-root", defaultResultsRoot, "the results directory to read and write under")
	configsFlag := flag.String("configs", "", "comma-separated config IDs to drive, in order (default: the main matrix's 14 warm configs)")
	coldConfig := flag.String("cold-config", "", "drive this single cold config's full lifecycle (prepare-session, session, teardown, repeat, finalize) instead of the warm-matrix loop; --configs is ignored when set")
	scoring := flag.Bool("scoring", false, "drive the single shared scoring session (prepare-session --scoring, one live session, wait for its \"scored\" outcome) instead of the warm-matrix loop; --configs and --cold-config are ignored when set")
	flag.Parse()

	if *scoring {
		if err := runScoring(*ladderPath, *resultsRoot); err != nil {
			fmt.Fprintln(os.Stderr, "runmatrix:", err)
			os.Exit(1)
		}
		return
	}

	if *coldConfig != "" {
		if err := runCold(*ladderPath, *resultsRoot, *coldConfig); err != nil {
			fmt.Fprintln(os.Stderr, "runmatrix:", err)
			os.Exit(1)
		}
		return
	}

	configIDs := defaultConfigIDs
	if *configsFlag != "" {
		configIDs = strings.Split(*configsFlag, ",")
		for i, id := range configIDs {
			configIDs[i] = strings.TrimSpace(id)
		}
	}

	if err := run(*ladderPath, *resultsRoot, configIDs); err != nil {
		fmt.Fprintln(os.Stderr, "runmatrix:", err)
		os.Exit(1)
	}
}

// runCold drives one cold config's full lifecycle to completion: repeatedly resolve the next pending
// repetition via next-run, prepare-session it (which unconditionally rebuilds that repetition's
// disposable worktree -- see this file's own top comment for why that makes the loop self-healing against
// the worktree not existing), launch and wait for exactly one live session, tear down that repetition's
// worktree unconditionally regardless of outcome, and either loop to the next repetition (on "ingested")
// or halt the whole run (on anything else, matching run's own halt-on-non-ingested rule). Once next-run
// reports nothing pending, it finalises the cold cell (`cold-cell`, no flags) and returns.
//
// warm is never called here, per this file's own top comment.
//
// prepare-session can itself fail here in one expected, recoverable way: the cold-before gate finding a
// live daemon still resident from the previous attempt and self-aborting -- ladder.go's own
// prepareColdSessionAfterGate already invalidates that attempt and records why (see abortColdAttempt in
// cmd/ladderbench/preparesession.go); this loop treats that specific failure as "retry immediately", not
// as a fatal error, by checking the printed message for its own fixed "aborted as attempt" substring.
func runCold(ladderPath, resultsRoot, configID string) error {
	if err := runVisible("go", "build", "-o", binPath, ladderDir+"/cmd/ladderbench"); err != nil {
		return fmt.Errorf("build ladderbench: %w", err)
	}

	// Defensive, same reasoning as run's own release-before-start call.
	_ = runVisible(binPath, "prepare-session", "--release", "--ladder", ladderPath, "--results-root", resultsRoot)

	for {
		nextOut, err := ladderbenchOutput("next-run", "--config-id", configID, "--ladder", ladderPath, "--results-root", resultsRoot)
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

		fmt.Printf("== %s rep %s: preparing cold session (rebuilding disposable worktree) ==\n", configID, rep)
		prepOut, err := ladderbenchOutput("prepare-session", "--config-id", configID, "--rep", rep, "--ladder", ladderPath, "--results-root", resultsRoot)
		if err != nil {
			if strings.Contains(err.Error(), "aborted as attempt") {
				fmt.Printf("== %s rep %s: cold-before gate found a live daemon, self-invalidated -- retrying next attempt ==\n", configID, rep)
				continue
			}
			return fmt.Errorf("prepare-session --config-id %s --rep %s: %w", configID, rep, err)
		}
		scratchDir := fieldValue(prepOut, "scratch_dir")
		if scratchDir == "" {
			return fmt.Errorf("prepare-session --config-id %s --rep %s printed no scratch_dir: line", configID, rep)
		}

		fmt.Printf("== %s rep %s: launching in tmux session %q (no warm -- cold config) -- `tmux attach -t %s` to watch ==\n",
			configID, rep, tmuxSession, tmuxSession)
		outcome, waitErr := launchAndWait(scratchDir)

		// /ladder-run's own last step already releases the lock before it writes the outcome marker; this
		// is a defensive backstop for the operator-killed-it path, same as run's own call.
		_ = runVisible(binPath, "prepare-session", "--release", "--ladder", ladderPath, "--results-root", resultsRoot)

		if waitErr != nil {
			return fmt.Errorf("session for %s rep %s: %w", configID, rep, waitErr)
		}
		fmt.Printf("== %s rep %s: outcome = %s ==\n", configID, rep, outcome)

		fmt.Printf("== %s rep %s: tearing down disposable worktree ==\n", configID, rep)
		if err := runVisible(binPath, "cold-cell", "--teardown", "--rep", rep, "--ladder", ladderPath, "--results-root", resultsRoot); err != nil {
			return fmt.Errorf("cold-cell --teardown --rep %s: %w", rep, err)
		}

		if outcome != "ingested" {
			return fmt.Errorf("%s rep %s outcome was %q, not \"ingested\" -- halting rather than continuing past it, per ladder-run/SKILL.md's own halt-on-truncated rule", configID, rep, outcome)
		}
	}

	// Deliberately does NOT call `cold-cell` (finalize) here: ColdCellDisposition classifies a repetition
	// as "cold" vs "no_daemon_signal" by reading run.json's own state, and WriteRunJSON -- see its own doc
	// comment in runstate.go -- is written only by the scoring session, after score.json exists. Calling
	// finalize before every one of this config's repetitions has been scored makes every repetition look
	// incomplete to ColdCellDisposition, which falls through to a bogus "not-run" disposition even though
	// every repetition actually ran and ingested cleanly. Run `cold-cell` (no flags) by hand once the
	// scoring session has scored every ingested run.
	fmt.Printf("== %s: all repetitions ingested. Do NOT finalize yet -- run the scoring session first, then `ladderbench cold-cell --ladder %s --results-root %s` by hand ==\n", configID, ladderPath, resultsRoot)
	return nil
}

// runScoring drives the single shared scoring session to completion: prepare-session --scoring once,
// launch and wait for exactly one live session (ladder-run/SKILL.md's own scoring-session loop iterates
// every ingested-but-unscored run internally, inside that one session, so there is only ever one session
// to launch here -- unlike the per-repetition warm/cold loops), and report its outcome.
//
// Unlike run/runCold, a non-"scored" outcome is reported but not treated as fatal -- there is no next
// config or repetition to halt ahead of; this is the whole scoring pass.
func runScoring(ladderPath, resultsRoot string) error {
	if err := runVisible("go", "build", "-o", binPath, ladderDir+"/cmd/ladderbench"); err != nil {
		return fmt.Errorf("build ladderbench: %w", err)
	}

	// Defensive, same reasoning as run's own release-before-start call.
	_ = runVisible(binPath, "prepare-session", "--release", "--ladder", ladderPath, "--results-root", resultsRoot)

	fmt.Println("== scoring: preparing session ==")
	prepOut, err := ladderbenchOutput("prepare-session", "--scoring", "--ladder", ladderPath, "--results-root", resultsRoot)
	if err != nil {
		return fmt.Errorf("prepare-session --scoring: %w", err)
	}
	scratchDir := fieldValue(prepOut, "scratch_dir")
	if scratchDir == "" {
		return fmt.Errorf("prepare-session --scoring printed no scratch_dir: line")
	}

	fmt.Printf("== scoring: launching in tmux session %q -- `tmux attach -t %s` to watch ==\n", tmuxSession, tmuxSession)
	outcome, waitErr := launchAndWait(scratchDir)

	// /ladder-run's own last step already releases the lock before it writes the outcome marker; this is
	// a defensive backstop for the operator-killed-it path, same as run's/runCold's own call.
	_ = runVisible(binPath, "prepare-session", "--release", "--ladder", ladderPath, "--results-root", resultsRoot)

	if waitErr != nil {
		return fmt.Errorf("scoring session: %w", waitErr)
	}
	fmt.Printf("== scoring: outcome = %s ==\n", outcome)
	if outcome != "scored" {
		return fmt.Errorf("scoring outcome was %q, not \"scored\"", outcome)
	}
	return nil
}

func run(ladderPath, resultsRoot string, configIDs []string) error {
	if err := runVisible("go", "build", "-o", binPath, ladderDir+"/cmd/ladderbench"); err != nil {
		return fmt.Errorf("build ladderbench: %w", err)
	}

	// Defensive: clears any lock left behind by an earlier, separately-run session (a probe, a manual
	// test, or a prior interrupted run of this same command) so the first prepare-session below never
	// refuses to acquire it. Releasing an absent or already-released lock is not an error, so its own
	// exit status is ignored here.
	_ = runVisible(binPath, "prepare-session", "--release", "--ladder", ladderPath, "--results-root", resultsRoot)

	for _, configID := range configIDs {
		for {
			nextOut, err := ladderbenchOutput("next-run", "--config-id", configID, "--ladder", ladderPath, "--results-root", resultsRoot)
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
			prepOut, err := ladderbenchOutput("prepare-session", "--config-id", configID, "--rep", rep, "--ladder", ladderPath, "--results-root", resultsRoot)
			if err != nil {
				return fmt.Errorf("prepare-session --config-id %s --rep %s: %w", configID, rep, err)
			}
			scratchDir := fieldValue(prepOut, "scratch_dir")
			if scratchDir == "" {
				return fmt.Errorf("prepare-session --config-id %s --rep %s printed no scratch_dir: line", configID, rep)
			}

			fmt.Printf("== %s rep %s: warming daemon ==\n", configID, rep)
			if err := runVisible(binPath, "warm", "--config-id", configID, "--ladder", ladderPath, "--results-root", resultsRoot); err != nil {
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
			_ = runVisible(binPath, "prepare-session", "--release", "--ladder", ladderPath, "--results-root", resultsRoot)

			if outcome != "ingested" {
				return fmt.Errorf("%s rep %s outcome was %q, not \"ingested\" -- halting rather than continuing past it, per ladder-run/SKILL.md's own halt-on-truncated rule", configID, rep, outcome)
			}
		}
	}

	fmt.Printf("== matrix complete (%d configs) ==\n", len(configIDs))
	fmt.Println("Still separate: the cold cell and the scoring session, if applicable to this ladder file.")
	return nil
}

// mcpTrustDialogSignature is the fixed text Claude Code's own "New MCP server found in this project"
// startup dialog prints, confirmed empirically against a live pane. No settings.json field, ~/.claude.json
// project entry, or --strict-mcp-config flag this suite tried actually suppresses it (all three tested
// live and ruled out) -- it appears to require an actual keystroke, and every server this command ever
// launches with is the operator's own quarry-mcp binary, declared by this same harness's own generated
// .mcp.json, and already approved by hand dozens of times over the course of this matrix -- so
// launchAndWait sends that keystroke itself once it sees the dialog.
//
// This is safe specifically because it is scoped to tmuxSession, the one pane this command itself just
// created for a scratch directory it itself just prepared: it is not a general "answer any prompt"
// mechanism, and does not touch the operator's own separate, unrelated Claude Code sessions.
const mcpTrustDialogSignature = "New MCP server found in this project"

// autoAcceptCooldown bounds how often launchAndWait re-sends the accept keystroke while the dialog
// signature is still visible -- long enough for one send to register and the dialog to clear before
// considering it stuck again, short enough that a genuinely re-appeared dialog (a second, different
// server) is still answered promptly.
const autoAcceptCooldown = 3 * time.Second

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
// mcpTrustDialogSignature, sending "2" then Enter to select "Use this and all future MCP servers in this
// project" when it appears -- see mcpTrustDialogSignature's own doc comment for why sending it here is
// safe and why option 2 specifically, rather than the plain Enter accepting the single-session-only
// default option 1.
func launchAndWait(scratchDir string) (string, error) {
	if err := runVisible("tmux", "new-session", "-d", "-s", tmuxSession, ladderDir+"/launch-session.sh", scratchDir); err != nil {
		return "", fmt.Errorf("start tmux session: %w", err)
	}

	markerPath := filepath.Join(scratchDir, outcomeMarkerName)
	var lastAutoAccept time.Time
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

		if paneShowsMCPTrustDialog() && time.Since(lastAutoAccept) >= autoAcceptCooldown {
			fmt.Printf("== %s: auto-accepting Claude Code's MCP trust dialog (option 2: this and all future servers in this project) ==\n", scratchDir)
			_ = runVisible("tmux", "send-keys", "-t", tmuxSession, "2", "Enter")
			lastAutoAccept = time.Now()
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
