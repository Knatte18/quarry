MILL_REVIEW_BEGIN
# Review: Kick-start pack bench: pre-resolved glyph spans in the prompt (M4) — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 4.x-class model (Anthropic); exact build not self-verifiable
reviewed_file: plan/
date: 2026-09-05
```

## Findings

### [BLOCKING:design] Descriptive card's "human-only comment" has no mechanism
**Location:** batch 6 / card 27 (last paragraph) **Issue:** the card instructs writing the treatment-vs-descriptive asymmetry into `07-e2-files.md` "as a comment for a human reader rather than as prompt text", but card 8 defines `LoadCardFile` as reading the file whole with no extraction ("a card is prompt text in its entirety"), and card 17 itself states markdown comment lines "remain a literal, greppable line in the prompt text" — so this text lands verbatim in `e2-files`' measured prompt only, an arm-only block describing the experiment. **Fix:** decide where that note lives given no extraction exists (e.g. the ladder file's header comment or the task file's notes section), and say so in card 27.

### [BLOCKING:scope] Card 5 names gates.go and runstate.go symbols not in Context
**Location:** batch 2 / card 5 **Issue:** `Requirements:` names `CheckRenderedControlPrompt`, `CheckServerConnected` and `CheckBlinding`, declared in `bench/loomyard-eval/ladder/internal/ladder/gates.go`, and `RunState`, declared in `runstate.go:149`; neither file is in the card's `Context:` (config.go, summarize.go) or `Edits:` (run.go). **Fix:** add both paths to card 5's `Context:`.

### [NIT:consistency] Card 5's call-site count is eleven, not ten
**Location:** batch 2 / card 5 (first grep) **Issue:** `grep -n "IsControl()"` over non-test files yields config.go:72 (declaration), 131, 232; mcp.go:47; run.go:164, 182, 368, 415, 558, 664, 902, 912 — eleven call sites, not the "ten" the card says the confirmation should show; the card's own enumeration (6 switched + 2 baseline + 2 in config.go) omits mcp.go:47, which is card 6's site even though the sentence says cards 6 and 7 are included. **Fix:** state eleven and name mcp.go:47 explicitly in the breakdown.

### [NIT:consistency] Card 11 misplaces the assertion it replaces
**Location:** batch 2 / card 11 **Issue:** the `Config{Allowed: nil}.IsControl()` assertion is the `ControlCell_CallerGatesNotTheCheckItself` subtest at the foot of `TestCheckServerConnected` (gates_test.go:344-353), not "at the foot of the granted-tool-used test" (`TestCheckGrantedToolUsed` ends at line ~280). **Fix:** name the subtest and its enclosing test so the implementer edits the right block.

### [NIT:consistency] control/grants-tools decision's Applies-to omits two batches
**Location:** overview / `### Decision: control-and-grants-tools-are-two-predicates` **Issue:** `Applies to: control-sweep-and-card`, but the decision's own text says "after batch 1" and batch 1 card 1 is what implements the split, while batch 6 card 24's mandatory explicit `control: false` entries depend on it directly. **Fix:** list config-surface and benchmark-content alongside control-sweep-and-card.

## Verdict

REQUEST_CHANGES
One card prescribes card text with no mechanism; one Context list is incomplete.
MILL_REVIEW_END
