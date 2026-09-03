# Review: Engine core (T3)

```yaml
verdict: REQUEST_CHANGES
reviewer_model: orchestrator
reviewed_file: _mill/discussion.md
date: 2026-09-03
```

## Findings

### [BLOCKING:design] D16 strips `_test` from a unit unconditionally
**Section:** D16 (and D7's inverse mapping)
**Issue:** "maps the glyph's unit to a directory (stripping a trailing `_test` …)" rests on the premise that a trailing `_test` always denotes the external test unit — but a directory literally named `foo_test/` is legal Go, D7's walk would assign its symbols the unit `…/foo_test`, and D16's lookup would then strip it and search a `foo/` directory that need not exist: the same glyph string names two different units. Neither quarry nor Loomyard has such a directory today, so the round-trip criterion passes without ever exercising this — which is exactly why the rule must be right by construction, and T4's `resolve` inherits it.
**Suggested fix:** Amend D16: try the unit as a literal directory first; only when no such directory exists and the unit ends `_test` fall back to the external-test interpretation (strip + filter). Note the both-exist collision as `ambiguous`, deferred to T4's status vocabulary, and record the gap glyph.md §2 leaves open.

### [NIT:scope] README fix lives only in the gotchas list
**Section:** Scope / Technical context ("Gotchas found during exploration")
**Issue:** "Fix the stub while the engine is being renamed" (stale `map`/`members` — verified stale on the branch) is a work item, but the Scope "In" list does not carry it, and a plan writer building batches from Scope alone would drop it.
**Suggested fix:** Add the README verb fix to the Scope "In" list.

### [NIT:consistency] D9's pattern subset omits interior-slash anchoring
**Section:** D9
**Issue:** "exercise exactly the subset above and nothing more" — Loomyard's `plugins/prowler/bin/` needs the rule that a pattern *containing* a slash is anchored to the `.gitignore`'s directory; the subset states only leading-`/` anchoring and the without-slash-floats rule, leaving the interior-slash case implied rather than stated.
**Suggested fix:** Add "a pattern containing a slash is anchored to its `.gitignore`'s directory" to the supported-subset list.

## Verdict

REQUEST_CHANGES
D16's unconditional `_test` strip mis-resolves a legal directory shape the round trip never exercises; two NITs recorded.
