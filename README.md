# quarry

quarry is an LSP-backed code intelligence tool.
It answers "where is this symbol referenced / defined / declared" across five languages — Go, Python, C#, TypeScript, and Rust — by speaking the Language Server Protocol to each language's own server rather than reimplementing a parser per language.

## Verbs

quarry exposes four verbs, each with a batch-argument mode, and each printing a JSON `output.Ok`/`output.Err` envelope:

- `refs <symbol|file:line:col>` — every reference to the symbol.
- `definition <symbol|file:line:col>` — the symbol's definition site.
- `symbol <query>` — workspace symbol search.
- `assert-no-callers <symbol|file:line:col>` — an exit-code gate for delete/move safety, with `--except` and `--within` flags to scope the check.

All four verbs accept `--build-tags <a,b>` to scope the query to a Go build-tag set (see Configuration below); `--no-verify`, which opts out of `assert-no-callers`'s default-on declaration verification (see below), is `assert-no-callers`-only.

## Building and running

```
go build -o quarry ./cmd/quarry
./quarry refs mySymbol
```

Or run directly without a separate build step:

```
go run ./cmd/quarry refs mySymbol
```

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

That "on `$PATH`" is the whole story only when gopls happens to be there. Every `//go:build lsp` test in this repo skips via `exec.LookPath("gopls")`, so on a machine where the pinned binary lives in the toolchain cache instead — `os.UserCacheDir()/quarry/tools/go/<version>/` — the live tier reports success having exercised nothing. To actually run it, prepend that directory to `$PATH`:

```
PATH="$HOME/.cache/quarry/tools/go/v0.23.0:$PATH" go test -tags lsp ./...
```

(substitute your platform's `os.UserCacheDir()` and the pinned version currently in [`docs/servers.yaml.example`](docs/servers.yaml.example) if they differ from the linux/`v0.23.0` values above). Silence looks like a pass here — a live-tier run that skips every test still exits 0 — so getting the `$PATH` prefix right is what makes the run mean anything.

## Verification

`assert-no-callers` verifies each candidate caller by default: it resolves each candidate reference's own definition and keeps only the references that resolve back to the queried symbol's declaration, so an interface-method check is precise without needing a scoping flag. Verification is fail-closed — a reference it cannot verify (a timed-out or errored definition lookup, or an unusable declaration side) stays a violation rather than being silently dropped. `--no-verify` reinstates the older, noisier behaviour: every gopls reference reported unfiltered by declaration.
