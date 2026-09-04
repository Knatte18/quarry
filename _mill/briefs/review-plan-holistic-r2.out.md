MILL_REVIEW_BEGIN
# Review: Facade + CLI, toc (T5a) — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (opus-5, high reasoning effort)
reviewed_file: plan/
date: 2026-09-04
```

## Findings

### [BLOCKING:design] `--symbols` pass-through test rests on a false premise
**Location:** batch 4 / card 18 (*Flag pass-through*); gotcha 1 unaccounted for plan-wide
**Issue:** `walk.go`'s `unitFor` returns `""` for `dirRel == "."` and `unitSpellable("")` is false, so a Go file directly under the fixture root never carries `Symbols` even with `--symbols`; the card's fixture ("a directory with a Go file…") and its assertion "file entries carry `symbols`" do not say the queried target must be a non-root subdirectory. `discussion.md` lists this as gotcha 1 ("all of which the plan must account for") and no card records it.
**Fix:** state in card 18 that the `--symbols` assertion queries a named subdirectory of the fixture root, and record gotcha 1 as a Shared Decision or in cards 6/18 so the text/JSON views are never assumed to show a `symbols` key after `--symbols`.

### [BLOCKING:consistency] Copied `writeScratchTree` resolves the wrong module root in `quarry/`
**Location:** batch 1 / card 4 (vs. batch 3 / card 14)
**Issue:** `internal/engine/scratchtree_test.go` walks three `filepath.Dir` levels because it sits two directories below the module root; `quarry/scratchtree_test.go` sits one. "Mirroring" the helper verbatim resolves `moduleRoot` to the parent of the repository and writes `.scratch/quarry-tests/` *outside* the repo — tests still pass, so the `fixture-trees-live-under-scratch` decision is violated silently. `internal/cli`'s copy (card 14) is at the engine's depth and is unaffected.
**Fix:** card 4 should state the level count explicitly ("`quarry/` is one directory below the module root, so one `filepath.Dir` fewer than the engine's copy").

### [NIT:consistency] Golden invocation line may embed the `--root` machine path
**Location:** batch 5 / card 20 (assembly rule)
**Issue:** the card says to assemble `$ quarry toc ` + "the same flags and target the case passed", but every case passes `--root <absolute Loomyard path>`; taken literally that writes an absolute machine path into a tracked golden, which the task's hard constraint forbids and only card 21's post-hoc read catches, after regeneration.
**Fix:** say the invocation line uses the table's "args after the verb" column only and never `--root`.

### [NIT:consistency] Golden test function name is assumed but never fixed
**Location:** batch 5 / card 20 vs. cards 21 and 23
**Issue:** cards 21 and 23 both prescribe `go test ./internal/cli/ -run TestAfter -update`, but card 20 never names the test function it creates; a different name makes the `-update` run a silent no-op that produces no files.
**Fix:** card 20 should name the test function (e.g. `TestAfterGoldens`, prefix-matched by `-run TestAfter`).

### [NIT:scope] Step-8 error branches have no test path
**Location:** batch 4 / card 16 step 8, card 18
**Issue:** `discussion.md`'s Testing block asks for an error→code table including "an arbitrary error → 3" and the wrapped sentinels *from `TOC`*; steps 5 and 6 exclude both, and card 18's only exit-3 fixtures are the `ENOTDIR` stat and the failing writer, so step 8's three branches are unreachable and untested.
**Fix:** either name a small pure mapping helper in card 16 that card 18 can table-test, or record in card 18 that those branches are deliberately race-only and untested.

### [NIT:consistency] "Every token not beginning with `-` is a target" contradicts value consumption
**Location:** batch 3 / card 10 (last bullet)
**Issue:** `--depth 3` and `--root path` supply value tokens that do not begin with `-`; the bullet as written would count them as targets and yield "toc takes exactly one target, got 2" for a valid invocation.
**Fix:** qualify the bullet as "every token not beginning with `-` and not consumed as a preceding flag's value".

### [NIT:scope] Card 23's `Context:` omits files its Requirements tell it to read
**Location:** batch 5 / card 23
**Issue:** the card says to record what changed "reading the generated files", but `Context:` lists only `after/toc-dir.txt` and `after/toc-dir-text.txt` of the four, and neither `../toc-dir-compact.txt` nor `../toc-file-compact.txt`, which the mapping table must name.
**Fix:** add the remaining two `after/` outputs and the two compact before-side files to card 23's `Context:`.

## Verdict

REQUEST_CHANGES
Two blocking issues: unaccounted symbols gotcha and a scratch-tree helper that escapes the repo.
MILL_REVIEW_END
