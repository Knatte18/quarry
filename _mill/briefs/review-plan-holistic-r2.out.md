MILL_REVIEW_BEGIN
# Review: resolve + expand (T4) — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (Anthropic)
reviewed_file: plan/
date: 2026-09-04
```

## Findings

### [BLOCKING:consistency] not_found JSON assertion contradicts ID rule
**Location:** batch 3 / card 11 (`TestResolve_NotFoundBothWays`), against card 2 and card 7
**Issue:** Card 11 requires the marshalled `not_found` result to have `id` absent, but card 7 says
"Otherwise set `Target` to the argument and `ID` to the parsed glyph's `String()`" for every
successfully parsed glyph target — `not_found` included — and card 2 documents `ID` as present for
any glyph target. As written the test cannot pass against the prescribed implementation.
**Fix:** State one disposition — either drop `id` from card 11's absent-key list, or have card 7/card 2
say `ID` is set only on a non-miss and adjust `TestResolve_Found`'s `ID == Target` assertion accordingly.

### [BLOCKING:scope] Card 16 Context omits resolve.go and glyph/
**Location:** batch 4 / card 16 (`TestExpand_Struct`)
**Issue:** Requirements say to "read the expected pair from a `SpansOf` lookup of the same glyph in
the test itself"; `SpansOf` lives in `internal/engine/resolve.go` and takes a `glyph.Glyph`
(`glyph/glyph.go`), and neither file is in the card's `Context:` or `Edits:`. Cards 15 and 18 of the
same batch do list `resolve.go`, so this is an omission rather than a deliberate exclusion.
**Fix:** Add `internal/engine/resolve.go` and `glyph/glyph.go` to card 16's `Context:`.

### [NIT:consistency] matchesFor leaves a third copy of the same filter
**Location:** batch 3 / card 7
**Issue:** `matchesFor` is justified as the owner-chain-and-name filter "lifted so both verbs share
one copy", yet the same card forbids changing `SpansOf`, which keeps its own inline copy of that
filter in the same file — three call sites, two implementations, in one file.
**Fix:** Either state explicitly why the duplication is accepted (T3-code freeze) as a recorded gap,
or let `SpansOf` call `matchesFor` as a behaviour-preserving edit under a third scope exception.

### [NIT:scope] Batch 5 prerequisite is work with no card
**Location:** batch 5 / Batch Scope "Batch prerequisite"
**Issue:** The prerequisite prescribes locating a Loomyard checkout and *writing* `.scratch/ladder.env`,
with no card, `Creates:` or `Commit:` behind it; and since card 19 changes nothing in `loomyardRepo`
except the parameter type, that file is never read by any test — only the exported
`LADDER_LOOMYARD_REPO` matters.
**Fix:** Either make it a card, or reword it as an environment step that only resolves a path to export,
dropping the write-into-`.scratch/ladder.env` instruction.

### [NIT:design] "expand needs no edit" claim for a subset-head language
**Location:** overview / Shared Decision "the emitted key set is closed and `Symbol` is unchanged"; batch 4 / card 15
**Issue:** Substituting `Head.Start`/`Head.End` with `HeadStart`/`HeadEnd` makes the full declaration
span unrecoverable from `ExpandAnswer` for the first language whose head is a strict subset — which
docs/rewrite-plan.md §5 says is "available" ("the whole class is `start`–`end` of the type symbol").
For Go the pairs are identical so nothing is lost today, but the rationale's "expand needs no edit"
is stronger than the shape supports.
**Fix:** Record this as a fourth contract gap in the same place the head substitution is implemented,
rather than asserting no future edit is needed.

## Verdict

REQUEST_CHANGES
One test contradicts its own implementation card; one Context list is incomplete.
MILL_REVIEW_END
