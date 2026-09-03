MILL_REVIEW_BEGIN
# Review: Engine core (T3)

```yaml
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (claude-opus-5)
reviewed_file: /home/knatte/Code/quarry/wts/engine-core/_mill/discussion.md
date: 2026-09-03
```

## Findings

### [BLOCKING:design] D4's head-span premise is false for interfaces
**Section:** D4 (vs D5) **Issue:** D4 justifies "head == the type declaration's own span" with "a Go type never contains its methods", but D5 now lists `method_elem`s inside an `interface_type` as their own glyphs, so an interface's head span literally contains its member spans — the Python/C# case D4 says Go does not have. **Fix:** decide and state the interface head explicitly (whole declaration, or declaration minus `method_elem` spans, matching §5's "class span minus its member spans") and correct D4's rationale.

### [BLOCKING:design] Which `interface_type` nodes the walk visits is unstated
**Section:** D5 **Issue:** D5 says "`method_elem` inside an `interface_type`, owner = the interface's type name" but never bounds which interface types are walked; an anonymous interface in a struct field, a parameter, a `var`, or a generic constraint has no type name, and glyph.md §3 excludes struct fields — a walk that descends to any `interface_type` would emit owner-less or wrongly-owned glyphs. **Fix:** state that only an interface that is a file-scope `type_spec`'s own type is walked, and that anonymous interfaces are never listed. D5's own note applies: the round trip compares two readings of the same walk and cannot catch this.

### [BLOCKING:design] Only the root is decided among unspellable units
**Section:** D7 **Issue:** D7 answers `Unit == ""` (root) but nothing else the Go alphabet rejects; `glyph/golang.go:checkGoUnit` also returns `ReasonUnitBadRune` for any path segment containing a space, `\`, or a control rune, so a `.go` file under e.g. `test data/pkg/` yields an `id` that `glyph.Parse` rejects — failing Testing 15's parse assertion and leaving `SpansOf` nothing to invert. **Fix:** extend D7's rule to "a directory whose repository-relative path the alphabet cannot spell is listed but carries no `symbols`", or state the alternative disposition.

### [BLOCKING:design] Round-trip cost over Loomyard is unbounded and unaddressed
**Section:** Testing 14/15, D16, D22 **Issue:** `SpansOf` takes one glyph and re-parses its whole unit directory, and D22 forbids caching, so the headline criterion is O(glyphs x files-in-unit) parses; §5 measures 65 ms for one glyph in a 35-file package and §4 shows `internal/reedengine` holding 67 files, putting the Loomyard run in the minutes and within reach of `go test`'s 10-minute default timeout — the widened Kind set of D5 multiplies it further. **Fix:** state how the round-trip test batches (group listed glyphs by unit, one `SpansOf`-equivalent pass per unit) or state the accepted runtime budget and timeout.

### [NIT:design] gitignore containment rule not in D9's enumerated subset
**Section:** D9 **Issue:** D9 enumerates the supported pattern syntax precisely but never states that a matched directory excludes everything beneath it (and that a re-included directory such as `!/quarry/` reopens the subtree) — the rule its own `plugins/prowler/bin/` and `/quarry` fixtures turn on. **Fix:** name the directory-pruning semantics alongside the pattern list.

### [NIT:consistency] D9's supersession points at text D22 no longer contains
**Section:** D9 / D22 **Issue:** D9 says it supersedes "D22's 'once per call, walking root-to-target' phrasing", but D22 as written already carries the corrected wording, so a plan writer looking for the superseded statement finds nothing. **Fix:** drop the supersession note or mark it as history.

## Verdict

REQUEST_CHANGES
Four unresolved decisions: interface head span, interface walk scope, unspellable units, round-trip cost.
MILL_REVIEW_END
