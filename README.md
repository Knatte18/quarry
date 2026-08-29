# quarry

quarry is a code intelligence tool with two backends.
For "where is this symbol referenced / defined / declared" across five languages — Go, Python, C#, TypeScript, and Rust — it speaks the Language Server Protocol to each language's own server rather than reimplementing a parser per language; for "what is in this file / directory" it parses source itself with tree-sitter, needing no language server at all.

## Verbs

quarry exposes the following verbs, each with a batch-argument mode, and each printing a JSON `output.Ok`/`output.Err` envelope:

- `refs <symbol|file:line:col>` — every reference to the symbol.
- `definition <symbol|file:line:col>` — the symbol's definition site.
- `symbol <query>` — workspace symbol search.
- `assert-no-callers <symbol|file:line:col>` — an exit-code gate for delete/move safety, with `--except` and `--within` flags to scope the check.
- `impact <symbol|file:line:col>` — every caller of the symbol, each with its enclosing declaration's full line range including the preceding docstring — "what do I have to rewrite", not merely "where is it mentioned".
- `toc file <path>` — a table of contents for a single source file: its package/namespace, header, and every top-level function, method, and type declaration, tree-sitter-backed rather than LSP-backed.
- `toc dir <path>` — the same, one entry per supported file directly inside a directory (no recursion), plus each file's test/generated status where the language has a reliable rule for either.

`--build-tags <a,b>` (see Configuration below) is accepted by every LSP-backed verb — `refs`, `definition`, `symbol`, `assert-no-callers`, and `impact` — while the tree-sitter-backed `toc file`/`toc dir` do not take it; `--no-verify`, which opts out of `assert-no-callers`'s default-on declaration verification (see below), is `assert-no-callers`-only.

## Building and running

```
go build -o quarry ./cmd/quarry
./quarry refs mySymbol
```

Or run directly without a separate build step:

```
go run ./cmd/quarry refs mySymbol
```

## Third exposure layer: MCP

Beside the engine (`quarry/`) and the CLI (`cmd/quarry`),
`cmd/quarry-mcp` exposes quarry's seven tools — `textDocument_definition`, `textDocument_references`,
`workspace_symbol`, `assert_no_callers`, `impact`, `toc_file`, and `toc_dir` — over the Model Context
Protocol.

```
go build -o quarry-mcp ./cmd/quarry-mcp
```

A Claude Code session opened in this repository connects through the committed `.mcp.json` once the
one-time project-server trust prompt is accepted.
See [`docs/mcp-setup.md`](docs/mcp-setup.md) for cold-start behaviour, the warm-start alternative
above, and the launch-only flags.

### Build dependency: cgo and a C toolchain

The `toc` verbs' tree-sitter backend is a cgo binding.
Building quarry — `go build`, `go run`, or `go test` — therefore requires `CGO_ENABLED=1` (the Go
default on every platform this project supports) and a C toolchain on the build machine.
**Nothing extra is required to run the already-built binary or to run its LSP-backed verbs** — the C
toolchain is a build-time dependency only, never a runtime one.

On linux, `gcc` or `clang` from the platform's usual package manager is enough.

On windows there are two supported routes:

- **Natively**, with a mingw-w64 toolchain on `$PATH` — either MSYS2 or TDM-GCC.
- **Cross-compiled from linux or WSL2**:

  ```
  CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc go build -ldflags '-extldflags "-static"' -o quarry.exe ./cmd/quarry
  ```

  This needs the `gcc-mingw-w64-x86-64` package.
  `-static` is deliberate: it makes the produced `quarry.exe` link the mingw C runtime statically, so
  it does not depend on mingw runtime DLLs sitting beside the executable.

  **This recipe is documented but unverified**: no mingw-w64 toolchain was available on the machine
  that wrote this section, so the command above has not actually been run.
  It is recorded as the expected invocation, not a confirmed one; verify it on a machine with
  `gcc-mingw-w64-x86-64` installed before relying on it in a release pipeline.

## Supported platforms

quarry supports **linux and windows**.
darwin is explicitly out: `internal/proc` — the package responsible for killing and detaching daemon subprocesses — has a `proc_linux.go` and a `proc_windows.go` implementation but no darwin implementation, so a build for `GOOS=darwin` does not compile.
Writing a darwin implementation is tracked as a follow-up issue rather than done as part of this port.

## Windows: native strategy only

On linux, quarry's supervised daemon strategy owns a state file, an advisory spawn-race lock, and a deterministic socket path, and spawns the language server detached so repeated calls reuse one running process.
That strategy is unavailable on windows: it hard-codes a Unix domain socket, which windows does not have.
On windows quarry falls back to the native strategy (`gopls -remote=auto`) for every language.
This is the documented fallback path, and it works — but it is not an identical code path to linux, and a caller relying on daemon-specific behavior (for example, the spawn-race lock) should not assume it is present on windows.

A non-empty `--build-tags`/`$QUARRY_BUILD_TAGS` set changes what "native" means: instead of joining the shared `-remote=auto` daemon, quarry spawns a private gopls dedicated to that one call, because the shared daemon's address is not a function of the tag-keyed state directory at all — two callers with different tag sets but no `-remote=auto` flag would otherwise land in the same shared daemon and get whichever tag set it happened to initialize with first. A tagged query on the native path therefore always pays a cold start with no cross-invocation reuse. On windows, since native is the only strategy, this is the normal path for a tagged query, not a degraded fallback.

This compounds with batch mode: passing 2 or more positional arguments to `refs`, `definition`, or `symbol` makes each one an independent engine call, so an N-symbol tagged batch on the native path pays N cold starts within that one invocation.

## Upgrading from `lyx scout`

If you previously ran this tool as `lyx scout`, expect **one** re-download of `gopls` the first time you run `quarry`.
The Go toolchain cache path renamed its leading segment from `lyx` to `quarry` (`os.UserCacheDir()/lyx/tools/go/<version>` became `os.UserCacheDir()/quarry/tools/go/<version>`), which re-keys the cache.
This is expected and not repeated on subsequent runs.

## Configuration

The optional `servers.yaml` overlay is resolved with the following precedence, and an absent file at any tier is not an error:

1. `--config <path>` — an explicit path on the command line.
2. `$QUARRY_CONFIG` — an explicit path from the environment.
3. `os.UserConfigDir()/quarry/servers.yaml` — the per-OS user config directory.
4. The built-in registry — pinned defaults for Go, Python, C#, TypeScript, and Rust.

See [`docs/servers.yaml.example`](docs/servers.yaml.example) for every built-in entry at its default values and for how to add a language.

### `toc file`'s docstring length: `--doc-sentences`

`toc file` trims each symbol's docstring to its first sentence by default.
`--doc-sentences <N|all>` controls how much of it reaches the output: `0` omits the `docstring` key
entirely (the recommended discovery mode — see the verb's own `--help` for the two-phase read flow
this enables), `N` keeps the first `N` sentences, and `all` keeps the docstring unchanged.
This flag exists on `toc file` only: `toc dir` emits headers, never docstrings, so the setting has
nothing to affect there.

The effective value is resolved with the following precedence, and an absent config file at any tier
is not an error:

1. `--doc-sentences` on the command line — governs that one call.
2. `$QUARRY_TOC_CONFIG` — an absolute path to a config file, set per shell or per directory.
3. `.quarry.yaml` in the **target file's own directory only** — a `toc` mapping with a `doc_sentences`
   key holding an integer `≥ 0` or the string `all`. This lookup happens in that directory and
   nowhere else: there is no upward search toward a repository root, and no project detection, unlike
   most other config files a user has met. A `.quarry.yaml` in a parent directory is never consulted.
4. The built-in default, `1`.

`.quarry.yaml` is not `servers.yaml`: it is its own file, decoded with unknown keys rejected as an
error, and it is never loaded by the LSP-backed verbs.

`--build-tags <a,b>` scopes a query to a Go build-tag set, with the same numbered precedence every other flag/environment-variable pair in this document uses:

1. `--build-tags <a,b>` — an explicit comma-separated tag list on the command line.
2. `$QUARRY_BUILD_TAGS` — an explicit comma-separated tag list from the environment.
3. No tags — the default, and the only value every other supported language accepts today.

The resolved tag set is normalized before use — split on commas, each entry trimmed, deduplicated, and sorted — so two different spellings of the same set (`"a,b"` vs `"b, a, a"`) behave identically. Passing a non-empty tag set for a language whose registry entry carries no build-tag template (every built-in language except Go — see [`docs/servers.yaml.example`](docs/servers.yaml.example)) is a hard error, not a silent no-op. The consequence an operator will actually hit: `$QUARRY_BUILD_TAGS` exported globally in a shell, then a query run against a Python project, fails loudly rather than quietly ignoring the flag. This is deliberate — a silently-ignored precision flag is exactly the failure mode `--build-tags` exists to remove.

## State

Daemon state (`daemon.json`, `daemon.lock`, `daemon.sock`) is per-workspace and resolved with the following precedence:

1. `--state-dir <path>` — an explicit directory on the command line.
2. `$QUARRY_STATE_DIR` — an explicit directory from the environment.
3. `os.UserCacheDir()/quarry/<workspace-key>/` — a per-OS cache directory keyed by the target workspace, so two concurrent `quarry` processes targeting the same directory find the same daemon.

A non-empty `--build-tags`/`$QUARRY_BUILD_TAGS` set appends a `tags-<hex>` segment to the resolved leaf directory above, at all three precedence tiers — so each distinct tag set gets its own daemon, socket, state file, and lock, never sharing state with an untagged query or a differently-tagged one. An empty tag set leaves the resolved path exactly as it is today. The accepted cost: alternating tagged and untagged queries against the same workspace leaves two gopls daemons resident until each idles out on its own.

## Testing

quarry has two test tiers:

- `go test ./...` — the hermetic tier. No external binary required.
- `go test -tags lsp ./...` — the live tier. Requires a real language-server binary (e.g. `gopls`) on `$PATH`.

Both tiers now require a C toolchain to build at all, since `go test` compiles the `toc` package's
cgo-backed tree-sitter bindings the same way `go build` does — see "Build dependency: cgo and a C
toolchain" above. Neither tier's own scope changed: the hermetic tier still needs no external binary
and the live tier still needs a real language server. The C toolchain requirement sits underneath
both, at the compile step, not at either tier's own scope.

That "on `$PATH`" is the whole story only when gopls happens to be there. Every `//go:build lsp` test in this repo skips via `exec.LookPath("gopls")`, so on a machine where the pinned binary lives in the toolchain cache instead — `os.UserCacheDir()/quarry/tools/go/<version>/` — the live tier reports success having exercised nothing. To actually run it, prepend that directory to `$PATH`:

```
PATH="$HOME/.cache/quarry/tools/go/v0.23.0:$PATH" go test -tags lsp ./...
```

(substitute your platform's `os.UserCacheDir()` and the pinned version currently in [`docs/servers.yaml.example`](docs/servers.yaml.example) if they differ from the linux/`v0.23.0` values above). Silence looks like a pass here — a live-tier run that skips every test still exits 0 — so getting the `$PATH` prefix right is what makes the run mean anything.

## Verification

`assert-no-callers` verifies each candidate caller by default: it resolves each candidate reference's own definition and keeps only the references that resolve back to the queried symbol's declaration, so an interface-method check is precise without needing a scoping flag. Verification is fail-closed — a reference it cannot verify (a timed-out or errored definition lookup, or an unusable declaration side) stays a violation rather than being silently dropped. `--no-verify` reinstates the older, noisier behaviour: every gopls reference reported unfiltered by declaration.
