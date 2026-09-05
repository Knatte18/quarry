MILL_REVIEW_BEGIN
# Review: The glyph-maker: declaration to glyph (P1, roadmap 2b) — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 4.x-class model (Anthropic), running as the mill plan reviewer
reviewed_file: plan/
date: 2026-09-05
```

## Findings

### [BLOCKING:scope] Stale all-verbs flag test missed by the inventory
**Location:** batch 3 / card 11 (and card 8)
**Issue:** `internal/cli/flags_test.go:256-258` declares `TestParseArgs_TextAndRootValidForAllVerbs`, whose name and doc comment state "--text and --root are accepted on every verb"; card 8 makes `--root` a per-verb rejection for `name`, so both become false, and card 11 touches only rows 156/158 and `TestParseArgs_ThreeVerbGate`. The file is edited by batch 3, so batch 5's "files no earlier batch edits" carve-out does not reach it.
**Fix:** add that function's rename and doc-comment correction (and a `name --text` row) to card 11's requirements.

### [BLOCKING:design] Internal-reason CLI bytes cannot be reached as specified
**Location:** batch 3 / card 12 (with card 9)
**Issue:** the card requires asserting that the internal reason takes `fail`'s path — envelope on stdout, sentence on stderr, exit 3, no payload — then states the facade cannot produce that reason and directs the test to "construct the result directly and drive the renderer and mapper". Verified in `internal/cli/cli.go`: `runName` per card 9 calls `quarry.Name` directly with no seam, so driving the renderer and mapper exercises neither `runName`'s branch nor its stdout/stderr bytes; the required assertion has no route.
**Fix:** decide the mechanism in the plan — an injectable maker seam in card 9, or downgrade card 12's requirement to the mapper table plus a stated untested-branch disposition.

### [BLOCKING:design] The `var B` test row has no stated, and no plausible, expectation
**Location:** batch 1 / card 2
**Issue:** the card pairs `const B` (measured: clean parse, one symbol) with "its `var` counterpart, a fragment of exactly `var B`, gets a row too" without saying what that row asserts. A Go var spec always carries a type or an expression list, so `goGroupedConstOrVarSymbols` (`internal/engine/golang.go`, which prepends `"var "` to the spec's own text) can never emit a bare `var B`, and the discussion's measured table covers `const B` only — the const row's expectation cannot be inherited.
**Fix:** state the expected outcome for `var B` explicitly (id-and-kind, or `NameReasonParse`), or drop the row and keep the measured `const B` case alone.

### [NIT:consistency] runName's internal branch duplicates the exit-code mapping
**Location:** batch 3 / card 9
**Issue:** step 2 hardcodes exit 3 on the `fail` path while `codeForNameResult` also maps that reason, giving the verb two sources for one code; `runExpand` in `internal/cli/cli.go` deliberately passes `codeForExpandError(err)` into `fail` on every error branch so the mapping stays single-sourced.
**Fix:** have step 2 pass `codeForNameResult(result)` as `fail`'s code argument, matching `runExpand`.

### [NIT:design] Nobody is assigned the missing-counts-golden message
**Location:** batch 4 / cards 15, 16
**Issue:** card 16 requires the test to fail naming the regeneration command "rather than with a bare read error", but `compareGolden` (`internal/engine/golden_test.go:121-124`) `t.Fatalf`s on the failed `os.ReadFile` itself, and card 15 changes only its third parameter and doc comment — so the required message has no owner.
**Fix:** name the mechanism in card 16 (an existence pre-check before the `compareGolden` call, honouring the update flag).

### [NIT:consistency] "newlines intact" in JSON is an escaped sequence, not a byte
**Location:** batch 2 / card 7 and batch 3 / card 12
**Issue:** both cards pin the divergence as JSON echoing `target` "with its newlines intact"; `renderJSON` (`quarry/render.go`) disables only HTML escaping, so encoding/json still emits a newline as the two-character `\n`, and a raw-byte assertion would be wrong in both tests.
**Fix:** say the JSON half is asserted after decoding the payload's `target` value.

### [NIT:scope] No partition disposition for symbols from a lossy parse
**Location:** batch 4 / card 16, step 2
**Issue:** `fileEntry` (`internal/engine/walk.go`) sets `Lossy` on a partial parse but still fills `Symbols`, so the harvest can carry a symbol whose `Signature` was cut from a broken file; the two exclusion rules (multi-name spec, interface method) do not cover it, and such a symbol would land in-contract and fail step 3's zero-misses assertion.
**Fix:** state a disposition — exclude, or record the case as deliberately in-contract because the pinned checkout compiles.

## Verdict

REQUEST_CHANGES
Three blocking gaps: a missed stale test, an unreachable CLI assertion, an unspecified test row.
MILL_REVIEW_END
