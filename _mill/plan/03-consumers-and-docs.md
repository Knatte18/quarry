# Batch: consumers-and-docs

```yaml
task: 'Batch-answer contract: per-target coverage + fail-closed Status helpers'
batch: consumers-and-docs
number: 3
cards: 4
verify: go test ./internal/cli/ ./quarry/ ./glyph/
depends-on: [1, 2]
```

## Batch Scope

This batch spends what batches 1 and 2 built. It rewrites both in-repository sites that branch on
the rejection state to use `ResolveResult.Rejected()`, deletes the two consumer-side arity guards
the producer verifiers made unreachable, and states both contracts normatively in `docs/glyph.md`
§5. It is one batch because every card here is a consumer of the same two producer-side changes and
none of them adds new API — this is the dogfooding-and-publishing batch, and splitting it would
leave the repository shipping helpers it does not use.

It depends on batch 1 for `Rejected()` (cards 7 and 8 do not compile without it) and on batch 2 for
the verifiers (card 9 deletes guards that are only safe to delete once the producer panics first).

Batch-local decision, stated because it is a deliberate omission rather than an oversight:
`Status.Known()` is not used at either rewritten site, and neither `default:` arm is rewritten with
it. Both switches are already total, ending in an explicit `default` that routes an unknown value
somewhere safe, which is the fail-closed property `Known()` sells. Replacing a total switch's
`default` with a method call would add indirection and give the compiler less to check. `Known()`
earns its place at a caller that cannot write a total switch — a caller branching on a status inside
a larger condition, which is the loomyard shape — not at one that already has.

## Cards

### Card 7: dogfood `Rejected()` in `RenderResolveText`

- **Context:**
  - `internal/engine/answer.go`
- **Edits:**
  - `quarry/text.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  In `quarry/text.go`, in `RenderResolveText`'s branch switch, replace the first case arm
  `case r.Status == "":` with `case r.Rejected():`. The arm's body is unchanged — this rewrite is
  behaviour-preserving by construction, since `Rejected` is defined as exactly that comparison.

  Update the surrounding doc comment: its numbered branch list opens item 1 by spelling the
  condition out as `r.Status == ""`. Rewrite that opening so it names the method instead, and say in
  the same clause that the method is the engine's own name for the pre-resolution-rejection state,
  so the renderer and the engine read one spelling rather than two.

  Change nothing else in this file. In particular the neighbouring positive comparisons against
  `StatusFound`, `StatusNotFound` and `StatusAmbiguous` further down the same function stay exactly
  as they are: those are positive tests against named constants, not fail-open unknown-value
  handling, and they are not what this task is closing.
- **Commit:** `refactor(quarry): read the rejection state through ResolveResult.Rejected`

### Card 8: lift the CLI's rejection case out of `codeForResolveResult`

- **Context:**
  - `internal/engine/answer.go`
  - `quarry/quarry.go`
- **Edits:**
  - `internal/cli/cli.go`
  - `internal/cli/cli_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  In `internal/cli/cli.go`, rewrite `codeForResolveResult`. Lift its `case "": return exitNegative`
  arm out ahead of the switch as an early `if r.Rejected() { return exitNegative }`, leaving a switch
  over the four named constants — `quarry.StatusFound` and `quarry.StatusMultipart` returning
  `exitOK`, `quarry.StatusNotFound` and `quarry.StatusAmbiguous` returning `exitNegative` — plus its
  existing `default: return exitInternal`. Remove the now-empty `case "":` arm. The function's
  behaviour for all six inputs is unchanged; only the shape is.

  Update the function's doc comment, which currently spells the rejection case out in prose. It must
  name the method, and it must say why the case moved rather than merely that it did: the early
  return separates the two questions the single switch conflated — whether this target reached
  resolution at all, and what outcome it got — which is the same separation `Rejected()` exists to
  give callers. Keep the existing sentence explaining that the `default` is unreachable because the
  vocabulary is closed and exists so a value the engine never produces cannot silently route to a
  zero exit code.

  Leave `codeForExpandAnswer` in the same file untouched. It has no rejection case to lift, matching
  the decision that `ExpandAnswer` gains no `Rejected()` because it has no such state, and its own
  total switch is already fail-closed for the same reason.

  In `internal/cli/cli_test.go`, extend `TestCodeForResolveResult`. It has five rows today — the four
  statuses and the empty status — and no row exercising the `default` arm. Add a sixth row named for
  the bogus-value case, passing a `quarry.ResolveResult` whose `Status` is a value the engine never
  produces, and expecting `exitInternal`. With that row the table covers all six inputs the doc
  comment describes, which is what makes the rewrite provably exit-code-neutral — exit codes being
  the CLI's own contract.
- **Commit:** `refactor(cli): lift codeForResolveResult's rejection case out of the switch`

### Card 9: delete both consumer-side arity guards

- **Context:**
  - `internal/engine/name.go`
  - `internal/engine/resolve.go`
- **Edits:**
  - `internal/cli/cli.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Delete both consumer-side arity guards in `internal/cli/cli.go`:

  1. In `runResolve`, the `if len(results) != 1` block returning the internal-error message
     `"internal error: resolve returned "` … `" results for one target"`, which sits immediately
     before `result := results[0]`.
  2. In `runName`, the identical guard returning `"internal error: name returned "` …
     `" results for one declaration"`, again immediately before `result := results[0]`.

  In both functions the unconditional `results[0]` that follows stays. It is now guaranteed by the
  producer contract batch 2 added: `verifyResolveCoverage` and `verifyNameCoverage` panic on an
  arity violation before any slice is returned, so both deleted branches were unreachable by
  construction. Keeping unreachable code here would be worse than deleting it, because it reads as
  doubt about a contract this repository itself publishes.

  `strconv` is imported at the top of `internal/cli/cli.go` for those two call sites and nothing
  else. Deleting both guards makes the import unused, which is a compile error in Go, so remove the
  `strconv` import line in this same edit. Verify with a grep for `strconv` over the file after the
  edit that no occurrence remains.

  `Run`'s own long doc comment, further up the same file, documents both deleted guards as numbered
  pipeline steps, and both paragraphs must be disposed of in this same edit or the comment will
  describe code that no longer exists:

  1. `runResolve`'s numbered pipeline carries a step reading "A returned slice whose length is not
     exactly one is exit 3, named with the count — the facade contracts a positional one-to-one
     mapping, so this is unreachable and is stated so a contract change cannot silently produce a
     zero exit code." Delete that step and close the numbering up, so the list runs 1..5 with no
     gap.
  2. `runName`'s own numbered pipeline opens with a step that folds the identical sentence into its
     first item. Strike that sentence from the item, leaving the item's surviving instruction — call
     the facade with a one-element slice and take the single result — intact.

  In place of the deleted prose, add one sentence to the surrounding comment stating that the single
  result is taken unconditionally because the engine's own `verifyResolveCoverage` and
  `verifyNameCoverage` panic on an arity violation before any slice is returned, so the CLI has no
  arity condition of its own left to check. Say it once, in whichever of the two paragraphs reads
  more naturally, rather than twice.

  Renumbering `runResolve`'s list invalidates no cross-reference in this comment, and nothing else
  in it needs touching. The four paragraph openers reading "continuing from step 4 above" — one each
  for `runTOC`, `runResolve`, `runExpand` and `runDelta` — all cite `Run`'s own shared step 4,
  "Resolve the repository root by calling internal/repopath.ResolveRoot", not any step of
  `runResolve`'s inner list. Leave all four exactly as they are. Do not renumber them to step 3.

  No test change is needed here. Neither deleted message is asserted anywhere in the repository's
  tests — a grep over every `.go` file for `results for one target` and `results for one
  declaration` matches only these two lines in `internal/cli/cli.go` itself. Re-run that grep after
  the deletion and confirm it matches nothing.
- **Commit:** `refactor(cli): delete the arity guards the producer contract makes unreachable`

### Card 10: state both contracts in `docs/glyph.md` §5

- **Context:**
  - `glyph/docs_test.go`
  - `internal/engine/name.go`
  - `internal/engine/resolve.go`
- **Edits:**
  - `docs/glyph.md`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Add two things to §5 of `docs/glyph.md`, the section headed `## 5. Resolution`, which runs from
  its heading to the `## 6.` heading below it.

  1. **The closed-vocabulary sentence**, placed immediately after the four-row status table, because
     it is about that table. It states that the four statuses above are a closed vocabulary a caller
     can check rather than a set of strings to compare against, and that an absent status names a
     pre-resolution rejection of the target string itself rather than a resolution outcome.

  2. **The batch coverage paragraph**, placed with or immediately after the paragraph that begins
     `toc` takes paths; `resolve` takes glyphs — that is where the document already talks about what
     the verbs take and return.

     This paragraph is the first place in the whole document to mention the `name` verb: the
     document names only `toc`, `resolve` and `expand` today, and the one other occurrence of the
     word — §3's Python spelling rule — is an identifier, not the verb. So open the paragraph with
     one introductory clause defining what `name` is before any claim is made about it: the maker
     verb, which takes a unit plus a declaration head and predicts the id and kind that declaration
     will get once written, without reading the repository. Keep it to a single clause — a full
     section for the maker is out of this task's scope, and the contract sentence that follows is
     what the task is here to publish.

     The paragraph then states, once, for both batch verbs, that on a call that returns
     answers at all there is one answer per input, in argument order, with the input echoed verbatim
     on every answer including a rejection, and that a repeated input is answered once per
     occurrence. The duplicate-target rule is stated here in the document, not left to godoc: it is
     what makes "positional coverage" unambiguous, and it is the property a keyed answer shape could
     not have honoured.

     Immediately beside it, in one sentence, state the whole-call failure exception explicitly:
     `resolve` can fail the entire call instead of answering, and an engine failure then returns no
     answers at all rather than a partial or padded slice, while a malformed target taints only its
     own answer; `name` has no such path and never fails batch-wide. Without this sentence the
     unqualified rule would be false on the error path, and a reader taking it as unconditional
     would be entitled to assume a non-nil slice after an error — exactly the fail-open reading this
     task exists to close. The distinction is already load-bearing in `resolve`'s own godoc in
     `internal/engine/resolve.go`, and `name`'s no-error posture in `internal/engine/name.go`, so
     this restates a rule the code already keeps rather than inventing one.

  Match the document's existing prose register: plain declarative sentences, backticked identifiers,
  no bullet lists in this section, and lines wrapped to the width the surrounding paragraphs already
  use. Change no existing sentence and no table row — this is purely additive.

  Do not introduce a literal glyph string as a new example. `glyph/docs_test.go` is §5's drift guard,
  and its `docsAccept` and `docsReject` tables hold parse cases cited by section; prose about answer
  shape and vocabulary needs no row there, but a new literal glyph string would need a `docsAccept`
  row in this same edit. Keeping the additions free of glyph-string examples is the simpler
  discipline and is what this card requires.
- **Commit:** `docs(glyph): state the batch answer contract and the closed status vocabulary`

## Batch Tests

`verify: go test ./internal/cli/ ./quarry/ ./glyph/` runs the three packages this batch touches.

- `./internal/cli/` covers card 8's extended `TestCodeForResolveResult` table — all six inputs,
  which is what proves the rewrite exit-code-neutral — and re-runs the CLI's end-to-end `Run` tests,
  which are the guard that card 9's deletions changed no observable behaviour.
- `./quarry/` covers card 7. `quarry/text_test.go`'s `TestRenderResolveText` already holds three
  rejection-rendering rows — `RejectionWithReasonAndError`, `RejectionEmptyReason`, and
  `RejectionEmptyError_TotalityGuard` — so the rewritten branch is covered by existing assertions
  and no new row is required. Confirm those three rows are present and passing rather than assuming
  it; if any is missing, add it before the rewrite so the change is covered rather than believed.
- `./glyph/` covers card 10 by way of `glyph/docs_test.go`, §5's drift guard, which must keep
  passing across the documentation edit.

The scope is three packages rather than the whole module because no card here touches
`internal/engine`, and the two producer packages were already re-run in full by batches 1 and 2. The
overview's module-wide `go vet ./...` runs at this batch's boundary, and `pipeline.done_gate` runs
`go test ./... && golangci-lint run` once before the task is marked done — which is where the
deleted `strconv` import and any unused-symbol fallout from card 9 would surface if the compiler
somehow did not catch it first.
