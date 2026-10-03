# Plan: Resolve self-target path: per-call dirPackage memo (GH #34)

```yaml
task: 'Resolve self-target path: per-call dirPackage memo (GH #34)'
slug: resolve-self-dirpackage-memo
approved: false
started: 20261003-082232
parent_branch: main
root: ""
verify: null
discussion_sha: dced01f6d83c40c9e0a1f3e5e55eaee379473086
```

## Batch Index

_The fenced yaml block below is the authoritative DAG mill-go reads to schedule batches.
Every batch lives at `NN-<batch-slug>.md` in this directory and is mirrored as one entry here._

```yaml
batches:
  - number: 1
    name: single-parse memo
    file: 01-single-parse-memo.md
    depends-on: []
    verify: LADDER_LOOMYARD_REPO=/home/knatte/Code/quarry/wts/resolve-self-dirpackage-memo/.scratch/loomyard-pin go test -count=1 ./internal/engine/
  - number: 2
    name: parse-once tests and Loomyard measurement
    file: 02-parse-once-tests.md
    depends-on: [1]
    verify: LADDER_LOOMYARD_REPO=/home/knatte/Code/quarry/wts/resolve-self-dirpackage-memo/.scratch/loomyard-pin go test -count=1 ./internal/engine/
```

## Shared Decisions

### Decision: fileMemo is its own type, held by unitMemo

- **Decision:** a new unexported type `fileMemo` in a new file `internal/engine/memo.go` holds the per-file records, the memoised directory votes and the per-file record-build counter.
  `unitMemo` gains one field, `files *fileMemo`, built by `newUnitMemo` with `newFileMemo(r, true)`.
  Exported `TOC` builds `newFileMemo(r, false)` and delegates to an unexported `toc`.
  `SpansOf` builds a throwaway `newFileMemo(r, true)` for its one `symbolsOfUnit` call.
- **Rationale:** TOC needs the record memo but not `unitMemo`'s root ignore set or its unit maps, so the record memo stands alone.
  A memo is always a local of an exported entry point (`TOC`, `Resolve`, `Expand`, `SpansOf`), which keeps the engine's no-cache rule.
- **Applies to:** all batches

### Decision: symbol extraction per memo kind

- **Decision:** `newFileMemo(r, allSymbols)`.
  With `allSymbols` true (Resolve, Expand, SpansOf), every record build extracts symbols.
  With `allSymbols` false (exported TOC), a record build extracts symbols only when the requesting consumer wants that file's symbols.
  A request that wants symbols from a record built without them rebuilds the record and counts a second build.
  This is unreachable today, because one exported TOC call requests each file at most once, and the counter makes any future regression visible to the parse-once tests rather than silently re-parsing.
- **Rationale:** the discussion's "Symbol extraction cost" rule, with a defined behaviour for the case it rules out.
- **Applies to:** all batches

### Decision: record layout and stamping

- **Decision:** a `fileRecord` holds the file's read/UTF-8/parse error text, its final header (for a parsed file, `FirstParagraph` of the strategy header; for a non-language file, `HeaderForFile`), whether it was parsed by a strategy, its language, test, generated and lossy flags, its clause plus whether a clause was recorded, its package doc, and its symbols extracted with the placeholder unit `""`.
  It never holds `Symbol.File`.
  Consumers stamp a fresh copy through one helper, `stampSymbols(syms []Symbol, unit, file string) []Symbol`, which sets `Glyph.Unit`, recomputes `ID` as `Glyph.String()`, sets `File`, and returns a non-nil slice (`make([]Symbol, 0, len(syms))`), matching `goStrategy.Symbols`' own always-non-nil return so an empty symbol list still encodes as `[]`.
  The walk consumer passes `file == ""`; the member consumer passes the repository-relative path.
- **Rationale:** the discussion keeps the pre-truncation header as the extraction input; `FirstParagraph` is a pure function applied once per file, so storing its output gives the same answer with one fewer rule at the consumer.
  `Glyph.Owner` is shared between the record and its copies; no code mutates it.
- **Applies to:** all batches

### Decision: vote memo keyed by directory for the plain file set only

- **Decision:** `fileMemo.dirVote(dirRel, recs, memoise)` computes the directory's dominant clause, its language and its clauses map from records through `UnitsForClauseMap`, with no parse.
  It stores the result under `dirRel` only when `memoise` is true.
  `fileTargetAnswer` passes `memoise == false` when its target file is ignore-matched, since that file then joins the vote set; every other call passes true.
- **Rationale:** the discussion leaves the gitignored-target vote to the plan; recomputing it costs a map walk and no parse, and keeps the memo to one key per directory.
- **Applies to:** all batches

### Decision: names after the refactor

- **Decision:** `dirPackage` is deleted; its vote role moves to `fileMemo.dirVote` and its clause-read role to `buildRecord`.
  `fileEntry` keeps its name and becomes the walk consumer's record-to-`FileEntry` layer, so `text.go`'s and `delta.go`'s references to it stay pointed at the right function.
  `unitMemo.parses` is renamed `extractions`; `fileMemo.builds` is the only measure of parsing.
- **Rationale:** a counter named `parses` that no longer counts parses misleads every reader of the parse-once tests.
- **Applies to:** all batches

### Decision: Loomyard suite runs against a pinned scratch clone

- **Decision:** both batches' `verify:` sets `LADDER_LOOMYARD_REPO` to `.scratch/loomyard-pin`, a clone of the Loomyard prime checked out at `loomyardPin` (`72c23d9`).
  Card 1 makes that clone exist and records the baseline pass.
  The clone is made with `git clone` from the prime, which reads the prime and writes nothing to it.
- **Rationale:** without the variable the Loomyard goldens and round trip skip, and they are the discussion's main extraction guard through this refactor.
  The whole engine package with the Loomyard suite runs in about 9 s.
- **Applies to:** all batches

## All Files Touched

- `internal/engine/delta.go`
- `internal/engine/expand.go`
- `internal/engine/expand_test.go`
- `internal/engine/memo.go`
- `internal/engine/memo_test.go`
- `internal/engine/resolve.go`
- `internal/engine/resolve_test.go`
- `internal/engine/roundtrip_test.go`
- `internal/engine/text.go`
- `internal/engine/toc.go`
- `internal/engine/units.go`
- `internal/engine/walk.go`
