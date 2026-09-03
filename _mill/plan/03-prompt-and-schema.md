# Batch: prompt-and-schema

```yaml
task: "Ladder harness around headless claude -p (T2)"
batch: "prompt-and-schema"
number: 3
cards: 5
verify: go test ./bench/loomyard-eval/ladder/...
depends-on: [1]
```

## Batch Scope

This batch builds the prompt every cell receives and the extractor that reads a cell's answer back
out. It covers the ported fenced-JSON extractor, inclusion-based extraction of a task file's task
text and output schema, the one preamble that is identical for every cell, and the recovery of the
exploration output schema into the task file that lost it when the V1 protocol document was
deleted. It is one batch because all four are the same round trip — the schema goes into the
prompt and comes back as the answer through the same regexp. The external interface later batches
consume is `ExtractFencedJSON`, `LoadTaskFile` and `RenderPrompt`.

Batch-local decision: extraction is strictly inclusion-based. The extractor takes the task-text
block and the output-schema block and nothing else; it never looks for, matches or excludes the
answer-key heading. That heading is spelt five different ways across the five task files, and an
exclusion-based extractor keyed on one spelling would leak the answer key from the others — a
silent invalidation of every affected run rather than a visible error.

## Cards

### Card 11: port the fenced-JSON extractor

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/config.go`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/fenced.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** port `ExtractFencedJSON` and its regexp verbatim from
  `origin/v1-final:bench/loomyard-eval/ladder/internal/ladder/fenced.go` into a new `package
  ladder` file `fenced.go`, keeping the package-level compiled pattern, the exported
  `ErrNoFencedJSONBlock` sentinel, and the two-value return: the block **with** its fences, which
  the prompt embeds as measured stimulus, and the inner text, which the answer parser and the
  scorer-reply parser decode. Keep V1's deliberate divergence from the Python original: a selector
  other than `first` or `last` is an error, never a silent fallthrough to `last`. Carry over the
  doc comments explaining why both halves are returned and why the selector is validated, adjusting
  only wording that refers to V1's subcommand architecture. This is the one fenced-JSON regexp in
  the package; no other file compiles its own.
- **Commit:** `feat(ladder): port the fenced-json extractor from v1`

### Card 12: task-file extraction

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/fenced.go`
  - `bench/loomyard-eval/ladder/internal/ladder/config.go`
  - `bench/loomyard-eval/tasks/04-shedadapters-shuttle-impact.md`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/prompt.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** create `package ladder` file `prompt.go` with `LoadTaskFile(path string)
  (TaskContent, error)`, where `TaskContent` holds `TaskText string` and `SchemaBlock string` (the
  fenced block including its fences). Extraction is inclusion-based and takes exactly two things.
  First, the task-text block: locate the heading line whose text begins with the marker
  `## ` followed by a backtick-wrapped `<TASK TEXT>` token, then collect every following line that
  is part of the blockquote — lines beginning with `>` — until the first line that is neither a
  blockquote line nor blank, and dedent each collected line by stripping a leading `>` and at most
  one following space so an indented sub-list keeps its relative indentation. Blank lines inside
  the quote are preserved as blank. Second, the schema block: locate the **first** heading line
  beginning with the exact prefix `## Output schema` — case-sensitive, leading hashes exact,
  whatever parenthetical follows — and take the first fenced JSON block after it via
  `ExtractFencedJSON` on the remainder of the file with the `first` selector. A file with no such
  heading is a hard load error naming the file; do not return an empty schema. A file with no
  task-text heading is likewise a hard load error naming the file. Nothing else from the file is
  read: the setup section, the scope section and the answer-key section are all dropped by
  construction, and no code in this package may reference the answer-key heading's wording. Do not
  cross-check the schema heading's parenthetical against the ladder file's schema field — that
  field selects the scorer's rule and nothing else.
- **Commit:** `feat(ladder): extract task text and output schema from a task file`

### Card 13: the single per-cell preamble

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/config.go`
  - `bench/loomyard-eval/ladder/internal/ladder/fenced.go`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/prompt.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** add the three fixed constants to `prompt.go`, ported byte-for-byte from
  `origin/v1-final:bench/loomyard-eval/ladder/internal/ladder/preamble.go`: `PARALLEL_OPENING`,
  `PARALLEL_BLOCK` and the unexported `closingSentence`. Reproduce their text exactly, including
  line breaks and the double-hyphen spellings, and keep a doc comment recording that all three are
  byte-for-byte copies of the committed preambles the V1 measurement used, which is what carries
  the measurement's continuity. Do not port `B_PREAMBLE_BODY` or `mcpPreambleBody` — the divergent
  control and treatment bodies are exactly the confound this task removes. Write one fresh body
  instead, used identically by every cell, naming the target directory and listing the tool names
  the cell actually has, neutrally, with no preference language, no anti-grep language, and no
  mention of the arm, the rung, the server or the word quarry. Add `RenderPrompt(target
  TaskContent, targetDir string, toolNames []string) string` assembling, in order:
  `PARALLEL_OPENING`, the body, the task text, `PARALLEL_BLOCK`, `closingSentence`, and the schema
  block. The tool-name list the caller passes is the four built-in tools the overview's
  the-four-built-in-tools decision fixes, plus any granted MCP tool names for that cell;
  `RenderPrompt` lists what it is given and derives nothing. Add a doc comment
  stating that the word parallel in both constant names refers to parallel tool calls within one
  turn, not to parallel arms, so nothing in the text contradicts one-cell-at-a-time execution.
- **Commit:** `feat(ladder): render one identical preamble for every cell`

### Card 14: recover the exploration output schema

- **Context:**
  - `bench/loomyard-eval/tasks/04-shedadapters-shuttle-impact.md`
- **Edits:**
  - `bench/loomyard-eval/tasks/01-reed-geometry-exploration.md`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** insert a new section into the task file, headed
  `## Output schema (exploration tasks)`, positioned after the task-text blockquote and before the
  scorer-notes section, mirroring how the impact task file carries its own schema section between
  the same two neighbours. The section body is a single fenced JSON block whose content is exactly:
  an object with `relevant_files` as an array of file-path strings, `key_symbols` as an array of
  objects each carrying `name`, `file` and `role`, `summary` as a string described as three to six
  sentences explaining how the mechanism works end to end, `confidence` as the alternation
  high-medium-low, and `open_questions` as an array of strings. Take the block **verbatim** from the
  V1 benchmark protocol document, read with the shared decision's git-show form at the path
  `origin/v1-final:bench/loomyard-eval/README.md`, from the section headed `Output schemas` — copy
  its exploration block byte-for-byte, including its placeholder values, so the schema the cells
  receive is the one the fasit was written against rather than a paraphrase. Add one short sentence
  above the fence stating the schema was recovered from that document after it was deleted. Change nothing else in the file — the task
  text, the scope section, the setup section and the scorer notes stay exactly as they are, since
  the fasit was produced against them.
- **Commit:** `docs(bench): recover the exploration output schema into task 01`

### Card 15: prompt and extractor tests

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/prompt.go`
  - `bench/loomyard-eval/ladder/internal/ladder/fenced.go`
  - `bench/loomyard-eval/ladder/internal/ladder/config.go`
  - `bench/loomyard-eval/ladder/internal/ladder/match.go`
  - `bench/loomyard-eval/tasks/01-reed-geometry-exploration.md`
  - `bench/loomyard-eval/tasks/04-shedadapters-shuttle-impact.md`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/testdata/tasks/no-schema-heading.md`
  - `bench/loomyard-eval/ladder/internal/ladder/fenced_test.go`
  - `bench/loomyard-eval/ladder/internal/ladder/prompt_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** create a small task-file fixture carrying a task-text blockquote and a
  scorer-notes section but no output-schema heading at all. Write `fenced_test.go` asserting: with
  two fenced blocks present the `last` selector returns the second and the `first` selector the
  first; text with no fence returns the exported sentinel; an unrecognised selector returns an
  error rather than falling through; a fence body spanning many lines is captured whole; and the
  returned block half carries its fences while the inner half does not. Write `prompt_test.go`
  running `LoadTaskFile` against **both** ladder-referenced task files and asserting for each that
  the returned task text equals exactly the dedented blockquote and that no substring drawn from
  the setup section, the scope section or the scorer-notes section appears anywhere in the returned
  content — assert on equality with the expected extraction, not on the absence of one heading
  spelling. Assert the schema block is found in both files despite their differing parentheticals,
  and that the fixture with no schema heading produces an error naming the file rather than an
  empty schema. Add a `RenderPrompt` test asserting the six parts appear in the fixed order, that
  the rendered prompt for a control cell whose tool list is the four built-in tools named in the overview's the-four-built-in-tools decision contains neither the
  word quarry nor the token `toc` under the shared bare-token matcher, and that a rendered prompt
  for a granted cell lists the prefixed tool name it was passed.
- **Commit:** `test(ladder): cover fenced extraction, task-file parsing and prompt rendering`

## Batch Tests

`verify: go test ./bench/loomyard-eval/ladder/...` covers this batch through `fenced_test.go` and
`prompt_test.go`. The two real task files are the fixtures on purpose: they are the two files the
harness will actually run, their output-schema headings differ, and card 14 changes one of them —
a test that passed against a synthetic fixture and failed against the real file is precisely the
failure mode inclusion-based extraction exists to prevent. The committed
`no-schema-heading.md` fixture covers the hard-error path, which no real task file can exercise
once card 14 lands.
