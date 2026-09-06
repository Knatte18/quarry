MILL_REVIEW_BEGIN
# Review: M4 matrix run: execute the descoped kick-start batch (cards 29-32)

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (1M context)
reviewed_file: /home/knatte/Code/quarry/wts/kickstart-matrix-run/_mill/discussion.md
date: 2026-09-06
```

## Findings

### [BLOCKING:design] Whole-run abort has no disposition
**Section:** §Decisions `live-matrix-runs-detached` / `short-arm-disposition`
**Issue:** `repOutcome.abortRun` (run.go:625, memory-path taint via `ScanMemoryPaths`, provenance.go:518 — fatal on any memory file containing the bare token "quarry" or on a missing memory path) stops the whole matrix mid-run; `Run` then returns `nil` and `summarizeAndReport` (main.go:98, 172) still writes `summary.json`, writes and **prints** `table.txt`. The discussion's completion predicate ("process gone **and** the log's last lines carry the rendered table") therefore reads an abort at rep 1 as a normal completion, after which its own rule forbids any resume — voiding the root with no recovery, while the harness's own comment (run.go:615-618) exists precisely so a resume re-runs the tainted repetition.
**Fix:** State a disposition for an aborted invocation: how to detect it (realised reps in `table.txt` far below the ceiling, `blinding_failed_count` on the aborting cell, `raw/*/state`), whether clearing the taint and resuming is permitted, and why that is or is not optional stopping.

### [BLOCKING:design] Substitution consequence rests on a false premise
**Section:** §Decisions `glyph-substitution-contingency-is-fully-specified`, "Consequence, and how the conclusion handles it"
**Issue:** The prescribed rewording says e1's card "names the fasit's seven files verbatim inside its pack block while e2 names only the six its glyph set reaches", so "e2's inflation is no longer maximal". But `Pack` renders the block from the same `pack_targets` (pack.go:287-292, one `<target> → <file> <span>` line per target), so a substitution that vacates a file vacates it from **both** e1's block and e2's `Files:` list — the two arms name the identical file set by construction.
**Fix:** Restate the consequence as both non-control arms dropping to six files (or drop the asymmetry claim), and say what the coverage paragraph then asserts about the fasit's seventh file.

### [BLOCKING:design] Reserve-glyph selection rule is unsatisfiable for two targets
**Section:** §Decisions `glyph-substitution-contingency-is-fully-specified`, step list preamble
**Issue:** The rule is "replace the offending glyph with one of those three (same package, same mechanism)", but the reserves sit only in `internal/fabricengine` (two) and `internal/gitrepo` (one); `pack_targets` also carries `internal/fabriccli#addWeftVerbs` and `internal/mergeresolve#Resolver.Resolve`, for which no same-package reserve exists. The branch also does not say which reserve to pick when several qualify, or what to do if two glyphs fail or a reserve itself does not resolve `found`.
**Fix:** Give a deterministic selection rule covering the fabriccli and mergeresolve cases (or say those two failures halt), and state the multi-failure behaviour.

### [BLOCKING:decision] M4b follow-up condition disappears with roadmap point 1
**Section:** §Decisions `roadmap-card-32-adaptation`, edit 2 / §Scope Out
**Issue:** Roadmap point 1's last sentence ("If — and only if — e1 separates, an edit-task variant (M4b …) becomes a candidate follow-up.", docs/roadmap.md:23-24) is the only record of that condition, and the three prescribed edits delete it without replacement; the discussion lists M4b as out of scope but never says whether the condition survives, which the file's own charter ("only ever says what is ahead") makes load-bearing on a separating result.
**Fix:** State the disposition — carry the M4b conditional into the new standing-rule sentence or into the conclusion, or record that its deletion is intended.

### [NIT:design] Resume liveness test conflates two different pids
**Section:** §Decisions `live-matrix-runs-detached`, "Liveness test" / resume step 1-2
**Issue:** `$!` from `nohup go run … &` is the `go run` parent, while the lock file records `os.Getpid()` of the compiled ladder child (worktree.go:309); `kill -0 $!` and the lock file's `pid=` are therefore different processes, and neither alone proves the other is gone.
**Fix:** Say the authoritative check is the lock file's own `pid=` plus `pgrep -f`, with `$!` as a convenience only.

## Verdict

REQUEST_CHANGES
Four unresolved items: abort disposition, a false-premise consequence, and two under-specified branches.
MILL_REVIEW_END
