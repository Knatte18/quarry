# Review: Facade + CLI, toc (T5a)

```yaml
verdict: APPROVE
reviewer_model: orchestrator
reviewed_file: _mill/discussion.md
date: 2026-09-04
```

## Findings

### [NIT:consistency] Plan §5 still spells the verb `toc <dir|file>...`
**Section:** `single-target`
**Issue:** The decision (sound, well-grounded in §4's one-answer-shape argument, alternatives recorded) leaves `docs/rewrite-plan.md` §5's multi-target spelling standing, and the Out list forbids this task from touching that file — so after merge the plan text will claim an arity the CLI answers with exit 2.
**Suggested fix:** No change in T5a. The orchestrator amends plan §5's spelling (or annotates it) on the hub as a one-line follow-up; the plan writer need only carry the single-target contract as decided.

### [NIT:consistency] Cap-semantics claim overstates what mill-start does at the cap
**Section:** Constraints, "Review rounds are capped at 3"
**Issue:** "On reaching the cap the loop approves implicitly and proceeds to handoff; it never blocks" echoes the config comment, but mill-start's documented cap behaviour is a non-progress check with one possible extension round and otherwise set-blocked-and-halt — the config cap changes the round count, not the cap's exit semantics.
**Suggested fix:** Trim the bullet to "Review rounds are capped at 3 for the discussion and plan holistic loops" and drop the never-blocks sentence; the loop's own convergence handling is not this task's contract.

## Verdict

APPROVE
Every §4/engine/before-side claim checked against source held up; both findings are recorded follow-ups, neither blocks plan writing.
