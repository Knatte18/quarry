# Review: The glyph package (T1)

```yaml
verdict: APPROVE
reviewer_model: orchestrator
reviewer_self_id: claude-fable-5
reviewed_file: _mill/discussion.md
date: 2026-09-03
```

## Findings

### [NIT:consistency] Reserved-names citation points at the wrong section
**Section:** Scope → Out, first bullet
**Issue:** "§1 reserves the names" — the reservation of `Python`/`CSharp` as constant names is in `docs/glyph.md` §6 ("the names reserved for the alphabets below"), not §1.
**Suggested fix:** Change the citation to §6.

### [NIT:design] Whitespace ban is presented as required by §6, but §6 tolerates quote-needing glyphs
**Section:** Decisions → The Go unit alphabet; Open questions #2
**Issue:** The rationale says §6's writing-down rules "require" rejecting whitespace, yet §6 itself accepts glyphs that need quoting (C# `(`, `,`, `<`: "quote them where a format cares"), so the ban is a defensible choice, not a derived requirement.
**Suggested fix:** Phrase open question #2 (and the rationale) as a proposed rule the hub can accept or drop, not as a §6 consequence.

### [NIT:design] Reject precedence between language check and structural split is unstated
**Section:** Decisions → The split is at the first `#`; Testing → parse_test.go
**Issue:** For an input failing both checks (e.g. `Parse(Language("python"), "no-hash")`), the discussion does not say whether `unsupported_language` or `no_separator` wins, and the reason-asserting tests depend on it.
**Suggested fix:** State the order (language check first is the natural reading of the `unsupported_language` tests) so the plan encodes it.

## Verdict

APPROVE
Complete, spec-grounded, all decisions carry rationale and rejects; the two spec gaps are correctly routed to the hub.
