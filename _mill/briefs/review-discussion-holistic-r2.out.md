MILL_REVIEW_BEGIN
# Review: Delete V1, keep the tree-sitter extractors (T0)

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 4.x-class model (Anthropic); exact build not self-verifiable
reviewed_file: _mill/discussion.md
date: 2026-09-03
```

## Findings

### [BLOCKING:consistency] toc_integration_test.go reads internal/output
**Section:** Testing → `internal/quarryengine/toc`; Decisions → `five-commits-each-green` (commit 1)
**Issue:** `toc_integration_test.go:27-32` builds `<moduleRoot>/internal/output/output.go` via `runtime.Caller` and `t.Fatalf`s when `TOCFile` errors, and it asserts `Package == "output"` plus symbols `Ok`/`Err`/`ErrFields`; commit 1 deletes `internal/output`, so commit 1 is red — yet Testing lists this file under "must pass untouched. If one of them needs an edit, that is a signal the deletion went further than intended."
**Fix:** Give the file an explicit disposition in commit 1 (repoint at a surviving file, or delete), and state that commit closure was derived from import graphs only, so kept tests reaching repo paths at runtime need their own check.

### [BLOCKING:design] ErrNoLanguage has no surviving producer
**Section:** Decisions → `root-package-kept-trimmed`; Technical context → doc-comment sites (`errors.go:14-17`)
**Issue:** The rationale says "`toc` imports the root package for exactly those two sentinels", but `toc` references only `ErrLanguageUnsupported` (`toc.go:32,40,45,201`); `ErrNoLanguage`'s only kept references are `treesitter.go:72` (a comment saying WithTree deliberately does *not* use it) and `treesitter_test.go:137`'s negative assertion — so the instruction to restate its doc "in terms of its surviving producer in `toc`" names a producer that will not exist.
**Fix:** State that `ErrNoLanguage` survives with zero producers and why (deleting it would force an edit to `treesitter_test.go`), and fix the doc-rewrite instruction — noting `treesitter.go:71-77` and `treesitter_test.go:123-126` both pin the sentinel's current "detection-only" wording.

### [NIT:consistency] Plan §2 does not permit the table move
**Section:** Decisions → `extension-table-into-toc` (rationale)
**Issue:** §2's kept row reads "`internal/quarryengine/toc`, `treesitter`, the extension→language table in `registry`"; nowhere does the plan permit moving the table into `toc` or `treesitter`.
**Fix:** Drop the false citation; the true rationale ("`toc` is the table's only remaining caller", verified) already carries the decision.

### [NIT:decision] extension_test.go whole-test disposition unstated
**Section:** Testing → `extension_test.go`
**Issue:** "its `.rs`/`.ts`/`.tsx` expectations go" understates `TestLanguageForExtension_IgnoresDirectoryContext` (`extension_test.go:41-46`), whose entire body asserts `.ts` → `"typescript"` and whose premise is the deleted `DetectLanguage`; it goes red in commit 3 rather than merely losing a row.
**Fix:** Name that test explicitly with a delete disposition, alongside the `TestExtensionLanguages` want-list change.

### [NIT:scope] Build precondition absent from the gate
**Section:** Constraints; Decisions → `verification-gate`
**Issue:** `go build ./... && go test ./...` green at all five commits requires `CGO_ENABLED=1` plus a C toolchain (`cgoguard_nocgo.go` fails the build by design otherwise); the README stub text states this, the Constraints and gate sections do not.
**Fix:** State the cgo build precondition once in Constraints so the green gate is unambiguous.

## Verdict

REQUEST_CHANGES
Commit 1 is red as specified, and ErrNoLanguage's kept-doc instruction has no producer.
MILL_REVIEW_END
