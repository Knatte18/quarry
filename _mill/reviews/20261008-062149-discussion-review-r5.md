MILL_REVIEW_BEGIN
# Review: Enclose: path:line ranges -> enclosing member glyphs (batch, at revision, CLI verb)

```yaml
duration_s: 157.9
verdict: APPROVE
reviewer_model: opushigh
reviewed_file: _mill/discussion.md
date: 2026-10-08
```

## Findings

### [NIT:consistency] Revision Read error cannot fail the call via buildRecord
**Demoted-from:** BLOCKING
**Section:** Engine architecture, revision bullet ("A `GoFiles` or `Read` error fails the whole call") vs the byte-source seam bullet.
**Issue:** `buildRecord` (memo.go) turns every read failure into `rec.err`, so with `read = files.Read` a blob-read failure becomes an `unreadable` target or, for a sibling file, a silently non-voting record that can shift the unit, never the whole-call failure the discussion promises.
**Fix:** State how a seam read error escapes `buildRecord` on the revision path, either through a distinct record field or an adapter-captured error checked by `EncloseFrom`, and keep the working-tree default mapping read errors to `rec.err`/`unreadable`.

### [NIT:design] CLI whole-call failure sentences unspecified
**Section:** CLI verb, Exit codes.
**Issue:** `runDelta` spells its exit-2 messages from typed error fields (`delta: unknown revision X`, the top-level and not-a-repository sentences), and Run's doc forbids leaking wrapped chains on exit 2, but the discussion names only the code mapping for `enclose`.
**Fix:** Name the `enclose:`-prefixed sentences for unknown revision, root not top level and not a repository, mirroring `runDelta`.

### [NIT:consistency] Facade-decided rejections listed under engine tests
**Section:** Testing, "Engine, working tree".
**Issue:** `bad_location`, `.:1` and the space-segment case are decided by the facade (checks 1–4), so an engine-level test passing `Location` values cannot produce them; also, check 4's `glyph.Self` already rejects any unspellable directory segment, so check 8 is reachable only for root-level files.
**Fix:** Move those cases to the facade tests, and limit the engine test of pre-rejection to pass-through of a pre-rejected `Location`.

## Verdict

APPROVE
The revision path's one-build read seam contradicts its stated whole-call failure on a blob read error.
_Note: 1 finding(s) demoted from BLOCKING to NIT by the stage's blocking-class ceiling; current blocking_count is 0._
MILL_REVIEW_END
