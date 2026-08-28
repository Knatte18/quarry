# quarry benchmark: is there anything real to gain?

Self-contained orchestration protocol. A fresh agent with no memory of how this
was designed should be able to execute this end to end from this file alone.
Do not skip steps because they seem obvious — the design has specific
rationale behind each constraint (see "Design rationale" at the bottom);
deviating from it quietly invalidates the result.

## The question

Does `quarry` (specifically the `toc` and `impact` verbs) measurably help an
LLM coding agent do real exploration and code-review work on a large,
unfamiliar Go codebase — faster and/or cheaper, without losing correctness —
compared to the same agent with only standard tools (Read/Grep/Bash/Glob)?

Target codebase: Loomyard, at `/home/knatte/Code/loomyard/wts/loomyard`
(~69k LOC, ~90 internal packages, single Go module). If that path does not
exist on the machine running this, stop and ask for the correct path rather
than guessing or substituting a different repo — the task files below
reference specific real files and commits in Loomyard.

**Every task pins Loomyard to a fixed commit, never the live main checkout.**
Main moves fast (active daily development) — a task pointed at "whatever
main currently is" is not reproducible: re-running it later, or comparing a
past run's numbers to a new one, would be comparing against a different
codebase, not just a different quarry. Fixed pin for tasks 01–02
(exploration, no specific historical bug involved):

```
PINNED_SHA = 975578cda8d6f3a81580bd4e73725e060211b766
```

Tasks 03–04 pin to their own specific historical commit instead (the one
whose real bug they reconstruct) — see each task's own `Setup` section.
Every task's `Setup` section builds a disposable, read-only `git worktree`
from its pin and points `<TARGET_DIR>` at that worktree, never at
`/home/knatte/Code/loomyard/wts/loomyard` itself. Remove each worktree after
the task's three agents finish.

## Roles

Every task runs three independent agents, dispatched in parallel (no
dependency between them — do not run them sequentially):

- **A ("with quarry")** — gets the task plus quarry's tool surface.
- **B ("without quarry")** — gets the identical task text, standard tools
  only, and must never learn that quarry exists.
- **C ("fasit"/reference)** — high effort, no budget pressure, may use
  anything including quarry, sole job is to produce the most accurate
  possible answer to score A and B against.

### Prerequisites (run once, before any task)

1. Confirm the Loomyard checkout exists at the path above.
2. Confirm the verbs a task needs are actually implemented:
   ```
   go run ./cmd/quarry --help
   ```
   from the quarry repo root. Tasks 01–02 need only `toc` (present since
   `toc-verbs` landed). Tasks 03–04 need `impact` — if it is not listed,
   those tasks are **blocked**; report them as blocked rather than
   substituting `refs`/`definition` as a workaround, which would test a
   different tool than the one the task is about.
3. Build a quarry binary once (avoids recompiling on every invocation):
   ```
   go build -o /tmp/quarry-bench ./cmd/quarry
   ```
   from the quarry repo root. Use this binary's absolute path (`/tmp/quarry-bench`)
   everywhere below — do not rebuild per task.

## Prompt templates

Each task file under `tasks/` supplies `<TASK TEXT>` and a task-specific
output schema. Combine with the role preamble below verbatim — do not
paraphrase or shorten these preambles, the exact wording matters (especially
B's, which must not leak quarry's existence).

`<TARGET_DIR>` is the codebase A/B/C actually explore. It is always the
pinned, disposable worktree a task's own `Setup` section builds — never the
live main checkout at `/home/knatte/Code/loomyard/wts/loomyard`. Read that
task's `Setup` section for the exact worktree path before substituting
`<TARGET_DIR>`.

### Agent A preamble

```
You are working on a code task. You have access to a code-navigation CLI
called quarry, built from an LSP-backed Go tool. Binary: /tmp/quarry-bench
Target codebase for all quarry commands: <TARGET_DIR>

Verbs relevant to you:
- quarry toc dir <path>            -- lists every source file directly in a
  directory (not recursive) with its package, header comment, test/generated
  flags. <path> can be an absolute path under the target codebase.
- quarry toc file <path>           -- table of contents for one file: every
  function/method/type with signature, docstring, and precise line ranges
  (start/sigend/end) -- read start..sigend for signature+docstring, start..end
  for the whole body, without opening the rest of the file.
- quarry refs <symbol> --target-dir <TARGET_DIR>
                                    -- every reference to a symbol, LSP-resolved
  (finds interface-dispatched calls grep cannot). Add --within <subdir> to
  scope out noise from same-named methods on unrelated interfaces.
- quarry definition <symbol> --target-dir <TARGET_DIR>
                                    -- jump to a symbol's definition.
- quarry assert-no-callers <symbol> --target-dir <TARGET_DIR> [--except <path>]
                                    -- fails if the symbol has callers outside
  its declaration and --except paths.
- quarry impact <symbol> --target-dir <TARGET_DIR>
                                    -- (only if listed in `quarry --help`)
  resolves a symbol, finds its callers, and reports each caller's full
  enclosing-function line range.

All verbs output JSON on stdout. You may also use ordinary tools (Read, Grep,
Bash, Glob) -- quarry supplements exploration, it does not replace it. Prefer
quarry when it directly answers what you need (symbol lookups, "who calls
this", a file/directory survey); use grep/read for everything else.

<TASK TEXT>

When you are completely done, end your final message with ONLY a fenced json
code block matching the schema below -- no other trailing prose after it.

<SCHEMA>
```

### Agent B preamble

```
You are working on a code task in the codebase at <TARGET_DIR>. You have
standard tools: Read, Grep, Bash, Glob. Explore as needed to answer
thoroughly and correctly.

<TASK TEXT>

When you are completely done, end your final message with ONLY a fenced json
code block matching the schema below -- no other trailing prose after it.

<SCHEMA>
```

Do not give this agent filesystem access to the quarry repo (only to
`<TARGET_DIR>`), and never mention the word "quarry" anywhere in its prompt
or context. It must reach for grep/read the way an agent with no such tool
naturally would.

### Agent C preamble

```
You are the reference agent for a benchmark. Your only job is to produce the
most accurate, thoroughly verified answer to the task below. There is no time
or token budget pressure -- prioritize correctness over speed. You may use any
tool available to you, including a code-navigation CLI called quarry if
helpful (binary: /tmp/quarry-bench, target codebase: <TARGET_DIR> -- run
`/tmp/quarry-bench --help` and `/tmp/quarry-bench <verb> --help` for usage).
Cross-check anything important
using more than one method rather than trusting a single tool's default
output -- e.g. gopls' default view can miss build-tag-gated call sites, so
confirm a "no other callers" claim with both quarry and a manual grep across
build tags if the task depends on it.

<TASK TEXT>

You have not been shown any other agent's output, and there is none to
consider -- produce your answer purely from your own investigation.

When you are completely done, end your final message with ONLY a fenced json
code block matching the schema below -- no other trailing prose after it.

<SCHEMA>
```

## Output schemas

**Exploration tasks:**
```json
{
  "relevant_files": ["internal/reedengine/geometry.go", "..."],
  "key_symbols": [
    {"name": "FuncOrTypeName", "file": "path/to/file.go", "role": "one sentence"}
  ],
  "summary": "3-6 sentences explaining how the mechanism works end to end",
  "confidence": "high|medium|low",
  "open_questions": ["anything left uncertain, if any"]
}
```

**Code-review tasks:**
```json
{
  "verdict": "safe|needs-changes|unsafe",
  "issues_found": [
    {
      "description": "...",
      "location": "file:line",
      "severity": "blocking|nit",
      "evidence": "how this was verified, e.g. 'quarry refs shows caller X at file:line still assumes the old signature' or 'grepped for all call sites, none found'"
    }
  ],
  "confidence": "high|medium|low"
}
```

## Dispatch protocol

1. For the task at hand, read its file under `tasks/` and run its `Setup`
   section to build the pinned worktree it defines `<TARGET_DIR>` as.
2. Build the three full prompts (role preamble with `<TARGET_DIR>`,
   `<TASK TEXT>`, and `<SCHEMA>` substituted in).
3. Dispatch A, B, and C as three parallel agent calls in a single turn — they
   are independent, there is no ordering dependency, and dispatching them
   sequentially just wastes wall-clock time.
4. Each agent call reports back its final message (the JSON block) plus
   token/tool-call/duration usage. Record all of this — do not discard the
   usage numbers, they are half of what this benchmark measures.
5. Write results to `results/<YYYY-MM-DD>/<task-slug>/`:
   - `a.json`, `b.json`, `c.json` — each agent's parsed output block
   - `usage.json` — `{"a": {"tokens": N, "tool_uses": N, "duration_ms": N}, "b": {...}, "c": {...}}`
   - `scorecard.md` — see Scoring below

## Scoring

Matching "the same real finding" across two independently-worded answers
needs semantic judgment, not string equality — do this as one more ordinary
(non-independence-constrained) agent call, or by your own judgment if you are
the orchestrator, reading `a.json`/`b.json` against `c.json`.

- **Exploration:** recall = (C's relevant_files/key_symbols also present in
  A's, resp. B's) / (C's total); precision = (A's, resp. B's, entries
  corroborated by C) / (A's, resp. B's, total). Also judge qualitatively
  whether the `summary` describes the same actual mechanism C found, not just
  whether file names overlap.
- **Code review:** recall = C's real issues also caught by A, resp. B;
  precision = A's, resp. B's, flagged issues actually corroborated by C
  (an issue B raised that C did not find is not automatically a false
  positive — note it, but flag for a manual look rather than auto-scoring it
  wrong, since C is a strong reference but not infallible).
- **Efficiency:** tokens/tool_uses/duration_ms, A vs B, from `usage.json` —
  but only meaningful to compare between runs that scored equally on
  correctness. A faster wrong answer is not a win.

`scorecard.md` should end with one plain-language verdict sentence: was
quarry worth it on this task, and why.

## Design rationale — do not "simplify" these away

- **C never orchestrates A and B.** It would compromise C's independence (it
  might lock its own answer having seen A/B's first, even unintentionally)
  and conflates two different jobs — producing ground truth, and managing
  the experiment. Keep C a pure, blind investigator; the orchestrator (you)
  dispatches all three.
- **B must never learn quarry exists**, and must not be given filesystem
  access to the quarry repo — only to Loomyard. Telling it "don't use X" is
  not the same as it never encountering X; the blinding has to be structural,
  not just an instruction.
- **A must not see B's or C's output before finishing, and vice versa.**
  Dispatch truly in parallel.
- **impact-gated tasks (03, 04) are blocked, not worked around,** until
  `impact` actually lands in `quarry --help`. Substituting `refs`/`definition`
  tests a different tool.
- **One run per arm per task, first pass.** This is a temperature check, not
  a publishable study. If a task's result looks ambiguous, rerun that
  specific task before drawing a conclusion from it — don't add blanket
  repetition up front.

## Tasks

See `tasks/`:
- `01-reed-geometry-exploration.md` — exploration, `toc`, runnable now
- `02-shedadapters-exploration.md` — exploration, `toc`, runnable now
- `03-reed-attach-geometry-review.md` — code review, `impact`, blocked until impact-verb lands
- `04-refactor-review-TBD.md` — code review, `impact`, blocked + target diff not yet finalized
