MILL_REVIEW_BEGIN
# Review: Thin quarry/ facade over internal/quarryengine — holistic

```yaml
verdict: REQUEST_CHANGES
reviewer_model: sonnethigh
reviewed_file: plan/ + source
date: 2026-08-27
```

## Findings

### [BLOCKING:consistency] Close's doc comment is attached to the wrong method
**Location:** `internal/quarryengine/lsp/lspclient.go:584-597`
**Issue:** Card 6 required inserting a new `Closed()` accessor "with a doc comment," but the insertion landed the one-line `Closed` doc comment at the tail of `Close`'s existing multi-line doc block (lines 584-592), so the whole block now precedes and describes `func (c *Client) Closed() bool` instead of `Close`. The exported `Close()` at line 597 is left with no doc comment at all, and `Closed`'s doc reads as a one-line afterthought glued onto eight lines describing a different method. Every other exported identifier in this codebase (including the sibling `Kill`, `Call`, `Initialize`, etc.) carries its own accurate doc comment, so this is a real drift from the established convention, not a style nit.
**Fix:** Split the comment block: keep lines 584-591 ("Close runs the graceful...transport.") directly above `func (c *Client) Close()`, and give `Closed()` its own one-line comment ("Closed reports whether Close or Kill has already torn this client down.") directly above `func (c *Client) Closed() bool`.

### [NIT:consistency] Stale `lspClient` identifier in a doc comment
**Location:** `internal/quarryengine/lsp/lspclient_test.go:1`
**Issue:** The file header still reads "lspclient_test.go exercises lspClient's framing/protocol logic," but card 4 renamed the type `lspClient` -> `Client`; the doc comment was not updated to match.
**Fix:** Reword to "exercises Client's framing/protocol logic."

### [NIT:consistency] Stale lowercase `connKindSupervised` in test comments
**Location:** `internal/quarryengine/query/refs_integration_test.go:90,192`
**Issue:** Two prose comments still refer to "teardownConnection's connKindSupervised branch," but card 8 retargeted the identifier to `daemon.ConnKindSupervised`; the comment prose was not updated alongside the code.
**Fix:** Reword both comments to reference `daemon.ConnKindSupervised`.

## Verdict

REQUEST_CHANGES
One misattributed/missing doc comment on an exported method in lspclient.go; two stale-identifier NITs in comments.
MILL_REVIEW_END
