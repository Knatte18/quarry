# MCP setup

quarry exposes its seven tools over the Model Context Protocol (MCP), stdio transport, through the
`cmd/quarry-mcp` binary.
This document covers what the committed `.mcp.json` does,
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
- `--config` — an explicit `servers.yaml` overlay path, mirroring the CLI's `--config` flag.
- `--state-dir` — an explicit daemon state directory, mirroring the CLI's `--state-dir` flag.
- `--timeout` — the deadline applied per entry's facade call.
