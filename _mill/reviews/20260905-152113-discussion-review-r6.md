MILL_REVIEW_BEGIN
# Review: P2 — diff-to-symbols: changed file versions to symbol-table delta (roadmap 2c)

```yaml
duration_s: 360.0
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus (Anthropic); exact version not independently verifiable from inside the session — the harness names Opus 5
reviewed_file: _mill/discussion.md
date: 2026-09-05
```

## Findings

### [BLOCKING:design] Stated ordering is not total; goldens still unstable
**Section:** § Every array in the answer has a stated, deterministic order
**Issue:** `created`/`deleted`/`after` are sorted "file ascending, then `Start` ascending" with `sort.SliceStable`, but the pre-sort order is a range over the `(ID,Kind)` map, so equal `(File,Start)` pairs break randomly — and the discussion's own example `const a, b = 1, 2` produces exactly that (`goUngroupedConstOrVarSymbols`/`goGroupedConstOrVarSymbols` give every name in one spec the same `Start` and `End`). The `changed` array's element order is also never stated.
**Fix:** Add a total tie-break after `Start` (e.g. `id` then `kind`) and fix `changed`'s order to the closed set's own order.

### [BLOCKING:design] `DeltaEntry` cannot carry a pre-extraction refusal
**Section:** § Entry dispositions… + § Git plumbing (status-letter table)
**Issue:** The core owns the `files` array ("one entry per input entry, in the input's own order") and must emit `disposition: error` with message `unmerged path` for `U` and a message naming the letter for an unknown one — but the entry is specified as `(path, before-bytes, after-bytes, units)` with no field for a refusal or its message, so the git layer has no way to express it. The same missing seam leaves a working-tree disk-read failure during batch assembly with no stated disposition (it is not "a git command that failed", so exit 3 does not obviously cover it, and failing the batch contradicts "a failing entry never fails the batch").
**Fix:** Name the refusal-carrying field on `DeltaEntry` and state which dispositions the git layer may pre-set, including a read failure on either side.

### [BLOCKING:design] Top-level equality test has no normalisation rule
**Section:** § The repository root must be git's top-level
**Issue:** `DeltaGit` "requires [`git rev-parse --show-toplevel`] to equal `<root>`", but `repopath.ResolveRoot` only does `filepath.Join`+`Clean` and never `EvalSymlinks`, while git prints the physical path — so a `--root` reached through a symlink (and `t.TempDir()` on any platform whose temp dir is symlinked) yields a false `ErrRootNotTopLevel` exit 2 on a legitimate repository.
**Fix:** State how the two paths are compared (both `EvalSymlinks`'d, or both `Clean`'d) so the check cannot refuse a valid root.

### [NIT:design] git output parsing assumes unquoted, newline-delimited paths
**Section:** § Git plumbing: exactly which calls
**Issue:** The command list is presented as exhaustive, but `git diff --name-status`, `git ls-files` and `git ls-tree --name-only` C-quote paths under the default `core.quotePath` and delimit with `\n`; no `-z` is specified, so a non-ASCII or control-character path arrives mangled and is read at the wrong location.
**Fix:** State `-z` (or explicit unquoting) for every path-emitting call.

### [NIT:decision] `GitDeltaAnswer` has no stated home or renderer signature
**Section:** § The answer carries no revision information + Technical context
**Issue:** The new-types list names `DeltaEntry`, `DeltaAnswer`, `DeltaFile`, `ModifiedSymbol`, `RenamedPair`, `RenameCandidate`, `RenameSignals` but not `GitDeltaAnswer`; the facade is stated to hold aliases only, and the engine is stated to know nothing about git, so neither package is an unambiguous home. `RenderDeltaJSON`/`RenderDeltaText`'s parameter type (wrapped vs core) is likewise unstated while "the CLI renders the wrapped form".
**Fix:** Name `GitDeltaAnswer`'s package and the renderers' parameter type.

### [NIT:consistency] Requirements attributed to a "Done when" that does not exist
**Section:** Scope, § `--text`, § Exact-tier scope, Testing (Goldens)
**Issue:** Four passages justify requirements by "the task's 'Done when'" (the seven golden cases, both views, the synthetic exact-tier test), but `_mill/status.md`'s `task_description` is one line and `docs/roadmap.md` point 2c carries no "Done when" clause.
**Fix:** Restate these as this discussion's own decisions, or cite the artefact that actually carries them.

## Verdict

REQUEST_CHANGES
Ordering not total, entry cannot carry a refusal, top-level comparison unspecified.
MILL_REVIEW_END
