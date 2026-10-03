MILL_REVIEW_BEGIN
# Review: Resolve self-target path: per-call dirPackage memo (GH #34)

```yaml
duration_s: 137.4
verdict: APPROVE
reviewer_model: opus
reviewed_file: _mill/discussion.md
date: 2026-10-03
```

## Findings

### [NIT:design] Stamp list omits Symbol.File
**Section:** Decisions / Unit stamping after the vote
**Issue:** The stamp names only `Glyph.Unit` and `ID`, but `symbolsOfDir` also sets `sym.File`, and `answer.go` requires `File` to stay empty (`omitempty`) inside a TOC answer.
A per-file record keyed by path invites storing `File` in it, which would change TOC JSON.
**Fix:** Add `File` to the member consumer's stamp, and state that records hold no `File` and the walk consumer never sets it.

### [NIT:design] Member-only directories now build non-language files
**Section:** Test seam; Lazy, per-directory, single-parse extraction
**Issue:** The expected set requires a build count of exactly 1 for every non-symlink file in a member glyph's directory, so the member path must now read files with no language (e.g. `README.md`), which today's `dirPackage` never opens.
The lazy decision's "the first time a call touches a file" does not say this, and "the directory of each member glyph" does not say it means `unitDirs`' result (stripped `_test` directory, both directories on collision).
**Fix:** State that the first touch of a directory builds records for all of its filtered non-symlink files on every path, and define the member glyph's directories as `unitDirs(g.Unit)`.

## Verdict

APPROVE
Source claims check out; the two NITs are an incomplete stamp list and an unstated member-path read.
MILL_REVIEW_END
