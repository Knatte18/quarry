# Discussion: Enclose: path:line ranges -> enclosing member glyphs (batch, at revision, CLI verb)

```yaml
task: 'Enclose: path:line ranges -> enclosing member glyphs (batch, at revision, CLI verb)'
slug: enclose-locations
status: discussing
parent_branch: main
```

## Problem

Loomyard turns tool output into kick-start packs for recovery agents and fixers.
That output cites `file:line`, not glyphs: `go test` failure lines, `go vet` and compile errors (`file:line:col`), comment-lint findings, Webster audit findings, and review findings that cite path:line.
Loomyard's standing rule is that glyph<->path conversion happens only in quarry, so the path:line -> glyph direction must be a quarry API.
Caller-side intersection of `Glyphs(target)` spans was rejected: "innermost enclosing member" is quarry semantics (doc-sibling `Start`, lossy files, nesting) and must not be re-derived by consumers.

The most important case is mapping at a revision.
Findings are produced against the reviewed revision while the fixer's tree moves as it edits;
Loomyard maps at the review revision, then `Resolve`s the glyphs in the working tree for current spans.
That is the line-shift problem glyphs exist to solve.

Requested by ly:orch (Loomyard hub) on 2026-10-08 and approved by the operator.
The requirements and the nine "Agreed decisions" below were settled between qu:orch and ly:orch and are input, not open questions;
this discussion adds the architecture and the gaps the agreement left.

## Scope

**In:**

- Engine (`internal/engine`): `EncloseResult`, the closed rejection-reason vocabulary with an enumerating slice, a `Location` input type, the innermost-member selection, the working-tree and revision answer paths, a coverage verifier, and a byte-source seam on `fileMemo`.
- Facade (`quarry/`): `Enclose(targets)`, `EncloseAt(rev, targets)`, location-spelling parsing, path normalisation through `internal/repopath`, a git-backed revision source, type/constant/slice aliases, and `RenderEncloseJSON`.
- `internal/repopath`: re-point its two sentinel references from `quarry` to `engine` so the facade can import it without an import cycle.
- CLI (`internal/cli`): an `enclose` verb with `--rev` and `--stdin`, JSON output, usage text, an unexported stdin-taking `run` behind `Run`.
- Tests listed under Testing, including a CLI golden.

**Out:**

- `DeltaGit` and the diff case (already covered).
- Markdown or any non-Go language: a `.md` location is an `unsupported_language` rejection.
- An MCP tool (agreed decision 9).
- A `--text` rendering for `enclose`.
- Any change to `Statuses`, `Status.Known()`, `ResolveResult`'s key set, or docs/glyph.md §5.
- Clipping ranges past EOF, guessing the package of a bare file name, struct-field members.

## Decisions

The agreed decisions from the task (status set, innermost rule, doc lines, lossy, result type, one rev per call, rejection reasons, CLI input, no MCP) hold as written in the task body and are restated here where the plan needs them, extended where they left a gap.

### Status set and result shape

- Decision: `EncloseResult` is a new engine type, aliased by the facade, with JSON keys in this order: `target` (input spelling verbatim, always present), `file` (normalised repo-relative path, omitempty), `start`, `end` (omitempty), `status` (`found` or `not_found`, omitempty — empty means rejected), `symbols` (`[]Symbol`, found only), `unit` (the file's self glyph string, e.g. `a/b.go#`, not_found only), `lossy` (bool, omitempty), `error` and `reason` (rejection only, omitempty).
  `Rejected()` returns `Status == ""`, as on `ResolveResult`.
  `Status` reuses the engine's `Status` type and constants; no new value.
  A range overlapping several members is `found` with all of them.
  `file`/`start`/`end` are filled on a rejection whenever they were determined before the rejection (e.g. `past_eof` carries all three; `bad_location` carries none).
- Rationale: agreed decisions 1 and 5; key order mirrors `ResolveResult`.
- Rejected: reusing `ResolveResult` (would open its closed key set); a new `multiple` status (agreed decision 1).

### The `unit` key holds a glyph string, not a Status

- Decision: keep the agreed key name `unit`, holding the file's self glyph (`<file>#`, built with `glyph.Self(glyph.Go, file)`), present only on not_found.
  Its doc comment states explicitly that, unlike `ResolveResult.Unit` (a `Status` drawn from found/not_found), this `unit` is a glyph and its presence alone means "not a member".
  At a revision it is the same spelling: the file self glyph does not depend on the clause vote.
- Rationale: the key name is settled with the consumer; the clash with docs/glyph.md §5's `unit` vocabulary is confined to a different result type and documented at the field.
- Rejected: renaming to `self` (reopens a settled consumer contract); the package self glyph `a/b#` (the requirement says "cite the file").

### Innermost rule by span containment

- Decision: candidates are the file's symbols whose `[Start, End]` overlaps the location's `[Start, End]`.
  A child of candidate P is another candidate whose span lies within P's span and is not identical to it.
  P survives when the location touches at least one line of P that none of P's children covers;
  a P with no children always survives.
  Survivors are returned in source order (by `Start`, then `End`), deduped by identity (same ID and same `Start`).
  Nesting is decided by span containment, not by `Glyph.Owner`: a receiver method has an owner but sits outside its type's span, while an interface method sits inside it.
- Consequences: a line in an interface method returns the method;
  a range over an interface's head lines plus one method returns both, type first;
  a line inside a closure maps to the enclosing function (closures are not symbols);
  a line inside a struct body maps to the type;
  a one-line interface `type I interface{ M() }` (no doc on either) gives the type and its method identical spans, so neither is the other's child and a location on that line returns both, type first — intended, not a case to special-case;
  a line on `const (`, `var (`, `type (` or a group's closing `)` is not_found (each spec is its own symbol, the group has none).
- Rationale: agreed decision 2's "none of its own child members overlaps the range", read literally, would drop the interface type when the range spans head and method, contradicting that decision's own head-plus-method example.
  The uncovered-line form satisfies both sentences.
- Rejected: owner-chain nesting (misclassifies receiver methods); returning only the deepest symbol (loses the head lines).

### Doc lines and lossy

- Decision: a line inside a sibling doc comment belongs to that member because `Symbol.Start` already includes it — no extra logic.
  A partial parse sets `lossy: true` on found and not_found answers and never rejects the item.
- Rationale: agreed decisions 3 and 4.

### Rejection vocabulary

- Decision: the closed vocabulary is eight reasons, declared as engine constants with an `EncloseReasons` slice (the `NameReasons` pattern, aliased by the facade as the same slice):
  `bad_location`, `bad_range`, `outside_root`, `unaddressable`, `unsupported_language`, `missing_file`, `unreadable`, `past_eof`.
  Checked in this order, first match wins:
  1. `bad_location` — the target is not `path:line`, `path:line-line` or `path:line:col` (non-numeric or overflowing line, empty path, missing line).
  2. `bad_range` — Start < 1, or Start > End.
  3. `outside_root` — the path normalises outside the repository root (including `..` escapes and absolute paths elsewhere).
  4. `unaddressable` (path half) — the normalised path cannot be spelled as a Go glyph: `repopath` reports `ErrTargetHasSeparator`, or `glyph.Self(glyph.Go, file)` fails (a backslash, whitespace or control character in a segment).
     Needs the normalised path, hence its place after `outside_root`.
  5. `unsupported_language` — the extension maps to no language or no registered strategy (`LanguageForExtension` + `StrategyFor`); checked before existence, so a missing `.md` is `unsupported_language`.
  6. `missing_file` — no regular file at the path (working tree: `os.Lstat` not-exist, a directory, or an `r.toc` error matching `ErrTargetNotFound`; revision: the base name is not among the directory's Go files at that revision). A bare `foo_test.go:42` lands here unless a root-level file of that name exists.
  7. `unreadable` — working tree: the path is a symlink, the file cannot be read, is not valid UTF-8, or tree-sitter fails outright (`FileEntry.Error`), or `r.toc` returns any other error for this item (a `.gitignore` read failure, an `os.ReadDir` failure, the vanished-target race).
     Revision: the record's `err` (invalid UTF-8, a failed parse).
  8. `unaddressable` (unit half) — the file's glyph unit, `unitFor(dir, v.pkg, rec.clause)`, fails `unitSpellable`.
     Every file directly under the repository root hits this (`unitFor(".")` returns `""`), as does a directory segment the alphabet rejects.
     Checked identically on both paths before any symbol is stamped, so neither path mints a unit `glyph.Parse` rejects and both answer the same file the same way.
  9. `past_eof` — End is greater than the file's line count.
  An `unaddressable` rejection carries `start`, `end` and, when normalisation produced one, `file`, but never `unit` (no glyph can name the file's members, and the item is rejected rather than not_found).
  The `ErrTargetHasSeparator` case has no normalised path (`RepoRelTarget` returns `""` with that error), so it omits `file`, per the "filled whenever determined" rule.
- Rationale: agreed decision 7 fixed five reasons but left three real inputs unanswered: an input string that is not a location at all (a stray line piped through `--stdin`), a file that exists but cannot be read as source, and a file whose members quarry's glyph contract cannot name (root-level files, unspellable segments).
  None may fail the whole call (requirement 1), and folding them into an existing reason would mislabel them: `missing_file` for a permission error tells the caller to fix the path, and a silent `not_found` for a line inside a real root-level function tells the caller the line is outside every member.
  `unaddressable` mirrors the walk's own rule (`unitSpellable`'s doc comment): emitting nothing is the honest answer to a name the contract cannot spell; here "nothing" must be a visible rejection, since Enclose is asked about one specific file.
  Three added words keep the vocabulary closed and honest; the consumer reads the slice, not a hard-coded list.
- Rejected: whole-call failure for a malformed spelling (one stray stdin line would kill a pack); mapping unreadable to `not_found + lossy` (a symlink or EACCES is not a partial parse); folding the unspellable unit into `bad_location` (the location is well formed, the limit is the glyph contract's).

### Line count

- Decision: a file's line count is the number of `\n` bytes plus one when the file is non-empty and does not end in `\n`; an empty file has zero lines, so every location in it is `past_eof`.
  `fileRecord` gains a `lines int` field computed in `buildRecord` from the bytes it already read, for every record that read bytes.
- Rationale: matches how editors and `go vet` number lines; no second read.

### Location spelling

- Decision: one target string is parsed with the lazy pattern `^(.+?):(\d+)(?:-(\d+)|:(\d+))?$`, which takes the shortest path that leaves a valid suffix: `a.go:12:5` is path `a.go`, line 12, column 5 — never path `a.go:12`, line 5.
  `path:N` is Start = End = N; `path:N-M` is a range; `path:N:C` is line N with column C ignored.
  Line numbers parse with `strconv.Atoi`; overflow is `bad_location`.
  The column only has to match `\d+` and is never converted, so an overflowing column is accepted and ignored like any other column.
  The parse is a pure, unexported facade function in `quarry/enclose.go`.
- Rationale: compiler output prints `file:line:col`; Windows-style absolute paths contain `:` before the line suffix.
- Rejected: a structured `[]Location` public input (the target echo must be the caller's spelling, and both callers start from strings).

### Path normalisation

- Decision: the facade normalises each path with `repopath.RepoRelTarget(root, root, path)` — relative paths are repo-root-relative, absolute paths inside the root are accepted, `./` and redundant segments are cleaned.
  `ErrTargetOutsideRepo` maps to `outside_root`; `ErrTargetHasSeparator` maps to `unaddressable`.
  A path normalising to `.` (the root itself) is `unaddressable` by check 4: `glyph.Self(glyph.Go, ".")` rejects the dot segment, and check 4 runs before the extension check.
  `internal/repopath/target.go` switches its import from `quarry` to `internal/engine` for the two sentinels, including the doc comments that name `quarry.ErrTargetOutsideRepo`/`quarry.ErrTargetHasSeparator`; its internal test `target_test.go` switches with it (see Testing).
  The values are identical, so every `errors.Is` caller in the CLI and MCP server is unaffected.
  Doc comments that enumerate callers are updated in the same edit: `repopath`'s package doc, `repoRelTarget`'s doc ("toc and delta are the only two verbs", "an explicit error for both") and `RepoRelTarget`'s caller list in `target.go`, and `quarry.Open`'s doc in `quarry/repo.go`, which names `DeltaGit` as the only git exception (`EncloseAt` is a second).
  The same applies to `engine.Status`'s doc in `internal/engine/answer.go` ("shared by ResolveResult's Status and Unit keys and by ExpandAnswer's") and the `quarry.Status` alias doc in `quarry/quarry.go` ("both ResolveResult and ExpandAnswer draw from"), which `EncloseResult.Status` makes stale.
  Rewrite them to name the subsystem rather than list every caller, so the next caller does not stale them again.
  After the change, nothing under `internal/repopath` — code or tests — imports `quarry`.
- Rationale: requirement 3 says reuse `internal/repopath`; today `repopath` imports `quarry`, so the facade importing it would cycle.
  Comparison is lexical against the root as opened; symlinked aliases of the root are not resolved (same as `toc`).
- Rejected: CLI-only normalisation (Go API callers would lose absolute-path support); new path arithmetic in the facade.

### Engine architecture: one memo, two byte sources

- Decision:
  - `fileMemo` gains a byte-source function `read func(rel string) ([]byte, error)`; `newFileMemo` defaults it to reading under the root (today's `os.ReadFile(filepath.Join(r.absDir(dirRel), base))`), and `buildRecord` reads through it instead of calling `os.ReadFile`.
    Every existing caller is unchanged in behaviour.
  - The engine exports `Location{Target, File string; Start, End int; Reason, Error string}`: the facade passes one per input, either normalised (File/Start/End set) or pre-rejected (Reason/Error set).
    The facade decides checks 1–4 (`bad_location`, `bad_range`, `outside_root`, path-half `unaddressable`); the engine passes pre-rejected locations through unchanged and decides checks 5–9.
  - `(*engine.Repo).Enclose(locs []Location) ([]EncloseResult, error)` answers against the working tree;
    `(*engine.Repo).EncloseFrom(files RevisionFiles, locs []Location) ([]EncloseResult, error)` answers against a revision.
    `RevisionFiles` is an engine interface with two methods — `GoFiles(dirRel string) ([]string, error)` (immediate Go children, repo-relative) and `Read(rel string) ([]byte, error)` — so the engine stays ignorant of git.
  - Both build one memo with `allSymbols` true, so every file built for a directory vote already carries symbols and no file is rebuilt when a second location in the same directory needs its symbols.
  - Working tree, per distinct file: `os.Lstat` decides first — not-exist, ENOTDIR (a file used as a directory segment) or a directory is `missing_file`; a symlink, or any other `Lstat` error (e.g. EACCES on a parent directory), is `unreadable`; then the file's answer comes from the memo-aware `r.toc(file, TOCOptions{Symbols: &on}, m)` file-target path, which already owns the ignore chain, the explicitly-named-gitignored-target rule, the clause vote and unit stamping.
    Any error `r.toc` returns for one item becomes that item's rejection, never the call's: `errors.Is(err, ErrTargetNotFound)` maps to `missing_file`, every other error (`.gitignore` read, `os.ReadDir`, and `fileTargetAnswer`'s "no longer exists in directory" race, a bare `fmt.Errorf` with no sentinel) to `unreadable`.
    `toc.go` is not changed to wrap a sentinel for the race: that would also move `toc`'s own CLI exit code for it, and a file vanishing mid-call is rare enough that `unreadable` is an honest answer.
    The working-tree `Enclose` therefore has no per-file whole-call error; its only whole-call failure is the engine's own setup (none today beyond what `newFileMemo` needs).
    Its `FileEntry.Error` means `unreadable`; `FileEntry.Lossy` is the `lossy` flag.
    The unit is `unitFor(dir, answer.Package, rec.clause)`, where `answer` is the `DirAnswer` this same `r.toc` call returned (its `Package` is that call's `v.pkg`, empty when the directory has none) and `rec` is the target's memo record.
    It is never read from `m.votes`: for an explicitly named gitignored target `fileTargetAnswer` votes with `memoise` false, so `m.votes` either lacks the directory or holds the plain vote from another location, which can differ from the vote `toc` stamped `FileEntry.Symbols` with.
    The unit is gated by `unitSpellable` before anything else reads the symbols — an unspellable unit is `unaddressable`, never read as a nil `FileEntry.Symbols` meaning not_found.
    For a spellable unit, `FileEntry.Symbols` (stamped with the unit, `File` then set to the repo-relative path) is the candidate list; the line count is read from the memo record (a memo hit, not a rebuild).
  - Revision, per distinct directory: `files.GoFiles(dir)` gives the vote set; a target base absent from it is `missing_file`.
    Records are built through the memo with `read` = `files.Read`; the vote is `m.dirVote(dir, recs, true)` (the same `UnitsForClauseMap` vote `revisionClauseMap` reaches, and the record clause equals `PackageClause` by that function's own contract).
    A record with `err` set is `unreadable`; otherwise the unit is `unitFor(dir, v.pkg, rec.clause)`, gated by `unitSpellable` exactly as on the working-tree path (`unaddressable` on failure), and only then are the symbols stamped: `stampSymbols(rec.symbols, unit, file)`.
    A `GoFiles` or `Read` error fails the whole call, matching `revisionClauseMap` (git listed the name, so a read failure means the call is broken); with the unknown-rev check these are the only whole-call failures of the revision path.
    `buildRecord` turns every read failure into `rec.err`, so the `Read` error cannot escape through it: `EncloseFrom` installs as the memo's `read` a closure over `files.Read` that captures the first error it sees before returning it, and after building each directory's records `EncloseFrom` checks the captured error and fails the call with it.
    Without that check a failed blob read would surface as an `unreadable` target, or silently drop a sibling's vote and shift the unit.
    The working-tree default `read` captures nothing; there a read failure stays `rec.err` and the item is `unreadable`.
  - Known asymmetries between the two paths, all shared with `DeltaGit`'s `revisionClauseMap` and all rare, named in `EncloseAt`'s doc comment so the consumer can recognise them, and accepted as is:
    1. No ignore set applies at a revision: a tracked-and-gitignored `.go` file votes at the rev but not in the working tree, so if its deviating clause tips the vote, the unit minted at the rev differs from the working-tree unit and the glyph will not `Resolve` there.
    2. A symlink committed under a `.go` name yields its link text at the rev, which parses as garbage and answers like any other bytes, while the same symlink in the working tree is `unreadable`.
    3. `DirFilesAtRevision` filters `ls-tree` names by suffix only: a directory (tree) named `x.go` is read as its tree listing and answers as garbage where the working tree says `missing_file`, and a submodule (gitlink) named `x.go` makes `ReadBlob` fail and fails the whole call for every location in that directory.
  - Distinct files and directories are processed once per call; results are assembled positionally and checked by `verifyEncloseCoverage` (panics on a length or `Target` mismatch, `verifyResolveCoverage`'s pattern).
  - Known working-tree limits, named in `Enclose`'s doc comment beside the revision asymmetries: Enclose mints exactly the glyphs `toc`/`glyphs` mint for the file, and `Resolve` does not find three of those cases, so a `found` answer there will not `Resolve`:
    1. An explicitly named gitignored file: `symbolsOfDir` drops ignored entries, and the unit may come from a vote that includes the file, which `Resolve` never computes.
    2. A parsed file with no package clause: `fileEntry` stamps its symbols, `symbolsOfDir` skips `!rec.hasClause`.
    3. A path through a symlinked directory: `toc` answers it, `unitDirs`' `dirExists` (`Lstat`) does not.
    These are the existing `toc`/`glyphs` versus `Resolve` gaps, inherited rather than introduced; rejecting them would mean restating `Resolve`'s filters inside Enclose, a second implementation of one rule.
    All three are rare in tool output (a compiler error in a gitignored file, a file with no clause, a symlinked package path).
- Rationale: requirement 9 (one parse per file per call, memo reads through a byte source, nothing outlives the call); reusing `toc` keeps working-tree glyphs identical to what `toc`/`glyphs` mint, which `Resolve` finds in every case except the three limits above.
- Rejected: a Delta-style pure core where the facade reads all bytes (would duplicate the walk's ignore and vote rules for the working tree); calling `revisionClauseMap` (a second parse per file via `PackageClause`, breaking the one-build count).

### Facade surface

- Decision: in a new `quarry/enclose.go`:
  - `func (r *Repo) Enclose(targets []string) ([]EncloseResult, error)` — working tree, equal to `EncloseAt("", targets)`.
  - `func (r *Repo) EncloseAt(rev string, targets []string) ([]EncloseResult, error)` — empty rev means working tree and needs no git; a non-empty rev opens `gitsrc.Open(r.root)`, calls `VerifyRevision(rev)` and returns its error unchanged (unknown rev, not a repository, root not top level fail the whole call), then passes a small unexported `gitRevisionFiles{gr, rev}` adapter (`GoFiles` → `DirFilesAtRevision`, `Read` → `ReadBlob`) to `EncloseFrom`.
  - Aliases in `quarry/quarry.go`: `EncloseResult`, the eight reason constants, and `EncloseReasons` as the engine's own slice.
  - `RenderEncloseJSON(results []EncloseResult) ([]byte, error)` in `quarry/render.go`, through the existing `renderJSON` (two-space indent, no HTML escaping, one trailing newline); a nil or empty slice renders `[]`.
  A nil `targets` yields an empty, non-nil slice.
- Rationale: requirement 10 and the acceptance line "public facade method(s) plus a JSON renderer"; the git layer stays in the facade as with `DeltaGit`.

### CLI verb

- Decision: `quarry enclose [--rev <rev>] [--root <path>] (<location>... | --stdin)`.
  - Positional args are the targets, in order; `--stdin` reads one location per line from standard input (trailing `\r` and surrounding whitespace trimmed, blank lines skipped).
    Giving both, or neither, is a usage error (exit 2).
    Stdin is read whole with `io.ReadAll` and split on `\n`, so there is no line-length limit (no `bufio.Scanner` buffer cap); a read error goes through `fail` as `exitInternal` (3) with an `internal error:` message, like every other I/O failure.
    `--stdin` with empty input answers `[]` with exit 0.
  - `--rev` is valid for `enclose` only; `--stdin` for `enclose` only; `--text` is a usage error for `enclose` (no text view exists).
    `--root` works as for the other repository verbs.
    `--rev` with an empty value (`--rev ""` or `--rev=`) is a usage error, as it is for `--from`/`--to`; it would otherwise reach `EncloseAt("")` and silently answer against the working tree.
  - Relative location paths resolve against the repository root, not the working directory — unlike `toc` and `delta`, whose pipelines pass `base = cwd` to `repopath.RepoRelTarget`.
    `runEnclose` must not copy that step: tool output is repo-relative, and the consumer joins bare names with the package path itself.
    The usage text says so on the `enclose` line.
  - Output: `RenderEncloseJSON` of the whole batch on stdout.
  - Exit codes: 0 whenever the batch was answered, whatever the per-item statuses (a batch has no single negative answer, like `delta`); a whole-call failure goes through `fail` with the `codeForDeltaError` mapping (unknown revision, not a repository, root not top level → 2; anything else → 3).
    The exit-2 sentences mirror `runDelta`'s, built from the typed errors' fields (`errors.As`) rather than the wrapped chain: `enclose: unknown revision <rev>`, `enclose: root <root> is not the repository top level (top level is <toplevel>)`, `enclose: root is not a git repository: <root>`, each with usage on stderr as `runDelta` does; anything else is `internal error: <err>` with exit 3.
  - `Run(args, stdout, stderr)` keeps its signature and delegates to a new unexported `run(args, stdin, stdout, stderr)` with `os.Stdin`; tests call `run` with a `strings.Reader`.
    `main.go` is unchanged.
  - Usage text gains the `enclose` line and the two flags; the verb gate and its error messages list `enclose`.
- Rationale: requirement 10 and agreed decision 8; the stdin seam avoids touching ~80 existing `Run` test call sites.
- Rejected: exit 1 when any item is not_found (a pack builder would treat a normal batch as failure); JSON-lines output (every other verb emits one JSON document).

## Technical context

- `internal/engine/memo.go`: `fileRecord`, `fileMemo` (`records`, `votes`, `builds` test seam), `buildRecord`, `record`, `dirRecords`, `dirVote`, `stampSymbols`. The byte-source seam and `lines` field land here.
- `internal/engine/toc.go`: `toc` (memo-aware), `fileTargetAnswer` (ignore chain, gitignored-target rule, symlink as name-only entry).
- `internal/engine/walk.go`: `fileEntry` (stamps the unit, leaves `Symbols` nil for an unspellable unit), `unitFor`, `unitSpellable`, `joinRel`, `absDir`.
- `internal/engine/units.go`: `UnitsForClauseMap`, `PackageClause` (documents equivalence with the record clause).
- `internal/engine/resolve.go`: `Resolve`/`resolve`/`verifyResolveCoverage` — the positional and verifier pattern to mirror; `unitMemo` shows how a test reads `files.builds`.
- `internal/engine/answer.go`: `Status`, `Symbol`, `ResolveResult` and `Rejected()` — doc-comment style to follow for `EncloseResult`. The header says the emitted key set is closed per a Shared Decision; `EncloseResult` is a new type, so existing key sets are untouched. Put `EncloseResult` in a new `internal/engine/enclose.go` (or `answer.go`; plan's choice) with the verb.
- `internal/engine/name.go`: `NameReasons` — the enumerating-slice pattern for `EncloseReasons`.
- `quarry/delta.go`: `DeltaGit` (gitsrc open + `VerifyRevision`), `revisionClauseMap`.
- `quarry/quarry.go`: alias conventions, including slices aliased as the engine's own value (`quarry_test.go` checks `&NameReasons[0] == &engine.NameReasons[0]`; do the same for `EncloseReasons`).
- `quarry/render.go`: `renderJSON`, `RenderResolveJSON`.
- `internal/gitsrc/gitsrc.go`: `Open`, `VerifyRevision`, `ReadBlob` (a missing path is a generic git error, hence existence via `DirFilesAtRevision`), `DirFilesAtRevision` (immediate Go children, no ignore set).
- `internal/repopath/target.go`: `RepoRelTarget(root, base, target)`; currently imports `quarry` for `ErrTargetOutsideRepo`/`ErrTargetHasSeparator`.
- `internal/cli/cli.go`: `Run`, `fail`, `codeForDeltaError`, per-verb `runX` pipelines; `internal/cli/flags.go`: `parseArgs`, verb gate, per-verb flag scoping (every verb today takes exactly one target — `enclose` takes many); `internal/cli/usage.go`: `usageText`.
- `glyph`: `glyph.Self(lang, path)` builds a self glyph and fails on an unspellable path.
- Code style: dense explanatory doc comments in the surrounding files; match them. Go only; no Python.

## Constraints

- No-cache rule: nothing built by an exported call outlives it (memo is a local of the call).
- One build per file per call, on both the working-tree and the revision path.
- No change to `Statuses`, `Known()`, `ResolveResult`'s key set, docs/glyph.md §5.
- Engine stays git-ignorant; git access lives in the facade.
- `go test ./...` and `golangci-lint run` green.

## Testing

TDD candidates: the location-spelling parser, the innermost selection, the line-count function, and the rejection-order table — all pure.

- **Engine vocabulary:** an `EncloseReasons` completeness test mirroring `TestName_ReasonCompleteness` — exactly the eight constants, each once.
- **Facade parser (table test):** `path:N`, `path:N-M`, `path:N:C` (explicitly `a.go:12:5` → `a.go`, 12, 12), an overflowing column (accepted, ignored), a path containing `:` (e.g. `C:\x\a.go:12`), empty path, missing line, non-numeric, overflow, `-` without an end.
- **Facade checks 1–4 (scratch tree):** every `bad_location`, `bad_range`, `outside_root` and path-half `unaddressable` case — a `.:1` location, a path segment with a space, a `#` segment (no `file` on the answer).
- **Engine, working tree (scratch tree fixture):** single line; multi-line range spanning two top-level members (found, two symbols, source order); a line in an interface method; a range over an interface head plus a method (both); a line inside a closure; a line inside a doc comment; a `const (` line (not_found + `unit`); an import block line (not_found + `unit` = `<file>#`); a lossy file (`lossy: true`, not rejected); a range ending past EOF; an empty file; a `_test.go` member (unit is the `_test` unit for an external test package); a gitignored file named explicitly whose inclusion changes the directory's vote, located both alone and after another location in the same directory (unit from that call's `DirAnswer.Package`); a one-line interface (type and method both returned); a path through a file used as a directory segment (`a.go/b.go:1`, `missing_file`); every engine-decided reason (`unsupported_language`, `missing_file`, `unreadable` — symlink, invalid UTF-8 — and `past_eof`); a pre-rejected `Location` passed through unchanged; a root-level `.go` file (unit-half `unaddressable`, no `unit`, never not_found — the only way to reach check 8, since check 4 already rejects any unspellable directory segment); an unreadable directory or `.gitignore` beside a good location in the same batch (the bad item is rejected, the good one answered, no whole-call error); a nil and an empty target slice; duplicate targets answered twice.
- **Build count:** N locations in one file → `builds[file] == 1`; locations in two files of one directory → each built once; the same at a revision.
  Needs an unexported worker taking the memo (the `resolve`/`unitMemo` pattern) so the test can read `builds`.
- **Coverage verifier:** panics on length mismatch and on a `Target` mismatch.
- **Revision (git fixture repo, as `quarry/delta_test.go` builds one):** `--rev` mapping where the working tree has shifted the member's lines (rev answer has the old lines and the right glyph; working-tree answer differs); a file absent at the rev → `missing_file`; unknown rev → whole-call error matching `ErrUnknownRevision`; a `RevisionFiles` fake whose `Read` fails → whole-call error, not an `unreadable` item; a directory whose dominant clause differs at the rev → unit from the rev's vote; a root-level `.go` file at the rev → `unaddressable`, matching the working-tree answer.
- **Facade:** absolute path inside the root; path outside the root; `./a/../a/b.go` normalised; `EncloseReasons` is the engine's own slice; `Enclose` equals `EncloseAt("")`.
- **repopath:** `internal/repopath/target_test.go` moves from `quarry.ErrTargetOutsideRepo`/`quarry.ErrTargetHasSeparator` to the `engine` sentinels. It is an internal test (`package repopath`), so keeping its `quarry` import would make `go test ./internal/repopath` fail with an import cycle once `quarry/enclose.go` imports `repopath`.
- **CLI:** args form; `--stdin` (including blank lines and `\r\n`); both/neither is exit 2; `--text` with `enclose` is exit 2; `--rev` on another verb is exit 2; an empty `--rev` value is exit 2; a bare `go test` file name → `missing_file` with exit 0; `path:line:col` input; a run from a subdirectory with a repo-relative path (resolved against the root, not the cwd); unknown `--rev` → exit 2 with the error envelope; golden JSON under `internal/cli/testdata/enclose/` against a scratch fixture tree (the `name_golden_test.go` payload-only pattern), not the pinned Loomyard checkout.

## Q&A log

- **Q:** Are the nine agreed decisions open for discussion? **A:** [auto-pick] No — treat them as settled input. **Why:** the task body says so; the operator approved them with the consumer.
- **Q:** The agreed `unit` key holds a glyph string while `ResolveResult.Unit` holds a Status — rename it? **A:** [auto-pick] Keep `unit`, holding the file self glyph `<file>#`, and document the difference at the field. **Why:** the key name is settled with the consumer; the clash is confined to a separate type.
- **Q:** How should the innermost rule treat a range over an interface head plus a method? **A:** [auto-pick] Line-coverage form: a parent survives when the range touches a line of it that no overlapping child covers. **Why:** satisfies both sentences of agreed decision 2, including its head-plus-method example.
- **Q:** What answers a malformed location string or an existing-but-unreadable file? **A:** [auto-pick] Two added reasons, `bad_location` and `unreadable`. **Why:** neither may fail the call, and no existing reason describes them truthfully.
- **Q:** Where does path normalisation live, given `repopath` imports `quarry`? **A:** [auto-pick] In the facade via `repopath.RepoRelTarget`, after re-pointing `repopath`'s sentinel import to `engine`. **Why:** requirement 3 says reuse `repopath`; the sentinel values are identical, so no caller changes.
- **Q:** How does the revision path keep one build per file? **A:** [auto-pick] A byte-source function on `fileMemo` plus an engine `RevisionFiles` interface the facade implements over gitsrc; the vote reads memo records instead of calling `revisionClauseMap`. **Why:** requirement 9; `revisionClauseMap` parses each file a second time.
- **Q:** Working-tree path: reuse `toc`'s file-target answer or write a new reader? **A:** [auto-pick] Reuse memo-aware `r.toc`. **Why:** inherits the ignore, gitignored-target and vote rules, so the glyphs equal what `glyphs`/`Resolve` see.
- **Q:** CLI exit code when some items are not_found or rejected? **A:** [auto-pick] 0 whenever the batch was answered; whole-call failures use the `delta` mapping. **Why:** a batch has no single negative answer; per-item status lives in the JSON.
- **Q:** How does the CLI read stdin without churning every `Run` test call? **A:** [auto-pick] `Run` delegates to an unexported `run(args, stdin, stdout, stderr)`. **Why:** no signature change for `main.go` or existing tests.
- **Q:** Support `--text` for `enclose`? **A:** [auto-pick] No; `--text` is a usage error for this verb. **Why:** the requirement asks for JSON only.
- **Q:** (review r1) What answers a location in a repository-root file, whose unit `unitFor(".")` is the unspellable `""`? **A:** [auto-pick] A new rejection reason `unaddressable`, gated by `unitSpellable` identically on both paths before stamping; the path-unspellable case moves from `bad_location` to it. **Why:** a silent not_found would lie about a line inside a real function, and stamping would mint IDs `glyph.Parse` rejects.
- **Q:** (review r1) Can an `r.toc` error for one file fail the working-tree call? **A:** [auto-pick] No — not-found/vanished maps to `missing_file`, every other `r.toc` error to `unreadable`. **Why:** requirement 1, one bad item never fails the call.
- **Q:** (review r2) Where does the working-tree path read the directory's vote for the unit? **A:** [auto-pick] From the `DirAnswer.Package` the same `r.toc` call returned, never `m.votes`. **Why:** an explicitly named gitignored target votes unmemoised, so `m.votes` can hold a different vote than the one `toc` stamped with.
- **Q:** (review r2) The vanished-target race in `fileTargetAnswer` has no sentinel — wrap one or accept `unreadable`? **A:** [auto-pick] Accept `unreadable`; leave `toc.go` unchanged. **Why:** wrapping would also change `toc`'s CLI exit code for the race, and the race is rare.
- **Q:** (review r3) What does an `os.Lstat` error other than not-exist answer? **A:** [auto-pick] ENOTDIR is `missing_file`; any other `Lstat` error is `unreadable`. **Why:** no per-file whole-call error on the working-tree path.
- **Q:** (review r3) What does a location on a one-line interface (type and method share a span) return? **A:** [auto-pick] Both, type first. **Why:** follows from the containment rule; no special case.
- **Q:** (review r4) Working-tree `found` answers `Resolve` cannot find (explicit gitignored file, no package clause, symlinked directory) — reject or document? **A:** [auto-pick] Document as named limits in `Enclose`'s doc comment. **Why:** they are the existing `toc`/`glyphs` versus `Resolve` gaps; rejecting would restate `Resolve`'s filters in a second place.
- **Q:** (review r4) How does `--stdin` handle a read error or a very long line? **A:** [auto-pick] `io.ReadAll` (no line cap); a read error is exit 3 via `fail`. **Why:** matches every other I/O failure in the CLI.
