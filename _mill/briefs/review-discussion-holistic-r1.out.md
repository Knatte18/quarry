MILL_REVIEW_BEGIN
# Review: Improve gopls query precision (build tags + scoping)

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (claude-opus-5), Anthropic
reviewed_file: _mill/discussion.md
date: 2026-08-27
```

## Findings

### [BLOCKING:design] Call is single-flight, not ID-multiplexed
**Section:** `### verification-concurrency`
**Issue:** The decision's premise — "`Call` is already request-ID-multiplexed over a read loop ... concurrent in-flight requests are within its existing design" — is false in source: `lspclient.go:341` does an unsynchronized `c.nextID++`, `writeMessage` (`:263`) has no write mutex so two concurrent frames interleave, and `Call`'s loop (`:353-376`) silently *drops* any response whose ID does not match its own, with no pending-request map. Eight concurrent verification calls would race the ID counter, corrupt framing, and have callers consume each other's responses.
**Fix:** Decide explicitly whether `lsp.Client` gains a pending-request registry plus a write mutex (and add that to Scope, which currently touches `lsp` only for the `Initialize` signature), or drop to sequential verification under one phase deadline.

### [BLOCKING:design] Tag keying does not cover the native strategy
**Section:** `### daemon-keyed-by-tag-set` / `## Scope` (Out)
**Issue:** `EnsureServer` falls back to `ensureNative` on *any* `ensureSupervised` failure (`ensureserver.go:67-73`), on every platform, and `README.md:38` says windows uses native for every language. `ensureNative` spawns `gopls -remote=auto`, whose shared daemon identity is not a function of `stateDir` at all, so the `tags-<hex>` segment partitions nothing on that path — exactly the "unverified assumption about gopls's internals" the decision exists to avoid. Scope's "windows native-strategy fallback" also mischaracterizes native as windows-only.
**Fix:** State the disposition for the native path (accept shared-daemon tag mixing and say so, or reject native when tags are non-empty), and correct the Scope wording to "native fallback on all platforms".

### [BLOCKING:consistency] Hard-error placement contradicts its own test siting
**Section:** `### unsupported-language-hard-error` (Note) vs `## Testing`
**Issue:** The Note places the error "after `DetectLanguage` resolves the entry but before `acquireConnection`" — both are engine-internal steps inside `query.lookup` (`refs.go:176`, `:181`) — while Testing assigns the test to `internal/cli`. `internal/cli` never calls `DetectLanguage` today (only `quarry/facade.go:104` re-exports it), so the CLI-sited check implies a second, duplicate detection pass with its own `ErrNoLanguage` surface.
**Fix:** Pick one home for the check (engine `lookup`/`Symbol`, or a CLI-side `quarry.DetectLanguage` call accepting double detection) and align the Testing bullet to it.

### [BLOCKING:design] New entry point's return shape omits the declaration set
**Section:** `### one-connection-verification-entry-point` / `### filter-ordering`
**Issue:** `filter-ordering` keeps `filterUnexpectedCallers` (`cli.go:671`) as the last stage, which needs `declRefs` in the public 1-based form; verification keeps the declaration site (its own definition matches the declaration set), so the CLI must still exclude it. Replacing the two `quarry.Definition` + `quarry.References` calls with one entry point leaves the declaration set with no stated route back to the CLI, and the `--no-verify` path's shape (same entry point with filtering off, or today's two calls) is unstated.
**Fix:** State the entry point's return contract (filtered references plus the declaration set) and which code path `--no-verify` takes.

### [NIT:scope] Two-daemon resource cost not stated
**Section:** `### daemon-keyed-by-tag-set`
**Issue:** The rejected alternative names respawn thrash, but the chosen option's own cost — two concurrently resident gopls daemons each indexing the whole module when an agent alternates tagged/untagged queries, both held for `daemonIdleTimeout` (10m, `ensureserver.go:139`) — is never stated as an accepted trade.
**Fix:** Record the memory/indexing cost as an accepted consequence.

### [NIT:decision] `tags-<hex>` derivation unspecified
**Section:** `### daemon-keyed-by-tag-set` / `## Testing`
**Issue:** The segment is described only as "derived from the normalized set", while the neighbouring `workspaceKey` pins sha256-first-6-bytes (`paths.go:63-64`); the hermetic test asserting byte-identical and distinct paths needs the derivation pinned.
**Fix:** Name the hash and truncation length.

## Verdict

REQUEST_CHANGES
Concurrency premise is false in source; native-path keying, error placement, and entry-point contract unresolved.
MILL_REVIEW_END
