# Batch: lsp-implementation-and-spike

```yaml
task: "Improve gopls query precision (build tags + scoping)"
batch: "lsp-implementation-and-spike"
number: 1
cards: 4
verify: go vet -tags lsp ./... && go test ./internal/quarryengine/lsp/
depends-on: []
```

## Batch Scope

This batch delivers the one LSP capability the verification filter needs (`textDocument/implementation` on `lsp.Client`) and then runs the measurement spike the discussion makes mandatory before any verification code is written. The spike's output — a recorded decision between symmetric and directional implementation-widening, plus the observed reference counts on the fixture — is the external interface batch 4 and batch 6 both consume: batch 4 implements the recorded widening mode, and batch 6's live test asserts against the recorded counts rather than against issue #1's pre-fix "31 → 2" figure.

It is one batch because the spike cannot run without the client method, and the client method has no other consumer until batch 4. Batch-local decision: the fixture module lives at the repo root under `testdata/` (the Go tool ignores any directory named `testdata`, so a nested `go.mod` there is invisible to `go build ./...` and `go test ./...`) rather than under one package's own directory, because batch 6's `internal/cli` live test drives the same fixture.

## Cards

### Card 1: textDocument/implementation on lsp.Client

- **Context:**
  - `internal/quarryengine/lsp/wire.go`
  - `internal/quarryengine/lsp/lspclient_guard_test.go`
- **Edits:**
  - `internal/quarryengine/lsp/lspclient.go`
  - `internal/quarryengine/lsp/lspclient_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  - Add a field `ImplementationProvider capabilityFlag` with json tag `implementationProvider` to the `capabilities` struct in `internal/quarryengine/lsp/lspclient.go`, alongside the existing `WorkspaceSymbolProvider` and `DocumentSymbolProvider` fields. The existing `capabilityFlag.UnmarshalJSON` already normalizes the bool-or-object shape; no change is needed there.
  - Add `func (c *Client) SupportsImplementation() bool` returning `c.caps.ImplementationProvider.Supported`, written and documented exactly like the existing `SupportsWorkspaceSymbol` / `SupportsDocumentSymbol` accessors immediately above it.
  - Add `func (c *Client) Implementation(ctx context.Context, fileURI string, pos Position) ([]Location, error)` issuing one `textDocument/implementation` request via `c.Call` with phase name `"implementation"` and the same `{"textDocument": {"uri": fileURI}, "position": pos}` params shape `Definition` uses. Decode the response with the existing `parseDefinitionResult` helper: `textDocument/implementation`'s result type is `Location | Location[] | LocationLink[] | null`, identical to `textDocument/definition`'s, so it needs no second parser. Say so in the method's doc comment, naming `parseDefinitionResult` as the shared decoder.
  - Update that file's package doc comment, which currently enumerates the exact request surface this client speaks and states "No callHierarchy, no implementation". Add `textDocument/implementation` to the enumerated surface and remove the "no implementation" clause, keeping the "no callHierarchy" half and the deferral paragraph that follows it intact. Leaving a package comment that denies the method the same file now implements would be a false statement in the file this card edits.
  - Do not add any import to `internal/quarryengine/lsp/lspclient.go`. Its import set is guarded to the standard library plus `github.com/Knatte18/quarry/internal/quarryengine`, and this card needs nothing beyond what is already imported.
  - Extend the fake-server-driven tests in `internal/quarryengine/lsp/lspclient_test.go` with: an initialize response advertising `implementationProvider: true` making `SupportsImplementation()` report true, an initialize response omitting the key making it report false, and an `Implementation` call whose fake response is a `Location[]` returning those locations. Follow the file's existing pipe-transport fake-server construction rather than introducing a new harness.
- **Commit:** `feat(lsp): add textDocument/implementation and implementationProvider capability`

### Card 2: issue #1 clock fixture module

- **Context:**
  - `go.mod`
- **Edits:** none
- **Creates:**
  - `testdata/clockfixture/go.mod`
  - `testdata/clockfixture/builder/poll.go`
  - `testdata/clockfixture/runner/tick.go`
  - `testdata/clockfixture/sched/wait.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  - Create a self-contained module `testdata/clockfixture/go.mod` declaring `module clockfixture` and a `go` directive matching the one in the repo's own `go.mod`. The module must have no dependencies beyond the standard library, so it needs no `go.sum`.
  - `testdata/clockfixture/builder/poll.go` is package `builder` and declares the symbol under test: an unexported `type clock interface` with methods `Now() time.Time` and `Sleep(d time.Duration)`, a concrete `type realClock struct{}` implementing both methods, an exported `func Poll(c clock) time.Time` that calls `c.Now()` and `c.Sleep(...)`, and an exported `func Run() time.Time` that constructs a `realClock` and both passes it to `Poll` (an interface-dispatch call site) and calls `realClock.Now()` on it directly (a concrete call site). Both call shapes must be present in this package — they are the two directions the verification filter has to distinguish.
  - `testdata/clockfixture/runner/tick.go` is package `runner` and declares its own structurally-identical, unrelated `type clock interface` with the same two methods, its own concrete implementer, and its own callers of `Now()` and `Sleep(...)`. It must not import `builder`.
  - `testdata/clockfixture/sched/wait.go` is package `sched` and does the same again, independently, so the fixture has three unrelated structurally-identical `clock` interfaces. It must not import `builder` or `runner`.
  - Keep every file small and free of build tags. The fixture exists to be queried by gopls, not to be run.
- **Commit:** `test(fixture): add three-package structurally-identical clock fixture for issue #1`

### Card 3: implementation-widening spike test

- **Context:**
  - `internal/quarryengine/query/refs_integration_test.go`
  - `internal/quarryengine/lsp/lspclient.go`
  - `testdata/clockfixture/builder/poll.go`
  - `testdata/clockfixture/runner/tick.go`
  - `testdata/clockfixture/sched/wait.go`
- **Edits:** none
- **Creates:**
  - `internal/quarryengine/query/implementation_spike_lsp_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  - Create a `//go:build lsp`-tagged test file in package `query` holding one test, `TestImplementationWidening_Spike`, that skips via `exec.LookPath("gopls")` when the binary is absent, exactly as `internal/quarryengine/query/refs_integration_test.go` does.
  - The test resolves the fixture root from the repo root the same way `refs_integration_test.go`'s own `repoRoot` helper does, spawns a gopls client with `lsp.NewClient`, initializes it rooted at the fixture module directory, and then, at each of two positions, issues `textDocument/definition`, `textDocument/implementation`, and `textDocument/references` and logs every returned location with `t.Logf`. The two positions are (a) the `Now` method inside the `clock` interface declaration in package `builder`, and (b) the `Now` method on the concrete `realClock` type in package `builder`.
  - Log the counts and the full location lists in a form that can be pasted into a findings document: one line per location, `file:line:character`, grouped under a heading naming the position and the LSP method.
  - The test asserts nothing about counts. It is a measurement harness, and hard-coding an expectation here would defeat the point of running it. It may assert only that each call returned without error, so a broken harness fails loudly instead of logging nothing.
  - Locate the two query positions by scanning the fixture source for the declaration text rather than hard-coding line numbers, so an edit to the fixture does not silently move the spike off-target.
- **Commit:** `test(query): add implementation-widening measurement spike against the clock fixture`

### Card 4: run the spike and record the widening decision

- **Context:**
  - `internal/quarryengine/query/implementation_spike_lsp_test.go`
  - `testdata/clockfixture/builder/poll.go`
  - `testdata/clockfixture/runner/tick.go`
  - `testdata/clockfixture/sched/wait.go`
- **Edits:** none
- **Creates:**
  - `docs/implementation-widening-spike.md`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  - Run the spike with the pinned gopls on `PATH`: `PATH="$HOME/.cache/quarry/tools/go/v0.23.0:$PATH" go test -tags lsp -run TestImplementationWidening_Spike -v ./internal/quarryengine/query/`. If that binary is missing, install nothing and treat the spike as inconclusive per the decision rule below.
  - Write `docs/implementation-widening-spike.md` recording: the gopls version measured, the fixture the measurement ran against, the raw location lists for both query positions and all three LSP methods, and then the decision.
  - Apply this decision rule verbatim and record which branch fired. If `textDocument/implementation` at the interface-method position returned only types related to the queried `clock` interface in package `builder`, record `mode: symmetric`. If it also returned the unrelated structural satisfiers declared in packages `runner` and `sched`, record `mode: directional`. If the spike could not be run at all, or its result is ambiguous, record `mode: symmetric` and state that it was chosen as the fail-closed default because the measurement was inconclusive.
  - The file must carry a machine-legible `mode:` line — the literal token `mode: symmetric` or `mode: directional` — because card 14 in batch 4 selects its implementation from it and card 25 in batch 6 reads the recorded counts.
  - Also record three counts measured from the spike's own `textDocument/references` output at the interface-method position, each under its own literal label so a later card can cite one figure unambiguously. Label them `references-unfiltered:` (the total number of references returned), `references-verified:` (how many remain after applying the recorded widening mode's filter by hand against the logged definition and implementation results), and `callers-verified:` (`references-verified` minus the declaration sites the definition result names, which is what `assert-no-callers` actually reports, since its `callers` list excludes every returned declaration).
  - `callers-verified:` is the figure batch 6's live test asserts against, and the three labels must be distinct because the three numbers are different: `textDocument/references` is issued with `includeDeclaration: true`, so the declaration site is in the raw and verified sets but never in the reported `callers` list. Do not carry issue #1's "31 → 2" figure forward — that is a pre-fix measurement taken on a different repo.
- **Commit:** `docs: record the implementation-widening spike measurement and decision`

## Batch Tests

`verify:` first runs `go vet -tags lsp ./...` so the `//go:build lsp` spike file card 3 creates is type-checked even though the hermetic tier never builds it, then runs the hermetic `lsp` package tests, which is where card 1's new capability accessor and `Implementation` method are exercised against the existing in-memory fake server. The spike itself (card 4) is run by hand with the pinned gopls prepended to `PATH`, because `verify:` must stay fast and must not depend on a language server being present. Cards 2 and 4 create no runnable Go surface in the main module: the fixture lives under `testdata/`, which the Go tool ignores, and the findings document is prose.
