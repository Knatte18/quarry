**If you find issues, REPORT them — do NOT fix them.**

You are an independent plan reviewer for **Add file/dir toc verbs (Tree-sitter-backed)**.
You evaluate the complete plan (all batches) and produce a structured review.

Reviewer model: **opushigh**.
Round **7**.

**You MAY use Read, Grep, and Glob to verify claims against source files.**
**CRITICAL: The one exception beyond that is Write -- use it exactly once, to write your full report to the file named in this brief's output-contract footer.**
**CRITICAL: Do NOT use Edit, or run git/bash.**
**CRITICAL: Review-only. Do NOT suggest modifications. Findings only.**
**CRITICAL: Do NOT read `reviews/`. Evaluate fresh each round.**

## Constraints


## Files included (N=46)

- /home/knatte/Code/quarry/wts/toc-verbs/_mill/plan/00-overview.md
- /home/knatte/Code/quarry/wts/toc-verbs/_mill/plan/01-treesitter-backend.md
- /home/knatte/Code/quarry/wts/toc-verbs/_mill/plan/02-toc-scaffolding.md
- /home/knatte/Code/quarry/wts/toc-verbs/_mill/plan/03-go-strategy.md
- /home/knatte/Code/quarry/wts/toc-verbs/_mill/plan/04-python-csharp-strategies.md
- /home/knatte/Code/quarry/wts/toc-verbs/_mill/plan/05-toc-entry-points.md
- /home/knatte/Code/quarry/wts/toc-verbs/_mill/plan/06-facade-and-cli.md
- /home/knatte/Code/quarry/wts/toc-verbs/_mill/plan/07-doc-sentences-config.md
- /home/knatte/Code/quarry/wts/toc-verbs/_mill/plan/08-docs-and-sweep.md
- /home/knatte/Code/quarry/wts/toc-verbs/go.mod
- /home/knatte/Code/quarry/wts/toc-verbs/go.sum
- /home/knatte/Code/quarry/wts/toc-verbs/internal/quarryengine/layering_test.go
- /home/knatte/Code/quarry/wts/toc-verbs/internal/quarryengine/errors.go
- /home/knatte/Code/quarry/wts/toc-verbs/internal/quarryengine/seam_enforcement_test.go
- /home/knatte/Code/quarry/wts/toc-verbs/quarry/facade.go
- /home/knatte/Code/quarry/wts/toc-verbs/quarry/facade_test.go
- /home/knatte/Code/quarry/wts/toc-verbs/internal/cli/cli.go
- /home/knatte/Code/quarry/wts/toc-verbs/internal/cli/paths.go
- /home/knatte/Code/quarry/wts/toc-verbs/README.md
- /home/knatte/Code/quarry/wts/toc-verbs/internal/quarryengine/doc.go
- /home/knatte/Code/quarry/wts/toc-verbs/mill-config.yaml
- /home/knatte/Code/quarry/wts/toc-verbs/.scratch/cgobench/go.mod
- /home/knatte/Code/quarry/wts/toc-verbs/internal/quarryengine/log.go
- /home/knatte/Code/quarry/wts/toc-verbs/internal/quarryengine/errors.go
- /home/knatte/Code/quarry/wts/toc-verbs/internal/quarryengine/registry/registry_test.go
- /home/knatte/Code/quarry/wts/toc-verbs/internal/quarryengine/registry/registry.go
- /home/knatte/Code/quarry/wts/toc-verbs/internal/quarryengine/registry/detect.go
- /home/knatte/Code/quarry/wts/toc-verbs/internal/quarryengine/registry/detect_test.go
- /home/knatte/Code/quarry/wts/toc-verbs/internal/quarryengine/doc.go
- /home/knatte/Code/quarry/wts/toc-verbs/internal/quarryengine/query/refs.go
- /home/knatte/Code/quarry/wts/toc-verbs/internal/output/output.go
- /home/knatte/Code/quarry/wts/toc-verbs/quarry/facade.go
- /home/knatte/Code/quarry/wts/toc-verbs/internal/cli/cli.go
- /home/knatte/Code/quarry/wts/toc-verbs/internal/cli/exec.go
- /home/knatte/Code/quarry/wts/toc-verbs/internal/cli/cwdcontext.go
- /home/knatte/Code/quarry/wts/toc-verbs/internal/cli/cli_test.go
- /home/knatte/Code/quarry/wts/toc-verbs/internal/quarryengine/registry/load.go
- /home/knatte/Code/quarry/wts/toc-verbs/internal/cli/paths.go
- /home/knatte/Code/quarry/wts/toc-verbs/internal/cli/paths_test.go
- /home/knatte/Code/quarry/wts/toc-verbs/docs/scout-multilang.md
- /home/knatte/Code/quarry/wts/toc-verbs/go.mod
- /home/knatte/Code/quarry/wts/toc-verbs/quarry/facade_test.go
- /home/knatte/Code/quarry/wts/toc-verbs/internal/quarryengine/layering_test.go
- /home/knatte/Code/quarry/wts/toc-verbs/internal/quarryengine/seam_enforcement_test.go
- /home/knatte/Code/quarry/wts/toc-verbs/README.md
- /home/knatte/Code/quarry/wts/toc-verbs/docs/scout-vs-grep.md

## Plan files to review
- Overview: `/home/knatte/Code/quarry/wts/toc-verbs/_mill/plan/00-overview.md`
- Batches:
- `/home/knatte/Code/quarry/wts/toc-verbs/_mill/plan/01-treesitter-backend.md`
- `/home/knatte/Code/quarry/wts/toc-verbs/_mill/plan/02-toc-scaffolding.md`
- `/home/knatte/Code/quarry/wts/toc-verbs/_mill/plan/03-go-strategy.md`
- `/home/knatte/Code/quarry/wts/toc-verbs/_mill/plan/04-python-csharp-strategies.md`
- `/home/knatte/Code/quarry/wts/toc-verbs/_mill/plan/05-toc-entry-points.md`
- `/home/knatte/Code/quarry/wts/toc-verbs/_mill/plan/06-facade-and-cli.md`
- `/home/knatte/Code/quarry/wts/toc-verbs/_mill/plan/07-doc-sentences-config.md`
- `/home/knatte/Code/quarry/wts/toc-verbs/_mill/plan/08-docs-and-sweep.md`

Read the overview and every batch listed above. Then read the source files referenced across all batches:
- `/home/knatte/Code/quarry/wts/toc-verbs/go.mod`
- `/home/knatte/Code/quarry/wts/toc-verbs/go.sum`
- `/home/knatte/Code/quarry/wts/toc-verbs/internal/quarryengine/layering_test.go`
- `/home/knatte/Code/quarry/wts/toc-verbs/internal/quarryengine/errors.go`
- `/home/knatte/Code/quarry/wts/toc-verbs/internal/quarryengine/seam_enforcement_test.go`
- `/home/knatte/Code/quarry/wts/toc-verbs/quarry/facade.go`
- `/home/knatte/Code/quarry/wts/toc-verbs/quarry/facade_test.go`
- `/home/knatte/Code/quarry/wts/toc-verbs/internal/cli/cli.go`
- `/home/knatte/Code/quarry/wts/toc-verbs/internal/cli/paths.go`
- `/home/knatte/Code/quarry/wts/toc-verbs/README.md`
- `/home/knatte/Code/quarry/wts/toc-verbs/internal/quarryengine/doc.go`
- `/home/knatte/Code/quarry/wts/toc-verbs/mill-config.yaml`
- `/home/knatte/Code/quarry/wts/toc-verbs/.scratch/cgobench/go.mod`
- `/home/knatte/Code/quarry/wts/toc-verbs/internal/quarryengine/log.go`
- `/home/knatte/Code/quarry/wts/toc-verbs/internal/quarryengine/errors.go`
- `/home/knatte/Code/quarry/wts/toc-verbs/internal/quarryengine/registry/registry_test.go`
- `/home/knatte/Code/quarry/wts/toc-verbs/internal/quarryengine/registry/registry.go`
- `/home/knatte/Code/quarry/wts/toc-verbs/internal/quarryengine/registry/detect.go`
- `/home/knatte/Code/quarry/wts/toc-verbs/internal/quarryengine/registry/detect_test.go`
- `/home/knatte/Code/quarry/wts/toc-verbs/internal/quarryengine/doc.go`
- `/home/knatte/Code/quarry/wts/toc-verbs/internal/quarryengine/query/refs.go`
- `/home/knatte/Code/quarry/wts/toc-verbs/internal/output/output.go`
- `/home/knatte/Code/quarry/wts/toc-verbs/quarry/facade.go`
- `/home/knatte/Code/quarry/wts/toc-verbs/internal/cli/cli.go`
- `/home/knatte/Code/quarry/wts/toc-verbs/internal/cli/exec.go`
- `/home/knatte/Code/quarry/wts/toc-verbs/internal/cli/cwdcontext.go`
- `/home/knatte/Code/quarry/wts/toc-verbs/internal/cli/cli_test.go`
- `/home/knatte/Code/quarry/wts/toc-verbs/internal/quarryengine/registry/load.go`
- `/home/knatte/Code/quarry/wts/toc-verbs/internal/cli/paths.go`
- `/home/knatte/Code/quarry/wts/toc-verbs/internal/cli/paths_test.go`
- `/home/knatte/Code/quarry/wts/toc-verbs/docs/scout-multilang.md`
- `/home/knatte/Code/quarry/wts/toc-verbs/go.mod`
- `/home/knatte/Code/quarry/wts/toc-verbs/quarry/facade_test.go`
- `/home/knatte/Code/quarry/wts/toc-verbs/internal/quarryengine/layering_test.go`
- `/home/knatte/Code/quarry/wts/toc-verbs/internal/quarryengine/seam_enforcement_test.go`
- `/home/knatte/Code/quarry/wts/toc-verbs/README.md`
- `/home/knatte/Code/quarry/wts/toc-verbs/docs/scout-vs-grep.md`

## Source-grounding rule

**Never guess.**
A `## Files included` manifest at the top of the artefact section above lists every file delivered to you in this prompt.
Before emitting `verdict: NEED_CONTEXT`, scan the manifest and confirm the file you claim is missing is genuinely absent from the list.
If a file IS in the manifest but you cannot find its content via the `--- FILE: <path> ---` delimiter, that is a long-context recall failure on your side — re-scan;
do not emit NEED_CONTEXT for files in the manifest.
Only emit `verdict: NEED_CONTEXT` for paths that are NOT in the manifest, and explain under `## Missing context` why each path is needed (one line per path).
The orchestrator will re-fire the review with those files added.
Fabricating file contents — or inferring them from filename / position alone — is a worse failure than halting honestly.

## Criteria (apply to the plan as a whole)

- **Constraint violations** — BLOCKING.
- **Alignment** — plan covers all task requirements.
- **Decision alignment** — every `### Decision:` in `## Shared Decisions` faithfully implemented.
- **Completeness** — every card has `Creates`/`Edits`, `Context`, `Moves`, `Requirements`, `Commit`.
- **Moves well-formed** — each `Moves:` sub-bullet is an `` `old` -> `new` `` pair (backtick-wrapped paths, ASCII ` -> ` arrow);
  bare `none` on the label line is valid;
  any other format is a finding.
- **Rename mechanic present** — any batch whose cards contain a non-empty `Moves:` must include a `## Rename mechanic` section describing the `git mv` + surgical-edit approach;
  absence is a finding.
- **No full-file rewrites of relocated files** — prescribing a write-from-scratch for a file that appears in `Moves:` (rather than `git mv` + surgical edits) is a finding.
- **Sequencing + batch dependencies** — correct order within and across batches;
  `batch-depends` accurate;
  no forward deps.
- **Batch Index DAG integrity** — BLOCKING if the `batches:` block in `00-overview.md` has a cycle, references a batch name not declared, or names a `file:` not present in the plan directory.
- **Edge cases + risks** — failures, empty states, boundaries addressed.
- **Over-engineering** — unneeded abstractions or unrequested features.
- **Codebase consistency** — follows patterns in the source files provided.
- **Test coverage** — error paths + edges.
- **Language pitfalls** — BLOCKING if high-risk (Python: mutable defaults, import side-effects, Windows path sep, CRLF/LF).
- **Integration test reachability** — BLOCKING if integration tests added but `verify:` doesn't run them.
- **Explore targets** — purpose-driven;
  subset of `Context:`.
- **Step granularity + atomicity** — each card small and self-contained.
- **Requirements specificity** — BLOCKING if `Requirements:` uses vague prose ("refactor X", "update to use helper") without naming the specific function, class, or constant being changed.
  Stable identifiers are required.
- **Context field** — non-empty per card;
  Edits: files are implicitly read.
- **Context completeness** — BLOCKING if `Requirements:` mentions a function, class, or constant from a file not listed in `Context:` or `Edits:`.
  The implementer may only read files in `Context:`;
  a missing entry means cold-start exploration.
- **Global step numbering** — unique, sequential, no gaps across batches.
- **All Files Touched scope** — the overview's `## All Files Touched` section lists the union of `Edits:`/`Creates:`/Move-target paths across all batches;
  `Deletes:` tokens and Move-source paths are excluded by convention.
  A Deletes-only or Move-source-only path missing from that list is correct, not a finding.
- **Platform-behavior-claim verification** — BLOCKING if a plan or discussion claim describes Claude Code's own platform/harness behavior (e.g. agent auto-discovery, plugin manifest semantics) and a manifest or doc file that could confirm or refute the claim is present in your context, bulked or Read-able,
  but the claim was accepted without checking that file.
  Tool-use-mode reviewers may Read `plugin.json`/platform docs directly even when not bulked.

Independently state, in the `reviewer_self_id:` field below, what model/version you believe yourself to be — this is your own best-effort assessment, distinct from the `reviewer_model:` value already dictated to you above.

## Output format — STRICT

Wrap your entire output in `MILL_REVIEW_BEGIN` / `MILL_REVIEW_END` markers, each on its own line.
Everything outside these markers is ignored by the backend.
**No preamble inside the markers.**
Per finding: 3–5 lines, short and factual.
The consumer has full context of the plan;
do NOT explain background.
Cite the batch/card, state what's wrong, propose the fix.

Target length: ~300 tokens for APPROVE, ~600–1200 tokens for REQUEST_CHANGES across multiple batches.
If you produce more than ~1500 tokens, compress.

```
MILL_REVIEW_BEGIN
# Review: Add file/dir toc verbs (Tree-sitter-backed) — holistic

```yaml
verdict: APPROVE | REQUEST_CHANGES | NEED_CONTEXT
reviewer_model: opushigh
reviewer_self_id: <your own model self-identification, if known>
reviewed_file: plan/
date: <UTC YYYY-MM-DD>
```

## Findings

### [BLOCKING:design] <short title, <60 chars>
**Location:** <batch / card number> **Issue:** <one sentence> **Fix:** <one sentence>

### [NIT:consistency] <short title>
**Location:** <batch / card> **Issue:** <one sentence> **Fix:** <one sentence>

## Missing context
(include ONLY when verdict is NEED_CONTEXT — omit the section otherwise)

- `path/to/file.py` — <one-line reason the reviewer needs this file>

## Verdict

<APPROVE | REQUEST_CHANGES | NEED_CONTEXT>
<one sentence — max 20 words>
MILL_REVIEW_END
```

Severity / verdict rules match review-plan-batch.md.

**Severity vocabulary is closed.** Use ONLY `BLOCKING` or `NIT` as the bracketed label in a finding heading -- never invent another word (e.g. `MAJOR`, `MINOR`, `CRITICAL`, `MEDIUM`, `HIGH`). If a finding's severity feels ambiguous, default to `BLOCKING`, never `NIT` -- an over-cautious BLOCKING can be pushed back on by the orchestrator; a mislabeled NIT (or an unrecognized label) can silently skip review entirely.

**Class is the second axis, encoded in the same bracket as severity, colon-separated, lowercase: `### [BLOCKING:design] <title>`.**
A finding with no class, or a class outside the four names below, is a reviewer defect.
The four recognised classes, identical in meaning across every review stage:

- `design` — a decision is missing, wrong, or rests on a false premise.
  Example: a card's `Requirements:` never states which of two conflicting approaches from the discussion to implement.
- `scope` — the work inventory is incomplete, or the enumeration method is unreliable.
  Example: a batch's `Context:` list omits a file the card's own `Requirements:` names.
- `decision` — a named artifact with no stated disposition.
  Example: a Shared Decision references a config key the plan never says whether the card should add, migrate, or leave alone.
- `consistency` — the artefact contradicts itself, carries a superseded statement, or violates an established repo convention.
  Example: two cards in different batches prescribe different commit messages for the same file.

**Class governs who decides and when the loop stops, never whether a finding gets fixed.**

Omit `## Findings` if zero findings. Never invent findings to pad.

## Out of scope for this stage

- Per-line code correctness belongs to code review, not to plan review.
- A plan reviewer judges whether the plan's method for enumerating work is reliable, not whether it re-enumerates the work itself.


---

## Output contract

Write your full report to this file: /home/knatte/Code/quarry/wts/toc-verbs/_mill/briefs/review-plan-holistic-r7.out.md

Any format the prompt above asks for (including a `MILL_REVIEW_BEGIN` / `MILL_REVIEW_END` wrapped report) is the content of /home/knatte/Code/quarry/wts/toc-verbs/_mill/briefs/review-plan-holistic-r7.out.md -- write it there, not into chat.

Your final chat message must be exactly one line and nothing else: `WROTE /home/knatte/Code/quarry/wts/toc-verbs/_mill/briefs/review-plan-holistic-r7.out.md`
