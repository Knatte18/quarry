MILL_REVIEW_BEGIN
# Review: P2 — diff-to-symbols: changed file versions to symbol-table delta (roadmap 2c) — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: claude-opus-5 (harness-reported; no independent self-knowledge)
reviewed_file: plan/
date: 2026-09-05
```

## Findings

### [BLOCKING:consistency] Card 28 drops --exclude-standard the spec requires
**Location:** batch 5 / card 28
**Issue:** Card 28 requires that "neither [enumeration] applies any ignore filtering of its own", but `_mill/discussion.md:640` fixes the working-tree directory listing as `git ls-files --cached --others --exclude-standard -- <dir>/`, and its Rejected list (line 721) names `--others` without `--exclude-standard` as the option that "would sweep in build output and every ignored artefact"; an ignored untracked `.go` file would then vote in the clause map on the working-tree side only and can shift the whole directory's `dirPkg`.
**Fix:** Require `--exclude-standard` on the working-tree enumeration and restate the symmetry claim as tracked-inclusive plus ignored-untracked-excluded, matching card 27's own rule.

### [BLOCKING:scope] Card 7's "every call site" reaches an unlisted file
**Location:** batch 2 / card 7
**Issue:** `unitFor` has a second call site at `internal/engine/resolve.go:559` inside `symbolsOfDir` (which also calls `dirPackage` at line 546), so card 7's "removed in favour of it at every call site" branch edits `resolve.go`, a file in no card's `Edits:` and absent from `## All Files Touched`.
**Fix:** Either restrict card 7 to the thin-wrapper option, or add `internal/engine/resolve.go` to card 7's `Edits:` and to the overview's `## All Files Touched`.

### [BLOCKING:scope] Batch 2 verify misses the suites the refactor can break
**Location:** batch 2 / verify + card 10
**Issue:** `-run 'TestWalk|TestRepoTOC'` selects nothing in `internal/engine/resolve_test.go` or `expand_test.go`, yet `symbolsOfDir` (resolve.go:528–579) consumes both `dirPackage` and `unitFor`, which cards 6 and 7 rewrite; a behaviour change on the resolve/expand path passes this batch's verify silently, and card 10 names only the walk and toc suites.
**Fix:** Extend batch 2's `-run` and card 10's named suites with `TestSpansOf|TestResolve|TestExpand`.

### [BLOCKING:decision] The "--to was given" flag has no stated consumer
**Location:** batch 8 / card 42 (with cards 33, 44, 47)
**Issue:** Card 42 adds a third field recording whether `--to` was supplied "since ... an explicitly empty one is a different thing", but `DeltaGit(from, to, target string)` (card 33) has no way to express the distinction, and no card says what `quarry delta X --from R --to ""` does; the discussion (line 738) defines only "absent means the working tree".
**Fix:** State the disposition — reject an explicitly empty `--to` (and `--from`) as a usage error in the parser, or drop the third field and let empty mean working tree.

### [NIT:consistency] Three more doc claims go stale unnamed
**Location:** batch 6 / cards 32, 35; batch 2 / cards 6–7; batch 8 / card 44
**Issue:** `quarry/repo.go:12-14` says `Repo` "holds only the engine handle", which card 32 falsifies but card 35 does not list; `internal/engine/walk.go:1-3` enumerates `dirPackage ... and the unexported free function unitFor`, which cards 6–7 move or remove; `internal/cli/cli.go:302` carries an inline "three verbs" comment beside the doc comments card 44 does name.
**Fix:** Name these three sites in the cards that invalidate them.

### [NIT:consistency] Decision 1 does not carve out the plan's own departures
**Location:** overview / Shared Decisions
**Issue:** "The discussion is the specification ... an implementer who finds the two disagreeing follows the discussion" is stated without exception, while the goldens-location decision is a stated departure from the discussion's Testing section (`discussion.md:1127`); an implementer applying decision 1 literally writes the goldens under `internal/engine/testdata/delta/`, contradicting card 48 and `## All Files Touched`.
**Fix:** Name the stated departures as the explicit exceptions inside decision 1.

### [NIT:consistency] Card 5 says "six builder shapes" then lists eight
**Location:** batch 1 / card 5
**Issue:** The shape list (ungrouped function, method, struct type, grouped type, interface method element, ungrouped const, grouped var, type alias) has eight entries under the label "all six builder shapes", against six builders in `internal/engine/golang.go`.
**Fix:** Say "all six builders, across these eight shapes".

## Verdict

REQUEST_CHANGES
One spec contradiction, one unlisted edited file, one verify gap, one undecided flag.
MILL_REVIEW_END
