# Review: Enclose: path:line ranges -> enclosing member glyphs (batch, at revision, CLI verb)

```yaml
verdict: REQUEST_CHANGES
reviewer_model: orchestrator
reviewed_file: _mill/discussion.md
date: 2026-10-08
```

## Findings

### [BLOCKING:design] Repository-root files have an unspellable unit, and neither path handles it
**Section:** Decisions / "Engine architecture: one memo, two byte sources" (working-tree and revision bullets); "Rejection vocabulary" step 3b.
**Issue:** `unitFor(".", pkg, clause)` returns `""` (internal/engine/walk.go:66-69), and `unitSpellable("")` is false (walk.go:82-104: "the repository root's empty unit" is rejected). Step 3b only checks `glyph.Self(glyph.Go, file)`, the *file* self glyph: `main.go#` is spellable, so a root-level `main.go:12` passes every rejection check. Then:
- Working tree: `fileEntry` leaves `Symbols` nil for an unspellable unit (walk.go:195-206). The discussion reads "`FileEntry.Symbols` ... is the candidate list" and never says what a nil list answers. The likely result is a silent `not_found` for a line inside a real function.
- Revision: the bullet stamps `stampSymbols(rec.symbols, unitFor(dir, v.pkg, rec.clause), file)` with no `unitSpellable` check. That mints IDs like `#Foo`, which `glyph.Parse` rejects. This is exactly the "minting a unit spelling glyph.Parse itself rejects" that walk.go:86-88 forbids. The two paths would also answer differently for the same file.
**Suggested fix:** Run the same `unitSpellable(unitFor(...))` gate on both paths before stamping. Decide the per-item answer for an unspellable unit explicitly: a rejection, either as a documented case of `bad_location` or a dedicated reason in `EncloseReasons`, never a silent `not_found`. Add tests for a root-level `.go` file on the working-tree and the revision path.

### [BLOCKING:design] Errors returned by `r.toc` on the working-tree path can fail the whole call
**Section:** Decisions / "Engine architecture" working-tree bullet ("Its `FileEntry.Error` means `unreadable`").
**Issue:** Requirement 1 says one bad item never fails the call. The working-tree path answers each file through `r.toc(file, ...)`, which returns its own errors besides `FileEntry.Error`:
- `resolveTarget` (`ErrTargetNotFound`, a TOCTOU after the `os.Lstat` check);
- `.gitignore` read failures for an ancestor or the file's own directory (internal/engine/toc.go:61-64, 118-121);
- `os.ReadDir` failure on the directory, e.g. EACCES (toc.go:124-127);
- the race "target ... no longer exists in directory" (toc.go:155-157).

The discussion maps only `FileEntry.Error`. A plan that propagates `r.toc`'s error would fail a whole pack over one unreadable directory.
**Suggested fix:** State that any error `r.toc` returns for one item becomes that item's rejection: `missing_file` for not-found or vanished, `unreadable` otherwise. State that the working-tree `Enclose` has no per-file whole-call error. The whole-call errors are only the revision-side ones already named (unknown rev, `DirFilesAtRevision`/`ReadBlob` failure). Add a test with an unreadable directory, or an unreadable `.gitignore`, beside a good location in the same batch.

### [NIT:consistency] The location regex as written is greedy and contradicts "shortest path"
**Section:** Decisions / "Location spelling".
**Issue:** `^(.+):(\d+)(?:-(\d+)|:(\d+))?$` with a greedy `.+` parses `a.go:12:5` as path `a.go:12`, line 5. That breaks the compiler-output form, the main input. The prose says "shortest path", but an implementer copying the regex gets the greedy behaviour.
**Suggested fix:** Spell it lazy, `^(.+?):(\d+)(?:-(\d+)|:(\d+))?$`, or describe the right-to-left parse without a regex. Add `a.go:12:5` → (`a.go`, 12, 12) explicitly to the parser table.

### [NIT:consistency] Relative paths are resolved against the root, unlike the CLI's `toc`/`delta`
**Section:** Decisions / "Path normalisation" and "CLI verb".
**Issue:** `repopath.RepoRelTarget(root, root, path)` makes relative paths root-relative. That matches requirement 3, but the other CLI verbs pass `base = cwd` (internal/cli/cli.go:388, 432, 594). The discussion does not call out the departure, so a plan writer copying `runToc`'s pipeline would reintroduce cwd-relative resolution.
**Suggested fix:** Say explicitly in the CLI section and the usage text that `enclose` resolves relative paths against the repository root, not the working directory, and why: tool output is repo-relative, and the consumer joins bare names itself. Add a CLI test run from a subdirectory.

### [NIT:consistency] Asymmetries between working-tree and revision answers should be documented
**Section:** Decisions / "Engine architecture" revision bullet.
**Issue:** There are two asymmetries:
- The working-tree vote excludes gitignored files (`ig.match` in walk.go:251 and toc.go:146-152). The revision vote includes every tracked `.go` file, tracked-and-gitignored ones too (gitsrc.go `DirFilesAtRevision` doc). The consumer maps at the rev and then `Resolve`s in the working tree. If a tracked, gitignored file with a deviating clause tips the vote, the unit minted at the rev differs from the working-tree unit, and the glyph will not resolve.
- A committed symlink answers as parsed link text at the rev, but as `unreadable` in the working tree.

Both match `DeltaGit`, and both are rare.
**Suggested fix:** No design change needed. Name both as known limits in `EncloseAt`'s doc comment so the consumer can recognise them.

## Verdict

REQUEST_CHANGES
The design honours all nine agreed decisions and the revision memo seam is sound, but two failure modes break the contract: repository-root files with an unspellable unit, and `r.toc` errors escaping as whole-call failures.
