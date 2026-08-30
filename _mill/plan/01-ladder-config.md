# Batch: ladder-config

```yaml
task: Port the capability-ladder bench harness to Go
batch: ladder-config
number: 1
cards: 7
verify: go test ./bench/loomyard-eval/ladder/...
depends-on: []
```

## Batch Scope

Creates the `ladder` Go package under `bench/loomyard-eval/ladder/internal/ladder/` and ports
`ladder_config.py` in full: the canonical tool constants, the four config value types, `LoadLadder` with
every one of its validation rules, the lookup helpers, the two pin-enforcement variants, deny-list and settings
derivation, preamble generation, and fenced-JSON extraction. It also adds the two new `ladder.yaml`
fields (`run_effort`, `session_dir_template`) and blanks `max_turns` to `null`.

This is the root batch: every later batch imports this package. The external interface the next batches
consume is the `Ladder` / `LadderConfig` / `TaskEntry` / `ScorerConfig` types, `LoadLadder`, `MCPName`,
`QuarryTools`, `DaemonBackedTools`, `MCPPrefix`, and `ExtractFencedJSON`.

Batch-local decision: the Go package is named `ladder` and lives under a `bench/loomyard-eval/ladder/internal/`
prefix so Go's own `internal/` visibility rule makes it unimportable from the product tree.

## Cards

### Card 1: Package scaffold, tool constants, and config types

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/ladder_config.py`
  - `internal/cli/cli.go`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/ladder.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Create package `ladder` with a package doc comment modelled on the package doc in
  `internal/cli/cli.go`, stating that this package is the ported capability-ladder harness logic and that
  it is unimportable from the product tree. Port `QUARRY_TOOLS` as `QuarryTools []string` holding the
  same canonical seven names in the same order, `MCP_PREFIX` as `MCPPrefix = "mcp__quarry__"`,
  `DAEMON_BACKED_TOOLS` as `DaemonBackedTools []string` derived from `QuarryTools` by excluding
  `toc_dir` and `toc_file` (derived, never a literal), and `mcp_name` as `MCPName(tool string) string`.
  Port the four frozen dataclasses as structs with `yaml` tags: `LadderConfig` (ID, Ladder, Task,
  Allowed, Cold, WarmCounterpart), `TaskEntry` (TaskFile, PinnedSHA, Worktree, Schema, Fasit),
  `ScorerConfig` (Model, Effort), and `Ladder` carrying RunModel, Reps, MaxTurns, RunEffort,
  SessionDirTemplate, Scorer, QuarryTools, Tasks, SourceRepo, Configs, ColdWorktreeTemplate.
  `RunModel` and `MaxTurns` must be pointer or otherwise nil-distinguishable types so an unset YAML
  value is distinguishable from a zero value — `RequirePins` depends on that distinction. Port
  `LadderConfigError` as an exported `ConfigError` error type. Preserve each Python docstring's
  rationale in the corresponding Go doc comment.
- **Commit:** `feat(ladder): add Go ladder package with tool constants and config types`

### Card 2: LoadLadder and its validation rules

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/ladder_config.py`
  - `bench/loomyard-eval/ladder/ladder.yaml`
  - `bench/loomyard-eval/ladder/tests/test_ladder_config.py`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/ladder.go`
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/ladder_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Port `load_ladder` as `LoadLadder(path string) (*Ladder, error)` using
  `gopkg.in/yaml.v3`, preserving every one of its rejection rules and the `_fail` message shape
  (each error names the offending file and the specific rule). The enumeration below is authoritative;
  do not treat it as a count to match: duplicate config id; a
  `ladder` value outside `a`/`b`; an unknown task key; an unknown entry in a config's `allowed`;
  `quarry_tools` drifting from `QuarryTools`; zero controls on a ladder; more than one control on a
  ladder; more than one cold config; `warm_counterpart` set on a non-cold config; a cold config with
  no `warm_counterpart`; and a `warm_counterpart` naming an unknown id or a cold config. Write
  table-driven tests covering every rejection rule plus a positive load of the committed ladder file.
- **Commit:** `feat(ladder): port LoadLadder with its validation rules`

### Card 3: Lookups and the two pin-enforcement variants

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/ladder_config.py`
  - `bench/loomyard-eval/ladder/tests/test_ladder_config.py`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/ladder.go`
  - `bench/loomyard-eval/ladder/internal/ladder/ladder_test.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Port `config_by_id` as `ConfigByID`, `control_for` as `ControlFor`, and
  `warm_counterpart_for` as `WarmCounterpartFor`, each keeping the Python's error-on-miss behaviour.
  Port `require_pins` as `RequirePins(l *Ladder) error`, extended to reject an unset `RunEffort`
  alongside `RunModel`, `MaxTurns`, `Scorer.Model`, and `Scorer.Effort`. Add a second, narrower
  `RequireSessionPins(l *Ladder, runModelOverride string) error` enforcing only `RunModel` (satisfied
  instead by a non-empty `runModelOverride`), `RunEffort`, `SessionDirTemplate`, `Scorer.Model`, and
  `Scorer.Effort` — it deliberately does not check `MaxTurns`, because the ceiling is a gate-time value
  nothing about session preparation touches. Its doc comment must state that reason. Test both:
  `RequirePins` rejecting each unset pin individually, and `RequireSessionPins` passing with `MaxTurns`
  unset while failing with `RunModel` unset unless an override is supplied.
- **Commit:** `feat(ladder): port config lookups and split pin enforcement`

### Card 4: New ladder.yaml fields and the blanked max_turns

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/ladder.go`
- **Edits:**
  - `bench/loomyard-eval/ladder/ladder.yaml`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** Add `run_effort: medium` and
  `session_dir_template: /tmp/ladder-session-{config_id}-{n}` to the ladder file, and change
  `max_turns: 60` to `max_turns: null`. Add a comment above `max_turns` recording that its meaning is
  now "maximum number of assistant records in a run's subagent transcript", that the committed 60 was
  calibrated against a different accounting basis, and that the operator must set it before the matrix
  starts. Add a comment above `session_dir_template` stating that `{config_id}` and `{n}` are
  substituted and that `{n}` is the 1-based repetition index. Leave `run_model: null` as it is. Do not
  edit the file header block in this card — the header refresh belongs to the documentation batch.
- **Commit:** `feat(ladder): add run_effort and session_dir_template, blank max_turns`

### Card 5: Deny-list and settings derivation

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/ladder_config.py`
  - `bench/loomyard-eval/ladder/tests/test_ladder_config.py`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/settings.go`
  - `bench/loomyard-eval/ladder/internal/ladder/settings_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Port `deny_list_for` as `DenyListFor(l *Ladder, c LadderConfig) []string`,
  `settings_document_for` as `SettingsDocumentFor`, and `write_settings` as `WriteSettings`.
  `DenyListFor` must always derive from `l.QuarryTools` through `MCPName`, never from a literal list —
  its doc comment must record that a mutated already-loaded ladder has to keep producing a correct
  deny-list. `SettingsDocumentFor` produces `permissions.allow` of `Read`, `Grep`, `Glob`, `Bash` and
  `permissions.deny` of the derived quarry deny-list plus `Task`, except for a config whose `Allowed`
  is empty, where `permissions.deny` is exactly `["Task"]` and nothing else — no quarry name may appear
  in a blinded config's settings document, because that document sits in the blinded agent's own cwd.
  Test the drift-guard case where a loaded ladder's `QuarryTools` is mutated in memory and the deny-list
  must track it, plus both the blinded and the rung settings shapes.
- **Commit:** `feat(ladder): port deny-list and settings derivation`

### Card 6: Preamble generation

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/ladder_config.py`
  - `bench/loomyard-eval/ladder/tests/test_ladder_config.py`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/preamble.go`
  - `bench/loomyard-eval/ladder/internal/ladder/preamble_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Port `PARALLEL_OPENING`, `PARALLEL_BLOCK`, `B_PREAMBLE_BODY`, `_CLOSING_SENTENCE`,
  and `_TOOL_DESCRIPTIONS` as Go constants and a map, reproducing the committed prompt text byte for
  byte — this text is the measured stimulus and any drift silently changes what the matrix measures.
  Port `_mcp_preamble_body` as an unexported `mcpPreambleBody` and `preamble_for` as
  `PreambleFor(l *Ladder, c LadderConfig, targetDir, taskText, schemaJSON string) string`. Test both
  shapes: a control config reproduces the committed body with `<TARGET_DIR>` substituted, and a rung's
  generated body lists exactly its allowed tools in canonical `QuarryTools` order with their
  descriptions.
- **Commit:** `feat(ladder): port preamble generation`

### Card 7: Fenced-JSON extraction

- **Context:**
  - `bench/loomyard-eval/ladder/scripts/ladder_config.py`
  - `bench/loomyard-eval/ladder/tests/test_ladder_config.py`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/fenced.go`
  - `bench/loomyard-eval/ladder/internal/ladder/fenced_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** Port `_FENCED_JSON_RE` and `extract_fenced_json` as
  `ExtractFencedJSON(text, which string) (block, inner string, err error)` supporting the `first` and
  `last` selectors with the same semantics, including the error on no block found and on an unknown
  selector. Both halves are returned because both are load-bearing and neither is cheaply re-derivable
  from the other: `block` includes the fences and is what the schema extractor embeds into the preamble
  as measured stimulus, while `inner` is the decode-ready content the answer parser and the
  scorer-reply parser consume. Every call site names which half it takes. Go's
  `regexp` has no `DOTALL` flag — use the `(?s)` inline flag so the fence body may span lines. Test the
  first selector, the last selector, the multiple-block case, and the no-block case, asserting on both
  returned halves in each.
- **Commit:** `feat(ladder): port fenced-JSON extraction`

## Batch Tests

`verify: go test ./bench/loomyard-eval/ladder/...` covers `ladder_test.go`, `settings_test.go`,
`preamble_test.go`, and `fenced_test.go` — every test file this batch creates, and nothing outside the
ladder subtree, since no other package imports it.
