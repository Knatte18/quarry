MILL_REVIEW_BEGIN
# Review: Engine core (T3) — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: claude-opus-5 (Anthropic)
reviewed_file: plan/
date: 2026-09-03
```

## Findings

### [BLOCKING:consistency] Unspellable units error, they do not return []
**Location:** batch 5 card 34 (last bullet), against batch 5 card 32; premise forwarded from batch 4 card 30
**Issue:** Card 32 has `SpansOf` validate every glyph through `glyph.Parse(g.Lang, g.String())` and "return the resulting parse error wrapped on failure", explicitly covering "the empty unit"; both of card 34's unspellable fixtures are exactly such failures — unit `internal/engine/testdata/units/test data/pkg` trips `ReasonUnitBadRune` (`checkGoUnit`'s `unicode.IsSpace` test in `glyph/golang.go`) and the `testdata/units`-as-own-root case has the empty unit and trips `ReasonUnitEmpty` — so card 34's "yield an empty slice for any name" is unsatisfiable, and its claim that the walk's "no symbols there" and the lookup's "no spans there" are "one statement rather than two" rests on a false premise.
**Fix:** State one disposition for an unspellable unit reaching `SpansOf` — either card 34 asserts the wrapped `*glyph.ParseError` (matching card 34's own earlier argument-validation bullet) and drops the one-statement claim, or card 32 carves unspellable units out of the round-trip validation and returns an empty slice; card 30's forwarded sentence must be amended to match.

### [NIT:consistency] Batch 3 Batch Scope names a seam no card declares
**Location:** batch 3, Batch Scope line 25
**Issue:** The external interface is listed as "the two unexported walk seams `dirPackage` and `fileUnitPackage`", but no card declares `fileUnitPackage`; card 17's seams are `dirPackage`, `dirDoc`, `fileEntry`, `walkDir`, and unit derivation is batch 4's `unitFor`.
**Fix:** Replace `fileUnitPackage` with the seams card 17 actually declares.

### [NIT:consistency] Batch 6 verify deviates from the Shared Decision
**Location:** `00-overview.md` "every batch leaves the tree building" vs `06-goldens-and-round-trip.md` yaml `verify:`
**Issue:** The Decision states each batch's `verify:` is `go build` then `go test ./internal/...`, with `go vet` appearing only in the module-wide gate; batch 6's batch-level `verify:` inserts `go vet ./...`, and the Decision is not amended to allow it.
**Fix:** Amend the Decision to permit a batch adding `vet` when vet-cleanliness is its own subject, or drop it from batch 6's batch-level verify since the module-wide gate already runs it.

### [NIT:design] File-target answer never names which seams build it
**Location:** batch 3 card 18 (file-target bullet)
**Issue:** The card specifies the file target's answer shape ("the enclosing directory's `dir`, `package`, `language` and `doc`") without saying it must run card 17's `dirPackage` tie-break and the `PackageDoc`/`dirDoc` selection over the whole enclosing directory; an implementer reading only this card could parse the target file alone and emit that file's clause as the directory `package`.
**Fix:** Name `dirPackage` and `dirDoc` as the seams a file target reuses, and state that a file target therefore reads every `.go` file in its enclosing directory.

## Verdict

REQUEST_CHANGES
One card prescribes an assertion the plan's own validation rule makes unsatisfiable.
MILL_REVIEW_END
