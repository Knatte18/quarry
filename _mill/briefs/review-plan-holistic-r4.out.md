MILL_REVIEW_BEGIN
# Review: The glyph package (T1) — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (claude-opus-5)
reviewed_file: plan/
date: 2026-09-03
```

## Findings

### [BLOCKING:design] golangci-lint verify rests on a refuted premise
**Location:** `00-overview.md` `verify:` line and Shared Decision "golangci-lint is green at the parent tip"
**Issue:** The decision asserts "`golangci-lint run` exits 0 at the parent tip, so adding it costs nothing", but `mill-config.yaml:122` — the same hub config the decision cites for `pipeline.done_gate` — records verbatim that "golangci-lint is not installed on this machine, so a done_gate naming it would fail for a reason unrelated to this task's changes"; the module-wide verify would therefore fail command-not-found at both batch boundaries.
**Fix:** Drop `&& golangci-lint run` from the overview `verify:` and retire the decision, or state the evidence that the tool is now installed and green.

### [BLOCKING:scope] "green at parent tip" unverified; no lint config exists
**Location:** `00-overview.md` Shared Decision "golangci-lint is green at the parent tip"
**Issue:** The repository has no `.golangci.yml`/`.golangci.yaml` at any path, so the claim is about golangci-lint's *default* linter set (errcheck, staticcheck, unused, …) running for the first time over the pre-existing cgo engine packages — nothing in the plan or discussion records that this was ever executed, and the discussion's own verification list names only `go build`, `go test`, `go vet` and `go list -deps`.
**Fix:** Either record the observed exit-0 run and the config it used, or scope the new command to `./glyph/` where the plan can be held responsible for it.

### [NIT:consistency] Tie-break rule points at the aliasing bug card 1 forbids
**Location:** `00-overview.md` Shared Decision 1 vs. `01-types-and-printer.md` card 1, item 4
**Issue:** Card 1 explicitly bans `append(g.Owner, g.Name)` because it can write into the caller's backing array, but the discussion's Decisions section writes the print form literally as `strings.Join(append(Owner, Name), ".")`, and Shared Decision 1 says the discussion wins wherever the two disagree — so the stated tie-break resolves this disagreement toward the defect.
**Fix:** Carve the print-form expression out of Shared Decision 1's "discussion wins" clause, naming card 1's `make`+two-`append` construction as the authoritative reading.

### [NIT:consistency] `section` column is written but never read
**Location:** `02-parser-and-go-alphabet.md` cards 5 and 6 (`rejectCase.section`, `acceptCase.section`)
**Issue:** Both table types carry a `section` field that every row sets and no test ever reads; under the default `unused` linter the overview's module-wide verify prescribes, a write-only unexported struct field is a plausible report, and the field's traceability purpose is served equally by a comment.
**Fix:** State in the card that `section` is asserted or surfaced somewhere (e.g. included in the `t.Run` subtest name) so it is genuinely read.

## Verdict

REQUEST_CHANGES
Module-wide verify names a linter the hub config records as not installed.
MILL_REVIEW_END
