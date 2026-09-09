# Discussion: Batch-answer contract: per-target coverage + fail-closed Status helpers

```yaml
task: 'Batch-answer contract: per-target coverage + fail-closed Status helpers'
slug: batch-answer-contract
status: discussing
parent: main
```

## Problem

Quarry's two batch verbs — `Resolve` and `Name` — already answer every input positionally: `resolve`
allocates `make([]ResolveResult, len(targets))` and fills it in argument order, `Name` allocates
`make([]NameResult, 0, len(decls))` and appends once per declaration, `ResolveResult.Target` echoes
its argument verbatim on every result including rejections, and `NameResult` echoes `Unit` and
`Target` verbatim on every result. Both godocs already promise argument order. The implementation is
correct today.

What does not exist is a **contract**. Nothing states the guarantee normatively in the published
documentation, and nothing verifies it before the answer leaves the producer. Because the guarantee
is only an implementation property, every consumer that depends on it has to re-establish it by
hand. Loomyard carries four such guards — `ensureResolveCoverage`, `matchHandleResults`, the
donecheck per-key guard, and `CanonicalizeHandles`' length-plus-echo check — all four doing the same
work the producer could do once.

The same failure mode appears in the `Status` vocabulary. `Status` is a closed four-value vocabulary
(`found`, `not_found`, `ambiguous`, `multipart`) plus one documented absent state (`Status == ""`,
the pre-resolution rejection carried by `Error`/`Reason` instead). Callers must currently compare
`Status` against string literals, so an unexpected or empty value **fails open** — it silently takes
whatever the last `else` branch does. Five loomyard review findings (crucible-loom-glyph-hardening
R4-14, R5-6, R6-27, R9-6, R10 F1) are the same fail-open `Status` comparison; three of the four
coverage guards trace to review findings (R5-6, R10 F2, R10 F3).

**Why now:** the loomyard adoption round (#226) has just built on the slice shape, and the guards
are accumulating at the consumer instead of being paid for once at the producer. Both defects live
in the same three engine files and the same documentation section, and loomyard consumes both
through a single `go.mod` bump, so they land together and are tagged together. GitHub issues #30 and
#31 both close on this task's merged commit.

## Scope

**In:**

- `internal/engine/resolve.go` — a producer-side coverage verifier for `resolve`, run before the
  result slice is returned.
- `internal/engine/name.go` — the same verifier shape for `Name`.
- `internal/engine/answer.go` — `func (s Status) Known() bool`, `func (r ResolveResult) Rejected() bool`,
  and `var Statuses []Status` enumerating the closed vocabulary.
- `quarry/quarry.go` — `Statuses` surfaced through the facade as the engine's own value (not a copy),
  matching how `NameReasons` is surfaced.
- `quarry/text.go` — the first of two in-repo rejection-state sites (`case r.Status == "":`, line
  ~513) rewritten to use `ResolveResult.Rejected()`.
- `internal/cli/cli.go` — the second rejection-state site (`codeForResolveResult`'s `case "":`,
  lines 85–96) rewritten to use `Rejected()`; and the two consumer-side arity guards
  (`runResolve`, line ~524, and `runName`, line ~661) deleted, since the producer now guarantees
  what they re-check.
- `docs/glyph.md` §5 — the batch coverage contract stated normatively (both batch verbs, plus the
  duplicate-target rule), and a sentence naming the `Status` vocabulary as closed and
  caller-checkable.
- Tests: the verifier's own behaviour, the `Statuses` completeness test, `Known()` and `Rejected()`
  truth tables, and a facade test that `quarry.Statuses` is the engine's own slice.

**Out:**

- Any change to the emitted JSON key set. No new key, no renamed key, no removed key. This is
  purely additive Go API surface plus documentation.
- Any change to `Resolve`'s or `Name`'s exported signatures. `Resolve` keeps
  `([]ResolveResult, error)`; `Name` keeps `[]NameResult` with no error.
- Keyed / map-shaped batch answers (issue #30's option 1). Explicitly rejected — see Decisions.
- `ExpandAnswer` gains no `Rejected()`. See Decisions.
- Loomyard. Deleting its four guards happens in loomyard's own repository, driven by its own
  `go.mod` bump. This task ships the producer-side contract only.
- Cutting the `v0.2.0` tag. The operator cuts it after merge; the task does not tag.
- Closing GitHub issues #30 and #31. The task makes them closeable; the operator closes them with a
  pointer to the merged commit.
- `Expand`, `TOC`, `Delta`, `Glyphs`, and `internal/mcpserver`. The CLI is **not** wholly out —
  `internal/cli/cli.go` is in scope for the two edits listed above. Nothing else in the CLI changes:
  its answer shapes, exit-code semantics, and text/JSON output are untouched.
- `codeForExpandAnswer` (`internal/cli/cli.go`, lines 104–113). See Decisions.

## Decisions

### #30 shape: contract-enforcing slices, not keyed answers

- **Decision:** Keep the positional slice shape. Make the guarantee real by (a) stating it
  normatively in `docs/glyph.md` §5 and (b) verifying it producer-side before the slice is returned.
- **Rationale:** The implementation is already positionally exact; what is missing is enforcement
  and statement, both of which are additive and semver-minor. Loomyard's #226 adoption was just
  built on the slice shape.
- **Rejected:** Issue #30's option 1, keyed answers. It is breaking, and it breaks every consumer
  mid-adoption. A keyed shape also has no answer for a repeated target — the positional shape
  answers a duplicate twice, which is exactly the property a caller mapping answers back to plan
  cards needs.

### Enforcement is a panic from a producer-side verifier

- **Decision:** Each batch verb runs an unexported verifier over its own result slice immediately
  before returning. On a violation the verifier panics. There are two violation kinds and they take
  **two different message shapes**, because an arity violation has no offending index to name:
  - **Arity** (`len(results) != len(inputs)`): the message names the verb and the got/want lengths,
    and no index — e.g. `engine: resolve returned 3 results for 4 targets`.
  - **Echo mismatch** (`results[i]` does not echo `inputs[i]`): the message names the verb, the
    index, and both the got and want echo values — e.g.
    `engine: resolve result 2 answers "a/b#X"; want "c/d#Y"`. For `Name`, whose echo is two fields,
    the message names whichever field diverged, with the same verb-plus-index prefix.
- **Rationale:** A coverage violation is unreachable by construction; if it fires, the engine is
  broken and every answer in that batch is untrustworthy. A panic is the honest signal for an engine
  bug and it works uniformly at both verbs. `Name` returns no error at all — deliberately, per its
  own godoc, because with no I/O nothing can fail batch-wide — so an error return is not available
  there without a breaking signature change. And an error return from `Resolve` would hand the
  caller a condition to check, which is precisely the fifth hand-written guard this task exists to
  eliminate. The repository already panics on an invariant violation of this class:
  `internal/engine/strategy.go:74` panics on a duplicate `Strategy` registration.
- **Rejected:** (a) Return an error from `Resolve` and document-only for `Name` — asymmetric, and it
  re-creates the consumer-side guard. (b) Documentation and tests only, no runtime check — leaves the
  guarantee exactly as unenforced as it is today, which is the defect.

### What the verifier checks

- **Decision:** For `resolve`: `len(results) == len(targets)`, and for every index `i`,
  `results[i].Target == targets[i]`. For `Name`: `len(results) == len(decls)`, and for every index
  `i`, `results[i].Unit == decls[i].Unit && results[i].Target == decls[i].Decl`.
- **Rationale:** These are exactly the two properties a consumer guard re-establishes: arity and
  per-index echo. Both fields are documented as always present on every result, rejection results
  included, so the check is total — there is no result shape it cannot examine.
- **Rejected:** Checking arity only. Arity alone does not rule out a reordering, and
  `matchHandleResults` in loomyard exists specifically because arity alone was not enough.

### One verifier per verb, not a shared generic

- **Decision:** Two small unexported functions, each living in its own verb's file
  (`internal/engine/resolve.go` and `internal/engine/name.go`).
- **Rationale:** The echo fields differ — `ResolveResult` echoes one field, `NameResult` echoes two
  from a struct input. A generic helper parameterised over an `echoes func(In, Out) bool` predicate
  is more machinery than the two short loops it would replace, and it would move the interesting
  part (which fields count as the echo) out of the file that owns the answer shape.
- **Rejected:** A shared `assertPositional[In, Out any]` helper in a new `contract.go`.

### The facade does not re-verify

- **Decision:** `quarry.Resolve`, `(*quarry.Repo).Resolve` and `quarry.Name` keep delegating
  unchanged — no filtering, no re-shaping, no re-checking.
- **Rationale:** "Producer-side" means once, at the engine. Every facade doc comment in
  `quarry/repo.go` and `quarry/name.go` already states the pure-delegation posture; adding a check
  there would be a second implementation of the same guarantee, which is the drift this codebase's
  one-implementation rule exists to prevent.
- **Rejected:** Defence-in-depth re-verification at the facade.

### `ResolveResult.Rejected()` is `Status == ""`

- **Decision:** `func (r ResolveResult) Rejected() bool { return r.Status == "" }`.
- **Rationale:** `answer.go` documents that `Status` and `Error` are never both set, and that
  `Status` is absent only when the target never reached resolution. `Status == ""` is therefore the
  primary marker; `Error != ""` is the derived one. `quarry/text.go:513` already reads it exactly
  this way, which is the in-repo proof that this is the state callers branch on.
- **Rejected:** `r.Error != ""` — derived rather than primary, and it would be false for a
  hypothetical rejection whose message was somehow empty, where `Status == ""` still holds.

### `Status.Known()` is true only for the four blessed values

- **Decision:** `func (s Status) Known() bool` returns true for exactly `StatusFound`,
  `StatusNotFound`, `StatusAmbiguous`, `StatusMultipart`, and false for everything else, the empty
  string included. **Its body is a `switch` over the four constants, not a range over `Statuses`:**

  ```go
  switch s {
  case StatusFound, StatusNotFound, StatusAmbiguous, StatusMultipart:
      return true
  default:
      return false
  }
  ```

- **Rationale:** Fail-closed is the entire point: a caller writes `if !res.Status.Known() { … }` and
  an unexpected value takes the explicit unknown branch instead of falling through a chain of
  equality tests. The empty string must be false — it is the rejection marker, which `Rejected()`
  names, not a resolution outcome. This also makes `res.Unit.Known()` correctly false when `Unit` is
  absent, since `Unit` is only ever set on a `not_found`.

  The switch form is chosen over ranging `Statuses` for two reasons. First, **`Statuses` is exported
  and is a slice, so it is mutable by any caller** — `quarry.Statuses[0] = "nonsense"` is legal Go,
  and under the range form that call would silently rewrite `Known()`'s answer for every consumer in
  the process. A predicate whose truth set can be mutated at a distance is the opposite of
  fail-closed. Second, the switch keeps `Known()` and `Statuses` **two independently written
  enumerations of one vocabulary**, which is exactly what makes the truth-table test meaningful —
  see the next bullet and the Testing section.
- **Consequence for the test, stated so the plan does not write a tautology:** because the two
  enumerations are independent, the test that ranges `Statuses` and asserts `Known()` is true for
  every element is a real cross-check — it fails if a constant is added to the switch but not to
  `Statuses`, or to `Statuses` but not to the switch. Under the rejected range form that same test
  would assert nothing at all, since `Known()` would be reading the very slice the test ranges.
- **Rejected:** (a) Treating `""` as known — it would let a rejection result slip through a
  `Known()` gate as if it carried an outcome. (b) Implementing `Known()` as a range over `Statuses`
  — it makes the predicate caller-mutable and turns the vocabulary test into a tautology.

### `Statuses` is exported and enumerated, mirroring `NameReasons`

- **Decision:** Add `var Statuses = []Status{StatusFound, StatusNotFound, StatusAmbiguous, StatusMultipart}`
  to `internal/engine/answer.go`, in the same order as the constant block, with a doc comment stating
  that adding a constant means adding it here in the same edit. Surface it through the facade as
  `var Statuses = engine.Statuses` — the engine's own slice, not a copy.
- **Rationale:** The task names the `NameReasons` enumerate-in-a-slice pattern explicitly, and
  `NameReasons` is exported, doc-commented with exactly that same-edit rule
  (`internal/engine/name.go`), and aliased through the facade as the engine's own value
  (`quarry/quarry.go`). Go cannot reflect over package-level constants, so the slice is the only way
  a test — or a caller enumerating the vocabulary for a UI or a validator — can range over it. It is
  also what lets one test tie `Known()` to the vocabulary rather than restating four literals.
- **Rejected:** A test-local slice with no exported var. It would leave `Known()`'s truth set and the
  constant block synchronised only by review, and it would deny callers the enumeration
  `NameReasons` gives them for the sibling vocabulary.

### `ExpandAnswer` gains no `Rejected()`

- **Decision:** No `Rejected()` on `ExpandAnswer`.
- **Rationale:** `ExpandAnswer` has no pre-resolution-rejection state to name. It is single-target,
  its `Status` field is a required key (`json:"status"`, no `omitempty`), and a rejected target comes
  back as a returned error, not as an answer with an empty status. A method that is always false
  would be worse than no method.
- **Rejected:** Adding it for symmetry.

### Dogfood `Rejected()` at both in-repo rejection-state sites

- **Decision:** There are **two** in-repo sites branching on the rejection state, not one, and both
  are rewritten:
  1. `quarry/text.go`'s `case r.Status == "":` (around line 513) becomes `case r.Rejected():`, and
     the surrounding doc comment (around line 478) that spells the condition out as `r.Status == ""`
     is updated to name the method.
  2. `internal/cli/cli.go`'s `codeForResolveResult` (lines 85–96) currently switches on `r.Status`
     with an explicit `case "": return exitNegative`. Lift that case out ahead of the switch as
     `if r.Rejected() { return exitNegative }`, leaving a switch over the four named constants plus
     its `default: exitInternal`. Update the function's doc comment, which spells the rejection case
     out in prose, to name the method.
- **Rationale:** Both sites read exactly the state the helper names, so using it in the producing
  repository proves the helper compiles and reads well and makes this repo's own code the pattern
  loomyard copies. Lifting the CLI's rejection case out of the switch also separates the two
  questions the switch currently conflates — "did this target reach resolution at all?" and "what
  outcome did it get?" — which is the same separation `Rejected()` and `Known()` exist to give
  callers.
- **`Known()` is deliberately not used at either site.** Both switches are already total: each ends
  in an explicit `default` that routes an unknown value somewhere safe (`exitInternal` in the CLI).
  A total switch over named constants is already fail-closed, which is the property `Known()` sells;
  rewriting `default:` as `if !s.Known()` would add an indirection without changing behaviour, and
  would leave the compiler with less to check, not more. `Known()` earns its place at a caller that
  *cannot* write a total switch — one branching on a status inside a larger condition, which is the
  loomyard shape — not at one that already has.
- **The neighbouring positive comparisons stay.** `quarry/text.go` lines 534/538/554
  (`r.Status == StatusFound` / `StatusNotFound` / `StatusAmbiguous`) are positive tests against named
  constants, not fail-open unknown-value handling. Nothing here changes them.
- **Rejected:** (a) Leaving both sites untouched — shipping a helper the producing repository does
  not use is a weaker signal to the consumer than shipping one it does. (b) Rewriting the `default`
  arms with `Known()` — behaviour-neutral indirection, as above.

### `codeForExpandAnswer` is left exactly as it is

- **Decision:** No change to `internal/cli/cli.go`'s `codeForExpandAnswer` (lines 104–113).
- **Rationale:** It has no rejection case to lift — consistent with the decision that `ExpandAnswer`
  gains no `Rejected()`, because it has no such state. Its switch over three constants plus a total
  `default: exitInternal` is already fail-closed for the same reason `codeForResolveResult`'s is, so
  `Known()` has nothing to add there either.
- **Rejected:** Touching it for symmetry with its sibling. The sibling changes because it has a
  rejection case; this one does not have one.

### The CLI's own two arity guards are deleted, not kept

- **Decision:** Delete both consumer-side arity guards in `internal/cli/cli.go`: `runResolve`'s
  `if len(results) != 1 { … "resolve returned N results for one target" }` (around line 524) and the
  identical `runName` guard `… "name returned N results for one declaration"` (around line 661).
  Both are followed by an unconditional `results[0]`, which is exactly what the producer contract now
  guarantees for a one-element input.
- **Rationale:** These are in-repo instances of precisely the guard class the Problem section counts
  four of in loomyard. Deleting loomyard's four while the producing repository keeps its own two
  would say the contract is trusted abroad but not at home — and would leave a live counter-example
  in the tree for the next consumer to copy. The producer-side verifier panics on an arity violation
  before any result is returned, so these guards are now unreachable by construction; keeping
  unreachable code is worse than deleting it, because it reads as doubt about the contract.
- **Note for the plan:** `strconv` is imported in `internal/cli/cli.go` (line 13) for these two call
  sites and no others. Deleting both guards makes the import unused, which is a compile error in Go —
  the import must be removed in the same edit.
- **Rejected:** (a) Keeping both because "a CLI should be defensive" — a panic from the producer is
  strictly louder than an `exitInternal` from the consumer, and the guard cannot fire before the
  panic does. (b) Keeping them and documenting them as belt-and-braces — that is the
  compounding-debt pattern this whole task exists to remove.

### `docs/glyph.md` §5 carries both statements, and covers both batch verbs

- **Decision:** Add to §5: (a) a normative paragraph stating the batch answer contract — one answer
  per input, in argument order, with the input echoed verbatim on every answer including a
  rejection, and a repeated input answered once per occurrence — stated once for both batch verbs
  (`resolve` and `name`); (b) a sentence stating that the four statuses above are a closed
  vocabulary a caller can check, with the absent status naming a pre-resolution rejection rather
  than an outcome.
- **Rationale:** §5 is the section the task names and the only home in the published documentation
  for batch answer shapes — the maker has no section of its own, and inventing one for two sentences
  would be worse than a shared statement. Stating the duplicate-target rule in the document (not
  only in godoc) is what makes "positional coverage" unambiguous, and it is the rule a keyed shape
  could not have honoured.
- **Rejected:** Documenting `resolve` in §5 and leaving `name` to godoc — it would leave the
  published contract half-stated for the exact defect family this task closes.

## Technical context

**Files that change**

| File | What changes |
|---|---|
| `internal/engine/answer.go` | `Statuses` var, `Status.Known()`, `ResolveResult.Rejected()` |
| `internal/engine/resolve.go` | coverage verifier + call from `resolve` |
| `internal/engine/name.go` | coverage verifier + call from `Name` |
| `quarry/quarry.go` | `var Statuses = engine.Statuses` |
| `quarry/text.go` | `case r.Rejected():` and its doc comment |
| `docs/glyph.md` | §5 additions |

**The producer sites, verbatim as they stand.**

`internal/engine/resolve.go`, the unexported worker (`resolve`, around line 413) is where the
verifier belongs — not in the exported `Resolve` (around line 396), which only builds the `unitMemo`
and delegates. `resolve` has exactly one production caller — that same exported `Resolve`
(`internal/engine/resolve.go:401`) — plus one white-box test that calls it directly with a
hand-built memo (`internal/engine/resolve_test.go:651`). There are no other production call sites to
hunt for. Verifying in the unexported worker rather than the exported wrapper still earns its place:
it covers the white-box test's own path and any future in-package caller, and it puts the check
beside the loop whose invariant it guards.

```go
func (r *Repo) resolve(targets []string, m *unitMemo) ([]ResolveResult, error) {
	results := make([]ResolveResult, len(targets))
	for i, target := range targets {
		res, err := r.resolveGlyphTarget(target, m)
		if err != nil {
			return nil, err
		}
		results[i] = res
	}
	return results, nil
}
```

Note the early `return nil, err` path: on an engine failure the whole call fails with a nil slice.
The verifier must not run on that path — a nil slice with a non-nil error is the documented failure
shape, not a coverage violation. Verify only on the success return.

`internal/engine/name.go`, `Name` (around line 82):

```go
func Name(decls []Declaration) []NameResult {
	results := make([]NameResult, 0, len(decls))
	for _, d := range decls {
		results = append(results, nameOne(d))
	}
	return results
}
```

`Name` has one return path, so the verifier call is unconditional there.

**Echo fields are documented as always-present.** `ResolveResult.Target` — "Target is the caller's
argument, verbatim. Always present." `NameResult.Unit` / `.Target` — "echoes the input
Declaration.Unit verbatim. Always present." / "echoes the input Declaration.Decl verbatim. Always
present." `nameFailure` (`internal/engine/name.go`) exists precisely so the echo is written once
rather than at each failure site, which is why the failure results echo too.

**Empty-input shapes must not regress.** `Resolve(nil)` returns an empty, non-nil slice and a nil
error — asserted by `TestResolve_ArgumentOrderAndArity` in `internal/engine/resolve_test.go` (around
line 727). `Name` with an empty input returns an empty, non-nil slice, guaranteed by the
`make(..., 0, len(decls))` allocation and stated in both `internal/engine/name.go` and
`quarry/name.go` godoc. A verifier over a zero-length input must be a no-op, not a panic.

**The `NameReasons` pattern to mirror, exactly.** `internal/engine/name.go` declares
`var NameReasons = []string{...}` with a doc comment stating the same-edit rule; `quarry/quarry.go`
re-exports it as `var NameReasons = engine.NameReasons` with a doc comment explaining that it is the
engine's own value, not a copy, "so a caller enumerating the vocabulary and the engine's own test are
reading one slice". `internal/engine/name_test.go:276`'s `TestName_ReasonCompleteness` is the test to
mirror: it checks length, duplicates, unexpected values, and missing values against a locally-written
`want` set. `glyph/errors.go`'s `Reasons` and `glyph/golang_test.go:231`'s
`TestReasons_Completeness` are the older instance of the same pattern.

**The `Status` type's existing doc block** (`internal/engine/answer.go`) already says "The four
Status values docs/glyph.md §5 defines. No other value is valid." The new `Statuses` var and
`Known()` are what make that sentence checkable rather than aspirational. `Status` is also the type
of `ResolveResult.Unit` and `ExpandAnswer.Unit`, which carry only `StatusFound` or `StatusNotFound`
and are absent otherwise — so `Known()` returning false for `""` is load-bearing there too.

**Facade aliasing means the methods surface for free.** `quarry/quarry.go` declares
`type Status = engine.Status` and `type ResolveResult = engine.ResolveResult` — **type aliases**, not
defined types. Methods declared on the engine types are therefore reachable as `quarry.Status.Known`
and `quarry.ResolveResult.Rejected` with no facade work at all. The only facade edit needed is the
`Statuses` var, because a package-level var is not carried by a type alias.

**The `internal` rule and why aliases exist.** Go enforces the internal-import rule on import paths,
not on types reached through an alias, so an external importer (loomyard) can name
`quarry.ResolveResult` and call `.Rejected()` without importing `internal/engine`. This is stated in
every alias doc comment in `quarry/quarry.go` and must not be disturbed.

**`docs/glyph.md` §5** starts at line 192 and runs to line 234 (`## 6.`). It is prose plus a
four-row status table (`found` / `multipart` / `ambiguous` / `not_found`), then paragraphs on
never-guessing, multi-language ambiguity, the `toc`-takes-paths / `resolve`-takes-glyphs split, the
`#`-in-a-path-segment rule, the root-not-addressable rule, and a measured-cost line. The new
vocabulary sentence belongs immediately after the status table (it is about that table); the batch
coverage paragraph belongs with the `toc`/`resolve` paragraph or after it, since that is where the
document talks about what the verbs take and return.

**`glyph/docs_test.go` is the drift guard for §5, but only for examples.** Its header says a row
added to the document without a matching row in its `docsAccept`/`docsReject` tables is the drift it
exists to catch — but those tables hold *parse* cases (glyph strings `glyph.Parse` accepts or
rejects), cited by section. The additions here are normative prose about answer shape and vocabulary,
not new glyph-string examples, so no new row is required. If the plan's wording ends up introducing a
new literal glyph string as an example, that string does need a `docsAccept` row.

**The CLI is a real consumer of both defects, and is in scope.** `internal/cli/cli.go` holds four
relevant call sites, verified against the tree:

- `codeForResolveResult` (lines 85–96) — `switch r.Status` with `case quarry.StatusFound,
  quarry.StatusMultipart: return exitOK`, `case quarry.StatusNotFound, quarry.StatusAmbiguous:
  return exitNegative`, `case "": return exitNegative`, `default: return exitInternal`. The `case
  "":` arm is the rejection state; its doc comment (lines 79–84) already explains that an empty
  status means a pre-resolution rejection and that the `default` is unreachable because the
  vocabulary is closed. This is the site the Decisions rewrite with `Rejected()`.
- `codeForExpandAnswer` (lines 104–113) — the same shape over three constants, with no `case ""`.
  Unchanged, per its own Decision.
- `runResolve` (around line 524) — `if len(results) != 1 { … strconv.Itoa(len(results)) … }`
  immediately before `result := results[0]`. Deleted.
- `runName` (around line 661) — the identical guard, `"name returned "+strconv.Itoa(len(results))+
  " results for one declaration"`. Deleted. The r2 review named only the `runResolve` one; both are
  the same class and both go.

`strconv` (line 13) is imported for those two guards and nothing else — `grep -n strconv
internal/cli/cli.go` matches exactly lines 13, 525 and 661 — so the import is removed in the same
edit or the package will not compile.

`internal/mcpserver` does not call `Resolve` or `Name` directly and is untouched.

**Neighbouring style to follow.** This codebase writes long, load-bearing doc comments that state
*why* a shape is what it is and name the alternative that was rejected. Every new exported symbol
here (`Statuses`, `Known`, `Rejected`) needs a doc comment in that register, and every new
unexported verifier needs one saying what invariant it guards and why a panic rather than an error.
File header comments (the block above `package engine`) describe what the file declares; adding
exported API to `answer.go` may warrant a clause in its header, which currently enumerates the file's
declarations.

## Constraints

- No `CONSTRAINTS.md` at the hub root — no repository-level constraint file to enumerate.
- `CLAUDE.md`: "This is a Go repo. Do not introduce Python."
- Go 1.26 (`go.mod`). Generics are available, but the Decisions above reject a generic verifier on
  readability grounds, not availability.
- **Semver-minor, additive only.** Every change here must compile against existing callers unchanged.
  No exported signature changes, no removed or renamed symbols, no JSON key-set change. This is what
  makes `v0.2.0` a minor tag and what lets loomyard adopt via a plain `go.mod` bump.
- **The emitted key set is closed.** `internal/engine/answer.go`'s file header states that every JSON
  tag in it is the exact emitted key set fixed by a Shared Decision, and that no field is added or
  renamed without a corresponding Shared Decision change. The new methods and the `Statuses` var add
  no field and no tag, so this rule is satisfied without a Shared Decision change — the plan must not
  introduce one that would violate it.
- **One implementation of a rule.** `docs/glyph.md` §6's "there is one implementation of the glyph
  grammar" principle generalises across this codebase: the coverage check lives at the producer,
  once, and the facade does not restate it.
- Gates: `go test ./...` and `golangci-lint run` must both pass.
- Tagging `v0.2.0` and closing issues #30/#31 are operator actions after merge, not task steps.

## Testing

TDD is the right posture for all three new units — the verifier, `Known()`, and `Rejected()` are
small, pure, and fully specified above, so their tests can be written before their bodies.

**`internal/engine` — the coverage verifiers (TDD candidates).** The verifiers are unexported, so
their tests are white-box, in-package, and can call them directly with hand-built slices. Cover: a
correct slice passes; a short slice, a long slice, and a slice whose element at some index echoes the
wrong input each trip the guard; and a zero-length input is a no-op rather than a trip.

Testing a panic needs `defer`/`recover`, and the package already has the house style for it:
`TestRegister_PanicsOnDuplicateLanguage` (`internal/engine/classify_test.go:107`) covers
`strategy.go:74`'s duplicate-registration panic with a deferred `recover()` that fails when
`recover()` returns nil. Follow that shape. Note what it does **not** do: it asserts only that a
panic occurred, never anything about the message. Asserting message content here is a deliberate
step beyond the existing precedent, taken because the panic message is the only diagnostic an engine
bug of this class will ever produce. Assert it accordingly, matching the two message shapes the
Decisions fix: an **arity** trip's message names the verb and the got/want lengths and must *not* be
asserted to contain an index (there is none); an **echo-mismatch** trip's message names the verb, the
offending index, and both echo values.

**`internal/engine` — the vocabulary (TDD candidates).** A `Statuses` completeness test mirroring
`TestName_ReasonCompleteness` (`internal/engine/name_test.go:276`): length, no duplicates, no
unexpected values, nothing missing, against a locally-written `want` set — so adding a constant
without adding it to `Statuses` fails.

A `Known()` truth table: true for every element of `Statuses` — ranged over, not restated as four
literals — plus false for `Status("")` and false for at least one plausible-looking bogus value such
as `Status("found ")` or `Status("FOUND")`, pinning that there is no trimming and no case folding.
The range is a genuine assertion **because `Known()`'s body is a switch, not a range over the same
slice** (see the `Status.Known()` Decision): the switch and `Statuses` are two independently written
enumerations, so this test fails when a constant is added to one and not the other. If a plan writer
were to implement `Known()` by ranging `Statuses`, this test would become a tautology and the
completeness test above would be the only remaining guard — which is exactly why the implementation
form is fixed by decision rather than left open.

**`internal/engine` — `Rejected()` (TDD candidate).** True for a result whose `Status` is empty and
whose `Error` is set; false for a result carrying each of the four statuses. The existing
`TestResolve_SelfForm` in `internal/engine/resolve_test.go` already produces real rejection results
from `Resolve` (a bare path and a doubled separator), so an end-to-end assertion that
`Rejected()` is true for exactly those and false for the resolved ones in the same call is available
without new fixtures.

**`internal/engine` — the existing positional test still passes unchanged.**
`TestResolve_ArgumentOrderAndArity` (`internal/engine/resolve_test.go:725`) already asserts arity,
per-index echo, the repeated-target case, and the `Resolve(nil)` empty-non-nil-slice case. It must
keep passing verbatim; if the verifier is right, it will. `internal/engine/name_test.go`'s existing
batch-semantics coverage is the same check for `Name`.

**`quarry` — the facade.** There is no existing facade-level test to copy: `NameReasons` is covered
only engine-side (`internal/engine/name_test.go:278`), and no test in `quarry/` touches it. So write
the `quarry.Statuses` identity assertion fresh — that `quarry.Statuses` and `engine.Statuses` are the
same slice, not a copy, which is the property `quarry/quarry.go`'s own doc comment claims for
`NameReasons` and will claim for `Statuses`. Add the missing `quarry.NameReasons` twin assertion in
the same test, so the sibling vocabulary the new code is modelled on is finally covered at the
facade too. Also assert `Known()` and `Rejected()` are callable through the facade's aliased types —
a compile-level assertion is enough, and it is what proves an external importer gets the methods
without importing `internal/engine`.

**`quarry` — `text.go` regression.** The `text.go` rewrite is behaviour-preserving by construction,
so the existing `quarry/text_test.go` rendering coverage of a rejection result is the guard. Confirm
a rejection-rendering case exists there; if none does, add one before the rewrite so the change is
covered rather than merely believed.

**`internal/cli` — the exit-code rewrite and the deleted guards.** `codeForResolveResult` is a pure
mapping written to be table-tested directly (its sibling `codeForNameResult`'s doc comment says so
explicitly), so the guard for the `Rejected()` rewrite is a table over all six inputs: the four
statuses, the empty status, and one bogus value reaching `default`. Confirm such a table already
exists in `internal/cli`'s tests and extend it if it is missing a row; the rewrite must be provably
exit-code-neutral, since exit codes are the CLI's contract. For the two deleted arity guards there is
nothing to add — their branches were unreachable — but check whether any existing test asserts the
`"resolve returned N results for one target"` or `"name returned N results for one declaration"`
message, and delete that assertion in the same edit if so, since the message no longer exists.

**Docs.** `glyph/docs_test.go` must keep passing. No new row is expected (the additions are prose,
not glyph-string examples); if the plan's wording introduces a literal glyph string as an example,
that string needs a `docsAccept` row in the same edit.

**Gates.** `go test ./...` and `golangci-lint run`, both clean, are the done bar alongside the two
issues being closeable.

## Q&A log

- **Q:** #30 shape — contract-enforcing slices, or keyed answers? **A:** [auto-pick] Contract-enforcing slices (additive). **Why:** The implementation is already positionally exact, so only statement and enforcement are missing; loomyard's #226 adoption was just built on the slice shape, and keyed answers break every consumer mid-adoption.
- **Q:** How is the coverage contract enforced on violation? **A:** [auto-pick] Panic from a producer-side verifier. **Why:** `Name` returns no error at all by design, so an error return is unavailable there without a breaking change; and an error from `Resolve` would hand callers one more condition to guard, which is the defect. A coverage violation is an engine bug, and the repo already panics on this class at `internal/engine/strategy.go:74`.
- **Q:** One shared generic verifier, or one per verb? **A:** [auto-pick] One small unexported verifier per verb, in its own file. **Why:** The echo fields differ between the two answer shapes; a generic parameterised over an echo predicate is more machinery than the two short loops, and it would move the interesting part away from the file that owns the shape.
- **Q:** Does the `quarry` facade re-verify coverage? **A:** [auto-pick] No. **Why:** Producer-side means once, at the engine; every facade doc comment already states pure delegation, and a second check would be a second implementation of one rule.
- **Q:** Is `Rejected()` defined as `Status == ""` or `Error != ""`? **A:** [auto-pick] `Status == ""`. **Why:** `answer.go` documents `Status` as absent exactly when the target never reached resolution, and `quarry/text.go:513` already branches on precisely that; `Error != ""` is the derived marker, not the primary one.
- **Q:** Export a `Statuses` slice, or keep the enumeration test-local? **A:** [auto-pick] Export `engine.Statuses` and surface it through the facade, with a completeness test. **Why:** The task names the `NameReasons` pattern, which is exported, doc-commented with the same-edit rule, and facade-aliased as the engine's own value; the exported slice is also what lets one test tie `Known()`'s truth set to the constant block.
- **Q:** Does `ExpandAnswer` get a `Rejected()` for symmetry? **A:** [auto-pick] No. **Why:** It has no pre-resolution-rejection state — it is single-target, `Status` is a required key, and a rejected target returns an error; an always-false method is worse than none.
- **Q:** Is `Status("")` "known"? **A:** [auto-pick] No — `Known()` is false for the empty string. **Why:** `""` is the rejection marker that `Rejected()` names, not a resolution outcome; making it known would let a rejection slip through a `Known()` gate as if it carried one. It also makes `res.Unit.Known()` correctly false when `Unit` is absent.
- **Q:** Rewrite the one in-repo consumer (`quarry/text.go`'s `case r.Status == "":`) to use `Rejected()`? **A:** [auto-pick] Yes. **Why:** It is the single in-repo site branching on exactly that state; using the helper in the producing repository is a stronger signal to loomyard than shipping one the producer ignores. The neighbouring positive `== StatusFound` comparisons stay — those are not fail-open unknown-value handling.
- **Q:** Does `docs/glyph.md` §5 cover the `name` batch too, or only `resolve`? **A:** [auto-pick] Both, stated once. **Why:** §5 is the only doc home for batch answer shapes — the maker has no section of its own — and leaving `name` to godoc would leave the published contract half-stated for the same defect family.
- **Q:** State the duplicate-target rule in the document, or leave it in godoc? **A:** [auto-pick] State it in §5. **Why:** It is what makes "positional coverage" unambiguous, it is cheap, and it is the property a keyed shape could not have honoured — so it belongs where the shape decision is published.
- **Q:** How is `Known()` implemented — a switch over the four constants, or a range over the exported `Statuses`? **A:** [auto-pick] A switch over the four constants. **Why:** `Statuses` is an exported, non-copied slice, so the range form would make `Known()`'s truth set mutable by any caller at a distance — the opposite of fail-closed — and it would turn the prescribed truth-table test into a tautology. The switch keeps the method and the slice two independent enumerations, which is what makes that test a real cross-check.
- **Q:** What disposition do the CLI's own status branches get (`codeForResolveResult`, `codeForExpandAnswer`)? **A:** [auto-pick] `codeForResolveResult`'s `case "":` is lifted out as `if r.Rejected()`; `codeForExpandAnswer` is untouched; neither `default` arm is rewritten with `Known()`. **Why:** The rejection case is exactly what `Rejected()` names, so the CLI is a second dogfood site, not an exception. But both switches are already total with an explicit `default: exitInternal`, which is the fail-closed property `Known()` sells — replacing a total switch's default with a method call adds indirection and gives the compiler less to check. `codeForExpandAnswer` has no rejection case at all, consistent with `ExpandAnswer` getting no `Rejected()`.
- **Q:** Do the CLI's own consumer-side arity guards stay or go? **A:** [auto-pick] Both are deleted — `runResolve`'s (line ~524) and `runName`'s (line ~661). **Why:** They are in-repo instances of the exact guard class the task exists to pay for once at the producer; deleting loomyard's four while keeping two at home would say the contract is trusted abroad but not where it is written, and would leave a live counter-example for the next consumer to copy. The verifier panics before any result is returned, so both branches are now unreachable. The r2 review named only the `runResolve` one; both go. `strconv` is imported for these two sites alone and must be removed in the same edit.
- **Q:** What does the panic message contain — the review noted an arity trip has no offending index? **A:** [auto-pick] Two message shapes, specified separately. **Why:** An arity violation genuinely has no index, so its message names the verb plus got/want lengths; an echo mismatch names the verb, the index, and both echo values. One combined spec would have forced the plan to invent an index for a case that has none.
