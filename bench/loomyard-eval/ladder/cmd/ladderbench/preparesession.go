// preparesession.go implements ladderbench's prepare-session subcommand: the CLI-level caller that
// turns internal/ladder's PrepareRunSession/PrepareScoringSession/PrepareProbeSession into a launchable
// session. It is the one place in this batch that owns the server path, the target worktree, and the
// session lock those three internal/ladder functions all take as parameters rather than resolve
// themselves -- and, for the cold config, the extra worktree-lifecycle and gate-abort steps a cold
// session's own preparation needs on top of a warm one's.

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Knatte18/quarry/bench/loomyard-eval/ladder/internal/ladder"
)

// prepareSessionMode names the four mutually exclusive session kinds prepare-session materialises.
type prepareSessionMode int

const (
	// modeRun materialises one main-matrix run session for --config-id's n-th repetition.
	modeRun prepareSessionMode = iota
	// modeScoring materialises the single shared scoring session.
	modeScoring
	// modeProbe materialises one of the two permission probe sessions, named by --probe.
	modeProbe
	// modeRelease clears the cross-session lock and does nothing else.
	modeRelease
)

// prepareSessionFlags is prepare-session's parsed, not-yet-executed flag set, factored out so
// resolvePrepareSessionMode is testable against flag combinations directly, without exercising the
// worktree/build/session seams a full run touches.
type prepareSessionFlags struct {
	configID string
	rep      int
	repGiven bool
	scoring  bool
	probe    string
	release  bool
	runModel string
}

// resolvePrepareSessionMode validates f and returns the selected mode: an error naming the conflict when
// two or more of --scoring/--probe/--release are given together, an error when --probe names neither
// ladder.ProbeKindAllowlist nor ladder.ProbeKindDenylist, and -- for the default run-session mode -- an
// error when --rep or --config-id is missing, since a repetition is required for every run session of
// every config.
func resolvePrepareSessionMode(f prepareSessionFlags) (prepareSessionMode, error) {
	exclusiveCount := 0
	if f.scoring {
		exclusiveCount++
	}
	if f.probe != "" {
		exclusiveCount++
	}
	if f.release {
		exclusiveCount++
	}
	if exclusiveCount > 1 {
		return 0, fmt.Errorf("prepare-session: --scoring, --probe, and --release are mutually exclusive")
	}

	if f.release {
		return modeRelease, nil
	}
	if f.scoring {
		return modeScoring, nil
	}
	if f.probe != "" {
		if f.probe != ladder.ProbeKindAllowlist && f.probe != ladder.ProbeKindDenylist {
			return 0, fmt.Errorf("prepare-session: --probe must be %q or %q, got %q", ladder.ProbeKindAllowlist, ladder.ProbeKindDenylist, f.probe)
		}
		return modeProbe, nil
	}

	if !f.repGiven {
		return 0, fmt.Errorf("prepare-session: --rep is required for a run session")
	}
	if f.configID == "" {
		return 0, fmt.Errorf("prepare-session: --config-id is required for a run session")
	}
	return modeRun, nil
}

// resolveRunModel applies runModelOverride to a shallow copy of l -- never l itself, so the override is
// never written back to the ladder file -- and validates the result against ladder.RequireSessionPins,
// returning the *ladder.Ladder every downstream call in this command should use.
//
// A run session's own pin check must accept an override that satisfies it -- that is the override's
// entire purpose, kept for the follow-up matrix task's probe dispatches and first real runs, which need
// a way to run against the committed ladder.yaml before the operator writes run_model's pin into it.
func resolveRunModel(l *ladder.Ladder, override string) (*ladder.Ladder, error) {
	effective := l
	if override != "" {
		clone := *l
		model := override
		clone.RunModel = &model
		effective = &clone
	}
	if err := ladder.RequireSessionPins(effective, override); err != nil {
		return nil, err
	}
	return effective, nil
}

// mcpConfigFilename is the fixed name Claude Code's own project-scope MCP discovery reads a server
// declaration from, matching internal/ladder/session.go's own unexported serverDeclarationFilename.
// prepare-session's probe path writes to this same literal directly, since PrepareProbeSession itself
// deliberately writes no .mcp.json content -- see its doc comment.
const mcpConfigFilename = ".mcp.json"

// installedSkillSourcePath returns the tracked orchestration skill's repo-relative path, resolved
// against repoRoot.
func installedSkillSourcePath(repoRoot string) string {
	return filepath.Join(repoRoot, ".claude", "skills", "ladder-run", "SKILL.md")
}

// installedSkillDestRoot returns the Claude Code user-scope skills root every session type installs the
// tracked orchestration skill under: ~/.claude/skills. InstallSkill never writes into a scratch
// directory (see its own doc comment), so this destination is the same for every session prepared.
func installedSkillDestRoot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("prepare-session: resolve home directory for skill install: %w", err)
	}
	return filepath.Join(home, ".claude", "skills"), nil
}

// installOrchestrationSkill installs the tracked orchestration skill for any session type prepared
// against repoRoot.
func installOrchestrationSkill(repoRoot string) error {
	destRoot, err := installedSkillDestRoot()
	if err != nil {
		return err
	}
	if _, err := ladder.InstallSkill(installedSkillSourcePath(repoRoot), destRoot); err != nil {
		return err
	}
	return nil
}

// printLaunchInfo prints inputs.ScratchDir as its own "scratch_dir: " line, then the launch command
// ladder.LaunchCommand(inputs) prints -- the single print site every prepare-session path uses, so a
// caller (a driver script automating many sessions in sequence, in particular) can always find the
// scratch directory on its own line regardless of whether the launch command itself carries a
// --mcp-config flag naming it (it never does for a blinded config, which has no server declaration at
// all).
func printLaunchInfo(out io.Writer, inputs ladder.SessionInputs) {
	fmt.Fprintf(out, "scratch_dir: %s\n", inputs.ScratchDir)
	fmt.Fprintln(out, ladder.LaunchCommand(inputs))
}

// sessionLabel names the cross-session lock label a run, scoring, or probe session takes --
// "<kind>:<id>:<n>" -- so a second, conflicting session attempt is refused with a label that already
// names what is holding it.
func sessionLabel(kind, id string, n int) string {
	return fmt.Sprintf("%s:%s:%d", kind, id, n)
}

// writeMCPConfigDocument marshals ladder.MCPConfigDocument(serverPath, targetDir) as indented JSON with
// a trailing newline to path, creating path's parent directory first. Used by the probe path only, which
// is the one caller in this batch that owns a server path and target directory but calls a
// PrepareXSession function (PrepareProbeSession) that writes no .mcp.json content of its own.
func writeMCPConfigDocument(path, serverPath, targetDir string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("prepare-session: create %s: %w", filepath.Dir(path), err)
	}
	data, err := json.MarshalIndent(ladder.MCPConfigDocument(serverPath, targetDir), "", "  ")
	if err != nil {
		return fmt.Errorf("prepare-session: marshal mcp config document: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("prepare-session: write %s: %w", path, err)
	}
	return nil
}

// firstTaskWorktree returns the worktree path for l.Tasks's alphabetically-first task key, resolved
// through worktrees -- never through map iteration order, which Go leaves undefined. The probe path uses
// this as its single arbitrary target directory, mirroring scripts/run_ladder.py's own
// worktrees[next(iter(ladder.tasks))] -- an arbitrary but deterministic choice, since a permission probe
// establishes an enforcement-layer fact independent of which task worktree it runs against.
func firstTaskWorktree(l *ladder.Ladder, worktrees map[string]string) (string, error) {
	if len(l.Tasks) == 0 {
		return "", fmt.Errorf("prepare-session: ladder.yaml declares no tasks")
	}
	keys := make([]string, 0, len(l.Tasks))
	for key := range l.Tasks {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	dir, ok := worktrees[keys[0]]
	if !ok {
		return "", fmt.Errorf("prepare-session: no worktree resolved for task %q", keys[0])
	}
	return dir, nil
}

func prepareSessionCommand() *cobra.Command {
	var configID string
	var rep int
	var scoring bool
	var probeKind string
	var release bool
	var runModelOverride string

	cmd := &cobra.Command{
		Use:   "prepare-session",
		Short: "materialise one session's scratch directory and print its launch command",
		Long: `prepare-session materialises the scratch directory a session launches from -- a
main-matrix run (--config-id and --rep), the single shared scoring session (--scoring), or one of the
two permission probes (--probe allowlist|denylist) -- and prints the launch command the operator runs.
--release clears this results root's cross-session lock and does nothing else; it is mutually exclusive
with --scoring and --probe.

--run-model overrides the ladder's pinned run model for this invocation only -- never written back to
ladder.yaml. The discussion scoped this flag to a smoke launch this plan does not perform, so it ships
with no caller in this batch: it is kept for the follow-up matrix task, whose two probe dispatches and
first real runs need a way to run against the committed ladder.yaml before the operator writes the pin
into it.

prepare-session enforces the narrower session pin set (ladder.RequireSessionPins) rather than the full
one (ladder.RequirePins), on every path including the scoring one -- resolving a conflict inside the
discussion, which says in one place that the scoring path calls the full check and in another that this
command enforces the narrower set: the narrower set already covers the scorer model and effort a scoring
session stamps, and the full check additionally demands the turn ceiling, which ships blank and which
nothing about preparing a session touches.

The run-session path additionally runs the environment precondition against the operator's own shell
environment (never the scrubbed environment run and warm-up dispatch use, which forces both keys empty
and would make the check pass unconditionally) and the skill-listing leak scan, hard-failing only for a
config whose allowed set is empty.

For the cold config, the run-session path also drains the daemons the warm sessions left resident on
both task worktrees, builds a fresh per-repetition worktree off the cold worktree template, clears its
resolved state directory, and asserts the cold-before gate before preparing the session. Unlike a warm
run session, a cold one never takes the cross-session lock: a failed cold attempt relaunches the whole
session rather than retrying in place (the session's server process outlives an attempt, and
re-clearing the state directory mid-session would delete the state file whose pid the cold-before gate
reads, leaving a surviving warm daemon invisible), so on a cold-before gate failure this instead records
a cold_abort.json and invalidates the repetition's run directory, bounding repeated aborts the same way
a failed run's own retries are bounded.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			f := prepareSessionFlags{
				configID: configID,
				rep:      rep,
				repGiven: cmd.Flags().Changed("rep"),
				scoring:  scoring,
				probe:    probeKind,
				release:  release,
				runModel: runModelOverride,
			}
			mode, err := resolvePrepareSessionMode(f)
			if err != nil {
				return err
			}

			resultsRoot, err := resolveResultsRoot(cmd)
			if err != nil {
				return err
			}
			if mode == modeRelease {
				return ladder.ReleaseSessionLock(resultsRoot)
			}

			l, err := resolveLadder(cmd)
			if err != nil {
				return err
			}
			effective, err := resolveRunModel(l, f.runModel)
			if err != nil {
				return err
			}

			repoRoot, err := resolveRepoRoot()
			if err != nil {
				return err
			}

			switch mode {
			case modeScoring:
				return runPrepareScoringSession(cmd, effective, resultsRoot, repoRoot)
			case modeProbe:
				return runPrepareProbeSession(cmd, effective, resultsRoot, repoRoot, f.probe)
			default:
				return runPrepareRunSession(cmd, effective, resultsRoot, repoRoot, f.configID, f.rep)
			}
		},
	}

	cmd.Flags().StringVar(&configID, "config-id", "", "the LadderConfig id to prepare a run session for (required for a run session)")
	cmd.Flags().IntVar(&rep, "rep", 0, "1-based repetition index (required for every run session of every config)")
	cmd.Flags().BoolVar(&scoring, "scoring", false, "prepare the single shared scoring session instead of a run session")
	cmd.Flags().StringVar(&probeKind, "probe", "", fmt.Sprintf("prepare a permission probe session (%q or %q) instead of a run session", ladder.ProbeKindAllowlist, ladder.ProbeKindDenylist))
	cmd.Flags().BoolVar(&release, "release", false, "clear this results root's cross-session lock and do nothing else")
	cmd.Flags().StringVar(&runModelOverride, "run-model", "", "override the pinned run model for this invocation only; never written to ladder.yaml -- see the Long help for why this flag ships with no caller in this batch")

	return cmd
}

// runPrepareScoringSession materialises the shared scoring session, taking the cross-session lock like
// any other session -- scoring never touches the target codebase, so it needs no worktree and no server
// build.
func runPrepareScoringSession(cmd *cobra.Command, l *ladder.Ladder, resultsRoot, repoRoot string) error {
	if err := ladder.AcquireSessionLock(resultsRoot, sessionLabel("scoring", "scoring", 1)); err != nil {
		return err
	}

	inputs, err := ladder.PrepareScoringSession(l, repoRoot)
	if err != nil {
		return err
	}
	if err := installOrchestrationSkill(repoRoot); err != nil {
		return err
	}
	printLaunchInfo(cmd.OutOrStdout(), inputs)
	return nil
}

// runPrepareProbeSession materialises one of the two permission probe sessions, taking the cross-session
// lock like any other session. It ensures the task worktrees are at their pins, builds the server
// binary, and -- since PrepareProbeSession deliberately writes no .mcp.json content itself -- writes the
// probe's own server declaration, pointed at the deterministic first task worktree (see
// firstTaskWorktree).
func runPrepareProbeSession(cmd *cobra.Command, l *ladder.Ladder, resultsRoot, repoRoot, kind string) error {
	if err := ladder.AcquireSessionLock(resultsRoot, sessionLabel("probe", kind, 1)); err != nil {
		return err
	}

	sourceRepo, err := ladder.ResolveSourceRepo(l)
	if err != nil {
		return err
	}
	worktrees, err := ladder.EnsureTaskWorktrees(l, sourceRepo, ladder.RunGit)
	if err != nil {
		return err
	}
	targetDir, err := firstTaskWorktree(l, worktrees)
	if err != nil {
		return err
	}

	serverPath, err := ladder.BuildServer(repoRoot, realBuild)
	if err != nil {
		return err
	}

	inputs, err := ladder.PrepareProbeSession(l, repoRoot, kind)
	if err != nil {
		return err
	}
	if err := writeMCPConfigDocument(filepath.Join(inputs.ScratchDir, mcpConfigFilename), serverPath, targetDir); err != nil {
		return err
	}

	if err := installOrchestrationSkill(repoRoot); err != nil {
		return err
	}
	printLaunchInfo(cmd.OutOrStdout(), inputs)
	return nil
}

// selectsColdPath reports whether config routes prepare-session's run-session path through
// runPrepareColdSession, keyed on config.Cold rather than parsed from config.ID -- so an id merely
// containing "cold" never misroutes, and a5-bundle-cold routes there only because its own Cold field is
// true.
func selectsColdPath(config ladder.LadderConfig) bool {
	return config.Cold
}

// runPrepareRunSession materialises one main-matrix run session for config-id's n-th repetition: the
// environment precondition and skill-leak scan first, then either the cold path (see
// runPrepareColdSession) or the warm path -- ensuring the task worktrees, building the server, taking
// the lock, and materialising the session.
func runPrepareRunSession(cmd *cobra.Command, l *ladder.Ladder, resultsRoot, repoRoot, configID string, rep int) error {
	config, err := ladder.ConfigByID(l, configID)
	if err != nil {
		return err
	}

	if err := ladder.CheckEnvironmentPrecondition(os.Environ()); err != nil {
		return err
	}
	if len(config.Allowed) == 0 {
		_, offenders, err := ladder.ScanSkillsForLeak(ladder.DefaultSkillRoots())
		if err != nil {
			return err
		}
		if len(offenders) > 0 {
			return fmt.Errorf("prepare-session: config %q exposes no quarry tools but the skill-leak scan found: %s", config.ID, strings.Join(offenders, ", "))
		}
	}

	sourceRepo, err := ladder.ResolveSourceRepo(l)
	if err != nil {
		return err
	}

	if selectsColdPath(config) {
		return runPrepareColdSession(cmd, l, config, rep, resultsRoot, repoRoot, sourceRepo, ladder.RunGit, realBuild)
	}

	worktrees, err := ladder.EnsureTaskWorktrees(l, sourceRepo, ladder.RunGit)
	if err != nil {
		return err
	}
	targetDir, ok := worktrees[config.Task]
	if !ok {
		return fmt.Errorf("prepare-session: config %q references unknown task %q", config.ID, config.Task)
	}

	serverPath, err := ladder.BuildServer(repoRoot, realBuild)
	if err != nil {
		return err
	}

	if err := ladder.AcquireSessionLock(resultsRoot, sessionLabel("run", config.ID, rep)); err != nil {
		return err
	}

	inputs, err := ladder.PrepareRunSession(l, config, rep, repoRoot, serverPath, targetDir)
	if err != nil {
		return err
	}
	if err := installOrchestrationSkill(repoRoot); err != nil {
		return err
	}
	printLaunchInfo(cmd.OutOrStdout(), inputs)
	return nil
}

// coldDaemonLang mirrors internal/ladder's own unexported daemonLang -- the suite only ever supervises
// the Go language server, so this is the fixed lang segment every daemon-state read in the cold path
// resolves against.
const coldDaemonLang = "go"

// coldAbortFilename is cold_abort.json's fixed name, read nowhere else by any other name: the cold-cell
// disposition (cli-run-commands, out of scope here) is the only other reader of this file.
const coldAbortFilename = "cold_abort.json"

// coldAbortCauseLiveDaemon is the sole cause token cold_abort.json ever records: GateColdBefore's only
// failure mode is a daemon already alive at the worktree's state directory, mirroring
// scripts/run_ladder.py's own rep_not_run_cause == "live_daemon_before_start" literal -- the token the
// cold-cell disposition (a later batch) matches on.
const coldAbortCauseLiveDaemon = "live_daemon_before_start"

// coldAbortRecord is cold_abort.json's fixed schema, written once per aborted cold attempt: it is the
// only on-disk source for the cold cell's live-daemon not-run cause, which scripts/run_ladder.py held
// only in the driver's own memory.
type coldAbortRecord struct {
	ConfigID string `json:"config_id"`
	Rep      int    `json:"rep"`
	Attempt  int    `json:"attempt"`
	Cause    string `json:"cause"`
}

// coldWorktreeDir substitutes "{n}" with rep in l.ColdWorktreeTemplate, mirroring plan.go's own
// SessionDir substitution for the per-repetition cold worktree path.
func coldWorktreeDir(l *ladder.Ladder, rep int) string {
	return strings.ReplaceAll(l.ColdWorktreeTemplate, "{n}", strconv.Itoa(rep))
}

// runPrepareColdSession extends the run-session path for config.Cold: it drains the daemons the warm
// sessions left resident on both task worktrees using the bounded exit wait at its committed timeout,
// builds a fresh per-repetition worktree off the cold worktree template, clears its resolved state
// directory, and asserts the cold-before gate before preparing the session -- pointing that session's
// server declaration at the freshly built worktree via prepareColdSessionAfterGate.
func runPrepareColdSession(cmd *cobra.Command, l *ladder.Ladder, config ladder.LadderConfig, rep int, resultsRoot, repoRoot, sourceRepo string, git ladder.GitRunner, build ladder.Builder) error {
	cacheDir, err := ladder.UserCacheDir()
	if err != nil {
		return err
	}
	env := ladder.ScrubbedEnv()

	for _, task := range l.Tasks {
		if err := ladder.WaitForDaemonExit(task.Worktree, cacheDir, env, ladder.DaemonExitTimeout, coldDaemonLang); err != nil {
			return err
		}
	}

	task, ok := l.Tasks[config.Task]
	if !ok {
		return fmt.Errorf("prepare-session: cold config %q references unknown task %q", config.ID, config.Task)
	}
	targetDir := coldWorktreeDir(l, rep)
	if err := ladder.BuildWorktree(sourceRepo, targetDir, task.PinnedSHA, git); err != nil {
		return err
	}
	if err := ladder.ClearStateDir(targetDir, cacheDir, env); err != nil {
		return err
	}

	gateFindings := ladder.GateColdBefore(targetDir, cacheDir, env)
	return prepareColdSessionAfterGate(cmd, l, config, rep, resultsRoot, repoRoot, targetDir, gateFindings, build)
}

// prepareColdSessionAfterGate continues cold-session preparation once the cold-before gate has already
// been evaluated -- gateFindings is that gate's own return value -- so the gate-failure and gate-pass
// branches are each testable without needing a live daemon process to produce a genuine gate failure.
//
// The lock is never taken on this path, gate-pass or gate-fail: unlike a warm run session,
// runPrepareColdSession/prepareColdSessionAfterGate call ladder.AcquireSessionLock nowhere -- see
// prepare-session's own Long help for why a cold session's per-attempt worktree lifecycle makes the
// cross-session lock the wrong guard for it.
func prepareColdSessionAfterGate(cmd *cobra.Command, l *ladder.Ladder, config ladder.LadderConfig, rep int, resultsRoot, repoRoot, targetDir string, gateFindings []ladder.GateFinding, build ladder.Builder) error {
	if len(gateFindings) > 0 {
		return abortColdAttempt(resultsRoot, config.ID, rep)
	}

	serverPath, err := ladder.BuildServer(repoRoot, build)
	if err != nil {
		return err
	}

	inputs, err := ladder.PrepareRunSession(l, config, rep, repoRoot, serverPath, targetDir)
	if err != nil {
		return err
	}
	if err := installOrchestrationSkill(repoRoot); err != nil {
		return err
	}
	printLaunchInfo(cmd.OutOrStdout(), inputs)
	return nil
}

// abortColdAttempt records this cold attempt's live-daemon abort by creating the repetition's run
// directory (ladder.RunDirPath), writing cold_abort.json inside it, and calling ladder.Invalidate -- the
// same invalidation the run session uses -- which renames the run directory aside to its own
// <n>.invalid-<k> sibling, bounding repeated aborts the same way a failed run's own retries are bounded.
// No scratch directory is written on this path, and no lock is taken (see prepareColdSessionAfterGate).
func abortColdAttempt(resultsRoot, configID string, rep int) error {
	attempt, err := ladder.NextAttempt(resultsRoot, configID, rep)
	if err != nil {
		return fmt.Errorf("prepare-session: cold config %q rep %d: %w", configID, rep, err)
	}

	runDir := ladder.RunDirPath(resultsRoot, configID, rep)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return fmt.Errorf("prepare-session: cold abort: create %s: %w", runDir, err)
	}
	record := coldAbortRecord{ConfigID: configID, Rep: rep, Attempt: attempt, Cause: coldAbortCauseLiveDaemon}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return fmt.Errorf("prepare-session: cold abort: marshal %s: %w", coldAbortFilename, err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(runDir, coldAbortFilename), data, 0o644); err != nil {
		return fmt.Errorf("prepare-session: cold abort: write %s in %s: %w", coldAbortFilename, runDir, err)
	}

	next, err := ladder.Invalidate(runDir)
	if err != nil {
		return fmt.Errorf("prepare-session: cold config %q rep %d exhausted ladder.MaxAttempts: %w", configID, rep, err)
	}
	return fmt.Errorf(
		"prepare-session: cold config %q rep %d found a live daemon before this attempt started; aborted as attempt %d, next attempt is %d",
		configID, rep, attempt, next,
	)
}
