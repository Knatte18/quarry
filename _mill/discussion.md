# Discussion: Rethink quarry-mcp's per-call targetDir ergonomics

```yaml
task: Rethink quarry-mcp's per-call targetDir ergonomics
slug: mcp-target-dir-ergonomics
status: discussing
parent: main
```

## Problem

Every one of quarry-mcp's seven MCP tools (`toc_file`, `toc_dir`, `workspace_symbol`,
`textDocument_definition`, `textDocument_references`, `impact`, `assert_no_callers`) publishes an
optional call-wide `targetDir` property that overrides the server's launch-time target directory.
That mirrors the CLI's per-invocation `--target-dir` flag rather than how a language server is
actually scoped — an LSP workspace root is set once at `initialize` and never re-specified per
request. The MCP wrapper's whole justification over the CLI was removing per-call flag-guessing; a
parameter the model must consider on every single call reintroduces exactly that cost.

**Why now:** this surfaced while designing the quarry-mcp-vs-cli benchmark. The benchmark could not
be run honestly with the parameter present — `bench/loomyard-eval/ladder/scripts/ladder_config.py`
has to carry an explicit "Never set targetDir or buildTags on any of these calls" instruction in
its prompt, and `bench/loomyard-eval/ladder/scripts/gates.py:119-127` had to add a **fatal** gate
that kills a run if any `mcp__quarry__*` call carries `targetDir`. A parameter that has to be
suppressed by prompt text and enforced by a fatal gate is a parameter that should not be in the
schema.

A safety concern was investigated first and ruled out: daemon state is keyed by
`workspaceKey(targetDir)` (`internal/cli/paths.go:76-97`), a hash of the **absolute path**, so one
long-lived server already isolates multiple worktrees and repos cleanly, with separate state file,
lock, socket, and gopls process per distinct `targetDir`, and no restart needed. Nothing here is a
correctness fix. This is purely an API-surface decision.

## Scope

**In:**

- Remove the call-wide `targetDir` field from all five input structs in `internal/mcpserver/`:
  `lspInput` (`tools_lsp.go:33-34`, shared by `textDocument_definition` and
  `textDocument_references`), `workspace_symbol`'s input (`tools_symbol.go:58-59`), `impact`'s
  input (`tools_impact.go:27-28`), `assert_no_callers`'s input (`tools_assert.go:33-34`), and both
  toc inputs (`tools_toc.go:35-37` and `tools_toc.go:50-52`).
- Delete `effectiveTargetDir` (`internal/mcpserver/callcontext.go:49-59`) and drop `resolveCall`'s
  `targetDirOverride` parameter, so `callContext.TargetDir` is always `cfg.TargetDir`.
- Update the five `resolveCall(cfg, in.TargetDir, in.BuildTags)` call sites
  (`tools_lsp.go:142`, `tools_lsp.go:167`, `tools_symbol.go:119`, `tools_impact.go:103`,
  `tools_assert.go:119` — five sites across four files) and the two toc handlers
  (`tools_toc.go:167`, `tools_toc.go:189`) which call `effectiveTargetDir` directly.
- Reword every jsonschema description string that names `targetDir` as if it were a parameter:
  `nativeentry.go:32`, `nativeentry.go:44`, `nativeentry.go:49`, `lspentry.go:21`,
  `lspentry.go:55`. Go doc comments naming `targetDir` as a concept (e.g. `callcontext.go`'s
  header, `nativeentry.go:76`'s `query(targetDir string)`) stay — the *concept* survives, only the
  *parameter* goes.
- Add a regression test asserting no registered tool's published input schema carries a
  `targetDir` property.
- Update `docs/mcp-setup.md`: document that the server is scoped once at launch, that the cwd
  default is what makes per-project scoping automatic, and that a second named server entry with an
  explicit `--target-dir` is the cross-repo escape hatch.
- Update `bench/loomyard-eval/ladder/scripts/ladder_config.py:383`'s prompt line to drop the
  `targetDir` half of "Never set targetDir or buildTags".
- Update / delete the existing tests that exercise per-call `targetDir` overrides.

**Out:**

- `buildTags` — stays a per-call parameter, unchanged. Not touched anywhere in this task.
- `lang` — stays a per-call parameter, unchanged.
- The `--target-dir` launch flag itself (`cmd/quarry-mcp/main.go:22`) and
  `mcpserver.ResolveLaunchTargetDir` — both unchanged, including the cwd default and the
  `quarry-mcp: resolved target directory <path>` stderr line.
- Any new helper, script, or slash-command for regenerating `.mcp.json` (see the
  `no-repointing-helper` decision).
- The CLI's own `--target-dir` flag on `cmd/quarry`. The CLI is a different product surface and
  keeps per-invocation targeting.
- The committed root `.mcp.json` — it already passes no `--target-dir` and needs no change.
- `gates.py`'s `targetDir` check — deliberately retained (see `bench-ladder-update`).
- The native `-remote=auto` shared-daemon build-tag leakage caveat noted in the task body. Already
  mitigated at `daemon/ensureserver.go:145-165`; unrelated to this task.
- Any change to `internal/cli/`, `quarry/` (the facade), or the daemon.

## Decisions

### drop-per-call-targetdir

- **Decision:** Remove the call-wide `targetDir` property outright from all seven tools. The
  server's target directory is fixed at launch (`--target-dir`, defaulting to the process cwd) and
  cannot be changed per call.
- **Rationale:** The value of the MCP wrapper over the CLI is that the model does not have to get
  per-invocation arguments right. A "documented rare escape hatch" does not deliver that — an
  optional property is still in the published schema, still described to the model, and still a
  decision the model spends attention on for every call. The benchmark evidence is decisive: the
  parameter had to be suppressed with a prompt instruction *and* a fatal run-killing gate before a
  fair benchmark could run. The safety motivation that might have justified per-call targeting
  never existed — `workspaceKey` already isolates by absolute path. quarry-mcp is at
  `serverVersion = "0.1.0"` (`internal/mcpserver/mcpserver.go:19`) with no external consumers, so
  there is no deprecation obligation.
- **Rejected:** *Keep as a deprecated/documented escape hatch* — leaves the schema property in
  place, which is the entire cost being removed; "deprecated" is invisible to a model reading a
  JSON schema. *Keep as-is with reworded descriptions* — rewording does not reduce the number of
  decisions per call.

### buildtags-out-of-scope

- **Decision:** `buildTags` remains a per-call parameter on every tool that has it today. No
  launch-time `--build-tags` flag is added.
- **Rationale:** `targetDir` and `buildTags` look alike (both feed `workspaceKey`) but are
  semantically different. `targetDir` is workspace *identity* — one value per server by
  definition, exactly what LSP's `rootUri` encodes. `buildTags` is query *scoping*: the CLI
  exposes it per-verb (`internal/cli/cli.go:223`, `cli.go:398`, `cli.go:500`,
  `internal/cli/impact.go:179`), and asking "what calls this under the `integration` tag" is a
  legitimate per-query question a single session may want asked both ways. The bench ladder groups
  the two only for run-to-run determinism, not because they share an ergonomic defect.
- **Rejected:** *Also drop `buildTags`, add a launch flag* — removes real capability with no
  replacement short of restarting the server, and widens the task beyond its brief. *Keep per-call
  and add a launch default* — reintroduces the exact "optional property still in the schema"
  problem this task exists to remove, for a parameter nobody has complained about.

### hard-removal-not-graceful-ignore

- **Decision:** A call that still sends `targetDir` fails as a whole-call error, via the SDK's
  call-wide schema validation. No compatibility shim, no silent ignore, no custom migration error.
- **Rationale:** `inputSchemaFor` (`internal/mcpserver/schema.go`) deliberately clears
  `additionalProperties` **only** on the `targets` item schema and leaves the call-wide
  `additionalProperties: false` that inference produced — a decision already recorded in the
  function's doc comment ("the call-wide properties keep whatever inference produced, so a
  call-level violation stays a whole-call failure") and asserted by `schema_test.go:106-111`.
  Removing the Go field therefore inherits the existing, already-tested behaviour for stray
  call-wide keys with no new code. The resulting error is loud and self-correcting: a model that
  sends the property gets told it is not accepted and retries without it.
- **Rejected:** *Accept and silently ignore* — the model would believe it had retargeted the call
  and receive results from the wrong root, the single worst failure mode available here. *Keep the
  field, hide it from the schema via a call-wide analogue of `dropEntryProperty`, return a custom
  migration message* — real machinery for a migration path with zero external consumers.

### keep-cwd-default-launch-flag

- **Decision:** `--target-dir` stays optional at launch, defaulting to the process working
  directory via `ResolveLaunchTargetDir`. Unchanged code; recorded because it is load-bearing for
  the decision above.
- **Rationale:** cwd inheritance is what makes per-project scoping automatic and makes dropping the
  per-call override cheap. A project-scoped `.mcp.json` is launched by the client with the project
  root as cwd, so each worktree's session gets its own server process rooted at its own worktree,
  with its own `workspaceKey` and its own gopls — no configuration, no per-call parameter, no
  repointing. Making the flag required would break the committed argument-free `.mcp.json` and
  force every consumer to hard-code an absolute path.
- **Rejected:** *Make `--target-dir` required* — trades a working zero-config default for
  mandatory per-machine absolute paths.

### no-repointing-helper

- **Decision:** Do not build a config-generator script or slash-command. Document the cwd-inheritance
  contract in `docs/mcp-setup.md` instead.
- **Rationale:** The task body proposed a helper on the premise that repointing "today means
  hand-editing `.mcp.json`'s `args` and reconnecting". That premise does not hold against the
  committed file: the root `.mcp.json` is `{"command": "go", "args": ["run", "./cmd/quarry-mcp"]}`
  — there is no `--target-dir` argument to edit. You do not repoint one shared server; each
  session already launches its own, correctly rooted. A generator would be a tool for a problem
  that does not occur, and a new maintained surface for zero gain.
- **Rejected:** *A `quarry mcp-config` subcommand* — solves the non-problem, and adds a CLI verb
  whose only output is a four-line JSON file. *A shell script or slash-command* — same, plus it
  would be outside the Go codebase.

### cross-repo-escape-hatch-is-a-second-server

- **Decision:** The documented answer for a genuinely cross-repo or cross-worktree query is a
  second named server entry in the client's MCP config with an explicit `--target-dir`. Document
  it in `docs/mcp-setup.md`; add no code.
- **Rationale:** This is the one capability the removal actually costs: a session rooted at repo A
  can no longer ask about repo B. MCP clients already support multiple named servers, and
  `workspaceKey` guarantees the two instances share no daemon, lock, or gopls, so the mechanism is
  proven safe by the same investigation that de-risked this whole task. Documenting the cost
  honestly beats leaving a schema property in place to cover it.
- **Note on the partial hatch that survives:** `toc_file` and `toc_dir` accept absolute target
  paths (`cli.ResolveTOCPath(targetDir, arg)` — an absolute `arg` ignores `targetDir`), so those two
  tools can still read a file outside the launch root. The five language-server-backed tools cannot:
  even with an absolute `file`/`uri`, the query is served by the gopls rooted at `cfg.TargetDir`, so
  a path outside that root will not resolve correctly. The documentation must state this asymmetry
  rather than implying absolute paths are a general escape hatch.

### reword-dangling-targetdir-descriptions

- **Decision:** Reword the five jsonschema description strings that reference `targetDir` as a
  parameter, to name "the server's target directory" instead.
- **Rationale:** After removal, strings like `"a plain path (absolute, or relative to targetDir)"`
  (`nativeentry.go:32`) point at a parameter that no longer exists in the schema, which is a
  documentation bug the model reads on every call. The affected strings are `nativeentry.go:32`
  (`file`), `nativeentry.go:44` (`within`), `nativeentry.go:49` (`except`), `lspentry.go:21`
  (`uri`), and `lspentry.go:55` (`within`).
- **Rejected:** *Leave them* — dangling parameter references in schema text that the model consumes
  as its only documentation.

### delete-effectivetargetdir

- **Decision:** Delete `effectiveTargetDir` entirely. `resolveCall` drops its `targetDirOverride`
  parameter and assigns `cfg.TargetDir` straight to `callContext.TargetDir`. The two toc handlers
  read `cfg.TargetDir` directly and lose their error branch.
- **Rationale:** With no override, `effectiveTargetDir` degenerates to `return cfg.TargetDir, nil`
  and its `filepath.Abs` error path becomes unreachable — `cfg.TargetDir` is already absolute,
  guaranteed by `ResolveLaunchTargetDir` and re-checked by `NewServer`'s
  `filepath.IsAbs` guard (`mcpserver.go:75-77`). Keeping the function would leave every toc handler
  with a dead error branch that can never fire and that tests cannot honestly cover.
- **Rejected:** *Keep it as a one-line passthrough* — preserves an unreachable error path and a
  function whose doc comment ("This is the only place a per-call override becomes absolute") would
  become false.

### bench-ladder-update

- **Decision:** Edit `bench/loomyard-eval/ladder/scripts/ladder_config.py:383` to drop the
  `targetDir` half of its "Never set targetDir or buildTags" prompt instruction. Leave
  `bench/loomyard-eval/ladder/scripts/gates.py:119-127` checking both keys.
- **Rationale:** The prompt line becomes stale the moment the property is gone, and it spends
  prompt tokens telling the model not to do something the schema no longer permits — which for a
  benchmark measuring ergonomic overhead is measuring the wrong thing. The gate is different: it
  costs nothing, and retaining the `targetDir` arm turns it into a cheap end-to-end assertion that
  the property really is gone from the live server. Editing an existing file inside the
  already-sanctioned Python exception is not "extending" it under `CLAUDE.md`'s rule; no new Python
  file is created and no new Python is introduced anywhere else.
- **Rejected:** *Leave bench untouched* — leaves a stale instruction in the measured prompt.
  *Also remove the gate's `targetDir` arm* — throws away a free regression assertion.

## Technical context

**Package layout.** Everything in scope lives in `internal/mcpserver/` plus one doc and one bench
file. The package binds `quarry/facade.go` onto MCP tools and imports `internal/cli` only for
resolution helpers; `layering_test.go` mechanically enforces that it never imports
`internal/quarryengine` directly. Nothing in this task changes an import.

**How a call resolves its target directory today.** Two paths exist, deliberately:

1. The five language-server-backed tools (`textDocument_definition`, `textDocument_references`,
   `workspace_symbol`, `impact`, `assert_no_callers`) call
   `resolveCall(cfg, in.TargetDir, in.BuildTags)` (`callcontext.go:70-101`), which calls
   `effectiveTargetDir`, then `cli.ResolveBuildTags`, `cli.ResolveConfigPath`,
   `quarry.LoadRegistry`, and `cli.ResolveStateDir` — in that exact order, mirroring
   `resolveContext` in `internal/cli/cli.go`, because a divergence would silently spawn a second
   gopls daemon and forfeit warm-daemon reuse.
2. The two toc tools call `effectiveTargetDir` **directly** and never `resolveCall`
   (`tools_toc.go:167`, `tools_toc.go:189`). This is intentional and documented in
   `callcontext.go`'s header: `tocFileCommand`/`tocDirCommand` in `internal/cli/toc.go` never load
   the registry or resolve a state dir either, so a malformed `servers.yaml` must not fail a toc
   call. **Preserve this split.** The refactor must not "simplify" the toc handlers onto
   `resolveCall`.

**Schema derivation.** `inputSchemaFor[T]` (`schema.go`) infers from the Go struct via
`jsonschema.For`, then patches the `targets` property (forces `Type: "array"` because slice
inference emits `["null","array"]`, sets `MinItems`/`MaxItems` to `minTargets`/`maxTargets` = 1/64,
and clears `additionalProperties` on the item schema only). Because schemas are *derived from the
Go types*, deleting the `TargetDir` struct field is sufficient to remove the property from every
published schema — there is no separate schema literal to edit. `dropEntryProperty` exists for
dropping *entry-level* properties a tool does not accept; there is no call-wide analogue, and this
task should not add one.

**Where the launch value comes from.** `cmd/quarry-mcp/main.go` parses `--target-dir`, passes it to
`mcpserver.ResolveLaunchTargetDir` (absolutise if non-empty, else `os.Getwd()`), logs
`quarry-mcp: resolved target directory <path>` to stderr, and puts the result in `Config.TargetDir`.
`NewServer` rejects a non-absolute `Config.TargetDir`. Nothing on this path changes.

**Daemon isolation (the investigation that de-risked this task).**
`cli.ResolveStateDir(cfg.StateDir, absTargetDir, tags)` derives per-workspace state via
`workspaceKey` (`internal/cli/paths.go:76-97`), a SHA-256 over the absolute target path and the
normalized build tags. `ensureSupervised` (`daemon/ensureserver.go:328-335`) derives separate state
file, lock, and socket paths from it. Two worktrees of the same repo therefore never share a
daemon, lock, or gopls process. This is why a long-lived server was already safe with per-call
overrides — and equally why one server per session is safe now.

**Downstream consumers of `callCtx.TargetDir`.** After the change these all still receive the same
value, just always the launch value: `entry.query(callCtx.TargetDir)` in `tools_lsp.go:121`,
`tools_impact.go:77`, `tools_assert.go:85`; `cli.FilterWithin(..., callCtx.TargetDir)` in
`tools_lsp.go:128`, `tools_assert.go:100`; `cli.FilterImpactWithin` in `tools_impact.go:84`;
`exceptSet(callCtx.TargetDir, entry.Except)` in `tools_assert.go:103`; and
`callCtx.options(lang, query)` which threads it into `quarry.Options.TargetDir`
(`callcontext.go:103-114`). None of these signatures change.

**Gotcha — stale doc comments.** Several comments describe the override as existing:
`callcontext.go`'s file header ("either the launch default or an absolutised per-call override" at
line 28-30, and the `effectiveTargetDir` paragraph), `mcpserver.go:33-35` ("used whenever a call
omits its own targetDir override"), and `tools_lsp.go:3` ("plus lang/buildTags/targetDir
overrides"). All must be corrected; a grep for `targetDir` and `TargetDir` across
`internal/mcpserver/` after the change should surface only intentional survivors.

**Gotcha — `Config.TargetDir` doc.** `mcpserver.go:29-31`'s comment "a handler never sees these raw
values, only what `resolveCall` derives from them per call" becomes only half true once
`callContext.TargetDir` is unconditionally `cfg.TargetDir`; the toc handlers will now read
`cfg.TargetDir` directly. Reword rather than delete.

**Test files that touch `targetDir`:** `callcontext_test.go`, `tools_lsp_test.go`,
`tools_impact_test.go`, `tools_assert_test.go`, `tools_toc_test.go`, `nativeentry_test.go`,
`translate_test.go`, `transport_test.go`, `transport_errors_test.go`. Not all of these test the
*override*; several use the word only when constructing a `Config` or resolving a path. Each needs
reading before editing — do not blanket-delete on a grep hit.

## Constraints

- No `CONSTRAINTS.md` exists at the hub root.
- **`CLAUDE.md`: this is a Go repo; do not introduce Python.** The two sanctioned exceptions are
  `bench/loomyard-eval/scripts/gen_compact_toc.py` and `bench/loomyard-eval/ladder/`, and they must
  not be extended. This task edits one line of an existing file inside the second exception
  (`ladder_config.py`) and creates no Python.
- **Nothing in `internal/mcpserver` or `cmd/quarry-mcp` may write to `os.Stdout`** — that stream is
  reserved for the framed MCP protocol. Diagnostics go to stderr. This is why `quarry-mcp` is a
  separate binary from `cmd/quarry` with no cobra command reachable.
- **State-directory derivation must stay bit-for-bit identical to the CLI's.** `resolveCall` must
  keep going through the exported `internal/cli` helpers in the same order; a local reimplementation
  would silently spawn a second gopls daemon.
- **The toc tools must keep bypassing `resolveCall`** — a malformed `servers.yaml` must not fail a
  toc call.
- `layering_test.go`'s facade-only rule stays satisfied; no new imports.
- The build requires `CGO_ENABLED=1` and a C toolchain (tree-sitter grammars), so a cold
  `go build ./...` is slow. Expect it; it is not a failure.

## Testing

**`internal/mcpserver/schema_test.go` — new regression test (TDD candidate).** Write it first, watch
it fail, then remove the fields. Assert that the derived input schema for every registered tool
carries no `targetDir` property at the call-wide level. Prefer driving this off the real
registration path rather than the five input structs individually, so a future tool that
reintroduces the property is caught too. Keep the existing
`TestInputSchemaFor_CallWideAdditionalPropertiesUntouched`-style assertion
(`schema_test.go:106-111`) intact — it is what makes a stray `targetDir` a whole-call failure, and
this task depends on it.

**`internal/mcpserver/callcontext_test.go`.** Delete the cases that exercise an override (including
the relative-path-absolutisation case, which loses its subject with `effectiveTargetDir`). Retain
and, if not already present, add a case asserting `resolveCall` yields
`callContext.TargetDir == cfg.TargetDir` and a `StateDir` derived from it — that is the invariant
the whole task now rests on.

**Transport-level tests (`transport_test.go`, `transport_errors_test.go`, `stdio_lsp_test.go`).**
Add one end-to-end case: a call carrying `targetDir` is rejected as a whole-call error, not a
per-entry `status: "error"`. This is the behavioural claim of the `hard-removal-not-graceful-ignore`
decision and it is only observable through the transport, since it is the SDK's validator that
rejects it. Do not assert the SDK's exact error string — assert that the call failed as a whole and
that no `results` array came back.

**Per-tool tests (`tools_lsp_test.go`, `tools_impact_test.go`, `tools_assert_test.go`,
`tools_toc_test.go`).** Remove override-specific cases. Where a test used an override to point at a
fixture directory, convert it to setting `Config.TargetDir` at construction — the coverage of "the
tool resolves relative paths against the target directory" is worth keeping, only the mechanism for
setting that directory changes.

**`tools_toc_test.go` specifically.** Keep a case proving an **absolute** `target` still resolves
outside the launch root (`cli.ResolveTOCPath` ignores `targetDir` for an absolute arg). That is the
partial escape hatch named in `cross-repo-escape-hatch-is-a-second-server`, and it should be
pinned by a test so the documentation stays true.

**No test is needed for** the launch flag, `ResolveLaunchTargetDir`, `workspaceKey`, or daemon
isolation — none of them change, and all are covered today.

**Verification.** `go build ./...` and `go test ./...` (cgo build required). A final
`grep -rn 'targetDir' internal/mcpserver/ docs/mcp-setup.md` should return only intentional
survivors — Go-level identifiers and prose about the concept, never a schema property name.

## Q&A log

- **Q:** Drop the per-call `targetDir` override, keep it as a documented escape hatch, or keep as-is? **A:** [auto-pick] Remove it entirely from all seven tools. **Why:** an optional property is still in the published schema and still costs the model a decision per call, so a "documented hatch" does not deliver the ergonomic win; the benchmark already had to suppress it with a prompt line *and* a fatal gate, and at `serverVersion 0.1.0` with no external consumers there is no deprecation obligation.
- **Q:** Does `buildTags` get the same treatment? **A:** [auto-pick] No — out of scope, unchanged. **Why:** `targetDir` is workspace identity (one per server by definition); `buildTags` is genuine per-query scoping that the CLI exposes per-verb, and removing it would delete real capability with no replacement short of a restart.
- **Q:** How should a call that still sends `targetDir` behave? **A:** [auto-pick] Hard removal — the whole call fails on the SDK's call-wide `additionalProperties: false`. **Why:** `inputSchemaFor` already leaves call-wide `additionalProperties` strict by design (asserted at `schema_test.go:106-111`), so this is inherited behaviour with no new code; silently ignoring it would return results from the wrong root while the model believed otherwise.
- **Q:** Does `--target-dir` become required at launch? **A:** [auto-pick] No — keep the cwd default. **Why:** cwd inheritance is what makes per-project scoping automatic and is load-bearing for dropping the per-call override; requiring the flag would break the committed argument-free `.mcp.json`.
- **Q:** Build the "low-overhead repointing" helper the task body asks for? **A:** [auto-pick] No — document the cwd-inheritance contract in `docs/mcp-setup.md` instead. **Why:** the body's premise ("hand-editing `.mcp.json`'s `args`") does not hold — the committed `.mcp.json` passes no `--target-dir` at all, and each session already launches its own correctly-rooted server, so there is nothing to repoint.
- **Q:** What is the escape hatch for a genuinely cross-repo query? **A:** [auto-pick] A second named server entry with an explicit `--target-dir`, documented only. **Why:** MCP clients already support multiple named servers and `workspaceKey` proves the instances are isolated; documenting the real cost beats keeping a schema property to cover it.
- **Q:** What about the `targetDir` mentions inside entry-field description strings? **A:** [auto-pick] Reword them to name "the server's target directory". **Why:** after removal they point at a parameter that no longer exists, in text the model reads as its only documentation.
- **Q:** Keep `effectiveTargetDir` as a passthrough, or delete it? **A:** [auto-pick] Delete it; drop `resolveCall`'s override parameter and have the toc handlers read `cfg.TargetDir` directly. **Why:** it degenerates to a one-line return with an unreachable `filepath.Abs` error path, since `cfg.TargetDir` is already absolute and guarded by `NewServer`.
- **Q:** What happens to the bench ladder's `targetDir` prompt line and fatal gate? **A:** [auto-pick] Drop the `targetDir` half of the prompt line at `ladder_config.py:383`; keep `gates.py`'s check on both keys. **Why:** the prompt line goes stale and distorts a benchmark that measures ergonomic overhead, while the gate is free and becomes a live end-to-end assertion that the property is gone; editing an existing file in the sanctioned Python exception is not extending it.
