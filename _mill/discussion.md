# Discussion: Resolve self-target path: per-call dirPackage memo (GH #34)

```yaml
task: 'Resolve self-target path: per-call dirPackage memo (GH #34)'
slug: resolve-self-dirpackage-memo
status: discussing
parent_branch: main
```

## Problem

One `Repo.Resolve` call over 311 glyph targets from Loomyard's `lyx webster validate` takes 64–74 s at 100% of one core (GH #34, Knatte18/loomyard#345).
Most targets are self glyphs (`<file>#`) in one directory with about 240 files (`internal/fabricengine`).

Each self target goes `resolveSelfTarget` → `r.TOC` → `fileTargetAnswer`, which makes two whole-directory parse passes: `dirPackage` for the clause vote, then `fileEntry` over every file for header, doc and flags.
That is 2 × N tree-sitter parses per target, so the call does about 311 × 2 × 240 parses.
`unitMemo` already deduplicates member glyphs per unit, but the self path bypasses it.
The member path is not single-parse either: `symbolsOfDir` runs `dirPackage` and then its own symbol parse, so each file is parsed twice.

Why now: Loomyard's plan validator calls Resolve over every glyph in a plan, and a 74 s validation blocks its workflow.
Loomyard (ly:orch) ordered the fix and is waiting for v0.2.1.

## Scope

**In:**

- A per-call parse memo on `unitMemo` that guarantees every file is parsed **at most once per Resolve call**, however many targets point into its directory and whatever mix of self and member glyphs they are.
- One tree-sitter parse per file extracts everything any path needs: package clause, header, test flag, generated flag, lossy flag, package doc, and symbols.
- The self-target path (file targets and directory targets) and the member path (`symbolsOfUnit` / `symbolsOfDir`) both read from that memo.
- Expand gets the same guarantee as a side effect, since it reads symbols through `unitMemo.symbolsOf`.
- A per-file parse counter seam and the tests that pin the guarantee.
- Comment updates that keep the docs true after the refactor:
  - `walk.go`'s header: replace the "two parse passes" explanation with single-parse extraction plus unit stamping after the vote.
  - `fileTargetAnswer`'s doc comment in `toc.go`: it says "A gitignored file still does not vote in the package tie-break", but the code keeps an explicitly named gitignored target in the entries `dirPackage` votes over. Correct the comment to say the named target votes, matching the code and the two-level memo decision below.
  - `unitMemo.parses`: once `symbolsOf` reads per-file records, this counter counts unit extractions requested, not parses. Restate its doc comment and `TestResolve_ParsesEachUnitOnce`'s comment that way, keeping the name or renaming it as the plan chooses. The new per-file counter is the only parse count.

**Out:**

- No persistent cache: nothing is stored on `Repo`, and the memo dies with the call. The engine's no-cache rule stands.
- No eager whole-repository parse. Cost must scale with the directories a call touches, not with repository size.
- No change to any exported signature, `TOCOptions`, or any answer's JSON shape. This is v0.2.1, pure performance.
- The issue's "clause without full parse" idea: unnecessary once each file is parsed once.
- Concurrency or parallel parsing.
- `SpansOf` keeps its current behaviour. Whether it builds its own throwaway memo is the plan's choice.

## Decisions

### Lazy, per-directory, single-parse extraction

- Decision: the first time a call touches a file, it is read and parsed once, and a per-file record is extracted from that one parse: clause, header (pre-truncation input to `FirstParagraph`), test, generated, lossy, package doc, symbols, or the read, UTF-8 or parse error that `fileEntry` would report today. Every later need for that file within the call reads the record.
- Rationale: this is the acceptance criterion as written, and it covers both hot spots in the SIGQUIT dumps.
- Rejected: memoising `dirPackage` only, which roughly halves the cost and fails the criterion. Also rejected: parsing the whole repository once per call, where cost scales with repo size. This repo's 469 files take 616 ms (see the `walk.go` header), so even a one-target call would pay that, which breaks `TestResolve_TwentyGlyphsUnder150ms`.

### Unit stamping after the vote

- Decision: symbols are extracted in the same parse with no unit, and `Glyph.Unit` and `ID` (`= Glyph.String()`) are stamped when a consumer reads them, once the directory's clause vote has fixed the unit through `unitFor`. The stamp goes on a copy, never into the memoised record.
- Rationale: the unit is a directory-level fact, so it is not known until every clause in the directory is read. That is why `walk.go` needs two passes today. `goStrategy.Symbols` uses `unit` only to build `Glyph` and `ID` (see `goDeclSymbol` and its siblings in `golang.go`), so stamping afterwards gives byte-identical symbols. The unspellable-unit rule (`unitSpellable`) is applied at stamp time, exactly where `fileEntry` applies it now.
- Rejected: changing the `Strategy` interface. `Symbols(unit, ...)` keeps its signature, and the memo passes a placeholder unit and restamps afterwards.

### Two-level memo: files by path, votes by directory

- Decision: the memo holds per-file records keyed by repository-relative file path, which is where the parse counter lives, and the directory vote (`dirPackage`'s result) keyed by `dirRel`. The vote is computed from the file records without parsing.
- Rationale: an explicitly named gitignored file target joins the vote set (`fileTargetAnswer` keeps it through its `!isTarget && ig.match` check), so its directory's vote can differ from the plain one. Keying file records by path keeps that case at one parse per file. The plan decides whether the vote for that rare set is recomputed per target or memoised under a second key; it costs no parse either way.
- Rejected: a single record keyed only by `dirRel`. It is wrong for the gitignored-target case.

### Threading through TOC without changing its contract

- Decision: unexported memo-aware variants of `TOC`, `fileTargetAnswer` and `walkDir` take the memo. Exported `TOC` builds its own fresh memo, so TOC also becomes single-parse and there is one extraction code path instead of two. `resolveSelfTarget` takes `unitMemo` and calls the memo-aware variant.
- Rationale: this matches `unitMemo`'s existing shape (`resolve(targets, m)`). With one code path, no second extraction can drift from the first. The committed goldens, including Loomyard's, pin TOC's output byte for byte, which guards the refactor.
- Rejected: plumbing the memo through `TOCOptions`, which changes the exported surface. Also rejected: a memo-only path beside the untouched `fileEntry`/`dirPackage` path, which leaves two implementations of the same extraction.
- Symbol extraction cost: a record built for a TOC call extracts symbols only when that call wants them. A record built inside Resolve or Expand always extracts them, since a later member glyph in the same call may need them and a re-parse is not allowed.

### Directory self targets are in scope

- Decision: a directory self glyph (`internal/x#`) goes through the same memo. `walkDir` at depth 0 with symbols off reads its own files' records and each subdirectory's identity (package and doc) from records.
- Rationale: the criterion is per call, not per target kind, so a call mixing `internal/x#` with file targets in `internal/x` must not parse `internal/x` twice.

### Test seam

- Decision: the memo has a per-file parse counter, incremented before the parse, like `unitMemo.parses` (calls made, not calls that succeeded). Tests assert that the per-file parse count is at most 1 for every file and that the total equals the number of distinct files touched.
- Rationale: wall-clock is the thing the guarantee's test must not depend on.

## Technical context

- `internal/engine/resolve.go`: `unitMemo` (fields `symbols`, `dirs`, `parses`; `newUnitMemo`, `symbolsOf`, `dirsOf`), `resolveGlyphTarget`, `resolveSelfTarget`, `resolve`, `symbolsOfUnit`, `symbolsOfDir`. Expand rejects self glyphs (`expand.go`, `g.IsSelf()` → `*SelfGlyphError`), so the self path is Resolve-only.
- `internal/engine/toc.go`: `TOC`, `fileTargetAnswer`, `ancestorChain`, `splitDirBase`.
- `internal/engine/walk.go`: `dirPackage`, `fileEntry` (the single-parse extraction today minus the clause), `walkDir`, `dirDoc` (selects among already-extracted docs, no parse), `unitFor`, `unitSpellable`. The header comment explains why there are two passes; this task replaces that explanation with unit stamping after the vote.
- `internal/engine/units.go`: exported `PackageClause`, `UnitsForClauseMap`, `ClauseMapForFiles`, which have callers outside the package. The vote rule stays in `UnitsForClauseMap`; the memo calls it over clauses taken from records. `PackageClause` keeps its exported contract.
- `internal/engine/golang.go`: `goDeclSymbol` and its siblings build `glyph.Glyph{Unit: unit, ...}` and `ID: g.String()`. That is the only use of `unit`.
- Ignore-set filtering: TOC builds a fresh `ignoreSet` per target and extends it along `ancestorChain`, and `symbolsOfUnit` extends the memo's set along `dirChainBelowRoot`. Both produce the same filter for a given directory. The memo must not change which files are filtered or vote.
- Symlinks are never parsed and never vote; a symlink target is answered name-only.
- Per-file error semantics must survive: `fileEntry` returns `Error` for a read failure, invalid UTF-8, or a `WithTree` error, and in each of those cases `dirPackage` records no clause.

## Constraints

- Go only; no Python (repo `CLAUDE.md`).
- Engine no-cache rule: the memo is a local of the exported entry point and dies with it.
- No exported API or JSON-shape change, because the release is v0.2.1.
- Gates: `go test ./...` and `golangci-lint run`.

## Testing

- **TDD candidate, parse-once guarantee:** use `r.resolve(targets, m)` with a constructed memo. Name several self file targets in one directory, a directory self target for that same directory, member glyphs in that directory, and a target in a second directory. Assert at most one parse per file and a total equal to the distinct files touched.
- **Equivalence:** for every self target in such a call, the memoised `Listing` must equal `r.TOC(unit, TOCOptions{Symbols: &false})` from a fresh call. Member results must equal today's results.
- **Gitignored explicit target:** an `openScratchRepo` fixture with a gitignored file named explicitly next to normal targets in the same directory. Its answer must equal its stand-alone answer, and the per-file parse count must stay at most 1.
- **Existing suites unchanged:** TOC goldens, Loomyard goldens and round trip, `TestResolve_ParsesEachUnitOnce`, and `TestResolve_TwentyGlyphsUnder150ms`.
- **Before/after measurement (manual, recorded in the handoff):** the Loomyard repro below.

## Verification against Loomyard

Target mix, from ly:orch: of quarry34-plan's 549 card entries, 526 are plain file paths, 23 are member glyphs and 1 is a directory. After Loomyard's planglyph filter, 311 targets reach Resolve, almost all of them self file targets.
The densest directories are `internal/fabricengine` (89 entries), then `reedengine` (23), `loomcli` (17), and `websterengine` and `gitrepo` (15 each).
Most member glyphs sit in those same directories (`fabricengine#CloneHub`, `gitkit#GitStatusPorcelain`, `battencli#battenCLI.wire`), so the shared self/member memo is load-bearing.

Quarry runs the measurement. Procedure, agreed with ly:orch:

1. Build `lyx` against this branch using a scratch `GOWORK` file outside the Loomyard prime, which lists the prime's module and this worktree. Write the binary into this worktree's `.scratch/`. Never install it on `PATH` or in the Go bin dir, and never modify any file in the prime.
2. With cwd set to the Loomyard prime (`/home/knatte/Code/loomyard-LYXHUB/loomyard`; `validate` is read-only), run `time <that-binary> webster validate --plan-dir /home/knatte/Code/loomyard-LYXHUB/loomyard/.scratch/hub/quarry34-plan`.
3. Capture the findings JSON from the same command run with the v0.2.0 `lyx`, and diff the two. They must be byte-identical.

Baseline: 74 s with quarry v0.2.0.
**Acceptance: ≤ 5 s wall** on that command (ly:orch expects 1–2 s), and a byte-identical findings diff.
Record both timings and the diff result in the task's handoff.

Release: v0.2.1, cut on the operator's word after merge.
When the tag is cut, send ly:orch the version, the before and after wall times, and the findings-diff result. ly:orch takes the tag into Loomyard itself. No changelog line is needed.

## Q&A log

- **Q:** Memoise `dirPackage` only, or both passes? **A:** (operator) Both. Every file is parsed at most once per Resolve call; halving is not enough.
- **Q:** Parse the whole repository once per call, or lazily per directory? **A:** (operator) Lazily, one directory at a time, with one memo shared by self and member glyphs. A whole-repo parse makes cost scale with repo size.
- **Q:** Thread the memo through `TOCOptions` or an unexported variant? **A:** [auto-pick] Unexported memo-aware variant. **Why:** it keeps the exported contract and matches `unitMemo`'s shape.
- **Q:** Directory self targets in scope? **A:** [auto-pick] Yes. **Why:** the criterion is per call, not per target kind.
- **Q:** Counter granularity? **A:** (operator, via the wiki task) Per file, not per directory pass.
- **Q:** Does the draft cover what Loomyard ordered? **A:** (ly:orch) Yes. The exported TOC and Resolve contracts and the JSON shape must stay byte-identical, because Loomyard's goldens pin them.
- **Q:** Who measures, and what is the threshold? **A:** (ly:orch) Quarry measures with a branch-built `lyx` held in `.scratch`. The threshold is ≤ 5 s wall, with byte-identical findings against v0.2.0.
- **Q:** What does ly:orch need on release? **A:** (ly:orch) A message with the version, before/after wall times and the findings-diff result. No changelog line.
