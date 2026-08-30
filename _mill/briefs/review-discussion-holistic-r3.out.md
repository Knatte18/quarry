MILL_REVIEW_BEGIN
# Review: Rethink quarry-mcp's per-call targetDir ergonomics

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: claude-opus-5 (Anthropic Opus-class model, per runtime environment)
reviewed_file: _mill/discussion.md
date: 2026-08-30
```

## Findings

### [BLOCKING:consistency] Q&A log contradicts what the bench gate proves
**Section:** Q&A log (last entry) vs `### bench-ladder-update` → "What the gate is not"
**Issue:** The Q&A says the gate "becomes a live end-to-end assertion that the property is gone", while the decision body says it is explicitly **not** a schema check (it reads transcript `tool_input` only) and that its `targetDir` arm becomes "close to unreachable" — verified: `gates.py:117-136` iterates `iter_tool_use_blocks(events)` and never touches a published schema.
**Fix:** Correct the Q&A entry to match the decision body, so no plan writer treats the gate as a substitute for the `internal/mcpserver` schema regression test.

### [BLOCKING:design] Reworded bench prompt line left undefined
**Section:** `### bench-ladder-update`; Scope bullet on `test_ladder_config.py:311`
**Issue:** `ladder_config.py:383-384` is `Never set targetDir or buildTags on any of these calls -- the server is already rooted at the correct target codebase.`; dropping the `targetDir` half orphans a rationale clause that only justifies `targetDir`, and the discussion never states the resulting literal that `test_ladder_config.py:311`'s assertion must be narrowed to.
**Fix:** State the exact replacement sentence (including its rationale clause) and the exact literal the test asserts.

### [NIT:consistency] `schema_test.go` citation is weaker than described
**Section:** `### hard-removal-not-graceful-ignore`; Testing
**Issue:** The named test is `TestInputSchemaFor_CallWidePropertySurvives` (not `...CallWideAdditionalPropertiesUntouched`), it asserts only `s.AdditionalProperties != nil` on the local `fixtureCall` fixture, and no existing test shows an unknown call-wide key failing a real tool call at the transport.
**Fix:** Cite the real test name and note the behavioural claim is only proven by the new transport case, not inherited assertion.

### [NIT:design] Regression test's file and mechanism disagree
**Section:** Testing, first paragraph
**Issue:** The test is placed in `schema_test.go` (which only exercises `inputSchemaFor[T]` over fixtures) yet is told to drive off "the real registration path"; `transport_test.go:167` `TestToolsList_PerToolParameterMatrix` plus its `schemaProperties` helper already asserts call-wide property absence across all seven registered tools via `ListTools`.
**Fix:** Name whether the new assertion extends that existing matrix test or lives in `schema_test.go` using `newConnectedPair`.

### [NIT:scope] Entry-field doc comments still say "the call's targetDir"
**Section:** Scope bullet 4; "Gotcha — stale doc comments"
**Issue:** The gotcha names three sites, and Scope says concept-level Go comments stay, but `nativeentry.go:29,43,45` and `lspentry.go:19,54` read "relative to **the call's** targetDir" — per-call phrasing that goes stale exactly like the three listed.
**Fix:** State whether "the call's targetDir" phrasing counts as an intentional survivor or must be reworded.

### [NIT:scope] Ladder pytest verification command unnamed
**Section:** Testing (bench ladder paragraph); Verification
**Issue:** Verification names only `go build ./...`, `go test ./...`, and a grep; the ladder suite run has no command, and `ladder/README.md:361` documents `uv run --no-project --with pytest --with pyyaml python -m pytest bench/loomyard-eval/ladder/tests -q` as the invocation (`conftest.py` handles sys.path; no `PYTHONPATH` prefix).
**Fix:** Name the README's invocation in Verification.

## Verdict

REQUEST_CHANGES
Gate-claim self-contradiction and an undefined bench prompt/test literal must be resolved first.
MILL_REVIEW_END
