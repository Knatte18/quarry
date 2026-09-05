// delta_test.go covers DeltaGit against four properties: it agrees with the pure Delta over
// hand-assembled entries, the status-letter table it maps is exactly what the discussion fixes, the
// working-tree side's untracked-file and gitignore rules hold, and every git-layer error surfaces
// through the facade's own aliased identity. Each test builds its own throwaway git repository under
// a fresh temporary directory and skips cleanly when no git binary is available on this machine —
// the same skip-versus-fail asymmetry internal/gitsrc/fixture_test.go's own newFixtureRepo already
// establishes.

package quarry

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Knatte18/quarry/internal/gitsrc"
)

// deltaFixture is a throwaway git repository built fresh for one test under t.TempDir(). This is a
// deliberate per-package copy of internal/gitsrc/fixture_test.go's fixtureRepo, because Go test
// helpers are not importable across packages.
type deltaFixture struct {
	t    testing.TB
	root string
}

// newDeltaFixture initialises a repository under a fresh temporary directory, with a fixed identity
// and a fixed default branch name so no machine's global git configuration can change the fixture's
// behaviour. It skips the whole test, with the reason, when no git binary is available.
func newDeltaFixture(t testing.TB) *deltaFixture {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not found on this machine")
	}

	f := &deltaFixture{t: t, root: t.TempDir()}
	f.git("init", "--quiet", "--initial-branch=main")
	f.git("config", "user.name", "quarry-delta-fixture")
	f.git("config", "user.email", "quarry-delta-fixture@example.com")
	return f
}

// git runs one git invocation against the fixture's root, failing the test immediately on error: a
// fixture-construction step that fails is a broken test, never a normal state worth skipping.
func (f *deltaFixture) git(args ...string) string {
	f.t.Helper()
	cmd := exec.Command("git", append([]string{"-C", f.root}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		f.t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

// write writes content to path, repository-relative, creating parent directories as needed. It does
// not stage the write.
func (f *deltaFixture) write(path, content string) {
	f.t.Helper()
	full := filepath.Join(f.root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		f.t.Fatalf("mkdir %q: %v", filepath.Dir(full), err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		f.t.Fatalf("write %q: %v", full, err)
	}
}

// commit stages every pending change and commits it with message, returning the resulting commit's
// identifier.
func (f *deltaFixture) commit(message string) string {
	f.git("add", "-A")
	f.git("commit", "--quiet", "-m", message)
	return f.git("rev-parse", "HEAD")
}

// writeAndCommit writes content to path and commits it in one step, returning the resulting commit's
// identifier.
func (f *deltaFixture) writeAndCommit(path, content, message string) string {
	f.write(path, content)
	return f.commit(message)
}

// removeUnstaged deletes path from the working tree without staging the deletion, leaving it present
// in the index but absent from disk.
func (f *deltaFixture) removeUnstaged(path string) {
	f.t.Helper()
	full := filepath.Join(f.root, filepath.FromSlash(path))
	if err := os.Remove(full); err != nil {
		f.t.Fatalf("remove %q: %v", full, err)
	}
}

// leaveUntracked writes content to path without staging or committing it, leaving it untracked.
func (f *deltaFixture) leaveUntracked(path, content string) {
	f.write(path, content)
}

// stage runs "git add" over path, without committing.
func (f *deltaFixture) stage(path string) {
	f.git("add", path)
}

// makeUnmergedPath produces a conflicted path reachable on the working-tree side during a merge: it
// commits a base version of path, changes it on a side branch, changes it again on main, then
// attempts to merge the side branch into main and expects, rather than resolves, the resulting
// conflict.
func (f *deltaFixture) makeUnmergedPath(path string) {
	f.t.Helper()
	f.writeAndCommit(path, "package pkg\n\nfunc Base() {}\n", "base commit for "+path)
	f.git("checkout", "--quiet", "-b", "delta-fixture-conflict")
	f.writeAndCommit(path, "package pkg\n\nfunc Side() {}\n", "conflict branch change")
	f.git("checkout", "--quiet", "main")
	f.writeAndCommit(path, "package pkg\n\nfunc Main() {}\n", "main branch change")

	cmd := exec.Command("git", "-C", f.root, "merge", "--quiet", "--no-edit", "delta-fixture-conflict")
	// The merge is expected to fail with a conflict on path; a merge that succeeds cleanly, leaving
	// no unmerged path at all, is the fixture-construction failure worth stopping the test over.
	if err := cmd.Run(); err == nil {
		f.t.Fatalf("merge of delta-fixture-conflict into main succeeded cleanly; wanted a conflict on %q", path)
	}
}

// openDeltaRepo opens a *Repo rooted at root, failing the test immediately on error.
func openDeltaRepo(t testing.TB, root string) *Repo {
	t.Helper()
	r, err := Open(root)
	if err != nil {
		t.Fatalf("Open(%q) returned error: %v", root, err)
	}
	return r
}

// findFile returns the DeltaFile in files whose Path equals path, and whether it was found.
func findFile(files []DeltaFile, path string) (DeltaFile, bool) {
	for _, f := range files {
		if f.Path == path {
			return f, true
		}
	}
	return DeltaFile{}, false
}

// hasSymbolID reports whether syms contains a Symbol whose ID equals id.
func hasSymbolID(syms []Symbol, id string) bool {
	for _, s := range syms {
		if s.ID == id {
			return true
		}
	}
	return false
}

// TestDeltaGit_AgreesWithDelta is the load-bearing agreement test: it runs DeltaGit on a fixture and,
// separately, assembles the same entries by hand and runs the pure Delta on them, asserting the two
// answers are equal apart from the revision echo -- the two paths must not be able to disagree, since
// one is documented as a convenience over the other.
func TestDeltaGit_AgreesWithDelta(t *testing.T) {
	f := newDeltaFixture(t)
	const before = "package pkg\n\nfunc Foo() {}\n"
	const after = "package pkg\n\nfunc Foo() {}\n\nfunc Bar() {}\n"
	from := f.writeAndCommit("pkg/a.go", before, "add Foo")
	to := f.writeAndCommit("pkg/a.go", after, "add Bar")

	r := openDeltaRepo(t, f.root)

	gitAnswer, err := r.DeltaGit(from, to, ".")
	if err != nil {
		t.Fatalf("DeltaGit(%q, %q, %q) returned error: %v", from, to, ".", err)
	}

	manual := []DeltaEntry{
		{
			Path:       "pkg/a.go",
			Before:     []byte(before),
			After:      []byte(after),
			BeforeUnit: "pkg",
			AfterUnit:  "pkg",
		},
	}
	pureAnswer, err := r.Delta(manual)
	if err != nil {
		t.Fatalf("Delta(manual) returned error: %v", err)
	}

	if !reflect.DeepEqual(gitAnswer.DeltaAnswer, pureAnswer) {
		t.Errorf("DeltaGit's answer disagrees with the hand-assembled Delta call:\ngit:  %+v\npure: %+v", gitAnswer.DeltaAnswer, pureAnswer)
	}
	if gitAnswer.From != from {
		t.Errorf("GitDeltaAnswer.From = %q; want %q", gitAnswer.From, from)
	}
	if gitAnswer.To == nil || *gitAnswer.To != to {
		t.Errorf("GitDeltaAnswer.To = %v; want a pointer to %q", gitAnswer.To, to)
	}
}

// TestDeltaGit_StatusLetterMapping asserts the disposition each status letter produces: added,
// modified, deleted and typechange are extracted normally; an unmerged path yields an error
// disposition with no extraction attempted, since a conflicted file's content is conflict markers and
// extracting it as source would be a silent lie; and an unrecognised letter yields an error naming
// the letter.
func TestDeltaGit_StatusLetterMapping(t *testing.T) {
	t.Run("AddedModifiedDeletedTypechange", func(t *testing.T) {
		f := newDeltaFixture(t)
		f.write("modified.go", "package pkg\n\nfunc Foo() {}\n")
		f.write("deleted.go", "package pkg\n\nfunc Gone() {}\n")
		f.write("typechange.go", "package pkg\n\nfunc TC() {}\n")
		base := f.commit("base")

		f.write("modified.go", "package pkg\n\nfunc Foo() {}\n\nfunc Bar() {}\n")
		f.removeUnstaged("deleted.go")
		full := filepath.Join(f.root, "typechange.go")
		if err := os.Remove(full); err != nil {
			t.Fatalf("remove %q: %v", full, err)
		}
		if err := os.Symlink("modified.go", full); err != nil {
			t.Fatalf("symlink %q: %v", full, err)
		}
		f.write("added.go", "package pkg\n\nfunc Added() {}\n")
		f.stage("added.go")

		r := openDeltaRepo(t, f.root)
		answer, err := r.DeltaGit(base, "", ".")
		if err != nil {
			t.Fatalf("DeltaGit(%q, \"\", \".\") returned error: %v", base, err)
		}

		cases := []struct {
			path string
			want Disposition
		}{
			{"added.go", DispositionAdded},
			{"modified.go", DispositionChanged},
			{"deleted.go", DispositionRemoved},
			{"typechange.go", DispositionChanged},
		}
		for _, c := range cases {
			df, ok := findFile(answer.Files, c.path)
			if !ok {
				t.Errorf("Files is missing an entry for %q", c.path)
				continue
			}
			if df.Disposition != c.want {
				t.Errorf("Files[%q].Disposition = %q; want %q", c.path, df.Disposition, c.want)
			}
		}
	})

	t.Run("Unmerged", func(t *testing.T) {
		f := newDeltaFixture(t)
		start := f.writeAndCommit("README.md", "root\n", "init")
		// pkg/conflict.go sits in its own directory with a spellable unit, so an assertion that no
		// symbol from it was extracted is meaningful -- at the repository root a symbol would be
		// suppressed anyway (the empty unit is unspellable), which would prove nothing about the
		// refusal path this subtest exists to cover.
		f.makeUnmergedPath("pkg/conflict.go")

		r := openDeltaRepo(t, f.root)
		answer, err := r.DeltaGit(start, "", ".")
		if err != nil {
			t.Fatalf("DeltaGit(%q, \"\", \".\") returned error: %v", start, err)
		}

		df, ok := findFile(answer.Files, "pkg/conflict.go")
		if !ok {
			t.Fatalf("Files is missing an entry for %q", "pkg/conflict.go")
		}
		if df.Disposition != DispositionError {
			t.Errorf("Files[%q].Disposition = %q; want %q", "pkg/conflict.go", df.Disposition, DispositionError)
		}
		if df.Error != "unmerged path" {
			t.Errorf("Files[%q].Error = %q; want %q", "pkg/conflict.go", df.Error, "unmerged path")
		}
		if hasSymbolID(answer.Created, "pkg#Base") || hasSymbolID(answer.Created, "pkg#Side") || hasSymbolID(answer.Created, "pkg#Main") {
			t.Errorf("Created = %+v; an unmerged path must contribute no extraction attempt", answer.Created)
		}
	})

	t.Run("UnrecognisedLetter", func(t *testing.T) {
		f := newDeltaFixture(t)
		base := f.writeAndCommit("x.go", "package pkg\n\nfunc X() {}\n", "base")

		gr, err := gitsrc.Open(f.root)
		if err != nil {
			t.Fatalf("gitsrc.Open(%q) returned error: %v", f.root, err)
		}
		r := openDeltaRepo(t, f.root)

		entry := r.deltaEntryForChange(gr, base, "", gitsrc.Change{Path: "x.go", Status: "Z"})
		if entry.Refusal == "" {
			t.Fatalf("deltaEntryForChange with an unrecognised status returned no Refusal")
		}
		if !strings.Contains(entry.Refusal, "Z") {
			t.Errorf("Refusal = %q; want it to name the letter %q verbatim", entry.Refusal, "Z")
		}
	})
}

// TestDeltaGit_WorkingTreeSide asserts the working-tree side's own rules: an untracked, never-staged
// source file is enumerated, reaches the delta as added and has its symbols in the created array; an
// untracked file matched by a gitignore pattern does not; a tracked-but-gitignored file is kept,
// asserted deliberately so the documented divergence from the table-of-contents listing rule cannot
// be silently reverted; a source file deleted from the working tree but still in the index does not
// fail the call; and both sides of a directory's clause vote enumerate the same file set, so an
// unrelated, unchanged file sharing a directory with a tracked-and-gitignored file is never reported
// as a spurious create-plus-delete pair.
func TestDeltaGit_WorkingTreeSide(t *testing.T) {
	f := newDeltaFixture(t)

	f.write("keep/normal.go", "package keep\n\nfunc Normal() {}\n")
	f.write("keep/tracked_ignored.go", "package keep\n\nfunc Keep() {}\n")
	f.write("sub/a.go", "package sub\n\nfunc A() {}\n")
	// B's body is deliberately non-trivial and unrelated to AA's, added to sub/a.go below: two
	// empty-bodied functions of the same kind, in the same unit, differing only in name would
	// satisfy the exact-tier rename classifier's structural-plus-token conditions, misreporting this
	// unrelated delete-plus-add as a rename.
	f.write("sub/b.go", "package sub\n\nfunc B() {\n\treturn\n}\n")
	f.commit("base")
	f.write(".gitignore", "keep/tracked_ignored.go\nignoreddir/*\n")
	from := f.commit("add gitignore")

	// An untracked, never-staged source file: reaches the delta as added.
	f.leaveUntracked("created/unstaged.go", "package created\n\nfunc New() {}\n")
	// An untracked file matched by the gitignore pattern above: must never be enumerated.
	f.leaveUntracked("ignoreddir/skip.go", "package ignoreddir\n\nfunc Skip() {}\n")
	// A tracked-but-gitignored file, modified in the working tree: must still be reported.
	f.write("keep/tracked_ignored.go", "package keep\n\nfunc Keep() {}\n\nfunc KeepMore() {}\n")
	// A source file deleted from the working tree but still in the index: must not fail the call.
	f.removeUnstaged("sub/b.go")
	// sub/a.go also changes, so "sub" enters the directory clause vote alongside the now-missing
	// b.go. AA's body is deliberately distinct from B's (see the comment above sub/b.go's write).
	f.write("sub/a.go", "package sub\n\nfunc A() {}\n\nfunc AA() {\n\tvar x = 2\n\t_ = x\n}\n")

	r := openDeltaRepo(t, f.root)
	answer, err := r.DeltaGit(from, "", ".")
	if err != nil {
		t.Fatalf("DeltaGit(%q, \"\", \".\") returned error: %v", from, err)
	}

	if df, ok := findFile(answer.Files, "created/unstaged.go"); !ok {
		t.Errorf("Files is missing an entry for the untracked %q", "created/unstaged.go")
	} else if df.Disposition != DispositionAdded {
		t.Errorf("Files[%q].Disposition = %q; want %q", "created/unstaged.go", df.Disposition, DispositionAdded)
	}
	if !hasSymbolID(answer.Created, "created#New") {
		t.Errorf("Created = %+v; want it to contain %q", answer.Created, "created#New")
	}

	if _, ok := findFile(answer.Files, "ignoreddir/skip.go"); ok {
		t.Errorf("Files wrongly contains an entry for the gitignored untracked path %q", "ignoreddir/skip.go")
	}

	if df, ok := findFile(answer.Files, "keep/tracked_ignored.go"); !ok {
		t.Errorf("Files is missing an entry for the tracked-but-gitignored %q", "keep/tracked_ignored.go")
	} else if df.Disposition != DispositionChanged {
		t.Errorf("Files[%q].Disposition = %q; want %q", "keep/tracked_ignored.go", df.Disposition, DispositionChanged)
	}
	if !hasSymbolID(answer.Created, "keep#KeepMore") {
		t.Errorf("Created = %+v; want it to contain %q", answer.Created, "keep#KeepMore")
	}

	// keep/normal.go never changed and must never be reported: neither created, deleted nor
	// modified, which is the evidence a divergent enumeration between the two sides of the "keep"
	// directory's clause vote would otherwise produce.
	if hasSymbolID(answer.Created, "keep#Normal") || hasSymbolID(answer.Deleted, "keep#Normal") {
		t.Errorf("Created/Deleted wrongly report %q; the directory's clause vote must enumerate the same set on both sides", "keep#Normal")
	}
	for _, m := range answer.Modified {
		if m.ID == "keep#Normal" {
			t.Errorf("Modified wrongly reports %q, which never changed", "keep#Normal")
		}
	}

	if df, ok := findFile(answer.Files, "sub/b.go"); !ok {
		t.Errorf("Files is missing an entry for %q", "sub/b.go")
	} else if df.Disposition != DispositionRemoved {
		t.Errorf("Files[%q].Disposition = %q; want %q", "sub/b.go", df.Disposition, DispositionRemoved)
	}
	if !hasSymbolID(answer.Created, "sub#AA") {
		t.Errorf("Created = %+v; want it to contain %q", answer.Created, "sub#AA")
	}
}

// TestDeltaGit_ErrorIdentity asserts that an unresolvable revision, a root that is a subdirectory of
// a repository, and a root outside any repository each surface through the facade's own aliased
// sentinels, matched by identity, and that the two typed errors yield their fields when extracted by
// type.
func TestDeltaGit_ErrorIdentity(t *testing.T) {
	t.Run("UnknownRevision", func(t *testing.T) {
		f := newDeltaFixture(t)
		f.writeAndCommit("a.go", "package pkg\n\nfunc A() {}\n", "base")

		r := openDeltaRepo(t, f.root)
		_, err := r.DeltaGit("does-not-exist-rev", "", ".")
		if !errors.Is(err, ErrUnknownRevision) {
			t.Fatalf("DeltaGit error = %v; want errors.Is(err, ErrUnknownRevision)", err)
		}
		var revErr *UnknownRevisionError
		if !errors.As(err, &revErr) {
			t.Fatalf("DeltaGit error = %v; want errors.As(err, &revErr) against *UnknownRevisionError to succeed", err)
		}
		if revErr.Rev != "does-not-exist-rev" {
			t.Errorf("revErr.Rev = %q; want %q", revErr.Rev, "does-not-exist-rev")
		}
	})

	t.Run("RootNotTopLevel", func(t *testing.T) {
		f := newDeltaFixture(t)
		f.writeAndCommit("sub/a.go", "package sub\n\nfunc A() {}\n", "base")

		subRoot := filepath.Join(f.root, "sub")
		r := openDeltaRepo(t, subRoot)
		_, err := r.DeltaGit("HEAD", "", ".")
		if !errors.Is(err, ErrRootNotTopLevel) {
			t.Fatalf("DeltaGit error = %v; want errors.Is(err, ErrRootNotTopLevel)", err)
		}
		var rootErr *RootNotTopLevelError
		if !errors.As(err, &rootErr) {
			t.Fatalf("DeltaGit error = %v; want errors.As(err, &rootErr) against *RootNotTopLevelError to succeed", err)
		}
		if rootErr.Root != subRoot {
			t.Errorf("rootErr.Root = %q; want %q", rootErr.Root, subRoot)
		}
		if rootErr.TopLevel == "" {
			t.Errorf("rootErr.TopLevel = %q; want a non-empty top-level path", rootErr.TopLevel)
		}
	})

	t.Run("NotARepository", func(t *testing.T) {
		// A directory outside any git repository is needed here, which by definition rules out
		// writeScratchTree's own tree (nested inside this repository's working copy): t.TempDir() is
		// the precedent internal/gitsrc/gitsrc_test.go's own "outside any repository" case already
		// sets for exactly this one need.
		root := t.TempDir()
		r := openDeltaRepo(t, root)
		_, err := r.DeltaGit("HEAD", "", ".")
		if !errors.Is(err, ErrNotARepository) {
			t.Fatalf("DeltaGit error = %v; want errors.Is(err, ErrNotARepository)", err)
		}
	})
}
