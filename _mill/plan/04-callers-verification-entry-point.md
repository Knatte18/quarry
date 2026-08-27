# Batch: callers-verification-entry-point

```yaml
task: "Improve gopls query precision (build tags + scoping)"
batch: "callers-verification-entry-point"
number: 4
cards: 5
verify: go vet -tags lsp ./... && go test ./internal/quarryengine/query/ ./quarry/
depends-on: [1, 3]
```

## Batch Scope

This batch delivers `query.Callers`, the one-connection entry point that resolves a position, builds a declaration match set from `textDocument/definition` and `textDocument/implementation`, issues `textDocument/references`, and then verifies each reference by issuing `textDocument/definition` at it — all on a single LSP connection, sequentially, under one phase deadline. It also re-exports the four new identifiers through `quarry/facade.go`, which is what makes them reachable from `internal/cli` in batch 5.

It is one batch because the pipeline generalization, the pure filter, and the entry point that composes them are a single design and cannot be reviewed apart. The external interface batch 5 consumes is `quarry.Callers`, `quarry.Options.SkipVerification`, `quarry.NormalizeBuildTags`, `quarry.ErrBuildTagsUnsupported` and its sentinel.

Batch-local decision, restating the overview's `verification-is-fail-closed-everywhere` Shared Decision in this batch's own terms: a declaration-side `textDocument/definition` that errors does not fail the call. Verification is skipped, every reference is kept, the returned declaration set is empty, and the error is not propagated — with one exception, that a server timeout on that call still sets the pipeline's `timedOut` flag so teardown disposes of a possibly-wedged connection correctly. That is deliberately different from today's `assert-no-callers`, which errors out when its separate `Definition` call fails; the gate staying red on a degraded server is the fail-closed outcome this task exists to produce.

## Cards

### Card 13: generalize the lookup pipeline to a multi-call connection scope

- **Context:**
  - `internal/quarryengine/query/symbol.go`
  - `internal/quarryengine/query/definition.go`
  - `internal/quarryengine/daemon/ensureserver.go`
- **Edits:**
  - `internal/quarryengine/query/refs.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  - Add `func runOnConnection(ctx context.Context, opts Options, fn func(ctx context.Context, client *lsp.Client, fileURI string, pos lsp.Position, timedOut *bool) error) error` to `internal/quarryengine/query/refs.go`. It performs exactly the steps `lookup` performs today up to and including `resolvePosition` — `detectAndRender`, `acquireConnection`, the deferred `teardownConnection` closing over a `timedOut` local, then `resolvePosition` — and then calls `fn` with the resolved client, file URI, position, and a pointer to that same `timedOut` local.
  - Rewrite `lookup` as a thin wrapper over `runOnConnection`: its `fn` runs the single `lspCall` under a fresh `context.WithTimeout(ctx, opts.Timeout)`, sets `*timedOut` when the error satisfies `errors.Is(err, quarryengine.ErrServerTimeoutSentinel)`, and stores the resulting locations for the wrapper to convert with `toSortedReferences`. `References` and `Definition` must behave identically to today.
  - Preserve `lookup`'s two existing invariants explicitly, and say so in `runOnConnection`'s doc comment: `timedOut` is captured by reference because a phase after the defer is registered can still set it, and `teardownConnection`'s `ConnKindSupervised` branch must keep returning with no shutdown handshake and no kill.
  - Do not change `acquireConnection`, `teardownConnection`, `resolvePosition`, or `toSortedReferences`. This card is a refactor with no behaviour change; the existing tests in that package are the proof.
- **Commit:** `refactor(query): factor the lookup pipeline into a reusable single-connection scope`

### Card 14: pure declaration-match and verification filter

- **Context:**
  - `internal/quarryengine/query/refs.go`
  - `internal/quarryengine/lsp/wire.go`
  - `internal/quarryengine/lsp/lspclient.go`
  - `docs/implementation-widening-spike.md`
- **Edits:** none
- **Creates:**
  - `internal/quarryengine/query/verify.go`
  - `internal/quarryengine/query/verify_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  - Create `internal/quarryengine/query/verify.go` in package `query` holding only pure functions over in-memory values — no `context`, no client, no I/O. Everything in this file must be testable without a fake server.
  - Declare `type locationKey struct { URI string; Line int; Character int }` and `func keyOf(loc lsp.Location) locationKey` built from `loc.URI` and `loc.Range.Start`. All comparison happens in LSP wire coordinates — 0-based line, UTF-16 character — before any conversion to the public 1-based `Reference` type, because `Reference.Character` and `quarryengine.Position.Character` do not share a coordinate system and comparing after conversion would reintroduce the byte-column hazard `internal/cli/cli.go`'s existing declaration-exclusion comment already documents.
  - Declare `type verificationOutcome struct { Locations []lsp.Location; Err error; Attempted bool }`, one per candidate reference. `Attempted` is false when the phase deadline expired before that reference's own definition call was made.
  - Declare `func filterVerifiedReferences(refs []lsp.Location, matchSet map[locationKey]bool, outcomes []verificationOutcome) []lsp.Location`. A reference at index i is dropped if and only if `outcomes[i].Attempted` is true, `outcomes[i].Err` is nil, `outcomes[i].Locations` is non-empty, and none of those locations' keys is present in `matchSet`. Every other case keeps the reference. This is the fail-closed rule expressed as one predicate; write it as one predicate rather than as several early returns, so a future edit cannot half-invert it.
  - Read the `mode:` line recorded in `docs/implementation-widening-spike.md` and implement exactly one of the two shapes below. Do not implement both, and do not add a runtime switch between them.
  - If the recorded mode is `symmetric`: declare `func declarationMatchSet(defLocs, implLocs []lsp.Location) map[locationKey]bool` returning the union of both location sets keyed by `keyOf`.
  - If the recorded mode is `directional`: declare `func declarationMatchSet(defLocs, implLocs []lsp.Location, interfaceDecl map[locationKey]bool) map[locationKey]bool` returning every `defLocs` key plus only those `implLocs` keys present in `interfaceDecl`. Also declare `func isInterfaceDeclaration(symbols []lsp.DocumentSymbol, pos lsp.Position) bool`, which walks the hierarchical `DocumentSymbol` tree, follows the chain of symbols whose `Range` contains `pos`, and reports whether any symbol in that chain has `Kind` equal to the LSP `SymbolKind` value for an interface, 11. Declare that 11 as a named constant with a comment naming it, rather than as a bare literal at the comparison site. The direction is read off the result this way, never guessed from the query.
  - Write `internal/quarryengine/query/verify_test.go` covering, as tables over hand-built `lsp.Location` values: a reference whose definition matches the match set is kept; one whose definition points elsewhere is dropped; one whose definition call errored is kept; one whose definition returned an empty list is kept; one never attempted is kept; two same-named declarations one line apart in the same file are distinguished, proving the match is positional and not file-level; and an empty match set keeps nothing and drops nothing because `filterVerifiedReferences` is never called with one (assert the behaviour its caller relies on: with an empty match set every attempted, non-empty, non-matching reference would be dropped, which is exactly why card 15 must not call it in that state).
  - If the recorded mode is `directional`, additionally cover `isInterfaceDeclaration`: a position inside an interface's method returns true, a position inside a concrete type's method returns false, a position matching no symbol returns false, and a nested chain where the interface is an ancestor rather than the innermost symbol returns true.
- **Commit:** `feat(query): add the pure declaration-match and fail-closed verification filter`

### Card 15: query.Callers, the one-connection verification entry point

- **Context:**
  - `internal/quarryengine/query/symbol.go`
  - `internal/quarryengine/query/definition.go`
  - `internal/quarryengine/lsp/lspclient.go`
  - `internal/quarryengine/query/verify.go`
  - `docs/implementation-widening-spike.md`
- **Edits:**
  - `internal/quarryengine/query/refs.go`
- **Creates:**
  - `internal/quarryengine/query/callers.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  - Add a `SkipVerification bool` field to `Options` in `internal/quarryengine/query/refs.go`, documented as negative polarity on purpose: `Options` is re-exported verbatim as the public `quarry.Options`, so the zero value must mean "verify" and a non-CLI caller must have to opt into the noisier, unverified behaviour explicitly.
  - Create `internal/quarryengine/query/callers.go` declaring `func Callers(ctx context.Context, opts Options) (references []Reference, declaration []Reference, err error)` implemented over `runOnConnection`, and an unexported `callersFromClient` seam holding all the logic, taking an already-built `*lsp.Client` plus the resolved `fileURI`, `pos`, the timeout, a `*bool` for the timed-out flag, and `opts.SkipVerification`. The seam exists so the pipeline can be driven against a hand-built client with no spawn, exactly as `symbolFromClient` in `internal/quarryengine/query/symbol.go` already is.
  - The order of LSP calls on the single connection is: `textDocument/definition` at the query position, then `textDocument/implementation` at the query position, then `textDocument/references` at the query position, then one `textDocument/definition` per returned reference. Each of the first three phases is bracketed by its own `context.WithTimeout(ctx, timeout)`, matching how the existing pipeline brackets its phases.
  - The `textDocument/references` result is the primary answer: an error from that call is returned as the function's error, unchanged, and sets the timed-out flag when it satisfies `errors.Is(err, quarryengine.ErrServerTimeoutSentinel)`.
  - Return contract: `references` is the filtered reference set and `declaration` is the **definition-only** result. The widened match set is internal and is never returned. Say so in the doc comment and say why: `internal/cli`'s existing `filterUnexpectedCallers` removes every returned declaration from the violation list, so returning the widened union would silently exclude every implementer's own declaration site from the gate. Both returned values are converted with `toSortedReferences`.
  - Verification is skipped entirely — every reference kept, `declaration` returned as whatever the definition call produced — in each of these cases: `opts.SkipVerification` is set; the declaration-side definition call errored; the declaration-side definition call returned an empty location set; `client.SupportsImplementation()` reports false; or the implementation call errored. In the two error cases, set the timed-out flag if the error is a server timeout, then continue rather than returning the error.
  - When verification does run, build the match set with `declarationMatchSet` as card 14 declared it. If card 14 implemented the directional shape, first issue one `textDocument/documentSymbol` per distinct file URI among the implementation results and classify each implementation location with `isInterfaceDeclaration`; a `documentSymbol` call that errors classifies that file's locations as not-interface, which keeps them out of the match set and can therefore only drop references that the symmetric shape would have kept — so treat that as a reason to skip verification entirely rather than to narrow the match set silently.
  - Per-reference definition calls run strictly sequentially, never concurrently: `lsp.Client` is single-flight — `Call` increments an unsynchronized `nextID`, `writeMessage` holds no write lock, and the response loop reads one shared channel with no pending-request registry — so two concurrent calls would consume and drop each other's responses. State that in the loop's comment so a future reader does not "optimize" it.
  - The whole per-reference loop is bracketed by one `context.WithTimeout(ctx, timeout)` — one deadline for the phase, not one per call. When that context is done, stop issuing calls and record every remaining reference as an outcome with `Attempted` false, so `filterVerifiedReferences` keeps them.
  - A verification-phase deadline sets the timed-out flag **and** still returns a successful result. Comment this at the call site as the one place in this pipeline where the flag is set on a non-error return: the flag answers "might this server be stalled?" and governs disposal, while the return value answers "is this answer usable?" and is governed by the fail-closed rule. Note there that the flag is deliberately inert for `ConnKindSupervised`, Go's primary strategy, whose teardown returns without killing or closing.
- **Commit:** `feat(query): add Callers, the one-connection verified caller lookup`

### Card 16: hermetic tests for the Callers pipeline

- **Context:**
  - `internal/quarryengine/query/callers.go`
  - `internal/quarryengine/query/verify.go`
  - `internal/quarryengine/query/refs.go`
  - `internal/quarryengine/query/symbol_test.go`
  - `internal/quarryengine/query/refs_test.go`
  - `internal/quarryengine/daemon/ensureserver.go`
  - `internal/quarryengine/lsp/lspclient.go`
- **Edits:** none
- **Creates:**
  - `internal/quarryengine/query/callers_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  - Drive `callersFromClient` against a hand-built client over the in-memory pipe transport, following the fake-server construction `internal/quarryengine/query/symbol_test.go` already uses. Do not drive the exported `Callers`, which would require a spawn.
  - The single most important test, written first: given a non-empty reference set and a declaration-side definition call that returns an empty location list, every reference is kept and no error is returned. Then the same with the declaration-side definition call returning an LSP error. An empty declaration set is a silent success, not an error, and nothing intersects an empty set — verifying against one would drop every reference and turn the gate green, which is the exact fail-open this whole design exists to prevent.
  - Both directions of the interface relation, on synthetic fake-server responses: querying an interface method, a reference whose own definition resolves to a concrete method is dropped; querying a concrete method, a reference whose own definition resolves to the interface's declaration is kept because the implementation half of the match set matches it. The second is the fail-open the implementation-widening exists to close and is worth writing before the first.
  - A reference matching neither half of the match set is dropped in both directions — the property that removes the unrelated structurally-identical interfaces issue #1 measures.
  - A server that does not advertise `implementationProvider` skips verification and keeps every reference. So does one whose implementation call returns an LSP error.
  - `SkipVerification` set keeps every reference and issues no per-reference definition calls at all; assert the call count the fake server observed, not just the result.
  - The declaration return value is the definition-only set, not the union: with an implementation result that is not in the definition result, assert the returned declaration does not contain it. Assert too that the declaration site survives verification and is present in the returned reference set, since `filterUnexpectedCallers` depends on being able to remove it.
  - A verification phase that hits its deadline returns a successful result with the remaining references kept and sets the timed-out flag. Assert both halves — asserting only the return value would let a teardown regression through. Drive the deadline with a fake server that stalls on the per-reference definition call and a short timeout.
  - Assert the flag's consequence per `ConnKind` by calling `teardownConnection` directly with `timedOut` true: `ConnKindNative` and `ConnKindLegacy` tear the client down, and `ConnKindSupervised` leaves it neither killed nor closed, observable via the client's exported `Closed` accessor.
- **Commit:** `test(query): cover Callers fail-closed verification, both interface directions, and deadline teardown`

### Card 17: facade re-exports and the identifier-count comment

- **Context:**
  - `internal/quarryengine/query/callers.go`
  - `internal/quarryengine/registry/buildtags.go`
  - `internal/quarryengine/errors.go`
- **Edits:**
  - `quarry/facade.go`
  - `quarry/facade_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  - Add to `quarry/facade.go`, each following the file's existing one-line delegation or alias convention exactly: `Callers` delegating to `query.Callers`; `NormalizeBuildTags` delegating to `registry.NormalizeBuildTags` with the same variadic signature; the type alias `ErrBuildTagsUnsupported = quarryengine.ErrBuildTagsUnsupported`; and `var ErrBuildTagsUnsupportedSentinel = quarryengine.ErrBuildTagsUnsupportedSentinel`.
  - Rewrite the file's package doc comment sentence that currently claims it re-exports exactly 29 identifiers so it names 33 instead, and name the four additions in that sentence. The sentence's whole value is that it is checkable, so leaving it stale would be worse than not having it.
  - Extend `quarry/facade_test.go`: add `ErrBuildTagsUnsupported` to the alias round-trip block and its `init` body following the existing pairs; add `ErrBuildTagsUnsupportedSentinel` to the sentinel-identity table; and add two blank-identifier signature assertions, one for `Callers` against `func(context.Context, Options) ([]Reference, []Reference, error)` and one for `NormalizeBuildTags` against `func(...string) []string`.
  - Do not add any behaviour to the facade. Every declaration must remain an alias, an identical sentinel value, or a one-line delegation.
- **Commit:** `feat(quarry): re-export Callers, NormalizeBuildTags, and the build-tag error`

## Batch Tests

`verify:` leads with `go vet -tags lsp ./...`, which type-checks the `//go:build lsp` files in `query` and `daemon` against the changed `Options` struct and pipeline — files the hermetic tier never builds. It then runs `internal/quarryengine/query/` — card 13's refactor is proved by the package's existing `References`/`Definition`/`Symbol` tests continuing to pass unchanged, and cards 14, 15 and 16 add their own — then `quarry/`, whose facade test is a compile-time check that the four new re-exports really are aliases, identical sentinels, and signature-matched delegations. No live server is involved anywhere in this batch: every test drives a hand-built client over the in-memory pipe transport.
