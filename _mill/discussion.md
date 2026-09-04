# Discussion: resolve + expand (T4)

```yaml
task: resolve + expand (T4)
slug: resolve-expand
status: discussing
parent: main
```

## Problem

Quarry's rewrite is built on one identifier — the glyph — and three queries. T3 (engine core) gives
the engine a symbol model that knows its glyph, its owner chain and its head span, one recursive
directory answer, and an internal span lookup (`SpansOf`) that inverts the walk: glyph in,
declarations out. What T3 deliberately does not give it is a *vocabulary*. `SpansOf` returns an
empty slice for a miss, not `not_found`; it records a unit that names two directories on an
unexported helper's second return rather than calling it `ambiguous`; it says nothing about whether
the unit exists; and it has no answer at all for a target that is a path rather than a glyph.

T4 is the task that turns that lookup into the two public verbs plan §5 specifies. `resolve` is the
query Loomyard's plan validator calls before any agent is dispatched (§8.1) and that an implementer
calls before every read and every edit, because lines move. Its whole value is in the distinctions
its status vocabulary makes: a Create card needs `not_found` **with** `unit: found`, and a typo in
the unit needs `not_found` with `unit: not_found` — "the two are not the same finding" (§5). `expand`
is how a large type is worked on without reading the file it lives in: the type's head plus every
member, across files, one line each.

**Why now.** T4 sits in wave 3 with T5a. It is off the critical path to the measurement — T7 needs
only `toc` over MCP — but T5b (the facade and CLI for both verbs, wave 4) is blocked on it, and T5b
is what Loomyard's validator consumes. T4 is also the last task that can settle these two answer
shapes cheaply: T5a pins the envelope around them in the same wave, and every shape decision made
after that is a change to a published contract rather than a choice.

## Scope

**In:**

- `Repo.Resolve(targets []string) ([]ResolveResult, error)` in `internal/engine/resolve.go`: the
  four-value status vocabulary (`found`, `not_found`, `ambiguous`, `multipart`), `unit: found|not_found`
  on a glyph miss, targets without `#` answered as repository-relative paths, one result per
  argument in argument order.
- The result types `ResolveResult`, `ExpandAnswer` and `Status`, with their JSON tags, in
  `internal/engine/answer.go` beside T3's `Symbol`/`DirAnswer`/`FileEntry`.
- `Repo.Expand(target string) (ExpandAnswer, error)` in a new `internal/engine/expand.go`: the Go
  head plus every glyph whose owner chain begins with the type, across the unit's files, type
  glyphs only.
- `*NotATypeError`, the typed failure `expand` returns for a glyph naming any other kind.
- The per-call unit memo that makes §5's "each unit is parsed once" true for a many-glyph call, and
  the test that proves it.
- Ordering guarantees on both verbs, stated and tested.
- Fixtures under `internal/engine/testdata/` covering every status in `docs/glyph.md` §5, including
  a build-tag `ambiguous` pair and the `_test`-directory collision.
- A Loomyard timing test asserting §12's "twenty glyphs across five units under 150 ms on this host",
  and a `Benchmark` of the same call kept alongside it.

**Out:**

- The `quarry/` facade, the CLI, the `ok`/`status` envelope, exit codes and the lossless text view —
  **T5b**. T4 emits payload objects and typed errors; the envelope wraps them later, exactly as T3
  did for `toc`.
- MCP — **T6**, and T6 exposes only `toc` regardless.
- Any change to `internal/engine`'s walk, its `toc` answer, its ignore matcher, its strategies, or
  its `Symbol` key set. T4 consumes T3's engine and adds to it; it does not revise it. The one
  permitted exception is the mechanical follow-through T3 itself scheduled (see Technical context).
- Any change to `glyph/`. The engine never re-implements the grammar (glyph.md §6); T4 imports it
  and believes its answers.
- Any change to `docs/glyph.md` or `docs/rewrite-plan.md`. T4 records two contract gaps it runs into
  and closes neither — see D18.
- `impact`, `assert-no-callers`, `verified`, a type checker — phase 2.
- Any language but Go, and any cache, index, daemon or concurrency (plan §5, §10; T3's D22).
- Changes to `bench/`. The Loomyard timing test is a Go test in `internal/engine`, not a ladder cell.

## Provenance — what is inherited, and what the plan revision must verify

This task was discussed while T3 was still in flight, so every fact below is labelled by where it
came from. **Nothing in the `internal/engine` column exists on `main` at the time of writing.**

**On `main` today (verified in this worktree):** the `glyph` package (`Parse`, `Glyph`, `String`,
`Language`, `ParseError`, `Reason`); the V1 extractor under `internal/quarryengine/` and
`internal/quarryengine/toc/`, which T3 deletes; `docs/glyph.md`; `docs/rewrite-plan.md`; the ladder
harness under `bench/`.

**Inherited from `.portals/engine-core/` and additionally confirmed by reading the `engine-core`
worktree's committed code** (so these are decided *and* written, but unmerged — verify the merge did
not change them):

| fact | artifact | code seen in `wts/engine-core` |
|---|---|---|
| package `internal/engine`, files per concern; `treesitter` a subpackage; `internal/cgoguard` | D1, D2, plan 01 cards 1–4 | yes |
| `Symbol{Glyph, ID, Kind, File, Start, SigEnd, End, Signature, Doc, HeadStart, HeadEnd}` and the five-value `Kind` | D3, D5, plan 04 card 23 | `internal/engine/answer.go` |
| `HeadStart`/`HeadEnd` populated for `KindType` only, equal to that symbol's own `Start`/`End` (doc block included), for every Go type including interfaces | D4, plan 04 cards 23/28 | `answer.go` doc comment |
| `DirAnswer`, `FileEntry`, `TOCOptions{Depth, Symbols *bool}`, `DepthAll` | D12, D13 | `answer.go` |
| `Repo`, `Open(root)`, `resolveTarget`, `ErrTargetOutsideRepo`, `ErrTargetNotFound`, `os.Lstat` never `os.Stat`, a gitignored explicit target answered not refused | D15, D20 | `repo.go` |
| `ignoreSet`, `newIgnoreSet(root)`, `extend(dirRel) (n, err)`, `trim(n)`, `match(pathRel, isDir)` | D9, D22 | `ignore.go` |
| `unitFor(dirRel, dirPkg, fileClause)`, `unitSpellable(unit)`, `dirPackage`, `fileEntry`, `walkDir` | D7, plan 04 card 25 | `walk.go` |
| `Strategy.Symbols(unit string, root, src) []Symbol`; the Go walk over func/method/type/const/var/interface-method; generic receiver owner stripped; `_` skipped | D5, plan 04 cards 24–28 | `strategy.go`, `golang.go` |
| `ErrLanguageUnsupported` exists (its doc comment is rewritten by T3's own card 33) | D21, plan 05 card 33 | `errors.go` |

**Inherited as design only — not implemented anywhere at the time of writing.** T3's batch 5
(`spans`) was still `pending` when this discussion was written; batch 4 was `running`. Everything
T4 stands on most directly is in that batch:

- `func (r *Repo) unitDirs(unit string) (dirs []string, collision bool)` — the unit→directory
  inverse, **literal-first**: the directory named exactly by the unit wins; only if it does not
  exist and the unit ends `_test` is the suffix stripped; both existing returns both with
  `collision == true`. (D16, plan 05 card 31.)
- `func (r *Repo) symbolsOfUnit(unit string, ig *ignoreSet) ([]Symbol, error)` — parses each of the
  unit's `.go` files once, returns every symbol with `File` set to the repository-relative path,
  ordered by file then start line, filtered through the same ignore set the walk uses, and returns
  the union when `unitDirs` reports two directories. Its caller hands it a set carrying the root's
  own patterns only (`newIgnoreSet(root)` then one `extend(".")`); it extends and trims per
  directory itself. (D16, plan 05 card 32.)
- `func (r *Repo) SpansOf(g glyph.Glyph) ([]Symbol, error)` — rejects a non-Go `Lang` with
  `ErrLanguageUnsupported` first, then validates the glyph by round-tripping it through
  `glyph.Parse(g.Lang, g.String())`, then calls `symbolsOfUnit` and filters by owner chain and name.
  Zero matches is an empty slice and a nil error; there is no status vocabulary. (D16, plan 05
  card 32.)
- The binding decision this task's own brief carries: mapping a unit to a directory tries the
  literal directory first and the `_test`-stripped fallback second, and **both existing is exactly
  T4's `ambiguous` case**.

**The plan revision's verification list**, to be run after `main` is merged in (`mill-merge-in`):

1. Do `unitDirs`, `symbolsOfUnit` and `SpansOf` exist with those signatures, in
   `internal/engine/resolve.go`? If batch 5 shipped a different split, D9 and D5 below are the
   decisions to re-derive.
2. Is `collision` still the second return of `unitDirs`, and is it still unexported? D6 promotes it.
3. Are `HeadStart`/`HeadEnd` actually set to the type symbol's own `Start`/`End` (with the doc block)
   for interfaces as well as structs? D12 reads them.
4. Is `symbolsOfUnit`'s `ig` parameter still an `*ignoreSet` the caller builds, or did it start
   building its own? D9's per-call memo hands it one set for the whole call.
5. Does `Symbol.File` come back set from `symbolsOfUnit`, and is the order file-then-line? D15 rests
   on it.
6. Did T3's batch 6 add anything to `Symbol`'s key set? D3's result types embed `Symbol` verbatim.

## Decisions

### D1 — Both verbs live in `internal/engine`, `resolve.go` grows and `expand.go` is new

- Decision: `Resolve` and its helpers go into the existing `internal/engine/resolve.go`, beside
  `SpansOf`, `symbolsOfUnit` and `unitDirs`. `Expand` and `NotATypeError` go into a new
  `internal/engine/expand.go`. The result types go into `internal/engine/answer.go` beside `Symbol`,
  `DirAnswer` and `FileEntry`. No new package, no new subpackage.
- Rationale: plan §12 T3's scope says the engine is "one package, files per concern, never a package
  per verb", and T3's D16 put the span lookup in `resolve.go` explicitly so that "T4 grows the file
  rather than a file T4 has to move". `answer.go` is already the file whose header comment declares
  itself the home of the emitted key set; splitting the two verbs' answer types away from `Symbol`
  would put half the contract in one file and half in another.
- Rejected: a `resolve/` subpackage (a package per verb, which the plan forbids in as many words); a
  separate `results.go` for the answer types (`answer.go` already says it is that file); putting
  `Expand` in `resolve.go` too (they are two verbs with two answers and no shared helper beyond
  `symbolsOfUnit`, which is neither's).

### D2 — `Resolve` takes many targets and returns one result per argument, in argument order

- Decision:

  ```go
  func (r *Repo) Resolve(targets []string) ([]ResolveResult, error)
  ```

  The returned slice has exactly `len(targets)` elements and `result[i]` answers `targets[i]`. A
  target repeated twice is answered twice. The returned `error` is reserved for a failure of the
  call as a whole (an I/O error the walk could not attribute to one target); every per-target outcome
  — including a malformed target — is expressed in that target's own `ResolveResult`.
- Rationale: §8.1 is explicit that "All glyphs of a draft go in one `resolve` call", and the
  validator that makes the call has to map each answer back to the card that asked for it. A
  positional 1:1 slice makes that mapping total and free, and it is the only shape where a duplicate
  or a malformed target cannot silently vanish. Returning `error` for one bad glyph would throw away
  the other thirty-nine answers of a plan-validation call, which is the exact call the verb exists
  for.
- Rejected: one target per call (§5's whole grouping guarantee is about many targets at once, and
  §8.1 asks for one call); a `map[string]ResolveResult` keyed by target (loses argument order and
  collapses duplicates); returning `error` on a malformed target (D8).

### D3 — The result types, and the status vocabulary as one closed type

- Decision:

  ```go
  // Status is the closed per-entry vocabulary of docs/glyph.md §5.
  type Status string

  const (
      StatusFound     Status = "found"
      StatusNotFound  Status = "not_found"
      StatusAmbiguous Status = "ambiguous"
      StatusMultipart Status = "multipart"
  )

  type ResolveResult struct {
      Target     string     `json:"target"`
      ID         string     `json:"id,omitempty"`
      Status     Status     `json:"status,omitempty"`
      Unit       Status     `json:"unit,omitempty"`
      Symbols    []Symbol   `json:"symbols,omitempty"`
      Candidates []Symbol   `json:"candidates,omitempty"`
      Dir        *DirAnswer `json:"dir,omitempty"`
      Error      string     `json:"error,omitempty"`
      Reason     string     `json:"reason,omitempty"`
  }
  ```

  - `Target` is the argument verbatim, always present — it is what makes D2's mapping readable in
    JSON as well as in Go.
  - `ID` is the **canonical** glyph spelling `glyph.Parse` returned, present only for a glyph target.
    §8.1 layer 1 says Parse "returns the canonical spelling, so `Draw (int)` comes back as
    `Draw(int)`"; emitting it is how a caller learns the canonical form of what it asked for.
  - `Status` carries §4's per-entry status. It is absent only in D8's case, where the target never
    reached resolution.
  - `Unit` reuses `Status` and is set **only** on a glyph `not_found`, to `StatusFound` or
    `StatusNotFound`. Its JSON spelling is §5's own: `"unit": "found"`.
  - `Symbols` carries the matches for `found` (exactly one) and `multipart` (every part).
  - `Candidates` carries the matches for `ambiguous`. §4 names this key explicitly — "`ambiguous`
    (with `candidates`)" — and the separate key is the honest signal that nothing was chosen.
  - `Dir` carries a path target's answer (D7).
  - `Error`/`Reason` carry D8's non-resolution failures.
- Rationale: every key here is either §4's or §5's own word. Reusing `Status` for the `unit` key
  rather than minting a two-value `UnitStatus` keeps one closed vocabulary in the package instead of
  two overlapping ones; the doc comment states that only `found` and `not_found` are ever written
  there. `Symbols` and `Candidates` are separate rather than one key plus a status test because a
  consumer filtering for usable answers should not have to read the status to know whether the
  entries it holds were chosen or merely listed.
- The `Symbol` values are T3's, unchanged and complete, with `File` set — §4's "One symbol entry
  everywhere: `resolve`, `expand` and `toc` all return this entry and nothing else for a symbol."
  T4 adds no field to `Symbol` and emits no symbol shape of its own.
- Rejected: an output grouped by unit (D4's note); a single `Locations` key for both found and
  ambiguous (erases §4's `candidates`); a bespoke `UnitStatus` type (two vocabularies for one set of
  words); embedding the glyph struct in the result (`ID` is the wire form, exactly as on `Symbol`).

### D4 — "Grouped by unit" is the parse guarantee, not the output shape

- Decision: the answer is flat (D2). The grouping §5 requires is an **execution** property: within
  one `Resolve` call, each distinct unit is parsed exactly once, however many targets name it. D9
  gives the mechanism and Testing 8 proves it.
- Rationale: §5's sentence is about cost — "Many glyphs in one call are grouped by unit and each unit
  is parsed once" — and it is in the paragraph that ends with §5's measured timing table. Making it
  an output shape instead would collide head-on with the vocabulary: a per-unit group's natural key
  is `unit` holding a *path*, and §5 already spells `unit` as a key holding *`found` or `not_found`*.
  One of the two would have to be renamed, and the one with a literal spelling in the specification
  is the one that must not move. A grouped output also breaks D2's 1:1 mapping for the caller §8.1
  names, and has nowhere to put a path target, which belongs to no unit.
- Rejected: grouping the output and renaming the existence key to `unit_status` or similar (changes
  a spelled contract to gain a shape nothing asked for); grouping the output and naming the group
  key `dir` (a unit is not a directory — that is the whole point of `unitDirs`).

### D5 — `multipart` is `init` and nothing else in Go; every other multi-match is `ambiguous`

- Decision: after collecting matches for a glyph, with `n = len(matches)`:
  - `n == 0` → `not_found` (plus D10's `unit`).
  - `n == 1` → `found`.
  - `n > 1` and the glyph is a bare package-level `init` (`len(g.Owner) == 0 && g.Name == "init"`) →
    `multipart`, every declaration returned in `Symbols`, file-then-line order.
  - `n > 1` otherwise → `ambiguous`, every declaration returned in `Candidates`, same order.
- Rationale: glyph.md §5 defines `multipart` as "one symbol the language lets be declared in several
  places" and gives Go exactly one instance — `init` — while `ambiguous` is "several *different*
  declarations match: Go build-tag duplicates". The discriminator is therefore the glyph's own name,
  not any property of the declarations, and it is decidable without reading build tags, which
  tree-sitter does not evaluate and which the engine has no business interpreting. A build-tagged
  pair of `init` functions is still `init` and still `multipart`, which is what glyph.md's "with
  several `init` functions it resolves `multipart`, every one returned in run order" says.
- Rationale for not reading build constraints: doing so would make the answer depend on a `GOOS`/
  `GOARCH` the engine does not know and the caller did not state, and §3's no-silent-pick rule makes
  guessing one worse than reporting both. `ambiguous` with both candidates and their files is the
  honest answer, and §8.1 makes it a plan rejection rather than something a consumer must resolve.
- Rejected: evaluating build tags to disambiguate (a silent pick over an unstated platform);
  treating any multi-match as `multipart` (erases the distinction §4 and §5 both draw); a
  per-language "multipart names" table (one entry, in a Go-only tree — YAGNI, and glyph.md is where
  a second language's answer would be written first).

### D6 — The unit-directory collision promotes to `ambiguous`

- Decision: when `unitDirs(unit)` returns `collision == true`:
  - with at least one match across the two directories → `ambiguous`, `Candidates` holding the union
    in file-then-line order, whatever `n` is. A single match under a collision is still `ambiguous`.
  - with no match in either → `not_found` with `unit: found`.
- Rationale: T3's D16 records this in as many words — "T4 promotes that second return into the
  `ambiguous` status when it builds the status vocabulary" — and this task's own brief carries it as
  a binding inherited decision. The single-match case is the one worth stating: `n == 1` would
  otherwise fall through D5 to `found`, and a `found` whose glyph string names two different units
  is precisely the "one glyph string names two different units" failure the literal-first rule exists
  to prevent. The `not_found` case is honest for the opposite reason: both directories exist, so the
  unit is there and only the member is missing, which is exactly §5's `unit: found`.
- Neither quarry nor Loomyard has a directory literally named `<something>_test`, so this case is
  reachable only from a fixture. That is why it is decided by construction and tested with a
  purpose-built tree, not left to be discovered.
- Rejected: reporting `found` on a single match under a collision (a glyph that names two units
  answering as if it named one); reporting `ambiguous` with zero matches (there is nothing to be
  ambiguous between, and it would hide the `unit: found` a Create card needs).

### D7 — A target without `#` is a path, answered with T3's directory answer

- Decision: a target containing no `#` is a repository-relative path. It is answered by calling
  `r.TOC(target, TOCOptions{Depth: 0, Symbols: &falseValue})`:
  - success → `Status: found`, `Dir` set to the returned `DirAnswer` (which, for a file target, is
    the enclosing directory's answer holding exactly that one file entry, per T3's D13), `ID` and
    `Unit` absent.
  - `errors.Is(err, ErrTargetNotFound)` → `Status: not_found`, `Dir` absent, `Unit` absent.
  - `errors.Is(err, ErrTargetOutsideRepo)` → D8's error entry.
- Rationale: §5 says a path target "answers with the file entry or directory answer and `found` or
  `not_found`", and §4 says a file query *is* a directory answer with one file entry and that "the
  file never repeats its parent's facts" — so a bare `FileEntry` would be an answer a consumer
  cannot read a package or a language off. Reusing `TOC` rather than restating the rule means the
  gitignored-explicit-target rule (T3 D20), the never-follow-a-symlink rule (D19) and the
  `""`/`"."`-mean-the-root rule all hold here for free and cannot drift.
- `Symbols` is switched **off** explicitly rather than left to `TOC`'s per-target default (which
  would turn it on for a file target). `resolve` answers where a thing is and whether it exists;
  what is inside it is `toc`'s question, and a plan card whose target is `viewer.html` or a Markdown
  page has no symbols to want. Paying a tree-sitter parse per `.go` path target in the call §5
  measures at 150 ms would be a cost with no consumer.
- `Unit` is never set on a path result: units are the glyph alphabet's, and a path has none.
- Rejected: a bare `FileEntry` (not self-describing, contradicts §4); defaulting the symbols knob
  (a per-target-kind default inside a verb whose question is existence); a separate `ResolvePaths`
  entry point (§5 and §8.1 both put paths and glyphs in the same call, deliberately — "a target
  without `#` is a path ... and follows the same rows with the file entry in place of the symbol").

### D8 — A target that never reached resolution carries `error`, not a status

- Decision: two cases produce a `ResolveResult` with `Status` **empty** and `Error` set:
  - a target containing `#` that `glyph.Parse(glyph.Go, target)` rejects — `Error` is the
    `*glyph.ParseError`'s message and `Reason` is its `Reason` value (`"member_too_deep"`,
    `"unit_bad_rune"`, …), so a caller gets the machine-readable cause without string matching;
  - a path target that `resolveTarget` rejects with `ErrTargetOutsideRepo` — `Error` is that error's
    message, `Reason` empty.

  Neither aborts the call: the other results are answered normally.
- Rationale: the four statuses are *resolution* outcomes — glyph.md §5's table is headed by what a
  declaration search found. A string that is not a glyph was never searched for, and calling it
  `not_found` would be exactly the guess §5 forbids ("Resolution never guesses ... no 'did you
  mean'"): it would tell a Create card's validator that the name is free when the truth is that the
  name is unspellable. §8.1's layering says the same thing structurally — `glyph.Parse` is layer 1
  and `resolve` is layer 2, so a layer-1 failure is not a layer-2 answer. Carrying it per entry
  rather than failing the call preserves D2's 1:1 contract and keeps one malformed card from blinding
  the validator to the other thirty-nine.
- The `Reason` key is a string, not `glyph.Reason`, so `answer.go` needs no exported alias and the
  emitted JSON is a plain word. The value is `string(parseErr.Reason)`.
- Rejected: a fifth status (`invalid`) — the vocabulary is closed by §4 and glyph.md §5, and this is
  not a resolution outcome; `not_found` (a guess, and the wrong one for a validator); failing the
  whole call (throws away every other answer in the one call §8.1 asks for).

### D9 — One unit parsed once per call: the memo lives in the call and dies with it

- Decision: `Resolve` builds, per call:
  1. one `ignoreSet` — `newIgnoreSet(root)` followed by a single `extend(".")`, which is exactly
     what T3's plan 05 card 32 specifies `symbolsOfUnit`'s caller must hand it;
  2. a `map[string][]Symbol` memo keyed by unit, plus a `map[string]unitDirsResult` memo for
     `unitDirs`.

  Every glyph target is parsed first; the distinct units among them are then resolved through
  `symbolsOfUnit` **once each**, and each target's matches are filtered out of its unit's slice.
  Both memos are local variables and are discarded when `Resolve` returns. Nothing is stored on
  `Repo`.
- Rationale: this is §5's own requirement — "Many glyphs in one call are grouped by unit and each
  unit is parsed once" — and it is the difference between the done-criterion passing and failing.
  §5's measurements are 65 ms for one glyph in a 35-file package and 109 ms for twenty glyphs across
  five packages *when grouped*; twenty ungrouped lookups over the same five units would be four
  parses of each unit and land far outside 150 ms. T3's D16 says the same from the other side:
  `symbolsOfUnit` "is the seam T4 needs anyway, and building it here means T4 inherits it rather
  than refactoring `SpansOf` to get it."
- The call-scoped lifetime is what keeps this inside T3's D22 ("nothing is cached"): D22 forbids
  state that outlives a call, because T6's long-lived process would serve stale answers from it. A
  memo that cannot survive the call it was built in cannot go stale, and the alternative — reading
  the same five directories twenty times inside one answer — is not freshness, it is waste.
- `SpansOf` is **not** used by `Resolve`. It is the per-glyph wrapper; `Resolve` needs the per-unit
  primitive under it, plus `unitDirs` for D10's `unit` key and D6's collision. `SpansOf` keeps its
  place as the single-glyph convenience and as what T3's round trip is written against. `Expand`
  likewise calls `symbolsOfUnit` directly, since it needs the whole unit to find members.
- Rejected: calling `SpansOf` per target (four parses per unit in the criterion's own call — the
  precise failure T3's D16 spells out for the round trip); memoising on `Repo` (D22, and stale under
  T6); sharing one memo across calls behind an mtime check (the phase-1 cache §10 forbids, under
  another name).

### D10 — `unit: found|not_found` is `unitDirs`'s answer, restated nowhere

- Decision: on a glyph `not_found`, `Unit` is `StatusFound` when `unitDirs(unit)` returned at least
  one directory and `StatusNotFound` when it returned none.
- Rationale: §5 defines the distinction as "`unit: found` when the directory, module or namespace is
  there and only the member is missing" — directory existence, which is the exact question
  `unitDirs` answers, including its literal-first rule and its `_test`-stripping fallback. Deriving
  it any other way would be a second implementation of unit→directory in the same package, and the
  two would drift on precisely the corner case the rule was written for.
- Consequence worth recording: a `_test` unit whose stripped directory exists but which holds no
  external test package at all reports `unit: found`, because the directory is there. That is what
  the specification's wording says and what `unitDirs` decides; requiring a matching package clause
  would be the engine adding a rule the contract does not have. A Create card for the first file of
  an external test package is exactly the case that wants `unit: found`, so the literal reading is
  also the useful one.
- `Unit` is set **only** on `not_found`. §5 attaches it to the miss and to nothing else, and a
  `found` that also said `unit: found` would be the "defaults never" clutter §4 removes.
- Rejected: testing for a directory holding at least one `.go` file (a rule the contract does not
  state, and wrong for the Create case); testing for a matching package clause (same); computing it
  from `os.Stat` directly (a second unit→directory implementation).

### D11 — `Expand` takes a string, and its answer carries a status

- Decision:

  ```go
  func (r *Repo) Expand(target string) (ExpandAnswer, error)

  type ExpandAnswer struct {
      ID         string   `json:"id"`
      Status     Status   `json:"status"`
      Unit       Status   `json:"unit,omitempty"`
      Head       *Symbol  `json:"head,omitempty"`
      Members    []Symbol `json:"members,omitempty"`
      Candidates []Symbol `json:"candidates,omitempty"`
  }
  ```

  `Status` is `found`, `not_found` or `ambiguous`, decided by exactly D5's and D6's rules over the
  matches for `target`, with `Unit` set on a miss by D10. `Head` and `Members` are populated only on
  `found`; `Candidates` only on `ambiguous`.
- Rationale: a caller that asks `expand` a question should not have to call `resolve` first to learn
  that the glyph is a typo or that two build-tagged types answer to it. Reusing the one `Status`
  type and the one set of rules costs a shared helper and means the two verbs can never disagree
  about what a glyph resolves to. Taking a `string` rather than a `glyph.Glyph` matches `Resolve`
  and matches what T5b will have in hand — argv — and it lets `expand` report a malformed glyph the
  same way, through the error return (below) rather than by making the caller pre-parse.
- A target `glyph.Parse` rejects, or one with no `#`, returns the wrapped `*glyph.ParseError` (or a
  plain error naming the missing `#`) as `Expand`'s `error`. Unlike `Resolve`, `Expand` answers one
  target, so there is no other answer to protect and no reason to move the failure into the payload;
  D8's whole argument is about not losing thirty-nine good answers, and it does not apply here.
  `toc` never takes a glyph and `expand` never takes a path — plan §5 states both.
- `multipart` is unreachable for a Go type and is not special-cased: §5 says "a Go type never splits
  (only `init` does)", and D5 already routes every non-`init` multi-match to `ambiguous`. When a
  language with partial types arrives, that language's `expand` reads §5's C# rule ("the head is the
  sum of the parts' own lines") and the status follows from the same shared helper. Stated in the
  doc comment; no code.
- Rejected: `Expand(g glyph.Glyph)` (T5b holds a string, and pre-parsing would put layer 1 in the
  caller for one verb and not the other); a bare `[]Symbol` return with the head first by convention
  (loses the status, and "the first element is the head" is the kind of positional contract §4's
  key-set discipline exists to avoid); returning an error for `not_found` (it is a legitimate `ok:
  true` answer with a §4 status, not a failure).

### D12 — The head is the type's own symbol entry, with its span read from `HeadStart`/`HeadEnd`

- Decision: `Head` is the matched type's `Symbol`, emitted as §4's ordinary symbol entry — `id`,
  `kind`, `file`, `start`, `sigend`, `end`, `signature`, `doc` — with two substitutions:
  `Start = sym.HeadStart` and `End = sym.HeadEnd`. Every other field is the symbol's own. If a
  `KindType` symbol comes back with `HeadStart == 0`, `Expand` returns an error naming the id: that
  is a T3 invariant violation, and a silent fallback to `Start`/`End` would hide it.
- Rationale: §4's rule is absolute — "One symbol entry everywhere ... `resolve`, `expand` and `toc`
  all return this entry and nothing else for a symbol" — so `expand` emits entries, never source
  text and never a shape of its own. Reading the span from `HeadStart`/`HeadEnd` rather than from
  `Start`/`End` is what gives T3's D4 field its consumer, and D4's own stated purpose is exactly
  this: "Making it explicit now means `expand` reads a field instead of re-deriving a rule." For Go
  the two pairs are identical, so nothing observable changes today; for the first language whose
  head is a strict subset of the declaration, `expand` needs no edit.
- **On §5's "the class span minus its member spans".** That phrase describes what a *reader* ends up
  reading, not a span arithmetic `expand` performs. For a Go struct the subtraction is empty — a
  struct's methods live outside the declaration entirely. For a Go interface the member spans do lie
  inside the head range, and the answer already carries them: a consumer that wants only the head's
  non-member lines has every member's `start` and `end` in the same answer and can omit them. T3's
  D4 says the subtraction "is the consumer's, not the extractor's" and concludes that one span pair
  suffices and "no discontiguous span type is needed"; emitting one contiguous head entry is the
  only reading of that conclusion consistent with §4's closed entry shape. **This is a deliberate
  reading of D4's prose, which also calls `expand` the thing that "renders" the subtraction — the
  plan revision should confirm nothing in T3's merged code assumed `expand` would emit a line set.**
- Rejected: emitting a discontiguous head (`[]span`) — a symbol shape §4 does not have, in the one
  place §4 says there is only one shape; emitting rendered head *text* (that is T5a's lossless text
  view, over this payload, not a payload of its own); falling back to `Start`/`End` when `HeadStart`
  is zero (hides a T3 regression behind an answer that happens to be right for Go).

### D13 — Members are the unit's symbols whose owner chain begins with the type

- Decision: given the matched type symbol with glyph `g`, `Members` is every symbol `s` in
  `symbolsOfUnit(g.Unit, ig)` with `len(s.Glyph.Owner) > 0 && s.Glyph.Owner[0] == g.Name`, sorted by
  file then start line, each with `File` set. The type symbol itself is excluded (it is `Head`).
  One unit only: the external `_test` unit is a different unit and cannot declare methods on the
  package's types.
- Rationale: §5 defines `expand` as "the head, plus every glyph whose owner chain begins with the
  type", and that is a filter over the owner chain, not over spans or files — which is what makes
  "across files" free: `symbolsOfUnit` already parses every file of the unit and returns them in
  file-then-line order. Matching on `Owner[0]` rather than on the whole chain is the general form;
  in Go the chain is at most one element, because glyph.md rejects a deeper member outright
  (`ReasonMemberTooDeep`), so the two are the same today and the general form is what a nested-type
  language needs.
- Interface methods are members by the same rule with no special case: T3's card 28 gives a
  `method_elem` the interface's type name as its owner, so an interface's methods fall out of the
  same filter that finds a struct's methods, and they sort into place by file and line — which for
  an interface means inside the head range, exactly as D12 describes.
- A type with no members answers `found` with `Head` set and `Members` absent (`omitempty`). That is
  not an error and not a `not_found`: the type exists and consists of its head.
- Rejected: filtering by file (methods live in other files — that is the whole point of the verb);
  filtering by span containment (wrong for structs, and would make an interface's members a
  different rule from a struct's); searching the `_test` unit as well (a different package; its
  declarations cannot be members of this unit's types).

### D14 — A glyph naming any other kind is a typed error, `*NotATypeError`

- Decision:

  ```go
  type NotATypeError struct {
      ID   string
      Kind Kind
  }

  func (e *NotATypeError) Error() string // "engine: expand <id>: not a type, kind <kind>"
  ```

  `Expand` returns it when the target resolves `found` to exactly one symbol whose `Kind` is not
  `KindType`. It is returned as the `error`, not carried in `ExpandAnswer`.
- Rationale: §5 says "The glyph must name a type; on any other kind the answer is `ok: false` naming
  the kind." `ok` is the envelope's, and T3's D20 established the pattern for exactly this: the
  engine returns typed, `errors.Is`/`errors.As`-able sentinels and T5a/T5b map them to `ok`, a
  status word and an exit code. A struct with the kind on it is what lets T5b's message name the
  kind without parsing a string, which is the same argument D20 made for splitting
  `ErrTargetOutsideRepo` from `ErrTargetNotFound`.
- The precedence is fixed and testable: a malformed target errors before anything is read; a
  `not_found` or `ambiguous` is a payload answer with `ok: true` and never reaches this check
  (there is no single kind to name); only a unique, `found`, non-type match produces
  `*NotATypeError`.
- Rejected: an `ok bool` + `kind` pair in `ExpandAnswer` (duplicates the envelope T5a owns inside
  the payload — the exact "`ok: true` inside data" clutter §4 lists); a bare sentinel `ErrNotAType`
  (cannot name the kind, which §5 requires); reporting it as a `not_found` (the symbol is there —
  saying it is not is a lie a Delete card's done-check would act on).

### D15 — Ordering, stated for both verbs

- Decision:
  - `Resolve`: results in argument order, one per argument (D2). Within a result, `Symbols` and
    `Candidates` are ordered by file, then by start line — the order `symbolsOfUnit` already
    returns, preserved by the filter.
  - `Expand`: `Head` first as its own field; `Members` by file, then by start line.
  - File comparison is the raw repository-relative string with forward slashes, `sort.Strings`
    semantics, no case folding and no locale — the same rule T3's D18 applies to `Files` and `Dirs`.
  - Every ordering is the engine's. No caller sorts.
- Rationale: glyph.md §5 states it as part of the contract — "Results are ordered by file and then by
  start line, so the answer is deterministic" — and §12's T4 row names "ordering guarantees" as
  deliverables. T3's D18 already made the engine the sorter and gave the reason: an unpinned order
  makes golden comparison untestable and freezes whatever `os.ReadDir` produced on the machine that
  generated the golden. The same argument applies unchanged here, and argument order for the
  top-level results is the only order that keeps D2's mapping usable.
- Rejected: sorting results by unit or by id (breaks the 1:1 mapping §8.1's validator needs);
  leaving order to T5b (the answers would not be reproducible and the fixtures could not assert).

### D16 — The 150 ms criterion is a floor measurement, and the benchmark is kept beside it

- Decision: two artefacts in `internal/engine`, both gated on Loomyard by T3's D17 rule exactly
  (skip when `LADDER_LOOMYARD_REPO` is unset or names a nonexistent directory; **fail** when it is
  set but `git -C <repo> rev-parse HEAD` is not `72c23d9`; skip when the `git` binary is missing or
  errors):
  1. `TestResolve_TwentyGlyphsUnder150ms` — skipped under `-short`. It first asserts that all twenty
     targets resolve `found`, then times one `Resolve` call five times and asserts the **minimum**
     elapsed is under 150 ms.
  2. `BenchmarkResolveTwentyGlyphs` — the same call, kept as `§12`'s "timing test against Loomyard,
     kept as a benchmark", so a regression is measurable rather than merely detectable.
- Rationale: a wall-clock assertion is a flaky assertion on a shared or loaded machine, and a flaky
  test in the gate is worse than no test. Taking the minimum of five runs measures the floor — the
  thing the criterion is actually about, since §5's numbers are themselves best-case in-process
  measurements with no cache. A single run would fail on an unrelated build running in another
  worktree. Asserting `found` first is what stops the measurement being meaningless: twenty misses
  resolve fast, and a drifted glyph list would turn the criterion green by measuring nothing.
- The twenty glyphs are four each from five Loomyard packages of differing size, pinned as literal
  strings in the test file and chosen at implementation time from the checkout at `72c23d9`. The
  pin is enforced (D17's fail-on-mismatch), so they cannot silently go stale. At least one of the
  five must be a large package — §5 measures `internal/reedengine` at 67 files and 65 ms for a
  single glyph — so the grouping of D9 is what the number depends on.
- No tracked file carries the Loomyard path; `.scratch/ladder.env` holds `LADDER_LOOMYARD_REPO` per
  machine and is gitignored, exactly as T2 and T3 do.
- Rejected: a single timed run (flaky); an average or a p95 over runs (measures the machine's load,
  not the code); asserting only in the benchmark (a benchmark is not run by `go test ./...`, so the
  done-criterion would never be checked); committing a Loomyard fixture (large, and the number is
  meant to be over a real repository).

### D17 — Every status in glyph.md §5 has a committed fixture

- Decision: `internal/engine/testdata/` gains the fixtures below, and each maps to one named status
  case. Fixtures that cannot be committed — a tree whose own `.gitignore` excludes one of its files,
  and the both-directories collision — are built at run time under `.scratch/`, via the
  `writeScratchTree` helper T3's `scratchtree_test.go` already provides, and removed in
  `t.Cleanup`. **Never `t.TempDir()`**: it writes to the system temp directory, which the
  constraints ban.
  - `found` — a plain package-level function and a method, in the existing `testdata/tree/pkg`.
  - `multipart` — three `func init()` in one package (T3's `testdata/glyphs/inits.go` already
    carries three; if it lives in a package the resolve tests can address, reuse it rather than add).
  - `ambiguous`, build tags — a new `testdata/tags/` holding two files, `//go:build linux` and
    `//go:build !linux`, each declaring the same `func Dup()`. `testdata/` is invisible to the go
    tool, so the pair never reaches a build.
  - `ambiguous`, unit collision — a run-time `.scratch/` tree with both a `foo/` holding an external
    test package and a directory literally named `foo_test/`, per T3's plan 05 card 34's own
    collision fixture.
  - `not_found` with `unit: found` — an existing unit, a name that is not in it.
  - `not_found` with `unit: not_found` — a unit whose directory does not exist.
- Rationale: §12's T4 done-when is "glyph.md §5 statuses each have a fixture", so the fixtures are
  the deliverable, not a by-product. Committing them where possible keeps the tests readable and
  reviewable; the two exceptions are the two shapes a committed tree provably cannot express, and
  T3 already established both the reason and the mechanism.
- Rejected: `t.TempDir()` (banned by the constraints); synthesising sources as string literals in
  the test (a fixture nobody can open and read is a worse fixture); one fixture package covering
  several statuses at once (a build-tag pair inside the package the other cases query would make
  every other case's symbol list depend on it).

### D18 — Two contract gaps are recorded, neither is closed

- Decision: T4 changes no line of `docs/glyph.md` or `docs/rewrite-plan.md`. Two gaps it runs into
  are written down in code comments, where the rule they affect lives:
  1. **The external test unit versus a real directory of the same name.** glyph.md §2 gives the
     external test unit the pseudo-path `<dir>_test` without saying what happens when a real
     directory spells it. T3 fixed quarry's behaviour (literal-first, both existing recorded as a
     collision); D6 promotes it to `ambiguous`. This is quarry's choice, not the contract's, and it
     is recorded as such where D6's rule is implemented.
  2. **`ambiguous` candidates carry no language marker.** §5 says that in a multi-language repository
     the candidates are "marked by language". `Symbol`'s key set is §4's and is closed, and it has no
     `language` key. With Go the only alphabet the case is unreachable, so no key is added; the
     comment on `Candidates` records that a second language adds the marker, against a real case, in
     that language's task.
- Rationale: T3 took the same position on the repository-root unit and gave the reason: a
  single-file task changing the shared identifier contract is exactly the coupling §7's ordering
  avoids, and both candidate answers for a gap should be decided against a repository that needs
  one. Recording the gap where the behaviour is implemented is what keeps it findable; recording it
  only in a discussion file would lose it.
- Rejected: amending glyph.md here (out of scope, and it would put two authors on one contract in
  one wave); adding a `language` field to `Symbol` now (changes a key set §4 pins and T3's goldens
  assert, for a case no test can reach); leaving the gaps unrecorded.

## Technical context

**Everything T4 builds on is unmerged.** See the Provenance section above for the full table and the
six-item verification list the plan revision must run after `mill-merge-in`. The single most
important item: `unitDirs`, `symbolsOfUnit` and `SpansOf` are **design only** — T3's batch 5
(`spans`) had not started when this was written. If that batch shipped a different split, D9's memo
and D10's unit test are the two decisions to re-derive first.

**The pipelined hold.** Do not begin implementation until T3 is merged to `main`. Then merge `main`
into this branch with the `mill-merge-in` skill and revise the plan against the code T3 actually
chose, not the layout its discussion assumed.

**Files T4 touches:**

| file | change |
|---|---|
| `internal/engine/answer.go` | add `Status` and its four constants, `ResolveResult`, `ExpandAnswer` |
| `internal/engine/resolve.go` | add `Resolve`, the per-call memo, the glyph/path split, the status rules |
| `internal/engine/expand.go` | new: `Expand`, `NotATypeError` |
| `internal/engine/resolve_test.go` | extend with the status fixtures and the grouping test |
| `internal/engine/expand_test.go` | new |
| `internal/engine/loomyard_timing_test.go` | new: the 150 ms test and the benchmark |
| `internal/engine/testdata/tags/` | new: the build-tag `ambiguous` pair |

**Helpers to reuse rather than rewrite** (all T3's, all in `internal/engine`):

- `symbolsOfUnit(unit, ig)` — the per-unit primitive. D9's memo wraps it; nothing else parses.
- `unitDirs(unit) (dirs, collision)` — unit→directory, literal-first. D6 and D10 both read it.
- `newIgnoreSet(root)` + one `extend(".")` — the set `symbolsOfUnit`'s caller must supply. Built
  once per `Resolve`/`Expand` call and passed down; `symbolsOfUnit` extends and trims per directory
  itself.
- `Repo.TOC(target, TOCOptions{})` — D7's path answer, with every T3 target rule inherited.
- `resolveTarget`, `ErrTargetNotFound`, `ErrTargetOutsideRepo` — reached through `TOC`, matched with
  `errors.Is`.
- `glyph.Parse(glyph.Go, s)` — the one grammar. `Glyph.String()` for the canonical `ID`.
- `Symbol`, with `File`, `HeadStart`, `HeadEnd` — emitted verbatim; T4 adds no field.
- `scratchtree_test.go`'s `writeScratchTree` — the run-time fixture builder, for the two fixtures a
  committed tree cannot express.

**Gotchas.**

- `SpansOf` is the wrong seam for `Resolve` (D9): it is per-glyph, and using it would re-parse each
  unit once per target — four times over in the criterion's own call. It stays as the single-glyph
  convenience T3's round trip is written against.
- `symbolsOfUnit` returns the **union** when `unitDirs` reports a collision, and `SpansOf` ignores
  the flag. `Resolve` must call `unitDirs` itself to see `collision` at all.
- `glyph.Parse` rejects an empty unit, a `.`/`..` segment and a bad rune, so an unspellable unit is
  a D8 error entry and never reaches a directory read. T3's walk *lists* such a directory's files
  with no symbols; the lookup *rejects* the glyph. Two dispositions of one fact — assert them
  separately, as T3's plan 05 card 34 does.
- A glyph target is anything containing `#`, tested before parsing. A path target is everything
  else. `resolve` never guesses which one was meant — §4 states the rule in one line.
- `treesitter.WithTree` invalidates the root node when it returns; nothing may retain a `*ts.Node`
  past its callback. This is `symbolsOfUnit`'s problem, not T4's, but it constrains any helper T4
  adds that touches a tree.
- The build needs cgo: `CGO_ENABLED=1`, and `go build ./... && go test ./...` is the gate.

## Constraints

- Go repository. **No Python** (`CLAUDE.md`).
- No new module dependency. `go.mod` keeps tree-sitter and the Go grammar and nothing else. `glyph`
  stays pure Go with no dependencies, and T4 imports it without modifying it.
- The engine never re-implements the glyph grammar: parsing, printing and canonicalisation are
  `glyph`'s alone. Every alphabet question is one call to `glyph.Parse`.
- The `glyph/` package is the only glyph parser; **this task adds no parsing**.
- No tracked file may carry a machine path. The Loomyard checkout comes from `LADDER_LOOMYARD_REPO`;
  the gitignored `.scratch/ladder.env` holds it per machine, as T2 and T3 do.
- Scratch goes under `.scratch/`, never `/tmp` or any system temporary directory. `t.TempDir()` is
  banned for the same reason.
- No cache, index, daemon or concurrency. The per-call memo (D9) is a local variable and is
  discarded when the call returns; nothing is stored on `Repo` (T3's D22).
- The emitted key set is §4's and is closed. T4 adds two payload types and adds no key to `Symbol`,
  `DirAnswer` or `FileEntry`.
- The status vocabulary is glyph.md §5's four words and is closed (D8's error entry carries no
  status rather than inventing a fifth).
- Implementation does not begin until T3 is merged to `main`, `main` is merged into this branch, and
  the plan is revised against the merged code.
- `CGO_ENABLED=1 go build ./... && go test ./...` green, `go vet ./...` clean, `go mod tidy` leaving
  no diff, and one merge to `main`.

## Testing

**TDD candidates** (pure decision functions over data, no tree-sitter, write the table first):

1. The status rule of D5/D6 — a function from `(glyph, matches, collision)` to a `Status`. Table:
   zero matches; one match; one match under a collision; several `init`; several non-`init`; several
   under a collision; an owned name (`T.M`) with several matches. This is the decision the whole
   verb turns on and it needs no parse to test.
2. The target split — a function from a target string to glyph-or-path. Table: `a/b#C`, `a/b`,
   `#x`, `a#b#c`, `README.md`, `` (empty), `.`.

**Fixture-driven unit tests** over `internal/engine/testdata/`, per D17:

3. `found` — a function and a method resolve to exactly one symbol each, with `File`, `Start`,
   `SigEnd`, `End`, `Signature` and `Doc` as the walk reports them, and `ID` the canonical glyph.
4. `multipart` — the three-`init` package: one result, `status: multipart`, three symbols, in
   file-then-line order.
5. `ambiguous` by build tags — `testdata/tags/`: one result, `status: ambiguous`, both declarations
   in `Candidates` with their files, `Symbols` absent.
6. `ambiguous` by unit collision — the run-time `.scratch/` tree: `status: ambiguous` with the union,
   including the single-match case that D6 says is still ambiguous, and the zero-match case that is
   `not_found` with `unit: found`.
7. `not_found` both ways — an existing unit with a missing name gives `unit: found`; a unit whose
   directory does not exist gives `unit: not_found`. Assert the *marshalled JSON* on at least one of
   each, so §5's `"unit": "found"` spelling and the `omitempty` on every absent key are pinned.
8. The grouping guarantee (D9) — a call naming several glyphs across several units parses each unit
   exactly once. Assert on a counter the test can read directly rather than on a package-level
   global: the memo is an explicit type the test constructs and drives, so production code carries no
   test-only variable.
9. Argument order and 1:1 (D2) — a call mixing glyphs and paths, valid and invalid, returns exactly
   `len(targets)` results in order, with a repeated target answered twice.
10. Path targets (D7) — an existing file (a one-entry `DirAnswer` carrying the enclosing directory's
    `dir`/`package`/`language`/`doc`, with `symbols` **absent**), an existing directory, a
    nonexistent path (`not_found`, no `unit` key), a gitignored path that exists (answered, per T3's
    D20), and an absolute or `..`-escaping path (D8's error entry).
11. D8's error entries — a malformed glyph per distinct `glyph.Reason` the engine can reach
    (`member_too_deep`, `unit_bad_rune`, `unit_dot_segment`, `member_keyword`), each with `Status`
    empty, `Error` non-empty and `Reason` the grammar's own word; and one call mixing a malformed
    target with valid ones, asserting the valid answers survive.
12. `expand` on a struct — head is the type entry with span from `HeadStart`/`HeadEnd`, members are
    its methods from **several files**, sorted by file then line, the type symbol itself absent from
    `Members`.
13. `expand` on an interface — head covers the whole declaration, every member span lies inside the
    head range, and the members are the `method_elem`s with the interface as owner. This is D12's
    shape assertion and the reason no discontiguous span is emitted.
14. `expand` on a type with no members — `found`, `Head` set, `Members` absent.
15. `expand` failures — a `func` glyph and a `const` glyph each give `*NotATypeError` with the right
    `Kind`, asserted with `errors.As`; a malformed glyph and a target with no `#` each give an error
    and no answer; a nonexistent name gives `not_found` with `Unit` set, not an error; a build-tag
    pair of types gives `ambiguous` with candidates and no head.
16. Ordering (D15) — a unit whose files are not in lexicographic `os.ReadDir` order, asserting both
    verbs' outputs come back file-then-line regardless.

**Loomyard, env-gated per D17's skip/fail rule, skipped under `-short`:**

17. `TestResolve_TwentyGlyphsUnder150ms` — asserts all twenty resolve `found` first, then the
    minimum of five timed calls is under 150 ms.
18. `BenchmarkResolveTwentyGlyphs` — the same call, kept.
19. A spot check that `expand` of a real Loomyard type returns members from more than one file — the
    property the verb exists for, and one no committed fixture proves as convincingly.

**Gate.** `CGO_ENABLED=1 go build ./... && CGO_ENABLED=1 go test ./...`, plus `go vet ./...` and
`go mod tidy` leaving no diff. Every T3 test must keep passing untouched: T4 adds files and adds
types to `answer.go`, and changes no walk rule, no `toc` answer and no existing key.

## Q&A log

- **Q:** Where do the two verbs live? **A:** [auto-pick] `resolve.go` grows and `expand.go` is new, both in `internal/engine`. **Why:** plan §12 forbids a package per verb, and T3's D16 put the span lookup in `resolve.go` precisely so T4 would grow that file rather than move it.
- **Q:** What is `Resolve`'s signature? **A:** [auto-pick] `Resolve(targets []string) ([]ResolveResult, error)`, one result per argument in argument order. **Why:** §8.1 puts every glyph of a draft in one call, and the validator has to map each answer back to the card that asked; a positional 1:1 slice makes that total and free.
- **Q:** Does the output group by unit? **A:** [auto-pick] No — grouping is the parse guarantee, not the shape. **Why:** §5's sentence is about cost, and a grouped output's natural `unit` key collides with §5's own `unit: found|not_found` spelling; the spelled contract is the one that must not move.
- **Q:** What is `Expand`'s signature? **A:** [auto-pick] `Expand(target string) (ExpandAnswer, error)`. **Why:** symmetry with `Resolve` and with T5b's argv; a `glyph.Glyph` parameter would put §8.1's layer 1 in the caller for one verb and not the other.
- **Q:** Which alphabet is a target parsed against? **A:** [auto-pick] `glyph.Go`, hardcoded, with the multi-alphabet rule named in the comment as the extension point. **Why:** the alphabet loop is behaviour, not shape — unlike T3's D11 `language` key, nothing about the answer's key set depends on it, and Go-only cannot produce a cross-language `ambiguous`.
- **Q:** How are `multipart` and `ambiguous` told apart in Go? **A:** [auto-pick] A bare package-level `init` with several matches is `multipart`; every other multi-match is `ambiguous`. **Why:** glyph.md §5 gives Go exactly one multipart case, and the discriminator is the glyph's own name, decidable without evaluating build tags the engine has no business interpreting.
- **Q:** Where does `unit: found|not_found` come from? **A:** [auto-pick] `len(unitDirs(unit)) > 0`. **Why:** §5 defines it as directory existence, which is exactly `unitDirs`'s question; deriving it separately would be a second unit→directory implementation in one package, drifting on the very corner case the rule was written for.
- **Q:** What does the `unitDirs` collision resolve to? **A:** [auto-pick] `ambiguous` over the union whenever there is at least one match — a single match included — and `not_found` with `unit: found` when there is none. **Why:** T3's D16 says T4 promotes the flag to `ambiguous`, and a `found` whose glyph names two different units is the exact failure literal-first exists to prevent.
- **Q:** How is a target that `glyph.Parse` rejects reported? **A:** [auto-pick] A per-entry `error` plus the grammar's `reason`, with `status` omitted. **Why:** the four statuses are resolution outcomes and a string that is not a glyph was never searched for; calling it `not_found` would tell a Create card the name is free when it is unspellable.
- **Q:** What does a path target's answer carry? **A:** [auto-pick] T3's `DirAnswer` from `TOC`, with symbols explicitly off. **Why:** §4 says a file query *is* a directory answer and the file never repeats its parent's facts, so a bare `FileEntry` is unreadable; reusing `TOC` inherits the gitignored-target, symlink and root rules for free.
- **Q:** How is §5's "each unit is parsed once" made true? **A:** [auto-pick] A per-call memo over `symbolsOfUnit`, discarded when the call returns; `SpansOf` is not used by `Resolve`. **Why:** twenty ungrouped lookups over five units is four parses each and lands far outside 150 ms; a memo that cannot outlive its call cannot go stale, so T3's D22 is untouched.
- **Q:** What exactly is `expand`'s head? **A:** [auto-pick] The type's own symbol entry with `Start`/`End` read from `HeadStart`/`HeadEnd`. **Why:** §4 says the three verbs return that entry and nothing else for a symbol; reading the head field rather than re-deriving it is D4's stated purpose, and the "minus its member spans" reading is the consumer's, done from the member entries already in the answer.
- **Q:** How does `expand` report a glyph naming a non-type? **A:** [auto-pick] A typed `*NotATypeError` carrying the id and the kind. **Why:** `ok: false` is the envelope's and T5b maps typed errors to it, exactly as T3's D20 split `ErrTargetOutsideRepo` from `ErrTargetNotFound` so a message could name the cause without parsing a string.
- **Q:** What are the ordering guarantees? **A:** [auto-pick] Resolve: argument order, 1:1; symbols and candidates by file then start line. Expand: head first, members by file then start line. **Why:** glyph.md §5 states the file-then-line rule as contract, and T3's D18 already made the engine the sorter; argument order is the only top-level order that keeps the 1:1 mapping usable.
- **Q:** How is the 150 ms criterion asserted without flakiness? **A:** [auto-pick] Best-of-five minimum, env-gated with D17's pin rule, skipped under `-short`, with the twenty glyphs asserted `found` first and a `Benchmark` kept beside it. **Why:** the minimum measures the floor the criterion is about; asserting `found` first stops a drifted glyph list turning the criterion green by timing twenty misses.
- **Q:** Which fixtures cover glyph.md §5? **A:** [auto-pick] Committed fixtures for `found`, `multipart`, build-tag `ambiguous` and both `not_found` shapes; run-time `.scratch/` trees for the collision `ambiguous` and the ignore-filter case. **Why:** §12's done-when makes the fixtures the deliverable, and the two exceptions are the shapes a committed tree provably cannot express — the same split T3 made, with the same helper.
- **Q:** Does T4 amend `docs/glyph.md`? **A:** [auto-pick] No — two gaps are recorded in code comments and neither is closed. **Why:** T3 took the same position for the same reason: a single task changing the shared identifier contract is the coupling §7's ordering avoids, and both gaps should be decided against a repository that needs one.
