MILL_REVIEW_BEGIN
# Review: Improve gopls query precision (build tags + scoping)

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 4.x-class model (Anthropic); exact build not self-verifiable
reviewed_file: /home/knatte/Code/quarry/wts/gopls-query-precision/_mill/discussion.md
date: 2026-08-27
```

## Findings

### [BLOCKING:design] Implementation-widening precision claim is self-contradictory
**Section:** `verification-over-scope-heuristics` → "Precision impact of the widening, and why it is small"
**Issue:** The claim that the unrelated `clock` interfaces "are not in any implements relation with the queried symbol, so `textDocument/implementation` does not re-admit them" contradicts this discussion's own Problem statement, which attributes the noise to gopls matching "every structurally-compatible interface in the workspace" — Go's implements relation is structural, so the concrete types satisfying the *other* packages' `clock` interfaces also satisfy the queried one and are legitimate `textDocument/implementation` results; references to them (issue #1's "plus their test files") would then match the widened declaration set and be kept.
**Fix:** State the disposition for the case where `textDocument/implementation` returns structurally-satisfying types from unrelated packages — either verify gopls v0.23.0's actual implementation result on the `clock` fixture before committing to the symmetric widening, or record what the expected post-fix count is if 31→2 is not achievable, so the live-tier test in Testing ("reports only the callers of the interface under test") has a decidable expectation.

### [BLOCKING:design] "Declaration set" names two different sets in the return contract
**Section:** `one-connection-verification-entry-point` vs `declaration-match-is-positional`
**Issue:** `declaration-match-is-positional` defines the declaration set as `definition ∪ implementation`, while the entry point's pipeline is still written as "resolve position → `textDocument/definition` (the declaration set) → references" and justifies returning it so `filterUnexpectedCallers` can "exclude that site — exactly as it uses today's `declRefs`"; the two readings differ behaviourally, because `filterUnexpectedCallers` (`internal/cli/cli.go:671-688`) removes every returned declaration from the violation list, so returning the union silently excludes every implementer's declaration site from the gate.
**Fix:** Say explicitly whether `query.Callers`'s second return value is the definition-only declaration (today's `declRefs` semantics) or the widened union, and update the entry point's pipeline sentence to include the implementation call.

### [NIT:consistency] `--no-verify` help scoped to four verbs, not one
**Section:** Scope → Documentation, third bullet
**Issue:** "Cobra help text on the four verbs — `--build-tags` and `--no-verify` need flag help" reads as adding `--no-verify` to all four, contradicting `verification-scoped-to-assert-no-callers` ("`refs`, `definition`, and `symbol` ... gain no verification flag").
**Fix:** Split the bullet so `--build-tags` is the four-verb item and `--no-verify` is explicitly `assert-no-callers`-only.

### [NIT:consistency] Test commands stated without the mandated verify: prefix
**Section:** Testing, tier headings
**Issue:** The tiers are named as `go test ./...` and `go test -tags lsp ./...`, while `mill-config.yaml:223-236` records that every non-null per-batch `verify:` must begin with the literal `PYTHONPATH= ` token (enforced by `_plan_validate.verify-not-isolated`), so a plan writer copying these strings verbatim into `verify:` would trip the validator.
**Fix:** Note in Testing that the plan's `verify:` lines carry the `PYTHONPATH= ` prefix, or state the tier commands as the bare `go test` invocations they are and leave `verify:` shaping to the plan.

## Verdict

REQUEST_CHANGES
Implementation-widening premise and the ambiguous returned declaration set both need a decision.
MILL_REVIEW_END
