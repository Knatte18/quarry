MILL_REVIEW_BEGIN
# Review: Engine core (T3) — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: "self-assessed Claude Opus-class model; harness reports claude-opus-5, which I cannot independently confirm"
reviewed_file: plan/
date: 2026-09-03
```

## Findings

### [BLOCKING:scope] Card 5 omits the moved tests' import retargets
**Location:** batch 1 / card 5
**Issue:** `toc_test.go` imports both `internal/quarryengine` (line 22, used at line 306 for `quarryengine.ErrLanguageUnsupported`) and `internal/quarryengine/treesitter` (line 23), and `golang_test.go` imports the latter (line 13); after cards 2–3 neither package path exists, but card 5's Requirements prescribe only a package-clause change plus the `toc_integration_test.go` path fix.
**Fix:** State in card 5 that the `quarryengine` import is dropped and `ErrLanguageUnsupported` referred to unqualified, and that the treesitter import becomes `internal/engine/treesitter` in both test files.

### [BLOCKING:scope] fakeStrategy is never updated for the new Strategy interface
**Location:** batch 3 / cards 14, 20 and batch 4 / cards 24, 29
**Issue:** `classify_test.go:97-104` declares `fakeStrategy` implementing the full `Strategy` interface; card 14 adds `PackageDoc` and drops the `known` returns, and card 24 changes `Symbols` to take `unit`, but card 20 says only "calls to `Generated` and `TestFile` now take one return value" and card 29 does not edit `classify_test.go` at all — so both batches fail `go build ./...`.
**Fix:** Add to card 20 that `fakeStrategy` gains a `PackageDoc` method and the one-value `Generated`/`TestFile`, and to card 29 that its `Symbols` gains the `unit` parameter (adding `classify_test.go` to card 29's `Edits:`).

### [BLOCKING:decision] Ported toc_test.go keeps t.TempDir, against a Shared Decision
**Location:** batch 3 / card 20 (Shared Decision "fixture trees split by whether they involve .gitignore")
**Issue:** the decision states "`t.TempDir()` is never used" and the system temp directory is banned by the task's constraints, yet `toc_test.go` calls `t.TempDir()` 19 times (including the `writeTempFile` helper at line 29) and card 20 never says to replace them; card 20's `Context:` also omits `internal/engine/scratchtree_test.go`.
**Fix:** State in card 20 that every `t.TempDir()` fixture in `toc_test.go` is rebuilt through `writeScratchTree`, and list `scratchtree_test.go` in its `Context:`.

### [BLOCKING:design] Round trip compares a File the walk never sets
**Location:** batch 6 / card 38 (against batch 4 / card 23)
**Issue:** card 23 fixes `Symbol.File` as empty inside a `toc` answer, filled only by the span lookup; card 38 asserts equality of `(File, Start, SigEnd, End)` tuple sets between the walk side and `symbolsOfUnit`, so the walk side contributes `File: ""` and the two sets can never be equal.
**Fix:** State that the walk-side tuple's `File` is composed from the enclosing `DirAnswer.Dir` plus the `FileEntry.Name`, forward-slash joined, before the comparison.

### [BLOCKING:consistency] Card 10 forward-depends on card 11
**Location:** batch 2 / cards 10, 11
**Issue:** card 10 populates `extensionHeaderRules map[string]headerRule` while stating that `headerRule` "is a named function type declared in `headers.go` (card 11)" — the type does not exist when card 10 lands, and card 10's `Context:` cannot list a file that does not yet exist.
**Fix:** Swap the two cards so `headers.go` (the type and the rules) lands before the tables that reference them, and put `internal/engine/headers.go` in the tables card's `Context:`.

### [NIT:scope] Card 3 cannot read the comment it must merge
**Location:** batch 1 / card 3
**Issue:** the card rewrites `doc.go`'s comment as "the merge of the two old ones", but the second old one is `internal/quarryengine/toc/doc.go`, present only as a `Deletes:` token and absent from `Context:`; `sentences.go` is listed instead, and nothing in the Requirements needs it.
**Fix:** List `internal/quarryengine/toc/doc.go` in `Context:`.

### [NIT:scope] Round-trip cards omit ignore.go from Context
**Location:** batch 6 / cards 38, 39
**Issue:** `symbolsOfUnit(unit string, ig *ignoreSet)` requires the caller to build and extend an ignore set, but neither card lists `internal/engine/ignore.go`, so `newIgnoreSet`/`extend` are cold-start exploration.
**Fix:** Add `internal/engine/ignore.go` to both cards' `Context:`.

### [NIT:scope] writeScratchTree cannot build the trees three cards require
**Location:** batch 2 / card 9; batch 3 / cards 21, 22; batch 5 / card 34
**Issue:** the helper's signature is `files map[string]string`, yet card 21 needs symlinks (to a file, to a directory, and a cycle) and card 22 needs an unreadable file, and no card extends the helper or says the test creates them on the returned path.
**Fix:** State in card 9 (or in cards 21/22) that symlink and permission fixtures are created by the test on the path `writeScratchTree` returns.

### [NIT:consistency] D-numbers are unresolvable from the plan
**Location:** all batches (Batch Scope sections and cards 12, 21, 22, 25, 30)
**Issue:** D1–D22 are cited throughout, but the plan's own `## Shared Decisions` carries named decisions with no D-numbers and no card's `Context:` includes `_mill/discussion.md`, so an implementer cannot resolve a reference such as "the two cases D8 turns on".
**Fix:** Either drop the D-numbers in favour of the restated rule each card already carries, or list the discussion file in `Context:` where a D-number is load-bearing.

### [NIT:design] The target directory's .gitignore is extended twice
**Location:** batch 3 / cards 17, 18
**Issue:** card 18 extends the set "along the root-to-target chain" and card 17's `walkDir` extends `dirRel` again on entry; whether the chain stops at the target's parent or includes the target is unstated. Behaviour is unchanged (last-match-wins over a duplicated ordered set) but the trim accounting is ambiguous.
**Fix:** State that the chain covers root through the target's parent and that `walkDir` owns the target's own `.gitignore`.

## Verdict

REQUEST_CHANGES
Five blocking gaps: three break the build, one makes the headline round trip unsatisfiable.
MILL_REVIEW_END
