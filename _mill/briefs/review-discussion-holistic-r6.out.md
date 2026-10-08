MILL_REVIEW_BEGIN
# Review: Enclose: path:line ranges -> enclosing member glyphs (batch, at revision, CLI verb)

```yaml
verdict: APPROVE
reviewer_model: opushigh
reviewed_file: _mill/discussion.md
date: 2026-10-08
```

## Findings

### [NIT:design] Revision listing can name a non-blob `*.go` entry
**Section:** Engine architecture, revision bullet ("git listed the name, so a read failure means the call is broken").
**Issue:** `DirFilesAtRevision` filters `ls-tree` names by suffix only, so a tree named `x.go` is read as its tree listing (answered as garbage where the working tree says `missing_file`), and a gitlink named `x.go` makes `ReadBlob` fail and kills the whole call for every location in that directory.
**Fix:** Name it as a third revision asymmetry beside the two in `EncloseAt`'s doc comment (it is shared with `revisionClauseMap`), or state that it is accepted as is.

### [NIT:scope] `Status` doc comments enumerate their users
**Section:** Status set and result shape; Path normalisation ("name the subsystem rather than list every caller").
**Issue:** `engine.Status`'s doc in `internal/engine/answer.go` ("shared by ResolveResult's Status and Unit keys and by ExpandAnswer's") and the `quarry.Status` alias doc in `quarry/quarry.go` ("both ResolveResult and ExpandAnswer draw from") go stale once `EncloseResult.Status` reuses the type, and the discussion's doc-rewrite list omits them.
**Fix:** Add both to the doc-comment updates, under the same name-the-subsystem rule.

### [NIT:design] Column overflow disposition unstated
**Section:** Location spelling.
**Issue:** Line overflow is `bad_location`, but the discussion does not say whether an overflowing `:C` column is parsed (and rejected) or ignored unparsed, and the parser table test does not pin it.
**Fix:** State one disposition and add the case to the parser table.

### [NIT:scope] No completeness test for `EncloseReasons`
**Section:** Testing.
**Issue:** The engine has `TestName_ReasonCompleteness` for `NameReasons`, but Testing only checks that the facade's `EncloseReasons` is the engine's slice, not that it holds exactly the eight constants once each.
**Fix:** Add an engine completeness test that mirrors the `NameReasons` one.

## Verdict

APPROVE
No blocking gaps; source claims checked against memo, toc, walk, repopath, gitsrc and cli hold.
MILL_REVIEW_END
