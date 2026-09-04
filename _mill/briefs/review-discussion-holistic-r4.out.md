MILL_REVIEW_BEGIN
# Review: resolve + expand (T4)

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (anthropic), high reasoning effort
reviewed_file: /home/knatte/Code/quarry/wts/resolve-expand/_mill/discussion.md
date: 2026-09-04
```

## Findings

### [BLOCKING:consistency] D14 collision row says "any count", D6 says otherwise
**Section:** D14 (disposition table, row 2) vs D6 / D14 row 1
**Issue:** Row 2 is keyed on "**any count**, `unitDirs` reported `collision`" → `ambiguous` with `Candidates` set, while row 1 sends zero matches to `not_found` and D6 explicitly rejects "`ambiguous` with zero matches (there is nothing to be ambiguous between, and it would hide the `unit: found` a Create card needs)"; a plan writer taking the collision row at its word emits `ambiguous` with an empty `Candidates` for the §8.1 Create case.
**Fix:** Restate row 2 as "at least one match, `unitDirs` reported `collision`" and say the zero-match row is evaluated first, so `Expand` matches D6's zero-match disposition rather than contradicting it.

### [BLOCKING:design] Testing 16's ordering fixture rests on a false premise
**Section:** Testing 16 (and D17's last fixture row)
**Issue:** The fixture is "a `.scratch/` tree whose files are created in reverse order, so `os.ReadDir` does not hand them back lexicographically" — but `symbolsOfDir` (`wts/engine-core/internal/engine/resolve.go:183`), `walkDir` (`walk.go:320`) and `TOC` (`toc.go:116`) all call the package function `os.ReadDir`, which always returns entries sorted by filename; creation order cannot perturb it, so the test asserts a property that holds by construction and can never fail.
**Fix:** Name a fixture where the engine's sort is actually load-bearing — e.g. the collision union, where `symbolsOfUnit` appends the literal `foo_test/` directory before `foo/` and only the final sort restores file order — or state that the ordering guarantee is asserted, not the read order.

### [NIT:design] The `HeadStart == 0` invariant failure has no error type
**Section:** D12
**Issue:** `Expand` "returns an error naming the id" for a `KindType` symbol with `HeadStart == 0`, but no type or sentinel is named, while D14 gives `*NotATypeError` a struct precisely so T5b can map it to `ok`/status/exit code without string matching (T3's D20 pattern).
**Fix:** Either name a typed error for the invariant violation or state that it is deliberately an untyped `fmt.Errorf` because T5b maps it to the generic failure exit code.

### [NIT:consistency] Scope's "Nothing else under `internal/engine` is touched" is contradicted
**Section:** Scope (Out) vs Scope (In) and Technical context's file table
**Issue:** The Out list closes the exception set at "exactly two" and ends "Nothing else under `internal/engine` is touched", yet the In list and the file table modify existing T3 files `answer.go`, `resolve.go` and `resolve_test.go`; a plan writer could read the sentence as requiring an exception card for extending `resolve_test.go`.
**Fix:** Scope the sentence to behaviour ("nothing else under `internal/engine` changes behaviour; additions to `answer.go`, `resolve.go` and `resolve_test.go` are the task itself").

### [NIT:design] `dirExists`'s Lstat only refuses a final-component symlink
**Section:** Provenance (`unitDirs` bullet) / D10
**Issue:** The discussion asserts `dirExists` is "an `os.Lstat` test (deliberately not `os.Stat`, so a symlinked directory is not a hit and the inverse agrees with the walk it inverts)"; `Lstat` still resolves *intermediate* components, so a unit like `link/pkg` under a symlinked `link/` is a hit even though `walkDir` never descends it — making D10's published `unit: found` and a `found` status reachable for declarations `toc` never lists.
**Fix:** Either narrow the claim to the final component and record the intermediate-symlink case as a third D18 contract gap, or state that T4 inherits the behaviour unchanged as T3's.

## Verdict

REQUEST_CHANGES
One contradictory `expand` collision row and one ordering test that cannot fail.
MILL_REVIEW_END
