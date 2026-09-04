MILL_REVIEW_BEGIN
# Review: MCP, thin (T6) — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude (Anthropic), Opus-class; exact release self-reported as Opus 5
reviewed_file: plan/
date: 2026-09-04
```

## Findings

### [BLOCKING:design] Negative-depth handler branch unreachable via protocol
**Location:** batch 2 card 8 step 1 / batch 3 card 15
**Issue:** Card 8 justifies the handler's `depth < -1` check by calling the schema's `minimum` "advisory", but go-sdk v1.7.0 validates server-side before the handler: `toolForErr`'s wrapper calls `applySchema(input, inputResolved, false)` → `resolved.Validate` (`mcp/server.go:360`, `mcp/tool.go:129`) and on failure returns `CallToolResult{IsError:true}` whose text is `validating "arguments": …`. With `Minimum: -1` in the schema, `depth: -2` / `-7` never reach the handler, so card 15's required assertion — text equal to `quarry.RenderErrorJSON("--depth must be -1 (whole tree) or a non-negative integer, got <n>")` — cannot pass, and batch 3's own rule forbids calling the handler directly.
**Fix:** State one disposition: either drop `Minimum` from `tocInputSchema()` so the handler owns the wording, or keep it and have card 15 assert the SDK's validation result for `< -1` plus a separate direct-call test for the handler branch.

### [BLOCKING:design] Broken-symlink case is a success, not a failure
**Location:** batch 3 card 16
**Issue:** Card 16 requires the broken-symlink target to return `isError` with "the CLI's own wording". It does not: `os.Lstat` succeeds on a broken link (card 8 step 3 passes), `engine.resolveTarget` also Lstats (`internal/engine/repo.go:63-71`), and `fileTargetAnswer` emits the link as a name-only `FileEntry` (`internal/engine/toc.go:131-134`) with a nil error — so the call returns a normal answer. The card also names expected wording for only two of its three cases, leaving the third's expectation undefined.
**Fix:** Replace the case with a genuine third failure (e.g. an unreadable directory, or drop to two cases) or restate it as a success-shape assertion pinning the name-only entry.

### [BLOCKING:scope] Card 16 Context omits files its Requirements name
**Location:** batch 3 card 16
**Issue:** The card instructs the implementer to use "the same per-package `writeScratchTree` approach card 7 uses" and to "open that tree as its own repository", but `internal/mcpserver/root_test.go` (where card 7 declares `writeScratchTree`) and `quarry/repo.go` (`quarry.Open`) are absent from `Context:`; only `toc.go`, `fixture_test.go`, `cli/cli.go` and `render.go` are listed.
**Fix:** Add `internal/mcpserver/root_test.go` and `quarry/repo.go` to card 16's `Context:`.

### [NIT:consistency] `jsonschema-go v0.4.3` pin attributed to a decision that omits it
**Location:** batch 2 card 5
**Issue:** The card asserts "Both versions are pinned by discussion D1"; D1 pins only `go-sdk v1.7.0` and never mentions `jsonschema-go`. The version is in fact correct (it is go-sdk v1.7.0's own `require`), but the plan cites a source that does not carry it.
**Fix:** Attribute the `jsonschema-go` pin to go-sdk v1.7.0's `go.mod`, not to D1.

### [NIT:decision] Stale README sentence left with no disposition
**Location:** batch 4 card 17
**Issue:** `README.md` still says "There is no command to build yet", which the new section's `go build -o quarry-mcp ./cmd/quarry-mcp` line directly contradicts; card 17 says only "do not restructure the sections already there" and never states whether that sentence is corrected or left.
**Fix:** Give the sentence an explicit disposition in card 17's Requirements.

### [NIT:consistency] No batch verify runs the repo's stated done gate
**Location:** `00-overview.md` `verify:` / all batch `verify:` blocks
**Issue:** Discussion "Gate" names `go test ./... && golangci-lint run`; the overview's cross-batch verify is `go vet ./...` and every batch verify is package-scoped, so the module-wide suite and the linter run at no point in the plan.
**Fix:** Note in the overview why the done gate is deferred to merge, or raise the overview `verify:` to the stated gate.

## Verdict

REQUEST_CHANGES
Two test requirements contradict actual SDK and engine behaviour; one Context list is incomplete.
MILL_REVIEW_END
