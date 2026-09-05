# Batch: provenance

```yaml
task: 'Kick-start pack bench: pre-resolved glyph spans in the prompt (M4)'
batch: provenance
number: 3
cards: 3
verify: go test ./bench/loomyard-eval/ladder/internal/ladder/ -run 'TestReadProvenance|TestWriteProvenance|TestWriteReadProvenance|TestMergeProvenance'
depends-on: []
```

## Batch Scope

This batch changes only the committed provenance record: it makes `WriteProvenance` create the
results root it is asked to write into, and it adds the `kickstart_pack` block plus the merge policy
that carries it forward. It has no dependency on batch 1 or 2 — nothing here reads a `Config` — so it
can run in parallel with them, and it deliberately touches no file either of those batches touches.

The external interface batch 4 and 5 consume is: the `KickstartPack` type and the
`Provenance.KickstartPack` field, and the guarantee that `MergeProvenance` neither invents nor drops
one.

Batch-local decision: `MergeProvenance` carries an existing block forward and a `next` `Invocation`
never sets one. The block is written onto the merged record by the pack command after the merge
returns, not by the collector, because it describes an artefact rather than an invocation's own
environment — and because putting it on `Invocation` would give every ordinary `run` a field it must
remember to leave empty.

## Cards

### Card 13: Create the results root inside WriteProvenance

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/run.go`
  - `docs/roadmap.md`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/provenance.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Have `WriteProvenance` call `os.MkdirAll(resultsRoot, 0o755)` before writing, wrapping a failure as
  `write provenance: %w`, and extend its doc comment to say it creates the results root when absent.
  This is the roadmap's standing small item. The fix belongs here rather than at the call sites
  because this is the single write that fails first on a fresh root — `Run` writes provenance
  immediately after the invocation merge, before anything else in the run has created a directory —
  and because putting it here also covers the pack command batch 4 adds, which writes provenance into
  a root that may not exist yet.
  Do not strike the roadmap bullet in this card; that edit is batch 7's, so the roadmap is touched
  once, in one commit, alongside the line pointing at the new results root.
- **Commit:** `fix(ladder): create the results root inside WriteProvenance`

### Card 14: Add the kickstart_pack block and carry it forward on merge

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/run.go`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/provenance.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Add a `KickstartPack` struct with exactly these JSON keys, in this order: `generated_at`,
  `quarry_commit`, `quarry_dirty`, `loomyard_commit`, `targets` as a string slice, `pack_sha256`,
  `resolve_sha256`, `card_file`. Give every field a doc comment. `card_file` is
  repository-relative — it is the ladder file's own `card:` value for the pack cell — never an
  absolute path, since this record is committed.
  Add `KickstartPack *KickstartPack` to the `Provenance` struct with the JSON tag
  `kickstart_pack,omitempty`, so a results root that has no pack carries no key at all and every
  existing committed `provenance.json` still round-trips unchanged.
  In `MergeProvenance`, set `KickstartPack: existing.KickstartPack` on the merged record in the
  branch that takes an existing record, and leave it nil in the fresh-root branch. A `next`
  `Invocation` never carries one and no code path may derive one from an invocation.
  Extend `MergeProvenance`'s doc comment with one sentence naming this: `kickstart_pack` is carried
  forward from the existing record, is never derived from an invocation, and is never compared for
  equality the way `ladder_file` and the other three identity fields are — it is an artefact record,
  not an identity.
  Explain in a comment on the field itself why it must be a real struct field rather than something
  the pack command writes into the JSON directly: `Run` rewrites the whole file through
  `MergeProvenance` at startup, so an unknown key would be silently dropped on the first run after
  the pack.
- **Commit:** `feat(ladder): record the kick-start pack in provenance`

### Card 15: Test the mkdir fix and the pack block's merge behaviour

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/provenance.go`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/provenance_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Add `TestWriteProvenance_CreatesMissingResultsRoot`: write into a path under a fresh temporary
  directory that does not exist, assert no error, and read the record back. Written against the
  pre-fix code this test fails with `no such file or directory`, which is what makes it the
  regression for card 13 rather than a restatement of it.
  Extend `TestMergeProvenance` with three cases. First, an existing record carrying a
  `kickstart_pack` block merged with an invocation that has none: the block survives byte for byte.
  Second, a fresh-root merge: the merged record's block is nil, so nothing invents one. Third — the
  case that makes the pack command's design load-bearing — a record written by a pack invocation
  followed by an ordinary run invocation carrying the same ladder file, target repository hash,
  server name and effective repetition count merges cleanly and yields two entries in the invocation
  list.
  That third case is the regression for the decision that the pack writes a real, complete
  invocation rather than a pack-only stub. A stub leaving those four fields at their zero values
  fails `MergeProvenance`'s own identity checks on the very first run — `reps_effective 0` against
  the file's own value, then an empty ladder file path — before rep 1, every time, and would need an
  exemption path carved into machinery three existing results roots depend on.
  Leave `TestMergeProvenance_NoAbsolutePathAnywhereInOutput` covering the new field too: give the
  merged record a pack block whose `card_file` is a repository-relative path, so the standing
  no-machine-paths assertion extends over the new key rather than stopping at the old ones.
- **Commit:** `test(ladder): cover the provenance mkdir fix and pack-block merge`

## Batch Tests

`verify:` runs `go test` against `./bench/loomyard-eval/ladder/internal/ladder/` with
`-run 'TestReadProvenance|TestWriteProvenance|TestWriteReadProvenance|TestMergeProvenance'`, which
covers every test in `provenance_test.go` that touches the record's shape or its merge policy —
including `TestWriteReadProvenance_RoundTrips` and
`TestMergeProvenance_NoAbsolutePathAnywhereInOutput`, both of which read the whole struct and would
notice a mis-tagged new field.

The collector and scan tests are deliberately outside the pattern: this batch adds no field the
collector fills and touches no path the memory scan walks. Adding a field to a struct is only a
compile break for a caller that constructs it positionally, and any such caller in a test file is
caught by this batch's own `go test` rather than by the module-wide `go build ./...`, which does not
compile test files.
