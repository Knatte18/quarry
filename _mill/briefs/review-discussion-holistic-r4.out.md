MILL_REVIEW_BEGIN
# Review: Facade + CLI, resolve + expand (T5b)

```yaml
verdict: APPROVE
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (claude-opus-5)
reviewed_file: _mill/discussion.md
date: 2026-09-04
```

## Findings

### [NIT:consistency] D5 and Technical context disagree on renderer file placement
**Section:** D5 / Technical context **Issue:** D5 decides "two new files hold the new renderers", while Technical context puts the text renderers in the existing `quarry/text.go` and offers `quarry/render.go` *or* a new `resolve.go`/`expand.go` pair as "mill-plan's call". **Fix:** state one disposition, or say plainly that file placement is delegated and drop D5's "two new files" clause.

### [NIT:consistency] D2's "four payload-free paths" is five in D3
**Section:** D2 **Issue:** D2 enumerates usage errors, internal errors, `*NotATypeError`, and the exit-1 grammar rejection as "the complete set of payload-free paths", but D3 adds a fifth — the `repoRelPath`/`filepath.Rel` failure, exit 1 with the error envelope and `target outside repository: <target>` — which is neither a usage nor an internal error. **Fix:** add that row to D2's list, or replace "the complete set" with a pointer to D3.

### [NIT:consistency] D15's exit-code column is not total over after-side files
**Section:** D15 **Issue:** D15 claims "**every** after-side file has a row, so every one of them records its exit code", but its own table folds `resolve-glyph-text.txt` and `expand-type-text.txt` into their siblings' rows as "(+ `-text`)" — unlike T5a's precedent it cites in the same paragraph, which gave `toc-dir-text.txt`/`toc-file-text.txt` their own `*(none)*` rows. **Fix:** give the two `-text` goldens their own rows, or say the shared cell covers both files.

### [NIT:scope] D13 leaves usageText's `flags:` block unspecified
**Section:** D13 **Issue:** D13 fixes the `usage:` block and the exit-code line to the character but never says whether `usage.go`'s existing `flags:` section (which describes `--depth`, `--symbols`, `--no-symbols`, `--text`, `--root`, `-h`) is kept, per-verb, or replaced by the inline bracketed flag lists in the snippet. **Fix:** spell the full replacement `usageText` the way D7 spells the text grammar.

### [NIT:design] CLI and engine can disagree on glyph-vs-path after rebasing
**Section:** D8 **Issue:** the CLI classifies argv with `strings.Contains(target, "#")` but hands the engine the *rebased* form, and `internal/engine/resolve.go`'s `isGlyphTarget` re-classifies it — so `quarry resolve foo.go` run inside a directory whose name contains `#` reaches the engine as a glyph and returns a grammar rejection instead of a path answer. **Fix:** state the disposition (accepted edge, or classify once and pass the decision down).

## Verdict

APPROVE
Decisions are complete, source-grounded and verified against the engine; only consistency nits remain.
MILL_REVIEW_END
