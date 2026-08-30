MILL_REVIEW_BEGIN
# Review: Rethink quarry-mcp's per-call targetDir ergonomics — holistic

```yaml
verdict: APPROVE
reviewer_model: sonnethigh
reviewed_file: plan/ + source
date: 2026-08-30
```

## Findings

No findings. Verified against both batch files and every listed source file:

- Card 1 (transport_test.go): `TestToolsList_PerToolParameterMatrix` extended with the call-wide `targetDir` absence loop across all seven tools; `TestCallTool_TargetDirIsRejectedAsWholeCallError` added exactly as specified, asserting `IsError` true and no decodable `results` array, without pinning the SDK's error string.
- Card 2: all six input structs (`lspInput`, `symbolInput`, `impactInput`, `assertInput`, `tocFileInput`, `tocDirInput`) have `TargetDir` removed; `effectiveTargetDir` deleted from `callcontext.go`; `resolveCall(cfg, buildTags)` now uses `cfg.TargetDir` directly and preserves the exact `ResolveBuildTags → ResolveConfigPath → LoadRegistry → ResolveStateDir` order; all five `resolveCall` call sites updated; both toc handlers read `cfg.TargetDir` and skip the unreachable error branch; `callcontext_test.go` deletes the two effectiveTargetDir tests and adds `TestResolveCall_TargetDirIsAlwaysConfigTargetDir` pairing `TargetDir`/`StateDir` assertions as required.
- Card 3/4: every `jsonschema:` tag and Go doc comment naming `targetDir` as a call parameter now reads "the server's target directory"; the three "call-wide resolution overrides" doc comments correctly say "two" (lang, buildTags); `exceptSet`'s doc comment names `Config.TargetDir` rather than paraphrasing the deleted helper; `tocFileInput`/`tocDirInput` no longer carry the stale "plus the per-call overrides" phrasing; live parameter names (`nativeEntry.query`'s `targetDir`, `resolveEntryFile`'s, `exceptSet`'s, the toc resolve-entry functions', and the toc handlers' local `targetDir` variables) are correctly left untouched.
- Card 5 (tools_toc_test.go): `TestTOCFileHandler_AbsoluteTargetResolvesOutsideLaunchRoot` added, resolving a fixture outside `cfg.TargetDir` via an absolute path, with the two stale-comment corrections applied verbatim.
- Card 6 (docs/mcp-setup.md): new "The server's scoping contract" section correctly hedges both the cwd-inheritance claim and the state-keying claim (cross-checked against `internal/cli/paths.go`'s actual tier precedence), documents the toc partial escape hatch asymmetry, and never spells the literal token `targetDir` anywhere in the file — confirmed by a direct grep.
- Card 7 (ladder_config.py / test_ladder_config.py): the two-line prompt replacement is byte-for-byte the text `_mill/discussion.md` prescribes; the `DAEMON_BACKED_TOOLS` comment is reworded to `cfg.TargetDir`/`tocPreflight` phrasing; the test's narrowed literal matches; `gates.py` is untouched and still checks both `targetDir` and `buildTags` keys as required.
- Card 8's four completeness checks all pass on inspection: a repo-wide `effectiveTargetDir` grep returns zero hits outside `_mill/`; the production-file and test-file `targetdir` greps return only the enumerated intentional survivors (Go identifiers, field reads, local variables, test names, doc-comment prose); no stray `targetDir` key appears in any JSON/input-struct literal besides the one deliberately-rejected fixture in `transport_test.go`; and a re-read of all six input structs' and `exceptSet`'s doc comments confirms each accurately describes its subject post-removal.

Cross-batch contracts hold: batch 2's documentation and bench prompt describe exactly the launch-scoped behaviour batch 1 establishes, and no shared decision is contradicted by either batch.

## Verdict

APPROVE
Every card across both batches is fully and consistently realised; no deviation, omission, or stale reference found.
MILL_REVIEW_END
