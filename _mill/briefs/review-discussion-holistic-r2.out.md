MILL_REVIEW_BEGIN
# Review: Facade + CLI, resolve + expand (T5b)

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: claude-opus-5 (Anthropic Claude Opus 5, high reasoning effort)
reviewed_file: /home/knatte/Code/quarry/wts/facade-cli-resolve-expand/_mill/discussion.md
date: 2026-09-04
```

## Findings

### [BLOCKING:consistency] Payload `error` leaks `engine:` on exit 1
**Section:** D2 + D8 (vs D4) **Issue:** D8 routes an escaping path to `resolvePathTarget`, which sets `Error: err.Error()` (`internal/engine/resolve.go:338`) from `resolveTarget`'s wrap (`internal/engine/repo.go:81`), so `quarry resolve ../x` prints `"error": "engine: resolve target \"../x\": engine: target outside repository"` on stdout at exit 1 — and D7 renders the same doubled internal prefix as prose — while D4 cites T5a's rule (`internal/cli/cli.go` `Run` doc) that no exit-1 output may leak the `engine: ` prefix. **Fix:** state the disposition explicitly — either that the rule binds only `fail()`'s own sentence and the payload `error` is engine data emitted verbatim, or that D8's route is revised — and say which string the `after/`/e2e tests pin.

### [BLOCKING:consistency] D3's table has no row for `expand`'s grammar rejection
**Section:** D3 (vs D10's final bullet) **Issue:** D10 assigns a `#`-bearing target the grammar rejects (`#x`, `member_keyword`, `unit_bad_rune`) to D4's exit-1 envelope path, but D3 — presented as *the* exit-code table and the source for the table-testable `codeForExpandError` — has no such row, and its catch-all `both | … | 3` row is what a mapping written from the table would return. **Fix:** add the explicit `expand | glyph.Parse rejection (target contains "#") | 1 | the error envelope` row to D3, and state where `expand`'s `HeadStart == 0` invariant error lands.

### [BLOCKING:design] No stated route for the CLI to obtain `<reason>`
**Section:** D10 (final bullet) + D5 **Issue:** D10 fixes the message as `expand <target>: <reason>`, but `expand` wraps the parse error as `engine: expand %s: %w` (`internal/engine/expand.go:146`) so `err.Error()` is barred by D4's own rule; reaching `parseErr.Reason` needs `*glyph.ParseError`, yet D10 says "`internal/cli` therefore needs no import of `glyph/`", D5's facade surface adds no `ParseError`/`Reason` alias, and `quarry/quarry.go` aliases none today — while Technical context lists `glyph/errors.go` as "needed by D10". **Fix:** decide one of the three — the CLI imports `glyph/`, the facade gains `ParseError`/`Reason` aliases (extending D5), or the message drops `<reason>` — and reconcile the Technical-context line with D10.

### [NIT:scope] `afterGoldenCase` also needs a verb field
**Section:** D14 / Technical context **Issue:** the change list says only "eight rows + an expected-exit-code field", but `after_test.go:78` hardcodes `"toc"` into argv and line 89 hardcodes `"$ quarry toc "` into the recorded invocation line. **Fix:** name the verb field alongside the exit-code field.

### [NIT:scope] Test files edited but absent from the change list
**Section:** Technical context, "Files this task changes" **Issue:** the Testing section requires editing `internal/cli/cli_test.go`, `flags_test.go`, `target_test.go` and `quarry/render_test.go`, `text_test.go`, `repo_test.go`, but only `after_test.go` appears in the file list. **Fix:** list them, so a plan writer's card inventory matches the Testing section.

## Verdict

REQUEST_CHANGES
Three underspecified points around `expand` rejections and engine-prefix leakage into payload output.
MILL_REVIEW_END
