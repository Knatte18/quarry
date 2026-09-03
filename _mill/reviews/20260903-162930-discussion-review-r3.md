MILL_REVIEW_BEGIN
# Review: Ladder harness around headless claude -p (T2)

```yaml
duration_s: 267.0
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus (Anthropic); environment reports model id claude-opus-5
reviewed_file: _mill/discussion.md
date: 2026-09-03
```

## Findings

### [NIT:decision] Prompt constants named but never sourced
**Demoted-from:** BLOCKING
**Section:** one-preamble-for-every-cell
**Issue:** `PARALLEL_OPENING` and `PARALLEL_BLOCK` are the two largest parts of the prompt, yet they appear nowhere else in the discussion or tree, `preamble.go` is absent from Technical context (which gives file:line for every other verbatim port), and their disposition is only implied by the rejected alternative.
**Fix:** State the V1 path and line range for each constant and whether it is ported verbatim, and say explicitly whether the A/B/C parallel-arm framing survives a design where cells run one at a time.

### [BLOCKING:design] Answer extraction from the transcript is undefined
**Section:** scorer / resume-and-failure / results-raw-ignored
**Issue:** `answer.json` and `answer.redacted.json` are named outputs and "missing answer block" is a named failure, but nothing says how the harness locates the answer in the cell's final assistant text (fenced block? last JSON object? schema-key match?), and Testing names no test for it — `fenced.go`'s non-dispatch half is neither listed for porting nor described.
**Fix:** Specify the extraction rule and its failure semantics, cite the V1 source if it is being ported, and add it to the TDD list.

### [NIT:consistency] `mcp__quarry__` hardcoded while server name is yaml data
**Demoted-from:** BLOCKING
**Section:** server-block vs metrics, gates (check a), scorer
**Issue:** `server:` declares `name:` with "tool names become `mcp__<name>__<tool>`", but `quarry_tool_uses`, gate 2's check (a) and the scorer redaction all hardcode the literal `mcp__quarry__`; a differently named server silently zeroes gate 1 and makes the blinding check miss real leakage.
**Fix:** State that the prefix is derived from `server.name` everywhere (or that `name` is fixed and the field is decorative), in one place all three consumers reference.

### [BLOCKING:design] Harness scratch path carries both tokens the gate kills on
**Section:** no-tmp-paths vs gates (checks b and c)
**Issue:** The two startup assertions cover only the worktree root, but `<quarry-repo>/.scratch/ladder/` — which contains the repo root and a `quarry` token by construction — is passed into every cell as `--mcp-config <path>`, including controls; the probe established `mcp_servers: []`, not that the config path is absent from the marshalled transcript.
**Fix:** State whether the mcp-config path (and any other quarry-rooted argument) can reach the stream, and if unproven, move generated per-cell configs beside the worktree root or probe and record the result.

### [BLOCKING:design] Gate 2's carve-out misses the agent's own restatement
**Section:** gates (check c)
**Issue:** The rationale concedes Loomyard's tracked prose contains "quarry", and the carve-out spares only `tool_result` payloads — so a control agent that quotes or paraphrases such a file in its own assistant text fatally fails the rep, and resume-and-failure then pays for three more reps of the same deterministic failure before recording `incomplete[]`.
**Fix:** Decide how a target-origin `quarry` mention re-emitted in assistant text is classified, and whether a gate-2 failure is retried at all or fails the rep once.

### [NIT:decision] yaml dependency and module boundary left to prose
**Section:** Technical context ("`gopkg.in/yaml.v3` is the obvious choice")
**Issue:** The only new external dependency is chosen in hedged prose with no decision block, and the discussion never says whether the harness joins the root module — HANDOFF §1 describes that `go.mod` as tree-sitter and nothing else.
**Fix:** Make the module placement and the yaml library an explicit decision with the rejected alternatives.

### [NIT:consistency] HANDOFF §2 quoted more broadly than written
**Section:** one-ladder-file rationale
**Issue:** HANDOFF §2 says "Nothing in the tree describes a *language* it does not support"; the discussion generalises it to "something it does not support" and rests the five-file deletion on the wider reading.
**Fix:** Quote §2 as written and justify the deletions on their own terms.

## Verdict

REQUEST_CHANGES
Prompt text, answer extraction, tool prefix and two gate-2 false-positive paths remain unsettled.
_Note: 2 finding(s) demoted from BLOCKING to NIT by the stage's blocking-class ceiling; current blocking_count is 3._
MILL_REVIEW_END
