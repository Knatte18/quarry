# Task 06 — config-reconciliation cold-start orientation

Type: exploration
Verb under test: `toc`
Status: runnable now

## Setup

Pinned to `PINNED_SHA` from the top-level README (`975578cda8d6f3a81580bd4e73725e060211b766`),
not the live main checkout:

```
git -C "$LADDER_LOOMYARD_REPO" worktree add /tmp/loomyard-eval-06 975578cda8d6f3a81580bd4e73725e060211b766
```

`<TARGET_DIR>` for this task is `/tmp/loomyard-eval-06`. Remove the worktree
when done (`git -C "$LADDER_LOOMYARD_REPO" worktree remove /tmp/loomyard-eval-06`).

## Scope

This task's scope is the whole repository. The prompt deliberately names no
package: the agent does not know where to look, so the value under test is
whether the first cheap survey (a directory-level table of contents) is worth
its prompt cost when the target is completely unknown going in.

## `<TASK TEXT>` (identical for A, B, C)

> This repository is unfamiliar to you. Somewhere in it, each module keeps an
> on-disk YAML configuration file that must stay in sync with that module's
> built-in template as the template's own set of keys changes over time --
> keys can be added to or removed from a template between releases, and an
> existing on-disk file must be reconciled against that change rather than
> silently drifting from it.
>
> Find this mechanism and explain how it works. Your explanation must cover:
>
> 1. Which package(s) implement the actual reconciliation logic -- computing
>    which keys were added to or removed from the template relative to an
>    existing on-disk file -- and which package(s) own the registry of module
>    names and their default templates.
> 2. What entry points (CLI commands, or exported functions a CLI command
>    calls) trigger this reconciliation, and which package(s) those entry
>    points live in.
> 3. Any module whose config is handled as a special case rather than by the
>    ordinary per-module logic, and why.
> 4. Which files and functions form this path end to end, from an entry point
>    down to the lowest-level key-comparison logic.

## Output schema (exploration tasks)

This schema was recovered from the V1 benchmark protocol document after that document was deleted.

```json
{
  "relevant_files": ["path/to/file.go", "..."],
  "key_symbols": [
    {"name": "FuncOrTypeName", "file": "path/to/file.go", "role": "one sentence"}
  ],
  "summary": "3-6 sentences explaining how the mechanism works end to end",
  "confidence": "high|medium|low",
  "open_questions": ["anything left uncertain, if any"]
}
```

## Notes for whoever prepares C's fasit / scores this

The chosen subject is config-file reconciliation against templates: how a
module's on-disk YAML config is kept in sync with its built-in template,
spanning `internal/configsync` (the reconciliation orchestration and its
`ReconcileAll`/`ReconcileFabricAt` entry points), `internal/configreg` (the
module registry `configsync` iterates over) and `internal/yamlengine` (the
actual template-vs-existing key diff in `Reconcile`/`MissingKeys`), with
`internal/configengine` supplying the per-module config file path. This
satisfies (a) because the real answer requires naming at least these four
packages, not one. It satisfies (b) because none of `configsync`,
`configreg`, `configengine` or `yamlengine` appears anywhere in the rendered
prompt above or in the schema block -- the prompt describes the behaviour in
plain language instead. It satisfies (c) because the whole mechanism --
`configsync.ReconcileAll`, `configsync.ReconcileFabricAt`,
`configreg.Modules`, `configengine.ConfigFile` and `yamlengine.Reconcile` /
`yamlengine.MissingKeys` -- is present and unchanged at the pinned SHA, and
the CLI entry point in `internal/configcli/configcli.go` and the clone-time
entry points in `internal/fabriccli/clone.go` and `internal/fabriccli/fabric.go`
that call into it are likewise present at the pin. No subject swap was
needed: card 9's exhaustive read confirmed all three constraints as pinned
above.
