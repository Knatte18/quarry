# Batch: answer-types

```yaml
task: "resolve + expand (T4)"
batch: "answer-types"
number: 1
cards: 2
verify: CGO_ENABLED=1 go build ./... && CGO_ENABLED=1 go vet ./internal/engine/
depends-on: []
```

## Batch Scope

This batch adds the two verbs' answer shapes — `Status` with its four constants, `ResolveResult`, and
`ExpandAnswer` — to `internal/engine/answer.go`, beside the `Symbol`, `DirAnswer` and `FileEntry`
types already declared there. It is one batch because it is one file and one contract: every later
batch's code and every test in this plan reads these types, and nothing in this batch has behaviour
to test beyond compiling and vetting. It touches no existing declaration in `answer.go` — no field
is added to `Symbol`, `DirAnswer` or `FileEntry`, and no existing JSON tag changes.

The external interface the next batches consume is exactly these three type declarations plus the
four `Status` constants.

## Cards

### Card 1: Status and its four constants

- **Context:**
  - `docs/glyph.md`
  - `internal/engine/resolve.go`
- **Edits:**
  - `internal/engine/answer.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add a `Status` string type and its four constants to `internal/engine/answer.go`,
  placed after the existing `Kind` constant block and before the `Symbol` declaration, so the file's
  two closed vocabularies sit together. Declare exactly `type Status string` and the constants
  `StatusFound Status = "found"`, `StatusNotFound Status = "not_found"`,
  `StatusAmbiguous Status = "ambiguous"` and `StatusMultipart Status = "multipart"`, in that order.
  There is no fifth value: a target string the grammar rejects carries no status at all (see card 2's
  `Error`/`Reason` keys), and the vocabulary is closed by docs/glyph.md §5. Write the doc comments in
  this package's established register — a sentence per constant naming what the value means, and a
  type doc comment stating that `Status` is the closed per-entry vocabulary of docs/glyph.md §5,
  shared by `ResolveResult`'s `Status` and `Unit` keys and by `ExpandAnswer`'s. State in the type doc
  comment that the `Unit` key of both result types draws from this same type but only ever carries
  `StatusFound` or `StatusNotFound`, so the package holds one vocabulary rather than two overlapping
  ones. Change no existing declaration, tag or comment in this file.
- **Commit:** `feat(engine): add the closed Status vocabulary to answer.go`

### Card 2: ResolveResult and ExpandAnswer

- **Context:**
  - `docs/glyph.md`
  - `internal/engine/repo.go`
- **Edits:**
  - `internal/engine/answer.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add `ResolveResult` and `ExpandAnswer` to `internal/engine/answer.go`, after
  `FileEntry` and before the `DepthAll` constant, with exactly these fields, types and JSON tags and
  no others.

  `ResolveResult` — `Target string` tagged `target`; `ID string` tagged `id,omitempty`;
  `Status Status` tagged `status,omitempty`; `Unit Status` tagged `unit,omitempty`;
  `Symbols []Symbol` tagged `symbols,omitempty`; `Candidates []Symbol` tagged `candidates,omitempty`;
  `Dir *DirAnswer` tagged `dir,omitempty`; `Error string` tagged `error,omitempty`;
  `Reason string` tagged `reason,omitempty`.

  `ExpandAnswer` — `ID string` tagged `id`; `Status Status` tagged `status`; `Unit Status` tagged
  `unit,omitempty`; `Head *Symbol` tagged `head,omitempty`; `Members []Symbol` tagged
  `members,omitempty`; `Candidates []Symbol` tagged `candidates,omitempty`.

  `Dir` and `Head` are pointers so an absent answer is dropped by `omitempty`; a non-pointer struct
  would always marshal. `Reason` is a plain `string`, deliberately not `glyph.Reason`: the emitted
  JSON is a plain word and `answer.go` needs no exported alias for the grammar's own type.

  Document each field to this file's standard. Record on `Target` that it is the caller's argument
  verbatim and is always present. Record on `ID` that it is the parsed glyph's `String()` form, the
  same wire form `Symbol.ID` carries, present only for a glyph target, and that for Go it is
  byte-identical to `Target` on every non-error result because the Go alphabet normalises nothing —
  the key earns its place as parity with `Symbol.ID`, not as canonicalisation. Record on `Status`
  that it is absent only when the target never reached resolution. Record on `Unit` that it is set
  only on a glyph `not_found`, that it carries only `StatusFound` or `StatusNotFound`, and that it is
  never set on a path result because a path belongs to no unit. Record on `Symbols` that it carries
  the matches for `found` (exactly one) and for `multipart` (every declaration). Record on
  `Candidates` that it carries the matches for `ambiguous`, that the separate key is the signal that
  nothing was chosen, and — this is docs/glyph.md §5's second recorded gap, closed by nobody in this
  task — that §5 says candidates in a multi-language repository are marked by language while
  `Symbol`'s key set has no language key; with Go the only alphabet the case is unreachable, so no
  key is added and the marker is a second language's task to add against a real case. Record on
  `Error` and `Reason` that they carry a pre-resolution rejection of the target string itself and
  never an engine read failure, and that `Status` is empty whenever `Error` is set. Record on
  `ExpandAnswer.Status` that it is `found`, `not_found` or `ambiguous` and never `multipart`, because
  the kind gate sends every match set holding no type to the `*NotATypeError` batch 4 adds — which is
  where a several-declaration `init` glyph lands, however many declarations it has — and docs/glyph.md
  §5's rule that a Go type never splits closes the remaining type-only cases; a language with partial
  types adds its row then, not now. Change no existing declaration, tag or comment in this file.
- **Commit:** `feat(engine): add ResolveResult and ExpandAnswer payload types`

## Batch Tests

`verify:` is a build plus a vet of the engine package, not a test run: this batch adds type
declarations and doc comments only, and has no behaviour of its own to assert. The types are
exercised by every test in batches 3, 4 and 5 — the JSON tags specifically by batch 3's card 11,
which marshals a `not_found` result and pins docs/glyph.md §5's `"unit": "found"` spelling and the
`omitempty` on every absent key. `go vet` is the meaningful gate here because struct-tag syntax
errors are exactly what it catches and the compiler does not.
