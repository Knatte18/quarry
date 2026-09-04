MILL_REVIEW_BEGIN
# Review: Facade + CLI, toc (T5a)

```yaml
duration_s: 257.0
verdict: APPROVE
reviewer_model: opushigh
reviewer_self_id: Claude Opus 5 (claude-opus-5), high reasoning effort
reviewed_file: _mill/discussion.md
date: 2026-09-04
```

## Findings

### [NIT:consistency] Golden test contradicts cwd-relative targets
**Demoted-from:** BLOCKING
**Section:** `golden-tests-run-the-cli-in-process` vs `root-discovery-and-target-interpretation`
**Issue:** The latter decides the target is resolved against the *process* cwd, then converted to repo-relative, with anything outside the root exiting 1 — so `cli.Run("toc","internal/logger","--root",<loomyard>)` from the quarry test's own cwd resolves to `<quarry>/…/internal/logger`, lands outside the Loomyard root, and returns exit 1, never a golden; the same applies to the §4 `layout.go` byte-for-byte case.
**Fix:** State which frame `--root` puts the target in (e.g. `--root` also rebases target interpretation, or the test passes an absolute target under `$LADDER_LOOMYARD_REPO`), and reconcile it with the `no t.Chdir` rule.

### [NIT:consistency] `(exit code: 0)` line is not the before side's layout
**Demoted-from:** BLOCKING
**Section:** `after-outputs-are-the-goldens`
**Issue:** "Each `.txt` mirrors the before side's own layout: … a blank line, and `(exit code: 0)`" is false for exactly the files being paired: `docs/research/output-formats/toc-dir.txt` ends at line 96 with `}`, `toc-file.txt` at line 226 with `}`, and both `toc-*-compact.txt` end at their last text line — none carries an exit-code line (only `impact*`, `definition*`, `assert-no-callers` do, and the before `INDEX.md`'s "the exit code is at the bottom of each file" is itself wrong). Since the after files *are* byte-compared goldens, the trailer decides their bytes.
**Fix:** Decide the trailer on its own merits and drop or correct the "mirrors the before side" premise, since either choice makes one of the two claims untrue.

### [NIT:design] `--text` failure output unstated
**Section:** `failure-envelope-and-exit-codes`
**Issue:** "The JSON envelope always goes to stdout, including on failure" is written without reference to `--text`, so it is unclear whether a text-mode caller's stdout carries JSON on exit 1/2/3.
**Fix:** Say explicitly that the failure envelope is JSON on stdout regardless of `--text` (or name the text failure form).

### [NIT:design] File-form path when the answer's `dir` is `.`
**Section:** `text-view-grammar`, file form
**Issue:** The template `<dir>/<name>` yields `./README.md` for a root-level file target (engine `joinRel` spells it `README.md`), contradicting the same paragraph's "full repository-relative path" for a grammar claimed fixed to the character.
**Fix:** State that the file-form path is the engine's repo-relative join, so `dir == "."` emits the bare name.

### [NIT:design] `[error …]` tag outside prose normalisation
**Section:** `text-view-grammar`, tags
**Issue:** Normalisation is scoped to `doc`, `header` and `signature`; `FileEntry.Error` is arbitrary text emitted inside a bracketed tag, so a multi-line message would break the one-record-per-line property the format rests on.
**Fix:** Extend the collapse rule to the error message, or state that it is single-line by construction.

### [NIT:decision] `--help` has no stated disposition
**Section:** `cli-shape` / `flag-semantics`
**Issue:** The flag set is closed at five, and `--help`/`-h` therefore falls through to "unknown flag → exit 2", which is never said out loud for the one flag every CLI user tries first.
**Fix:** Record `--help` explicitly — either exit 0 with usage on stdout, or exit 2 as an unknown flag.

## Verdict

APPROVE
Two verified contradictions: golden-test target frame, and the after-file exit-code trailer.
_Note: 2 finding(s) demoted from BLOCKING to NIT by the stage's blocking-class ceiling; current blocking_count is 0._
MILL_REVIEW_END
