MILL_REVIEW_BEGIN
# Review: Resolve self-target path: per-call dirPackage memo (GH #34) — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewed_file: plan/
date: 2026-10-03
```

## Findings

### [BLOCKING:design] record() rebuilds symbol-less records in Resolve memo
**Location:** Batch 1 / Card 2 (`fileMemo.record`, `buildRecord` steps 1-3, 5), interacting with Card 4 (`symbolsOfDir`'s `func(string) bool { return true }`) and Card 6 tests 1 and 4.
**Issue:** `hasSymbols` is set only inside the step-4 `WithTree` callback, so a non-language, unreadable, invalid-UTF-8 or parse-failed record never has it.
`record` therefore rebuilds such a record on every `wantSymbols` request, even in an `allSymbols` memo.
In Resolve, a self target builds `pkg/README.md` (test 1) or `g/.gitignore` (test 4) with symbols off, then the member target's `symbolsOfDir` asks for symbols on every filtered file, which bumps `builds` to 2.
Card 6's `assertBuilds` then fails on the plan's own fixtures.
The overview's claim that the rebuild path is "unreachable today" is false for the Resolve memo.
**Fix:** Key the reuse test on whether symbols were requested at build time, for example a `withSymbols` flag on the record or `m.allSymbols || !wantSymbols || rec.hasSymbols`, and correct the Shared Decision's unreachability claim.

### [BLOCKING:design] Parse-once tests cannot see the member path's builds
**Location:** Batch 2 / Card 6 tests 1, 3 and 4; Batch 1 / Card 4 (`unitMemo.symbolsOf` call site).
**Issue:** In every fixture, each directory a member target reaches is also reached earlier by a self target.
If `symbolsOf` passed `symbolsOfUnit` a fresh `fileMemo` instead of `m.files`, the member builds would never land in `m.files.builds`, and every `assertBuilds` would still pass.
Card 4 never names the `symbolsOf` call site or says it must pass `m.files`.
The test therefore does not pin the sharing between member and self targets that is the task's headline.
**Fix:** Card 4 should require `symbolsOf` to pass `m.files`.
Card 6 should add a member-only target in a directory no self target reaches, so the expected set includes a file only the member path can build.

### [NIT:consistency] Stale "parses" wording left in resolve.go
**Location:** Batch 1 / Card 4.
**Issue:** `resolve`'s doc comment ("read parses afterwards") and `Resolve`'s "each distinct unit is parsed exactly once per call by the memo" are not in Card 4's edit list, and Card 5's grep does not cover them.
**Fix:** Name both doc comments in Card 4's resolve.go edits.

## Verdict

REQUEST_CHANGES
The record reuse rule double-builds non-language files, and the tests cannot detect an unshared member memo.
MILL_REVIEW_END
