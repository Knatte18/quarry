MILL_REVIEW_BEGIN
# Review: Engine core (T3) — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: claude-opus-5 (Anthropic Claude Opus-class; ID per environment, not independently verifiable from inside)
reviewed_file: plan/
date: 2026-09-03
```

## Findings

### [BLOCKING:scope] Card 30 uses SpansOf a batch before it exists
**Location:** batch 4 / card 30 (`glyph_test.go`)
**Issue:** Requirements end with "assert ... that `SpansOf` returns nothing for any name in either", but `SpansOf` is first declared in batch 5 / card 32; batch 4's own Batch Scope external-interface list does not include it, and `internal/engine` will not compile under batch 4's `verify:`.
**Fix:** Drop the `SpansOf` clause from card 30 and re-assert it in batch 5 / card 34, or move that assertion into a card that depends on batch 5.

### [BLOCKING:consistency] HeadStart/HeadEnd defined two different ways
**Location:** batch 4 / cards 23, 26, 28
**Issue:** Card 23 says `HeadStart`/`HeadEnd` "for Go equal the type declaration's own span"; card 26 says "set `HeadStart`/`HeadEnd` on both to the same `Start`/`End` the type symbol already carries" — but `Symbol.Start` is the first line of the attached comment block (`goUngroupedTypeSymbol`/`goGroupedTypeSymbol` in `golang.go`), not the declaration's own first line. The two prescriptions differ for every documented type.
**Fix:** Pick one and state it in both cards — either the declaration node's own `Line(decl)` pair, or `Start`/`End` including the doc block — and say which the expand verb's head render expects.

### [BLOCKING:design] The walk parses every Go file up to three times
**Location:** batch 3 / card 17 (`dirPackage`, `dirDoc`, `fileEntry`)
**Issue:** `dirPackage` parses every `.go` file in the directory for its clause, `dirDoc` re-parses the `pkg`-matching candidates for `PackageDoc`, and `fileEntry` parses each file again for header/symbols; no card says to reuse a parse or a clause. Card 38 treats parse cost as load-bearing ("would land in the minutes and inside reach of the default test timeout"), and card 39 runs the whole Loomyard tree through this walk, so the 3x is not a free choice.
**Fix:** State in card 17 whether one `treesitter.WithTree` per file feeds all three consumers, or record the repeat parse as an accepted cost with the reason.

### [BLOCKING:decision] The Loomyard goldens have no disposition without a checkout
**Location:** batch 6 / card 36
**Issue:** `testdata/loomyard/render-dir.json` and `render-layout-file.json` are unconditional `Creates:`, but they can only be produced by running `-update` against a `LADDER_LOOMYARD_REPO` checkout — and the batch's own Batch Tests calls "a machine with no checkout ... the normal case". The card never says what the implementer does when the gate skips.
**Fix:** State the disposition explicitly: either the card is blocked without a checkout, or it commits no goldens and the comparison cases skip on an absent file.

### [NIT:scope] Card 20's toc_test.go inventory is incomplete
**Location:** batch 3 / card 20
**Issue:** The delete list names three cases and the port list eleven, leaving `TestTOCFile_GoFileWithSymbols` and `TestTOCFile_NonexistentPath` unaccounted; the latter's subject (`TOCFile` returning `os.ErrNotExist`) is deleted, and its successor lives in card 21's `repo_test.go`, so "rewrite every surviving case" is ambiguous for it.
**Fix:** Name both explicitly — port the first, delete the second as superseded by card 21.

### [NIT:consistency] Stale-comment instructions do not match the source
**Location:** batch 1 / cards 2, 4; batch 2 / card 11
**Issue:** Card 2 says to update `treesitter.go`'s "file header comment's reference to the old package path" — that header carries no package path, only the stale phrase "the parsing backend for the toc verbs". `FirstParagraph`'s doc comment in `text.go` ("Both TOCFile and TOCDir call FirstParagraph...") is never updated by any card although both entry points die in batch 3, and `extension.go`'s header ("all views over the single map below") is left claiming one map after card 11 adds two.
**Fix:** Name each of these three comments in the card that invalidates it.

### [NIT:consistency] The root's .gitignore is extended twice in the SpansOf recipe
**Location:** batch 5 / card 32; batch 6 / card 38
**Issue:** The caller builds `newIgnoreSet(root)` plus one `extend(".")`, then `symbolsOfUnit` "extends that set down the chain from the root to that directory **inclusive**" — read literally the root's own patterns land in the set twice. Harmless for match polarity, but the surrounding prose claims exact accounting.
**Fix:** Say the chain starts at the first directory *below* the root.

### [NIT:scope] Card 17 does not say whether dirPackage sees filtered entries
**Location:** batch 3 / card 17
**Issue:** `dirPackage(dirRel, entries []os.DirEntry)` is handed entries by `walkDir`, but the card never states whether `ig.match` filtering happens before that call — so a gitignored `.go` file could vote in the package tie-break. Card 32 makes exactly this point explicit for `symbolsOfUnit`.
**Fix:** State that `dirPackage` and `dirDoc` see the already-filtered entry set.

## Verdict

REQUEST_CHANGES
One forward dependency, one contradicted field, one unpriced parse cost, one undecided artifact.
MILL_REVIEW_END
