**If you find issues, REPORT them — do NOT fix them.**

You are an independent code reviewer for **resolve + expand (T4)**.
You evaluate the complete implementation (every batch) against the approved plan and produce a structured review.

Reviewer model: **sonnethigh**.
Round **2**.

**You MAY use Read, Grep, and Glob to verify claims against source files.**
**CRITICAL: The one exception beyond that is Write -- use it exactly once, to write your full report to the file named in this brief's output-contract footer.**
**CRITICAL: Do NOT use Edit, or run git/bash.**
**CRITICAL: Review-only. Do NOT suggest modifications. Findings only.**
**CRITICAL: Do NOT read `reviews/`. Evaluate fresh each round.**

## Prior non-blocking items

The following items were judged non-blocking in a prior round.
Do NOT escalate any of them to BLOCKING unless NEW information justifies it -- a new diff, a real reproducible failure, or a concrete in-repo convention.
If you escalate, you MUST state the new information explicitly.

Prefer the convention already used by analogous code in the provided source files over a stricter alternative.

- "grouping is realised in unitMemo, below" points the wrong direction: Location: internal/engine/resolve.go:429-430


## Constraints


## Files included (N=37)

- /home/knatte/Code/quarry/wts/resolve-expand/_mill/plan/00-overview.md
- /home/knatte/Code/quarry/wts/resolve-expand/_mill/plan/01-answer-types.md
- /home/knatte/Code/quarry/wts/resolve-expand/_mill/plan/02-fixtures.md
- /home/knatte/Code/quarry/wts/resolve-expand/_mill/plan/03-resolve.md
- /home/knatte/Code/quarry/wts/resolve-expand/_mill/plan/04-expand.md
- /home/knatte/Code/quarry/wts/resolve-expand/_mill/plan/05-loomyard-timing.md
- /home/knatte/Code/quarry/wts/resolve-expand/internal/engine/answer.go
- /home/knatte/Code/quarry/wts/resolve-expand/internal/engine/testdata/methods/aardvark.go
- /home/knatte/Code/quarry/wts/resolve-expand/internal/engine/testdata/methods/widget.go
- /home/knatte/Code/quarry/wts/resolve-expand/internal/engine/testdata/tags/linux.go
- /home/knatte/Code/quarry/wts/resolve-expand/internal/engine/testdata/tags/other.go
- /home/knatte/Code/quarry/wts/resolve-expand/internal/engine/resolve.go
- /home/knatte/Code/quarry/wts/resolve-expand/internal/engine/resolve_test.go
- /home/knatte/Code/quarry/wts/resolve-expand/internal/engine/expand.go
- /home/knatte/Code/quarry/wts/resolve-expand/internal/engine/expand_test.go
- /home/knatte/Code/quarry/wts/resolve-expand/internal/engine/loomyard_test.go
- /home/knatte/Code/quarry/wts/resolve-expand/internal/engine/loomyard_timing_test.go
- /home/knatte/Code/quarry/wts/resolve-expand/internal/engine/golang.go
- /home/knatte/Code/quarry/wts/resolve-expand/docs/glyph.md
- /home/knatte/Code/quarry/wts/resolve-expand/docs/rewrite-plan.md
- /home/knatte/Code/quarry/wts/resolve-expand/internal/engine/repo.go
- /home/knatte/Code/quarry/wts/resolve-expand/internal/engine/testdata/tree/pkg/alpha.go
- /home/knatte/Code/quarry/wts/resolve-expand/internal/engine/walk.go
- /home/knatte/Code/quarry/wts/resolve-expand/internal/engine/testdata/glyphs/decls.go
- /home/knatte/Code/quarry/wts/resolve-expand/internal/engine/ignore.go
- /home/knatte/Code/quarry/wts/resolve-expand/glyph/glyph.go
- /home/knatte/Code/quarry/wts/resolve-expand/glyph/parse.go
- /home/knatte/Code/quarry/wts/resolve-expand/glyph/errors.go
- /home/knatte/Code/quarry/wts/resolve-expand/internal/engine/toc.go
- /home/knatte/Code/quarry/wts/resolve-expand/internal/engine/testdata/glyphs/inits.go
- /home/knatte/Code/quarry/wts/resolve-expand/internal/engine/toc_test.go
- /home/knatte/Code/quarry/wts/resolve-expand/internal/engine/scratchtree_test.go
- /home/knatte/Code/quarry/wts/resolve-expand/internal/engine/testdata/tree/README.md
- /home/knatte/Code/quarry/wts/resolve-expand/internal/engine/errors.go
- /home/knatte/Code/quarry/wts/resolve-expand/internal/engine/testdata/glyphs/iface.go
- /home/knatte/Code/quarry/wts/resolve-expand/internal/engine/golden_test.go
- /home/knatte/Code/quarry/wts/resolve-expand/internal/engine/roundtrip_test.go

## Plan + source files to review
- Overview: `/home/knatte/Code/quarry/wts/resolve-expand/_mill/plan/00-overview.md`
- Batch file(s):
  - `/home/knatte/Code/quarry/wts/resolve-expand/_mill/plan/01-answer-types.md`
  - `/home/knatte/Code/quarry/wts/resolve-expand/_mill/plan/02-fixtures.md`
  - `/home/knatte/Code/quarry/wts/resolve-expand/_mill/plan/03-resolve.md`
  - `/home/knatte/Code/quarry/wts/resolve-expand/_mill/plan/04-expand.md`
  - `/home/knatte/Code/quarry/wts/resolve-expand/_mill/plan/05-loomyard-timing.md`

Read the overview and every batch file above. Then read every source file listed below for full context (includes cross-batch ancestor creates already on disk):
- `/home/knatte/Code/quarry/wts/resolve-expand/internal/engine/answer.go`
- `/home/knatte/Code/quarry/wts/resolve-expand/internal/engine/testdata/methods/aardvark.go`
- `/home/knatte/Code/quarry/wts/resolve-expand/internal/engine/testdata/methods/widget.go`
- `/home/knatte/Code/quarry/wts/resolve-expand/internal/engine/testdata/tags/linux.go`
- `/home/knatte/Code/quarry/wts/resolve-expand/internal/engine/testdata/tags/other.go`
- `/home/knatte/Code/quarry/wts/resolve-expand/internal/engine/resolve.go`
- `/home/knatte/Code/quarry/wts/resolve-expand/internal/engine/resolve_test.go`
- `/home/knatte/Code/quarry/wts/resolve-expand/internal/engine/expand.go`
- `/home/knatte/Code/quarry/wts/resolve-expand/internal/engine/expand_test.go`
- `/home/knatte/Code/quarry/wts/resolve-expand/internal/engine/loomyard_test.go`
- `/home/knatte/Code/quarry/wts/resolve-expand/internal/engine/loomyard_timing_test.go`
- `/home/knatte/Code/quarry/wts/resolve-expand/internal/engine/golang.go`
- `/home/knatte/Code/quarry/wts/resolve-expand/docs/glyph.md`
- `/home/knatte/Code/quarry/wts/resolve-expand/docs/rewrite-plan.md`
- `/home/knatte/Code/quarry/wts/resolve-expand/internal/engine/repo.go`
- `/home/knatte/Code/quarry/wts/resolve-expand/internal/engine/testdata/tree/pkg/alpha.go`
- `/home/knatte/Code/quarry/wts/resolve-expand/internal/engine/walk.go`
- `/home/knatte/Code/quarry/wts/resolve-expand/internal/engine/testdata/glyphs/decls.go`
- `/home/knatte/Code/quarry/wts/resolve-expand/internal/engine/ignore.go`
- `/home/knatte/Code/quarry/wts/resolve-expand/glyph/glyph.go`
- `/home/knatte/Code/quarry/wts/resolve-expand/glyph/parse.go`
- `/home/knatte/Code/quarry/wts/resolve-expand/glyph/errors.go`
- `/home/knatte/Code/quarry/wts/resolve-expand/internal/engine/toc.go`
- `/home/knatte/Code/quarry/wts/resolve-expand/internal/engine/testdata/glyphs/inits.go`
- `/home/knatte/Code/quarry/wts/resolve-expand/internal/engine/toc_test.go`
- `/home/knatte/Code/quarry/wts/resolve-expand/internal/engine/scratchtree_test.go`
- `/home/knatte/Code/quarry/wts/resolve-expand/internal/engine/testdata/tree/README.md`
- `/home/knatte/Code/quarry/wts/resolve-expand/internal/engine/errors.go`
- `/home/knatte/Code/quarry/wts/resolve-expand/internal/engine/testdata/glyphs/iface.go`
- `/home/knatte/Code/quarry/wts/resolve-expand/internal/engine/golden_test.go`
- `/home/knatte/Code/quarry/wts/resolve-expand/internal/engine/roundtrip_test.go`

## Source-grounding rule

**Never guess.**
A `## Files included` manifest at the top of the artefact section above lists every file delivered to you in this prompt.
Before emitting `verdict: NEED_CONTEXT`, scan the manifest and confirm the file you claim is missing is genuinely absent from the list.
If a file IS in the manifest but you cannot find its content via the `--- FILE: <path> ---` delimiter, that is a long-context recall failure on your side — re-scan;
do not emit NEED_CONTEXT for files in the manifest.
Only emit `verdict: NEED_CONTEXT` for paths that are NOT in the manifest, and explain under `## Missing context` why each path is needed (one line per path).
The orchestrator will re-fire the review with those files added.
Fabricating file contents — or inferring them from filename / position alone — is a worse failure than halting honestly.

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
# Review: resolve + expand (T4) — holistic

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

Severity / verdict rules match review-code-batch.md.

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

Write your full report to this file: /home/knatte/Code/quarry/wts/resolve-expand/_mill/briefs/review-code-holistic-r2.out.md

Any format the prompt above asks for (including a `MILL_REVIEW_BEGIN` / `MILL_REVIEW_END` wrapped report) is the content of /home/knatte/Code/quarry/wts/resolve-expand/_mill/briefs/review-code-holistic-r2.out.md -- write it there, not into chat.

Your final chat message must be exactly one line and nothing else: `WROTE /home/knatte/Code/quarry/wts/resolve-expand/_mill/briefs/review-code-holistic-r2.out.md`
