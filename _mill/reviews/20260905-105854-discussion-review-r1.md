# Review: Glyph self-form and the resolve contract (C1)

```yaml
verdict: APPROVE
reviewer_model: orchestrator
reviewed_file: _mill/discussion.md
date: 2026-09-05
```

## Findings

### [NIT:consistency] D4's decided message text omits what Risks declares required
**Section:** D4 / Risks ("Cwd-relative resolve targets stop working")
**Issue:** D4's rendered line is quoted as `a glyph needs a "#"; a path is addressed as its own
glyph by appending one (internal/logger#)`, but the Risks section states — as "a required detail
of D4's message wording, not an optional polish" — that the message must say the appended form is
repository-relative, since `Detail` echoes the caller's cwd-local spelling verbatim and would
otherwise mislead exactly the caller it is aimed at.
**Suggested fix:** The plan should treat the Risks wording as the authoritative version of D4's
message text (e.g. `…by appending one to its repository-relative form (internal/logger#)`), and
the one message-shape test in Testing item 2 should pin that sentence, not D4's quoted draft.

## Verdict

APPROVE
All four settled contract decisions are carried intact with rationale and rejected alternatives; code claims verified against the worktree (isGlyphTarget, ReasonMemberEmpty, RepoRelPath's sole caller, the `dir` field, `String()`'s self-form output, the sixteen-reason count, the existing goldens, MCP's RepoRelTarget path); scope, testing and the D8 tension are recorded honestly — nothing blocks plan writing.
