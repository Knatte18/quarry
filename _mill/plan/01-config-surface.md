# Batch: config-surface

```yaml
task: 'Kick-start pack bench: pre-resolved glyph spans in the prompt (M4)'
batch: config-surface
number: 1
cards: 4
verify: go test ./bench/loomyard-eval/ladder/internal/ladder/ -run 'TestLoadLadder|TestConfigIsControlAndGrantsTools'
depends-on: []
```

## Batch Scope

This batch changes only the ladder file's decoded shape and its struct-level validation: the three
new `Config` fields (`control`, `card`, `pack`), the new `Ladder` field (`pack_targets`), the split
of the single control predicate into `IsControl()` and `GrantsTools()`, and the validation rules
those fields require. Nothing outside `config.go` changes here — every call site that must switch
from one predicate to the other is batch 2's work, and this batch deliberately leaves them all
reading `IsControl()` so the package still compiles and every existing test still passes.

The external interface batch 2, 4 and 6 consume is exactly: `Config.Control`, `Config.Card`,
`Config.Pack`, `Ladder.PackTargets`, `Config.IsControl()` with its new meaning, and the new
`Config.GrantsTools()`.

Batch-local decision: `control` is a `*bool`, not a `bool`. An omitted key must be distinguishable
from an explicit `control: false`, because the default when the key is absent is today's
`len(Allowed) == 0` and an explicit `false` on a tool-less cell is what makes ladder e's two rungs
rungs.

## Cards

### Card 1: Split the control predicate and add the three new config fields

- **Context:**
  - `bench/loomyard-eval/ladder/ladder-toc.yaml`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/config.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Add three fields to the `Config` struct: `Control *bool` with yaml tag `control`, `Card string`
  with yaml tag `card`, and `Pack bool` with yaml tag `pack`. Add one field to the `Ladder` struct:
  `PackTargets []string` with yaml tag `pack_targets`. `Control` is a pointer precisely so an
  omitted key is distinguishable from an explicit `false`.
  Rewrite `IsControl` to return `*c.Control` when `c.Control` is non-nil and `len(c.Allowed) == 0`
  otherwise, and add a new method `GrantsTools` on the same value receiver returning
  `len(c.Allowed) > 0`.
  Rewrite `IsControl`'s doc comment so it states the new meaning — the cell is its ladder letter's
  comparison baseline — and states the default explicitly. Give `GrantsTools` its own doc comment
  saying it reports whether the cell has an MCP server attached, and that this is a different
  question from `IsControl`.
  Rewrite `ControlFor`'s doc comment, which currently says "whose allowed list is empty": it now
  returns the single config for the letter that `IsControl` reports true for.
  Add a `Card` doc comment stating it is a repository-relative markdown file rendered into the
  prompt, resolved the same way `Task.TaskFile` is, and that an empty value renders today's prompt
  unchanged. Add a `Pack` doc comment stating it marks the one cell whose card the generated
  kick-start pack is written into. Add a `PackTargets` doc comment stating it is the glyph list the
  `pack` subcommand resolves, and that it is a single top-level list resolved once, which is why at
  most one config per file may set `pack`.
  Do not touch any other file in this batch; every existing call site keeps reading `IsControl()`
  and keeps compiling.
- **Commit:** `feat(ladder): split IsControl from GrantsTools and add card/pack config fields`

### Card 2: Validate the new fields and reword the control-count message

- **Context:** none
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/config.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Extend `validate` with four struct-level rules, all filesystem-free — `validate` takes no
  repository root and must continue to open no file, since every existing `config_test.go` fixture
  depends on that:
  1. at most one config in the whole file may set `Pack` true — per file, not per ladder letter;
  2. a config with `Pack` true must declare a non-empty `Card`;
  3. `PackTargets` is non-empty if and only if at least one config sets `Pack` true — both
     directions are errors, each with its own message;
  4. every `PackTargets` entry is non-empty and unique.
  Each rule returns an error naming the offending config id or the offending entry, matching the
  style of the existing `configs.%s.allowed` messages.
  Change the existing control-count error message from
  `expected exactly one control (empty allowed list), found %d` to
  `expected exactly one control, found %d`. The parenthetical is now wrong: an empty allowed list is
  no longer what makes a cell the control.
  Extend `validate`'s own doc comment to name the four new rules and to restate that the sentinel
  check on a card's contents deliberately lives elsewhere, where the repository root is in hand.
- **Commit:** `feat(ladder): validate pack, pack_targets and card at load time`

### Card 3: Test control defaulting and GrantsTools independence

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/config.go`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/config_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Add a table-driven `TestConfigIsControlAndGrantsTools` covering the four defaulting cases —
  `Control` unset with an empty `Allowed` is a control; `Control` unset with a non-empty `Allowed`
  is not; an explicit `true` and an explicit `false` each override both — and asserting
  `GrantsTools` reads only `Allowed` in every one of those four cases, so the two predicates are
  independent.
  Add `TestLoadLadder_ControlFieldDefaulting`, driving the same matrix through `LoadLadder` against
  fixtures written by the existing `writeLadderFile` helper, so the yaml decoding of `control:` is
  covered and not only the method.
  Extend the existing rejected-file table in `TestLoadLadder_Rejected` with a case for three
  tool-less configs under one ladder letter where two set `control: true`, and one where none does,
  both expecting the reworded `expected exactly one control` message; and add an accepted case for
  three tool-less configs under one letter with exactly one `control: true`.
  Leave `TestLoadLadder_RealTocFile` asserting exactly what it asserts today — it is the regression
  fixture for the backwards-compatibility claim, since every config in the real file omits
  `control:`.
- **Commit:** `test(ladder): cover control defaulting and GrantsTools independence`

### Card 4: Test the pack, pack_targets and card validation rules

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/config.go`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/config_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Add `TestLoadLadder_PackValidation`, a table over rejected fixtures: two configs setting
  `pack: true` under one ladder letter; two setting it under different letters (asserting the rule
  is per file, not per letter); `pack: true` with no `card`; `pack_targets` set with no pack cell;
  a pack cell with an empty `pack_targets`; an empty string among the `pack_targets` entries; and a
  duplicated entry. Add the accepted counterpart: one pack cell with a `card` and a non-empty,
  unique `pack_targets`.
  Add a case asserting that a typo'd key near the new ones — for instance `pack_target` — is still
  rejected by the decoder's unknown-key rule, so the new fields do not open a hole in
  `KnownFields(true)`.
  Add `TestLoadLadder_ValidateOpensNoFile`, which loads a fixture whose `card` names a path that
  does not exist on disk and asserts the load succeeds. This is the standing guard for the
  struct-level-only rule; without it a later reviewer's "surely validate should check the card
  exists" would silently break every fixture in this file.
- **Commit:** `test(ladder): cover pack and card validation rules`

## Batch Tests

`verify:` runs `go test` against `./bench/loomyard-eval/ladder/internal/ladder/` with
`-run 'TestLoadLadder|TestConfigIsControlAndGrantsTools'`. That pattern covers every test this batch
adds — `TestConfigIsControlAndGrantsTools`, `TestLoadLadder_ControlFieldDefaulting`,
`TestLoadLadder_PackValidation`, `TestLoadLadder_ValidateOpensNoFile` — and every existing
`TestLoadLadder_*` test, including `TestLoadLadder_RealTocFile`, which is the backwards-compatibility
regression for this batch's change to `IsControl`. Nothing outside `config.go` changes, so no other
package test can be affected; the module-wide `go build ./...` at the batch boundary confirms the new
`GrantsTools` method and the four new fields compile against every existing caller.
