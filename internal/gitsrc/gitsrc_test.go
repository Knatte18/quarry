// gitsrc_test.go asserts every behaviour cards 26 to 28 promise, over the throwaway repositories
// fixture_test.go builds: opening, revision verification, changed and untracked paths, directory
// enumeration on both sides, and non-ASCII path handling. The status-letter mapping is deliberately
// not asserted here -- this package returns the raw letter and maps nothing, so the mapping's own
// test belongs to the layer that builds entries.

package gitsrc

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// openFixture opens f's root, failing the test immediately if Open itself errors: a fixture that
// Open rejects is a broken test, not a case any of these tests means to exercise.
func openFixture(t testing.TB, f *fixtureRepo) *Repo {
	t.Helper()
	repo, err := Open(f.root)
	if err != nil {
		t.Fatalf("Open(%q) = _, %v; want nil error", f.root, err)
	}
	return repo
}

// changesByPath collapses changes into a path-to-status-letter map for comparison, since
// ChangedPaths makes no ordering promise.
func changesByPath(changes []Change) map[string]string {
	m := make(map[string]string, len(changes))
	for _, c := range changes {
		m[c.Path] = c.Status
	}
	return m
}

// containsPath reports whether path appears anywhere in paths.
func containsPath(paths []string, path string) bool {
	for _, p := range paths {
		if p == path {
			return true
		}
	}
	return false
}

func TestOpen(t *testing.T) {
	t.Run("TopLevelAccepted", func(t *testing.T) {
		f := newFixtureRepo(t)
		f.writeAndCommit("a.go", "package a\n", "init")

		if _, err := Open(f.root); err != nil {
			t.Errorf("Open(%q) = _, %v; want nil error", f.root, err)
		}
	})

	t.Run("SubdirectoryOfRepositoryRejected", func(t *testing.T) {
		f := newFixtureRepo(t)
		f.write("sub/a.go", "package sub\n")
		f.commit("init")
		sub := filepath.Join(f.root, "sub")

		_, err := Open(sub)
		var topErr *RootNotTopLevelError
		if !errors.As(err, &topErr) {
			t.Fatalf("Open(%q) = _, %v; want a *RootNotTopLevelError", sub, err)
		}
		if topErr.Root != sub {
			t.Errorf("RootNotTopLevelError.Root = %q; want %q", topErr.Root, sub)
		}
	})

	t.Run("OutsideAnyRepositoryRejected", func(t *testing.T) {
		outside := t.TempDir()

		_, err := Open(outside)
		if !errors.Is(err, ErrNotARepository) {
			t.Fatalf("Open(%q) = _, %v; want ErrNotARepository", outside, err)
		}
	})

	t.Run("SymlinkedRootAccepted", func(t *testing.T) {
		f := newFixtureRepo(t)
		f.writeAndCommit("a.go", "package a\n", "init")

		link := filepath.Join(t.TempDir(), "link-to-repo")
		if err := os.Symlink(f.root, link); err != nil {
			t.Skipf("symlink creation not supported on this machine: %v", err)
		}

		if _, err := Open(link); err != nil {
			t.Errorf("Open(%q) = _, %v; want nil error for a repository reached through a symlink", link, err)
		}
	})
}

func TestVerifyRevision(t *testing.T) {
	f := newFixtureRepo(t)
	f.writeAndCommit("a.go", "package a\n", "init")
	repo := openFixture(t, f)

	if err := repo.VerifyRevision("HEAD"); err != nil {
		t.Errorf("VerifyRevision(%q) = %v; want nil", "HEAD", err)
	}

	err := repo.VerifyRevision("does-not-exist")
	var revErr *UnknownRevisionError
	if !errors.As(err, &revErr) {
		t.Fatalf("VerifyRevision(%q) = %v; want a *UnknownRevisionError", "does-not-exist", err)
	}
	if revErr.Rev != "does-not-exist" {
		t.Errorf("UnknownRevisionError.Rev = %q; want %q", revErr.Rev, "does-not-exist")
	}
	if !errors.Is(err, ErrUnknownRevision) {
		t.Errorf("VerifyRevision(%q) error does not match ErrUnknownRevision by sentinel identity", "does-not-exist")
	}
}

func TestChangedPaths(t *testing.T) {
	t.Run("AddedModifiedDeletedAndUncommitted", func(t *testing.T) {
		f := newFixtureRepo(t)
		f.write("keep.go", "package p\n\nvar Keep = 1\n")
		f.write("gone.go", "package p\n\nvar Gone = 1\n")
		before := f.commit("init")

		// None of these edits are committed: this exercises the working-tree-as-after-side form
		// against changes that were never committed. "added.go" is staged with a plain "git add"
		// but not committed, since ChangedPaths -- a rev-vs-worktree diff -- only ever considers
		// paths git already knows through the index or the named revision; a file that has never
		// been staged at all is invisible to it and is UntrackedPaths' job instead, exactly as
		// TestUntrackedPaths asserts.
		f.write("keep.go", "package p\n\nvar Keep = 2\n")
		f.remove("gone.go", true)
		f.write("added.go", "package p\n\nvar Added = 1\n")
		f.git("add", "added.go")

		repo := openFixture(t, f)
		changes, err := repo.ChangedPaths(before, "", ".")
		if err != nil {
			t.Fatalf("ChangedPaths(%q, %q, %q) = _, %v", before, "", ".", err)
		}

		want := map[string]string{"keep.go": "M", "gone.go": "D", "added.go": "A"}
		got := changesByPath(changes)
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("ChangedPaths(%q, %q, %q) mismatch (-want +got):\n%s", before, "", ".", diff)
		}
	})

	t.Run("RenameArrivesAsDeletePlusAdd", func(t *testing.T) {
		f := newFixtureRepo(t)
		before := f.writeAndCommit("old.go", "package p\n\nvar Renamed = 1\n", "init")
		f.rename("old.go", "new.go")

		repo := openFixture(t, f)
		changes, err := repo.ChangedPaths(before, "", ".")
		if err != nil {
			t.Fatalf("ChangedPaths(%q, %q, %q) = _, %v", before, "", ".", err)
		}

		want := map[string]string{"old.go": "D", "new.go": "A"}
		got := changesByPath(changes)
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("ChangedPaths(%q, %q, %q) mismatch (-want +got); rename detection must be disabled:\n%s", before, "", ".", diff)
		}
	})

	t.Run("UnmergedPathReportsTheUnmergedLetter", func(t *testing.T) {
		f := newFixtureRepo(t)
		f.writeAndCommit("unrelated.go", "package p\n", "init")
		before := f.git("rev-parse", "HEAD")
		f.makeUnmergedPath("conflict.go")

		repo := openFixture(t, f)
		changes, err := repo.ChangedPaths(before, "", ".")
		if err != nil {
			t.Fatalf("ChangedPaths(%q, %q, %q) = _, %v", before, "", ".", err)
		}

		got := changesByPath(changes)
		if got["conflict.go"] != "U" {
			t.Errorf("ChangedPaths(%q, %q, %q)[%q] = %q; want %q", before, "", ".", "conflict.go", got["conflict.go"], "U")
		}
	})

	t.Run("TwoRevisionForm", func(t *testing.T) {
		f := newFixtureRepo(t)
		before := f.writeAndCommit("a.go", "package p\n\nvar A = 1\n", "init")
		after := f.writeAndCommit("a.go", "package p\n\nvar A = 2\n", "change")

		repo := openFixture(t, f)
		changes, err := repo.ChangedPaths(before, after, ".")
		if err != nil {
			t.Fatalf("ChangedPaths(%q, %q, %q) = _, %v", before, after, ".", err)
		}

		want := map[string]string{"a.go": "M"}
		got := changesByPath(changes)
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("ChangedPaths(%q, %q, %q) mismatch (-want +got):\n%s", before, after, ".", diff)
		}
	})
}

func TestUntrackedPaths(t *testing.T) {
	f := newFixtureRepo(t)
	f.writeAndCommit("committed.go", "package p\n", "init")
	f.writeGitignore("ignored.go")
	f.leaveUntracked("untracked.go", "package p\n")
	f.leaveUntracked("ignored.go", "package p\n")

	repo := openFixture(t, f)
	paths, err := repo.UntrackedPaths(".")
	if err != nil {
		t.Fatalf("UntrackedPaths(%q) = _, %v", ".", err)
	}

	if !containsPath(paths, "untracked.go") {
		t.Errorf("UntrackedPaths(%q) = %v; want it to contain %q", ".", paths, "untracked.go")
	}
	if containsPath(paths, "ignored.go") {
		t.Errorf("UntrackedPaths(%q) = %v; want it to not contain gitignored %q", ".", paths, "ignored.go")
	}
}

func TestDirFilesSymmetry(t *testing.T) {
	t.Run("TrackedGitignoredFileVotesOnBothSidesOrNeither", func(t *testing.T) {
		f := newFixtureRepo(t)
		f.write("pkg/tracked.go", "package pkg\n")
		before := f.commit("init")
		f.writeGitignore("pkg/tracked.go")

		repo := openFixture(t, f)
		atRev, err := repo.DirFilesAtRevision(before, "pkg")
		if err != nil {
			t.Fatalf("DirFilesAtRevision(%q, %q) = _, %v", before, "pkg", err)
		}
		wt, err := repo.DirFilesInWorkingTree("pkg")
		if err != nil {
			t.Fatalf("DirFilesInWorkingTree(%q) = _, %v", "pkg", err)
		}

		if !containsPath(atRev, "pkg/tracked.go") {
			t.Errorf("DirFilesAtRevision(%q, %q) = %v; want it to contain the tracked, gitignored %q", before, "pkg", atRev, "pkg/tracked.go")
		}
		if !containsPath(wt, "pkg/tracked.go") {
			t.Errorf("DirFilesInWorkingTree(%q) = %v; want it to contain the tracked, gitignored %q", "pkg", wt, "pkg/tracked.go")
		}
	})

	t.Run("SubdirectoryFileExcludedFromBothSides", func(t *testing.T) {
		f := newFixtureRepo(t)
		f.write("pkg/a.go", "package pkg\n")
		f.write("pkg/deeper/b.go", "package deeper\n")
		before := f.commit("init")

		repo := openFixture(t, f)
		atRev, err := repo.DirFilesAtRevision(before, "pkg")
		if err != nil {
			t.Fatalf("DirFilesAtRevision(%q, %q) = _, %v", before, "pkg", err)
		}
		wt, err := repo.DirFilesInWorkingTree("pkg")
		if err != nil {
			t.Fatalf("DirFilesInWorkingTree(%q) = _, %v", "pkg", err)
		}

		for _, files := range [][]string{atRev, wt} {
			if !containsPath(files, "pkg/a.go") {
				t.Errorf("dir listing %v is missing the immediate child %q", files, "pkg/a.go")
			}
			if containsPath(files, "pkg/deeper/b.go") {
				t.Errorf("dir listing %v wrongly includes the subdirectory file %q", files, "pkg/deeper/b.go")
			}
		}
	})

	t.Run("RemovedFromWorkingTreeStillInIndexListed", func(t *testing.T) {
		f := newFixtureRepo(t)
		f.write("pkg/a.go", "package pkg\n")
		f.commit("init")
		f.remove("pkg/a.go", false)

		repo := openFixture(t, f)
		wt, err := repo.DirFilesInWorkingTree("pkg")
		if err != nil {
			t.Fatalf("DirFilesInWorkingTree(%q) = _, %v", "pkg", err)
		}

		if !containsPath(wt, "pkg/a.go") {
			t.Errorf("DirFilesInWorkingTree(%q) = %v; want it to still contain %q, removed from disk but not from the index", "pkg", wt, "pkg/a.go")
		}
	})
}

func TestNonASCIIPathRoundTrips(t *testing.T) {
	const trackedPath = "café.go"
	const untrackedPath = "naïve.go"
	const trackedContent = "package p\n"

	f := newFixtureRepo(t)
	before := f.writeAndCommit(trackedPath, trackedContent, "init")
	f.write(trackedPath, "package p\n\nvar X = 1\n")
	f.leaveUntracked(untrackedPath, "package p\n")

	repo := openFixture(t, f)

	changes, err := repo.ChangedPaths(before, "", ".")
	if err != nil {
		t.Fatalf("ChangedPaths(%q, %q, %q) = _, %v", before, "", ".", err)
	}
	if got := changesByPath(changes)[trackedPath]; got != "M" {
		t.Errorf("ChangedPaths(%q, %q, %q)[%q] = %q; want %q", before, "", ".", trackedPath, got, "M")
	}

	untracked, err := repo.UntrackedPaths(".")
	if err != nil {
		t.Fatalf("UntrackedPaths(%q) = _, %v", ".", err)
	}
	if !containsPath(untracked, untrackedPath) {
		t.Errorf("UntrackedPaths(%q) = %v; want it to contain %q", ".", untracked, untrackedPath)
	}

	blob, err := repo.ReadBlob(before, trackedPath)
	if err != nil {
		t.Fatalf("ReadBlob(%q, %q) = _, %v", before, trackedPath, err)
	}
	if string(blob) != trackedContent {
		t.Errorf("ReadBlob(%q, %q) = %q; want %q", before, trackedPath, blob, trackedContent)
	}

	atRev, err := repo.DirFilesAtRevision(before, ".")
	if err != nil {
		t.Fatalf("DirFilesAtRevision(%q, %q) = _, %v", before, ".", err)
	}
	if !containsPath(atRev, trackedPath) {
		t.Errorf("DirFilesAtRevision(%q, %q) = %v; want it to contain %q", before, ".", atRev, trackedPath)
	}

	wt, err := repo.DirFilesInWorkingTree(".")
	if err != nil {
		t.Fatalf("DirFilesInWorkingTree(%q) = _, %v", ".", err)
	}
	if !containsPath(wt, trackedPath) {
		t.Errorf("DirFilesInWorkingTree(%q) = %v; want it to contain %q", ".", wt, trackedPath)
	}
	if !containsPath(wt, untrackedPath) {
		t.Errorf("DirFilesInWorkingTree(%q) = %v; want it to contain %q", ".", wt, untrackedPath)
	}
}

func TestGitFailureSurfacesAsErrorNotPanic(t *testing.T) {
	f := newFixtureRepo(t)
	f.writeAndCommit("a.go", "package p\n", "init")
	repo := openFixture(t, f)

	_, err := repo.ReadBlob("HEAD", "does-not-exist.go")
	if err == nil {
		t.Fatalf("ReadBlob(%q, %q) = _, nil; want a non-nil error for a path absent at that revision", "HEAD", "does-not-exist.go")
	}
}
