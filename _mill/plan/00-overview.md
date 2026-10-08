# Plan: Enclose: path:line ranges -> enclosing member glyphs (batch, at revision, CLI verb)

```yaml
task: 'Enclose: path:line ranges -> enclosing member glyphs (batch, at revision, CLI verb)'
slug: enclose-locations
approved: false
discussion_sha: 95c7ff4d97a44cec9b772c29785898a6503d0e81
started: 20261008-063251
parent_branch: main
root: ""
verify: null
```

## Batch Index

_The fenced yaml block below is the authoritative DAG mill-go reads to schedule batches.
Every batch lives at `NN-<batch-slug>.md` in this directory and is mirrored as one entry here._

```yaml
batches:
  - number: 1
    name: engine-enclose
    file: 01-engine-enclose.md
    depends-on: []
    verify: go test ./internal/engine/
  - number: 2
    name: facade-enclose
    file: 02-facade-enclose.md
    depends-on: [1]
    verify: go test ./internal/repopath/ ./quarry/ ./internal/mcpserver/ ./internal/cli/
  - number: 3
    name: cli-enclose
    file: 03-cli-enclose.md
    depends-on: [2]
    verify: go test ./internal/cli/
```

## Shared Decisions

### Decision: discussion.md is the specification

- **Decision:** `_mill/discussion.md` is the authoritative specification for every rule this plan implements: the rejection order, the innermost rule, the line count, the location spelling, path normalisation, the two engine answer paths and the CLI verb.
  Cards name the identifiers and the files; where a card restates a rule it restates the discussion, and a disagreement between a card and the discussion is resolved in the discussion's favour.
  Every card's Context lists `_mill/discussion.md` for that reason.
- **Rationale:** the discussion went through six review rounds; re-deriving its rules in card prose would create a second, drifting copy.
- **Applies to:** all batches

### Decision: engine names

- **Decision:** the engine's new surface lives in a new file `internal/engine/enclose.go` and is named:
  `EncloseResult` (with `Rejected()`),
  the reason constants `EncloseReasonBadLocation` (`bad_location`), `EncloseReasonBadRange` (`bad_range`), `EncloseReasonOutsideRoot` (`outside_root`), `EncloseReasonUnaddressable` (`unaddressable`), `EncloseReasonUnsupportedLanguage` (`unsupported_language`), `EncloseReasonMissingFile` (`missing_file`), `EncloseReasonUnreadable` (`unreadable`), `EncloseReasonPastEOF` (`past_eof`), declared as untyped string constants in that order,
  `EncloseReasons` (a `[]string` in the same order),
  `Location`, `RevisionFiles`, `(*Repo).Enclose`, `(*Repo).EncloseFrom`,
  and the unexported `innermostSymbols`, `enclose`, `encloseFrom`, `verifyEncloseCoverage`.
  The facade aliases `EncloseResult`, the eight constants under the same names, and `EncloseReasons` as the engine's own slice.
- **Rationale:** mirrors the `NameReason*`/`NameReasons` naming and keeps facade and engine spellings identical.
- **Applies to:** all batches

### Decision: rejection message texts

- **Decision:** every rejected `EncloseResult` carries `Error` as one lower-case sentence, with no package prefix, built from these formats (`%s` substitutes the named value verbatim, `%d` an integer):
  - `bad_location`: `not a location: want path:line, path:line-line or path:line:col: %s` (the target)
  - `bad_range`: `bad range: start %d must be at least 1 and not after end %d`
  - `outside_root`: `path outside repository: %s` (the location's path half as given)
  - `unaddressable`, path half: `path cannot be spelled as a glyph unit: %s` (the normalised path when one exists, else the path half as given)
  - `unsupported_language`: `unsupported language: %s` (the file)
  - `missing_file`: `file not found: %s` (the file)
  - `unreadable`: `file unreadable: %s: %s` (the file, then the underlying error or `FileEntry.Error` text; a symlink uses the detail `symlink`)
  - `unaddressable`, unit half: `file's glyph unit cannot be spelled: %s` (the file)
  - `past_eof`: `range ends past end of file: end %d, file has %d lines`
- **Rationale:** the CLI golden pins these bytes, so the facade and engine cards must agree on them.
  The `unreadable` detail can carry an engine-prefixed error text (e.g. a `FileEntry.Error` such as `engine: a.go: not valid UTF-8`), the same text `toc` already emits for that file.
- **Applies to:** all batches

### Decision: test fixtures

- **Decision:** working-tree fixtures use each package's own `writeScratchTree` (under `.scratch/`, never `t.TempDir()`).
  Git fixtures reuse the package's existing git fixture type: `deltaFixture` in `quarry/delta_test.go` and `deltaCLIFixture` in `internal/cli/cli_test.go`, which build under `t.TempDir()` as they do today.
  A test that needs a real symlink, a `chmod 000` entry or invalid UTF-8 bytes creates it itself on the scratch root.
  A test using `chmod` skips when `os.Geteuid() == 0`.
- **Rationale:** matches the established helpers; git fixtures already use `t.TempDir()`, because a nested repository under `.scratch/` would sit inside this repository's own worktree.
- **Applies to:** all batches

## All Files Touched

- `internal/cli/cli.go`
- `internal/cli/doc.go`
- `internal/cli/enclose_golden_test.go`
- `internal/cli/enclose_test.go`
- `internal/cli/flags.go`
- `internal/cli/flags_test.go`
- `internal/cli/testdata/enclose/batch.json`
- `internal/cli/usage.go`
- `internal/engine/answer.go`
- `internal/engine/enclose.go`
- `internal/engine/enclose_revision_test.go`
- `internal/engine/enclose_test.go`
- `internal/engine/enclose_worktree_test.go`
- `internal/engine/memo.go`
- `internal/engine/memo_test.go`
- `internal/engine/toc.go`
- `internal/repopath/doc.go`
- `internal/repopath/target.go`
- `internal/repopath/target_test.go`
- `quarry/enclose.go`
- `quarry/enclose_test.go`
- `quarry/quarry.go`
- `quarry/quarry_test.go`
- `quarry/render.go`
- `quarry/render_test.go`
- `quarry/repo.go`
