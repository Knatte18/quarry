# Batch: initialization-options-plumbing

```yaml
task: "Improve gopls query precision (build tags + scoping)"
batch: "initialization-options-plumbing"
number: 3
cards: 4
verify: go vet -tags lsp ./... && go test ./internal/quarryengine/lsp/ ./internal/quarryengine/daemon/ ./internal/quarryengine/query/
depends-on: [1, 2]
```

## Batch Scope

This batch carries the rendered `initializationOptions` map from `query` down to the `initialize` request through both `EnsureServer` strategies and the legacy cold-spawn path, adds `Options.BuildTags` as the engine's carrier for the normalized tag set, and raises `ErrBuildTagsUnsupported` inside the engine's own detect step before any server is spawned. It also makes the native strategy spawn a private gopls whenever the tag set is non-empty, since the shared `-remote=auto` daemon's identity is not a function of the state directory and would otherwise mix tag sets.

It is one batch because the `Initialize` signature change, the two `EnsureServer` strategies, and the legacy path are one compile unit: none of the three can land alone. The external interface batches 4 and 5 consume is `query.Options.BuildTags` plus the guarantee that a non-empty tag set either reaches the server or hard-errors before spawn.

## Cards

### Card 9: initializationOptions on the initialize request

- **Context:**
  - `internal/quarryengine/lsp/lspclient_guard_test.go`
- **Edits:**
  - `internal/quarryengine/lsp/lspclient.go`
  - `internal/quarryengine/lsp/lspclient_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  - Change `lsp.Client.Initialize` in `internal/quarryengine/lsp/lspclient.go` to `func (c *Client) Initialize(ctx context.Context, rootURI string, initOptions map[string]any) error`.
  - Build the params map exactly as today (`processId`, `rootUri`, `capabilities`), then add the key `initializationOptions` set to `initOptions` if and only if `initOptions != nil`. A nil argument must produce a request byte-identical to today's — no key, not a key with a null or empty-map value.
  - Extend the method's doc comment to state that contract in one sentence, and to state that this client has no opinion on what the map contains: it receives an already-rendered map and never constructs, validates, or interprets one. The client must not learn about build tags, `registry`, or the `{{tags}}` placeholder.
  - Add no import. The guard test pins this file to the standard library plus `github.com/Knatte18/quarry/internal/quarryengine`.
  - Update every `client.Initialize(ctx, "file:///tmp/example")` call in `internal/quarryengine/lsp/lspclient_test.go` to pass a third argument, and add two fake-server assertions: initializing with a nil map produces a request whose decoded params carry no `initializationOptions` key at all, and initializing with a non-nil map produces a request carrying that exact map. The fake server in that file already sees the raw request, so both assertions are observable there.
- **Commit:** `feat(lsp): send initializationOptions on initialize when the caller supplies them`

### Card 10: thread rendered options through both daemon strategies

- **Context:**
  - `internal/quarryengine/lsp/lspclient.go`
  - `internal/quarryengine/daemon/probe.go`
- **Edits:**
  - `internal/quarryengine/daemon/ensureserver.go`
  - `internal/quarryengine/daemon/doc.go`
  - `internal/quarryengine/daemon/ensureserver_test.go`
  - `internal/quarryengine/daemon/ensureserver_integration_test.go`
  - `internal/quarryengine/daemon/supervised_test.go`
  - `internal/quarryengine/daemon/supervised_lsp_test.go`
  - `internal/quarryengine/daemon/supervised_integration_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  - Add a trailing `initOptions map[string]any` parameter to `finalizeConnection`, `ensureNative`, `ensureSupervised`, and the exported `EnsureServer` in `internal/quarryengine/daemon/ensureserver.go`, and pass it through to every `client.Initialize` call site inside that file. There are four `finalizeConnection` call sites in `ensureSupervised` and `ensureNative` combined; all must thread the same value.
  - Change `nativeArgv` to `func nativeArgv(binPath string, extraArgs []string, private bool) []string`. When `private` is false it returns today's argv unchanged, including `-remote=auto` and the `-remote.listen.timeout` companion. When `private` is true it returns the binary plus `extraArgs` and neither of those two flags, so gopls runs as an unshared process rather than joining its own auto-daemon.
  - `ensureNative` derives that argument as `initOptions != nil`. Per the overview's `rendered-options-non-nil-means-tagged` Shared Decision, a non-nil rendered map means and only means "the caller passed a non-empty build-tag set", so no separate tag parameter travels alongside it.
  - Document on `nativeArgv` why the private spawn exists: the `tags-<hex>` state-directory segment partitions only the supervised strategy, whose socket path derives from the state directory, whereas gopls itself picks the shared `-remote=auto` daemon's address, so on the native path two tag sets would otherwise land in one daemon. Note that native is the fallback on every platform and the only path on windows, so this is not a windows-only concern.
  - Update `internal/quarryengine/daemon/doc.go` where it spells out `EnsureServer`'s signature so the doc comment matches the new parameter list.
  - Update `internal/quarryengine/daemon/ensureserver_test.go`: pass the new argument at every `finalizeConnection`, `nativeArgv`, and `EnsureServer` call site, and add two `nativeArgv` assertions — `private == false` produces today's argv byte for byte (the back-compat assertion, written first), and `private == true` produces an argv containing neither `-remote=auto` nor any `-remote.listen.timeout` flag while preserving `binPath` first and `extraArgs` in order.
  - Update every call site of all four changed functions across this package's test files to pass a nil map, leaving those tests' behaviour unchanged. That means `EnsureServer` and `ensureNative` in `internal/quarryengine/daemon/ensureserver_integration_test.go`, `ensureSupervised` in `internal/quarryengine/daemon/supervised_test.go`, `internal/quarryengine/daemon/supervised_lsp_test.go` and `internal/quarryengine/daemon/supervised_integration_test.go`, and `finalizeConnection`, `nativeArgv` and `EnsureServer` in `internal/quarryengine/daemon/ensureserver_test.go`. Search for each of the four names rather than working from this list alone.
  - Three of those files are `//go:build lsp`-tagged and one is not, so a missed site surfaces in different places: `internal/quarryengine/daemon/supervised_test.go` breaks the hermetic `go test` this batch runs, while the tagged files break only the leading `go vet -tags lsp ./...`. Both are in this batch's `verify:`, but update them deliberately rather than discovering them from a failure.
- **Commit:** `feat(daemon): thread initializationOptions through both strategies and spawn privately when tagged`

### Card 11: Options.BuildTags and the pre-spawn hard error

- **Context:**
  - `internal/quarryengine/registry/buildtags.go`
  - `internal/quarryengine/registry/initoptions.go`
  - `internal/quarryengine/registry/detect.go`
  - `internal/quarryengine/errors.go`
  - `internal/quarryengine/daemon/ensureserver.go`
- **Edits:**
  - `internal/quarryengine/query/refs.go`
  - `internal/quarryengine/query/symbol.go`
  - `internal/quarryengine/query/refs_test.go`
  - `internal/quarryengine/query/symbol_test.go`
  - `internal/quarryengine/query/implementation_spike_lsp_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  - Add a `BuildTags []string` field to `Options` in `internal/quarryengine/query/refs.go`, documented as the normalized build-tag set, with the explicit note that the engine re-normalizes it defensively on entry because `Options` is public through the facade and an unnormalized value would otherwise fail silently — unlike `StateDir`, whose misuse fails loudly and which stays entirely the caller's obligation.
  - Document, on both the new `BuildTags` field and the existing `StateDir` field, the obligation a non-CLI caller inherits: the supervised daemon is partitioned by `StateDir` alone, so a caller that varies `BuildTags` while holding `StateDir` fixed will have two tag sets served by one daemon and will get the other tag set's answers. `internal/cli` discharges this by appending a `tags-<hex>` segment to the resolved leaf; a facade or SDK caller must supply a distinct `StateDir` per distinct tag set itself. State that this is the same "populating it is entirely the caller's obligation" contract `StateDir` already carries, now with a second reason, and that the engine deliberately does not re-key the directory on the caller's behalf.
  - Add an unexported helper in `internal/quarryengine/query/refs.go`, `func detectAndRender(opts Options) (string, registry.Entry, map[string]any, error)`, that calls `registry.DetectLanguage(opts.TargetDir, opts.Registry, opts.Lang)`, then `registry.NormalizeBuildTags(opts.BuildTags...)`, then `registry.RenderInitializationOptions(lang, entry, tags)`, returning the language, the entry, the rendered map, and any error. Both a detection failure and `ErrBuildTagsUnsupported` come back through its error return unchanged.
  - Replace the `registry.DetectLanguage` call at the top of `lookup` and the identical one at the top of `Symbol` in `internal/quarryengine/query/symbol.go` with `detectAndRender`, so the hard error is raised after detection (it must name the language) and before `acquireConnection` is ever called. A query that cannot honour its own build-tag scoping must not pay for a server launch.
  - Add a trailing `initOptions map[string]any` parameter to `acquireConnection` and pass it to `daemon.EnsureServer` and to the legacy path's own `client.Initialize` call inside that function.
  - Update the `client.Initialize` call sites in `internal/quarryengine/query/refs_test.go`, `internal/quarryengine/query/symbol_test.go`, and the `//go:build lsp`-tagged `internal/quarryengine/query/implementation_spike_lsp_test.go` that batch 1 created, to pass a nil map, leaving those tests' behaviour unchanged. The spike file is invisible to the hermetic tier, so a missed call site there surfaces only in this batch's leading `go vet -tags lsp ./...` — update it deliberately rather than discovering it from a vet failure.
- **Commit:** `feat(query): carry BuildTags on Options and hard-error before spawn for entries without a template`

### Card 12: build-tag behaviour tests in query

- **Context:**
  - `internal/quarryengine/query/refs.go`
  - `internal/quarryengine/query/symbol.go`
  - `internal/quarryengine/query/definition.go`
  - `internal/quarryengine/query/definition_test.go`
  - `internal/quarryengine/registry/registry.go`
  - `internal/quarryengine/registry/initoptions.go`
  - `internal/quarryengine/errors.go`
- **Edits:** none
- **Creates:**
  - `internal/quarryengine/query/buildtags_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  - Create an untagged test file in package `query` covering the hard error and the empty-set no-op, using a synthetic `registry.Registry` built in-test rather than the built-in registry, and a target directory containing only a `go.mod` marker file so detection succeeds without any server.
  - Assert that `References` with a non-empty `BuildTags` against an entry whose `InitializationOptions` is nil returns an error satisfying `errors.Is(err, quarryengine.ErrBuildTagsUnsupportedSentinel)` whose message names the language, and that the same holds for `Definition` and for `Symbol` — three entry points, one shared `detectAndRender` step.
  - Assert the error is raised before any connection attempt: point the synthetic entry's `Command` at a binary name that cannot exist on `$PATH` and assert the returned error is the build-tag error, not `ErrServerNotFound`. That ordering is the whole point of siting the check in the detect step, and it is observable exactly this way — the same technique `internal/quarryengine/query/definition_test.go` already uses for its legacy-path regression proof.
  - Assert the complements: `BuildTags` set to `nil`, to `[]string{""}`, and to `[]string{" , "}` against that same template-free entry are all not build-tag errors, because each normalizes to the empty set. Those calls will fail later with `ErrServerNotFound` from the fake binary name, which is the correct signal that they got past the build-tag check.
  - Assert that `BuildTags: []string{"b", "a"}` and `BuildTags: []string{"a", "b"}` against the same template-free entry produce the identical error message, proving the engine's defensive re-normalization runs.
- **Commit:** `test(query): cover the build-tag hard error and the empty-tag-set no-op`

## Batch Tests

`verify:` first runs `go vet -tags lsp ./...`, which is required rather than optional here, and then the hermetic tests of the three packages whose signatures change — `lsp` (card 9's `initialize` request shape), `daemon` (card 10's `nativeArgv` back-compat and private-spawn assertions plus the existing `finalizeConnection` coverage), and `query` (cards 11 and 12). The tagged vet pass leads because card 10 edits `internal/quarryengine/daemon/ensureserver_integration_test.go`, which is `//go:build lsp`-tagged and therefore invisible to the hermetic tier, so without the tagged vet pass a signature mistake in that file would not surface until batch 6.
