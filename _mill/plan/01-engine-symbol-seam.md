# Batch: engine-symbol-seam

```yaml
task: 'P2 — diff-to-symbols: changed file versions to symbol-table delta (roadmap 2c)'
batch: 'engine-symbol-seam'
number: 1
cards: 5
verify: go test ./internal/engine/ -run 'TestSymbolOffsets|TestSignatureSpan'
depends-on: []
```

## Batch Scope

This batch adds the one seam the whole delta core is built on: three JSON-hidden byte offsets on
`engine.Symbol` — `DeclStart`, `BodyStart`, `DeclEnd` — filled by every builder in the Go strategy
from the same node it already hands to the signature cutter.
Without them nothing in the extractor's output can reach a symbol's declaration node, and the token
streams batch 3 defines have nothing to key on.

The batch is one unit because the field declaration and the six builder call sites are one change:
a field added and left unfilled by some builders would produce zero offsets that read as a legal
byte range starting at position zero, which is the silently-wrong state this batch exists to avoid.
It is a root batch — it shares no file with batch 2 or batch 5.

The external interface batch 3 consumes: for every `Symbol` the Go strategy emits, `DeclStart` is
the declaration node's start byte, `DeclEnd` is that node's end byte, and `BodyStart` is the
body-bearing child's start byte when there is one and equals `DeclEnd` when there is not — so a
bodyless declaration has an empty body range and needs no separate branch downstream.

Batch-local decision: the three fields are added to the existing `Symbol` struct rather than to a
parallel type, following the precedent `HeadStart`/`HeadEnd` already set in that struct for
JSON-hidden fields that exist for exactly one consumer.

## Cards

### Card 1: declare the three JSON-hidden byte offsets on Symbol

- **Context:**
  - `internal/engine/nodes.go`
  - `internal/engine/answer_test.go`
- **Edits:**
  - `internal/engine/answer.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add three `int` fields to the `Symbol` struct — `DeclStart`, `BodyStart` and
  `DeclEnd` — each tagged `json:"-"`, declared together immediately after the existing `HeadStart`
  and `HeadEnd` pair.
  Their doc comment must state: they are byte offsets into the file's own source, not line numbers;
  they are JSON-hidden for the same reason `HeadStart` and `HeadEnd` are, namely that they exist for
  one consumer rather than for the wire; `DeclStart` and `DeclEnd` are the declaration node's start
  and end bytes; `BodyStart` is the body-bearing child's start byte when the declaration has one and
  equals `DeclEnd` when it does not, so `BodyStart == DeclEnd` is the marker for a declaration with
  no body at all and the body byte range is empty rather than absent; and that the span
  `[DeclStart, BodyStart)` is the same span `SignatureCut` is given, so a text comparison over the
  signature and a token comparison over it are over the same bytes by construction.
  Note in that comment that the emitted key set is unchanged by this addition, which is what keeps
  this file's own header rule — every JSON tag here is the closed emitted key set — satisfied
  without a Shared Decision change.
  Do not add or rename any emitted key.
- **Commit:** `feat(engine): add JSON-hidden declaration byte offsets to Symbol`

### Card 2: fill the offsets for functions, methods and interface methods

- **Context:**
  - `internal/engine/answer.go`
  - `internal/engine/nodes.go`
  - `internal/engine/strategy.go`
- **Edits:**
  - `internal/engine/golang.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Fill the three new offsets in `goDeclSymbol`, which builds the Symbol shared by
  function and method extraction, and in `goInterfaceMethodSymbols`, which builds one Symbol per
  interface `method_elem`.
  In `goDeclSymbol` the declaration node is its `decl` parameter and the body node is its `body`
  parameter — the same two nodes it already passes to `SignatureCut` — so `DeclStart` is
  `decl.StartByte()`, `DeclEnd` is `decl.EndByte()`, and `BodyStart` is `body.StartByte()` when
  `body` is non-nil and `DeclEnd` otherwise.
  In `goInterfaceMethodSymbols` the declaration node is the `method_elem` node it already passes to
  `SignatureCut` with a nil body, so `DeclStart` and `DeclEnd` come from that node and `BodyStart`
  equals `DeclEnd`.
  Introduce one unexported helper in this file that computes the triple from a declaration node and
  a possibly-nil body node, and call it from every builder this batch touches, so the
  nil-body rule has exactly one implementation rather than six.
  Change no existing field of any Symbol built here, and change no line number, signature or
  docstring derivation.
- **Commit:** `feat(engine): fill declaration byte offsets for func, method and interface-method symbols`

### Card 3: fill the offsets for the two type builders

- **Context:**
  - `internal/engine/answer.go`
  - `internal/engine/nodes.go`
- **Edits:**
  - `internal/engine/golang.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Fill the three offsets in `goUngroupedTypeSymbol` and in `goGroupedTypeSymbol`,
  using the same helper card 2 introduced.
  The node each builder passes is the node it already hands `SignatureCut`, and the two builders
  deliberately differ: `goUngroupedTypeSymbol` cuts from the `type_declaration` node it is given as
  `decl`, so its offsets come from that node; `goGroupedTypeSymbol` cuts from the spec node, so its
  offsets come from the spec.
  The body node in both is the result of `goTypeBody` applied to the spec — which returns a
  `field_declaration_list` for a struct, the bare `"{"` leaf for an interface, and nil for
  `type ID string` and `type Alias = T`.
  A nil result means `BodyStart` equals `DeclEnd`, matching the signature cutter's own nil-body
  branch and the zero `SigEnd` those shapes already carry.
  Requiring the offsets to come from whichever node the builder passed is what keeps the byte spans
  and the `Signature` string describing the same declaration for the grouped shape, whose
  `Signature` is a synthesized `"type "` prefix plus the spec's own cut text.
- **Commit:** `feat(engine): fill declaration byte offsets for type symbols`

### Card 4: fill the offsets for the const and var builders

- **Context:**
  - `internal/engine/answer.go`
  - `internal/engine/nodes.go`
- **Edits:**
  - `internal/engine/golang.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Fill the three offsets in `goUngroupedConstOrVarSymbols` and in
  `goGroupedConstOrVarSymbols`, using the same helper card 2 introduced.
  Both builders pass a nil body to `SignatureCut` today, so `BodyStart` equals `DeclEnd` for every
  const and var symbol; that is the intended outcome and is what the exact-tier rename classifier's
  condition 7 later keys on to keep bodyless kinds out of the tier quarry asserts.
  The node differs between the two, exactly as it does for the type builders:
  `goUngroupedConstOrVarSymbols` cuts from the declaration node, `goGroupedConstOrVarSymbols` from
  the spec node.
  Several names declared in one spec — the `const a, b = 1, 2` shape — each get their own Symbol
  carrying the identical offsets, since they genuinely share one declaration span; do not attempt to
  narrow a per-name span.
- **Commit:** `feat(engine): fill declaration byte offsets for const and var symbols`

### Card 5: offset and signature-span tests, per builder shape

- **Context:**
  - `internal/engine/answer.go`
  - `internal/engine/golang.go`
  - `internal/engine/nodes.go`
  - `internal/engine/golang_test.go`
  - `internal/engine/treesitter/treesitter.go`
- **Edits:** none
- **Creates:**
  - `internal/engine/offsets_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add `TestSymbolOffsets_PerShape` and `TestSignatureSpanInvariant_PerShape` in
  package `engine`, both driven by table cases over in-memory Go source parsed through
  `treesitter.WithTree` — no fixture files.
  `TestSymbolOffsets_PerShape` covers all six builder shapes: an ungrouped function, a method, an
  ungrouped type with a struct body, a grouped type, an interface method element, an ungrouped
  const, a grouped var, and a type alias.
  For each it asserts that `DeclStart` and `DeclEnd` bracket the declaration text the shape's
  builder cut from, and that `BodyStart` equals `DeclEnd` exactly for the shapes with no
  body-bearing child — every const, every var, a type alias, and an interface method element — and
  is strictly between `DeclStart` and `DeclEnd` for a function, a method, a struct type and an
  interface type.
  `TestSignatureSpanInvariant_PerShape` asserts the invariant the discussion states per shape, and
  deliberately not as a flat byte identity, which would be false: the source bytes in
  `[DeclStart, BodyStart)` with leading and trailing whitespace trimmed equal `Symbol.Signature`
  for an ungrouped function, method, type, const, var and interface method element; and equal
  `Symbol.Signature` with the synthesized `"type "`, `"const "` or `"var "` keyword prefix removed
  for the grouped type, grouped const and grouped var shapes.
  Both tests must fail if a builder is left unfilled, so assert on a non-zero `DeclEnd` explicitly
  rather than only on relations between the three values.
- **Commit:** `test(engine): assert declaration byte offsets and the signature-span invariant per builder shape`

## Batch Tests

`verify:` runs `go test ./internal/engine/ -run 'TestSymbolOffsets|TestSignatureSpan'`, which
selects exactly the two test functions card 5 creates in `internal/engine/offsets_test.go`.
The scope is deliberately narrow: this batch adds three fields and six assignments and changes no
existing behaviour, so the broader `internal/engine` suite has nothing new to say about it.
The module-wide `verify: go vet ./...` at this batch's boundary type-checks every package including
its test files, so a signature change that broke a caller fails there; it does not execute tests.
The whole `internal/engine` suite runs as a regression gate under the repository-wide
`pipeline.done_gate` — `go test ./... && golangci-lint run` — before the task is marked done, which
is where an accidental change to an existing line number, signature or docstring derivation would
surface.
