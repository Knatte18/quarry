# Section 9a live probe — operator report

This file is an **operator report**, not a transcript this task captured — no such transcript
exists anywhere in this tree to copy. It transcribes the answer that the round-1 discussion review
supplied when this task's own constraint asked whether the plan's section 9a live probe had been
reported done.

## What was probed

The merged `cmd/quarry-mcp` MCP server, built from the parent branch (`main`).

## Date

2026-09-04.

## Outcomes of the three section 9a properties

Written strictly to what the quoted source below states, and no further:

1. **Server connected.** The report states the server connected. It does **not** state that the
   connected server was listed in the measured session's `system` record — that half of this
   property is therefore recorded here as **not covered** by the operator's report, rather than
   asserted, because asserting it would be this card generating evidence rather than transcribing a
   report.
2. **`mcp__quarry__toc` call returned the section 4 envelope.** Yes — stated by the report, under
   the allowlist.
3. **A call outside the allowlist was refused and appeared in `permission_denials`.** Yes — stated
   by the report, without the allowlist.

## The flags that made the denial half meaningful

`--setting-sources ""` in particular. Without it, the operator's global `defaultMode: "auto"`
auto-approves read-only MCP calls, and the denial half of the probe would silently test nothing.
`bench/loomyard-eval/ladder/internal/ladder/run.go` passes this same flag for every measured
repetition (confirmed by reading the file for this card; the file was not changed), which is what
makes the operator's hand probe harness-faithful rather than a different test than the one the
matrix itself runs under.

## `claude` CLI version

The operator's report, quoted below, does not state the `claude` CLI version the probe ran under.
This is recorded here as **not recorded**, rather than silently omitted, because plan section 9a's
own probe table was taken on `2.1.259` while this host is on `2.1.236` — a silent omission would
read as an agreement this report has not earned, so the discrepancy is stated plainly: the version
the probe ran under is unknown, and it may or may not match either of those two.

## Source

The task constraint reads: the operator's section 9a live probe runs before the matrix; if it has
not been reported done, stop and ask. The answer arrived through discussion review round 1: it was
reported done. The verbatim answer, quoted below, is the `[NIT:consistency]` finding titled
"probe-before-matrix overrides the task's stop-and-ask constraint on a false premise" from that
review's **Issue:** paragraph, attributed to
`archive/ladder-toc-rerun:_mill/reviews/20260904-110823-discussion-review-r1.md` (the tag form,
since this task merges as one squash commit and its task-state tree survives only under that
archive tag — a bare `_mill/` path would be unresolvable on the parent branch):

> The task constraint reads "The operator's section 9a live probe ... runs before the matrix; **if
> it has not been reported done, stop and ask**." The discussion instead self-authorizes re-running
> the probe ("Rejected: halting and asking the orchestrator") and rewrites its own Constraints
> bullet to "This task runs it itself" — the artefact carries an altered statement of a task
> constraint. The premise "no evidence exists" is also false at the orchestrator level: **the §9a
> probe IS reported done** — run by the operator on 2026-09-04 against the merged `cmd/quarry-mcp`
> built from `main`, harness-faithfully: connect + `mcp__quarry__toc` returned the §4 envelope
> under the allowlist, and without the allowlist the call was refused and landed in
> `permission_denials`. (That run required `--setting-sources ""`, exactly as `run.go:586` passes
> it; the operator's global `defaultMode: "auto"` otherwise auto-approves read-only MCP calls. The
> discussion's planned manual probe names `--mcp-config` + `--strict-mcp-config` but omits
> `--setting-sources ""`, so as specified its denial half would misbehave anyway.)
