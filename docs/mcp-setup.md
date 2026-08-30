# MCP setup

quarry exposes its seven tools over the Model Context Protocol (MCP), stdio transport, through the
`cmd/quarry-mcp` binary.
This document covers what the committed `.mcp.json` does,
the server's scoping contract and its escape hatches,
what to expect on a cold start,
what a missing C toolchain looks like from the client's side,
and the warm-start alternative.

## What the committed `.mcp.json` does

The repository root's `.mcp.json` is a project-scoped MCP server declaration.
A Claude Code session opened anywhere inside this repository can connect to the `quarry` server with
no separate install step,
after the one-time trust prompt Claude Code shows before it will run a project-scoped server.

`.mcp.json` runs `go run ./cmd/quarry-mcp` with no `--target-dir` argument.
The server takes its own process working directory as the default target directory,
absolutises it once at startup,
and reports the resolved path on stderr as `quarry-mcp: resolved target directory <path>` —
so the effective default is always visible, even though `.mcp.json` never states it explicitly.

## The server's scoping contract

The server's target directory is fixed once, at launch, by `--target-dir` or by the process's own
working directory when that flag is absent.
There is no per-call way to change it — no tool accepts a target directory as an input property.
`ResolveLaunchTargetDir` is the function that resolves it, once, before any handler can run, and
`NewServer`'s absolute-path guard is what enforces that the resolved value is always absolute by
the time a handler sees it.

Cwd inheritance is what makes per-project scoping automatic, and it is the reason no per-call
override is needed.
Where a client launches a project-scoped server with the project root as the server process's own
working directory — which is what the committed argument-free `.mcp.json` relies on — each
worktree's session gets its own server process rooted at its own worktree, with its own daemon
state directory and its own gopls, needing no configuration and no repointing.
That is client behaviour this repository does not control and cannot verify, so treat it as scoped
to clients that launch a project-scoped server from the project root, and confirm it for a given
session by reading the `quarry-mcp: resolved target directory <path>` line the server writes to
stderr, rather than assuming it.

The automatic state-directory isolation this depends on holds for the default, user-cache tier of
state-directory resolution.
An explicit `--state-dir` or a set `$QUARRY_STATE_DIR` becomes the leaf verbatim, with the target
path never entering the key, so two servers pinned to one explicit state directory do share it —
see the `--state-dir` bullet under `## Launch-only flags` below.

The escape hatch for a genuinely cross-repository or cross-worktree query is a second named server
entry in the client's own MCP configuration, with an explicit `--target-dir` pointing at the other
root:

```json
{
  "mcpServers": {
    "quarry": {
      "command": "go",
      "args": ["run", "./cmd/quarry-mcp"]
    },
    "quarry-other": {
      "command": "go",
      "args": ["run", "./cmd/quarry-mcp", "--target-dir", "/path/to/other/repo"]
    }
  }
}
```

This is the one capability the design costs: a session rooted at one repository cannot ask about
another through the same server entry.

`toc_file` and `toc_dir` retain a partial escape the five language-server-backed tools do not.
Those two accept an absolute target path, which is resolved as given rather than against the
launch root, so they can read a file or directory outside it.
The five language-server-backed tools cannot: even given an absolute file or URI, the query is
served by the gopls rooted at the server's target directory, so a path outside that root will not
resolve correctly.
Absolute paths are not a general escape hatch — only `toc_file` and `toc_dir` carry this asymmetry.

## Cold-start behaviour

This is expected behaviour, not a bug.

quarry requires `CGO_ENABLED=1` and a C toolchain to build, because the `toc` verbs' tree-sitter
backend links C grammars (see the main [README](../README.md)'s "Build dependency: cgo and a C
toolchain" section).
`go run ./cmd/quarry-mcp` on a cold build cache therefore triggers a cgo compile before the server
can speak to its client at all,
and that compile can exceed an MCP client's connect timeout.

The practical consequence: the first connection attempt after a fresh clone,
or after `go clean -cache`,
may fail or hang.
A retry once the build has finished succeeds,
and every later launch is cache-fast, since `go run` reuses the build cache from then on.

## Missing-toolchain failure mode

If no C toolchain is available at all,
the build does not produce a mysterious linker dump.
It fails at compile time with the guard's own error naming
`quarry_requires_CGO_ENABLED_1_with_a_C_toolchain` (see
`internal/quarryengine/cgoguard_nocgo.go`).
From the client's side this surfaces as the server process exiting immediately,
with that identifier on stderr — the fix it names is exact: install a C toolchain and set
`CGO_ENABLED=1`.

## Warm-start alternative: a pre-built binary

Building the binary once and pointing the client at it directly skips the cgo compile on every
launch:

```
go build -o quarry-mcp ./cmd/quarry-mcp
```

`/quarry-mcp` is gitignored, so the built binary at the repository root never gets committed by
accident.
Point a client's own server configuration at the built binary's path in place of `go run
./cmd/quarry-mcp` to use it.

## Launch-only flags

Four flags configure the server process at launch; every other tool parameter is supplied by the
model per call, not by the launch command line.

- `--target-dir` — the project directory to detect the language in and root the server at.
  Defaults to the process's own working directory.
  It is the only way to set the server's target directory — no per-call property overrides it.
- `--config` — an explicit `servers.yaml` overlay path, mirroring the CLI's `--config` flag.
- `--state-dir` — an explicit daemon state directory, mirroring the CLI's `--state-dir` flag.
- `--timeout` — the deadline applied per entry's facade call.
