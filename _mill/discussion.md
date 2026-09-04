# Discussion: MCP, thin (T6)

```yaml
task: MCP, thin (T6)
slug: mcp-thin
status: discussing
parent: main
```

## Problem

The measurement this whole rewrite exists to produce (T7) drives a headless `claude -p` session
against a quarry MCP server and asks whether a directory table of contents pays for itself. The
harness that drives it (T2) is merged, the engine (T3) is merged, and the facade plus the `toc` CLI
verb (T5a) is merged. The one missing piece on the critical path is the server binary itself:
`bench/loomyard-eval/ladder/ladder-toc.yaml` names `build: ./cmd/quarry-mcp`, and that directory
does not exist on `main`. T7 cannot run until it does.

**Why now:** T6 is wave 4 on the critical path `T0 → T1 → T3 → T5a → T6 → T7`. Everything before it
has merged. Nothing else blocks T7.

The task is deliberately small and deliberately constrained. Plan §7 ranks MCP last of four
surfaces and calls it "a mirror of the CLI for an LLM that has the tools granted, and kept thin:
only the tools a ladder cell measures, `toc` first". Plan §12's T6 row spells the same thing: one
`toc` tool over the facade, JSON in `content[].text`, no text view, no `resolve` or `expand` until a
ladder cell measures them. The temptation this task must resist is building the surface V1 had.

## Scope

**In:**

- A new `internal/mcpserver` package: the MCP server construction and the single `toc` tool handler,
  written against the `quarry/` facade only.
- A new `cmd/quarry-mcp/main.go`: flag parsing, root resolution, one `quarry.Open`, server
  construction, stdio transport, exit-on-error. Nothing else.
- A new `internal/repopath` package holding the repository-root discovery and target-relativisation
  logic currently unexported inside `internal/cli` (`discoverRoot`, `resolveRoot`, `repoRelTarget`),
  so the MCP server and the CLI share one implementation instead of two.
- `internal/cli` refactored to call `internal/repopath` instead of its own copies. No change to the
  CLI's observable behaviour: same messages, same exit codes, same golden bytes.
- A new module dependency: `github.com/modelcontextprotocol/go-sdk` v1.7.0 (plus its transitive
  requirements) added to `go.mod` / `go.sum`.
- Tests: in-process MCP client↔server tests, a golden test on the `toc` tool's payload bytes, a
  not-found/error-path test, and a layering test that the MCP packages reach the engine only through
  the facade.
- A short MCP section in `README.md`.
- One operator-run §9a probe against the built binary (connect, `toc` call, allowlist denial),
  recorded in this task's `_mill/` artifacts, not in the tracked tree.

**Out:**

- Any second tool. No `resolve`, no `expand`, no file/dir split, no compact or text view. Plan §12
  and the task body both state that a second tool is a ladder cell first, not code.
- `structuredContent` and an output schema on the tool result.
- Batching: no `targets` array. One target per call, as T5a's discussion settled for the CLI verb
  and as plan §5 now records.
- Any change to `quarry/`, `internal/engine`, `glyph/`, or the ladder harness under
  `bench/loomyard-eval/ladder/`. The harness is merged and its contracts are inputs to this task,
  not outputs of it. In particular `ladder-toc.yaml` is **not** edited: its `server:` block carries
  no `args`, and that stays true.
- Any daemon, cache, parser pool, or long-lived state. Plan §10 phase-1 non-goals apply here exactly
  as they apply to the facade.
- Prompt or task-file changes for the ladder cells.
- `docs/mcp-setup.md` or any other new documentation file.

## Decisions

### D1 — Protocol implementation: the official Go SDK, v1.7.0

- **Decision:** depend on `github.com/modelcontextprotocol/go-sdk v1.7.0` and build the server with
  `mcp.NewServer` + `mcp.StdioTransport`.
- **Rationale:** V1's `cmd/quarry-mcp` used exactly this SDK at exactly this version (see
  `origin/v1-final:go.mod` and `origin/v1-final:cmd/quarry-mcp/main.go`), and plan §9a's probe of
  2026-09-03 — the probe whose reproduction is this task's done-when — was verified against a server
  built on it. Pinning the same version removes protocol compatibility from the risk surface of the
  last task before the measurement. The SDK also supplies the in-memory transport the tests in D9
  depend on.
- **Rejected:** hand-rolling stdio JSON-RPC 2.0 (initialize / notifications/initialized /
  tools/list / tools/call). It would be roughly 300 lines and zero new dependencies, which fits the
  repo's taste, but a subtle mismatch with Claude Code's client — a protocol version, a capability
  field, an initialisation ordering rule — would not fail loudly; it would fail as a cell that never
  called its tool, which the harness reports as a prompt-cost measurement rather than as an error.
  That failure mode is exactly what a critical-path task must not introduce. Revisit only if the
  dependency itself becomes a problem.

### D2 — Package layout: `internal/mcpserver` plus a thin `cmd/quarry-mcp`

- **Decision:** the server lives in `internal/mcpserver`; `cmd/quarry-mcp/main.go` does flags, root
  resolution, one `quarry.Open` (D17), one `NewServer` call, one `Run` call, and `os.Exit` on error,
  and nothing else.
- **Rationale:** it mirrors the split already in the tree — `cmd/quarry/main.go` is six lines over
  `internal/cli` precisely so every byte the binary emits is testable in-process without a build
  step (see that file's header comment). The same argument applies here and is stronger: an MCP
  server tested through `os/exec` is tested through a pipe protocol, which is slow and awkward;
  tested in-process through the SDK's in-memory transport it is a table test. V1 used this same
  split (`internal/mcpserver` + `cmd/quarry-mcp`).
- **Rejected:** everything in `package main`. Fewer files, but `package main` is where the CLI
  deliberately put nothing, and a layering test (D10) cannot be written against a `main` package
  from outside it.

### D3 — Root resolution: cwd discovery, with `--root` as the override

- **Decision:** the server resolves its repository root at startup, once, before any handler can
  run: `--root <path>` when given, otherwise git-root discovery upward from the process's working
  directory. The resolved root is always absolute. A root that does not resolve is a startup failure
  (stderr message, non-zero exit), never a per-call error.
- **Rationale:** the harness runs the measured `claude` process with `Dir: dest`, the pinned
  Loomyard worktree (`bench/loomyard-eval/ladder/internal/ladder/run.go`, `invokeMeasuredProcess`).
  A stdio MCP server is a child of that process and inherits its working directory, so cwd discovery
  finds the pinned worktree with no configuration at all — which is why `ladder-toc.yaml`'s
  `server:` block can carry no `args` and this task can leave the harness untouched. `--root` exists
  for two concrete reasons, not speculatively: `MCPConfigDocument` already implements a
  `{target_dir}` placeholder substitution into the server argument list, documented in
  `bench/loomyard-eval/ladder/internal/ladder/mcp.go` as "a new contract this plan defines… the
  substitution has no consumer until the MCP-server task writes one" — this task is that task — and
  an operator wiring the server into a client that spawns it from an arbitrary directory has no
  other way to say which repository it serves.
- **Rejected:** requiring `--root`. It would force an edit to `ladder-toc.yaml` (adding
  `args: ["--root", "{target_dir}"]`) inside a task whose scope excludes the harness, and it would
  make the natural single-project case need configuration. Also rejected: cwd only, no flag, which
  leaves the placeholder contract dead and the server unusable from a client that does not chdir.

### D4 — Identity: server `quarry`, tool `toc`

- **Decision:** the server's `mcp.Implementation` name is `quarry`; the single tool is named `toc`.
  Together they produce the fully qualified tool name `mcp__quarry__toc`.
- **Rationale:** the harness already fixes both halves. `ladder-toc.yaml` declares
  `server: {name: quarry}`, `quarry_tools: [toc]`, and per-cell `allowed: [toc]`; `Ladder.MCPPrefix`
  composes `"mcp__" + ServerName() + "__"`, and `invokeMeasuredProcess` passes
  `--allowedTools mcp__quarry__toc`. Any other spelling means the granted cell's tool is never
  allowed and T7 measures nothing. Plan §7 also fixes the shape of the name: tool names are verbs,
  never protocol methods, so `toc` — not `tools/toc`, not `table_of_contents`, not `toc_dir`.
- **Rejected:** nothing seriously. Recorded because it is a hard external contract that a plan must
  not paraphrase.

### D5 — Input schema: `target`, `depth`, `symbols` — the CLI verb, mirrored

- **Decision:** the tool's input object is exactly:
  - `target` (string, **required**) — a repository-relative path to a directory or file; `""` and
    `"."` both mean the repository root.
  - `depth` (integer, optional) — how far a directory query recurses; `-1` means the whole tree.
  - `symbols` (boolean, optional) — whether each file entry carries its symbols. Absent means the
    engine's per-target default (true for a file target, false for a directory target).
- **Rationale:** plan §7 defines MCP as a mirror of the CLI, and `quarry toc <target> [--depth N|all]
  [--symbols|--no-symbols]` is that CLI verb. Plan §5 fixes one target per call. `depth` and
  `symbols` are the same verb's own knobs, not additional tools, so exposing them does not touch the
  "thin means thin" rule, which is about tool count.
- **Rejected:** a `targets` array with per-entry status envelopes, as V1 had
  (`origin/v1-final:internal/mcpserver/tools_toc.go`). That whole per-entry status vocabulary exists
  to make a batch call partially fail; a one-target call has an answer or an error, which `isError`
  already expresses (D8). Also rejected: `target` alone with no knobs, which would make MCP a
  different surface from the CLI for no stated reason.

### D6 — `depth`: an integer, `-1` for the whole tree; no `"all"` sentinel string

- **Decision:** `depth` is a JSON integer. `-1` recurses to the bottom of the tree. The property
  description spells that out. There is no string form and no union type.
- **Rationale:** `quarry.DepthAll` is already `-1` (`internal/engine/answer.go:238`), so the wire
  value and the Go constant are the same value and the handler needs no mapping table. V1 recorded
  the cost of the alternative directly in code: `tools_toc.go`'s `docSentences` comment notes that
  5 of 6 benchmarked sessions that supplied its `"all"` sentinel copied the quoted word out of the
  schema description as `"\"all\""` — nested quotes — and burned a round trip on the validation
  error. On a task whose entire purpose is a token-cost measurement, a schema that provokes wasted
  round trips is a measurement defect.
- **Rejected:** a `oneOf: [integer, "all"]` union (LLM clients handle unions badly and the SDK's
  schema generation makes them awkward), and a second `depth_all` boolean (two properties for one
  concept).

### D7 — Success payload: `content[0].text` is `quarry.RenderJSON(answer)` verbatim

- **Decision:** a successful call returns exactly one content block, a text block whose text is the
  byte-for-byte output of `quarry.RenderJSON(answer)` — the §4 envelope, two-space indent, HTML
  escaping off, one trailing newline. No `structuredContent`. No output schema on the tool. No
  wrapper object around the answer, no echoed `target`, no `status` field.
- **Rationale:** the task body says "JSON in `content[].text`" and plan §12 repeats it. Reusing
  `RenderJSON` rather than re-encoding means the MCP payload and the CLI's stdout are the same bytes
  for the same query, which is what "mirror of the CLI" has to mean concretely and which makes a
  golden test cheap. Declaring `structuredContent` as well would send the identical payload twice in
  one result; the whole point of T7 is counting tokens, so duplicating the payload into every
  transcript would corrupt the metric the task exists to feed.
- **Rejected:** `structuredContent` alongside the text block (token duplication, as above); a
  quarry-invented envelope with `status`/`result` keys, as V1 had (that shape existed to serve
  batching, which D5 rejects).

### D8 — Errors: `isError: true` with the facade's failure envelope as the text

- **Decision:** every failure of a `toc` call returns a normal `CallToolResult` with `isError` set
  and one text content block whose text is `quarry.RenderErrorJSON(msg)` — the compact
  `{"ok":false,"error":"<msg>"}` plus a newline. Message wording follows the CLI's, verbatim where
  the condition is the same:
  - target outside the repository → `target outside repository: <target as given>`
  - target not found → `target not found: <repository-relative path>`
  - anything else (stat failure, engine failure, render failure) → `internal error: <err>`

  A malformed call the SDK rejects against the input schema is the SDK's own error and is left
  alone. No JSON-RPC protocol error is ever synthesised for a query outcome.
- **Rationale:** `isError` is MCP's analogue of the CLI's non-zero exit code, and the CLI's own rule
  is that stdout always carries a parseable object even on failure (`internal/cli/cli.go`, `fail`).
  Keeping the same rule here means a client always gets JSON it can parse, and the CLI's four-way
  exit-code taxonomy (0/1/2/3) collapses to the one bit MCP actually has without inventing a second
  vocabulary. Reusing `RenderErrorJSON` keeps the failure bytes shared with the CLI the same way
  D7 shares the success bytes.
- **Rejected:** returning `ok: false` inside a non-error result (an agent has no reason to read a
  payload field when the protocol has a flag for exactly this); raising a JSON-RPC error for
  not-found (that channel is for protocol faults, and clients surface it as a tool malfunction
  rather than as an answer).

### D9 — The `symbols` knob changes what T7 compares; record the caveat, do not hide the knob

- **Decision:** ship `symbols` (per D5) and state in the task's handoff — and, when T7 writes it, in
  `results/<date>-toc/conclusion.md` — that V1's `toc_dir` tool had no symbols knob, so a by-id
  comparison of `a2-toc-dir` against `results/2026-08-30` is no longer strictly like-for-like on the
  agent's available actions.
- **Rationale:** the cross-root comparison rule already in force (`HANDOFF.md` §3, plan §9a) is that
  cost numbers compare only within one results root; correctness may be compared by id. This
  decision narrows the second half slightly and says so, which is cheaper and more honest than
  bending the surface. Suppressing a knob the CLI has, purely to preserve a comparison, would make
  MCP a different surface from the CLI — which contradicts plan §7 — and would hide the divergence
  rather than record it.
- **Rejected:** hiding `symbols` from the schema.

### D10 — Layering: the MCP packages reach the engine only through `quarry/`

- **Decision:** neither `internal/mcpserver` nor `cmd/quarry-mcp` imports `internal/engine` or
  `internal/engine/treesitter`. Every engine type, constant, and sentinel they need comes through
  the `quarry/` facade's aliases. A test enforces this mechanically.
- **Rationale:** the task body states it as a constraint ("the facade is the only dependency — no
  engine import in the MCP binary beyond what `quarry/` re-exports"), and plan §7 ranks the facade
  as the primary surface with the CLI and MCP as mirrors of it. `internal/cli` already obeys the
  same rule. V1 enforced it with a `layering_test.go`; the same mechanism applies.
- **Rejected:** relying on review to catch a stray import.

### D11 — Shared root discovery: lift it into `internal/repopath`

- **Decision:** move `discoverRoot`, `resolveRoot`, and `repoRelTarget` out of `internal/cli`
  (`root.go`, `target.go`) into a new `internal/repopath` package with exported names. The moved
  functions return plain errors — a `repopath`-owned sentinel or error type for the "not a
  directory" / "no repository root found" conditions and `quarry.ErrTargetOutsideRepo` for the
  escape condition — and `internal/cli` wraps them back into its own unexported `usageError` at its
  call sites so its exit-code mapping and its messages are unchanged. `internal/mcpserver` uses the
  same functions for the same jobs.
- **Rationale:** the MCP server needs both behaviours — resolve a root from a flag or cwd, and turn
  a caller's target into a clean repository-relative path while rejecting escapes — and they are
  ~60 lines of subtle path handling (`filepath.Rel` semantics, the `".."` prefix check, `Lstat` on
  `.git`, the `"."`-means-root convention) that must not diverge between the two surfaces. They are
  currently unexported and return an unexported error type, so they cannot be reused as they stand.
- **Rejected:** copying them into `mcpserver` (two implementations of the same rule is exactly the
  drift this repo's facade discipline exists to prevent); exporting them from `internal/cli` and
  importing `internal/cli` from `internal/mcpserver` (V1 did this and it made the MCP server depend
  on the CLI's own concerns; a leaf package with no other content is cleaner).

### D12 — Startup output: one stderr line naming the resolved root; stdout is the transport

- **Decision:** the process writes nothing to stdout except framed MCP traffic. At startup, after
  the root resolves, it writes one line to stderr naming the absolute resolved root. Fatal startup
  errors go to stderr and exit non-zero.
- **Rationale:** stdout belongs entirely to the stdio transport — V1's `cmd/quarry-mcp` header
  comment makes the same point and it is the one way this binary can fail catastrophically and
  silently. The stderr line exists because a server rooted at the wrong directory is the most likely
  misconfiguration in a ladder run and is otherwise invisible: the harness tees only stdout to
  `transcript.jsonl`, so the line cannot pollute a transcript, and an absolute machine path on
  stderr is not a tracked file and does not violate the no-machine-paths constraint.
- **Rejected:** total silence (a misrooted server then presents as a cell whose answers are all
  not-found, with nothing to diagnose from); any stdout write whatsoever.

### D13 — The §9a probe: automated protocol tests plus one operator-run live probe

- **Decision:** two things, not one.
  1. Automated, in `go test`: connect and handshake over the SDK's in-memory transport,
     `tools/list` returning exactly one tool named `toc` with the D5 schema, a `toc` call returning
     the D7 bytes, and the D8 error paths. These run in the done gate on every change.
  2. Live, once, by the operator or the implementing agent at the end of the task: build the binary,
     write an MCP config pointing at it, and run `claude -p` with `--mcp-config`,
     `--strict-mcp-config`, `--output-format stream-json --verbose`,
     `--no-session-persistence`, `< /dev/null`, per plan §9a's table. Confirm three things from the
     stream: the server appears connected in the `system` init record; a `toc` call runs and returns
     the envelope; and a call to `toc` under an allowlist that does not grant it is refused and
     recorded in `permission_denials`. The transcript goes to gitignored `.scratch/`; the outcome is
     recorded in the task's `_mill/` artifacts and in the merge commit message.
- **Rationale:** the done-when is "the harness probe of §9a runs against it", which is a live claim
  about a real client and cannot be made by an in-process test. But a `go test` that shells out to
  `claude -p` costs real money on every run of the done gate, needs network and a logged-in CLI, and
  is flaky — it would make `go test ./...` unrunnable as the repo's gate. Splitting the gate keeps
  the fast check fast and the live check honest.
- **Rejected:** a network-dependent `go test`; skipping the live probe on the strength of the
  in-process tests.

### D14 — Allowlist denial with a single tool

- **Decision:** the denial half of the probe is performed by running the probe session with an
  allowlist that does not name `toc` — either empty, or naming a tool the server does not expose —
  while the server is still declared in the MCP config. The `toc` call is then denied and lands in
  `permission_denials`.
- **Rationale:** §9a's original probe demonstrated denial with `toc_file`, a second tool that has no
  successor in the thin server's surface (`ladder-toc.yaml`'s header comment says exactly this about
  the file-level toc tool). What the gate is actually testing is that the allowlist mechanism binds
  the server's tools at all; denying the only tool there is tests that as well as denying a second
  one would.
- **Rejected:** shipping a second, no-op tool purely so something can be denied — a direct violation
  of the task's own "thin means thin" constraint; dropping the denial half of the gate.

### D15 — Documentation: a README section, no new doc file

- **Decision:** add a short section to `README.md` covering what the server is, that it exposes one
  tool, and a generic `.mcp.json` snippet using a placeholder path. No `docs/mcp-setup.md`.
- **Rationale:** plan §2 deleted V1's `docs/mcp-setup.md` along with the rest of the V1 surface and
  kept the README as a stub. A one-tool server does not need a document; it needs four lines and an
  example. The snippet must use a placeholder, never a real path, per the no-machine-paths
  constraint.
- **Rejected:** recreating `docs/mcp-setup.md`; no documentation at all (the binary would then be
  undiscoverable outside the harness).

### D16 — No facade change

- **Decision:** `quarry/` is not modified. `Open`, `(*Repo).TOC`, `TOCOptions`, `DepthAll`,
  `RenderJSON`, `RenderErrorJSON`, and the three error sentinels are the entire surface this task
  consumes, and they all exist on `main` today.
- **Rationale:** verified against `quarry/repo.go`, `quarry/render.go`, and `quarry/quarry.go` in
  this worktree. If the implementation finds it needs something more from the facade, that is a
  signal the design above drifted, and it is worth stopping to check rather than adding to `quarry/`
  silently.
- **Rejected:** pre-emptively adding a convenience constructor or a combined "open and answer"
  helper to the facade for the server's benefit.

### D17 — Open the repository once, at startup

- **Decision:** `cmd/quarry-mcp` calls `quarry.Open(root)` exactly once, at startup, after the root
  resolves and before the transport starts, and hands the resulting `*quarry.Repo` to the server.
  A failure to open is a startup failure (D12): stderr, non-zero exit, no server. The handler never
  opens a repository.
- **Rationale:** `quarry.Repo`'s doc comment states it is safe for concurrent use by multiple
  goroutines because it holds only the engine handle, which holds only the root string and reads the
  filesystem fresh on every query (`quarry/repo.go`). So there is nothing per-call to gain by
  re-opening, and a per-call open would spread one failure mode — an unopenable root — across every
  call instead of failing once, loudly, at startup where D12's stderr line already reports the root.
  The CLI opens per invocation only because a CLI invocation *is* one call.
- **Rejected:** opening per call. Recorded as a contingency, not an alternative: if the handler's
  not-found path turns out to need a fresh open (it should not — the engine reads the filesystem
  fresh per query, so a target created after startup is found by a later call), that is a
  discrepancy with the facade's documented contract and is worth stopping and raising rather than
  quietly switching designs.

## Technical context

**What already exists on `main` and must be used unchanged:**

- `quarry/repo.go` — `Open(root string) (*Repo, error)` requires an **absolute path naming an
  existing directory**; it performs no git discovery and no cwd resolution (its doc comment says so
  explicitly: root discovery is the caller's job). `(*Repo).TOC(target string, opts TOCOptions)`
  takes a **repository-relative** target, where `""` and `"."` both mean the root, and returns the
  engine's own answer and error unchanged. `Repo` is safe for concurrent use — relevant because an
  MCP server may handle calls concurrently.
- `quarry/render.go` — `RenderJSON(a DirAnswer) ([]byte, error)` produces the §4 envelope:
  two-space indent, `SetEscapeHTML(false)`, exactly one trailing newline, key order from the
  struct declaration order in `internal/engine/answer.go`. `RenderErrorJSON(msg string) []byte`
  produces `{"ok":false,"error":"<msg>"}` plus a newline, no spaces after the colons, and cannot
  fail.
- `quarry/quarry.go` — the aliases (`DirAnswer`, `FileEntry`, `Symbol`, `Kind`, `TOCOptions`), the
  `Kind` constants, `DepthAll = -1`, and the three sentinels `ErrTargetNotFound`,
  `ErrTargetOutsideRepo`, `ErrLanguageUnsupported`. The sentinels are the engine's own values, not
  copies, so `errors.Is` works transitively without importing the engine.
- `internal/engine/answer.go:241` — `TOCOptions{Depth int; Symbols *bool}`. `Symbols` is a
  **pointer**: `nil` selects the per-target default (true for a file target, false for a directory
  target); a non-nil value overrides it at every depth. The MCP `symbols` property must therefore
  map an absent property to `nil`, not to `false`.

**The pipeline to mirror:** `internal/cli/cli.go`'s `Run` is the reference ordering, and the MCP
handler performs the same steps minus the flag parsing and the exit codes: relativise the target,
`Lstat` it (not `Stat` — a symlink named as the target is treated as a file and not followed,
matching the engine's own `resolveTarget`), open the repo, call `TOC`, map the error, render. Note
that `Run` opens the repo per invocation. The server does **not**: see D17. `codeForTOCError` in
that file is the error classification to mirror: `ErrTargetNotFound` and
`ErrTargetOutsideRepo` are answers, everything else is internal.

**What is being moved (D11):**

- `internal/cli/root.go` — `discoverRoot(startDir string)` walks up looking for a `.git` entry via
  `Lstat` and fails with `no repository root found above <startDir>; pass --root`;
  `resolveRoot(flagRoot, cwd string)` absolutises and `Stat`s a given `--root`, failing with
  `--root is not a directory: <flagRoot as given>`.
- `internal/cli/target.go:28` — `repoRelTarget(root, base, target string)` joins a relative target
  against `base`, `filepath.Rel`s it against `root`, rejects `".."` and `"../"` prefixes with
  `quarry.ErrTargetOutsideRepo`, and returns a slash-form cleaned path (`"."` for the root itself).
- Both currently return `usageError` (unexported, `internal/cli/flags.go`), which is why they cannot
  be reused as they stand. Note the `base` parameter: the CLI passes `cwd` normally and `root` when
  `--root` was given. The MCP server has no per-call cwd, so it always passes `root` as `base` —
  targets are repository-relative by definition on this surface.

**Harness contracts that constrain this task (all merged, all read-only here):**

- `bench/loomyard-eval/ladder/ladder-toc.yaml` — `server: {name: quarry, build: ./cmd/quarry-mcp}`,
  no `args`, no `env`. `quarry_tools: [toc]`. Cells `a2-toc-dir` and `b8-toc-dir` have
  `allowed: [toc]`; `a0-none` and `b0-none` have `allowed: []`.
- `.../internal/ladder/mcp.go` — `BuildServer` runs `go build -o <out> ./cmd/quarry-mcp` with
  `CGO_ENABLED=1` and cwd at the quarry repo root, then sha256s the binary for `provenance.json`.
  `MCPConfigDocument` writes `{"mcpServers":{"quarry":{"command":"<binary>","args":[...]}}}`, with
  `{target_dir}` substituted into each arg. A control cell gets an empty `mcpServers` map — so the
  server is not even declared for `a0-none`.
- `.../internal/ladder/run.go`, `invokeMeasuredProcess` — runs `claude` with `Dir: dest` (the pinned
  worktree), `--tools <builtins>`, `--allowedTools mcp__quarry__toc` for granted cells,
  `--mcp-config <path> --strict-mcp-config --output-format stream-json --verbose
  --no-session-persistence --setting-sources ""`, stdin from `/dev/null`, stdout teed to
  `transcript.jsonl`. **Stderr is not captured** — hence D12's line is safe.
- `.../internal/ladder/config.go:113` — `MCPPrefix()` returns `"mcp__" + ServerName() + "__"`.
- `.../internal/ladder/stream.go:150` — `ResultRecord.PermissionDenials` is parsed from the final
  result record; this is where D14's denial is observed.

**V1 as reference, not as a starting point:** `origin/v1-final` has `cmd/quarry-mcp/main.go` (57
lines) and `internal/mcpserver/` (mcpserver.go, tools_toc.go, result.go, facade.go, callcontext.go,
lspentry.go, nativeentry.go, layering_test.go, transport_test.go and more). Read `mcpserver.go` for
the `mcp.NewServer(&mcp.Implementation{Name: "quarry", Version: ...}, nil)` construction and
`main.go` for the `server.Run(ctx, &mcp.StdioTransport{})` shape; read `layering_test.go` and
`transport_test.go` for test mechanics. Do **not** port `tools_toc.go`, `result.go`, `nativeentry.go`
or `lspentry.go` — their batching, per-entry status vocabulary, compact view, `docSentences`, `lang`
override and daemon config resolution are all V1 surface this task deletes by not rebuilding.

**Build:** the binary links tree-sitter through the facade, so it needs `CGO_ENABLED=1`, which is
what `BuildServer` sets. `internal/cgoguard` is already blank-imported by
`internal/engine/treesitter` and so is transitively in this binary's build graph — no action needed.

**Gate:** `pipeline.done_gate` for this repo is `go test ./... && golangci-lint run`. There is no
`.golangci.yml` in the tree, so lint runs with defaults. Adding a dependency means `go.sum` changes;
`go mod tidy` must leave the tree clean.

## Constraints

There is no `CONSTRAINTS.md` at the hub root. The constraints below come from the task body,
`CLAUDE.md`, and the plan.

- **Go only.** No Python anywhere in this repo (`CLAUDE.md`).
- **No machine paths in any tracked file.** The README snippet (D15) uses a placeholder. The
  operator probe's config and transcript live in gitignored `.scratch/`.
- **Thin means thin.** One tool. If a second one feels necessary during implementation, that is a
  ladder cell first, not code — stop and raise it rather than adding it.
- **The facade is the only engine dependency of the MCP binary** (D10), enforced by test.
- **stdout is the transport.** Nothing but framed MCP traffic may reach it (D12).
- **The ladder harness is not modified by this task.** `ladder-toc.yaml` in particular keeps its
  empty `server.args`.
- **The CLI's observable behaviour does not change** despite the D11 refactor: same messages, same
  exit codes, same golden outputs under `docs/research/output-formats/after/`.
- **Done gate:** `go test ./... && golangci-lint run` green.

## Testing

TDD candidates are marked. The engine and facade are already tested; nothing here re-tests them.

**`internal/repopath` (TDD candidate — pure, no I/O beyond `Lstat`/`Stat`):** the move is a
refactor, so the existing table tests in `internal/cli/root_test.go` and
`internal/cli/target_test.go` move with the functions and are the starting point. Cover: discovery
finding `.git` at the start directory, at an ancestor, and nowhere (the failure message);
`--root` given as relative and absolute, pointing at a file, and pointing at nothing; target
relativisation for a plain relative path, an absolute path inside the root, the root itself
(yields `"."`), a `".."` escape, and an absolute path outside the root. `internal/cli`'s own tests
stay and must still pass unchanged — that is the regression gate on the refactor.

**`internal/mcpserver` — protocol surface (TDD candidate):** using the SDK's in-memory transport,
connect a client to the server and assert on `tools/list`: exactly one tool, named `toc`, with
`target` required and `depth`/`symbols` optional, and `depth` typed integer. This is the test that
catches an accidental second tool and a schema drift, both of which would silently break T7.

**`internal/mcpserver` — the happy path (golden):** against a small committed fixture repository at
`internal/mcpserver/testdata/` — a directory with a couple of Go files and a subdirectory is enough
— call
`toc` for a directory target, a file target, a `depth` value, a `depth: -1` value, and `symbols`
true and false, and compare `content[0].text` byte-for-byte against golden files.

The fixture style is the engine's, not the CLI's, and that choice is deliberate:
`internal/engine/testdata/` holds committed trees (`tree/`, `units/`, `methods/`, `tiebreak/`,
`loomyard/` and others) and is the precedent to follow. `internal/cli` has no `testdata/` at all —
its fixture-needing tests construct trees programmatically through `writeScratchTree`
(`internal/cli/scratchtree_test.go`), which writes under the gitignored `.scratch/cli-tests/`. That
helper is an unexported test helper in `package cli` and is not reachable from another package's
tests, so following the CLI here would mean duplicating it; a committed tree also makes golden bytes
obviously stable, which is the property these tests exist for. Go's toolchain ignores `testdata/`
when building, so committed `.go` fixture files are inert.

One of these tests must
assert the stronger property directly: for the same target and options, the MCP text and
`quarry.RenderJSON` of the facade's own answer are identical bytes. That single assertion is what
makes "mirror of the CLI" testable rather than aspirational.

**`internal/mcpserver` — absent-property semantics:** a call omitting `symbols` must produce the
engine's per-target default (file target → symbols present, directory target → absent), not
`symbols: false`. This is the pointer-vs-bool trap in `TOCOptions` and deserves its own named test.

**`internal/mcpserver` — error paths:** a target that does not exist, a target escaping the
repository (`"../elsewhere"`), and a target that is a broken symlink. Each must return `isError`
true, exactly one text block, and text equal to `quarry.RenderErrorJSON(<the CLI's own wording>)`.
Assert the wording, not just the flag — D8's whole claim is that the two surfaces say the same
thing.

**Layering (TDD candidate — cheap and load-bearing):** a test that walks the import graph of
`internal/mcpserver` and `cmd/quarry-mcp` (via `go list`-style package inspection, as V1's
`layering_test.go` does) and fails if `internal/engine` or `internal/engine/treesitter` appears.

**`cmd/quarry-mcp`:** keep it thin enough that it needs no test of its own beyond the package
compiling — that is the point of D2. If flag parsing grows past `--root`, it moves into
`internal/mcpserver` and gets a table test there.

**Not covered by `go test`:** the live §9a probe (D13 item 2), by design. Its result is recorded in
the task's `_mill/` artifacts and named in the merge commit.

## Q&A log

- **Q:** Which MCP protocol implementation — official Go SDK or hand-rolled stdio JSON-RPC? **A:** [auto-pick] `github.com/modelcontextprotocol/go-sdk` v1.7.0. **Why:** V1 used exactly this version and §9a's probe was verified against it; a protocol mismatch on the last critical-path task before the measurement fails silently as "the cell never called its tool".
- **Q:** Package layout — `internal/mcpserver` + thin `cmd/quarry-mcp`, or everything in `package main`? **A:** [auto-pick] `internal/mcpserver` + thin `cmd/quarry-mcp`. **Why:** mirrors the existing `internal/cli` + `cmd/quarry` split, keeps the server testable in-process, and lets a layering test run against the package from outside it.
- **Q:** How does the server learn its repository root? **A:** [auto-pick] cwd discovery, with `--root` as an override. **Why:** the harness runs `claude` with cwd at the pinned worktree so zero configuration works, and `--root` gives `mcp.go`'s already-implemented `{target_dir}` placeholder the consumer its comment says the MCP task must write.
- **Q:** Server name and tool name? **A:** [auto-pick] server `quarry`, tool `toc` → `mcp__quarry__toc`. **Why:** `ladder-toc.yaml` and `MCPPrefix()` already fix both halves; any other spelling means the granted cell's tool is never allowed.
- **Q:** Startup output on stdout/stderr? **A:** [auto-pick] nothing on stdout ever; one stderr line naming the resolved root. **Why:** stdout is the transport, and a misrooted server is otherwise invisible — the harness tees only stdout, so the line cannot pollute a transcript.
- **Q:** Input schema — full CLI mirror, `target` only, or V1-style batching? **A:** [auto-pick] full CLI mirror: `target` required, `depth` and `symbols` optional. **Why:** §7 makes MCP a mirror of the CLI and §5 fixes one target per call; `depth`/`symbols` are the same verb's knobs, not extra tools.
- **Q:** How is `depth: all` spelled on the wire? **A:** [auto-pick] integer, `-1` means the whole tree. **Why:** `quarry.DepthAll` is already `-1`, and V1's code records that 5 of 6 sessions mis-quoted its `"all"` sentinel and wasted a round trip — unacceptable in a token-cost measurement.
- **Q:** Success payload — text only, or text plus `structuredContent`? **A:** [auto-pick] `content[0].text` = `quarry.RenderJSON` bytes, nothing else. **Why:** the task says JSON in `content[].text`; `structuredContent` would duplicate the payload into every transcript and corrupt the token metric T7 exists to produce.
- **Q:** How does a failed query come back? **A:** [auto-pick] `isError: true` with `quarry.RenderErrorJSON(msg)` as the text, CLI wording verbatim. **Why:** `isError` is the protocol's analogue of the CLI's exit code, and reusing the failure envelope keeps both surfaces saying the same thing.
- **Q:** Does exposing `symbols` break T7's by-id comparison against the 2026-08-30 root? **A:** [auto-pick] ship the knob and record the caveat for T7's conclusion. **Why:** cost is already only comparable within a root; narrowing the correctness comparison honestly beats bending the surface to preserve it.
- **Q:** With one tool, what does the allowlist-denial gate deny? **A:** [auto-pick] run the probe with an allowlist that does not grant `toc`, and observe it in `permission_denials`. **Why:** §9a's `toc_file` has no successor in the thin surface, and shipping a no-op tool purely to be denied violates the task's own "thin means thin".
- **Q:** How is the §9a probe run as a gate? **A:** [auto-pick] automated in-process protocol tests in `go test`, plus one operator-run live `claude -p` probe recorded in `_mill/`. **Why:** a `go test` that shells to `claude -p` costs money per run, needs network, and would make the repo's done gate unrunnable.
- **Q:** What is the automated test surface? **A:** [auto-pick] in-memory-transport client↔server tests: `tools/list` shape, golden bytes on `content[0].text`, absent-`symbols` default semantics, the error paths with their wording, and a layering test. **Why:** it catches exactly the drifts that would silently invalidate T7, and runs in the ordinary gate.
- **Q:** Documentation? **A:** [auto-pick] a short `README.md` section with a placeholder-path `.mcp.json` snippet; no new doc file. **Why:** plan §2 deleted V1's `docs/mcp-setup.md` and kept the README a stub; a one-tool server needs four lines, and the no-machine-paths constraint rules out a real path.
- **Q:** Does `quarry/` need anything added? **A:** [auto-pick] no facade change. **Why:** `Open`, `TOC`, `TOCOptions`, `DepthAll`, `RenderJSON`, `RenderErrorJSON` and the three sentinels all exist on `main` and cover the whole need; needing more would be a signal the design drifted, not a licence to extend the facade.
- **Q:** How do the MCP server and the CLI share root discovery and target relativisation? **A:** [auto-pick] lift `discoverRoot`/`resolveRoot`/`repoRelTarget` into a new `internal/repopath` package, with `internal/cli` wrapping their errors back into its own `usageError`. **Why:** they are unexported and return an unexported type today, and duplicating ~60 lines of subtle path handling across two surfaces is exactly the drift the facade discipline exists to prevent.
