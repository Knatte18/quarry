# Review: MCP, thin (T6)

```yaml
verdict: APPROVE
reviewer_model: orchestrator
reviewed_file: _mill/discussion.md
date: 2026-09-04
```

## Findings

### [NIT:consistency] Testing cites an internal/cli testdata/ that does not exist
**Section:** Testing, "the happy path (golden)"
**Issue:** "reuse or mirror whatever `internal/cli` already has under `testdata/`" — `internal/cli` has no `testdata/` directory; its fixture-needing tests build scratch trees programmatically via `writeScratchTree` (`internal/cli/scratchtree_test.go`, writing under gitignored `.scratch/cli-tests/`), and the only committed fixture trees are `internal/engine/testdata` and the ladder's own.
**Suggested fix:** Point the plan at the real precedents — a small committed fixture in the engine's `testdata/` style, or `writeScratchTree`-style construction — and let it pick one explicitly.

### [NIT:decision] Repo open-once vs per-call left as a preference with a sanctioned fallback
**Section:** Technical context, "The pipeline to mirror"
**Issue:** "the server should open it once at startup instead … but if that turns out to complicate the not-found path, opening per call is also correct" names two acceptable designs without binding one, so two plan writers could produce different handlers.
**Suggested fix:** Have the plan pin one (open-once is the stated preference and `Repo` is documented concurrency-safe) and treat the fallback as a recorded contingency, not an open choice.

## Verdict

APPROVE
Every load-bearing claim checked out against the tree (facade surface, `TOCOptions.Symbols *bool`, `DepthAll = -1`, `ladder-toc.yaml`'s name/tool/no-args contract, `MCPPrefix`, `PermissionDenials`, uncaptured stderr, V1's SDK v1.7.0 and its `"all"`-sentinel cost note); scope is tightly fenced to one tool and the two NITs are plan-level details.
