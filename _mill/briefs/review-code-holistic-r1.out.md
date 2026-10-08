MILL_REVIEW_BEGIN
# Review: Enclose: path:line ranges -> enclosing member glyphs (batch, at revision, CLI verb) — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: sonnethigh
reviewed_file: plan/ + source
date: 2026-10-08
```

## Findings

### [BLOCKING:scope] quarry/doc.go left stale and absent from every batch
**Location:** `quarry/doc.go:7-33`
**Issue:** The package doc still states "six queries, not four", "thirteen renderers", names `DeltaGit` as the only convenience over the git layer, and enumerates the renderer sets without `RenderEncloseJSON`; `Enclose`/`EncloseAt` are not mentioned.
The plan rewrote the same kind of tally elsewhere (`render.go` header, `Open`, `Status` aliases, `cli/doc.go`) but never listed this file in any `Edits:`.
**Fix:** Add `quarry/doc.go` to a batch's reference lists and rewrite the passages to name the subsystem without counts or caller lists.

### [NIT:consistency] A verb tally survives in cli/doc.go
**Location:** `internal/cli/doc.go:14`
**Issue:** "it is not a sixth pipeline" keeps the count that card 10 asked to remove, and it no longer matches the verb set now that enclose has its own pipeline.
**Fix:** Say "glyphs has no pipeline of its own" without an ordinal.

### [NIT:consistency] flags_test.go comments and gate table not updated for enclose
**Location:** `internal/cli/flags_test.go:331-336`, `internal/cli/flags_test.go:375-378`
**Issue:** `TestParseArgs_FiveVerbGate` says "all six verbs" and has no `enclose` row; the `TestParseArgs_TextOnEveryVerbRootOnRepositoryVerbs` comment says `--text` is accepted "on every verb" and `--root` on "the five repository verbs", both false after this change.
**Fix:** Rewrite both comments without counts and state that `--text` is rejected for enclose.

### [NIT:consistency] Comment formatting residue from the edits
**Location:** `internal/cli/flags.go:73-74`, `quarry/repo.go:24-28`
**Issue:** `parseArgs`'s doc breaks mid-list (`"enclose" and` / `"name". --depth,`), and `Open`'s doc has one over-long line followed by a short one, unlike the wrapped neighbours.
**Fix:** Reflow both paragraphs.

## Verdict

REQUEST_CHANGES
One out-of-plan stale package doc (`quarry/doc.go`); engine, facade and CLI behaviour match the plan and discussion.
MILL_REVIEW_END
