# Batch: facade-enclose

```yaml
task: 'Enclose: path:line ranges -> enclosing member glyphs (batch, at revision, CLI verb)'
batch: facade-enclose
number: 2
cards: 3
verify: go test ./internal/repopath/ ./quarry/ ./internal/mcpserver/ ./internal/cli/
depends-on: [1]
```

## Batch Scope

This batch delivers the public Go surface in `quarry/`: `(*Repo).Enclose`, `(*Repo).EncloseAt`, the location-spelling parser and rejection checks 1–4, path normalisation through `internal/repopath`, the git-backed `RevisionFiles` adapter, the type/constant/slice aliases and `RenderEncloseJSON`.
It first re-points `internal/repopath`'s two sentinel references from `quarry` to `internal/engine`, which is what lets `quarry/enclose.go` import `repopath` without an import cycle.
The next batch's CLI verb consumes `EncloseAt`, `RenderEncloseJSON` and the aliased git error types it already uses for `delta`.
It is one batch because the repopath change exists only to unblock the facade file, and the aliases and renderer are the facade's own surface.

## Cards

### Card 6: re-point repopath's sentinels to the engine

- **Context:**
  - `_mill/discussion.md`
  - `internal/engine/repo.go`
  - `quarry/quarry.go`
  - `internal/mcpserver/toc.go`
- **Edits:**
  - `internal/repopath/target.go`
  - `internal/repopath/target_test.go`
  - `internal/repopath/doc.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  - In `internal/repopath/target.go`, replace the import of `github.com/Knatte18/quarry/quarry` with `github.com/Knatte18/quarry/internal/engine`, and every `quarry.ErrTargetOutsideRepo` / `quarry.ErrTargetHasSeparator` use and doc-comment mention with `engine.ErrTargetOutsideRepo` / `engine.ErrTargetHasSeparator`.
    The facade's sentinels are these same values (`quarry.go` aliases them), so every `errors.Is` caller in the CLI and the MCP server is unaffected.
  - Rewrite the caller-enumerating doc comments so they name the subsystem rather than list callers: `repoRelTarget`'s sentence "toc and delta are the only two verbs that reach this function's exported form, and a "#" in a path segment is an explicit error for both" becomes a statement that every caller taking a path through `RepoRelTarget` gets the escape and separator rejections; `RepoRelTarget`'s own caller list becomes "for callers outside this package that take a repository path from a user"; `doc.go`'s "It has two callers, the CLI and the MCP server" becomes a statement that its callers format their own user-facing sentences from its sentinels, naming no caller.
  - In `internal/repopath/target_test.go` (package `repopath`), switch every `quarry.` sentinel reference to `engine.` and its import accordingly.
  - After the change, `grep -rn '"github.com/Knatte18/quarry/quarry"' internal/repopath` prints nothing.
- **Commit:** `refactor(repopath): take the target sentinels from the engine`

### Card 7: facade aliases and RenderEncloseJSON

- **Context:**
  - `_mill/discussion.md`
  - `internal/engine/enclose.go`
  - `internal/engine/answer.go`
- **Edits:**
  - `quarry/doc.go`
  - `quarry/quarry.go`
  - `quarry/quarry_test.go`
  - `quarry/render.go`
  - `quarry/render_test.go`
  - `quarry/repo.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  - In `quarry/quarry.go`, add `type EncloseResult = engine.EncloseResult`, a const block aliasing the eight `EncloseReason*` constants under the same names, and `var EncloseReasons = engine.EncloseReasons`, each documented in the file's existing alias style (the slice documented as the engine's own value, as `NameReasons` is).
    Rewrite the doc comment of the `Status` alias so it names the subsystem ("the engine's per-target result types") rather than "both ResolveResult and ExpandAnswer".
    Rewrite the comment above the `StatusFound`… const block, "The four Status values a resolve or expand query ever emits", the same way: it names the per-target queries as a group, not resolve and expand.
  - In `quarry/render.go`, add `func RenderEncloseJSON(results []EncloseResult) ([]byte, error)`: a nil `results` is replaced by an empty slice so the output is `[]\n`, then it returns `renderJSON(results)`.
    Rewrite the file header comment so it no longer counts the renderers ("five of the six successful envelopes"): name the shared `renderJSON` configuration and where each renderer lives without a tally, and rewrite its alias sentence ("DirAnswer, ResolveResult, ExpandAnswer and NameResult are aliases for engine types, and GitDeltaAnswer is a facade type embedding one") to say the answer types are engine aliases or facade types embedding one, without listing them.
  - In `quarry/repo.go`, rewrite `Open`'s doc comment so the git exception names the methods that take a revision (the git-backed convenience methods, `DeltaGit` and `EncloseAt`) rather than `DeltaGit` alone.
  - Tests: in `quarry/quarry_test.go`, a subtest asserting `&EncloseReasons[0] == &engine.EncloseReasons[0]`, beside the existing `NameReasons` one; in `quarry/render_test.go`, `RenderEncloseJSON(nil)` and `RenderEncloseJSON([]EncloseResult{})` both return exactly `[]\n`, and a one-result slice renders with two-space indent and the declared key order.
- **Commit:** `feat(quarry): alias the enclose vocabulary and add RenderEncloseJSON`

### Card 8: Enclose, EncloseAt and the location parser

- **Context:**
  - `_mill/discussion.md`
  - `internal/engine/enclose.go`
  - `internal/engine/repo.go`
  - `internal/repopath/target.go`
  - `internal/gitsrc/gitsrc.go`
  - `glyph/self.go`
  - `glyph/glyph.go`
  - `quarry/quarry.go`
  - `quarry/repo.go`
  - `quarry/delta.go`
  - `quarry/delta_test.go`
  - `quarry/scratchtree_test.go`
- **Edits:** none
- **Creates:**
  - `quarry/enclose.go`
  - `quarry/enclose_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:**
  - Create `quarry/enclose.go` with a file header comment, declaring:
    - an unexported `locationPattern = regexp.MustCompile("^(.+?):(\\d+)(?:-(\\d+)|:(\\d+))?$")` and the pure `parseLocation(target string) (path string, start, end int, ok bool)` implementing the discussion's "Location spelling" decision: `path:N` gives start = end = N; `path:N-M` gives start N, end M; `path:N:C` gives start = end = N with the column matched by `\d+` only and never converted; a line or end that `strconv.Atoi` rejects (overflow) is `ok == false`.
    - an unexported `(r *Repo) locationFor(target string) engine.Location` applying checks 1–4 in order: `bad_location` when `parseLocation` fails; `bad_range` when start < 1 or start > end (carrying start and end); `outside_root` when `repopath.RepoRelTarget(r.root, r.root, path)` returns `ErrTargetOutsideRepo` (carrying start and end); path-half `unaddressable` when it returns `ErrTargetHasSeparator` (no file) or when `glyph.Self(glyph.Go, rel)` fails (file set to `rel`). Any other `RepoRelTarget` error (none exists today) is `outside_root`. A passing target becomes `engine.Location{Target, File: rel, Start, End}`.
      `Error` texts follow the overview's "rejection message texts" Shared Decision.
    - `func (r *Repo) Enclose(targets []string) ([]EncloseResult, error)`, returning `r.EncloseAt("", targets)`.
    - `func (r *Repo) EncloseAt(rev string, targets []string) ([]EncloseResult, error)`: builds one `engine.Location` per target with `locationFor`; for an empty `rev` returns `r.engine.Enclose(locs)` with no git involved; for a non-empty `rev` opens `gitsrc.Open(r.root)`, calls `VerifyRevision(rev)`, returns either error unchanged (as `DeltaGit` does), and returns `r.engine.EncloseFrom(gitRevisionFiles{gr: gr, rev: rev}, locs)`.
      A nil `targets` yields an empty, non-nil slice.
    - an unexported `gitRevisionFiles struct { gr *gitsrc.Repo; rev string }` whose `GoFiles(dirRel)` returns `gr.DirFilesAtRevision(rev, dirRel)` and whose `Read(rel)` returns `gr.ReadBlob(rev, rel)`.
  - Doc comments: `Enclose` states that relative paths are repository-root-relative (never cwd-relative), that absolute paths inside the root are accepted, the positional one-answer-per-target contract, and the three working-tree limits the discussion's "Engine architecture" decision lists (an explicitly named gitignored file, a file with no package clause, a path through a symlinked directory: a `found` there will not `Resolve`).
    `EncloseAt` states the one-revision-per-call rule, the whole-call failures (unknown revision, not a repository, root not top level, a git read failure), and the three revision asymmetries the same decision lists.
  - Tests in `quarry/enclose_test.go`:
    a table test of `parseLocation` covering `path:N`, `path:N-M`, `a.go:12:5` (path `a.go`, 12, 12), an overflowing column (accepted), `C:\x\a.go:12` (path `C:\x\a.go`), an empty path (`:12`), a missing line (`a.go`), a non-numeric line, an overflowing line, and `a.go:12-`;
    checks 1–4 on a scratch tree: each `bad_location` and `bad_range` case, `outside_root` for `../x.go:1` and for an absolute path outside the root, path-half `unaddressable` for `.:1`, for a segment holding a space, and for a `#` segment (no `file` on that answer);
    normalisation: an absolute path inside the root and `./a/../a/b.go:1` both answer with `file` `a/b.go`;
    `Enclose(targets)` equals `EncloseAt("", targets)`;
    revision tests on `newDeltaFixture`: a member whose lines moved in the working tree after the commit answers at the commit's line numbers with the right glyph while the working-tree answer differs; a file absent at the revision is `missing_file`; an unknown revision fails the whole call with `errors.Is(err, ErrUnknownRevision)`; a directory whose dominant clause at the revision differs from the working tree's gives the revision's unit; a root-level `.go` file at the revision is unit-half `unaddressable`, matching the working-tree answer.
- **Commit:** `feat(quarry): add Enclose and EncloseAt`

## Batch Tests

`verify: go test ./internal/repopath/ ./quarry/ ./internal/mcpserver/ ./internal/cli/` covers the three packages whose imports card 6 changes or reaches — `internal/repopath` itself and its two importers, the MCP server and the CLI, which compare `repopath` errors against the facade's sentinels — plus the facade package that cards 7 and 8 change.
New tests: the `EncloseReasons` identity subtest in `quarry_test.go`, the `RenderEncloseJSON` tests in `render_test.go`, and `quarry/enclose_test.go`.
