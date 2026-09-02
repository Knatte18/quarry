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
// --all runs the whole thing for one ladder file, end to end, in the only order that is valid: every
// warm config (in the file's own declared order), then every cold config, then the single scoring
// session, then cold-cell finalisation (when the file has a cold config), then summarize -- and finally
// writes provenance.json (which quarry commit, whether the tree was dirty, which quarry-mcp binary was
// built, which Loomyard commit) into the results root and prints a per-cell table. It is resumable:
// every step skips work already recorded under the results root. bench/loomyard-eval/ladder/run.sh is
// the operator-facing wrapper around it (preflight checks, shortnames for the ladder files, dated
// results-root default).
//
// Without --all, one thing is deliberately still not run by this command:
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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Knatte18/quarry/bench/loomyard-eval/ladder/internal/ladder"
)

const (
	binPath = "/tmp/ladderbench"

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

// repoRoot, ladderDir, defaultLadderPath, and defaultResultsRoot are resolved once, at the top of main(),
// from the process's current working directory -- never hardcoded to a specific machine's checkout path.
// This command is documented as run from the quarry repo root, matching every ladderbench subcommand's
// own resolveRepoRoot() convention (cmd/ladderbench/root.go).
var (
	repoRoot           string
	ladderDir          string
	defaultLadderPath  string
	defaultResultsRoot string
)

func main() {
	var err error
	repoRoot, err = os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "runmatrix: resolve repo root:", err)
		os.Exit(1)
	}
	ladderDir = filepath.Join(repoRoot, "bench/loomyard-eval/ladder")
	defaultLadderPath = filepath.Join(ladderDir, "ladder.yaml")
	defaultResultsRoot = filepath.Join(ladderDir, "results/2026-08-30")

	ladderPath := flag.String("ladder", defaultLadderPath, "path to the ladder file to drive")
	resultsRoot := flag.String("results-root", defaultResultsRoot, "the results directory to read and write under")
	configsFlag := flag.String("configs", "", "comma-separated config IDs to drive, in order (default: the main matrix's 14 warm configs)")
	coldConfig := flag.String("cold-config", "", "drive this single cold config's full lifecycle (prepare-session, session, teardown, repeat, finalize) instead of the warm-matrix loop; --configs is ignored when set")
	scoring := flag.Bool("scoring", false, "drive the single shared scoring session (prepare-session --scoring, one live session, wait for its \"scored\" outcome) instead of the warm-matrix loop; --configs and --cold-config are ignored when set")
	all := flag.Bool("all", false, "drive the whole ladder file end to end: every warm config in declared order, every cold config, the scoring session, cold-cell finalisation, summarize, provenance.json, and a per-cell table; --configs, --cold-config and --scoring are ignored when set; --results-root defaults to results/<today>[-<ladder suffix>]")
	flag.Parse()

	if *all {
		resultsRootGiven := false
		flag.Visit(func(f *flag.Flag) {
			if f.Name == "results-root" {
				resultsRootGiven = true
			}
		})
		root := *resultsRoot
		if !resultsRootGiven {
			root = defaultDatedResultsRoot(*ladderPath)
		}
		if err := runAll(*ladderPath, root); err != nil {
			fmt.Fprintln(os.Stderr, "runmatrix:", err)
			os.Exit(1)
		}
		return
	}

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
			recordServerHash(configID, rep)

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

// defaultDatedResultsRoot derives --all's default results root from today's date and the ladder file's
// own name: ladder.yaml -> results/<YYYY-MM-DD>, ladder-followup.yaml -> results/<YYYY-MM-DD>-followup.
// Re-running the same file on the same day therefore resumes into the same root, which is what every
// step under runAll expects; a fresh root on the same day is one --results-root away.
func defaultDatedResultsRoot(ladderPath string) string {
	base := strings.TrimSuffix(filepath.Base(ladderPath), filepath.Ext(ladderPath))
	suffix := strings.TrimPrefix(base, "ladder")
	// "ladder-followup" -> "-followup"; "ladder" -> ""; anything else keeps its own name as the suffix.
	if suffix == base {
		suffix = "-" + base
	}
	return filepath.Join(ladderDir, "results", time.Now().Format("2006-01-02")+suffix)
}

// runAll drives one ladder file to a written summary.json, in the only valid order, resuming past
// anything the results root already records. See the package comment's --all paragraph.
func runAll(ladderPath, resultsRoot string) error {
	l, err := ladder.LoadLadder(ladderPath)
	if err != nil {
		return err
	}
	if err := ladder.RequirePins(l); err != nil {
		return err
	}

	var warmIDs, coldIDs []string
	for _, c := range l.Configs {
		if c.Cold {
			coldIDs = append(coldIDs, c.ID)
		} else {
			warmIDs = append(warmIDs, c.ID)
		}
	}

	if err := os.MkdirAll(resultsRoot, 0o755); err != nil {
		return fmt.Errorf("create results root %s: %w", resultsRoot, err)
	}
	serverHashes = loadServerHashes(resultsRoot)
	prov := newProvenance(ladderPath, resultsRoot)
	prov.ServerHashes = serverHashes
	if err := prov.write(resultsRoot); err != nil {
		return err
	}
	fmt.Printf("== runmatrix --all: %s -> %s (%d warm, %d cold configs, reps %d) ==\n",
		ladderPath, resultsRoot, len(warmIDs), len(coldIDs), l.Reps)
	if prov.QuarryDirty {
		fmt.Println("== NOTE: the quarry working tree is dirty and quarry-mcp is rebuilt from it before every repetition;")
		fmt.Println("   do not edit quarry source while this matrix runs -- provenance.json/server_hashes records every build ==")
	}

	if err := run(ladderPath, resultsRoot, warmIDs); err != nil {
		return err
	}
	for _, id := range coldIDs {
		if err := runCold(ladderPath, resultsRoot, id); err != nil {
			return err
		}
	}

	nextOut, err := ladderbenchOutput("next-run", "--scoring", "--ladder", ladderPath, "--results-root", resultsRoot)
	if err != nil {
		return fmt.Errorf("next-run --scoring: %w", err)
	}
	if strings.Contains(nextOut, "nothing pending") {
		fmt.Println("== scoring: nothing pending, skipping the scoring session ==")
	} else if err := runScoring(ladderPath, resultsRoot); err != nil {
		return err
	}

	if len(coldIDs) > 0 {
		fmt.Println("== cold-cell: finalising ==")
		if err := runVisible(binPath, "cold-cell", "--ladder", ladderPath, "--results-root", resultsRoot); err != nil {
			return fmt.Errorf("cold-cell finalise: %w", err)
		}
	}

	fmt.Println("== summarize ==")
	summarizeErr := runVisible(binPath, "summarize", "--ladder", ladderPath, "--results-root", resultsRoot)

	// Written again now that prepare-session has built quarry-mcp, so the binary's own embedded VCS
	// stamp is on record next to the checkout's.
	prov.stampServerBinary()
	prov.ServerHashes = serverHashes
	if err := prov.write(resultsRoot); err != nil {
		return err
	}

	if err := printSummaryTable(os.Stdout, l, resultsRoot); err != nil {
		fmt.Fprintln(os.Stderr, "runmatrix: summary table:", err)
	}
	if summarizeErr != nil {
		return fmt.Errorf("summarize: %w", summarizeErr)
	}
	fmt.Printf("== done: %s/summary.json and provenance.json written ==\n", resultsRoot)
	return nil
}

// provenance is <results-root>/provenance.json: what exactly produced this results root. It exists
// because the 2026-09-01 follow-up run could not answer "was the fixed server actually under test?"
// from its committed artifacts -- nothing recorded the built quarry-mcp's own revision.
type provenance struct {
	WrittenAt          string `json:"written_at"`
	LadderFile         string `json:"ladder_file"`
	QuarryCommit       string `json:"quarry_commit"`
	QuarryDirty        bool   `json:"quarry_dirty"`
	QuarryDirtyFiles   string `json:"quarry_dirty_files,omitempty"`
	LoomyardRepo       string `json:"loomyard_repo"`
	LoomyardCommit     string `json:"loomyard_commit"`
	ServerBinary       string `json:"server_binary"`
	ServerVCSRevision  string `json:"server_vcs_revision,omitempty"`
	ServerVCSModified  string `json:"server_vcs_modified,omitempty"`
	ServerVCSTime      string `json:"server_vcs_time,omitempty"`
	ServerBinaryMtime  string `json:"server_binary_mtime,omitempty"`
	ServerStampMissing bool   `json:"server_stamp_missing,omitempty"`
	Hostname           string `json:"hostname"`
	GoVersion          string `json:"go_version"`
	// ServerHashes is the sha256 of the quarry-mcp binary as built by prepare-session for each
	// repetition, keyed "<config-id>/<rep>". The server is rebuilt from the working tree before every
	// repetition, so an edit to quarry source while a matrix runs changes the thing under test
	// mid-matrix; more than one distinct hash here means exactly that happened (2026-09-02 toc ladder).
	ServerHashes map[string]string `json:"server_hashes,omitempty"`
}

// serverHashes is filled by run() as repetitions are prepared and merged into provenance.json at the
// end; runAll seeds it from an existing provenance.json so a resumed root keeps earlier reps' hashes.
var serverHashes = map[string]string{}

// recordServerHash hashes the freshly built server binary for configID/rep into serverHashes.
func recordServerHash(configID, rep string) {
	data, err := os.ReadFile(filepath.Join(repoRoot, "quarry-mcp"))
	if err != nil {
		return
	}
	sum := sha256.Sum256(data)
	serverHashes[configID+"/"+rep] = hex.EncodeToString(sum[:])
}

// distinctServerHashes returns the distinct hash values in hashes, sorted.
func distinctServerHashes(hashes map[string]string) []string {
	seen := map[string]bool{}
	var out []string
	for _, h := range hashes {
		if !seen[h] {
			seen[h] = true
			out = append(out, h)
		}
	}
	sort.Strings(out)
	return out
}

// loadServerHashes reads server_hashes from an existing provenance.json under resultsRoot, if any.
func loadServerHashes(resultsRoot string) map[string]string {
	data, err := os.ReadFile(filepath.Join(resultsRoot, "provenance.json"))
	if err != nil {
		return map[string]string{}
	}
	var p provenance
	if err := json.Unmarshal(data, &p); err != nil || p.ServerHashes == nil {
		return map[string]string{}
	}
	return p.ServerHashes
}

func newProvenance(ladderPath, resultsRoot string) *provenance {
	p := &provenance{
		LadderFile:   ladderPath,
		QuarryCommit: gitOutput(repoRoot, "rev-parse", "HEAD"),
		LoomyardRepo: os.Getenv("LADDER_LOOMYARD_REPO"),
		ServerBinary: filepath.Join(repoRoot, "quarry-mcp"),
		GoVersion:    strings.TrimSpace(commandOutput("go", "version")),
	}
	dirty := gitOutput(repoRoot, "status", "--porcelain")
	p.QuarryDirty = dirty != ""
	p.QuarryDirtyFiles = dirty
	if p.LoomyardRepo != "" {
		p.LoomyardCommit = gitOutput(p.LoomyardRepo, "rev-parse", "HEAD")
	}
	p.Hostname, _ = os.Hostname()
	return p
}

// stampServerBinary reads the embedded VCS stamp out of the built quarry-mcp via `go version -m`.
func (p *provenance) stampServerBinary() {
	info, err := os.Stat(p.ServerBinary)
	if err != nil {
		p.ServerStampMissing = true
		return
	}
	p.ServerBinaryMtime = info.ModTime().Format(time.RFC3339)
	out := commandOutput("go", "version", "-m", p.ServerBinary)
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[0] != "build" {
			continue
		}
		key, value, ok := strings.Cut(fields[1], "=")
		if !ok {
			continue
		}
		switch key {
		case "vcs.revision":
			p.ServerVCSRevision = value
		case "vcs.modified":
			p.ServerVCSModified = value
		case "vcs.time":
			p.ServerVCSTime = value
		}
	}
	p.ServerStampMissing = p.ServerVCSRevision == ""
}

func (p *provenance) write(resultsRoot string) error {
	p.WrittenAt = time.Now().Format(time.RFC3339)
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal provenance: %w", err)
	}
	path := filepath.Join(resultsRoot, "provenance.json")
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func gitOutput(dir string, args ...string) string {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func commandOutput(name string, args ...string) string {
	out, err := exec.Command(name, args...).Output()
	if err != nil {
		return ""
	}
	return string(out)
}

// tableMetrics is the per-cell column set printSummaryTable shows, in order: the cost-shaped metrics the
// ladder is about, bracketed by the one tell that says whether a tool-granted cell measured anything at
// all (quarry_tool_uses) and the two correctness metrics.
var tableMetrics = []string{
	"quarry_tool_uses", "num_turns", "tool_uses", "cache_read_input_tokens", "cache_creation_input_tokens",
	"output_tokens", "duration_ms", "recall", "precision",
}

// printSummaryTable prints one row per cell of <results-root>/summary.json as median[min-max] per
// metric, and flags every tool-granted cell whose agent never called a granted tool -- such a cell is a
// control with an extra schema in its prompt, not a measurement of the tool, and the 2026-09-01
// follow-up's b1-symbol cell was exactly that without anyone noticing.
// serverChangedWarning returns the line printSummaryTable appends when provenance.json under
// resultsRoot records more than one distinct server binary, or "" when it does not.
func serverChangedWarning(resultsRoot string) string {
	hashes := loadServerHashes(resultsRoot)
	distinct := distinctServerHashes(hashes)
	if len(distinct) <= 1 {
		return ""
	}
	byHash := map[string][]string{}
	for key, h := range hashes {
		byHash[h] = append(byHash[h], key)
	}
	var groups []string
	for _, h := range distinct {
		keys := byHash[h]
		sort.Strings(keys)
		groups = append(groups, fmt.Sprintf("%s: %s", h[:12], strings.Join(keys, ",")))
	}
	return fmt.Sprintf("!! the quarry-mcp binary changed during this root: %d distinct builds (quarry source was edited while the matrix ran). Reps per build -- %s", len(distinct), strings.Join(groups, " | "))
}

func printSummaryTable(w io.Writer, l *ladder.Ladder, resultsRoot string) error {
	if warning := serverChangedWarning(resultsRoot); warning != "" {
		fmt.Fprintln(w, warning)
	}
	data, err := os.ReadFile(filepath.Join(resultsRoot, "summary.json"))
	if err != nil {
		return err
	}
	var summary ladder.Summary
	if err := json.Unmarshal(data, &summary); err != nil {
		return fmt.Errorf("parse summary.json: %w", err)
	}

	ids := make([]string, 0, len(summary.Cells))
	for id := range summary.Cells {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	fmt.Fprintln(w)
	fmt.Fprintf(w, "%-24s", "cell")
	for _, m := range tableMetrics {
		fmt.Fprintf(w, " %24s", m)
	}
	fmt.Fprintln(w)

	var unused []string
	for _, id := range ids {
		cell := summary.Cells[id]
		fmt.Fprintf(w, "%-24s", id)
		for _, m := range tableMetrics {
			st, ok := cell.Stats[m]
			if !ok {
				fmt.Fprintf(w, " %24s", "-")
				continue
			}
			fmt.Fprintf(w, " %24s", fmt.Sprintf("%s[%s-%s]", fmtNum(st.Median), fmtNum(st.Min), fmtNum(st.Max)))
		}
		if !cell.Complete {
			fmt.Fprint(w, "  INCOMPLETE")
		}
		fmt.Fprintln(w)

		if cfg, err := ladder.ConfigByID(l, id); err == nil && len(cfg.Allowed) > 0 {
			if st, ok := cell.Stats["quarry_tool_uses"]; ok && st.Max == 0 {
				unused = append(unused, id)
			}
		}
	}
	fmt.Fprintln(w)
	for _, id := range unused {
		fmt.Fprintf(w, "!! %s: tool-granted config whose agent never called a granted tool in any repetition -- this cell measures the tool's prompt cost, not the tool\n", id)
	}
	if len(summary.Incomplete) > 0 {
		fmt.Fprintf(w, "!! incomplete cells: %s\n", strings.Join(summary.Incomplete, ", "))
	}
	return nil
}

func fmtNum(v float64) string {
	if v == float64(int64(v)) {
		return fmt.Sprintf("%d", int64(v))
	}
	return fmt.Sprintf("%.2f", v)
}
