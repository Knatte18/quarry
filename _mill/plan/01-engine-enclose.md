# Batch: engine-enclose

```yaml
task: 'Enclose: path:line ranges -> enclosing member glyphs (batch, at revision, CLI verb)'
batch: engine-enclose
number: 1
cards: 5
verify: go test ./internal/engine/
depends-on: []
```

## Batch Scope

This batch delivers the whole engine side of enclose in `internal/engine`: a byte-source seam and a line count on the per-call record memo, the result and input types with the closed reason vocabulary, the pure innermost-member selection, and the two answer paths — `(*Repo).Enclose` over the working tree and `(*Repo).EncloseFrom` over a revision reached through the git-ignorant `RevisionFiles` interface.
The next batch's facade consumes `Location`, `EncloseResult`, the reason constants, `EncloseReasons`, `RevisionFiles`, `Enclose` and `EncloseFrom`.
The engine decides rejection checks 5–9 (`unsupported_language`, `missing_file`, `unreadable`, unit-half `unaddressable`, `past_eof`) and passes a pre-rejected `Location` (checks 1–4, decided by the facade) through unchanged.
It is one batch because every card edits or reads the same handful of engine files and the answer paths are untestable without the types and the seam.

## Cards

### Card 1: memo byte-source seam and line count

- **Context:**
  - `_mill/discussion.md`
  - `internal/engine/toc.go`
  - `internal/engine/walk.go`
  - `internal/engine/scratchtree_test.go`
  - `internal/engine/toc_test.go`
- **Edits:**
  - `internal/engine/memo.go`
  - `internal/engine/memo_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  - Add a field `read func(rel string) ([]byte, error)` to `fileMemo`, documented as the byte source every record build reads through, keyed by repository-relative file path.
    `newFileMemo` sets it to a closure that splits `rel` with `splitDirBase` and returns `os.ReadFile(filepath.Join(r.absDir(dir), base))`, which is byte-for-byte the read `buildRecord` performs today.
  - Change `buildRecord`'s signature to `func (r *Repo) buildRecord(dirRel, base string, withSymbols bool, read func(rel string) ([]byte, error)) *fileRecord` and replace its `os.ReadFile` call with `read(joinRel(dirRel, base))`; a read error still becomes `rec.err`.
    `fileMemo.record` passes `m.read`.
    Every existing caller keeps its behaviour, because the default closure reads the same path.
  - Add an unexported pure function `lineCount(src []byte) int`: the number of `\n` bytes, plus one when `src` is non-empty and its last byte is not `\n`; zero for an empty `src`.
  - Add a field `lines int` to `fileRecord`, set from `lineCount(src)` on every record whose read succeeded: the invalid-UTF-8 record, the header-only record, the failed-parse record and the parsed record.
    Update `fileRecord.err`'s doc comment so it says every other field except `lines` is unset on an error record.
  - Tests in `memo_test.go`:
    a table test `TestLineCount` covering the empty input, `a`, `a\n`, `a\nb`, a lone `\n`, and `a\n\n`;
    a test that replaces a fresh memo's `read` with a map-backed fake, builds a record through `fileMemo.record`, and asserts the record was built from the fake's bytes (its clause and its `lines`), with `builds` counting one build;
    a test that a fake `read` returning an error yields a record whose `err` carries that error's text.
- **Commit:** `feat(engine): read memo records through a byte source and count lines`

### Card 2: enclose result, input and reason vocabulary

- **Context:**
  - `_mill/discussion.md`
  - `internal/engine/name.go`
  - `internal/engine/name_test.go`
  - `internal/engine/resolve.go`
- **Edits:**
  - `internal/engine/answer.go`
- **Creates:**
  - `internal/engine/enclose.go`
  - `internal/engine/enclose_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  - Create `internal/engine/enclose.go` with a file header comment in the style of the surrounding files, declaring:
    - `EncloseResult`, with exactly these fields, in this order, and these JSON tags: `Target string` (`target`), `File string` (`file,omitempty`), `Start int` (`start,omitempty`), `End int` (`end,omitempty`), `Status Status` (`status,omitempty`), `Symbols []Symbol` (`symbols,omitempty`), `Unit string` (`unit,omitempty`), `Lossy bool` (`lossy,omitempty`), `Error string` (`error,omitempty`), `Reason string` (`reason,omitempty`).
      Each field gets a doc comment following the discussion's "Status set and result shape" decision.
      `Unit`'s doc comment states that, unlike `ResolveResult.Unit` (a `Status`), this key holds the file's self glyph string (e.g. `a/b.go#`), is present only on not_found, and that its presence alone means "not inside any member".
      `File`/`Start`/`End`'s doc comment states they are filled on a rejection whenever they were determined before it.
    - `func (r EncloseResult) Rejected() bool { return r.Status == "" }`, documented as `ResolveResult.Rejected` is.
    - The eight reason constants and `EncloseReasons` exactly as the overview's "engine names" Shared Decision lists them, each constant documented with the condition the discussion's "Rejection vocabulary" decision gives it, and `EncloseReasons` documented the way `NameReasons` is.
    - `Location`, a struct with fields `Target, File string`, `Start, End int`, `Reason, Error string`, documented as one normalised input (File, Start and End set) or one input the facade already rejected (Reason and Error set, plus whichever of File/Start/End were determined).
    - `RevisionFiles`, an interface with `GoFiles(dirRel string) ([]string, error)` (the immediate Go children of `dirRel` at the revision, repository-relative, `.` meaning the root) and `Read(rel string) ([]byte, error)` (one file's bytes at the revision), documented as the engine's git-ignorant view of one revision.
  - In `internal/engine/answer.go`, rewrite the doc comment of `Status` so it names the subsystem instead of listing its users: it is the closed per-entry vocabulary of docs/glyph.md §5 shared by the engine's per-target result types, and a result type's unit key drawn from it carries only StatusFound or StatusNotFound.
    Do not change `Statuses`, `Known`, or any JSON tag in `answer.go`.
  - Tests in `enclose_test.go`:
    `TestEnclose_ReasonCompleteness`, mirroring `TestName_ReasonCompleteness` — a literal want-map of the eight constants, `len` equality, no duplicates, no unexpected value;
    a test that `Rejected()` is true exactly when `Status` is empty;
    a test that `json.Marshal` of a fully populated `EncloseResult` emits its keys in the declared order.
- **Commit:** `feat(engine): declare EncloseResult, Location, RevisionFiles and the enclose reasons`

### Card 3: innermost-member selection

- **Context:**
  - `_mill/discussion.md`
  - `internal/engine/answer.go`
- **Edits:**
  - `internal/engine/enclose.go`
  - `internal/engine/enclose_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  - Add the pure function `innermostSymbols(candidates []Symbol, start, end int) []Symbol` to `enclose.go`, implementing the discussion's "Innermost rule by span containment" decision exactly:
    keep the symbols whose `[Start, End]` overlaps `[start, end]`;
    a child of kept symbol P is another kept symbol whose span lies within P's span and is not identical to it;
    P survives when some line in the overlap of `[start, end]` and P's span is covered by none of P's children, and a P with no children always survives;
    survivors are returned in source order, sorted stably by `Start` then `End` so input order breaks ties, deduplicated by identity (same `ID` and same `Start`).
    The result is never nil.
    Nesting is decided by span containment only, never by `Glyph.Owner`.
  - Its doc comment states the rule and names the consequences the discussion lists (interface method line, interface head plus method, receiver method outside its type's span, one-line interface returning type then method).
  - Add a table test `TestInnermostSymbols` in `enclose_test.go` over hand-built `Symbol` values covering: one line in one function; a range spanning two top-level members (both, source order); a line in an interface method (method only); a range over the interface head plus one method (type then method); a receiver method whose span lies outside its type's span (never the type's child); a one-line interface whose type and method share a span (both, type first); a line touching no symbol (empty, non-nil); a duplicate symbol in the input (returned once).
- **Commit:** `feat(engine): select the innermost enclosing members of a line range`

### Card 4: working-tree Enclose

- **Context:**
  - `_mill/discussion.md`
  - `internal/engine/toc.go`
  - `internal/engine/walk.go`
  - `internal/engine/memo.go`
  - `internal/engine/repo.go`
  - `internal/engine/answer.go`
  - `internal/engine/resolve.go`
  - `internal/engine/extension.go`
  - `internal/engine/strategy.go`
  - `internal/engine/scratchtree_test.go`
  - `internal/engine/toc_test.go`
  - `internal/engine/memo_test.go`
  - `glyph/self.go`
  - `glyph/glyph.go`
- **Edits:**
  - `internal/engine/enclose.go`
- **Creates:**
  - `internal/engine/enclose_worktree_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  - Add `func (r *Repo) Enclose(locs []Location) ([]EncloseResult, error)`, which returns `r.enclose(locs, newFileMemo(r, true))`, and the unexported worker `func (r *Repo) enclose(locs []Location, m *fileMemo) ([]EncloseResult, error)`, which takes the memo so a test can read `m.builds` (the `resolve`/`unitMemo` pattern in `resolve.go`).
    The worker allocates `make([]EncloseResult, len(locs))`, so a nil or empty input yields an empty, non-nil slice, fills every index in order, calls `verifyEncloseCoverage(locs, results)` and returns a nil error: the working-tree path has no whole-call failure.
  - Add `verifyEncloseCoverage(locs []Location, results []EncloseResult)`, which panics on a length mismatch or on `results[i].Target != locs[i].Target`, documented with `verifyResolveCoverage`'s rationale.
  - Per location, in this order (the discussion's "Rejection vocabulary" checks 5–9 and its "Engine architecture" working-tree bullet):
    1. A `Location` whose `Reason` is set is returned unchanged as an `EncloseResult` carrying its `Target`, `File`, `Start`, `End`, `Reason` and `Error`.
    2. `unsupported_language` when `LanguageForExtension(filepath.Ext(loc.File))` or `StrategyFor` of its language reports false.
    3. The file's outcome, computed once per distinct file per call and memoised in a call-local map:
       `os.Lstat` of the file under the root: a not-exist error, `syscall.ENOTDIR` (`errors.Is`), or a directory is `missing_file`; a symlink is `unreadable` with detail `symlink`; any other `Lstat` error is `unreadable`.
       Then `r.toc(loc.File, TOCOptions{Symbols: &on}, m)` with `on` true: `errors.Is(err, ErrTargetNotFound)` is `missing_file`, any other error is `unreadable` (never a whole-call error).
       `entry := answer.Files[0]`; a non-empty `entry.Error` is `unreadable` carrying that text.
       The record `rec := m.records[loc.File]` gives the clause and the line count; the unit is `unitFor(dir, answer.Package, rec.clause)` with `dir` from `splitDirBase(loc.File)` — read from the `DirAnswer` this same call returned, never from `m.votes`.
       When `r.unitSpellable(unit)` is false the outcome is unit-half `unaddressable`, decided before `entry.Symbols` is read.
       Otherwise the candidates are `*entry.Symbols` with each copy's `File` set to `loc.File`, `lines` is `rec.lines`, and `lossy` is `entry.Lossy`.
    4. `past_eof` when `loc.End > lines` (an empty file has zero lines, so every location in it is `past_eof`).
    5. Otherwise `innermostSymbols(candidates, loc.Start, loc.End)`: a non-empty result is `Status: StatusFound` with `Symbols`; an empty one is `Status: StatusNotFound` with `Unit` set to the `String()` of `glyph.Self(glyph.Go, loc.File)`.
       `Lossy` is set on both. Should `glyph.Self` fail (the facade's check 4 makes this unreachable for facade input), the item is path-half `unaddressable` instead.
    Every rejection after step 1 carries `Target`, `File`, `Start` and `End`; no rejection carries `Unit`.
    `Error` texts follow the overview's "rejection message texts" Shared Decision.
  - Keep the per-location answer assembly (steps 4–5 and the rejection shapes) in helpers the revision path in card 5 reuses, so the two paths answer a file outcome identically.
  - `Enclose`'s doc comment states the positional contract, the one-build-per-file guarantee, that nothing outlives the call, and that the working-tree limits are documented on the facade's `Enclose`.
  - Tests in `enclose_worktree_test.go`, against scratch trees, covering every working-tree case the discussion's "Testing" section lists under "Engine, working tree" and "Build count" (working-tree half) and "Coverage verifier":
    single line; a range over two top-level members; an interface method line; an interface head plus a method; a closure line; a doc-comment line; a `const (` line (not_found with `unit`); an import-block line (not_found, `unit` equal to `<file>#`); a lossy file; a range ending past EOF; an empty file; a `_test.go` member of an external test package (the `_test` unit); an explicitly named gitignored file whose inclusion changes the directory's vote, located alone and after another location in the same directory (unit from that call's `DirAnswer.Package`); a one-line interface; `a.go/b.go:1` (`missing_file`); `unsupported_language`, `missing_file`, `unreadable` for a symlink and for invalid UTF-8, and `past_eof`; a pre-rejected `Location` passed through unchanged; a root-level `.go` file (unit-half `unaddressable`, no `unit`); an unreadable directory and an unreadable `.gitignore` beside a good location in the same batch (the bad item rejected, the good one answered, nil error); nil and empty input; duplicate locations answered twice;
    `builds[file] == 1` for several locations in one file, and each file of a directory built once for locations in two of its files;
    `verifyEncloseCoverage` panicking on a length mismatch and on a `Target` mismatch.
- **Commit:** `feat(engine): answer enclose locations against the working tree`

### Card 5: revision EncloseFrom

- **Context:**
  - `_mill/discussion.md`
  - `internal/engine/toc.go`
  - `internal/engine/walk.go`
  - `internal/engine/memo.go`
  - `internal/engine/units.go`
  - `internal/engine/answer.go`
  - `internal/engine/scratchtree_test.go`
  - `internal/engine/toc_test.go`
  - `internal/engine/enclose_worktree_test.go`
- **Edits:**
  - `internal/engine/enclose.go`
- **Creates:**
  - `internal/engine/enclose_revision_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  - Add `func (r *Repo) EncloseFrom(files RevisionFiles, locs []Location) ([]EncloseResult, error)`, which returns `r.encloseFrom(files, locs, newFileMemo(r, true))`, and the unexported worker `func (r *Repo) encloseFrom(files RevisionFiles, locs []Location, m *fileMemo) ([]EncloseResult, error)`.
  - The worker installs as `m.read` a closure over `files.Read` that records the first error it sees before returning it.
    `buildRecord` turns every read failure into `rec.err`, so this capture is the only way a `Read` failure reaches the caller.
  - Per location, steps 1 (pass-through) and 2 (`unsupported_language`) as in card 4.
    Then, per distinct directory per call (memoised in a call-local map), with `dir` from `splitDirBase(loc.File)`:
    `files.GoFiles(dir)` lists the vote set (a `GoFiles` error fails the whole call); the bases are `path.Base` of each listed path; the records are `m.dirRecords(dir, bases, …)`; immediately after building them, a captured `Read` error fails the whole call with that error; the vote is `m.dirVote(dir, recs, true)`.
    Per file: a base absent from the listing is `missing_file`; a record with `err` set is `unreadable` carrying that text; the unit is `unitFor(dir, v.pkg, rec.clause)`, gated by `r.unitSpellable` exactly as in card 4 (`unaddressable` on failure); only then are the candidates built with `stampSymbols(rec.symbols, unit, loc.File)`, with `lines` from `rec.lines` and `lossy` from `rec.lossy`.
    Steps 4–5 reuse card 4's helpers.
  - On a whole-call failure the worker returns a nil slice and the error; on success it calls `verifyEncloseCoverage`.
  - `EncloseFrom`'s doc comment states that the only whole-call failures are a `GoFiles` or `Read` error, that each file is built once per call, and that the revision asymmetries are documented on the facade's `EncloseAt`.
  - Tests in `enclose_revision_test.go` with an in-test map-backed `RevisionFiles` fake (paths to bytes, with optional injected `GoFiles` and `Read` errors), on a `Repo` opened over a scratch tree whose working-tree contents differ from the fake's:
    a found answer with the fake's line numbers, not the working tree's;
    a base absent from the fake's directory listing (`missing_file`);
    a `Read` failure failing the whole call (nil slice, the error) rather than answering `unreadable`;
    a `GoFiles` failure failing the whole call;
    a directory whose dominant clause at the fake revision differs from the working tree's, giving the revision's unit;
    a root-level `.go` file (`unaddressable`, matching card 4's working-tree answer for the same file);
    invalid UTF-8 bytes (`unreadable`);
    a pre-rejected `Location` and an `unsupported_language` location;
    `builds[file] == 1` for several locations in one file, and once per file for locations in two files of one directory.
- **Commit:** `feat(engine): answer enclose locations against a revision`

## Batch Tests

`verify: go test ./internal/engine/` runs the whole engine package.
The scope is the package rather than the new test files alone, because card 1 changes `buildRecord`, which every extraction path in the package reads through, so every existing engine test is a regression test for it; the package suite runs in about a second.
New tests: `TestLineCount` and the read-seam tests in `memo_test.go`; the vocabulary, key-order and `TestInnermostSymbols` tests in `enclose_test.go`; `enclose_worktree_test.go`; `enclose_revision_test.go`.
