MILL_REVIEW_BEGIN
# Review: Add an MCP wrapper for quarry — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: claude-opus-5 (self-assessment; harness-reported)
reviewed_file: plan/
date: 2026-08-29
```

## Findings

### [BLOCKING:design] `impact` entry schema advertises `except`
**Location:** batch 4, cards 17–18 (and card 25's matrix assertion)
**Issue:** `impactInput.Targets` is `[]nativeEntry`, and `nativeEntry` declares `Except`, so `impact`'s derived entry schema advertises `except` while card 18's handler reports it via `unknownEntryKeys(entry.raw, "file","line","character","symbol","within")` as a per-entry error — contradicting `per-entry-vs-call-wide`'s authoritative matrix ("a tool exposes exactly these and nothing else", `impact` per-entry = `within` only).
**Fix:** state a disposition — either give `impact` its own entry type without `Except`, or prune `except` from the `targets` item schema in `registerImpactTool`, and assert its absence in card 25's per-tool matrix test.

### [BLOCKING:design] `classifyLSPError`'s nil-error branch unspecified
**Location:** batch 2, card 9; consumed by batch 3, card 14
**Issue:** card 9 enumerates only three branches (`errors.As` ambiguous, `errors.Is` not-found, "anything else yields `statusError` carrying `err.Error()`") while its sibling `classifySymbolError` explicitly specifies `nil → statusFound`; card 14 then says the handler "maps the outcome through `classifyLSPError`", which on `err == nil` falls into the else branch and dereferences a nil error.
**Fix:** state the nil branch of `classifyLSPError` explicitly, or say in card 14 that it is called only on a non-nil facade error (as cards 18 and 19 already do).

### [BLOCKING:scope] Two cards name identifiers from files absent from `Context:`
**Location:** batch 2, cards 8 and 11
**Issue:** card 8's Requirements name `quarry.Query.Pos.File`, `quarry.InFileQuery.File` and `quarry.Position` but omit `quarry/facade.go` from `Context:` (every sibling card in the batch lists it); card 11's Requirements name `tocFileCommand` and `tocDirCommand` but omit `internal/cli/toc.go`.
**Fix:** add `quarry/facade.go` to card 8's `Context:` and `internal/cli/toc.go` to card 11's.

### [NIT:consistency] `outputSchemaFor` contradicts the schema-derivation decision
**Location:** batch 2, card 10 vs `## Shared Decisions` → `schema-derivation-and-patching`
**Issue:** the Decision says every `jsonschema.For[T]` call passes the package's own `ForOptions` value, "never `nil`", but card 10 describes `outputSchemaFor[T any]()` as calling `jsonschema.For[T]` with no options named, leaving the implementer to choose.
**Fix:** say explicitly whether `outputSchemaFor` passes the same options value (harmless, since no output type embeds `docSentences`).

### [NIT:scope] Per-entry timeout has no test anywhere in the plan
**Location:** batches 2–6 (card 11's `callcontext_test.go`, cards 16/20/25)
**Issue:** `batching-execution-model` part 3 and the discussion's tier-1 list call for asserting that `--timeout` reaches each entry's `quarry.Options.Timeout` in full rather than being divided across the batch; card 11 documents the behaviour but no card asserts it, and the facade stubs make it a two-line check.
**Fix:** name the assertion in card 11's or card 16's test requirements.

## Verdict

REQUEST_CHANGES
One schema/matrix contradiction, one unspecified error branch, two Context omissions.
MILL_REVIEW_END
