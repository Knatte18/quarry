MILL_REVIEW_BEGIN
# Review: Batch-answer contract: per-target coverage + fail-closed Status helpers

```yaml
duration_s: 195.0
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: claude-opus-5 (per system context; no independent means to verify)
reviewed_file: _mill/discussion.md
date: 2026-09-09
```

## Findings

### [BLOCKING:design] "One in-repo consumer site" is false
**Section:** Scope (In) / Decisions §"Dogfood `Rejected()`" **Issue:** `internal/cli/cli.go:85-96` `codeForResolveResult` branches on exactly the rejection state — `case "": return exitNegative` — and carries a hand-written `default: exitInternal` unknown-value branch, so `quarry/text.go:513` is not the single in-repo site and the dogfood rationale rests on a false premise. **Fix:** State a disposition for `codeForResolveResult` (and `codeForExpandAnswer`, `cli.go:104-113`): rewritten with `Rejected()`/`Known()`, or explicitly left as a total switch, with the reason.

### [NIT:decision] CLI's own coverage guard has no disposition
**Demoted-from:** BLOCKING
**Section:** Scope (Out) — "No CLI or MCP impact" **Issue:** The out-of-scope line covers only `runResolve` passing a one-element slice, but `internal/cli/cli.go:524` (`if len(results) != 1 { … "resolve returned N results for one target" }`) is an in-repo consumer-side arity guard of exactly the class the Problem section says should be paid once at the producer — the discussion counts four such guards and places all of them in loomyard. **Fix:** Say whether this guard stays (and why an in-repo guard is acceptable while loomyard deletes its four) or is removed by this task.

### [BLOCKING:design] `Known()`'s relationship to `Statuses` undecided
**Section:** Decisions §"`Status.Known()`" / Testing §vocabulary **Issue:** Whether `Known()` is a switch over the four constants or a range over the exported `Statuses` var is never decided, and the two differ observably: under the range form the prescribed test ("true for every element of `Statuses` … which is what ties the method to the vocabulary") asserts nothing, and `Known()` becomes mutable by any caller through the exported, non-copied slice. **Fix:** Fix the implementation form and restate what the truth-table test then proves.

### [NIT:consistency] Panic-test precedent claim is wrong
**Section:** Testing §coverage verifiers **Issue:** "the package has no existing panic test" is false — `internal/engine/classify_test.go:107` `TestRegister_PanicsOnDuplicateLanguage` is in package `engine` and covers `strategy.go:74` with `defer`/`recover`; it asserts only that a panic occurred, never its message. **Fix:** Cite that test as the house style and note that asserting message content is a deliberate step beyond it.

### [NIT:consistency] Panic message spec has no index for arity trips
**Section:** Decisions §"Enforcement is a panic" / Testing **Issue:** The message is specified as naming "the verb, the index, and the mismatch", and the test requirement asserts verb-plus-index for every trip — but the short-slice and long-slice violations have no offending index. **Fix:** Specify the arity-violation message separately (verb plus got/want lengths).

## Verdict

REQUEST_CHANGES
CLI status branching and its arity guard need dispositions; `Known()`'s form is undecided.
_Note: 1 finding(s) demoted from BLOCKING to NIT by the stage's blocking-class ceiling; current blocking_count is 2._
MILL_REVIEW_END
