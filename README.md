# quarry

quarry is an LSP-backed code intelligence tool.
It answers "where is this symbol referenced / defined / declared" across five languages — Go, Python, C#, TypeScript, and Rust — by speaking the Language Server Protocol to each language's own server rather than reimplementing a parser per language.

## Verbs

quarry exposes four verbs, each with a batch-argument mode, and each printing a JSON `output.Ok`/`output.Err` envelope:

- `refs <symbol|file:line:col>` — every reference to the symbol.
- `definition <symbol|file:line:col>` — the symbol's definition site.
- `symbol <query>` — workspace symbol search.
- `assert-no-callers <symbol|file:line:col>` — an exit-code gate for delete/move safety, with `--except` and `--within` flags to scope the check.

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

## State

Daemon state (`daemon.json`, `daemon.lock`, `daemon.sock`) is per-workspace and resolved with the following precedence:

1. `--state-dir <path>` — an explicit directory on the command line.
2. `$QUARRY_STATE_DIR` — an explicit directory from the environment.
3. `os.UserCacheDir()/quarry/<workspace-key>/` — a per-OS cache directory keyed by the target workspace, so two concurrent `quarry` processes targeting the same directory find the same daemon.

## Testing

quarry has two test tiers:

- `go test ./...` — the hermetic tier. No external binary required.
- `go test -tags lsp ./...` — the live tier. Requires a real language-server binary (e.g. `gopls`) on `$PATH`.
