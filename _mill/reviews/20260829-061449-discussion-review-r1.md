MILL_REVIEW_BEGIN
# Review: Add an MCP wrapper for quarry

```yaml
duration_s: 271.0
verdict: REQUEST_CHANGES
reviewer_model: opus
reviewer_self_id: Claude (Anthropic); system identifies this build as claude-opus-5. Self-assessment cannot confirm the exact version beyond that label.
reviewed_file: /home/knatte/Code/quarry/wts/quarry-mcp-wrapper/_mill/discussion.md
date: 2026-08-29
```

## Findings

### [BLOCKING:design] Tier 1/2 need a stub seam no decision creates
**Section:** §Testing (tiers 1–2) / §test-strategy
**Issue:** Tier 2 says "facade calls stubbed" and tier 1 is declared build-tag-free, but `internal/cli` calls `quarry.References/Definition/Symbol/Callers/Impact` as package-level functions with no injection seam (cli.go:195,353,469,679; impact.go:151), so five of seven handlers cannot be exercised without gopls unless `internal/mcpserver` introduces a seam.
**Fix:** Decide explicitly whether `internal/mcpserver` carries an injectable facade seam (func vars / interface) or whether tiers 1–2 restrict to translation, parsing, options assembly and the two toc handlers.

### [NIT:decision] `within`-filter helpers have no disposition
**Demoted-from:** BLOCKING
**Section:** §Technical context ("`within` filtering is CLI-side") / §Constraints
**Issue:** "Prefer exporting if the logic is non-trivial" and "if chosen, exporting the within-filter helpers" leave `filterWithin`/`isWithinDir`/`filterImpactWithin` (cli.go:757,784; impact.go:275) undecided, while §Scope enumerates only four exported helpers and §Testing mandates a `within`-filtering test.
**Fix:** State export-vs-reimplement as a decision with the same bit-for-bit rationale used for the state-dir helpers.

### [BLOCKING:design] Array-batching execution model unspecified
**Section:** §array-batching / §param-split
**Issue:** Nothing states whether entries execute sequentially or concurrently, whether `--timeout` is a per-entry or whole-call budget, or whether array length is bounded; `lsp.Client` is documented single-flight ("two concurrent calls on it would consume and drop each other's responses", query/callers.go:52-55) and an MCP server may receive overlapping `tools/call` requests — a concurrency mode the one-shot CLI never exercised.
**Fix:** Decide entry execution order, timeout scoping, array-size bound, and whether the server serializes concurrent tool calls.

### [NIT:consistency] CLI batch-parity claim is inverted
**Demoted-from:** BLOCKING
**Section:** §array-batching (Decision + Rationale)
**Issue:** "New functionality relative to today's CLI for the LSP-mirrored three, and matches the existing CLI batch pattern for the rest" is backwards: refs/definition/symbol already take N positional args through `runBatch` (cli.go:140,291,433,203,378,489), while `assert-no-callers` is `cobra.ExactArgs(1)` (cli.go:631) with no batch envelope at all.
**Fix:** Correct the statement and name `assert_no_callers` as the one tool whose batch envelope has no CLI precedent to mirror.

### [BLOCKING:design] `assert_no_callers` batch/result semantics undefined
**Section:** §error-mapping / §param-split
**Issue:** The CLI emits `{"violation": true, "callers": [...]}` with exit 1 (cli.go:701-708); the four per-entry statuses (`found`/`not_found`/`ambiguous`/`error`) have no slot for "resolved but violating", and `except`/`within` are listed as call-wide parameters, contradicting in-file-is-per-entry's own heterogeneous-targets argument.
**Fix:** Specify where `violation` lives in the per-entry envelope and whether `except`/`within` are per-entry or call-wide.

### [NIT:consistency] toc path semantics contradict the output rule
**Demoted-from:** BLOCKING
**Section:** §file-reference-form vs §Technical context ("`toc` specifics")
**Issue:** "Always emit plain absolute paths on output" contradicts "preserve" of `toc dir`'s caller-relative per-file `path`, which is composed as `filepath.Join(arg, result.Files[i].Name)` (toc.go:392); separately, `toc file`'s `.quarry.yaml` base is the *file's own parent directory* (`targetDir := filepath.Dir(abs)`, toc.go:305), not the CLI/MCP `--target-dir`, so reusing the MCP `targetDir` there would break the claimed byte-comparability with CLI output.
**Fix:** State the exception for `toc_dir` file paths and pin `toc_file`'s config base to the resolved file's parent directory.

### [NIT:consistency] "structurally removes the hazard" is overstated
**Section:** §binary-shape vs §shared-resolution-helpers
**Issue:** `internal/mcpserver` importing `internal/cli` links cobra and `internal/output` into `quarry-mcp`, so stdout purity rests on discipline plus the tier-3 assertion, not on a structural absence of stdout writers.
**Fix:** Reword the rationale to "no CLI command ever runs in this process", which is what actually holds.

### [NIT:design] `.mcp.json` cold-start cost not addressed
**Section:** §mcp-json-wiring
**Issue:** quarry requires `CGO_ENABLED=1` and a C toolchain (`internal/quarryengine/cgoguard_nocgo.go`), so the first `go run ./cmd/quarry-mcp` is a cgo build; "cached after the first launch" does not address a client startup timeout or a missing C toolchain on the first connect.
**Fix:** State the expected cold-start behaviour and what a failed first build looks like to the client.

## Verdict

REQUEST_CHANGES
Six blocking gaps: test seam, within helpers, batching model, batch-parity claim, assert_no_callers semantics, toc paths.
_Note: 3 finding(s) demoted from BLOCKING to NIT by the stage's blocking-class ceiling; current blocking_count is 3._
MILL_REVIEW_END
