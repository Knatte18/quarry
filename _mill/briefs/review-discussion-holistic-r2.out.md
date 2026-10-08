MILL_REVIEW_BEGIN
# Review: Enclose: path:line ranges -> enclosing member glyphs (batch, at revision, CLI verb)

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewed_file: _mill/discussion.md
date: 2026-10-08
```

## Findings

### [BLOCKING:design] Working-tree unit read from a vote the memo never holds
**Section:** Engine architecture, working-tree bullet ("The unit is computed from the memo's vote and record").
**Issue:** `fileTargetAnswer` calls `m.dirVote(dirRel, recs, !targetIgnored)`, so for an explicitly named gitignored target (a listed test) the extended vote is never stored in `m.votes`; the key is absent, or holds the plain vote from an earlier location in the same directory, which can differ from the vote `toc` stamped `FileEntry.Symbols` with.
**Fix:** Name the vote source that matches what `toc` stamped, e.g. derive the unit from the returned `DirAnswer.Package` (which is that call's `v.pkg`) rather than from `m.votes`.

### [BLOCKING:consistency] Root path "." classified two ways
**Section:** Path normalisation vs Rejection vocabulary.
**Issue:** Path normalisation says a path normalising to `.` is `unsupported_language`, but check 4 (path-half `unaddressable`, `glyph.Self(glyph.Go, ".")`) runs before check 5, and `glyph.Self` delegates to `Parse(".#")`, which rejects a dot segment, so the order yields `unaddressable`.
**Fix:** State one answer for `.` and make the order or the sentence agree with it.

### [NIT:design] Vanished-target race has no sentinel to match
**Section:** Engine architecture, working-tree bullet (race maps to `missing_file`).
**Issue:** `fileTargetAnswer`'s "no longer exists in directory" error is a bare `fmt.Errorf` with no wrapped sentinel, so separating it from the `unreadable` errors needs a string match or a new sentinel, and the discussion picks neither.
**Fix:** Say whether to wrap `ErrTargetNotFound` there or accept `unreadable` for the race.

### [NIT:scope] Empty `--rev` value unspecified
**Section:** CLI verb.
**Issue:** `--from`/`--to` reject an empty value. `--rev ""` would reach `EncloseAt("")` and silently answer against the working tree.
**Fix:** State whether `--rev ""` is a usage error, as it is for `--from`/`--to`.

## Verdict

REQUEST_CHANGES
Gitignored-target unit source and the "." classification contradiction need resolving before planning.
MILL_REVIEW_END
