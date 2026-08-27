# Batch: doc-redistribution

```yaml
task: Thin quarry/ facade over internal/quarryengine
batch: doc-redistribution
number: 4
cards: 2
verify: go test ./... && go test -tags lsp -run "^$" ./...
depends-on: [2]
```

## Batch Scope

Batch 2 card 9 `git mv`-ed the 289-line `quarry/doc.go` to `internal/quarryengine/doc.go` and changed only its package clause, leaving prose that still describes `package quarry`, a flat file layout, and unexported identifiers that are now exported. This batch fixes that: card 13 rewrites what remains at the engine root into an overview of the five-package layout, and card 14 relocates the four sections that belong to individual subpackages into package doc comments there. It is a separate batch because it is prose work with no compile-time coupling to the move, and separating it keeps batch 2's diff readable as a rename.

It has no interface for a later batch to consume, and it does not overlap with batch 3 — batch 3 creates two test files, this batch edits doc comments in four production files and the engine overview.

## Cards

### Card 13: Rewrite the engine overview for the five-package layout

- **Context:**
  - `internal/quarryengine/errors.go`
  - `internal/quarryengine/position.go`
  - `internal/quarryengine/log.go`
  - `internal/quarryengine/seam_enforcement_test.go`
  - `quarry/facade.go`
  - `_mill/discussion.md`
- **Edits:**
  - `internal/quarryengine/doc.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Rewrite `internal/quarryengine/doc.go` so it documents `package quarryengine` and the tree beneath it. Keep, updated in place: the opening description of what quarry does across five languages; "The engine/CLI split", reworded so the negative rule is stated as a property of the whole `internal/quarryengine` tree plus the `quarry/` facade rather than of one package, and so its reference to `seam_enforcement_test.go` points at `internal/quarryengine/seam_enforcement_test.go` and describes the widened subtree walk; "The typed error vocabulary", which describes `errors.go` and now lives in the right package already; and "Scope boundaries — what this package deliberately does not do", reworded to "what this engine deliberately does not do". Add a new section naming each package and its role — `quarryengine` (typed errors, the caller-facing `Position`, the shared `Logger`), `lsp`, `registry`, `daemon`, `query`, and `daemon/daemontest` — and stating the allowed import directions, so the layering the guard enforces is also written down where a reader will find it. Remove the four sections card 14 relocates: "The generalized LSP client", "The language-server registry", "The EnsureServer seam", "Go toolchain manager", and "Daemon state and concurrency" — leaving a one-line pointer to the owning package for each rather than deleting the reader's trail. Throughout, retarget every reference to a now-renamed identifier: `ensureServer` -> `daemon.EnsureServer`, `lspClient` -> `lsp.Client`, `close()`/`kill()` -> `Close()`/`Kill()`, `connKind`/`connKindNative`/`connKindSupervised`/`connKindLegacy` -> `daemon.ConnKind*`, `resolvePosition` -> `query`'s `resolvePosition`, and `registry.Entry.PinnedVersion` to its new package-qualified form. Do not describe the facade's contents here; `quarry/facade.go` carries its own doc comment.
- **Commit:** `docs(quarry): rewrite the engine overview for the five-package layout`

### Card 14: Relocate the four package-specific doc sections

- **Context:**
  - `internal/quarryengine/doc.go`
  - `internal/quarryengine/lsp/wire.go`
  - `internal/quarryengine/registry/load.go`
  - `internal/quarryengine/registry/detect.go`
  - `internal/quarryengine/daemon/toolchain.go`
  - `internal/quarryengine/daemon/daemonstate.go`
  - `internal/quarryengine/query/symbol.go`
  - `internal/quarryengine/query/definition.go`
- **Edits:**
  - `internal/quarryengine/lsp/lspclient.go`
  - `internal/quarryengine/registry/registry.go`
  - `internal/quarryengine/daemon/ensureserver.go`
  - `internal/quarryengine/query/refs.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Give each of the four subpackages a package doc comment, hosted at the head of the named file, carrying the prose relocated from the old `quarry/doc.go`. `lsp` (in `lspclient.go`): "The generalized LSP client" minus its resolver paragraphs — the eight-method wire description, the JSON-RPC framing, and the per-phase timeout/teardown rules — plus the position-conversion paragraph that explains why the byte column is re-read against the file to reach LSP's UTF-16 offset. `registry` (in `registry.go`): "The language-server registry" whole, including the built-ins, the `servers.yaml` overlay semantics, and the detection precedence order. `daemon` (in `ensureserver.go`): "The EnsureServer seam", "Go toolchain manager", and "Daemon state and concurrency" — three sections, so if the resulting comment dwarfs `ensureserver.go` itself, host it in a new `internal/quarryengine/daemon/doc.go` instead and say so in the commit message. `query` (in `refs.go`): the `workspace/symbol` resolver paragraphs, the `--in-file`/`documentSymbol` resolver paragraphs, and the zero/one/many candidate mapping to `ErrSymbolNotFound`/success/`ErrAmbiguousSymbol` — these sit inside the old doc's "The generalized LSP client" section at lines 65–87 but describe `resolvePosition` in `refs.go`, so this one section splits mid-body rather than moving whole. In every relocated paragraph, retarget renamed identifiers the same way card 13 does, and retarget file references to their new paths. Preserve the existing per-file doc comments already at the head of each edited file — the package comment goes above them, or merges with them where they overlap; do not delete file-level prose that describes that specific file.
- **Commit:** `docs(quarry): relocate package-specific doc sections into their subpackages`

## Batch Tests

`verify:` runs `go test ./... && go test -tags lsp -run "^$" ./...`. This is a documentation batch with no runnable surface of its own, so the command is not proving new behaviour — it is proving that the edits stayed inside comments. A doc comment cannot change program behaviour, but a mis-terminated comment block or a stray unescaped backtick sequence absorbing a declaration is a real and easy mistake in an edit of this size, and a full build catches it immediately. No new test is added, and no existing test changes.

The correctness bar for the prose itself is a reading check the implementer performs before committing: every identifier named in a relocated paragraph must exist under that name in the package that now hosts the paragraph, and every file path named must resolve. Both are greppable, and both are the specific way relocated documentation goes stale.
