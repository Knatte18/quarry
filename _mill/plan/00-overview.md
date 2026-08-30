# Plan: Rethink quarry-mcp's per-call targetDir ergonomics

```yaml
task: "Rethink quarry-mcp's per-call targetDir ergonomics"
slug: "mcp-target-dir-ergonomics"
approved: false
started: "20260830-111123"
parent: "main"
root: ""
verify: go build ./...
```

## Batch Index

_The fenced yaml block below is the authoritative DAG mill-go reads to schedule batches.
Every batch lives at `NN-<batch-slug>.md` in this directory and is mirrored as one entry here._

```yaml
batches:
  - number: 1
    name: mcpserver-targetdir-removal
    file: 01-mcpserver-targetdir-removal.md
    depends-on: []
    verify: go test ./internal/mcpserver/...
  - number: 2
    name: docs-and-bench-alignment
    file: 02-docs-and-bench-alignment.md
    depends-on: [1]
    verify: uv run --no-project --with pytest --with pyyaml python -m pytest bench/loomyard-eval/ladder/tests -q
```

## Shared Decisions

_Cross-cutting decisions every batch inherits: naming conventions, error-handling posture, test frameworks, style/lint constraints.
One subsection per decision.
Batch-local decisions live in each batch file._

### Decision: schema-follows-the-go-type

- **Decision:** The published MCP input schemas are derived from the Go input structs by
  `inputSchemaFor[T]`, so deleting a struct field is the whole mechanism for removing a schema
  property. No schema literal is edited, and no call-wide analogue of `dropEntryProperty` is added.
- **Rationale:** `inputSchemaFor` infers from the Go type via `jsonschema.For` and then patches only
  the `targets` item schema. Adding a call-wide property-pruning mechanism would be real machinery
  built for a migration path with zero external consumers.
- **Applies to:** all batches

### Decision: hard-removal-is-the-error-behaviour

- **Decision:** A call that still sends `targetDir` fails as a whole-call error via the SDK's
  call-wide `additionalProperties: false`. No compatibility shim, no silent ignore, no custom
  migration message, and no new production code.
- **Rationale:** `inputSchemaFor` deliberately clears `additionalProperties` only on the `targets`
  item schema and leaves the call-wide strictness inference produced. The resulting failure is loud
  and self-correcting. Silently ignoring the property would serve results from the wrong root while
  the model believed it had retargeted the call — the worst available failure mode.
- **Applies to:** all batches

### Decision: toc-handlers-keep-bypassing-resolveCall

- **Decision:** `tocFileHandler` and `tocDirHandler` continue never to call `resolveCall`. With
  `effectiveTargetDir` deleted they read `cfg.TargetDir` directly instead.
- **Rationale:** `tocFileCommand`/`tocDirCommand` never load the registry or resolve a state dir
  either, so a malformed `servers.yaml` must not fail a toc call. The refactor must not "simplify"
  the two toc handlers onto `resolveCall`.
- **Applies to:** all batches

### Decision: state-dir-derivation-is-untouched

- **Decision:** `resolveCall` keeps calling `cli.ResolveBuildTags`, `cli.ResolveConfigPath`,
  `quarry.LoadRegistry`, and `cli.ResolveStateDir` in exactly that order, through the exported
  `internal/cli` helpers. Only the first step (target-directory resolution) changes, from
  `effectiveTargetDir(cfg, override)` to `cfg.TargetDir`.
- **Rationale:** A local reimplementation or a reordering would silently spawn a second gopls daemon
  and forfeit warm-daemon reuse.
- **Applies to:** all batches

### Decision: buildtags-and-lang-are-untouched

- **Decision:** `buildTags` and `lang` remain per-call properties on every tool that has them today.
  No launch-time `--build-tags` flag is added.
- **Rationale:** `targetDir` is workspace identity — one value per server by definition, exactly what
  LSP's `rootUri` encodes. `buildTags` is query scoping the CLI exposes per verb, and a single
  session may legitimately want the same question asked under two tag sets.
- **Applies to:** all batches

### Decision: no-serverversion-bump

- **Decision:** `serverVersion` stays at `"0.1.0"` even though this task edits `mcpserver.go` and
  changes every tool's published schema shape.
- **Rationale:** The constant tracks the package's own development. There is no release, changelog,
  or external consumer a bump would inform — the same absence of consumers that removes the
  deprecation obligation is what makes the bump pointless. Recorded explicitly so the omission does
  not read as an oversight to a reviewer.
- **Applies to:** all batches

### Decision: cgo-build-cost-is-expected

- **Decision:** Verification runs `go build ./...` module-wide at each batch boundary and
  `go test ./internal/mcpserver/...` for batch 1. A cold first compile is slow and is not a failure.
- **Rationale:** The build requires `CGO_ENABLED=1` and a C toolchain because the `toc` verbs link
  tree-sitter C grammars. Both commands were confirmed green against this worktree's tip before the
  plan was written, so any failure the implementer sees is its own.
- **Applies to:** all batches

### Decision: done-gate-stays-go-test-only

- **Decision:** `pipeline.done_gate` stays `go test ./...`; no linter is appended.
- **Rationale:** `golangci-lint` is not installed on this machine (`which golangci-lint` finds
  nothing), so appending it would make every task in this hub depend on a tool that is not present.
  The repo-wide Go test command is already cheap enough to serve as the gate on its own.
- **Applies to:** all batches

### Decision: python-exception-is-edited-not-extended

- **Decision:** Batch 2 edits two existing files under `bench/loomyard-eval/ladder/`. No new Python
  file is created, and no Python appears anywhere outside that already-sanctioned directory.
- **Rationale:** `CLAUDE.md` bans introducing Python and names `bench/loomyard-eval/ladder/` as an
  existing exception that must not be extended. Editing one prompt line and one assertion inside an
  existing file in that directory is not extending the exception.
- **Applies to:** docs-and-bench-alignment

### Decision: greps-are-necessary-but-not-sufficient

- **Decision:** Completeness is checked by three greps — a production-file token pass, a separate
  `_test.go` token pass with its own whitelist, and a zero-hit pass for the deleted helper — *plus* a
  mandatory re-read of all six input structs' doc comments and `exceptSet`'s. No grep alone is
  treated as proof the change is complete, and the production and test passes are kept separate
  because what counts as an intentional survivor differs between them.
- **Rationale:** Three input-struct doc comments state the override's existence purely by count
  ("the three call-wide resolution overrides"), containing neither `targetDir` nor `TargetDir`, so
  no token grep surfaces them. `exceptSet`'s doc comment paraphrases the deleted helper without
  naming it. A grep-only completeness check would pass over all four.
- **Applies to:** all batches

## All Files Touched

_Full union of every `Creates:` / `Edits:` / `Moves:` **target** path across every batch, sorted alphabetically (Move **source** paths are excluded — they disappear, like `Deletes:` tokens).
Cards are the source of truth;
this section is the input `_plan_validate.py`'s `all-files-touched-mismatch` check cross-references against the derived union of every card's `Edits:`/`Creates:`/Move-target paths, to catch drift between the hand/agent-maintained list here and that derived union._

- `bench/loomyard-eval/ladder/scripts/ladder_config.py`
- `bench/loomyard-eval/ladder/tests/test_ladder_config.py`
- `docs/mcp-setup.md`
- `internal/mcpserver/callcontext.go`
- `internal/mcpserver/callcontext_test.go`
- `internal/mcpserver/lspentry.go`
- `internal/mcpserver/mcpserver.go`
- `internal/mcpserver/nativeentry.go`
- `internal/mcpserver/tools_assert.go`
- `internal/mcpserver/tools_impact.go`
- `internal/mcpserver/tools_lsp.go`
- `internal/mcpserver/tools_symbol.go`
- `internal/mcpserver/tools_toc.go`
- `internal/mcpserver/tools_toc_test.go`
- `internal/mcpserver/translate.go`
- `internal/mcpserver/transport_test.go`
