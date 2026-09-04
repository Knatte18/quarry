MILL_REVIEW_BEGIN
# Review: resolve + expand (T4) — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (1M context)
reviewed_file: plan/
date: 2026-09-04
```

## Findings

### [BLOCKING:scope] Card 9's stale-comment inventory is short by two
**Location:** batch 3 / card 9
**Issue:** Card 9 says "Re-tense every comment in `internal/engine/resolve.go` that speaks of `resolve` as a verb that does not exist yet. There are three, and this card owns all of them", but the file holds five such clauses across four comments: the two it names plus the header's `Repo.SpansOf is the public, per-glyph wrapper the rest of this task's later verbs (resolve, expand) are built on` (resolve.go:3-5) and `symbolsOfUnit`'s `the later resolve verb needs the same grouping anyway for the many glyphs one card can name` (resolve.go:105-107). The header clause is not merely mistensed: the Shared Decision "one unit is parsed once per call" and card 5 both state `SpansOf` is used by neither verb, so leaving it standing ships a sentence the plan's own design makes false.
**Fix:** Enumerate both clauses in card 9's list and state that the header's `SpansOf` sentence must say the verbs are built on `symbolsOfUnit` through the memo, not on `SpansOf`.

### [BLOCKING:consistency] No card owns resolve_test.go's now-stale header
**Location:** batch 3 / cards 10–13
**Issue:** `internal/engine/resolve_test.go`'s header comment says the file "covers `Repo.unitDirs`, `Repo.symbolsOfUnit`'s ignore filtering, and the public `Repo.SpansOf`" and describes its own committed-vs-`.scratch` fixture split. Cards 10–13 add fourteen `Test` functions for `Resolve`, a new collision tree and a chmod fixture, and no card updates that enumeration — while cards 2, 9 and 23 each spend a paragraph on the principle that a file describing itself wrongly must be corrected.
**Fix:** Give one of cards 10–13 (or a sibling of card 9) ownership of that header comment, naming it explicitly as cards 2 and 9 do for their files.

### [BLOCKING:consistency] Lint-debt Decision contradicts mill-config.yaml's own record
**Location:** overview / "the configured done gate carries pre-existing lint debt this task does not own"
**Issue:** The Decision asserts `golangci-lint run` "exits 1 on this branch's tip today, before any of this plan's changes". `mill-config.yaml:122`, the very key the Decision quotes, carries the opposite record inline: "golangci-lint is now installed at ~/go/bin/golangci-lint and reports no finding on the current tip, so the gate carries no pre-existing debt." The plan neither cites nor reconciles that comment, so either the Decision's premise is wrong or the tracked config comment is stale after the T3 merge-in — and a reader of either artefact is misinformed.
**Fix:** State in the Decision which of the two is current and why (e.g. that `bench/loomyard-eval/ladder/` arrived on this branch after that comment was written), leaving the config file itself untouched as the Decision already requires.

### [NIT:consistency] Per-batch module-wide vet is claimed but not configured
**Location:** overview / "verify commands are cgo-enabled and scoped to the engine package"
**Issue:** The rationale says "`go vet` at module scope is the cheap cross-package regression check at each batch boundary", but only batch 1's `verify:` runs anything module-wide (`go build ./...`); batches 2–5 run `CGO_ENABLED=1 go test ./internal/engine/` only, and `go vet ./...` appears solely as the task-level `verify:`.
**Fix:** Either drop the "at each batch boundary" clause or add the module-wide vet to each batch's `verify:`.

## Verdict

REQUEST_CHANGES
Three stale-artefact and premise defects; the verb design itself is sound and faithful.
MILL_REVIEW_END
