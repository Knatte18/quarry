# Plan: resolve + expand (T4)

```yaml
task: "resolve + expand (T4)"
slug: "resolve-expand"
approved: false
started: "20260904-051705"
parent: "main"
root: ""
verify: CGO_ENABLED=1 go vet ./...
```

## Batch Index

_The fenced yaml block below is the authoritative DAG mill-go reads to schedule batches.
Every batch lives at `NN-<batch-slug>.md` in this directory and is mirrored as one entry here._

```yaml
batches:
  - number: 1
    name: answer-types
    file: 01-answer-types.md
    depends-on: []
    verify: CGO_ENABLED=1 go build ./... && CGO_ENABLED=1 go vet ./internal/engine/
  - number: 2
    name: fixtures
    file: 02-fixtures.md
    depends-on: []
    verify: CGO_ENABLED=1 go test ./internal/engine/
  - number: 3
    name: resolve
    file: 03-resolve.md
    depends-on: [1, 2]
    verify: CGO_ENABLED=1 go test ./internal/engine/
  - number: 4
    name: expand
    file: 04-expand.md
    depends-on: [1, 2, 3]
    verify: CGO_ENABLED=1 go test ./internal/engine/
  - number: 5
    name: loomyard-timing
    file: 05-loomyard-timing.md
    depends-on: [3, 4]
    verify: CGO_ENABLED=1 go test ./internal/engine/
```

## Shared Decisions

### Decision: both verbs grow the engine package, never a package per verb

- **Decision:** `Resolve` and its helpers go into the existing `internal/engine/resolve.go`, beside
  `SpansOf`, `symbolsOfUnit` and `unitDirs`. `Expand` and `NotATypeError` go into a new
  `internal/engine/expand.go`. `Status`, `ResolveResult` and `ExpandAnswer` go into
  `internal/engine/answer.go`, beside `Symbol`, `DirAnswer` and `FileEntry`. No new package, no new
  subpackage.
- **Rationale:** the rewrite plan says the engine is one package with files per concern and never a
  package per verb, and T3 put the span lookup in `resolve.go` so this task would grow that file
  rather than move it. `answer.go`'s own header comment already declares it the home of the emitted
  key set; splitting the two verbs' answer types away from `Symbol` would put half the contract in one
  file and half in another.
- **Applies to:** all batches

### Decision: the status vocabulary is closed, and a rejected target carries no status

- **Decision:** `Status` has exactly four values — `found`, `not_found`, `ambiguous`, `multipart` —
  and no fifth is added. A target string the grammar rejects, and a path target that leaves the
  repository, are reported per entry with `Error` and (for a grammar rejection) `Reason`, with
  `Status` left empty.
- **Rationale:** the four statuses are resolution outcomes, and a string that is not a glyph was never
  searched for. Calling it `not_found` would tell a plan validator the name is free when the truth is
  that the name is unspellable — exactly the guess the contract forbids. Parsing is layer 1 and
  resolution is layer 2, so a layer-1 failure is not a layer-2 answer.
- **Applies to:** answer-types, resolve, expand

### Decision: one error boundary, stated exhaustively

- **Decision:** a `ResolveResult` expresses either a resolution outcome or a pre-resolution rejection
  of the target string, and never an engine failure. The per-entry `Error` domain has exactly two
  members: a `glyph.Parse` rejection, and a path target rejected as outside the repository. Every
  other engine error — a directory read failure, an ignore-file read failure, anything a future engine
  change adds — fails the whole `Resolve` call, which returns a nil slice and that error. `Expand`
  answers one target and therefore returns every failure as its own error.
- **Rationale:** an engine failure is not an answer about a glyph, and putting "this target is not
  spellable" and "this machine could not read a directory" under one key would let a validator reject
  a card over a disk error. Failing the call is also the only disposition that cannot silently
  under-report: a unit that failed to read would otherwise answer `not_found` for every glyph in it,
  which a delete-card's done-check reads as success. A missing unit directory is not that case and
  never reaches this rule — `unitDirs` returns an empty slice and `symbolsOfUnit` over zero
  directories returns an empty slice and a nil error, which is what keeps `unit: not_found` reachable.
- **Applies to:** resolve, expand

### Decision: one decision function, so the two verbs cannot disagree

- **Decision:** `statusForMatches(g, matches, collision)` in `internal/engine/resolve.go` is the only
  place a match set becomes a status, and both verbs call it. Its row order is the rule: zero matches
  first, then the collision, then the single match, then the bare `init` multipart case, then
  ambiguous. `Expand` adds its kind rows after those two, never before them.
- **Rationale:** a caller that asks `expand` a question should not have to call `resolve` first to
  learn that the glyph is a typo or that two build-tagged types answer to it, and a shared rule is
  what stops `expand` answering `found` where `resolve` answers `ambiguous` for the same glyph. The
  zero-match row is first so a collision with nothing in either directory answers `not_found` with
  `unit: found` rather than an ambiguous with nothing to be ambiguous between; the collision row is
  next so a single match under a collision is never reported as if the glyph named one unit.
- **Applies to:** resolve, expand

### Decision: one unit is parsed once per call, and the guarantee has a seam

- **Decision:** a named unexported `unitMemo`, built by the exported entry point and discarded when it
  returns, memoises `symbolsOfUnit` and `unitDirs` per unit and carries a `parses` counter that
  production code never reads. Both exported verbs are thin shells over an unexported worker taking
  the memo, so a test can construct one, pass it in, and assert on the counter. `SpansOf` is used by
  neither verb.
- **Rationale:** grouping by unit is the difference between the timing criterion passing and failing —
  twenty ungrouped lookups over five units is four parses of each. A plain pair of maps gives the test
  nothing to assert on: the map's entry count is true by construction, and wall-clock is the very
  thing the test must be independent of. A memo that cannot outlive the call it was built in cannot go
  stale, so the engine's no-cache rule is untouched and nothing is stored on `Repo`.
- **Applies to:** resolve, expand, loomyard-timing

### Decision: the emitted key set is closed and `Symbol` is unchanged

- **Decision:** this task adds two payload types and adds no field to `Symbol`, `DirAnswer` or
  `FileEntry`, and changes no existing JSON tag. `expand`'s head is the type's own symbol entry with
  `Start` and `End` substituted from `HeadStart` and `HeadEnd`, not a shape of its own and not
  rendered source text.
- **Rationale:** all three verbs return one symbol entry and nothing else for a symbol. Reading the
  head span from the head fields rather than re-deriving it is what gives those fields their consumer,
  and for Go the two pairs are identical so nothing observable changes today. The substitution does
  mean the type's full declaration span is not recoverable from an `ExpandAnswer` for a language whose
  head is a strict subset of its declaration — recorded as the fourth contract gap below rather than
  answered here, since with Go the only alphabet the case is unreachable.
- **Applies to:** answer-types, expand

### Decision: contract gaps are recorded in code comments, none is closed

- **Decision:** no line of `docs/glyph.md` or `docs/rewrite-plan.md` changes. Four gaps are recorded
  as comments where the rule they affect is implemented: the external test unit versus a real
  directory of the same name (recorded in the glyph branch, where the collision flag is read from the
  memo — not on the decision function that promotes it, which sees a bare bool and knows nothing about
  unit directories); `ambiguous`
  candidates carrying no language marker (recorded on the `Candidates` key); a unit reached
  through an intermediate symlinked directory resolving here while the walk never lists it (recorded
  where the unit-existence key is derived); and the type's full declaration span being unrecoverable
  from an `ExpandAnswer` once the head span is substituted, for a language whose head is a strict
  subset of its declaration (recorded at that substitution). The first three are the discussion's own;
  the fourth surfaced during plan review and is recorded on the same footing, since the discussion's
  position is that this task records the gaps it runs into and closes none of them.
- **Rationale:** a single task changing the shared identifier contract is the coupling the plan's
  task ordering avoids, and both candidate answers for a gap should be decided against a repository
  that needs one. Recording a gap where the behaviour is implemented is what keeps it findable;
  recording it only in a discussion file would lose it. The third gap in particular is inherited
  unchanged rather than narrowed — changing `dirExists` would be a change to the walk's inverse, which
  this task's scope excludes.
- **Applies to:** resolve, expand

### Decision: fixtures are committed unless a named T3 assertion forbids it

- **Decision:** a fixture is committed under `internal/engine/testdata/` when it adds no package to a
  directory an existing test enumerates, and built at run time under `.scratch/` otherwise. The two
  new committed packages are siblings under `internal/engine/testdata/`, not children of its `tree/`
  fixture and not new files in its `glyphs/` fixture. The unit collision is built at run time, because
  `TestSpansOf_LiteralFirst` asserts the committed `foo_test` fixture has no sibling and committing
  one would break it. `t.TempDir` is never used — the system temporary directory is banned — and the
  existing `writeScratchTree` and `openScratchRepo` helpers are reused rather than rewritten.
- **Rationale:** the placement rule is narrower than "do not disturb testdata": what constrains each
  case is one named assertion, and a fixture that trips no named assertion is committed, where a
  reader can open it. Reading declarations already present in `glyphs/` cannot perturb the assertions
  over that package's whole symbol list; adding a file to it would.
- **Applies to:** fixtures, resolve, expand

### Decision: scope exceptions are exactly two, each its own card

- **Decision:** the only changes to code T3 wrote for its own purposes are one doc-comment sentence in
  `internal/engine/golang.go` and one parameter type in `internal/engine/loomyard_test.go`. Both are
  in batch 5, each as its own card. Adding the two verbs and their types to `answer.go`, `resolve.go`
  and `expand.go` and extending `resolve_test.go` is the task itself, not an exception to it.
- **Rationale:** neither exception changes behaviour, and giving each its own card is what lets a
  reviewer see it in isolation rather than buried in a batch of new code. Nothing else under
  `internal/engine` changes behaviour: no walk rule, no `toc` answer, no ignore matcher, no strategy,
  and no `Symbol` key.
- **Applies to:** loomyard-timing

### Decision: verify commands are cgo-enabled and scoped to the engine package

- **Decision:** every batch's `verify:` is a native `go` invocation with `CGO_ENABLED=1`, scoped to
  `./internal/engine/`, with one deliberate exception: batch 1's `verify:` opens with a module-wide
  `go build ./...` before its engine-scoped vet. That batch adds exported types to a package the rest
  of the module compiles against and has no test of its own to run, so a module-wide build is the only
  thing that can fail there and is what makes the batch verifiable at all. The module-wide `verify:`
  is `CGO_ENABLED=1 go vet ./...`.
- **Rationale:** the tree-sitter binding needs cgo, so a verify without it does not build. Scoping the
  test run to the one package every card in this plan touches is the right scope and not an over-broad
  one: the package's whole suite runs in well under a second on this host, and every existing test in
  it is a must-keep-passing gate for a task that grows files T3 wrote. The module-wide `go vet ./...`
  is the cheap cross-package regression check, and it runs at **every** batch boundary without being
  repeated in any batch's own `verify:` — that is what the overview-level `verify:` key is: mill-go
  runs it after each batch's own command passes, against a baseline computed once before the first
  batch. Copying it into each batch's `verify:` would run it twice per boundary and gain nothing.
- **Applies to:** all batches

### Decision: the configured done gate carries pre-existing lint debt this task does not own

- **Decision:** `pipeline.done_gate` in the hub config is `go test ./... && golangci-lint run`. This
  plan does not change it, and no card touches `bench/`.
- **Rationale:** `golangci-lint run` exits 1 on this branch's tip today, before any of this plan's
  changes, on three pre-existing `errcheck` findings in `bench/loomyard-eval/ladder/internal/ladder/`
  — two unchecked `release()` calls and one unchecked `Write`. The task's scope excludes changes to
  `bench/` in as many words, so fixing them here would be scope creep, and silently rewriting the hub
  config mid-task would be a config mutation with no bootstrap card behind it. The consequence is
  recorded rather than worked around: the done gate will fail on lint at the end of this task for
  reasons that predate it, and clearing those three findings is the operator's call — either as a
  separate task or by narrowing `done_gate` before this task finishes.
- **On the config file's own inline comment:** the `done_gate` key carries a trailing comment, written
  by the engine-core (T3) plan, saying golangci-lint "reports no finding on the current tip, so the
  gate carries no pre-existing debt". That comment is not current, and this plan's measurement is. The
  three findings sit in code committed by the ladder-harness (T2) commit, which predates the T3 commit
  that wrote the comment, so they were already present when it was written — this is not debt that
  arrived afterwards and made a true statement stale. Whatever produced the comment's conclusion, a
  direct `CGO_ENABLED=1 golangci-lint run` from the repository root on this branch's tip exits 1 with
  those three findings, checked while writing this plan against a clean tree. The config file itself is
  left untouched, as this Decision requires; correcting or removing that comment belongs to whoever
  changes the key.
- **Applies to:** all batches

## All Files Touched

- `internal/engine/answer.go`
- `internal/engine/expand.go`
- `internal/engine/expand_test.go`
- `internal/engine/golang.go`
- `internal/engine/loomyard_test.go`
- `internal/engine/loomyard_timing_test.go`
- `internal/engine/resolve.go`
- `internal/engine/resolve_test.go`
- `internal/engine/testdata/methods/aardvark.go`
- `internal/engine/testdata/methods/widget.go`
- `internal/engine/testdata/tags/linux.go`
- `internal/engine/testdata/tags/other.go`
