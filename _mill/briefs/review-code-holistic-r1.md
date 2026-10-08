**If you find issues, REPORT them — do NOT fix them.**

You are an independent code reviewer for **Enclose: path:line ranges -> enclosing member glyphs (batch, at revision, CLI verb)**.
You evaluate the complete implementation (every batch) against the approved plan and produce a structured review.

Reviewer model: **sonnethigh**.
Round **1**.

**You MAY use Read, Grep, and Glob to verify claims against source files.**
**CRITICAL: The one exception beyond that is Write -- use it exactly once, to write your full report to the file named in this brief's output-contract footer.**
**CRITICAL: Do NOT use Edit, or run git/bash.**
**CRITICAL: Review-only. Do NOT suggest modifications. Findings only.**
**CRITICAL: Do NOT read `reviews/`. Evaluate fresh each round.**

## Writing style

State the point first — no preamble, and no restating a point before making it.
Cut empty intensifiers ("actually", "really", "simply", "just", "completely"): remove the word, and if the sentence still means the same thing it was padding.
Say each thing once; do not restate it in a summary or a closing recap.
Do not narrate what is already visible in the quoted code or the surrounding context.
Don't pin a perishable specific — a tally, a list of the current callers of a symbol, or a name cited descriptively rather than as a stable identifier: name the source, not the snapshot.
Apply a per-sentence cut test: would the reader act differently if this sentence were missing? If not, cut it.
For any multi-line prose written into a file, use semantic line breaks — one sentence per line, never fixed-column hard-wrap, plain newlines only (never a trailing double-space or backslash).

## Prior non-blocking items

The following items were judged non-blocking in a prior round.
Do NOT escalate any of them to BLOCKING unless NEW information justifies it -- a new diff, a real reproducible failure, or a concrete in-repo convention.
If you escalate, you MUST state the new information explicitly.

Prefer the convention already used by analogous code in the provided source files over a stricter alternative.

(none)

## Constraints


## Path roots

Every unprefixed path below is relative to `/home/knatte/Code/quarry/wts/enclose-locations`.
- `wiki/` paths are relative to `/home/knatte/Code/quarry/wiki`

## Files included (N=52)

- _mill/plan/00-overview.md
- _mill/plan/01-engine-enclose.md
- _mill/plan/02-facade-enclose.md
- _mill/plan/03-cli-enclose.md
- internal/engine/memo.go
- internal/engine/memo_test.go
- internal/engine/answer.go
- internal/engine/enclose.go
- internal/engine/enclose_test.go
- internal/engine/toc.go
- internal/engine/enclose_worktree_test.go
- internal/engine/enclose_revision_test.go
- internal/repopath/target.go
- internal/repopath/target_test.go
- internal/repopath/doc.go
- quarry/quarry.go
- quarry/quarry_test.go
- quarry/render.go
- quarry/render_test.go
- quarry/repo.go
- quarry/enclose.go
- quarry/enclose_test.go
- internal/cli/flags.go
- internal/cli/flags_test.go
- internal/cli/cli.go
- internal/cli/usage.go
- internal/cli/doc.go
- internal/cli/enclose_test.go
- internal/cli/enclose_golden_test.go
- internal/cli/testdata/enclose/batch.json
- _mill/discussion.md
- internal/engine/walk.go
- internal/engine/scratchtree_test.go
- internal/engine/toc_test.go
- internal/engine/name.go
- internal/engine/name_test.go
- internal/engine/resolve.go
- internal/engine/repo.go
- internal/engine/extension.go
- internal/engine/strategy.go
- glyph/self.go
- glyph/glyph.go
- internal/engine/units.go
- internal/mcpserver/toc.go
- internal/gitsrc/gitsrc.go
- quarry/delta.go
- quarry/delta_test.go
- quarry/scratchtree_test.go
- internal/cli/cli_test.go
- internal/cli/scratchtree_test.go
- internal/cli/name_golden_test.go
- internal/cli/loomyard_test.go

## Plan + source files to review
- Overview: `_mill/plan/00-overview.md`
- Batch file(s):
  - `_mill/plan/01-engine-enclose.md`
  - `_mill/plan/02-facade-enclose.md`
  - `_mill/plan/03-cli-enclose.md`

Read the overview and every batch file above. Then read every source file listed below for full context (includes cross-batch ancestor creates already on disk):
- `internal/engine/memo.go`
- `internal/engine/memo_test.go`
- `internal/engine/answer.go`
- `internal/engine/enclose.go`
- `internal/engine/enclose_test.go`
- `internal/engine/toc.go`
- `internal/engine/enclose_worktree_test.go`
- `internal/engine/enclose_revision_test.go`
- `internal/repopath/target.go`
- `internal/repopath/target_test.go`
- `internal/repopath/doc.go`
- `quarry/quarry.go`
- `quarry/quarry_test.go`
- `quarry/render.go`
- `quarry/render_test.go`
- `quarry/repo.go`
- `quarry/enclose.go`
- `quarry/enclose_test.go`
- `internal/cli/flags.go`
- `internal/cli/flags_test.go`
- `internal/cli/cli.go`
- `internal/cli/usage.go`
- `internal/cli/doc.go`
- `internal/cli/enclose_test.go`
- `internal/cli/enclose_golden_test.go`
- `internal/cli/testdata/enclose/batch.json`
- `_mill/discussion.md`
- `internal/engine/walk.go`
- `internal/engine/scratchtree_test.go`
- `internal/engine/toc_test.go`
- `internal/engine/name.go`
- `internal/engine/name_test.go`
- `internal/engine/resolve.go`
- `internal/engine/repo.go`
- `internal/engine/extension.go`
- `internal/engine/strategy.go`
- `glyph/self.go`
- `glyph/glyph.go`
- `internal/engine/units.go`
- `internal/mcpserver/toc.go`
- `internal/gitsrc/gitsrc.go`
- `quarry/delta.go`
- `quarry/delta_test.go`
- `quarry/scratchtree_test.go`
- `internal/cli/cli_test.go`
- `internal/cli/scratchtree_test.go`
- `internal/cli/name_golden_test.go`
- `internal/cli/loomyard_test.go`

Every path listed above is relative to the root stated in the `## Path roots` block above and must be resolved against it before reading.

## Source-grounding rule

**Never guess.**
A `## Files included` manifest at the top of the artefact section above lists every file delivered to you in this prompt.
Before emitting `verdict: NEED_CONTEXT`, scan the manifest and confirm the file you claim is missing is genuinely absent from the list.
If a file IS in the manifest but you cannot find its content via the `--- FILE: <path> ---` delimiter, that is a long-context recall failure on your side — re-scan;
do not emit NEED_CONTEXT for files in the manifest.
Only emit `verdict: NEED_CONTEXT` for paths that are NOT in the manifest, and explain under `## Missing context` why each path is needed (one line per path).
The orchestrator will re-fire the review with those files added.
Fabricating file contents — or inferring them from filename / position alone — is a worse failure than halting honestly.

**Mechanism claims must be source-verified.**
A finding that rests on a claim about how the target repo's production code behaves — which branch executes, what a predicate selects, which value survives a mutation — must name the file and the function/method/construct it was verified against, in the finding's own text.
Do not assert a mechanism claim from memory, naming convention, or plausible-sounding inference.
If you cannot verify the claim against source in your context (not bulked into this prompt, and not Read-able in bulk mode), do not assert it: downgrade the finding to a question under `## Missing context`, or drop it — never write an unverified mechanism claim into a BLOCKING or NIT finding as fact.
Tool-use-mode reviewers may Read/Grep the target repo's source directly to verify a mechanism claim even when the relevant file was not bulked into this prompt; bulk-mode reviewers have no such option and must rely on this rule alone.

## Criteria (apply to the implementation as a whole)

- **End-to-end plan alignment** — every batch's cards are realised;
  every file listed across all batches' `Context:`/`Edits:`/`Creates:` is present in the source files provided.
- **Shared-decisions alignment** — the `## Shared Decisions` subsections are applied consistently across all batches;
  deviation is BLOCKING.
- **Out-of-plan files** — BLOCKING if any source file is present that is not accounted for in any batch's reference lists.
  If the implementer added it, the batch file must have been updated first;
  a review with surprise files means that discipline was skipped somewhere.
- **Cross-batch contracts** — interfaces produced by one batch and consumed by another are compatible.
  Dependency order implied by `depends-on:` is reflected in the code (consumers don't assume behaviour the producer doesn't guarantee).
- **Integration correctness** — the pieces work together, not just per-batch.
  Call sites match signatures;
  shared state is consistently managed;
  error surfaces compose.
- **Global utility duplication** — BLOCKING if two batches independently reimplement the same helper.
  Consolidate into a shared module.
- **Test coverage across the whole surface** — happy paths + errors for every batch's entry point.
  Integration tests reach across batch boundaries where appropriate.
- **Constraint violations** — BLOCKING.
- **Codebase consistency** — naming, error handling, imports, and style match the conventions visible in the source files provided.
- **Language pitfalls** — BLOCKING if high-risk (Python: mutable defaults, import side-effects, Windows path sep, CRLF/LF).

## Output format — STRICT

Wrap your entire output in `MILL_REVIEW_BEGIN` / `MILL_REVIEW_END` markers, each on its own line.
Everything outside these markers is ignored by the backend.
**No preamble inside the markers.**
Per finding: 3–5 lines, short and factual.
Cite file and line, state the issue, propose the fix.

Target length: ~400 tokens for APPROVE, ~800–1500 tokens for REQUEST_CHANGES across multiple batches.
If you produce more than ~1800 tokens, compress.

~~~markdown
MILL_REVIEW_BEGIN
# Review: Enclose: path:line ranges -> enclosing member glyphs (batch, at revision, CLI verb) — holistic

```yaml
verdict: APPROVE | REQUEST_CHANGES | NEED_CONTEXT
reviewer_model: sonnethigh
reviewed_file: plan/ + source
date: <UTC YYYY-MM-DD>
```

## Findings

### [BLOCKING:design] <short title, <60 chars>
**Location:** `path/to/file.py:42` (or `:42-58`)
**Issue:** <one sentence>
**Fix:** <one sentence>

### [NIT:consistency] <short title>
**Location:** `path/to/file.py:N`
**Issue:** <one sentence>
**Fix:** <one sentence>

## Missing context
(include ONLY when verdict is NEED_CONTEXT — omit the section otherwise)

- `path/to/file.py` — <one-line reason the reviewer needs this file>

## Verdict

<APPROVE | REQUEST_CHANGES | NEED_CONTEXT>
<one sentence — max 20 words>
MILL_REVIEW_END
~~~

Severity:
- `BLOCKING` — must fix before the task is approved.
- `NIT` — record but do not block.

Verdict:
- `APPROVE` — zero BLOCKINGs.
- `REQUEST_CHANGES` — one or more BLOCKINGs.
- `NEED_CONTEXT` — one or more missing source files; orchestrator will re-fire.

**Severity vocabulary is closed.**
Use ONLY `BLOCKING` or `NIT` as the bracketed label in a finding heading -- never invent another word (e.g. `MAJOR`, `MINOR`, `CRITICAL`, `MEDIUM`, `HIGH`).
If a finding's severity feels ambiguous, default to `BLOCKING`, never `NIT` -- an over-cautious BLOCKING can be pushed back on by the orchestrator;
a mislabeled NIT (or an unrecognized label) can silently skip review entirely.

**Class is the second axis, encoded in the same bracket as severity, colon-separated, lowercase: `### [BLOCKING:design] <title>`.**
A finding with no class, or a class outside the four names below, is a reviewer defect.
The four recognised classes, identical in meaning across every review stage:

- `design` — a decision is missing, wrong, or rests on a false premise.
  Example: the implementation fixes the symptom at one call site but never resolves which layer owns the validation.
- `scope` — the work inventory is incomplete, or the enumeration method is unreliable.
  Example: a card's `Edits:` file was converted but a sibling file with the identical helper was left unconverted.
- `decision` — a named artifact with no stated disposition.
  Example: a config key the plan introduced is added but never wired into the loader that reads it.
- `consistency` — the artefact contradicts itself, carries a superseded statement, or violates an established repo convention.
  Example: two batches' implementations of the same interface handle the error case differently.

**Class governs who decides and when the loop stops, never whether a finding gets fixed.**

Omit `## Findings` if zero findings.
Never invent findings to pad.

## Out of scope for this stage

- Re-litigating a decision already recorded in `discussion.md` is out of scope unless new evidence contradicts it.


---

## Output contract

Write your full report to this file: /home/knatte/Code/quarry/wts/enclose-locations/_mill/briefs/review-code-holistic-r1.out.md

Any format the prompt above asks for (including a `MILL_REVIEW_BEGIN` / `MILL_REVIEW_END` wrapped report) is the content of /home/knatte/Code/quarry/wts/enclose-locations/_mill/briefs/review-code-holistic-r1.out.md -- write it there, not into chat.

Your final chat message must be exactly one line and nothing else: `WROTE /home/knatte/Code/quarry/wts/enclose-locations/_mill/briefs/review-code-holistic-r1.out.md`
