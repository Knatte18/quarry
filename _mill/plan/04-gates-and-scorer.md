# Batch: gates-and-scorer

```yaml
task: "Ladder harness around headless claude -p (T2)"
batch: "gates-and-scorer"
number: 4
cards: 4
verify: go test ./bench/loomyard-eval/ladder/...
depends-on: [1, 2, 3]
```

## Batch Scope

This batch delivers the two surviving gates and the scorer. Both are judgment layers over a
finished rep, both consume the transcript records of batch 2 and the token matcher of batch 1, and
both must agree on what the giveaway token is — which is why they land together rather than either
one landing beside the run loop that calls them. Batch 6's run loop consumes `CheckBlinding`,
`CheckRenderedControlPrompt`, `CheckWorktreeDirtied`, `BuildScorerPrompt`, `RedactAnswer` and
`ParseScorerReply`; `CheckGrantedToolUsed` is consumed by **batch 7's summariser**, not by the run
loop, because gate 1 is a judgment over a whole cell's repetitions and only the summariser has all
of them.

Batch-local decision: `gates.go` takes every fact it cannot compute from the transcript as an
explicit argument — the pinned worktree's tracked-file token presence, the auto-loaded project
context's token presence, the quarry repository root path, the MCP prefix and the server name. Nothing in this
batch touches a worktree, resolves an environment variable or runs a process; the run loop
supplies those and the gates stay pure and directly table-testable.

## Cards

### Card 16: the two gates

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/stream.go`
  - `bench/loomyard-eval/ladder/internal/ladder/metrics.go`
  - `bench/loomyard-eval/ladder/internal/ladder/match.go`
  - `bench/loomyard-eval/ladder/internal/ladder/config.go`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/gates.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** create `package ladder` file `gates.go` with a `Finding` struct carrying
  `Gate string`, `Fatal bool`, `Message string` and an optional `Count int`, and exactly two gate
  entry points plus the pre-dispatch check.
  Gate 1, `CheckGrantedToolUsed(cfg Config, perRepQuarryToolUses []int) *Finding`: applies per
  **cell** and is never fatal. When the config's allowed list is non-empty and the maximum
  prefixed-tool-use count across its reps is zero, return a finding whose message is exactly
  `!! <id>: tool-granted config whose agent never called a granted tool in any repetition -- this
  cell measures the tool's prompt cost, not the tool`, with the config's id substituted. Otherwise
  return nil.
  Gate 2, `CheckBlinding(t *Transcript, in BlindingInput) []Finding`: applies per **rep** and only
  when the cell's allowed list is empty. `BlindingInput` carries `MCPPrefix string`,
  `ServerName string`, `QuarryRepoRoot
  string`, `TokenInTargetTrackedFiles bool` and `TokenInAutoLoadedContext bool`. The server name is
  its own field rather than something check (d) re-derives by trimming the prefix, matching how
  card 17's redaction input carries the two separately — the two consumers must agree on the
  giveaway token, and re-deriving it in one of them is how they stop agreeing. Run checks in
  order over the whole transcript re-marshalled to JSON via the transcript's marshal-all method —
  every record and every field, the session-init record's working directory included, never a
  selected subset — short-circuiting on the first fatal. Check (a): the marshalled transcript
  contains the MCP prefix, matched as a composed string; fatal. Check (b): it contains the quarry
  repository root path, matched as a composed string; fatal. Check (c): it contains the bare token
  `quarry`, matched with the shared bare-token matcher; recorded as the **always non-fatal**
  observation named `target_origin_quarry_mention`, carrying the occurrence count and the record
  types it appeared in. Check (c) must never set the fatal flag under any input — this is a
  property the tests assert directly, because a location-based fatal branch here is unreachable at
  the pinned commit and an unreachable check reads as protection while providing none. To let a
  reader tell an expected mention from a surprising one, the observation's message names which
  antecedents held: whether the token also appears in an earlier `tool_result` payload in the same
  transcript — computed by re-marshalling with every `tool_result` block's nested content replaced
  by the literal `REDACTED` and testing whether the token survives, the split V1's redaction
  helper performed and which here selects provenance rather than deciding a verdict — and the two
  booleans the caller supplies.
  Check (d), `CheckRenderedControlPrompt(prompt string, in BlindingInput, quarryTools []string)
  *Finding`: applies per rep, **before dispatch**, and is fatal. The fully rendered prompt for a
  control cell must contain neither the bare token `quarry`, nor the server name, nor any entry of
  the ladder file's tool list — each matched with the shared bare-token matcher, which is what
  keeps a three-character tool name from matching ordinary prose — nor the MCP prefix as a composed
  string. A failure fails the rep without spending an API call.
  Also add `CheckWorktreeDirtied(porcelain string) *Finding`, returning a never-fatal observation
  when a worktree's porcelain status output is non-empty. Do not port any V1 gate other than these:
  the run-prompt, max-turns, model-pinned, target-override, denied-tools and every cold or daemon
  gate are retired because the CLI now enforces or reports each directly.
- **Commit:** `feat(ladder): add the granted-tool and control-blinding gates`

### Card 17: the scorer prompt, redactor and reply parser

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/fenced.go`
  - `bench/loomyard-eval/ladder/internal/ladder/match.go`
  - `bench/loomyard-eval/ladder/internal/ladder/config.go`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/score.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** create `package ladder` file `score.go`. Port `ExplorationRule` and `ImpactRule`
  verbatim from `origin/v1-final:bench/loomyard-eval/ladder/internal/ladder/score.go`, keeping the
  schema-to-rule map, and port `StripFasitMeta`, which returns a shallow copy of a decoded fasit
  with its top-level `_meta` key -- that exact spelling, leading underscore included, as the committed fasit files carry it -- removed and leaves every other field verbatim. Port
  `scoreFieldsFromRule` and the key-matching regexp it uses, deriving the required field set from
  each rule's own fenced example so the rule text and its validator cannot drift, and build the
  per-schema required-field map from it. Port `ParseScorerReply`, decoding the **first** fenced
  block of the scorer's reply via the shared extractor and erroring when a required field for that
  schema is absent. Add `RedactAnswer(answer string, in RedactionInput) string`, where
  `RedactionInput` carries the ladder file's tool list, the server name, the MCP prefix, the quarry
  repository root path and the task worktree path. Build the bare-token half of the alternation
  through the shared matcher's alternation constructor over every tool name and the bare server
  name; apply the MCP prefix and the two paths as case-sensitive composed-string replacements.
  Every replacement writes the same fixed placeholder. The bare server name is in the alternation
  deliberately: without it an answer whose prose names the server identifies the arm, and it is the
  same token gate 2's check (c) treats as leakage — the two must agree. Do not build the
  alternation from a hardcoded tool list; the names come from the loaded file. Add
  `BuildScorerPrompt(rule string, taskText string, fasit map[string]any, redactedAnswer string)
  (string, error)` assembling V1's four parts in V1's order — the rule, a task heading with the
  task text, a reference-fasit heading with the meta-stripped fasit as a fenced JSON block, and an
  answer heading with the redacted answer as a fenced JSON block. Add `ScoreRecord` as a
  `map[string]any` and a helper producing the unscored stand-in shape carrying a false scored flag
  and a reason string, for the max-turns and scorer-failed cases batch 6 writes.
- **Commit:** `feat(ladder): add the blinded scorer prompt, redactor and reply parser`

### Card 18: gate fixtures

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/gates.go`
  - `bench/loomyard-eval/ladder/internal/ladder/stream.go`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/testdata/transcripts/leaked-prefix.jsonl`
  - `bench/loomyard-eval/ladder/internal/ladder/testdata/transcripts/target-origin-quarry.jsonl`
- **Deletes:** none
- **Moves:** none
- **Requirements:** hand-author two more line-delimited transcript fixtures in the same shape as
  batch 2's. The first carries a `tool_use` block whose name begins with the `mcp__quarry__`
  prefix, so check (a) fires fatally. The second carries the bare token `quarry` twice: once inside
  a `tool_result` block's content and once in an assistant text block that paraphrases it, with no
  MCP prefix and no repository-root path anywhere, so check (c) records a non-fatal observation
  whose antecedent report distinguishes the two occurrences. Neither fixture may contain an
  absolute host path.
- **Commit:** `test(ladder): add blinding-gate transcript fixtures`

### Card 19: gate and scorer tests

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/gates.go`
  - `bench/loomyard-eval/ladder/internal/ladder/score.go`
  - `bench/loomyard-eval/ladder/internal/ladder/match.go`
  - `bench/loomyard-eval/ladder/internal/ladder/prompt.go`
  - `bench/loomyard-eval/ladder/internal/ladder/config.go`
  - `bench/loomyard-eval/tasks/01-reed-geometry-exploration.md`
  - `bench/loomyard-eval/tasks/01-reed-geometry-exploration.fasit.json`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/gates_test.go`
  - `bench/loomyard-eval/ladder/internal/ladder/score_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** write `gates_test.go` covering, in this order: check (a) fatal on the
  leaked-prefix fixture and check (a) using a prefix passed as an argument rather than a literal;
  check (b) fatal on a transcript containing a supplied repository-root path; check (c) recorded as
  a non-fatal observation on the target-origin fixture, plus a table asserting check (c) is
  non-fatal for **every** combination of the two antecedent booleans and of the token appearing
  inside or outside a tool result — the assertion is that no input makes it fatal, so a
  location-based fatal branch cannot come back as code; check (d) fatal on a rendered prompt
  containing the word quarry, fatal on one containing the tool token `toc`, fatal on one containing
  the MCP prefix, fatal on one containing the bare server name alone — run with a server name that
  is not the word quarry, so the case fails if the implementation re-derives the name from the
  prefix or leans on the hardcoded default instead of reading the supplied field — and **passing**
  on the real prompt rendered from task 01 for a control cell whose
  tool list is the four built-in tools named in the overview's the-four-built-in-tools decision; check (d) passing on a prompt containing the word `protocol`,
  which is the three-character-token false positive the shared matcher exists to prevent. Add gate 1
  cases for a granted cell whose reps all report zero prefixed tool uses (finding, non-fatal), a
  granted cell where one rep reports a non-zero count (no finding), and a control cell with zero
  uses (no finding, because the allowed list is empty). Write `score_test.go` asserting: the derived
  required-field set for each rule equals exactly the field names that rule's own example declares;
  meta stripping removes only the top-level `_meta` key and leaves every other field byte-identical,
  run against the real task 01 fasit; the reply parser accepts a well-formed reply and rejects one
  missing a required field, naming it; the redactor removes the tool name bare, the tool name
  prefixed, the bare server name and both supplied paths, while leaving the word `protocol` intact;
  and the assembled scorer prompt carries the four parts in order with the fasit and the answer each
  inside their own fenced block.
- **Commit:** `test(ladder): cover both gates, the redactor and the scorer reply contract`

## Batch Tests

`verify: go test ./bench/loomyard-eval/ladder/...` covers this batch through `gates_test.go` and
`score_test.go`. Two assertions in that set are the point of the whole batch and must not be
weakened: that check (c) is non-fatal under every input, which is what keeps an unreachable fatal
branch from returning as code, and that check (d) passes on the real rendered control prompt for
task 01 while failing on a prompt carrying any giveaway token — the second is the only fatal
blinding check whose failure mode is a harness bug rather than a property of the target.
