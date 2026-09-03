MILL_REVIEW_BEGIN
# Review: Engine core (T3)

```yaml
duration_s: 302.0
verdict: APPROVE
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (claude-opus-5)
reviewed_file: _mill/discussion.md
date: 2026-09-03
```

## Findings

### [NIT:consistency] Ignore set never covers descended directories
**Demoted-from:** BLOCKING
**Section:** D9 / D22 (and D16) **Issue:** D9 collects patterns "from the repository root down to the directory being listed" while D22 fixes that as "once per call, walking root-to-target, not once per directory visited" — so under `--depth all` a `.gitignore` inside any descendant is never read, while `SpansOf`'s set *does* include the unit directory's own `.gitignore`, making the two filters asymmetric and turning a descendant-ignored `.go` file into a round-trip *miss* under Testing 14/15 — the exact failure D16's shared-filter paragraph exists to prevent. **Fix:** state one rule for both entry points — whether the pattern set is extended as the walk descends (per-directory, still uncached) or descendant `.gitignore` files are deliberately out of scope — and say which of D9's and D22's sentences is superseded.

### [NIT:consistency] `Symbol.File` has no producer in T3
**Demoted-from:** BLOCKING
**Section:** D3 vs D16 **Issue:** D3 says `File` "is filled by `SpansOf` and by T4's `resolve`/`expand`", but D16 gives `SpansOf` the signature `([]Span, error)` over a separate `Span{File,Start,SigEnd,End}` type that is not a `Symbol` — so no T3 code path ever sets `Symbol.File`, leaving a field the task adds with no writer, which is the same dead-carrier D14 and D21 reject elsewhere. **Fix:** either state that `File` is written only in T4 and why it is added now anyway, or make `SpansOf` return the symbol entry rather than a bare span.

### [NIT:consistency] D5's `const`/`var` `Start` contradicts §4's doc-inclusive start
**Demoted-from:** BLOCKING
**Section:** D5 (span/signature table) **Issue:** every row states Start/End as "the declaration's span" (or "the spec's span") with the doc listed as a separate column item, but the established rule — `internal/quarryengine/toc/golang.go:goDeclSymbol`, and §4's `placement` at `"start": 16` with its `type` clause on line 20 — sets `Start` to the doc block's first line when one is attached; the round trip cannot catch a wrong reading, since it compares `toc`'s output against a lookup built the same way. **Fix:** say explicitly in the table that Start is the attached comment block's first line when `CommentBlockAbove` returns one, and the node's own first line otherwise.

### [NIT:design] "Most common package clause" has no tie-break
**Section:** D7 step 2 **Issue:** a directory whose `.go` files split evenly between two clauses (a `//go:build ignore` `package main` file beside a one-file package, or a malformed tree) has no defined directory package, so `dir.package`, every file's `package` deviation key and every glyph unit below it become order-dependent — against D18's whole point. **Fix:** name the tie-break (e.g. lexicographically smallest clause) in D7.

### [NIT:consistency] `SpansOf` does not reject an unspellable unit
**Section:** D16 vs D7 **Issue:** D7 asserts "`SpansOf` never produces a root span", but D16 validates only `g.Lang` and maps literal-first, so a hand-built `Glyph{Unit: ""}` — the very input D21 says the engine must have a defined answer for — resolves to the repository root and returns spans `toc` never listed. **Fix:** state that `SpansOf` returns an empty slice (or a sentinel) for a unit `glyph.Parse` would reject.

### [NIT:scope] D22's freshness guarantee is untested
**Section:** D22 / Testing **Issue:** no listed test exercises "the ignore set is re-read per call"; the staleness it exists to prevent only manifests inside T6's long-lived process, so nothing in T3 would notice a pattern set accidentally cached on `Repo`. **Fix:** add a test that mutates a fixture `.gitignore` between two calls on one `Repo` and asserts the second answer changes.

## Verdict

APPROVE
Three contradictions — ignore-set scope, `Symbol.File`'s writer, and `const`/`var` start lines.
_Note: 3 finding(s) demoted from BLOCKING to NIT by the stage's blocking-class ceiling; current blocking_count is 0._
MILL_REVIEW_END
