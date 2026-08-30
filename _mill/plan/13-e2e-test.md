# Batch: e2e-test

```yaml
task: Port the capability-ladder bench harness to Go
batch: e2e-test
number: 13
cards: 2
verify: go test ./bench/loomyard-eval/ladder/...
depends-on: [12]
```

## Batch Scope

Adds the one test the per-unit suite structurally cannot provide: a synthetic end-to-end run driving
ingest, the gates, redaction, score recording, and summarisation over a hand-written subagent
transcript for a small synthetic ladder. Its purpose is catching field-name drift across those four
stages' handoffs, which per-unit tests cannot see because each one asserts against its own expectation
rather than against the next stage's expectation.

No dispatch of any kind happens: the scorer reply is canned, and the transcript is written by hand.

## Cards

### Card 66: Synthetic ladder and transcript fixtures

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/transcript.go`
  - `bench/loomyard-eval/ladder/internal/ladder/ladder.go`
  - `bench/loomyard-eval/ladder/internal/ladder/usage.go`
  - `bench/loomyard-eval/ladder/ladder.yaml`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/testdata/e2e-ladder.yaml`
  - `bench/loomyard-eval/ladder/internal/ladder/testdata/e2e-transcript.jsonl`
  - `bench/loomyard-eval/ladder/internal/ladder/testdata/e2e-answer-fasit.json`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Write a minimal synthetic ladder file that passes every load-time validation rule
  with two configs on one ladder — one control and one rung — a single repetition, and every pin set,
  so no test has to supply overrides. Write a hand-shaped subagent transcript for the rung: assistant
  records carrying the pinned model, an effort value, usage on all four token classes, at least one
  quarry tool call and one Bash grep, a user record with the corresponding tool results, and a final
  assistant record whose text ends with the answer as a fenced block. Write the matching answer key for
  the synthetic task, including a metadata section so the stripping step has something to strip. Every
  value must be internally consistent, since the whole point of the fixture is that one drifting field
  name shows up as a mismatch downstream rather than being absorbed.
- **Commit:** `test(ladder): add synthetic end-to-end fixtures`

### Card 67: The synthetic end-to-end test

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/runstate.go`
  - `bench/loomyard-eval/ladder/internal/ladder/usage.go`
  - `bench/loomyard-eval/ladder/internal/ladder/gates.go`
  - `bench/loomyard-eval/ladder/internal/ladder/score.go`
  - `bench/loomyard-eval/ladder/internal/ladder/summarize.go`
  - `bench/loomyard-eval/ladder/internal/ladder/correlate.go`
  - `bench/loomyard-eval/ladder/internal/ladder/testdata/e2e-ladder.yaml`
  - `bench/loomyard-eval/ladder/internal/ladder/testdata/e2e-transcript.jsonl`
  - `bench/loomyard-eval/ladder/internal/ladder/testdata/e2e-answer-fasit.json`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/e2e_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Build a synthetic results tree in the test's temp directory and drive the four
  stages in order: take transcript custody and extract usage, run the gates and write the ingest
  marker, redact the answer and assemble the scorer prompt, record a canned scorer reply and write the
  run marker, then summarise. Assert the final summary value in full rather than field by field, so a
  renamed field fails the test rather than silently dropping out of the comparison. Assert also that
  the provisional denial marker survives from the usage record onto the summarised cell. Dispatch
  nothing: the scorer reply is a literal in the test.
- **Commit:** `test(ladder): add the synthetic end-to-end test`

## Batch Tests

`verify: go test ./bench/loomyard-eval/ladder/...` runs `e2e_test.go` alongside every per-unit test in
the ladder subtree.
