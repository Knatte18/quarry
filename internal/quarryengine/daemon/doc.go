// Package daemon implements the EnsureServer seam, the Go toolchain manager, and the supervised
// strategy's runtime state file.
//
// # The EnsureServer seam
//
// ensureserver.go implements EnsureServer(ctx, lang, entry, targetDir,
// stateDir, timeout, initOptions) (*lsp.Client, ConnKind, error): given a
// registry entry whose HasNativeDaemon field is true, it resolves, spawns
// or dials, and hands back an already-initialized, already-probed
// connection ready for immediate use. entry.HasNativeDaemon is the gate
// that decides whether a language ever calls into this machinery at all —
// in V1 that is Go alone. Python, C#, TypeScript, and Rust all leave
// HasNativeDaemon at its zero value and never call EnsureServer; their
// callers keep using lsp.NewClient + a manual client.Initialize +
// Close()/Kill() in query's acquireConnection (refs.go), byte-for-byte the
// same code path this engine has always run, completely untouched by V1.
// initOptions is the already-rendered initializationOptions map
// (registry.RenderInitializationOptions's result: nil for an untagged
// query, non-nil for a tagged one); EnsureServer threads it to
// client.Initialize on every path and, on the native strategy, also uses
// its non-nilness to decide whether to spawn a private gopls (see
// ensureNative's and nativeArgv's doc comments).
//
// Two strategies implement the seam: ensureNative (native, Go's production
// path — for an untagged query, spawn gopls -remote=auto, a disposable
// local proxy subprocess; gopls itself dedups and owns the real shared
// daemon behind it, kept warm via an explicit -remote.listen.timeout
// override — see daemonIdleTimeout in ensureserver.go — sized for an
// agent's own reasoning gaps between calls, not gopls's 1-minute
// human-editing-rhythm default) and ensureSupervised (supervised — quarry
// owns a state file, an advisory spawn-race lock, a deterministic socket
// path, and detached-spawn/restart logic for a language server with no
// shared-daemon mode of its own). EnsureServer dispatches Go to
// ensureSupervised as its live V1 strategy: it resolves the toolchain once,
// then attempts supervised, falling back to ensureNative on any supervised
// error (a toolchain-resolution failure itself never reaches the fallback,
// since it is returned before ensureSupervised is ever attempted).
// ensureNative remains fully built, unit-tested, and integration-tested as
// this fallback — its own dedicated integration test still drives it
// directly, proving the -remote=auto proxy path independently of the
// supervised dispatch above it.
//
// Connection teardown differs by ConnKind, and getting this wrong is a
// protocol-correctness bug, not a style choice:
//
//   - ConnKindNative: safe to Close()/Kill(), exactly like the legacy path.
//     For an untagged query, what ensureNative hands back is quarry's own
//     disposable -remote=auto proxy subprocess for this one call, not the
//     shared daemon behind it — closing it ends only this session, never
//     gopls's real shared instance. For a tagged query, ensureNative spawns
//     a private, unshared gopls instead (no -remote=auto at all), so there
//     is no shared daemon behind it in the first place — closing it ends
//     that private instance outright, which is still the correct, safe
//     teardown; it is simply not "ending only this session" in the
//     untagged case's sense.
//   - ConnKindSupervised: never Close() or Kill() it. The connection is a
//     dial into a daemon quarry spawned to outlive this call — the entire
//     point of the supervised strategy — so the LSP graceful-shutdown
//     handshake Close() would send (shutdown+exit) is meaningless network
//     chatter at best (the daemon is meant to keep serving other callers)
//     and a needless RPC round trip quarry has no reason to spend. The
//     dialed socket's file descriptor is reclaimed by the OS when this
//     one-shot process exits a moment later.
//   - ConnKindLegacy: unchanged from before this task — the caller closes
//     the real server subprocess it directly owns, since it never went
//     through EnsureServer at all.
//
// A wedged daemon — live but hung, or never finished binding its listen
// socket, neither of which daemonStale's PID-plus-protocol-version check
// alone detects — no longer strands every caller indefinitely: see
// ensureSupervised's own doc comment for the re-dial-under-lock-then-
// one-restart escalation that recovers it, now that Go's registry entry
// dispatches here as a live V1 caller.
//
// # Go toolchain manager
//
// toolchain.go resolves (and, on a cold cache, installs) the Go language
// server. $PATH is never consulted for Go: resolveGoToolchain installs a
// pinned gopls version — the resolved registry.Entry's PinnedVersion field
// (registry.go's builtins()["go"] in V1), an exact version, not "latest" —
// into os.UserCacheDir()/quarry/tools/go/<version>, and ensureNative always
// launches that resolved binary, never whatever "gopls" happens to resolve
// to on the operator's PATH. The install itself is fenced by its own
// advisory lock (goToolchainInstallLock), a lock distinct from the daemon
// spawn-race lock ensureSupervised uses — an install-in-progress and a
// daemon-spawn-in-progress are unrelated races that must not serialize on
// each other.
//
// This cache root is a third path axis, distinct from the config and state
// axes internal/cli resolves (see "Daemon state and concurrency" below): it
// is machine-global, not workspace-scoped, so this file hand-joins
// os.UserCacheDir() directly rather than accepting a told directory. That
// is deliberate, not an oversight — a pinned gopls binary is shared across
// every workspace on the machine, which is the entire point of
// pinning-and-caching it once rather than per workspace.
//
// # Daemon state and concurrency
//
// daemonstate.go implements the supervised strategy's runtime state: a JSON
// state file plus a paired advisory lock per (stateDir, lang), resolved via
// this package's own DaemonStateFile/DaemonLock, which join only the
// language segment and the filename onto a told leaf stateDir —
// internal/cli resolves that directory via the
// --state-dir/$QUARRY_STATE_DIR/os.UserCacheDir() precedence documented in
// README.md. This state is ephemeral and machine-bound: a live daemon's
// PID, socket path, and spawn time mean nothing on another machine or after
// this one is rebooted, so it must never be treated as durable,
// version-controlled configuration.
//
// A recorded daemon is stale, forcing a fresh spawn rather than a reuse,
// under a two-part check (daemonStale): its PID is no longer alive
// (proc.IsAlive), or its recorded ProtocolVersion does not match this
// binary's supervisedProtocolVersion. That protocol version is quarry's own
// wire-compatibility marker for the supervised daemon protocol itself —
// bumped when a future quarry change to that protocol needs it — and must
// not be confused with gopls's own version, which registry.Entry.PinnedVersion
// pins separately.
//
// The daemon's socket path is a deterministic function of (stateDir,
// lang), never randomly chosen at spawn time. This is what makes
// stale-socket cleanup across restarts simple: a fresh spawn can always
// remove-if-exists the one predictable path before binding, with no
// separate bookkeeping needed to remember what the previous socket was
// called.
//
// A caller that keeps losing the spawn race, or keeps finding a state that
// reads healthy but never becomes dialable, is bounded by
// quarryengine.ErrServerSpawnTimeout: ensureSupervised's whole retry loop is
// wrapped in a deadline, and a losing caller that never observes a healthy,
// dialable daemon before that deadline expires gets this error rather than
// blocking indefinitely.
package daemon
