# Batch: scoring

```yaml
task: Port the capability-ladder bench harness to Go
batch: scoring
number: 6
cards: 4
verify: go test ./bench/loomyard-eval/ladder/...
depends-on: [1]
```

## Batch Scope

Ports `score_run.py`'s pure units: the ordered redaction alternation, answer redaction, the two scoring
rules, fasit stripping, scorer-prompt assembly, and validation of a scorer reply. The subprocess scorer
client has no counterpart — dispatch moves to the session — so `score_run`'s injected-runner seam
becomes the `redact` / `record-score` subcommand pair the CLI batches wire up.

This batch depends only on the config batch, because none of it reads a transcript.

The external interface later batches consume is `RedactText`, `RedactAnswer`, `WriteRedacted`,
`StripFasitMeta`, `BuildScorerPrompt`, and `ParseScorerReply`.

Batch-local decision: the `/tmp/quarry-bench` branch stays in the redaction alternation even though the
same literal was dropped from the blinding gate in batch 3. The two mechanisms make different claims —
a gate that cannot fire reads as coverage that is not there, while a redaction branch that never
matches removes nothing and asserts nothing, and a redactor's job is to over-remove.

## Cards

### Card 27: The ordered redaction alternation

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/score_run.py`
  - `bench/loomyard-eval/ladder/internal/ladder/ladder.go`
  - `bench/loomyard-eval/ladder/tests/test_score_run.py`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/score.go`
  - `bench/loomyard-eval/ladder/internal/ladder/score_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Port `REDACTION_TOKEN` as `RedactionToken`,
  `_BARE_TOOL_NAMES_EXCEPT_IMPACT` as an unexported `bareToolNamesExceptImpact` derived from
  `QuarryTools`, `_build_redaction_pattern` as an unexported `buildRedactionPattern`,
  `_ADJACENT_TOKEN_RUN` as `adjacentTokenRun`, and `redact_text` as `RedactText(text string) string`.
  The alternation must keep its exact ordering, most specific first, so a full prefixed name or the
  `quarry <verb>` CLI shell form claims a match before the bare fallback gets a chance to. `impact` is
  deliberately excluded from the bare-name pass because it is an ordinary English word every
  Ladder-B answer's prose uses; only its prefixed and CLI-invocation forms are redacted, and the doc
  comment must record that. Use Go's `(?i)` inline flag for case-insensitivity. Test every alternation
  branch, the `impact` exemption in both directions, case-insensitivity, and the adjacent-token
  collapse.
- **Commit:** `feat(ladder): port the ordered redaction alternation`

### Card 28: Answer redaction

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/score_run.py`
  - `bench/loomyard-eval/ladder/tests/test_score_run.py`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/score.go`
  - `bench/loomyard-eval/ladder/internal/ladder/score_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Port `redact_answer` as `RedactAnswer` and `write_redacted` as
  `WriteRedacted(runDir string) (map[string]any, error)`, keeping the exact field set the Python
  redacts for each schema shape: an exploration answer's `summary`, `open_questions`, and each
  `key_symbols[].role`; an impact answer's `open_questions`, each `callers_to_update[].evidence`, and
  each `excluded_lookalikes[].reason`. Every structural field — `relevant_files`, every `file`, every
  `line`, every `name`, and `confidence` — is returned untouched. `WriteRedacted` leaves the original
  answer byte-identical. Test both answer shapes, the untouched structural fields, and that the
  original file is unmodified after the redacted copy is written.
- **Commit:** `feat(ladder): port answer redaction`

### Card 29: Scoring rules, fasit stripping, and prompt assembly

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/score_run.py`
  - `bench/loomyard-eval/ladder/internal/ladder/ladder.go`
  - `bench/loomyard-eval/ladder/tests/test_score_run.py`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/score.go`
  - `bench/loomyard-eval/ladder/internal/ladder/score_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Port `EXPLORATION_RULE`, `IMPACT_RULE`, and `_RULE_BY_SCHEMA` as Go constants and a
  map, reproducing the committed rule text byte for byte. Port `strip_fasit_meta` as
  `StripFasitMeta` and `build_scorer_prompt` as
  `BuildScorerPrompt(l *Ladder, c LadderConfig, redactedAnswer, fasit map[string]any, taskText string) (string, error)`.
  The three-input blinded contract is unchanged: the prompt carries the redacted answer, the
  `_meta`-stripped fasit, and the task text, and carries none of the config id, the ladder, the allowed
  set, or the transcript. Port `ScoringError` as an exported `ScoringError` type. Test that all three
  inputs are present, that `_meta` is stripped, and — the assertion that matters most — that the config
  id, ladder letter, allowed set, and transcript are each absent from the assembled prompt.
- **Commit:** `feat(ladder): port scoring rules, fasit stripping, and prompt assembly`

### Card 30: Scorer reply parsing

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/score_run.py`
  - `bench/loomyard-eval/ladder/internal/ladder/fenced.go`
  - `bench/loomyard-eval/ladder/tests/test_score_run.py`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/score.go`
  - `bench/loomyard-eval/ladder/internal/ladder/score_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Port `_extract_fenced_json` as `ParseScorerReply(reply, schema string) (ScoreRecord, error)`,
  reusing `ExtractFencedJSON` rather than compiling a second fence pattern, and taking its `inner`
  half — the decode-ready content — never the fenced `block`. Validation is an addition, not a port:
  the Python validates nothing beyond decoding, so a scorer reply missing a field would reach
  `score.json` and surface later as an absent metric with nothing naming the cause. Take the task's
  schema as a parameter, because the required set is schema-dependent, and derive that set from the two
  rule constants' own declared output shape — the exploration rule's and the impact rule's — rather
  than from a hand-written list, so the rules stay the single source. Independently of schema, every
  metric the summariser reads out of the score record must be present, since a missing one silently
  drops a cell's measurement. Define `ScoreRecord` as the `score.json` shape, carrying the scorer's
  metrics plus the pinned scorer model, effort, and `prompt_template` — the task schema the template was
  chosen from — which `record-score` stamps in. `prompt_template` is kept rather than dropped: its
  purpose is making a drifting scorer prompt visible in the record, and that purpose is untouched by
  the dispatch swap. Do not port `run_scorer_client` or
  `score_run`'s dispatch half — dispatch happens in a live session, never in a subprocess, and the doc
  comment must say so. Test a well-formed reply for each schema, a reply with no fenced block, a reply whose block is
  not valid JSON, a reply missing a schema-specific required field, and a reply missing a
  summariser-read metric.
- **Commit:** `feat(ladder): port scorer reply parsing`

## Batch Tests

`verify: go test ./bench/loomyard-eval/ladder/...` covers `score_test.go` plus every other test file in
the ladder subtree.
