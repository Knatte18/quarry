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
`workspaceKey(targetDir)` (`internal/cli/paths.go:76-86`), a hash of the **absolute path**, so one
long-lived server already isolates multiple worktrees and repos cleanly, with separate state file,
lock, socket, and gopls process per distinct `targetDir`, and no restart needed. Nothing here is a
correctness fix. This is purely an API-surface decision.

## Scope

**In:**

- Remove the call-wide `targetDir` field from all six input structs in `internal/mcpserver/`:
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
  `lspentry.go:55`.
- Reword the Go doc comments that use **per-call** phrasing for the same concept —
  `nativeentry.go:29`, `nativeentry.go:43`, `nativeentry.go:45`, `lspentry.go:19`, and
  `lspentry.go:54` — and, on the test side, `tools_toc_test.go:212` ("never the call's targetDir") —
  all say "**the call's** targetDir", which goes stale for exactly the
  same reason the three sites in "Gotcha — stale doc comments" below do: there is no longer any such
  thing as *the call's* target directory. Reword these to "the server's target directory".
  Go doc comments naming `targetDir` as a plain Go *identifier* (e.g. `nativeentry.go:76`'s
  `query(targetDir string)` parameter, `translate.go:29`'s `resolveEntryFile(targetDir, raw string)`)
  stay unchanged — they name a live function parameter, not a removed tool property.
- Add a regression test asserting no registered tool's published input schema carries a
  `targetDir` property.
- Update `docs/mcp-setup.md`: document that the server is scoped once at launch, that the cwd
  default is what makes per-project scoping automatic, that a second named server entry with an
  explicit `--target-dir` is the cross-repo escape hatch, and that `toc_file`/`toc_dir` retain a
  partial absolute-path escape the five language-server-backed tools do not (the asymmetry spelled
  out in `cross-repo-escape-hatch-is-a-second-server`'s Note — it must not be dropped by a plan
  writer reading Scope alone).
- Update `bench/loomyard-eval/ladder/scripts/ladder_config.py:383-384`'s prompt line. It currently
  reads, across two source lines:

  ```
  Never set targetDir or buildTags on any of these calls -- the server is
  already rooted at the correct target codebase.
  ```

  Replace it with:

  ```
  Never set buildTags on any of these calls -- the default build-tag set is
  the one this run is scoped to.
  ```

  The rationale clause has to change too, not just the key list: "the server is already rooted at
  the correct target codebase" justifies *only* the `targetDir` half, and leaving it attached to a
  `buildTags`-only instruction would be a non-sequitur. Keep the two-line wrap so the surrounding
  prompt block's shape is unchanged.
- Update `bench/loomyard-eval/ladder/tests/test_ladder_config.py:311`, which asserts
  `"Never set targetDir or buildTags" in prompt` and therefore fails the moment the prompt line
  changes. Narrow the asserted literal to exactly `"Never set buildTags on any of these calls"` —
  still a literal assertion, and still short enough to sit on one source line despite the prompt's
  two-line wrap. Keeping it literal is the point: the test exists so the instruction survives
  prompt refactors, and that value holds as long as the literal tracks the prompt.
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
- `serverVersion` (`internal/mcpserver/mcpserver.go:19-20`) stays at `"0.1.0"`. This task does edit
  `mcpserver.go` (the `Config.TargetDir` doc comment), so the omission would otherwise read as an
  oversight: it is deliberate. The constant tracks the package's own development and there is no
  release, changelog, or consumer that a bump would inform — the same absence of external consumers
  that `drop-per-call-targetdir` cites as removing the deprecation obligation is what makes the bump
  pointless here. Do not bump it just because the tool schemas changed shape.
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
  `additionalProperties: false` that inference produced — a decision recorded in the function's own
  doc comment ("the call-wide properties keep whatever inference produced, so a call-level violation
  stays a whole-call failure"). Removing the Go field therefore needs no new production code. The
  resulting error is loud and self-correcting: a model that sends the property gets told it is not
  accepted and retries without it.
- **Strength of the existing evidence (be precise here).** The nearest existing test is
  `TestInputSchemaFor_CallWidePropertySurvives` (`internal/mcpserver/schema_test.go:93`), and it is
  weaker than "this behaviour is already tested": it runs `inputSchemaFor` over the local
  `fixtureCall` fixture and asserts only that `s.AdditionalProperties != nil`
  (`schema_test.go:106-111`). No existing test shows an unknown call-wide key actually failing a
  real tool call at the transport. The behavioural claim this decision rests on is therefore proven
  by the **new** transport case named under Testing, not inherited from an existing assertion. A
  plan writer must not treat it as already covered.
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
- **Every comment naming `effectiveTargetDir` must be dispositioned.** Deleting the function leaves
  dangling references in seven production comment sites, none of which a `targetDir` grep finds
  (the identifier is capitalised differently and appears in prose): `translate.go:23` ("callers only
  ever pass `ResolveLaunchTargetDir`'s or `effectiveTargetDir`'s result"), `callcontext.go:8`,
  `callcontext.go:62`, `tools_toc.go:5`, `tools_toc.go:162`, `tools_toc.go:185`, and
  `nativeentry.go:119` ("the **effective** absolute target directory" — the same concept by
  paraphrase, no identifier). Disposition for all of them: reword to name `Config.TargetDir` / "the
  server's target directory" as the single source, dropping the two-source framing entirely.
  `callcontext.go:42-59` disappears with the function itself. `tools_toc_test.go:370` carries the
  same reference and is rewritten with that file's other changes; `callcontext_test.go:15-39` tests
  the function directly and is deleted per Testing. **After the change, `grep -rn 'effectiveTargetDir'
  internal/mcpserver/` must return zero hits** — this identifier is explicitly *not* on the
  verification grep's intentional-survivor list.

### bench-ladder-update

- **Decision:** Edit `bench/loomyard-eval/ladder/scripts/ladder_config.py:383` to drop the
  `targetDir` half of its "Never set targetDir or buildTags" prompt instruction, and narrow
  `bench/loomyard-eval/ladder/tests/test_ladder_config.py:311`'s literal assertion to match. Leave
  `gate_no_target_override` (`bench/loomyard-eval/ladder/scripts/gates.py:117-136`) checking both
  keys, `fatal=True` unchanged.
- **Rationale:** The prompt line becomes stale the moment the property is gone, and it spends
  prompt tokens telling the model not to do something the schema no longer permits — which for a
  benchmark measuring ergonomic overhead is measuring the wrong thing. The gate is retained because
  it costs nothing and still guards the constraint it was written for: a run must not retarget away
  from its pinned worktree, which for `buildTags` remains fully reachable (that key is untouched by
  this task and still breaks the cold cell's daemon key). Its `targetDir` arm becomes close to
  unreachable rather than redundant, and `fatal=True` stays correct for both keys: the gate reads
  transcript `tool_input` maps, so if the `targetDir` arm ever does fire it means a run diverged
  from the pinned-worktree design and its numbers are not comparable with the rest of the matrix —
  killing it is the honest outcome. Editing existing files inside the already-sanctioned Python
  exception is not "extending" it under `CLAUDE.md`'s rule; no new Python file is created and no new
  Python is introduced anywhere else.
- **What the gate is not:** it is **not** a check that the property is gone from the published
  schema. `gate_no_target_override` inspects what the *model emitted* in the transcript, never the
  server's advertised tool schema, so it would pass unchanged with the property still in place.
  The schema-level guarantee is the `internal/mcpserver` regression test named under Testing; the
  two are independent and neither substitutes for the other.
- **Rejected:** *Leave bench untouched* — leaves a stale instruction in the measured prompt.
  *Also remove the gate's `targetDir` arm* — throws away a free regression assertion.
- **Ordering against #009 (`port-ladder-bench-to-go`, active in a sibling worktree):** that task
  ports `bench/loomyard-eval/ladder/` to Go, which will eventually remove
  `ladder/scripts/ladder_config.py` — the file this task edits one line of. This task is far
  smaller and is expected to merge well before #009's port reaches parity. If #009 lands first, or
  if the two collide in `mill-merge-in`, the conflict resolves **delete-wins**: drop this task's
  one-line prompt edit and carry the same "never set `targetDir`" removal into the Go port's own
  prompt construction instead. Nothing else in this task's scope touches `bench/`.

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
`cli.ResolveStateDir(cfg.StateDir, absTargetDir, tags)` (`internal/cli/paths.go:118-140`) derives
per-workspace state via `workspaceKey` (`internal/cli/paths.go:76-86`), which is the target
directory's base name plus the first 12 hex characters of a SHA-256 over the **cleaned absolute
target path only**. Build tags do not enter that digest: `buildTagsSegment`
(`internal/cli/paths.go:88-99`) is a second, independent `tags-<hex>` path segment that
`ResolveStateDir` appends to the resolved leaf, and only when the tag set is non-empty — two
composed path segments, not one combined hash. `ensureSupervised`
(`daemon/ensureserver.go:328-335`) derives separate state file, lock, and socket paths from the
result. Two worktrees of the same repo therefore never share a daemon, lock, or gopls process.
This is why a long-lived server was already safe with per-call overrides — and equally why one
server per session is safe now.

One wrinkle this refactor quietly *improves*: `ResolveStateDir`'s `--state-dir` and
`$QUARRY_STATE_DIR` tiers bypass `workspaceKey` entirely (documented in its own doc comment), so an
operator who pins `quarry-mcp --state-dir` today gets one state directory shared across every
per-call `targetDir` that server is handed. With the override removed, one server serves exactly
one target directory, so a pinned `--state-dir` can no longer collapse two workspaces onto one
socket. No action required — noted so nobody re-derives it as a bug later.

**Downstream consumers of `callCtx.TargetDir`.** After the change these all still receive the same
value, just always the launch value: `entry.query(callCtx.TargetDir)` in `tools_lsp.go:121`,
`tools_impact.go:77`, `tools_assert.go:85`; `cli.FilterWithin(..., callCtx.TargetDir)` in
`tools_lsp.go:128`, `tools_assert.go:100`; `cli.FilterImpactWithin` in `tools_impact.go:84`;
`exceptSet(callCtx.TargetDir, entry.Except)` in `tools_assert.go:103`; and
`callCtx.options(lang, query)` which threads it into `quarry.Options.TargetDir`
(`callcontext.go:103-114`). None of these signatures change.

**Gotcha — count-based doc comments a token grep cannot find.** Three input-struct doc comments
state the override's existence purely by **count**, containing neither `targetDir` nor `TargetDir`,
so no grep for the token will surface them: `tools_lsp.go:23-24`, `tools_impact.go:18-19`, and
`tools_assert.go:18-19` each say "the **three** call-wide resolution overrides every
language-server-backed tool in this package accepts". After this change that number is two
(`lang`, `buildTags`). All three must be corrected.

This is why the completeness check below cannot be a token grep alone. **Second enumeration
criterion, mandatory:** re-read the doc comment on every input struct in `internal/mcpserver/` —
`lspInput` (`tools_lsp.go:22`), `symbolInput` (`tools_symbol.go:48`), `impactInput`
(`tools_impact.go:18`), `assertInput` (`tools_assert.go:18`), `tocFileInput` (`tools_toc.go:22`),
`tocDirInput` (`tools_toc.go:40`) — and confirm each still describes the struct accurately.
`symbolInput`'s says "the same call-wide resolution overrides" and both toc comments say "plus the
**per-call overrides** `toc_file`/`toc_dir` accepts" (`tools_toc.go:22-24`, `tools_toc.go:40-43`) —
all three are back-references of the same shape, not enumerations, and the toc pair's "per-call
overrides" phrasing is itself the stale wording this task removes. Do not rely on grep to find this
class of staleness.

**Gotcha — stale doc comments.** Several comments describe the override as existing:
`callcontext.go`'s file header ("either the launch default or an absolutised per-call override" at
line 28-30, and the `effectiveTargetDir` paragraph), `mcpserver.go:33-35` ("used whenever a call
omits its own targetDir override"), and `tools_lsp.go:3` ("plus lang/buildTags/targetDir
overrides"). All must be corrected; a grep for `targetDir` and `TargetDir` across
`internal/mcpserver/` after the change should surface only intentional survivors — **necessary but
not sufficient**, since three doc comments state the fact by count and contain neither token; see
the second enumeration criterion above.

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

**Schema regression test — extend the existing matrix, do not add a fixture test (TDD candidate).**
Write it first, watch it fail, then remove the fields. The assertion belongs in
`TestToolsList_PerToolParameterMatrix` (`internal/mcpserver/transport_test.go:167`), which already
does exactly this job: it stands up a real client/server pair via `newConnectedPair`, calls
`ListTools`, and asserts call-wide and entry-level property presence/absence across the registered
tools using its `schemaProperties`/`entryProperties` helpers — including a call-wide-absence loop
for `buildTags` on the two toc tools that the new `targetDir` assertion can be modelled on
directly. Extend it to assert `targetDir` is absent from the call-wide properties of **all seven**
tools.

Do **not** put this in `schema_test.go`: that file exercises `inputSchemaFor[T]` over local
fixtures (`fixtureCall`), never the registration path, so an assertion there could not see a tool
that reintroduced the property. Leave `TestInputSchemaFor_CallWidePropertySurvives`
(`schema_test.go:93-111`) untouched — it pins the call-wide `additionalProperties` behaviour this
task's hard-removal decision depends on, but it is a fixture-level test and is not the regression
guard being added here.

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
`tools_toc_test.go`) — expect no deletions.** `callcontext_test.go:12-39` is the **only** file with
cases that exercise a per-call override. Across these four files every `targetDir`/`TargetDir`
occurrence is either `Config{TargetDir: ...}` at construction (`tools_lsp_test.go:58`,
`tools_toc_test.go:225`) or prose in a comment; none sets an input struct's `TargetDir`. So there is
no override-case removal or conversion work here — the fixture-directory mechanism these tests
already use is the one that survives. The only new work in this group is the toc absolute-path case
below.

**`tools_toc_test.go` specifically.** **Add** a new case — there is none today; every existing toc
test roots its fixture at `cfg.TargetDir` — proving an **absolute** `target` still resolves outside
the launch root (`cli.ResolveTOCPath` ignores `targetDir` for an absolute arg). That is the
partial escape hatch named in `cross-repo-escape-hatch-is-a-second-server`, and it should be
pinned by a test so the documentation stays true.

**Bench ladder pytest suite (`bench/loomyard-eval/ladder/tests/`).** One assertion changes:
`test_ladder_config.py:311`'s `assert "Never set targetDir or buildTags" in prompt` must be narrowed
to the reworded literal, or the prompt edit fails it. `test_gates.py:89` (which asserts `"targetDir"`
appears in the gate's finding messages) is unaffected — the gate keeps both arms. Run the ladder
suite once as part of verification to confirm exactly these, then leave it alone; it is not part of
this task's own test surface beyond that single assertion.

**No test is needed for** the launch flag, `ResolveLaunchTargetDir`, `workspaceKey`, or daemon
isolation — none of them change, and all are covered today.

**Verification.** `go build ./...` and `go test ./...` (cgo build required). Then the ladder suite,
using the invocation its own README documents (`bench/loomyard-eval/ladder/README.md:361`) — run
from the repo root, no `PYTHONPATH` prefix, since `conftest.py` handles `sys.path`:

```
uv run --no-project --with pytest --with pyyaml python -m pytest bench/loomyard-eval/ladder/tests -q
```

Finally, `grep -rni 'targetdir' internal/mcpserver/ docs/mcp-setup.md` — case-insensitive, so it
matches `TargetDir` as well as `targetDir`, agreeing with the completeness rule stated under
Technical context. It should return only intentional survivors: Go identifiers (`Config.TargetDir`,
`callContext.TargetDir`, `query(targetDir string)`) and prose about the server's target directory,
never a schema property name and never per-call phrasing.

Then `grep -rn 'effectiveTargetDir' internal/mcpserver/`, which **must return zero hits** — that
identifier has no intentional survivors (see `delete-effectivetargetdir`). It is deliberately
excluded from the "Go identifiers" whitelist above, which otherwise reads as if any identifier
spelling passes.

Neither grep is sufficient on its own — run the second enumeration criterion from Technical context
(re-read all six input structs' doc comments) alongside them, because the count-based comments
contain neither spelling of `targetDir` and `nativeentry.go:119` paraphrases the deleted helper
without naming it.

## Q&A log

- **Q:** Drop the per-call `targetDir` override, keep it as a documented escape hatch, or keep as-is? **A:** [auto-pick] Remove it entirely from all seven tools. **Why:** an optional property is still in the published schema and still costs the model a decision per call, so a "documented hatch" does not deliver the ergonomic win; the benchmark already had to suppress it with a prompt line *and* a fatal gate, and at `serverVersion 0.1.0` with no external consumers there is no deprecation obligation.
- **Q:** Does `buildTags` get the same treatment? **A:** [auto-pick] No — out of scope, unchanged. **Why:** `targetDir` is workspace identity (one per server by definition); `buildTags` is genuine per-query scoping that the CLI exposes per-verb, and removing it would delete real capability with no replacement short of a restart.
- **Q:** How should a call that still sends `targetDir` behave? **A:** [auto-pick] Hard removal — the whole call fails on the SDK's call-wide `additionalProperties: false`. **Why:** `inputSchemaFor` already leaves call-wide `additionalProperties` strict by design (asserted at `schema_test.go:106-111`), so this is inherited behaviour with no new code; silently ignoring it would return results from the wrong root while the model believed otherwise.
- **Q:** Does `--target-dir` become required at launch? **A:** [auto-pick] No — keep the cwd default. **Why:** cwd inheritance is what makes per-project scoping automatic and is load-bearing for dropping the per-call override; requiring the flag would break the committed argument-free `.mcp.json`.
- **Q:** Build the "low-overhead repointing" helper the task body asks for? **A:** [auto-pick] No — document the cwd-inheritance contract in `docs/mcp-setup.md` instead. **Why:** the body's premise ("hand-editing `.mcp.json`'s `args`") does not hold — the committed `.mcp.json` passes no `--target-dir` at all, and each session already launches its own correctly-rooted server, so there is nothing to repoint.
- **Q:** What is the escape hatch for a genuinely cross-repo query? **A:** [auto-pick] A second named server entry with an explicit `--target-dir`, documented only. **Why:** MCP clients already support multiple named servers and `workspaceKey` proves the instances are isolated; documenting the real cost beats keeping a schema property to cover it.
- **Q:** What about the `targetDir` mentions inside entry-field description strings? **A:** [auto-pick] Reword them to name "the server's target directory". **Why:** after removal they point at a parameter that no longer exists, in text the model reads as its only documentation.
- **Q:** Keep `effectiveTargetDir` as a passthrough, or delete it? **A:** [auto-pick] Delete it; drop `resolveCall`'s override parameter and have the toc handlers read `cfg.TargetDir` directly. **Why:** it degenerates to a one-line return with an unreachable `filepath.Abs` error path, since `cfg.TargetDir` is already absolute and guarded by `NewServer`.
- **Q:** What happens to the bench ladder's `targetDir` prompt line and fatal gate? **A:** [auto-pick] Drop the `targetDir` half of the prompt line at `ladder_config.py:383`; keep `gates.py`'s check on both keys. **Why:** the prompt line goes stale and distorts a benchmark that measures ergonomic overhead, while the gate costs nothing and still guards the constraint it was written for — a run must not retarget away from its pinned worktree, which stays fully reachable via `buildTags`. The gate is explicitly *not* a check that the property left the schema: it reads transcript `tool_input` maps only. That guarantee comes from the `internal/mcpserver` regression test instead. Editing an existing file in the sanctioned Python exception is not extending it.
