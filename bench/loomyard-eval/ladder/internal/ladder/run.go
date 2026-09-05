// run.go drives the sequential, cell-minor loop that turns a loaded ladder file into a filled
// results root: it prepares each cell's pinned worktree, renders its prompt, invokes the measured
// claude process through the injectable runner seam, tees its stream to disk, computes metrics,
// applies the pre- and post-dispatch blinding gates, dispatches the scorer, and writes the six
// per-repetition files runstate.go names, with the state file last. It also runs the scorer as a
// second, separate measured-binary invocation via RunScorer.
//
// The failure taxonomy is exactly five outcomes -- infrastructure failure, formatting miss,
// max-turns completion, fatal gate-2 finding, and scorer failure -- and each has its own
// disposition; see this file's per-outcome comments rather than the overview for the exact rule.

package ladder

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// maxTurnsTerminalReason is the result record's terminal_reason value when a repetition ended at its
// turn ceiling. A repetition carrying this reason is complete, not a failure: see run.go's header
// comment.
const maxTurnsTerminalReason = "max_turns"

// RunOptions carries everything Run needs to drive one invocation: the operator's own ladder file
// path and results root, the cells selected for this invocation (empty selects every cell the file
// declares), an optional repetition-count override (zero selects the file's own reps), the claude
// binary path, a starting point Run resolves up to the quarry repository root from, and the Runner
// every external process goes through.
type RunOptions struct {
	// LadderFilePath is the operator's own ladder file path.
	LadderFilePath string
	// ResultsRoot is the results root this invocation reads from and writes to.
	ResultsRoot string
	// SelectedCells is the cell ids this invocation runs. Empty selects every config the ladder file
	// declares.
	SelectedCells []string
	// RepsOverride is the per-cell repetition count this invocation uses instead of the ladder
	// file's own. Zero means "not given" -- the ladder file's reps field is used instead.
	RepsOverride int
	// ClaudeBinPath is the claude binary path, both the measured and the scorer invocations run.
	ClaudeBinPath string
	// QuarryRepoStart is a path inside the quarry repository Run resolves up from to the
	// repository's own root via ResolveQuarryRepoRoot.
	QuarryRepoStart string
	// Runner is the seam every external process -- claude, git, go build -- runs through.
	Runner Runner
}

// Run drives one invocation of the sequential, cell-minor loop over every selected cell and its
// repetitions, resuming a results root that already carries complete repetitions and reporting a
// non-zero exit signal when any cell ends incomplete or any repetition's blinding-failed flag is
// set. See this file's header comment for the write order and the failure taxonomy.
func Run(ctx context.Context, opts RunOptions) (exitNonZero bool, err error) {
	l, err := LoadLadder(opts.LadderFilePath)
	if err != nil {
		return false, err
	}

	selectedConfigs, err := resolveSelectedCells(l, opts.SelectedCells)
	if err != nil {
		return false, err
	}

	repsEffective := l.Reps
	if opts.RepsOverride > 0 {
		repsEffective = opts.RepsOverride
	}

	quarryRepoRoot, err := ResolveQuarryRepoRoot(opts.QuarryRepoStart)
	if err != nil {
		return false, err
	}
	targetRepoPath, err := ResolveLoomyardRepo(quarryRepoRoot)
	if err != nil {
		return false, err
	}
	worktreeRoot, err := ResolveWorktreeRoot(quarryRepoRoot)
	if err != nil {
		return false, err
	}

	release, err := AcquireRunLock(worktreeRoot, opts.ResultsRoot)
	if err != nil {
		return false, err
	}
	defer func() { _ = release() }()

	existing, err := ReadProvenance(opts.ResultsRoot)
	if err != nil {
		return false, err
	}
	if existing != nil && existing.RepsEffective != repsEffective {
		return false, fmt.Errorf(
			"run: results root %s was written with reps_effective %d, this invocation requests %d",
			opts.ResultsRoot, existing.RepsEffective, repsEffective,
		)
	}

	selectedIDs := make([]string, len(selectedConfigs))
	for i, c := range selectedConfigs {
		selectedIDs[i] = c.ID
	}

	inv, err := CollectInvocation(ctx, opts.Runner, CollectInput{
		QuarryRepoRoot: quarryRepoRoot,
		LadderFilePath: opts.LadderFilePath,
		TargetRepoPath: targetRepoPath,
		ServerName:     l.ServerName(),
		SelectedCells:  selectedIDs,
		RepsEffective:  repsEffective,
		ClaudeBinPath:  opts.ClaudeBinPath,
	})
	if err != nil {
		return false, err
	}

	prov, err := MergeProvenance(existing, inv)
	if err != nil {
		return false, err
	}

	// Writing here, immediately once the invocation is merged, is what keeps a run that dies
	// mid-matrix -- even during the memory-path scan or server build below -- leaving its own
	// invocation on disk.
	if err := WriteProvenance(opts.ResultsRoot, prov); err != nil {
		return false, err
	}

	// Verified immediately after the first provenance write and before the server-build block, so a
	// missing card or a stale pack costs one stat and one hash comparison, not an API call or a
	// server build.
	if err := verifyCardsAndPack(l, selectedConfigs, prov, quarryRepoRoot); err != nil {
		return false, err
	}

	// A resumed root whose provenance already carries memory-path hashes has its memory paths
	// scanned before the first new repetition -- a resumed run otherwise skips the very repetition
	// that would reveal them. A record carrying hashes whose paths file is missing is treated as
	// unknown, exactly like a fresh root, and re-derived from the next completed repetition.
	memoryPathsKnown := false
	if existing != nil && len(existing.MemoryPathHashes) > 0 {
		paths, ok, err := readMemoryPaths(opts.ResultsRoot)
		if err != nil {
			return false, err
		}
		if ok {
			memoryPathsKnown = true
			finding, err := ScanMemoryPaths(paths)
			if err != nil {
				return false, err
			}
			if finding != nil {
				return true, fmt.Errorf("run: %s", finding.Message)
			}
		}
	}

	// The server is built once, when and only when a selected cell grants a non-empty tool subset,
	// and its hash is recorded for every cell-and-repetition pair it will serve.
	var serverBinary string
	needsServer := false
	for _, c := range selectedConfigs {
		if c.GrantsTools() {
			needsServer = true
			break
		}
	}
	if needsServer {
		if l.Server == nil {
			return false, fmt.Errorf("run: a selected cell grants tools but the ladder file declares no server block")
		}
		serverBinary = filepath.Join(worktreeRoot, "bin", l.ServerName())
		hash, err := BuildServer(ctx, opts.Runner, quarryRepoRoot, l.Server.Build, serverBinary)
		if err != nil {
			return false, err
		}
		if prov.ServerHashes == nil {
			prov.ServerHashes = map[string]string{}
		}
		for _, c := range selectedConfigs {
			if !c.GrantsTools() {
				continue
			}
			for rep := 1; rep <= repsEffective; rep++ {
				prov.ServerHashes[repKey(c.ID, rep)] = hash
			}
		}
		// The server hash was not yet known at the first write above; persist it now so a
		// crash during the run below still leaves a provenance record naming the built server.
		if err := WriteProvenance(opts.ResultsRoot, prov); err != nil {
			return false, err
		}
	}

	exitNonZero = false
	for rep := 1; rep <= repsEffective; rep++ {
		for _, cfg := range selectedConfigs {
			task, ok := l.Tasks[cfg.Task]
			if !ok {
				return false, fmt.Errorf("run: cell %s references unknown task %q", cfg.ID, cfg.Task)
			}

			dir := RepDir(opts.ResultsRoot, cfg.ID, rep)
			if RepIsComplete(dir) {
				continue
			}

			outcome, err := runCellRepetition(ctx, opts, l, cfg, task, rep, dir, quarryRepoRoot, targetRepoPath, worktreeRoot, serverBinary, prov, &memoryPathsKnown)
			if err != nil {
				return false, err
			}
			if outcome.blindingFailed || outcome.incomplete {
				exitNonZero = true
			}

			if err := WriteProvenance(opts.ResultsRoot, prov); err != nil {
				return false, err
			}

			if outcome.abortRun {
				return exitNonZero, nil
			}
		}
	}

	return exitNonZero, nil
}

// resolveSelectedCells resolves the operator's own cell selection against l, defaulting to every
// config the file declares, and errors on an id the file does not contain.
func resolveSelectedCells(l *Ladder, selected []string) ([]Config, error) {
	ids := selected
	if len(ids) == 0 {
		ids = make([]string, len(l.Configs))
		for i, c := range l.Configs {
			ids[i] = c.ID
		}
	}
	configs := make([]Config, 0, len(ids))
	for _, id := range ids {
		cfg, ok := l.ConfigByID(id)
		if !ok {
			return nil, fmt.Errorf("run: selected cell %q is not a config in the ladder file", id)
		}
		configs = append(configs, cfg)
	}
	return configs, nil
}

// verifyCardsAndPack runs the two pre-rep-1 checks that keep a bad card or a stale pack from costing
// an API call: first, that every selectedConfigs entry declaring a non-empty card names a card file
// that actually exists under quarryRepoRoot; second, when l declares a pack cell, that the sha256 of
// that cell's card's sentinel-delimited block equals prov's own recorded kickstart_pack.pack_sha256.
// The second check is skipped entirely when l declares no pack cell, which is what keeps every
// existing ladder file and every committed results root behaving exactly as it does today.
//
// This deliberately never compares quarryRepoRoot's own commit or dirty flag against the
// kickstart_pack block's recorded QuarryCommit/QuarryDirty. Two reasons, both load-bearing: prov's
// top-level QuarryCommit and QuarryDirty are derived by MergeProvenance from the latest invocation, so
// the two sides differ after any commit between "ladder pack" and "ladder run" -- and committing the
// generated card is exactly such a commit, and the intended workflow, so a gate on it would brick the
// root for doing the right thing. And QuarryDirty is vacuously true on both sides the moment the pack
// command writes a tracked card, so a gate on it would always be satisfied and would not be a gate at
// all. What actually enforces the never-edit-the-code-under-test rule is the per-invocation record in
// prov.Invocations, which makes a mid-matrix edit visible and auditable afterwards -- not this gate.
//
// The operator workflow this gate assumes: run "ladder pack" to generate the pack, inspect its
// output, commit the generated card together with the ladder file and the task and fasit files, then
// start the matrix with "ladder run". Committing between the pack and the run is expected and is not
// a freshness violation.
func verifyCardsAndPack(l *Ladder, selectedConfigs []Config, prov *Provenance, quarryRepoRoot string) error {
	for _, cfg := range selectedConfigs {
		if cfg.Card == "" {
			continue
		}
		path := resolveRepoRelative(quarryRepoRoot, cfg.Card)
		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("verify cards and pack: cell %s: card %s: %w", cfg.ID, path, err)
		}
	}

	var packCfg *Config
	for i := range l.Configs {
		if l.Configs[i].Pack {
			packCfg = &l.Configs[i]
			break
		}
	}
	if packCfg == nil {
		return nil
	}

	if prov.KickstartPack == nil {
		return fmt.Errorf("verify cards and pack: results root %s carries no kickstart_pack record; run the pack subcommand first", prov.LadderFile)
	}

	cardPath := resolveRepoRelative(quarryRepoRoot, packCfg.Card)
	cardData, err := os.ReadFile(cardPath)
	if err != nil {
		return fmt.Errorf("verify cards and pack: read pack cell %s's card %s: %w", packCfg.ID, cardPath, err)
	}
	block, err := ExtractPackBlock(string(cardData))
	if err != nil {
		return fmt.Errorf("verify cards and pack: extract pack block from %s: %w", cardPath, err)
	}
	gotHash := PackBlockSHA256(block)
	if gotHash != prov.KickstartPack.PackSHA256 {
		return fmt.Errorf(
			"verify cards and pack: pack cell %s's card %s hashes to %s, provenance records %s -- run the pack subcommand again",
			packCfg.ID, cardPath, gotHash, prov.KickstartPack.PackSHA256,
		)
	}
	return nil
}

// repKey is the cell-and-repetition key Provenance.ServerHashes and Provenance.SessionFingerprints
// are keyed by.
func repKey(cellID string, rep int) string {
	return cellID + "/" + strconv.Itoa(rep)
}

// grantedToolNames returns cfg's fully granted tool set for prompt rendering and the CLI's own
// --tools value: the package-level BuiltinTools slice from config.go, plus each of cfg's allowed
// tools prefixed with l's MCP prefix.
func grantedToolNames(l *Ladder, cfg Config) []string {
	names := append([]string{}, BuiltinTools...)
	for _, a := range cfg.Allowed {
		names = append(names, l.MCPPrefix()+a)
	}
	return names
}

// resolveRepoRelative joins p onto quarryRepoRoot when p is not already absolute, so a ladder file's
// task_file and fasit paths -- written relative to the quarry repository -- resolve regardless of the
// harness process's own working directory.
func resolveRepoRelative(quarryRepoRoot, p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(quarryRepoRoot, p)
}

// memoryPathsFile is the untracked file, inside a results root's own raw tree, that the resolved
// auto-memory directories are written to. Only their hashes are ever written to the committed
// provenance record -- see provenance.go's header comment.
const memoryPathsFile = "memory-paths.json"

// readMemoryPaths reads resultsRoot/raw/memory-paths.json and reports whether it existed.
func readMemoryPaths(resultsRoot string) (paths []string, ok bool, err error) {
	path := filepath.Join(resultsRoot, "raw", memoryPathsFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("read memory paths %s: %w", path, err)
	}
	if err := json.Unmarshal(data, &paths); err != nil {
		return nil, false, fmt.Errorf("read memory paths %s: %w", path, err)
	}
	return paths, true, nil
}

// writeMemoryPaths writes paths to resultsRoot/raw/memory-paths.json, creating the raw tree when
// absent.
func writeMemoryPaths(resultsRoot string, paths []string) error {
	dir := filepath.Join(resultsRoot, "raw")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("write memory paths: %w", err)
	}
	data, err := json.MarshalIndent(paths, "", "  ")
	if err != nil {
		return fmt.Errorf("write memory paths: %w", err)
	}
	path := filepath.Join(dir, memoryPathsFile)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write memory paths %s: %w", path, err)
	}
	return nil
}

// repOutcome is what one cell-and-repetition pair's processing reports back to Run: whether a fatal
// gate finding discarded the repetition, and whether it was retried to exhaustion and recorded
// incomplete. Both flip Run's own exit signal; neither is mutually exclusive with the other being
// false.
type repOutcome struct {
	blindingFailed bool
	incomplete     bool
	// abortRun reports that the whole invocation must stop after this repetition, rather than
	// continuing to the remaining cells -- set only by a tainted memory-path scan, since a resumed
	// invocation must not skip past the very repetition that revealed it.
	abortRun bool
}

// runCellRepetition processes exactly one cell's one repetition through every step this file's
// header comment names, in order, and returns the disposition Run folds into its own exit signal.
// memoryPathsKnown is shared across every call in the invocation: it starts false for a fresh root or
// one whose memory-path hashes could not be read back, and flips true the first time this invocation
// derives them from a completed repetition.
func runCellRepetition(
	ctx context.Context,
	opts RunOptions,
	l *Ladder,
	cfg Config,
	task Task,
	rep int,
	dir string,
	quarryRepoRoot, targetRepoPath, worktreeRoot, serverBinary string,
	prov *Provenance,
	memoryPathsKnown *bool,
) (repOutcome, error) {
	dest := TaskWorktreePath(worktreeRoot, cfg.Task)
	if err := PrepareWorktree(ctx, opts.Runner, targetRepoPath, cfg.Task, task.PinnedSHA, dest); err != nil {
		return repOutcome{}, err
	}

	content, err := LoadTaskFile(resolveRepoRelative(quarryRepoRoot, task.TaskFile))
	if err != nil {
		return repOutcome{}, err
	}

	var card string
	if cfg.Card != "" {
		card, err = LoadCardFile(resolveRepoRelative(quarryRepoRoot, cfg.Card))
		if err != nil {
			return repOutcome{}, err
		}
	}

	toolNames := grantedToolNames(l, cfg)
	prompt := RenderPrompt(content, dest, toolNames, card)

	blindingIn := BlindingInput{
		MCPPrefix:      l.MCPPrefix(),
		ServerName:     l.ServerName(),
		QuarryRepoRoot: quarryRepoRoot,
	}

	// Check (d): a cell that grants no tools has its rendered prompt checked before dispatch, so a
	// finding costs no API call.
	if !cfg.GrantsTools() {
		if f := CheckRenderedControlPrompt(prompt, blindingIn, l.QuarryTools); f != nil {
			if err := writeVoidRepetition(dir, l, cfg, task, rep, []Finding{*f}); err != nil {
				return repOutcome{}, err
			}
			return repOutcome{blindingFailed: true}, nil
		}
	}

	mcpDoc, err := MCPConfigDocument(l, cfg, serverBinary, dest)
	if err != nil {
		return repOutcome{}, err
	}
	mcpConfigPath, err := WriteMCPConfig(quarryRepoRoot, fmt.Sprintf("%s-%d.json", cfg.ID, rep), mcpDoc)
	if err != nil {
		return repOutcome{}, err
	}

	// The measured-invocation attempt loop: an infrastructure failure, a formatting miss, or a
	// granted server that never connected renames the repetition directory away and retries, up to
	// MaxAttempts, after which the cell is recorded incomplete and the loop moves on to the next cell
	// -- unless, per the connectFailures check below, every one of those attempts failed to connect
	// the server, which aborts the whole run instead.
	var (
		transcript  *Transcript
		answerText  string
		maxTurnsHit bool
	)
	connectFailures := 0
	attempt := 0
	for {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return repOutcome{}, err
		}

		t, invokeErr := invokeMeasuredProcess(ctx, opts, l, cfg, prompt, dest, mcpConfigPath, dir)
		var serverFinding *Finding
		accepted := false
		// unparseableAnswerDetail names which of the two answer-extraction steps failed, for the
		// unparseable_answer cause below. It carries a fixed phrase, never the extraction error's
		// own text, and is meaningful only when this iteration lands in that cause.
		var unparseableAnswerDetail string
		if invokeErr == nil && t.Result != nil && !t.Result.IsError {
			// check (e): a granted cell whose server did not connect measured a toolless run --
			// caught here, against the candidate transcript, before this attempt is accepted and the
			// loop breaks. A cell that grants no tools is never this check's concern, matching how the
			// blinding checks are gated by GrantsTools at their own call site.
			if cfg.GrantsTools() {
				serverFinding = CheckServerConnected(t.Init, l.ServerName())
			}
			if serverFinding == nil {
				maxTurnsHit = t.Result.TerminalReason == maxTurnsTerminalReason
				if maxTurnsHit {
					transcript = t
					accepted = true
				} else {
					text := finalAssistantText(t.Records)
					_, inner, extractErr := ExtractFencedJSON(text, "last")
					if extractErr != nil {
						unparseableAnswerDetail = "final assistant text carries no fenced json block"
					} else {
						var probe map[string]any
						if json.Unmarshal([]byte(inner), &probe) == nil {
							transcript = t
							answerText = inner
							accepted = true
						} else {
							unparseableAnswerDetail = "the fenced json block did not decode"
						}
					}
				}
			}
		}
		if accepted {
			break
		}

		if serverFinding != nil {
			// connectFailures is incremented before the reason file below is written, so the
			// connectFailures == attempts whole-run abort further down fires on exactly the same
			// condition it does today.
			connectFailures++
		}

		attempt++
		reason := InvalidReason{Cell: cfg.ID, Repetition: rep, Attempt: attempt}
		switch {
		case invokeErr != nil:
			reason.Cause = CauseRunnerError
			reason.Detail = "the measured claude process failed"
			if code, ok := exitCodeOf(invokeErr); ok {
				reason.ExitCode = &code
				reason.Detail = fmt.Sprintf("%s: exit status %d", reason.Detail, code)
			}
		case t == nil || t.Result == nil || t.Result.IsError:
			reason.Cause = CauseResultError
			if t == nil || t.Result == nil {
				reason.Detail = "no result record in transcript"
			} else {
				reason.Detail = fmt.Sprintf(
					"result record reports terminal_reason=%q stop_reason=%q",
					t.Result.TerminalReason, t.Result.StopReason,
				)
			}
		case serverFinding != nil:
			reason.Cause = CauseServerNotConnected
			reason.Detail = serverFinding.Message
		default:
			reason.Cause = CauseUnparseableAnswer
			reason.Detail = unparseableAnswerDetail
		}
		// Persist the reason into the attempt directory, before InvalidateRep renames it away --
		// this path never reaches writeCompleteState, so the attempt's own directory is the only
		// place left for it to survive.
		if err := WriteInvalidReason(dir, reason); err != nil {
			return repOutcome{}, err
		}

		// Either an infrastructure failure (non-zero exit, unparseable stream, or an error flag in
		// the result record), a formatting miss (a missing or undecodable fenced answer), or a
		// granted server that did not connect -- invalidate this attempt and retry up to the ceiling.
		attempts, invalidateErr := InvalidateRep(dir)
		if invalidateErr != nil {
			return repOutcome{}, invalidateErr
		}
		if attempts >= MaxAttempts {
			if connectFailures == attempts {
				// Every attempt failed to connect the same server: a configuration or environment
				// fault, not bad luck, and the next repetition would reproduce it at the cost of
				// three more measured calls. Stop the whole invocation here instead, bounding the
				// blast radius at MaxAttempts calls rather than MaxAttempts times every remaining
				// repetition.
				return repOutcome{incomplete: true, abortRun: true}, nil
			}
			return repOutcome{incomplete: true}, nil
		}
	}

	metrics := ComputeMetrics(transcript, l.MCPPrefix())
	metrics.Effort = l.RunEffort

	// The first completed repetition of a root whose memory paths are not yet known derives them
	// from this repetition's own session-init record, writes them to the untracked raw tree and
	// their hashes into the provenance record at that moment -- not at the end of the run, so a kill
	// cannot lose them -- and scans them, discarding this repetition and aborting the run when
	// tainted.
	if !*memoryPathsKnown && transcript.Init != nil {
		paths := make([]string, 0, len(transcript.Init.MemoryPaths))
		for _, p := range transcript.Init.MemoryPaths {
			paths = append(paths, p)
		}
		if len(paths) > 0 {
			if err := writeMemoryPaths(opts.ResultsRoot, paths); err != nil {
				return repOutcome{}, err
			}
			hashes := make([]string, len(paths))
			for i, p := range paths {
				hashes[i] = sha256Hex(p)
			}
			prov.MemoryPathHashes = sortedUnion(prov.MemoryPathHashes, hashes)

			finding, err := ScanMemoryPaths(paths)
			if err != nil {
				return repOutcome{}, err
			}
			if finding != nil {
				// Tainted: discard this repetition exactly like a blinding failure -- write it
				// complete with the blinding-failed flag set rather than invalidating it, so its
				// transcript and the finding survive on disk -- and abort the run, since a resumed
				// invocation must not skip past the very repetition that revealed the taint.
				if err := writeCompleteState(dir, l, cfg, task, rep, metrics, []Finding{*finding}, true, false, false, "blinding_failed"); err != nil {
					return repOutcome{}, err
				}
				if err := RestoreWorktree(ctx, opts.Runner, dest); err != nil {
					return repOutcome{}, err
				}
				return repOutcome{blindingFailed: true, abortRun: true}, nil
			}
		}
		*memoryPathsKnown = true
	}

	if transcript.Init != nil {
		if prov.SessionFingerprints == nil {
			prov.SessionFingerprints = map[string]SessionFingerprint{}
		}
		prov.SessionFingerprints[repKey(cfg.ID, rep)] = NewSessionFingerprint(transcript.Init)
	}

	var observations []Finding
	if !cfg.GrantsTools() {
		if findings := CheckBlinding(transcript, blindingIn); findings != nil {
			for _, f := range findings {
				if f.Fatal {
					// Check (a) or (b), after the run: deterministic, so never retried. The
					// completeness predicate returns false for a repetition carrying the
					// blinding-failed flag, which is what makes a discarded control repetition
					// recoverable instead of permanently skipped.
					if err := writeCompleteState(dir, l, cfg, task, rep, metrics, []Finding{f}, true, false, false, "blinding_failed"); err != nil {
						return repOutcome{}, err
					}
					return repOutcome{blindingFailed: true}, nil
				}
				observations = append(observations, f)
			}
		}
	}

	porcelain, err := WorktreeStatus(ctx, opts.Runner, dest)
	if err != nil {
		return repOutcome{}, err
	}
	if f := CheckWorktreeDirtied(porcelain); f != nil {
		observations = append(observations, *f)
	}

	var (
		scoreRecord ScoreRecord
		scored      bool
		skipReason  string
	)
	if maxTurnsHit {
		// A max-turns terminal reason is complete, not a failure: full cost metrics, no answer
		// file, and the scorer is never invoked.
		skipReason = "max_turns"
		scoreRecord = UnscoredRecord(skipReason)
	} else {
		if err := writeAnswerFiles(dir, l, quarryRepoRoot, dest, answerText); err != nil {
			return repOutcome{}, err
		}
		redacted := redactedAnswerText(l, quarryRepoRoot, dest, answerText)

		scoreRecord, err = dispatchScorer(ctx, opts, l, task, content.TaskText, quarryRepoRoot, redacted)
		if err != nil {
			// A scorer failure retries only the scorer, up to MaxAttempts, never the measured run.
			skipReason = "scorer_failed"
			scoreRecord = UnscoredRecord(skipReason)
		} else {
			scored = true
		}
	}

	if err := writeUsageAndScore(dir, metrics, scoreRecord); err != nil {
		return repOutcome{}, err
	}

	if err := writeCompleteState(dir, l, cfg, task, rep, metrics, observations, false, maxTurnsHit, scored, skipReason); err != nil {
		return repOutcome{}, err
	}

	if err := RestoreWorktree(ctx, opts.Runner, dest); err != nil {
		return repOutcome{}, err
	}

	return repOutcome{}, nil
}

// exitCodeOf reports the exit code err's *exec.ExitError carries, and whether one was found. It
// never inspects err's own formatted message: ExecRunner.Run wraps a run failure as
// "run %s %s: %w" over the full argument vector, which carries the rendered prompt, the absolute
// --mcp-config path and the claude binary path -- exactly the kind of content the invalid-reason
// detail must never carry.
func exitCodeOf(err error) (int, bool) {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode(), true
	}
	return 0, false
}

// invokeMeasuredProcess runs the measured claude process for one cell-and-repetition pair through the
// runner seam, exactly as this file's header comment's argument vector requires, with its output
// teed to dir/transcript.jsonl as it arrives, and returns the parsed transcript. A non-zero exit or
// an unparseable stream is returned as an error, which the caller treats as an infrastructure
// failure; it never aborts the run by itself.
func invokeMeasuredProcess(ctx context.Context, opts RunOptions, l *Ladder, cfg Config, prompt, dest, mcpConfigPath, dir string) (*Transcript, error) {
	transcriptPath := filepath.Join(dir, TranscriptFile)
	f, err := os.Create(transcriptPath)
	if err != nil {
		return nil, fmt.Errorf("invoke measured process: %w", err)
	}
	defer f.Close()

	devNull, err := os.Open(os.DevNull)
	if err != nil {
		return nil, fmt.Errorf("invoke measured process: %w", err)
	}
	defer devNull.Close()

	args := []string{
		"-p", prompt,
		"--model", l.RunModel,
		"--effort", l.RunEffort,
		"--max-turns", strconv.Itoa(l.MaxTurns),
		"--tools", strings.Join(BuiltinTools, ","),
	}
	if cfg.GrantsTools() {
		prefixed := make([]string, len(cfg.Allowed))
		for i, a := range cfg.Allowed {
			prefixed[i] = l.MCPPrefix() + a
		}
		args = append(args, "--allowedTools", strings.Join(prefixed, ","))
	}
	args = append(args,
		"--mcp-config", mcpConfigPath,
		"--strict-mcp-config",
		"--output-format", "stream-json",
		"--verbose",
		"--no-session-persistence",
		"--setting-sources", "",
	)

	runErr := opts.Runner.Run(ctx, Cmd{
		Dir:    dest,
		Name:   opts.ClaudeBinPath,
		Args:   args,
		Stdin:  devNull,
		Stdout: f,
	})

	t, parseErr := parseTranscriptFile(transcriptPath)
	if runErr != nil {
		if parseErr == nil {
			return t, fmt.Errorf("invoke measured process: %w", runErr)
		}
		return nil, fmt.Errorf("invoke measured process: %w", runErr)
	}
	if parseErr != nil {
		return nil, fmt.Errorf("invoke measured process: %w", parseErr)
	}
	return t, nil
}

// parseTranscriptFile parses the tee'd transcript file at path.
func parseTranscriptFile(path string) (*Transcript, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("parse transcript file %s: %w", path, err)
	}
	defer f.Close()
	return ParseTranscript(f)
}

// finalAssistantText concatenates the text content of the final API call's assistant records --
// grouped by groupAssistantCalls, since Claude Code writes one transcript record per content block --
// which is both what the run's own answer and the scorer's reply are extracted from.
func finalAssistantText(records []Record) string {
	groups := groupAssistantCalls(records)
	if len(groups) == 0 {
		return ""
	}
	last := groups[len(groups)-1]
	var b strings.Builder
	for _, rec := range last {
		for _, block := range rec.Message.Content {
			if block.Type == "text" {
				b.WriteString(block.Text)
			}
		}
	}
	return b.String()
}

// writeVoidRepetition writes a control cell's repetition as complete with the blinding-failed flag
// set, for check (d)'s before-dispatch discard: no API call was spent, so no metrics or score exist
// to carry.
func writeVoidRepetition(dir string, l *Ladder, cfg Config, task Task, rep int, findings []Finding) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("write void repetition %s: %w", dir, err)
	}
	return writeCompleteState(dir, l, cfg, task, rep, Metrics{}, findings, true, false, false, "blinding_failed")
}

// redactionInputFor builds the RedactionInput common to redacting a repetition's answer.
func redactionInputFor(l *Ladder, quarryRepoRoot, taskWorktreePath string) RedactionInput {
	return RedactionInput{
		QuarryTools:      l.QuarryTools,
		ServerName:       l.ServerName(),
		MCPPrefix:        l.MCPPrefix(),
		QuarryRepoRoot:   quarryRepoRoot,
		TaskWorktreePath: taskWorktreePath,
	}
}

// redactedAnswerText returns answerText after RedactAnswer, using the ladder file, quarry
// repository root and pinned worktree path as the redaction input.
func redactedAnswerText(l *Ladder, quarryRepoRoot, taskWorktreePath, answerText string) string {
	return RedactAnswer(answerText, redactionInputFor(l, quarryRepoRoot, taskWorktreePath))
}

// writeAnswerFiles writes dir/answer.json and dir/answer.redacted.json from answerText, which is
// already-decoded-and-reencodable fenced JSON inner text.
func writeAnswerFiles(dir string, l *Ladder, quarryRepoRoot, taskWorktreePath, answerText string) error {
	if err := os.WriteFile(filepath.Join(dir, AnswerFile), []byte(answerText), 0o644); err != nil {
		return fmt.Errorf("write answer file: %w", err)
	}
	redacted := redactedAnswerText(l, quarryRepoRoot, taskWorktreePath, answerText)
	if err := os.WriteFile(filepath.Join(dir, RedactedAnswerFile), []byte(redacted), 0o644); err != nil {
		return fmt.Errorf("write redacted answer file: %w", err)
	}
	return nil
}

// dispatchScorer builds the scorer prompt from taskText, the fasit named by task.Fasit -- resolved
// against quarryRepoRoot exactly like task.TaskFile -- and redactedAnswer, and dispatches RunScorer,
// retrying only the scorer itself up to MaxAttempts on failure -- never the measured run. taskText is
// taken from the caller's already-loaded TaskContent rather than reloading the task file, so this
// function has no path of its own to resolve for it.
func dispatchScorer(ctx context.Context, opts RunOptions, l *Ladder, task Task, taskText, quarryRepoRoot, redactedAnswer string) (ScoreRecord, error) {
	rule, ok := ruleBySchema[task.Schema]
	if !ok {
		return nil, fmt.Errorf("dispatch scorer: unknown schema %q", task.Schema)
	}

	fasitPath := resolveRepoRelative(quarryRepoRoot, task.Fasit)
	data, err := os.ReadFile(fasitPath)
	if err != nil {
		return nil, fmt.Errorf("dispatch scorer: read fasit %s: %w", fasitPath, err)
	}
	var fasit map[string]any
	if err := json.Unmarshal(data, &fasit); err != nil {
		return nil, fmt.Errorf("dispatch scorer: decode fasit %s: %w", fasitPath, err)
	}

	prompt, err := BuildScorerPrompt(rule, taskText, fasit, redactedAnswer)
	if err != nil {
		return nil, fmt.Errorf("dispatch scorer: %w", err)
	}

	var lastErr error
	for attempt := 1; attempt <= MaxAttempts; attempt++ {
		record, err := RunScorer(ctx, opts.Runner, opts.ClaudeBinPath, l, task, prompt)
		if err == nil {
			return record, nil
		}
		lastErr = err
	}
	return nil, fmt.Errorf("dispatch scorer: exhausted %d attempts: %w", MaxAttempts, lastErr)
}

// RunScorer runs the scorer's own measured-binary invocation through r: a self-contained scratch
// directory and an MCP configuration document whose servers map is empty, exactly the argument vector
// this file's header comment requires, with its working directory set to that scratch directory
// rather than the pinned worktree the scorer must never see. It parses the reply from the final
// assistant record's concatenated text against task's schema.
func RunScorer(ctx context.Context, r Runner, claudeBin string, l *Ladder, task Task, prompt string) (ScoreRecord, error) {
	scratchDir, err := os.MkdirTemp("", "ladder-scorer-")
	if err != nil {
		return nil, fmt.Errorf("run scorer: %w", err)
	}
	defer os.RemoveAll(scratchDir)

	doc, err := MCPConfigDocument(l, Config{}, "", "")
	if err != nil {
		return nil, fmt.Errorf("run scorer: %w", err)
	}
	mcpConfigPath := filepath.Join(scratchDir, "mcp-config.json")
	if err := os.WriteFile(mcpConfigPath, doc, 0o644); err != nil {
		return nil, fmt.Errorf("run scorer: write mcp config: %w", err)
	}

	devNull, err := os.Open(os.DevNull)
	if err != nil {
		return nil, fmt.Errorf("run scorer: %w", err)
	}
	defer devNull.Close()

	var stdout bytes.Buffer
	args := []string{
		"-p", prompt,
		"--model", l.Scorer.Model,
		"--effort", l.Scorer.Effort,
		"--tools", "",
		"--max-turns", "1",
		"--mcp-config", mcpConfigPath,
		"--strict-mcp-config",
		"--output-format", "stream-json",
		"--verbose",
		"--no-session-persistence",
		"--setting-sources", "",
	}
	if err := r.Run(ctx, Cmd{
		Dir:    scratchDir,
		Name:   claudeBin,
		Args:   args,
		Stdin:  devNull,
		Stdout: &stdout,
	}); err != nil {
		return nil, fmt.Errorf("run scorer: %w", err)
	}

	t, err := ParseTranscript(&stdout)
	if err != nil {
		return nil, fmt.Errorf("run scorer: parse transcript: %w", err)
	}
	if t.Result == nil || t.Result.IsError {
		return nil, fmt.Errorf("run scorer: result record missing or reported an error")
	}

	reply := finalAssistantText(t.Records)
	record, err := ParseScorerReply(reply, task.Schema)
	if err != nil {
		return nil, fmt.Errorf("run scorer: %w", err)
	}
	return record, nil
}

// writeUsageAndScore writes dir/usage.json from metrics and dir/score.json from score, in that
// order.
func writeUsageAndScore(dir string, metrics Metrics, score ScoreRecord) error {
	usageData, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		return fmt.Errorf("write usage file: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, UsageFile), usageData, 0o644); err != nil {
		return fmt.Errorf("write usage file: %w", err)
	}

	scoreData, err := json.MarshalIndent(score, "", "  ")
	if err != nil {
		return fmt.Errorf("write score file: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ScoreFile), scoreData, 0o644); err != nil {
		return fmt.Errorf("write score file: %w", err)
	}
	return nil
}

// writeCompleteState writes dir/run.json, the state file, last -- after every other file this
// repetition writes, per the write-last rule the overview's six-per-repetition-filenames decision
// states. blindingFailed, maxTurnsHit, scored and skipReason flow straight onto the RunState fields
// of the same name; the score file itself is written separately by writeUsageAndScore.
func writeCompleteState(dir string, l *Ladder, cfg Config, task Task, rep int, metrics Metrics, observations []Finding, blindingFailed, maxTurnsHit, scored bool, skipReason string) error {
	controlForLadder := ""
	if cfg.IsControl() {
		controlForLadder = cfg.Ladder
	}

	state := RunState{
		State:            "complete",
		ConfigID:         cfg.ID,
		Ladder:           cfg.Ladder,
		Task:             cfg.Task,
		Allowed:          cfg.Allowed,
		IsControl:        cfg.IsControl(),
		ControlForLadder: controlForLadder,
		ServerName:       l.ServerName(),
		MCPPrefix:        l.MCPPrefix(),
		Rep:              rep,
		Model:            l.RunModel,
		Effort:           l.RunEffort,
		MaxTurns:         l.MaxTurns,
		Scored:           scored,
		ScoreSkipReason:  skipReason,
		Observations:     observations,
		BlindingFailed:   blindingFailed,
		MaxTurnsHit:      maxTurnsHit,
	}
	return WriteRunState(dir, state)
}
