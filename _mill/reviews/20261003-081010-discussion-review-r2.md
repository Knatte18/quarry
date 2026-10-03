MILL_REVIEW_BEGIN
# Review: Resolve self-target path: per-call dirPackage memo (GH #34)

```yaml
duration_s: 98.3
verdict: REQUEST_CHANGES
reviewer_model: opus
reviewed_file: _mill/discussion.md
date: 2026-10-03
```

## Findings

### [BLOCKING:design] Clause-less file: walk and member path diverge today
**Section:** Decisions / Unit stamping after the vote; Technical context (per-file error semantics)
**Issue:** A Go file that parses but yields an empty clause (the `packag mypkg` case in `toc_test.go`) gets symbols in the walk, because `fileEntry` receives `clauses[base] == ""` and `unitFor` maps it to `dirRel`, but `symbolsOfDir` skips it (`clause, ok := clauses[base]; if !ok { continue }`).
Stamping every record "through `unitFor`" for both consumers, as decided, would add those symbols to Resolve/Expand member results and silently change answers.
The error-semantics bullet lists only read, UTF-8 and `WithTree` failures, and none of the planned tests has a fixture with this case.
**Fix:** State that the member-path consumer still excludes records with no recorded clause and the walk consumer still includes them, and add an equivalence fixture with a clause-less file in a subdirectory.

### [NIT:design] Round trip loses its independent second reading
**Section:** Decisions / Threading through TOC; Testing / Existing suites unchanged
**Issue:** `assertSymbolRoundTrip` compares walk symbols against `symbolsOfUnit`. Once both read the same per-file records, the comparison checks only the stamping/filter layer, not two extractions, yet the discussion still counts it as a guard for the refactor.
**Fix:** State what the round trip still proves after the refactor, and say whether the new equivalence tests replace the extraction-level coverage it used to give.

### [NIT:design] Parse-counter unit undefined for unparsed files
**Section:** Decisions / Test seam; Testing / TDD candidate
**Issue:** Non-language files (read for `HeaderForFile`), unreadable files and invalid-UTF-8 files are read but never tree-sitter parsed, so "incremented before the parse" and "total equals the number of distinct files touched" can disagree in any directory that holds them.
**Fix:** Say whether the counter counts record builds (per file read) or tree-sitter parses, and define "files touched" to match.

## Verdict

REQUEST_CHANGES
The stamping decision, applied literally, changes member answers for files whose package clause is empty.
MILL_REVIEW_END
