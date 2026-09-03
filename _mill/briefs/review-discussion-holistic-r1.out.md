MILL_REVIEW_BEGIN
# Review: Delete V1, keep the tree-sitter extractors (T0)

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: claude-opus-5 (Anthropic Claude Opus 5, per runtime metadata)
reviewed_file: _mill/discussion.md
date: 2026-09-03
```

## Findings

### [BLOCKING:design] Rust test inventory misses a green-breaking case
**Section:** Testing → `internal/quarryengine/toc`; Technical context → Traps
**Issue:** The inventory names only `toc_test.go:765` as the Rust-dependent test to delete, but `toc_test.go:548` `TestTOCDir_UnimplementedLanguageOnlyDirectoryIsNonEmpty` writes `main.rs`, calls `TOCDir(dir, "")` and asserts `len(Files) == 1`; once `.rs` leaves the extension table, `toc.go:175-178` skips the file and the test fails, so commit 3 is red — contradicting `five-commits-each-green`. `toc_test.go:312-321` `TestTOCFile_DesignedButUnimplementedLanguage` still passes but only by falling into the unknown-extension branch, leaving a test whose name and comment describe a path that no longer exists.
**Fix:** State the disposition of both tests (and of the now-unreachable `buildDirEntry` no-strategy branch they covered) in the commit-3 edit list, and state how the inventory was derived so a plan writer can re-derive it rather than trust the enumeration.

### [NIT:consistency] treesitter_test.go has no Languages/Supported assertion
**Section:** Testing → `internal/quarryengine/treesitter`
**Issue:** The claim that "the `Languages()` and `Supported()` assertions must then read `{csharp, go, python}`" does not match the file: `treesitter_test.go` contains no call to either; the only `Supported()` assertion is `toc_test.go:803`, and `Languages()` has no test caller at all.
**Fix:** Correct the claim to name `toc_test.go`'s cross-check as the only set assertion, so nobody adds a test to satisfy a sentence.

### [NIT:design] "Repoint or delete" left open for compact_test.go
**Section:** Testing → `compact_test.go`; Traps
**Issue:** Both places say "repoint or remove"/"repoint or delete" the `bad.rs` / `"rust"` rows without choosing; deleting the row also forces the `"# internal/shed (package shed), 5 files"` count and the `want` list at `compact_test.go:46-54` to change, while repointing does not — two materially different diffs.
**Fix:** Pick one and say which, noting the file-count line if deletion is chosen.

### [NIT:scope] Comment-pass trap list narrower than the comment-pass rule
**Section:** `doc-comment-pass-is-narrow`; Traps
**Issue:** The rule ("fix comments naming a deleted package or Rust/TypeScript") also catches sites the trap list omits: `toc/doc.go:1-11` (names `registry` and `internal/cli`), `toc/toc.go:128`, `toc/strategy.go:76`, `errors.go:1-2` and `errors.go:14-17` (name `internal/cli`, `registry`, `DetectLanguage`), and `extension.go:1-7`, which arrives in `toc` still describing package `registry` and the LSP verbs.
**Fix:** Say explicitly that the trap list is illustrative and the rule governs, or extend the list.

## Verdict

REQUEST_CHANGES
One enumerated test set is incomplete, breaking commit 3's green requirement.
MILL_REVIEW_END
