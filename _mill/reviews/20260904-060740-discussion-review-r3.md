MILL_REVIEW_BEGIN
# Review: Facade + CLI, toc (T5a)

```yaml
duration_s: 195.0
verdict: REQUEST_CHANGES
reviewer_model: opushigh
reviewer_self_id: claude-opus-5 (runtime metadata; best-effort, not independently verifiable)
reviewed_file: _mill/discussion.md
date: 2026-09-04
```

## Findings

### [NIT:consistency] Q&A contradicts the after/ golden byte layout
**Demoted-from:** BLOCKING
**Section:** Q&A log ("What format do the `after/*.txt` files use?") vs `after-outputs-are-the-goldens`
**Issue:** The Q&A answer specifies "output, blank line, `(exit code: N)`"; the decision specifies "**No exit-code trailer**" and argues it at length — and since these files are byte-compared goldens, the trailer decides their bytes.
**Fix:** Delete or rewrite the superseded Q&A answer so one layout is stated.

### [BLOCKING:design] `targetIsFile` has no stated source
**Section:** `text-view-grammar` (file form) / `facade-entry-point`
**Issue:** `RenderText`'s form is "selected by `targetIsFile`, never inferred", but `TOC` returns `(DirAnswer, error)` only (verified `internal/engine/toc.go:41`) and the CLI holds only a path string — nothing says how `internal/cli.Run` computes the fact, whether via its own `os.Lstat` (required by gotcha 6, symlink-as-file), or how that stat orders against the engine's own not-found/outside-repo checks.
**Fix:** State where `targetIsFile` comes from (CLI-side `Lstat` before the call, with its ordering relative to exit 1/2 mapping), or give the facade a signal that carries it.

### [BLOCKING:design] The failure envelope's `error` value is unspecified
**Section:** `failure-envelope-and-exit-codes` / Testing ("Failure envelope")
**Issue:** The payload is fixed as `ok` + `error` only, but no rule says what `error` holds — the raw wrapped engine string (`engine: resolve target "x": engine: target not found`) or a CLI-authored sentence — nor whether it equals the stderr human message; the Testing block nevertheless asserts stdout is "exactly" the envelope.
**Fix:** State the derivation rule for `error` (and for the stderr message) so the assertion is writable.

### [NIT:consistency] `--help` breaks the exit-0 stdout rule
**Section:** `flag-semantics` vs `failure-envelope-and-exit-codes`
**Issue:** The table's exit-0 row says stdout carries "the directory answer" and the decision says the JSON envelope "always goes to stdout"; `--help` exits 0 with non-JSON usage text on stdout.
**Fix:** Scope the stdout/exit table to query invocations and name `--help` as the stated exception.

### [NIT:design] Render failure has no exit code
**Section:** `renderers-live-in-the-facade` / `failure-envelope-and-exit-codes`
**Issue:** `RenderJSON` is given an `error` return, but the mapping covers only "caught before `TOC`" → 2 and "from `TOC`" → 3, leaving a post-`TOC` render error unmapped.
**Fix:** Add a row (or state that a render error is exit 3).

### [NIT:scope] Scope's test list omits JSON rendering
**Section:** Scope, In (last bullet)
**Issue:** The bullet names flag parsing, exit-code mapping, root discovery and text rendering; the Testing section additionally requires a JSON-view test block (key order, absent defaults, unescaped `<`/`&`, trailing newline).
**Fix:** Add JSON rendering to the scope bullet.

## Verdict

REQUEST_CHANGES
One self-contradiction on golden bytes and two unspecified contracts block plan writing.
_Note: 1 finding(s) demoted from BLOCKING to NIT by the stage's blocking-class ceiling; current blocking_count is 2._
MILL_REVIEW_END
