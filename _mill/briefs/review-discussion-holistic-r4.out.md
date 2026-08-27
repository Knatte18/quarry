MILL_REVIEW_BEGIN
# Review: Improve gopls query precision (build tags + scoping)

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: claude-opus-5 (best-effort self-assessment; Anthropic Claude, Opus tier)
reviewed_file: /home/knatte/Code/quarry/wts/gopls-query-precision/_mill/discussion.md
date: 2026-08-27
```

## Findings

### [BLOCKING:consistency] Static template: "sent verbatim" vs "never sent"
**Section:** `template-substitution-semantics`
**Issue:** The first Decision says a template with no `{{tags}}` occurrence "is still sent verbatim when tags are absent-or-present (it is static configuration)", while the very next Decision says that with an empty tag set "no `initializationOptions` key is sent at all ... regardless of whether the entry carries a template", and Testing pins the second reading ("an empty tag set against a template-bearing entry produces no `initializationOptions` key at all").
**Fix:** State one rule for the {empty tag set, placeholder-free template} case and make the decision and the test bullet agree.

### [BLOCKING:design] Placeholder-free template silently swallows `--build-tags`
**Section:** `unsupported-language-hard-error` (with `template-substitution-semantics`)
**Issue:** The hard-error predicate is "resolved entry has an **empty** `initialization_options`", but a non-empty template containing no `{{tags}}` is explicitly blessed as legitimate static configuration — so `--build-tags integration` against such an entry (e.g. an operator `go:` block that overrides options but drops the placeholder; whole-entry replacement makes this the default outcome, `registry.go:53-60`, `docs/servers.yaml.example:1-6`) is accepted and silently ignored. That is the exact silent-ignore failure mode the decision exists to prevent.
**Fix:** Decide whether the predicate is "no template at all" or "no `{{tags}}` occurrence anywhere in the template", and say which.

### [BLOCKING:scope] CLI help text absent from the documentation inventory
**Section:** Scope → In (Documentation)
**Issue:** The inventory lists only `README.md`, `docs/servers.yaml.example`, and `docs/scout-multilang.md:17`. But default-on verification makes `assert-no-callers`' own Long help wrong — `internal/cli/cli.go:544-556` ("Use `--within <dir>` when checking an interface method ... Without `--within` ... can report ... a false 'violation'") and the flag help at `:653` ("required for a correct check on an interface method") — and `--build-tags` / `--no-verify` need help strings on the verbs. `filter-ordering` cites that same help text as authoritative without saying whether it changes.
**Fix:** Add the four verbs' cobra Long/flag help to the documentation inventory, with an explicit disposition for the now-stale `--within` interface-method guidance.

### [NIT:decision] Template renderer's home package left open
**Section:** Technical context → Layering enforcement; Testing (hermetic tier)
**Issue:** Rendering is sited as "`registry` or `query`" in both places with no choice made; both are legal under the enforced DAG (`daemon` imports `registry`; `query` imports both), so the function's package and the test file's location are undetermined.
**Fix:** Name the owning package.

### [NIT:design] Verification boolean's polarity is unspecified
**Section:** `one-connection-verification-entry-point`
**Issue:** "`--no-verify` takes the same entry point with verification disabled, via a boolean on `Options`" — polarity unstated. `query.Options` is re-exported verbatim as the public `quarry.Options` (`quarry/facade.go:41`), so a `Verify bool` makes the fail-open state the struct's zero value for every non-CLI caller.
**Fix:** State the field name/polarity so the zero value is the fail-closed default.

### [NIT:consistency] Stale r2 Q&A answer on `timedOut` teardown
**Section:** Q&A log, "(review r2) Does a verification-phase deadline set `lookup`'s `timedOut` flag?"
**Issue:** It answers "Yes — hard-kill on teardown", which `sequential-verification` later corrects: `teardownConnection` (`internal/quarryengine/query/refs.go:137-157`) does not kill or close for `ConnKindSupervised`, the dominant Go path. The r1 concurrency entry is explicitly struck through when superseded; this one is not.
**Fix:** Annotate or reword it the way the superseded r1 entry already is.

### [NIT:consistency] "the new entry field on every built-in block"
**Section:** Scope → In (Documentation)
**Issue:** `docs/servers.yaml.example` shows optional Go-only fields on the `go:` block only — `pinned_version` / `has_native_daemon` appear nowhere else (`:19-45`) — so adding `initialization_options` to all five blocks breaks the file's own omit-when-zero convention.
**Fix:** Say whether the field appears on `go:` only (with a header note) or on every block.

## Verdict

REQUEST_CHANGES
Two template-semantics contradictions and an incomplete documentation inventory must be resolved first.
MILL_REVIEW_END
