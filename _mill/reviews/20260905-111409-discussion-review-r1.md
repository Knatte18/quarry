# Review: Kick-start pack bench: pre-resolved glyph spans in the prompt (M4)

```yaml
verdict: APPROVE
reviewer_model: orchestrator
reviewed_file: _mill/discussion.md
date: 2026-09-05
```

## Findings

### [NIT:scope] D2's IsControl-to-GrantsTools switch misses the --allowedTools argv site
**Demoted-from:** BLOCKING
**Section:** D2 — "control" is decoupled from "grants no tools"
**Issue:** D2 enumerates the call sites that switch to `GrantsTools()` (`needsServer`, the
`ServerHashes` loop, `CheckRenderedControlPrompt`, `CheckBlinding`, `CheckServerConnected`) but
omits `run.go:664`, where `if !cfg.IsControl()` gates appending `--allowedTools` to the dispatch
argv. Under D2, `e1-pack` and `e2-files` are non-controls with an empty `Allowed` list, so that
branch appends `--allowedTools ""` (two argv entries, empty join) to their invocations while
`e0-names` gets no such flag — an unintended arm difference in the CLI invocation itself, exactly
the class of confound D1 exists to eliminate, with CLI-dependent semantics for an empty allowlist.
**Suggested fix:** Add `run.go:664` to D2's switch list — gate the `--allowedTools` append on
`cfg.GrantsTools()` — and extend the D2 regression tests with the case "tool-less non-control cell
dispatches with no `--allowedTools` flag". A sweep of every remaining `IsControl()` call site in
`run.go` (e.g. `:902`/`:912` correctly stay `IsControl()`) should be stated in the decision so the
enumeration is checked, not assumed.

## Verdict

APPROVE
One missed call site in D2's enumeration would give the two non-control arms a dispatch-argv
_Note: 1 finding(s) demoted from BLOCKING to NIT by the stage's blocking-class ceiling; current blocking_count is 0._
difference the design otherwise carefully eliminates; everything else verified clean.
