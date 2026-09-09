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
- `quarry/text.go` — the one in-repo consumer site (`case r.Status == "":`, line ~513) rewritten to
  use `ResolveResult.Rejected()`.
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
- `Expand`, `TOC`, `Delta`, `Glyphs`, and the CLI/MCP layers. `internal/cli/cli.go:runResolve`
  passes a one-element slice and is unaffected by an additive contract.

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
  before returning. On a violation the verifier panics with a message naming the verb, the index,
  and the mismatch.
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
  string included.
- **Rationale:** Fail-closed is the entire point: a caller writes `if !res.Status.Known() { … }` and
  an unexpected value takes the explicit unknown branch instead of falling through a chain of
  equality tests. The empty string must be false — it is the rejection marker, which `Rejected()`
  names, not a resolution outcome. This also makes `res.Unit.Known()` correctly false when `Unit` is
  absent, since `Unit` is only ever set on a `not_found`.
- **Rejected:** Treating `""` as known. It would let a rejection result slip through a `Known()`
  gate as if it carried an outcome.

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

### Dogfood `Rejected()` at the one in-repo consumer site

- **Decision:** Rewrite `quarry/text.go`'s `case r.Status == "":` (around line 513) as
  `case r.Rejected():`, and update the surrounding doc comment (around line 478) that spells the
  condition out as `r.Status == ""`.
- **Rationale:** It is the single in-repo consumer of exactly the state the helper names, so it
  proves the helper compiles and reads well, and it makes the repository's own code the pattern
  loomyard copies. The neighbouring `r.Status == StatusFound` / `StatusNotFound` / `StatusAmbiguous`
  comparisons at lines 534/538/554 stay as they are — those are positive tests against named
  constants, not fail-open unknown-value handling, and `Known()` has nothing to add to them.
- **Rejected:** Leaving `text.go` untouched. Shipping a helper the producing repository does not use
  is a weaker signal to the consumer than shipping one it does.

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
and delegates. `resolve` is also called directly elsewhere in the package, so verifying there covers
every path:

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

**No CLI or MCP impact.** `internal/cli/cli.go:runResolve` (around line 514) calls
`repo.Resolve([]string{req.target})` — a one-element batch. An additive contract and two new methods
change nothing there. `internal/mcpserver` does not call `Resolve` directly.

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
wrong input each trip the guard; and a zero-length input is a no-op rather than a trip. Testing a
panic needs `defer`/`recover` — the package has no existing panic test, so `internal/engine/strategy.go`'s
duplicate-registration panic and whatever test covers it are the local precedent to check for a
house style before inventing one. Also assert the panic message names the verb and the offending
index, since that message is the only diagnostic an engine bug will produce.

**`internal/engine` — the vocabulary (TDD candidates).** A `Statuses` completeness test mirroring
`TestName_ReasonCompleteness` (`internal/engine/name_test.go:276`): length, no duplicates, no
unexpected values, nothing missing, against a locally-written `want` set — so adding a constant
without adding it to `Statuses` fails. A `Known()` truth table: true for every element of `Statuses`
(range over it rather than restating the four, which is what ties the method to the vocabulary),
false for `Status("")`, and false for at least one plausible-looking bogus value such as
`Status("found ")` or `Status("FOUND")`, to pin that there is no trimming and no case folding.

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

**`quarry` — the facade.** Assert `quarry.Statuses` is the engine's own slice, in the same shape the
existing `NameReasons` facade coverage uses. Assert `Known()` and `Rejected()` are callable through
the facade's aliased types — a compile-level assertion is enough, and it is what proves an external
importer gets the methods without importing `internal/engine`.

**`quarry` — `text.go` regression.** The `text.go` rewrite is behaviour-preserving by construction,
so the existing `quarry/text_test.go` rendering coverage of a rejection result is the guard. Confirm
a rejection-rendering case exists there; if none does, add one before the rewrite so the change is
covered rather than merely believed.

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
