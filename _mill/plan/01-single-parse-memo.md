# Batch: single-parse memo

```yaml
task: 'Resolve self-target path: per-call dirPackage memo (GH #34)'
batch: single-parse memo
number: 1
cards: 5
verify: LADDER_LOOMYARD_REPO=/home/knatte/Code/quarry/wts/resolve-self-dirpackage-memo/.scratch/loomyard-pin go test -count=1 ./internal/engine/ ./internal/cli/ ./internal/mcpserver/ ./quarry/
depends-on: []
```

## Batch Scope

This batch replaces the two-pass extraction (`dirPackage` for clauses, then `fileEntry` re-reading every file) with one per-call record memo that reads and parses each file at most once, and routes both consumers through it: the walk consumer (`walkDir`, `fileTargetAnswer`, and therefore exported `TOC` and Resolve's self path) and the member consumer (`symbolsOfUnit` / `symbolsOfDir`, and therefore Resolve's member path, Expand and `SpansOf`).
It is one batch because the memo, its two consumers and the deletion of `dirPackage` form one refactor that must leave every existing answer byte-identical; splitting it would ship an intermediate state with two extraction paths.
No exported signature, `TOCOptions` field or JSON shape changes.
Batch 2 consumes `unitMemo.files` and `fileMemo.builds` as its test seam.

## Cards

### Card 1: Pinned Loomyard clone and baseline run

- **Context:**
  - `internal/engine/loomyard_test.go`
- **Edits:** none
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Make the Loomyard suite run instead of skip, before any code changes, so the batch's `verify:` guards extraction against the pinned goldens.
  Run every command from the worktree root `/home/knatte/Code/quarry/wts/resolve-self-dirpackage-memo`; never `cd` anywhere.
  1. Run `git -C .scratch/loomyard-pin rev-parse HEAD`. If it prints a hash starting with `72c23d9` (the `loomyardPin` constant), the clone exists; skip to step 3.
  2. Otherwise remove any partial `.scratch/loomyard-pin` directory, then run `git clone --quiet --no-checkout /home/knatte/Code/loomyard-LYXHUB/loomyard .scratch/loomyard-pin` followed by `git -C .scratch/loomyard-pin checkout --quiet 72c23d9`, and re-run step 1's check.
     Never run any command that writes inside `/home/knatte/Code/loomyard-LYXHUB/loomyard` itself; the clone only reads it.
  3. Run `LADDER_LOOMYARD_REPO=$PWD/.scratch/loomyard-pin go test -count=1 -v ./internal/engine/ ./internal/cli/ ./internal/mcpserver/ ./quarry/` (the four packages both batches' `verify:` runs) and confirm every package reports `ok`, and that no test whose name contains `Loomyard` or `TwentyGlyphs` in any of the four packages reads `SKIP` — the `internal/cli` Loomyard after-goldens gate on the same pin.
     If any fails or skips at this baseline, stop and report: the refactor cannot be judged against a broken baseline.
- **Commit:** none

### Card 2: Per-call file record memo

- **Context:**
  - `internal/engine/walk.go`
  - `internal/engine/units.go`
  - `internal/engine/answer.go`
  - `internal/engine/golang.go`
  - `internal/engine/strategy.go`
  - `internal/engine/extension.go`
  - `internal/engine/headers.go`
  - `internal/engine/text.go`
  - `internal/engine/treesitter/treesitter.go`
- **Edits:** none
- **Creates:**
  - `internal/engine/memo.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Create `internal/engine/memo.go` in package `engine` with a file header comment stating that it holds the per-call record memo every extraction path reads, that a memo is a local of one exported call and dies with it (the engine's no-cache rule), and that each file is read and parsed at most once per call.
  Declare, all unexported:

  - `type fileRecord struct` with fields: `err string` (the error text `fileEntry` reports today), `header string`, `parsed bool` (a registered strategy parsed the file without a `treesitter.WithTree` error), `lang string` (set only when `parsed`), `test`, `generated`, `lossy bool`, `clause string`, `hasClause bool`, `packageDoc string`, `symbols []Symbol`, `withSymbols bool`.
    `withSymbols` records whether symbols were requested when the record was built, whatever the outcome: a non-language, unreadable, invalid-UTF-8 or parse-failed file built with symbols requested has `withSymbols` true and no symbols, and is never rebuilt for them.
    Doc comment: symbols carry the placeholder unit `""` and no `File`; consumers stamp a copy through `stampSymbols`, never the record itself.
  - `type dirVote struct { pkg, lang string; clauses map[string]string }`: a directory's dominant clause, the language of that clause's files, and the base-name-to-clause map of every file whose record has `hasClause`.
  - `type fileMemo struct` with fields `repo *Repo`, `allSymbols bool`, `records map[string]*fileRecord` keyed by repository-relative file path (`joinRel(dirRel, base)`), `votes map[string]dirVote` keyed by `dirRel`, and `builds map[string]int` keyed like `records`.
    The `builds` doc comment states it is the test seam for the parse-once guarantee: it counts record builds per file, incremented before each build (builds started, not builds that succeeded), read by tests and never by production code; each build performs at most one tree-sitter parse, so a count of at most 1 per file means at most one parse per file.
  - `func newFileMemo(r *Repo, allSymbols bool) *fileMemo`, allocating all three maps.
  - `func (r *Repo) buildRecord(dirRel, base string, withSymbols bool) *fileRecord` — the one extraction, reproducing the per-file behaviour of today's `fileEntry` plus the clause read of today's `PackageClause`, in this order:
    1. `os.ReadFile` the file; on error return a record whose `err` field holds the read error's text.
    2. If `!utf8.Valid(src)`, return a record with `err` set to `fmt.Sprintf("engine: %s: not valid UTF-8", joinRel(dirRel, base))`.
    3. Look up `LanguageForExtension(filepath.Ext(base))` and `StrategyFor(lang)`; when either is missing, return a record whose `header` is `HeaderForFile(base, src)` and nothing else.
    4. Otherwise call `treesitter.WithTree(lang, src, ...)` once; inside the callback set `header = FirstParagraph(strategy.Header(root, src))`, `test = strategy.TestFile(base)`, `generated = strategy.Generated(root, src)`, `lossy = partial`, `clause = strategy.Package(root, src)`, `packageDoc = strategy.PackageDoc(root, src)`, and, only when `withSymbols`, `symbols = strategy.Symbols("", root, src)`.
    Set the record's `withSymbols` field from the argument on every return path (steps 1, 2, 3, 5 and 6), so the reuse test in `record` never depends on whether extraction succeeded.
    5. If `WithTree` returns an error, return a record holding only `err` set to that error's text (no clause, no header), exactly as `fileEntry` discards everything on that path today.
    6. Otherwise set `parsed = true`, `lang = lang`, `hasClause = clause != ""`, and return it.
  - `func (m *fileMemo) record(dirRel, base string, wantSymbols bool) *fileRecord`: return the memoised record when present and either `!wantSymbols` or its `withSymbols` is true; otherwise increment `builds[path]`, call `m.repo.buildRecord(dirRel, base, m.allSymbols || wantSymbols)`, store and return it.
    Its doc comment records the rebuild case from the overview's "symbol extraction per memo kind" decision: an `allSymbols` memo builds every record with `withSymbols` true and so never rebuilds; a TOC memo requests each file once per call; the rebuild is counted so a regression shows in the parse-once tests.
  - `func (m *fileMemo) dirRecords(dirRel string, bases []string, wantSymbols func(base string) bool) map[string]*fileRecord`: calls `record` for each base and returns the records keyed by base name.
  - `func (m *fileMemo) dirVote(dirRel string, recs map[string]*fileRecord, memoise bool) dirVote`: when `memoise` and `votes[dirRel]` exists, return it.
    Otherwise build `clauses` from every record with `hasClause`, take `pkg, _ := UnitsForClauseMap(dirRel, clauses)`, and derive `lang` with the exact loop `walkDir` uses today (the first clause equal to `pkg` whose base has a language per `LanguageForExtension`); store under `dirRel` only when `memoise`.
    Doc comment: the vote reads records and never parses; `memoise` is false only for a vote set extended by an explicitly named, gitignored target, which must not leak into the directory's plain vote.
  - `func stampSymbols(syms []Symbol, unit, file string) []Symbol`: returns a new slice `make([]Symbol, 0, len(syms))` holding a copy of each symbol with `Glyph.Unit = unit`, `ID = Glyph.String()` and `File = file`.
    Doc comment: `goStrategy.Symbols` uses its unit argument only to build `Glyph` and `ID`, so stamping after the vote yields byte-identical symbols; the returned slice is never nil, matching `Symbols`' own return, so an empty list still encodes as `[]`.

  Nothing calls these yet; card 3 and card 4 wire them in.
  The file must build (`go build ./internal/engine/`); an `unused` lint finding at this intermediate commit is expected and is cleared by card 4.
- **Commit:** `feat(engine): add per-call single-parse file record memo`

### Card 3: Walk consumer reads records

- **Context:**
  - `internal/engine/memo.go`
  - `internal/engine/answer.go`
  - `internal/engine/ignore.go`
  - `internal/engine/extension.go`
  - `internal/engine/repo_test.go`
- **Edits:**
  - `internal/engine/walk.go`
  - `internal/engine/toc.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Route `TOC`, `walkDir` and `fileTargetAnswer` through a `*fileMemo` with no change to any answer.

  In `internal/engine/walk.go`:
  - Change `fileEntry` to `func (r *Repo) fileEntry(dirRel, base string, rec *fileRecord, v dirVote, wantSymbols bool, spellable map[string]bool) FileEntry`.
    It no longer reads or parses.
    It returns `FileEntry{Name: base, Error: rec.err}` when `rec.err != ""`; otherwise sets `Header = rec.header`; when `!rec.parsed` returns there (a non-language file, as today).
    For a parsed record it sets `Test`, `Generated`, `Lossy` from the record, `Package = rec.clause` when `rec.clause != v.pkg`, `Language = rec.lang` when `rec.lang != v.lang`, and, when `wantSymbols`, derives `unit := unitFor(dirRel, v.pkg, rec.clause)`, consults `spellable` exactly as today (calling `r.unitSpellable` on a miss), and when spellable sets `Symbols` to a pointer to `stampSymbols(rec.symbols, unit, "")`.
    A clause-less parsed record therefore gets unit `dirRel` via `unitFor`, as today.
    Rewrite its doc comment to describe this record-to-entry layer; keep the paragraphs on Error/Lossy exclusivity and on the per-directory `spellable` cache.
  - Change `walkDir` to take a trailing `m *fileMemo` parameter and pass it on in its recursive call.
    Replace the `dirPackage` call and the `fileEntry` loop: compute `want := wantSymbols && !identityOnly` (an identity-only answer carries no files, so its records need no symbols); `recs := m.dirRecords(dirRel, bases, func(string) bool { return want })` over the base names of `fileEntries`; `v := m.dirVote(dirRel, recs, true)`; use `v.pkg` and `v.lang` where `dirPkg` and `dirLang` were used; build `docs` from each record's non-empty `packageDoc`; build each file entry with `r.fileEntry(dirRel, base, recs[base], v, want, spellable)`.
    Pass `v.clauses` to `dirDoc`.
    Keep `dirPackage` itself in place; card 4 deletes it once the member consumer stops calling it.
  - Update `dirDoc`'s doc comment: its docs come from the records' package docs, not from "pass two".

  In `internal/engine/toc.go`:
  - Move the body of `TOC` into a new unexported `func (r *Repo) toc(target string, opts TOCOptions, m *fileMemo) (DirAnswer, error)` that passes `m` to `walkDir` and `fileTargetAnswer`.
    `TOC` becomes `return r.toc(target, opts, newFileMemo(r, false))`; add one sentence to `TOC`'s doc comment that it builds a fresh record memo, so each file is parsed at most once per call.
    `toc`'s doc comment states it is the memo-aware variant Resolve's self path calls with its own call-wide memo.
  - Change `fileTargetAnswer` to take a trailing `m *fileMemo`.
    In its entry loop compute whether the target itself is ignore-matched (`ig.match(childRel, false)` for the target's own path, evaluated for the target too, while still keeping the target in `fileEntries` as today); call that `targetIgnored`.
    Replace `dirPackage` and the per-file `fileEntry` loop: `recs := m.dirRecords(dirRel, bases, func(base string) bool { return base == targetBase && wantSymbols })`; `v := m.dirVote(dirRel, recs, !targetIgnored)`; set the answer's Package and Language fields from `v`; build `docs` from every record's non-empty `packageDoc`; call `r.fileEntry` only for the target base, and only when the target is not a symlink (the other entries were built and discarded today, and their records already hold everything `dirDoc` needs).
    A symlink target never enters `fileEntries`, so `recs[targetBase]` is nil for it: guard the `r.fileEntry` call on `targetEntry.Type()&fs.ModeSymlink == 0` and let the existing symlink branch produce the name-only entry otherwise, as `TestRepoTOC_SymlinkTargetIsNameOnlyNotFollowed` in `repo_test.go` requires.
  - Correct `fileTargetAnswer`'s doc comment: the explicitly named target joins the vote even when gitignored, because it is kept in the entries the vote reads; every other gitignored file stays out of the vote, matching `walkDir`.

  Remove the imports this card leaves unused in both files: in `walk.go`, `unicode/utf8`, `ts` and `treesitter` (used only by the old `fileEntry`); in `toc.go`, `path/filepath` (used only by the `dirLang` loop that moved into `dirVote`); and any other `go build ./internal/engine/` reports.
  This card's commit must compile.
- **Commit:** `refactor(engine): walk and file-target answers read the record memo`

### Card 4: Member consumer and Resolve share the memo

- **Context:**
  - `internal/engine/memo.go`
  - `internal/engine/toc.go`
  - `internal/engine/answer.go`
  - `internal/engine/ignore.go`
- **Edits:**
  - `internal/engine/resolve.go`
  - `internal/engine/walk.go`
  - `internal/engine/expand.go`
  - `internal/engine/resolve_test.go`
  - `internal/engine/expand_test.go`
  - `internal/engine/roundtrip_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  In `internal/engine/resolve.go`:
  - Add field `files *fileMemo` to `unitMemo` and set it in `newUnitMemo` with `newFileMemo(r, true)`.
    Rename field `parses` to `extractions`; its doc comment now says it counts unit extractions requested (`symbolsOfUnit` calls), not parses, and that `files.builds` is the only measure of parsing.
    Update `symbolsOf`'s doc comment and its increment to the new name, and make `symbolsOf` call `m.repo.symbolsOfUnit(unit, m.ig, m.files)`: the member path must build into the same `files` memo the self path uses, never a fresh one, which is the self/member sharing card 6 pins.
    Update `resolve`'s doc comment ("read parses afterwards") to name `extractions` and `files.builds`, and `Resolve`'s "each distinct unit is parsed exactly once per call by the memo" sentence to say each file is parsed at most once per call and each distinct unit is extracted once.
    Update `unitMemo`'s own doc comment so it no longer says a unit lookup equals a parse: the memo shares one record per file across every self and member target in the call.
  - Change `resolveSelfTarget` to `func (r *Repo) resolveSelfTarget(unit string, m *unitMemo) (ResolveResult, error)`, calling `r.toc(unit, TOCOptions{Depth: 0, Symbols: &symbolsOff}, m.files)` in place of `r.TOC(...)`; the disposition switch is unchanged.
    `resolveGlyphTarget` passes `m`.
    Update `resolveSelfTarget`'s doc comment: it calls the memo-aware `toc`, so TOC's rules are still inherited, and the parse cost per self target is now a memo lookup for every file already built in the call.
    Restate its "Symbols are switched off explicitly" paragraph as an emission rule only: a self glyph answers where a thing is, not what is inside it, so its listing carries no symbols.
    Drop the cost justification (the per-target tree-sitter parse and the 150 ms budget), because the call-wide memo extracts symbols for every record it builds and the switch no longer saves any extraction.
  - Change `symbolsOfUnit` to `func (r *Repo) symbolsOfUnit(unit string, ig *ignoreSet, m *fileMemo) ([]Symbol, error)` and pass `m` to `symbolsOfDir`.
    Change `symbolsOfDir` to take a trailing `m *fileMemo`.
    Keep its `os.ReadDir` (a failure still fails the call, which `TestResolve_ReadFailureFailsTheCall` pins) and its directory/symlink/ignore filter.
    Replace `dirPackage` and the per-file re-read: `recs := m.dirRecords(dirRel, bases, func(string) bool { return true })` over every filtered file (non-language files included, matching the walk's build set — today's `symbolsOfDir` never opens a non-language file, so a member-only call such as Expand or `SpansOf` now also reads each non-Go file in the unit directory once and computes its header; this extra I/O is accepted per the discussion's "Lazy, per-directory, single-parse extraction" decision, which gives every path one build rule so a later self target in the same call reuses those records, and `TestResolve_TwentyGlyphsUnder150ms` guards its cost); `v := m.dirVote(dirRel, recs, true)`; for each file in `fileEntries` order, skip it when `!recs[base].hasClause` (the existing clause-less exclusion, comment kept and reworded to cite the record), skip it when `unitFor(dirRel, v.pkg, rec.clause) != unit`, otherwise append `stampSymbols(rec.symbols, unit, joinRel(dirRel, base))...`.
    Rewrite both doc comments: `symbolsOfUnit` reads each file's record once per call rather than parsing it; `symbolsOfDir` reuses the call's directory vote instead of `dirPackage`.
  - `SpansOf` passes a throwaway `newFileMemo(r, true)` to `symbolsOfUnit`; its behaviour is unchanged.
  - Update the file header comment: `symbolsOfUnit` reads each file once per call through the record memo, and Resolve's self and member targets share that memo.
  - Remove the now-unused imports (`ts`, `treesitter`, `unicode/utf8`, and any other) that `go build` reports.

  In `internal/engine/walk.go`: delete `dirPackage` and any import it alone used.

  In `internal/engine/expand.go`: retarget the two doc-comment references to the memo's `parses` counter to `extractions`.

  In `internal/engine/resolve_test.go`: `TestResolve_ParsesEachUnitOnce` reads `m.extractions`; restate its doc comment and failure message as unit extractions requested, not parses; its assertion value does not change.
  In `internal/engine/expand_test.go`: `TestExpand_SelfGlyph` reads `m.extractions` and its doc comment says extractions; assertion unchanged.
  In `internal/engine/roundtrip_test.go`: `assertSymbolRoundTrip` builds one `newFileMemo(r, true)` before its per-unit loop and passes it to every `r.symbolsOfUnit` call.
  Rewrite its doc comment: the walk (through exported `TOC`, with its own memo) and the member lookup now run the same extraction code, `buildRecord`, over separately built records, so the round trip no longer compares two independent extractions; it proves the two consumers' filter and stamping layers agree (every walked symbol is found by the member path and vice versa) and that each ID round-trips through `glyph.Parse`; extraction correctness stays guarded by the TOC goldens, the Loomyard goldens and the strategy tests in `golang_test.go`.
  Update the file header's first paragraph to match.
- **Commit:** `refactor(engine): member lookups and Resolve self targets share one record memo`

### Card 5: Doc comments describe single-parse extraction

- **Context:**
  - `internal/engine/memo.go`
  - `internal/engine/toc.go`
  - `internal/engine/resolve.go`
- **Edits:**
  - `internal/engine/walk.go`
  - `internal/engine/units.go`
  - `internal/engine/text.go`
  - `internal/engine/delta.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  Comment-only edits; no code line changes.
  - `internal/engine/walk.go` file header: drop `dirPackage` from the list of functions this file holds.
    Replace the "How many times a file is parsed, and why" paragraph's two-pass explanation: each file is read and parsed once per call by `buildRecord` in `memo.go`, which extracts the clause, header, flags, package doc and symbols from one parse with a placeholder unit; the unit is a directory-level fact, so it is stamped onto a copy of the symbols after the directory's clause vote (`fileMemo.dirVote`) fixes it.
    Keep the cost paragraph, restated for one parse per file (drop "roughly 1.2 s" if it no longer follows from the stated 616 ms figure).
  - `internal/engine/units.go`: retarget every reference to `dirPackage` in the file header, `UnitsForClauseMap`, `PackageClause`, `ClauseMapForFiles` and its inline comment.
    The clause vote is now applied by `fileMemo.dirVote`; the per-file clause rule is the one `buildRecord` applies, under the same four conditions `PackageClause` lists.
    State plainly that `buildRecord` reads the clause in its own single parse rather than calling `PackageClause`, so the four no-clause conditions are written in two places, and that `PackageClause` keeps its exported contract for callers outside the package.
    For `ClauseMapForFiles`, replace "the same extension guard dirPackage does ahead of its own read" with a statement that its guard skips only files that could never record a clause.
  - `internal/engine/text.go`: `FirstParagraph`'s doc comment names `buildRecord`, in `memo.go`, as the one caller computing a file's header, in place of `fileEntry`.
  - `internal/engine/delta.go`: in the paragraph citing "walk.go's fileEntry" as a UTF-8 rejection site, cite `memo.go`'s `buildRecord` instead.
  Run `grep -n "dirPackage\|pass one\|pass two\|two parse passes" internal/engine/*.go` afterwards; it must print nothing.
- **Commit:** `docs(engine): describe single-parse extraction and unit stamping`

## Batch Tests

`verify:` runs the whole `internal/engine` package with `LADDER_LOOMYARD_REPO` pointed at card 1's pinned clone, plus the three packages whose goldens pin engine answers byte for byte from the outside: `internal/cli` (the Loomyard after-goldens), `internal/mcpserver` (`toc_golden_test.go`) and `quarry`.
Together they run in about 9 s.
Whole packages are the right scope rather than a `-run` filter: this batch rewrites the extraction every TOC, Resolve, Expand, SpansOf and round-trip test reads, so every test in these packages is affected.
The byte-for-byte guards are the committed TOC goldens (`golden_test.go`, `answer_test.go`), the Loomyard goldens and round trip, `TestResolve_ParsesEachUnitOnce`, `TestResolve_TwentyGlyphsUnder150ms`, and the clause, tie-break, symlink and gitignore cases in `toc_test.go`, `walk_test.go` and `repo_test.go`.
None of their assertions change.
