MILL_REVIEW_BEGIN
# Review: P2 — diff-to-symbols: changed file versions to symbol-table delta (roadmap 2c)

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: claude-opus-5 (Anthropic Claude Opus 5, high reasoning effort)
reviewed_file: /home/knatte/Code/quarry/wts/diff-to-symbols/_mill/discussion.md
date: 2026-09-05
```

## Findings

### [BLOCKING:design] "One enumeration rule" uses two unequal git commands
**Section:** § Layering — "One enumeration rule, both sides" / § Git plumbing
**Issue:** `git ls-tree --name-only <rev> <dir>/` is non-recursive (immediate entries only) while `git ls-files --cached --others --exclude-standard -- <dir>/` is inherently recursive and has no non-recursive mode, so the working-tree side sweeps in subdirectory `.go` files (real case: `internal/engine/treesitter/treesitter.go`, clause `treesitter`) that the revision side never sees — the two sides' clause votes can disagree, which is the exact create-plus-delete storm this decision exists to prevent.
**Fix:** State the enumeration as "immediate `.go` children of the directory only" and name how each side is trimmed to that (e.g. reject any listed path containing a further `/` after `<dir>/`), or choose a pair of commands that genuinely agree.

### [BLOCKING:design] No ordering rule for the answer's top-level arrays
**Section:** § Two rename tiers / § Evidence-tier gating / § Testing (Goldens)
**Issue:** Ordering is fixed only for candidates *within* one entry, for `files` (input order) and for a `modified` entry's `after`/`before`; `created`, `deleted`, `modified`, `renamed` and the `rename_candidates` entries themselves have none, yet they are built from a table keyed by `(ID, Kind)` and Go map iteration is randomised — committed JSON/text goldens cannot be byte-stable.
**Fix:** State a deterministic ordering for every top-level array (the file-then-line rule `symbolsOfUnit` already uses, or `id` ascending), and say it is ordering, not ranking.

### [BLOCKING:design] Quarry's root need not be git's top-level
**Section:** § Git plumbing / § CLI shape
**Issue:** `repopath.ResolveRoot` accepts any existing directory for `--root` and skips `.git` discovery entirely, so `git -C <root>` can run inside a repo whose top-level is above `<root>`; `git diff --name-status` and `git ls-tree` then emit paths relative to the *git* top-level while quarry consumes them as `<root>`-relative, and a `--root` outside any repo makes every git call fail into exit 3 with git's raw message. The discussion asserts "`git -C <root>` interprets a pathspec against the root" and never states the root/top-level relationship.
**Fix:** Decide it explicitly — verify `git rev-parse --show-toplevel` equals `<root>` and give the mismatch and the not-a-repository case quarry's own sentence and exit code, or define the prefix translation both ways.

### [BLOCKING:design] Working-tree clause vote lists files that are not on disk
**Section:** § Layering — the clause-map seam
**Issue:** `git ls-files --cached` lists index entries regardless of working-tree presence, so a `.go` file deleted but not staged — the routine case this verb exists to report — is handed to `(*engine.Repo).ClauseMapForFiles`, which reads it from disk; the discussion never says whether a read failure is skipped (as `dirPackage` silently does today) or returned through that method's `error`, in which case a plain deletion fails the whole `DeltaGit` call.
**Fix:** State `ClauseMapForFiles`' disposition for a base name it cannot read or decode — skip and record no clause, matching `dirPackage` — and say so where the working-tree enumeration is chosen.

### [NIT:scope] The core must fill `Symbol.File`; nowhere said
**Section:** § Technical context / § What `modified` means
**Issue:** `Strategy.Symbols(unit, root, src)` leaves `File` empty — `symbolsOfDir` assigns `sym.File = fileRel` at the call site — yet `changed:["file"]`, the `before` block and the `renamed` pair all depend on it.
**Fix:** State that the delta core sets `File` from the entry's own `path` on each side.

### [NIT:consistency] Real-history pin's created/deleted sets are partial
**Section:** § Technical context — Real-history pin / § Testing
**Issue:** `internal/engine#SelfGlyphError` (the type declared alongside the listed `SelfGlyphError.Error` in `expand.go`) is absent from the created list, and two of the three listed deletes are outside the `glyph/`+`internal/engine/` scoping the test uses; Testing says "the created and deleted sets", which reads as exact-set assertions.
**Fix:** Say the assertions are presence-only over the in-scope subset, or complete the sets.

### [NIT:scope] The "adds no behaviour" claim sits in two files
**Section:** § Technical context — Facade aliasing
**Issue:** `quarry/repo.go:21` carries the same "facade adds no behaviour of its own" statement `quarry/doc.go:6` does; only `doc.go` is named for amendment.
**Fix:** Name both, so `DeltaGit` does not leave one doc comment false.

## Verdict

REQUEST_CHANGES
Four blocking gaps: enumeration asymmetry, missing output ordering, root-vs-git-toplevel, and unreadable-file disposition.
MILL_REVIEW_END
