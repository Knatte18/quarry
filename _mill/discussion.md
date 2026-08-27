# Discussion: Improve gopls query precision (build tags + scoping)

```yaml
task: Improve gopls query precision (build tags + scoping)
slug: gopls-query-precision
status: discussing
parent: main
```

## Problem

quarry answers "where is this symbol referenced" by asking a language server, and for Go that server is `gopls`. Two precision defects were inherited from loomyard's `internal/scoutengine` when the engine was extracted into this repo, and both were filed as GitHub issues (#2, #1) so they would not be lost in the move. Both trace to the same root cause: gopls's *default* query scope is not the scope a precise safety-gate tool needs.

The first defect is a false-negative one. `lspclient.go`'s `initialize` request sends no `initializationOptions`, so gopls never receives `buildFlags` and only ever loads the default build-tag view of a project. Every reference inside a file behind a `//go:build <tag>` constraint is invisible to every verb. Measured on loomyard — which uses `//go:build integration` / `//go:build smoke` for its test-tier separation — **40% of the true call sites** to the two most heavily-tested benchmark symbols live behind that tag boundary and are never reported (`docs/scout-multilang.md:115`). There is no `--build-tags` flag to thread a tag set through.

The second defect is a false-positive one. `assert-no-callers <interface-method>` with no scoping flag returns every method matching that name and signature across every structurally-compatible interface in the workspace. Measured on loomyard's `internal/builderengine/poll.go` local `clock` interface (`Now() time.Time; Sleep(d time.Duration)`): **31 reported callers, 2 real** — the other 29 belonged to two unrelated `clock` interfaces independently declared in different packages, plus their test files. This is documented gopls behaviour for `textDocument/references` on an interface method, and the wrapper passes the noise through as a genuine violation. A `--within <dir>` flag already scopes the query correctly, but it is opt-in: the caller must already know the risk exists.

**Why now:** `assert-no-callers` exists specifically to replace human/agent judgement with a deterministic exit code — a delete/move safety gate for a mill batch's `verify:` step. A false positive there is worse than one a reviewer might catch by re-reading the code, because the entire point is to make the re-reading unnecessary. And the build-tag blind spot is load-bearing for exactly the kind of build-tag-separated repo this tool is aimed at, including this one.

## Scope

**In:**

- A `--build-tags <a,b>` flag on all four verbs (`refs`, `definition`, `symbol`, `assert-no-callers`), with a `$QUARRY_BUILD_TAGS` environment fallback.
- A new `initialization_options` field on `registry.Entry`, holding a per-language nested map with a `{{tags}}` placeholder; Go's built-in entry populates it, the other four leave it empty.
- Rendering that map (comma-joined tag substitution) and sending it as `initializationOptions` on the LSP `initialize` request, for both `EnsureServer` strategies and the legacy cold-spawn path.
- A hard error when `--build-tags` is passed for a language whose registry entry carries no `initialization_options` template.
- Folding the normalized tag set into the resolved daemon state directory, so each distinct tag set gets its own daemon, socket, state file, and lock.
- A definition-verification filter for `assert-no-callers`: default-on, `--no-verify` to disable, fail-closed on verification failure.
- A new engine entry point that performs resolve → references → per-reference definition verification inside **one** LSP connection, re-exported through `quarry/facade.go`.
- Documentation: `README.md` (new flags, new env var, new state-dir keying), `docs/servers.yaml.example` (the new entry field on every built-in block).

**Out:**

- Any change to the four verbs' JSON envelope shape or exit-code contract. `assert-no-callers` keeps exit 0 / 1 / 2 and the `violation` + `callers` fields exactly as documented in `internal/cli/cli.go`'s package comment.
- Verification on `refs`, `definition`, or `symbol`. Those verbs' behaviour is untouched.
- Per-tag list expansion in the template renderer (the `cargo.features: [a, b]` shape rust-analyzer would want). Only comma-joined `{{tags}}` substitution is implemented.
- `initialization_options` templates for python, csharp, typescript, or rust. Their entries gain the field but leave it empty, so `--build-tags` hard-errors for them.
- Removing or changing `--within` and `--except`. Both stay exactly as they are.
- Any darwin support work, and any change to the windows native-strategy fallback beyond the fact that it also carries `initializationOptions`.
- Reopening issues #1 and #2 on GitHub — both are already CLOSED; this task is the follow-through recorded in the wiki.

## Decisions

### build-tags-cli-surface

- Decision: A per-verb `--build-tags <a,b>` string flag on all four verbs, resolved in the precedence `--build-tags` > `$QUARRY_BUILD_TAGS` > empty.
- Rationale: Matches the existing per-verb `--target-dir` / `--lang` / `--timeout` pattern rather than the root's `--config` / `--state-dir` infrastructure flags — build tags are a query-scoping concern, not process infrastructure. The environment fallback lets a build-tag-separated repo set the tag set once for a whole agent session instead of repeating the flag on every invocation.
- Rejected: A root persistent flag (one definition site, but groups a query flag with infrastructure flags). Per-verb flag with no env fallback (smallest surface; forces repetition). `servers.yaml`-only configuration (makes tags workspace config rather than per-call, and a per-call override is genuinely wanted).

### registry-owns-the-spelling

- Decision: `registry.Entry` gains an `InitializationOptions map[string]any` field (yaml key `initialization_options`) carrying a per-language, YAML-native nested map. Go's built-in entry is `{buildFlags: ["-tags={{tags}}"]}`. The engine deep-copies the map and substitutes the comma-joined tag list into every string value containing the `{{tags}}` placeholder.
- Rationale: This is what issue #2 itself proposes — keep gopls's spelling of the option out of `internal/cli`. A YAML-native nested map avoids stringly-typed JSON embedded in YAML, permits arbitrary nesting, and lets a future language contribute its own shape with no engine-code change. The existing whole-entry-replacement overlay semantics mean adding a field is compatible: an override that omits it simply gets the zero value, consistent with how `pinned_version` / `has_native_daemon` already behave.
- Rejected: Two flat fields (`build_tags_option` + `build_tags_template`) — only expresses a top-level key holding a one-element string list. A single JSON-fragment string field — maximum flexibility, worst readability, parse errors surface late. CLI hard-coding gopls's spelling — one language today, a code change per future language, and explicitly what the issue argues against.

### template-substitution-semantics

- Decision: Substitution is comma-join-and-replace: the normalized tag set is joined with `,` into one string, and every string value anywhere in the deep-copied map that contains `{{tags}}` has the placeholder replaced. No per-tag list expansion. Map keys are never substituted. A template with no `{{tags}}` occurrence is still sent verbatim when tags are absent-or-present (it is static configuration), but see `unsupported-language-hard-error` for how "no template at all" is treated.
- Rationale: YAGNI. rust-analyzer's `cargo.features: [a, b]` shape is the only known case needing per-tag expansion, and no non-Go language gets a template in this task. Implementing expansion now would be designing for a hypothetical.
- Rejected: Per-tag list expansion (a `{{tag}}` singular placeholder expanding its containing list element once per tag) — real machinery for zero current consumers.

### unsupported-language-hard-error

- Decision: Passing `--build-tags` (or having `$QUARRY_BUILD_TAGS` set non-empty) for a language whose resolved registry entry has an empty `initialization_options` is a hard error, raised **before** the language server is spawned, in the standard `output.Err` envelope with exit 1. Message shape: `quarry: --build-tags is not supported for language "python" (its registry entry has no initialization_options template)`.
- Rationale: A silently-ignored precision flag is precisely the failure mode issue #2 is about — the caller believes they have widened the view and they have not. The flag must either work or fail loudly. Raising it before spawn avoids paying a server launch for a query that cannot honour its own scoping.
- Rejected: Warn on stderr and proceed — the JSON-envelope contract makes stderr warnings easy for an agent caller to miss entirely. Silent ignore — no new failure mode and no signal.
- Note: because the error must name the language, it is raised after `DetectLanguage` resolves the entry but before `acquireConnection`. `$QUARRY_BUILD_TAGS` being set globally in a shell and then running a query against a python project therefore hard-errors. This is accepted as the correct behaviour under the "must fail loudly" rationale, and must be documented in `README.md`'s configuration section.

### daemon-keyed-by-tag-set

- Decision: The normalized tag set becomes part of the resolved daemon state directory. When the tag set is empty, the resolved path is byte-identical to today's (no new segment, no re-keying, existing daemons keep working). When it is non-empty, a `tags-<hex>` segment derived from the normalized set is appended to the resolved leaf directory — applied uniformly at all three precedence tiers, including the explicit `--state-dir` / `$QUARRY_STATE_DIR` tiers, which bypass `workspaceKey` entirely today.
- Rationale: `initializationOptions` are nominally per-session, so one daemon might serve two tag sets correctly — but that is an unverified assumption about gopls's internals, and a shared cache re-indexing on every tag flip is a latency cliff. A distinct key makes the behaviour explicit instead of implementation-dependent. Applying it at every tier matters because an operator who pins `--state-dir` would otherwise silently collide two tag sets on one socket, which is exactly the case the decision exists to prevent.
- Rejected: One daemon per workspace regardless of tags (ties correctness to unverified gopls behaviour). Recording the tag set in `daemon.json` and respawning on mismatch (one daemon at a time, but thrashes when callers alternate tag sets — and an agent alternating tagged and untagged queries is the expected usage).

### tag-set-normalization

- Decision: The raw flag/env string is split on `,`, each element trimmed of surrounding whitespace, empty elements dropped, duplicates removed, and the remainder sorted lexicographically. The normalized set is what feeds both the `{{tags}}` substitution and the state-dir key. A raw value that normalizes to the empty set is treated as "no build tags" everywhere, including the `unsupported-language-hard-error` check — so `--build-tags ""` and `--build-tags " , "` are not errors on python.
- Rationale: Sorting and deduplicating make the state-dir key a function of the tag *set*, not of the argument's spelling, so `--build-tags b,a` and `--build-tags a,b` reuse one daemon instead of two. Dropping empties makes trailing commas harmless. Treating an empty normalization as absent keeps the back-compat path (`empty set → unchanged state dir`) coherent with the error check.
- Rejected: Preserving caller order (would key two daemons for one logical set). Rejecting malformed input (a trailing comma is not worth an error).

### verification-over-scope-heuristics

- Decision: Fix the interface-method false positives with a definition-verification filter, not a scope heuristic. For each candidate reference returned by `textDocument/references`, issue `textDocument/definition` at that reference's position and keep the reference only if its definition resolves to the symbol's own declaration.
- Rationale: It is exact rather than heuristic, and it carries no false-negative risk from an invented scope. It operates at the LSP level, so it generalizes past Go without per-language knowledge. It also handles the case the alternatives cannot: a genuinely cross-package interface method whose real callers legitimately live outside the declaring package.
- Rejected: Kind-gated auto-scope (detect via `documentSymbol` that the declaration's enclosing symbol is an interface, then default the scope to the declaring package directory) — cheap, but silently drops a cross-package interface method's real callers, a false negative in a safety gate. Defaulting `--within` to `--target-dir` for every symbol kind, with a `--workspace-wide` opt-out (issue #1's own "open question") — trivial, but for plain functions it converts today's false positives into false negatives, so the gate would pass with real callers outstanding. Warn-only, telling the caller to pass `--within` — zero risk, leaves the default unsafe, which is what the issue objects to.
- Consequence worth stating explicitly: for an interface method, a call site on a *concrete* type resolves to that concrete method, not to the interface's declaration, and is therefore filtered out. That is correct for this gate's question — deleting the interface method does not break a call made directly on the concrete type — but it is a semantic the tests must pin down deliberately rather than discover.

### verification-scoped-to-assert-no-callers

- Decision: Verification applies to `assert-no-callers` only. It is on by default, and `--no-verify` disables it. `refs`, `definition`, and `symbol` are unchanged and gain no verification flag.
- Rationale: `assert-no-callers` is the deterministic gate where a false positive actually blocks something. `refs` is a listing a human or agent reads and filters; over-reporting there is recoverable, and N extra round trips would be visible latency for no gain.
- Rejected: All four verbs default-on (consistent, at a latency cost on the read-only verbs). All verbs default-off with a `--verify` opt-in (no regression risk, but leaves the complained-about default exactly as it is).

### fail-closed-verification

- Decision: A reference is dropped only when verification positively disproves it — that is, the definition call succeeded and returned a location set that does not intersect the declaration set. A reference whose definition call errors, times out, or returns an empty result is **kept** as a violation. Verification never aborts the command.
- Rationale: Fail-closed is mandatory for a safety gate. A degraded or flaky server must never be able to turn the gate green by quietly failing its own verification step. Keeping unverifiable references preserves today's behaviour for exactly the references verification could not improve on.
- Rejected: Dropping errored/empty verifications (quieter, but a flaky server silently weakens the gate). Aborting the whole command on any verification error (one hiccup on one reference kills an otherwise-complete answer).

### declaration-match-is-positional

- Decision: The declaration set is the location set returned by a single `textDocument/definition` at the resolved query position, and a reference matches it when its own definition result shares at least one location by exact `(file, line, character)` equality. All comparisons happen in LSP wire coordinates (0-based line, UTF-16 character) before any conversion to the public 1-based `Reference` type.
- Rationale: File-level matching would conflate two same-named methods declared in one file. Comparing in wire coordinates avoids the byte-column-vs-UTF-16-column hazard the existing code already documents at `internal/cli/cli.go`'s `declRefs` comment: `Position.Character` is a 1-based *byte* column, which coincides with the UTF-16 column only on a pure-ASCII line.
- Rejected: File-level match (cheaper, conflates same-file same-name declarations). Comparing after conversion to the public `Reference` type (works, but reintroduces a coordinate-system conversion in the middle of a correctness-critical comparison for no benefit).

### one-connection-verification-entry-point

- Decision: Verification is implemented as a new engine-level entry point in `internal/quarryengine/query` that performs, on **one** connection: resolve position → `textDocument/definition` (the declaration set) → `textDocument/references` → per-reference `textDocument/definition` → filter. It is re-exported through `quarry/facade.go`, and `internal/cli`'s `assert-no-callers` calls it instead of its current two separate `quarry.Definition` + `quarry.References` calls.
- Rationale: `assert-no-callers` today makes two top-level engine calls, each of which runs the whole `acquireConnection` → `teardownConnection` cycle independently (`internal/quarryengine/query/refs.go`'s `lookup`). Adding N more top-level `Definition` calls would mean N more connect/initialize cycles, which is not viable. Consolidating also removes one of the two existing connection cycles, so the verified path is cheaper than today's unverified one in connection terms.
- Rejected: Calling `quarry.Definition` N times from the CLI (N connection cycles). Exposing the raw `*lsp.Client` through the facade so the CLI can drive it (breaks the engine/CLI layering the package comments establish, and `internal/quarryengine/layering_test.go` / `seam_enforcement_test.go` exist to enforce that boundary).
- Note: `quarry/facade.go`'s doc comment currently states it re-exports "exactly the 29 identifiers this package exported before the engine-repackage move: no more, no less." That sentence must be updated as part of this change, not silently invalidated.

### verification-concurrency

- Decision: Per-reference definition calls run with bounded concurrency (8 in flight), and the whole verification phase is bracketed by a single `context.WithTimeout(ctx, opts.Timeout)` deadline, matching how `lookup` already brackets each of its phases. A verification phase that hits its deadline yields kept (unverified) references per `fail-closed-verification`, not an error.
- Rationale: 31 references against a warm daemon complete in roughly the time of four sequential calls. One deadline for the phase is consistent with the existing per-phase bracketing and avoids a worst case of N × timeout. The `lsp.Client`'s `Call` is already request-ID-multiplexed over a read loop (`internal/quarryengine/lsp/lspclient.go`'s `readLoop`/`Call`), so concurrent in-flight requests are within its existing design — but this must be confirmed against `Call`'s pending-request bookkeeping during implementation rather than assumed.
- Rejected: Strictly sequential with a per-call timeout (simplest, worst case N × timeout). Concurrency plus a hard cap on how many references get verified above some N (bounds cost, but reintroduces an unverified blind spot exactly where the noise is worst).

### filter-ordering

- Decision: The filters compose in a fixed order: verification first, then `--within`, then `--except`, then the declaration-site exclusion that `filterUnexpectedCallers` already performs. `--within` and `--except` keep their current semantics unchanged; `--no-verify` restores exactly today's pipeline.
- Rationale: Verification is a precision correction on the raw reference set, so it belongs closest to the source. `--within` is already documented as applying "before `--except` is applied" (`internal/cli/cli.go`'s `assert-no-callers` Long help), and that documented relationship is preserved.
- Rejected: Verification after `--within` (equivalent result, but makes the `--within` help text's ordering claim harder to state). Making `--within` and verification mutually exclusive (they are independent and composing them is coherent — a caller may want both).

## Technical context

**The `initialize` gap.** `lsp.Client.Initialize(ctx, rootURI)` at `internal/quarryengine/lsp/lspclient.go:402` sends `processId`, `rootUri`, and `capabilities`, and nothing else. Its two callers are `internal/quarryengine/daemon/ensureserver.go:94` (inside `finalizeConnection`, serving both the supervised and native strategies) and `internal/quarryengine/query/refs.go:116` (the legacy cold-spawn path for languages with `HasNativeDaemon == false`). Both must pass the rendered options, so the signature change touches both call sites. `initializationOptions` ride the per-session `initialize` request, so the `-remote=auto` native proxy and the `-listen=unix;...` supervised daemon both receive them with no protocol-level change.

**Where the option value lands.** For gopls the key is `buildFlags`, holding an argv-style list — `{"buildFlags": ["-tags=integration,smoke"]}`. gopls accepts this flat spelling as well as the newer nested `build.buildFlags`; the flat form is what the Go built-in entry should carry, and the exact spelling should be verified against the pinned `v0.23.0` (`registry.go`'s Go entry `PinnedVersion`) during implementation.

**Options plumbing.** `query.Options` (`internal/quarryengine/query/refs.go:74`) is the engine's config struct and is re-exported verbatim as `quarry.Options` via a type alias in `quarry/facade.go`. Adding a `BuildTags []string` (normalized) field there is the natural carrier and does not disturb the facade's identifier count, which counts identifiers rather than struct fields. `buildOptions` at `internal/cli/cli.go:492` is the single construction site for all four verbs.

**Registry shape.** `registry.Entry` (`internal/quarryengine/registry/registry.go`) already carries Go-only optional fields (`PinnedVersion`, `HasNativeDaemon`) with explicit yaml tags, so `InitializationOptions map[string]any` with tag `initialization_options` follows an established pattern. Note that `LoadRegistry` (`load.go`) decodes with `yaml.Decoder.KnownFields(true)`: adding the field is what makes it *permissible* in an operator's `servers.yaml`, and until then such a key is a loud error. `validateEntry` currently enforces non-empty markers / valid match / non-empty command / non-empty install hint; the new field is optional and must not be added to those required checks. Overlay semantics are whole-entry replacement, so an operator's `go:` block that omits `initialization_options` loses the built-in template — the same already-documented sharp edge that applies to `pinned_version`, and `docs/servers.yaml.example`'s header comment already warns about it.

**State-dir keying.** `workspaceKey` (`internal/cli/paths.go:55`) is `filepath.Base(targetDir) + "-" + hex(sha256(clean(abs))[:6])`. `resolveStateDir` (same file) picks `--state-dir`, then `$QUARRY_STATE_DIR`, then `filepath.Join(userCacheDir(), "quarry", workspaceKey(targetDir))` — the first two tiers bypass `workspaceKey` entirely, which is why the `tags-<hex>` segment is specified as applying to the *resolved* leaf rather than being folded into `workspaceKey`. `resolveContext` at `internal/cli/cli.go:462` is the single call site and needs the normalized tag set threaded in. `DaemonStateFile` / `DaemonLock` join their own `<lang>` segment below the leaf, so appending a segment to the leaf keeps the daemon-side layout unchanged.

**assert-no-callers today.** `assertNoCallersCommand` at `internal/cli/cli.go:508` already calls `quarry.Definition` before `quarry.References`, specifically so the declaration site can be excluded in matching coordinate systems — its inline comment explains why building a `Reference` from a caller-supplied `Position` would be wrong. That reasoning carries straight over to the new entry point: the declaration set must come from a real `textDocument/definition`, not from the query position. `filterWithin` / `isWithinDir` / `filterUnexpectedCallers` (`cli.go:670`–`730`) stay as they are.

**The `lookup` pipeline.** `internal/quarryengine/query/refs.go`'s `lookup` takes a single `lspCall` closure and therefore expresses exactly one LSP call per connection. The new entry point needs several calls on one connection, so `lookup` needs generalizing — either a sibling that hands the resolved `(client, fileURI, pos)` to a caller-supplied closure, or a variant `lspCall` signature. Whichever shape is chosen, `lookup`'s existing invariants must be preserved: the `timedOut` flag is captured by reference by the deferred `teardownConnection` precisely because a later phase can still set it, and `teardownConnection`'s `ConnKindSupervised` branch must keep returning without any shutdown handshake or kill.

**Layering enforcement.** `internal/quarryengine/layering_test.go` and `internal/quarryengine/seam_enforcement_test.go` encode the five-package DAG (root leaf, `lsp`, `registry`, `daemon`, `query`). Anything added must respect it — in particular, template rendering that needs the registry shape belongs in `registry` or `query`, never in `lsp`, and `internal/cli` must not reach past the `quarry/` facade.

**Test-tier convention.** Live tests are `//go:build lsp`-tagged in their own `*_integration_test.go` / `*_lsp_test.go` files, skip via `exec.LookPath("gopls")` when the binary is absent, and are run with `go test -tags lsp ./...`. `internal/quarryengine/query/refs_integration_test.go` is the closest model for a new query-level live test, and `internal/quarryengine/daemon/daemontest` provides the shared helpers. Note the pleasing recursion: the build-tag live test needs a fixture module containing a `//go:build`-tagged file, inside a repo whose own live tests are `//go:build lsp`-tagged.

**Prior evidence.** `docs/scout-multilang.md:115`–`136` carries the quantified build-tag gap and the analysis that ruled out a workspace-load race before attributing it to build tags. `docs/scout-agent-usage-findings.md` and `docs/scout-vs-grep.md` carry the interface-conflation measurement. These are the reference numbers any new benchmark should be compared against.

## Testing

No `CONSTRAINTS.md` exists at the hub root.

**Hermetic tier (`go test ./...`, no external binary).** These are the TDD candidates — every one is a pure function over in-memory values:

- Tag normalization (`internal/cli`): split, trim, drop empties, dedupe, sort. Scenarios: `"b,a"` and `"a,b"` normalize identically; `""`, `","`, `" , "` all normalize to the empty set; `"a,,b, a "` yields `[a b]`.
- Template rendering (`registry` or `query`): substitution into a nested map. Scenarios: nested map with `{{tags}}` in a list element renders correctly; the source entry map is not mutated (deep copy); a template with no placeholder passes through verbatim; map *keys* are never substituted; an empty tag set against a template-bearing entry.
- Unsupported-language error (`internal/cli`): non-empty normalized tags plus an entry with empty `initialization_options` produces the error, names the language, and is raised before any spawn. Complement: an empty normalization against such an entry is *not* an error.
- State-dir keying (`internal/cli`): empty tag set yields byte-identical paths to today at all three precedence tiers (this is the back-compat assertion and should be written first); a non-empty set yields distinct paths at all three tiers, including explicit `--state-dir`; `b,a` and `a,b` yield the same path.
- Registry decode (`registry`): a `servers.yaml` carrying `initialization_options` decodes without tripping `KnownFields(true)`; an entry omitting it validates fine (the field is optional); built-in Go carries the template and the other four do not.
- The verification filter as a pure function over `(declarationLocations, references, perReferenceDefinitionResults)`. Scenarios, mirroring issue #1's measurement: a reference whose definition matches the declaration is kept; one whose definition points elsewhere is dropped; one whose definition call errored is kept (fail-closed); one whose definition returned empty is kept (fail-closed); exact-position matching distinguishes two same-named declarations in one file; a concrete-type call site is dropped, pinning the semantic called out under `verification-over-scope-heuristics`.
- Filter ordering (`internal/cli`): verification → `--within` → `--except` → declaration exclusion, and `--no-verify` reproducing today's pipeline exactly.

**Live tier (`go test -tags lsp ./...`, real gopls on `$PATH`).** Two end-to-end tests, each on its own fixture module, each guarded by `exec.LookPath("gopls")`:

- Build-tag visibility: a fixture module with a symbol referenced from both an untagged file and a `//go:build sometag`-constrained file. Assert the tagged reference is absent without `--build-tags` and present with `--build-tags sometag`. This is the one thing a fake server cannot reproduce, since it is gopls's own view-loading behaviour.
- Interface conflation: a fixture module with two structurally-identical local interfaces in different packages, each with its own callers, reproducing the `clock` shape from issue #1. Assert that `assert-no-callers` reports only the callers of the interface under test, and that `--no-verify` still reports the inflated set — pinning both the fix and the escape hatch in one fixture.

An end-to-end assertion that the state-dir keying actually yields two distinct daemons for two tag sets also belongs in the live tier if it can be written without racing the daemon lifecycle; if it cannot be made deterministic, the hermetic path-derivation tests above are the contract and this should be left out rather than made flaky.

## Q&A log

- **Q:** How do build tags reach the CLI — per-verb flag, root persistent flag, per-verb only, or `servers.yaml` only? **A:** Per-verb `--build-tags` on all four verbs plus a `$QUARRY_BUILD_TAGS` fallback; it matches the existing `--target-dir`/`--lang` pattern and the env var avoids repeating the flag across an agent session.
- **Q:** Who owns the per-language spelling of the option — registry entry, CLI hard-code, or a generic passthrough? **A:** The registry entry, which is what issue #2 itself proposes; it keeps gopls-specific spelling out of `internal/cli` and is the right home for per-language variation.
- **Q:** Should the daemon be keyed by the build-tag set? **A:** Yes. Relying on gopls keeping per-session views separate is an unverified assumption; an explicit key makes the behaviour independent of that implementation detail and avoids the re-indexing cliff of a shared cache.
- **Q:** Mechanism for the interface-method false positives — definition verification, kind-gated auto-scope, defaulting `--within`, or warn-only? **A:** Definition verification — the only option with no false-negative risk in a safety gate; the others trade one error class for a worse one.
- **Q:** Where does the filter apply, and what is the escape hatch? **A:** `assert-no-callers` only, default-on, `--no-verify` to disable. The cost of N extra round trips belongs where a false positive actually blocks something; on `refs` it would be visible latency for no gain.
- **Q:** Concrete shape of the registry template — nested YAML map, two flat fields, or a JSON-fragment string? **A:** YAML-native nested map with `{{tags}}`; avoids stringly-typed JSON-in-YAML and lets future languages contribute their own shape without engine changes.
- **Q:** What happens when `--build-tags` is passed for a language with no template? **A:** Hard error before spawn. Silently ignoring a precision flag is the exact failure mode issue #2 complains about — it must either work or fail loudly.
- **Q:** What counts as a match, and what happens when verification itself fails? **A:** Exact positional match against the declaration set; an unverifiable reference stays a violation (fail-closed). A flaky server must never be able to turn the gate green by quietly weakening verification.
- **Q:** Round-trip budget for verification — bounded concurrency, sequential, or a cap on refs verified? **A:** Bounded concurrency sharing one phase deadline. Consistent with how `lookup` already brackets phases, avoids worst-case N × timeout, and introduces no threshold-dependent blind spot the way a cap would.
- **Q:** Test coverage split? **A:** Hermetic tests for the pure parts plus `-tags lsp` live tests for both end-to-end behaviours. Build-tag visibility and interface conflation are precisely the symptoms that cannot be reproduced without a real gopls; a fake server would test wiring, not the actual problem.
