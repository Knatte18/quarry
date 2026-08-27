MILL_REVIEW_BEGIN
# Review: Add file/dir toc verbs (Tree-sitter-backed)

```yaml
duration_s: 201.7
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Anthropic Claude, opus-class (runtime reports claude-opus-5); exact build not verifiable from inside the session
reviewed_file: /home/knatte/Code/quarry/wts/toc-verbs/_mill/discussion.md
date: 2026-08-27
```

## Findings

### [NIT:consistency] Spike section contradicts the grammar-set decision
**Demoted-from:** BLOCKING
**Section:** Technical context → Spike results (last paragraph) vs "Grammar set" Decision
**Issue:** The spike paragraph says "Binary-size impact of the default embedded set was not measured in the spike and should be checked during implementation" and names only `grammar_set_core` / `GOTREESITTER_GRAMMAR_SET` / `grammar_blobs_external`, while the Decision records a measured size table, mandates per-language `grammar_subset` tags, and explicitly rejects "deferring the choice to implementation".
**Fix:** Delete or rewrite the superseded paragraph so the only surviving statement is the measured, decided one, and add the `grammar_subset` tag family to the API-surface list.

### [BLOCKING:design] `toc dir` behaviour for `.ts` / `.rs` files undefined
**Section:** "`toc dir`: one level…" + "Batch mode" status table
**Issue:** `toc dir` lists "every file whose extension maps to a *supported* language", but the discussion never says whether a designed-but-unimplemented extension (`.ts`, `.rs`) counts as supported: skipped silently, listed with an `error` key, or failing the directory. The status table's "designed-but-unimplemented → error rank 3" row is a per-path row and does not answer the per-file case, and the empty-directory rule ("no files matching a supported extension") inherits the same ambiguity.
**Fix:** State explicitly what `toc dir` emits for a `.ts`/`.rs` file in the listing, and whether such a file counts toward "no supported files".

### [BLOCKING:design] Path resolution and emitted path form unspecified
**Section:** "Path-type validation" / Technical context → existing structure
**Issue:** The decision says `os.Stat` the positional argument, but never says relative arguments resolve against `CwdFrom(ctx)` (`internal/cli/cwdcontext.go:41`) — a raw `os.Stat(arg)` uses the process cwd and bypasses the seam that `RunCLIIn`/`WithCwd` exist to enforce, which the `internal/cli` tests depend on. Separately, the form of the paths emitted in `toc dir`'s file entries (basename, relative to the dir argument, or absolute) is never decided, and those paths are what an agent feeds back into `toc file`.
**Fix:** State that toc joins relative arguments against the seam cwd, and fix one path form for `toc dir` entries and the batch `"path"` key.

### [BLOCKING:design] `toc file` symbol schema incomplete: no kind/language fields
**Section:** "Output: flat symbol list" / "Symbol kinds"
**Issue:** The decisions fix `name`, `owner`, `docstring`, `start`, `end`, `signature`, `header`, `partial`, `test`, `generated`, `error` — but never say whether a symbol carries a `kind` field or what its cross-language vocabulary is (Python `class_definition`, C# `record`/`interface`/`struct` all collapse to what?), nor what the per-file language field in `toc dir` is called. The spike row hints at `function`/`method`/`type` but no Decision adopts it.
**Fix:** Add a decision naming the emitted key set for a symbol entry and a file entry, including the `kind` vocabulary shared across all five languages.

### [NIT:scope] Stale-doc inventory is incomplete, not "four"
**Demoted-from:** BLOCKING
**Section:** Scope → "Four exact-count / exact-claim doc invariants"
**Issue:** At least three more statements go stale on the same landing: `quarry/facade.go:3-4` ("the five-package DAG … root, lsp, registry, daemon, query"), `internal/quarryengine/doc.go:24-25,37,44-71` (which enumerates the DAG in full and calls it five-package), and the enumerating comments in `internal/quarryengine/layering_test.go:20,51-54,159` and `seam_enforcement_test.go:1-11,100-104`. The list's exhaustiveness claim ("Four") is therefore wrong, and the enumeration method that produced it is not stated.
**Fix:** Re-derive the invariant list by an enumeration method the plan can restate (e.g. grep for the DAG-count phrases and the verb-count phrases), and drop the fixed "four" framing.

### [NIT:consistency] `buildOptions` cited as a multi-line signature; it is not
**Demoted-from:** BLOCKING
**Section:** "Signature: exact source text" → Rejected
**Issue:** "the repo has such signatures — see `buildOptions` in `internal/cli/cli.go:492`" is false: line 492 is one line ending `… timeout time.Duration) quarry.Options {`, and a search for `^func` lines ending in `(` or `,` returns no match anywhere in the tree — the repo has no multi-line function signature at all.
**Fix:** Drop the false citation (the no-first-line-cut decision stands on its own), or replace it with a source location that actually demonstrates the shape.

### [NIT:decision] `Outliner` left as "worth evaluating"
**Section:** Technical context → gotreesitter API surface
**Issue:** The library's `Outliner` / `Owner` field is named as "worth evaluating for the owner-resolution step" with no disposition, leaving the plan writer to choose between it and the hand-written walk.
**Fix:** State use-it or don't-use-it for owner resolution, with one line of rationale.

## Verdict

REQUEST_CHANGES
Six blocking gaps: a superseded spike paragraph, three undecided contracts, one bad inventory, one false citation.
_Note: 3 finding(s) demoted from BLOCKING to NIT by the stage's blocking-class ceiling; current blocking_count is 3._
MILL_REVIEW_END
