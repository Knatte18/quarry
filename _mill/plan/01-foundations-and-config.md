# Batch: foundations-and-config

```yaml
task: "Ladder harness around headless claude -p (T2)"
batch: "foundations-and-config"
number: 1
cards: 7
verify: go test ./bench/loomyard-eval/ladder/...
depends-on: []
```

## Batch Scope

This batch lays the ground every other batch stands on: the one new module dependency, the
results-tree ignore rule, the shared token matcher, the yaml loader with every V1 coupling removed,
and the migration of the ladder configuration files to the shape that loader accepts. It is one
batch because the loader's validations and the ladder file's contents are the same decision seen
from two sides — a validation without a file that satisfies it, or a migrated file with nothing
that can load it, is untestable. The external interface later batches consume is the `Ladder`,
`Task` and `Config` types plus `LoadLadder`, `MCPPrefix`, `MatchesBareToken` and
`MatchesComposedString`.

Batch-local decision: the loader's table tests build their yaml inputs as in-test string constants
written to `t.TempDir()`, rather than as two dozen near-identical committed fixture files. The one
on-disk fixture case is the real migrated `ladder-toc.yaml`, loaded from its tracked path, which is
the case that must not drift. Committing twenty single-purpose invalid yaml files would make the
rejection table harder to read than the code it tests.

## Cards

### Card 1: module dependency and the results-tree ignore rule

- **Context:**
  - `CLAUDE.md`
- **Edits:**
  - `go.mod`
  - `go.sum`
- **Creates:**
  - `bench/loomyard-eval/ladder/.gitignore`
- **Deletes:** none
- **Moves:** none
- **Requirements:** add `gopkg.in/yaml.v3` as a direct require of the root module
  `github.com/Knatte18/quarry` by running `go get gopkg.in/yaml.v3` from the repository root, so
  `go.mod` and `go.sum` are both updated by the toolchain rather than hand-edited. Do not add any
  other dependency and do not create a nested module under `bench/loomyard-eval/ladder/` — the
  task's done-criterion is `go build ./... && go test ./...` at the repository root, and a nested
  module is silently excluded from both. Create
  `bench/loomyard-eval/ladder/.gitignore` containing a short comment and the single pattern
  `results/*/raw/`, and nothing else; the comment must state that the pattern anchors to this
  directory because it contains a slash, which is why the rule does not live in the repository-root
  ignore file. Leave the repository-root ignore file alone.
- **Commit:** `build(ladder): add yaml.v3 and ignore the ladder raw results tree`

### Card 2: the shared token matcher

- **Context:**
  - `CLAUDE.md`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/match.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** create `package ladder` file `match.go` implementing the two, and only the two,
  matching classes the token-matching decision defines. Export
  `BareTokenPattern(token string) *regexp.Regexp`, returning
  `regexp.MustCompile("(?i)\\b" + regexp.QuoteMeta(token) + "\\b")`, and
  `MatchesBareToken(text, token string) bool` built on it — case-insensitive, word-bounded, and
  regexp-quoted so a token carrying metacharacters is matched literally rather than interpreted.
  Export `MatchesComposedString(text, s string) bool` as a case-sensitive
  `strings.Contains`. Export `BareTokenAlternation(tokens []string) *regexp.Regexp`, which builds a
  single case-insensitive word-bounded alternation over the quoted tokens, skipping empty entries
  and returning `nil` for an empty or all-empty slice; the scorer's redactor and gate check (d)
  both consume it, and it is the only place an alternation over tool names is ever built. Add a
  package doc comment on this file stating that bare identifier tokens (the literal `quarry`, the
  server name, each entry of a ladder file's tool list) use the word-bounded form while composed
  strings (an MCP prefix, a repository root path, a worktree path) use the substring form, and why:
  a three-character tool name under substring matching would match ordinary prose.
- **Commit:** `feat(ladder): add the shared bare-token and composed-string matcher`

### Card 3: the ladder configuration types and loader

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/match.go`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/config.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** create `package ladder` file `config.go` declaring the loaded shape as Go
  structs with `yaml` tags and nothing beyond it: `Ladder` with `RunModel`, `RunEffort`,
  `MaxTurns int`, `Reps int`, `Scorer ScorerSpec`, `QuarryTools []string`, `SourceRepo string`,
  `Server *ServerSpec`, `Tasks map[string]Task`, `Configs []Config`; `ScorerSpec` with `Model` and
  `Effort`; `ServerSpec` with `Name`, `Build`, `Args []string`, `Env map[string]string`; `Task`
  with `TaskFile`, `PinnedSHA`, `Schema`, `Fasit`; `Config` with `ID`, `Ladder`, `Task`,
  `Allowed []string`. `MaxTurns` is a plain `int`, never a pointer-for-unset. Implement
  `LoadLadder(path string) (*Ladder, error)` decoding with `gopkg.in/yaml.v3` using
  `yaml.Decoder.KnownFields(true)` so an unrecognised key is a load error naming the key, then
  running the validations card 4 adds. Implement two accessors every later batch reads instead of
  re-deriving: `(*Ladder) ServerName() string`, returning the declared server name and the default
  `quarry` when no server block is present, and `(*Ladder) MCPPrefix() string`, returning
  `"mcp__" + ServerName() + "__"`. Implement `(*Ladder) ConfigByID(id string) (Config, bool)` and
  `(*Ladder) ControlFor(letter string) (Config, bool)`, the latter finding the single config whose
  ladder letter matches and whose allowed list is empty — by field, never by parsing an id. Add
  `(Config) IsControl() bool` returning whether the allowed list is empty.
  Declare here, in this same file, the package-level slice
  `BuiltinTools = []string{"Read", "Grep", "Glob", "Bash"}` that the overview's
  the-four-built-in-tools decision fixes. It lives in the earliest file of the DAG precisely so that
  every consumer — the prompt renderer of batch 3, the gate tests of batch 4, the run loop of batch
  6 and the live test of batch 8 — is strictly downstream of its declaration and none of them has to
  re-spell the names. Give it a doc comment recording that the CLI's session-init record reports the
  same set back in sorted order, `["Bash","Glob","Grep","Read"]`, so a test asserting the reported
  list compares against the sorted form rather than this one.
- **Commit:** `feat(ladder): add the ladder yaml shape and loader`

### Card 4: loader validations, kept and newly-refusing

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/match.go`
- **Edits:**
  - `bench/loomyard-eval/ladder/internal/ladder/config.go`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** add a `validate` method on `*Ladder`, called by `LoadLadder` after decoding,
  enforcing exactly these rules, each returning an error naming the offending key or id. Kept from
  V1: `source_repo` must be the literal string `env:LADDER_LOOMYARD_REPO`, and the environment
  variable it names is resolved lazily elsewhere, never here; every config id is unique; every
  config's task names a key present in the tasks map; every entry of a config's allowed list is a
  member of the file's own tool list; for every ladder letter that actually appears in the file
  there is exactly one control (a config with an empty allowed list) — zero or two is an error, and
  a letter absent from the file is not required to have one. Also required: `run_model`,
  `run_effort`, `max_turns`, `reps` and both scorer fields must be non-zero; every task must carry
  a task file, a pinned sha, a fasit path and a schema that is either `exploration` or `impact`.
  Newly refusing, each with its own named message so a stale file fails legibly rather than
  silently loading: the retired keys `worktree`, `session_dir_template`, `cold`,
  `warm_counterpart`, `cold_worktree_template`, `annex`, `annexes` and `toc_format` are rejected
  wherever they appear — because the decoder runs with known-fields checking, they surface as
  unknown-key errors, so wrap that error to name the retired key and state that the field was
  removed with the V1 architecture. Do not validate the tool list against any package constant and
  do not restrict the ladder letter to `a` or `b`; both were V1 couplings this task removes. The
  absence of a server block is never a load-time error — a file whose configs grant tools while
  declaring no server loads cleanly, and the failure is raised at run time by batch 6.
- **Commit:** `feat(ladder): validate the ladder file and refuse every retired key`

### Card 5: migrate ladder-toc.yaml to the new shape

- **Context:**
  - `bench/loomyard-eval/ladder/internal/ladder/config.go`
- **Edits:**
  - `bench/loomyard-eval/ladder/ladder-toc.yaml`
- **Creates:** none
- **Deletes:** none
- **Moves:** none
- **Requirements:** rewrite the tool list as the single entry `toc`. Give `a2-toc-dir` and
  `b8-toc-dir` an allowed list of `[toc]` and leave both ids exactly as they are — the ids are the
  cross-root join key and the rewrite plan's T7 row names `a0-none` and `a2-toc-dir` literally.
  Remove the configs `a1-toc-file` and `b9-toc-file` entirely. Remove the `session_dir_template`
  key and its comment, remove every `worktree` key from the tasks map, and remove every
  `cold: false` line. Add a `server` block carrying exactly `name: quarry` and
  `build: ./cmd/quarry-mcp`, with no `args` key at all — T2 cannot write an argument list the MCP
  server task has not chosen yet, and an absent args key means no arguments. Keep `run_model`,
  `run_effort`, `max_turns`, `reps`, the scorer block, the tasks map's remaining fields and
  `source_repo` unchanged. Rewrite the file's long header comment so it describes the new harness and
  the file's surviving contents, with no line describing something the file no longer has. Keep the
  design rationale for the a-versus-b contrast and the August result that motivates the matrix.
  Rewrite the design block so it lists exactly the four surviving cell ids and their roles: the two
  controls and the two directory-level toc cells; remove the two file-level toc lines, since those
  configs no longer exist. Rewrite the "deliberately not here" list the same way: the entry about
  granting the directory-level and file-level tools together no longer describes an option this
  file could express, since the file declares one tool, so replace it with a line stating that the
  file-level tool has no successor in the server's shipped surface; keep the entries about the
  missing exploration fasit and the grep-toc control, updating the latter to say the harness renders
  one identical preamble for every cell and allows one control per ladder letter, so a per-config
  prompt is a harness change rather than a configuration change. Keep the fourth entry — annex
  delivery and the compact output form — as forward work rather than dropping it with the deleted
  files: both need engine features that do not exist yet, so they are out of this harness's scope
  rather than out of the project's, and the entry stays as the record of why the two deleted ladder
  files' subject matter has no successor here. Add one line stating that the server block's future
  argument list may use the placeholder `{target_dir}`, for which the harness substitutes the pinned
  worktree path — the contract card 22 defines, written down here so the MCP-server task spells it
  identically. Drop every sentence about session
  scratch templates, cold cells, the daemon, the seven V1 tool names and the deleted shell entry
  point, and replace the closing run instruction with the documented entry
  point `go run ./bench/loomyard-eval/ladder/cmd/ladder run --config
  bench/loomyard-eval/ladder/ladder-toc.yaml --results
  bench/loomyard-eval/ladder/results/<date>-toc`. State in the header that T2 proves the harness
  with `--cells a0-none --reps 1` and that T7 runs `--cells a0-none,a2-toc-dir` at the file's own
  `reps: 5`.
- **Commit:** `refactor(ladder): migrate ladder-toc.yaml to the headless harness shape`

### Card 6: delete the five unmigrated ladder files

- **Context:**
  - `bench/loomyard-eval/ladder/ladder-toc.yaml`
- **Edits:** none
- **Creates:** none
- **Deletes:**
  - `bench/loomyard-eval/ladder/ladder.yaml`
  - `bench/loomyard-eval/ladder/ladder-compact.yaml`
  - `bench/loomyard-eval/ladder/ladder-annex.yaml`
  - `bench/loomyard-eval/ladder/ladder-task05.yaml`
  - `bench/loomyard-eval/ladder/ladder-followup.yaml`
- **Moves:** none
- **Requirements:** delete all five with `git rm`. Each declares the seven V1 tool names and
  variously annexes, cold cells, a session directory template or a daemon, so each is unloadable
  under the loader card 4 just finished — a configuration file that cannot be loaded is worse than
  no file. All five stay recoverable at `origin/v1-final`. Do not migrate any of them and do not
  relocate them into a holding directory.
- **Commit:** `refactor(ladder): delete the five ladder files the new loader cannot load`

### Card 7: loader and matcher tests

- **Context:**
  - `bench/loomyard-eval/ladder/ladder-toc.yaml`
  - `bench/loomyard-eval/ladder/internal/ladder/config.go`
  - `bench/loomyard-eval/ladder/internal/ladder/match.go`
- **Edits:** none
- **Creates:**
  - `bench/loomyard-eval/ladder/internal/ladder/config_test.go`
  - `bench/loomyard-eval/ladder/internal/ladder/match_test.go`
- **Deletes:** none
- **Moves:** none
- **Requirements:** write `config_test.go` as a table test over in-test yaml string constants
  written to `t.TempDir()`, covering the accepted minimal shape and one rejected case per
  validation: duplicate config id, a config naming an unknown task, an allowed entry outside the
  file's own tool list, a wrong `source_repo` literal, a ladder letter with zero controls, a ladder
  letter with two controls, a task with an unrecognised schema value, and one case per retired key
  asserting the error message names that key. Add a case asserting that a file whose configs grant
  tools while declaring no server block loads without error. Add a separate test that loads the
  real migrated `bench/loomyard-eval/ladder/ladder-toc.yaml` from its tracked path and asserts:
  four configs with ids `a0-none`, `a2-toc-dir`, `b0-none`, `b8-toc-dir`; the tool list is exactly
  `[toc]`; `ControlFor("a")` returns `a0-none` and `ControlFor("b")` returns `b0-none`;
  `MCPPrefix()` is `mcp__quarry__`; and a `Ladder` value with no server block reports the same
  prefix. Write `match_test.go` as a table test with the tool token `toc`, asserting that
  `protocol`, `stochastic` and `October` do not match, that `toc`, `TOC`, `the toc tool` and a
  backtick-wrapped `toc` do match, that a token containing regexp metacharacters is matched
  literally rather than interpreted, that `MatchesComposedString` matches an MCP prefix and a
  repository-root-shaped path as a case-sensitive substring and does not match a differently-cased
  copy, and that `BareTokenAlternation` returns nil for an empty slice.
- **Commit:** `test(ladder): cover the loader validations and the shared token matcher`

## Batch Tests

`verify: go test ./bench/loomyard-eval/ladder/...` runs the whole new package, which at the end of
this batch is `config_test.go` and `match_test.go`. The scope is the package this batch creates, so
no wider run is needed; the overview's module-wide `go build ./...` catches the `go.mod` change
breaking any other package. The one on-disk fixture is the migrated `ladder-toc.yaml` at its
tracked path, which is deliberate: it is the file T7 will run, and a loader that accepts a
synthetic fixture but not the real file is the failure this test exists to catch.
