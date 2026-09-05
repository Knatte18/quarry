MILL_REVIEW_BEGIN
# Review: The glyph-maker: declaration to glyph (P1, roadmap 2b)

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (claude-opus-5), high reasoning effort
reviewed_file: /home/knatte/Code/quarry/wts/glyph-maker/_mill/discussion.md
date: 2026-09-05
```

## Findings

### [BLOCKING:design] Complete interface declarations cannot be named
**Section:** "Accepted declaration forms" vs "Exactly one symbol, or the entry fails"
**Issue:** `goTypeSymbols` (internal/engine/golang.go:170,183) appends `goInterfaceMethodSymbols` to the type symbol, so `type R interface { Read() error }` yields 1+N symbols and is rejected `several_declarations` — yet the decision says every shape is accepted "in either their head-only or their complete form", and the interface-method non-goal tells the consumer to create the method "as part of the interface's own type declaration".
**Fix:** State that an interface fragment must be head-only (or an empty body), and say so in the non-goal text and in the "complete form and head-only form agree" test case, which as written fails for any interface with a method.

### [BLOCKING:consistency] Round-trip harvest does not carry Signature or Kind
**Section:** Testing → "the round trip (the done-when gate)", step 1
**Issue:** "the same harvest `assertSymbolRoundTrip` already performs" is false: `roundTripSymbol` (internal/engine/roundtrip_test.go:33-37) carries only `id`, `unit` and a `spanTuple` of File/Start/SigEnd/End — no `Signature`, no `Kind`, both of which every step of the new test needs.
**Fix:** State the disposition of the existing shared struct — extend `roundTripSymbol` (also consumed by `TestRoundTrip_QuarryItself`) or harvest separately — rather than claiming the data is already collected.

### [BLOCKING:scope] Verb-count statements outside the three named docs
**Section:** Scope → Docs; Testing → Docs; Q&A "Which docs change?"
**Issue:** The docs inventory was assembled by naming three files, not by searching for statements of the verb set; at least `docs/rewrite-plan.md:12` ("three queries"), `internal/cli/doc.go:11` ("The command has three verbs"), `README.md:3` ("three"), `usageText`'s exit-code-1 prose (which enumerates negative-answer causes), and `internal/cli/flags.go:45,61,66` ("expected: toc, resolve, or expand", twice, plus the `parseArgs` doc comment) all go stale, and the discussion states "nothing else moves" in usage.go.
**Fix:** Replace the three-file list with an enumeration derived from a search for the verb set/count, and state the disposition of each site found.

### [NIT:design] Partition key "same unit and span" may not include File
**Section:** Testing → round trip, step 2
**Issue:** Excluding "symbols sharing a signature with another symbol in the same unit and span" is ambiguous about whether File is part of the key; build-tag twins in one unit can share signature and line span across two files, and would then be excluded and asserted (step 4) to produce an error they will not produce.
**Fix:** Say explicitly that the key is (unit, File, Start, End, Signature), or define the exclusion as "several symbols from one spec".

### [NIT:design] "Implausibly small" is an unquantified test threshold
**Section:** Testing → round trip, step 5
**Issue:** The vacuous-pass guard is stated as failing when the in-contract count is "implausibly small", with no number or ratio, so two plan writers would implement two different gates.
**Fix:** Give a concrete floor (an absolute count, or an in-contract share of the harvest).

### [NIT:design] Text failure line is not single-line for a multi-line head
**Section:** "Text view"
**Issue:** The contract is "one line ... exactly one trailing `\n`", but the failure form echoes `<target>` verbatim and `normalizeProse` is applied to the message only (quarry/text.go:241-250); a declaration head spanning lines — the shape ungrouped var/const signatures take — breaks the invariant.
**Fix:** State how a multi-line target is rendered in the text view, or scope the one-line claim to single-line heads.

## Verdict

REQUEST_CHANGES
Interface completeness, the round-trip harvest claim, and the docs inventory need resolving.
MILL_REVIEW_END
