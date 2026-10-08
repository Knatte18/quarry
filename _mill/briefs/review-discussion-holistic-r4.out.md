MILL_REVIEW_BEGIN
# Review: Enclose: path:line ranges -> enclosing member glyphs (batch, at revision, CLI verb)

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewed_file: _mill/discussion.md
date: 2026-10-08
```

## Findings

### [BLOCKING:design] Working-tree found answers that never Resolve
**Section:** Engine architecture (working tree), Rationale "round-trip through `Resolve`".
**Issue:** The round-trip premise is false for inputs `r.toc` answers but `symbolsOfDir`/`unitDirs` skip.
These are an explicitly named gitignored file (`symbolsOfDir` drops `ig.match` entries, and the unit can come from a vote `Resolve` never computes), a parsed file with no package clause (`fileEntry` stamps it, `symbolsOfDir` skips `!rec.hasClause`), and a path through a symlinked directory (`dirExists` uses `Lstat`).
In each case Enclose returns `found` glyphs that the consumer's working-tree `Resolve` answers `not_found`, and the consumer cannot tell these apart from a deleted member.
**Fix:** Choose a disposition for each case: either list them as named working-tree asymmetries in `Enclose`'s doc comment beside the two revision ones, or reject them with a reason.
Then narrow the round-trip claim to match.

### [BLOCKING:consistency] `unaddressable` cannot carry `file` for a `#` path
**Section:** Rejection vocabulary (closing sentence) vs. Path normalisation.
**Issue:** The discussion says an `unaddressable` rejection carries `file`, `start` and `end`.
But `RepoRelTarget` returns `("", ErrTargetHasSeparator)` for a `#` segment, so the facade has no normalised path to put in `file`.
**Fix:** State that the `ErrTargetHasSeparator` case omits `file`, following the "filled whenever determined" rule.
Alternatively, name where the facade gets a normalised path for it.

### [NIT:consistency] Caller-enumerating doc comments go stale
**Section:** Path normalisation; Facade surface.
**Issue:** Only `repopath`'s package doc is scheduled for update.
`repoRelTarget`'s doc ("toc and delta are the only two verbs", and "`#`… is an explicit error for both") and `RepoRelTarget`'s caller list in `internal/repopath/target.go` are not, nor is `quarry.Open`'s doc in `quarry/repo.go`, which names `DeltaGit` as the only git exception.
**Fix:** Add these doc comments to the scope.

### [NIT:design] `--stdin` read failure unspecified
**Section:** CLI verb.
**Issue:** No exit code or behaviour is given for a stdin read error, or for a line over a `bufio.Scanner` default buffer if one is used.
**Fix:** Name the exit code (presumably 3 via `fail`) and the line-length handling.

## Verdict

REQUEST_CHANGES
Working-tree found glyphs can fail Resolve unacknowledged, and the unaddressable `file` contract contradicts `RepoRelTarget`.
MILL_REVIEW_END
