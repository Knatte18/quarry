// fixture_test.go builds a small, throwaway git repository per test, driven entirely by process
// invocations exactly as the rest of this package's own operations are, so what a test exercises is
// the same git this package itself shells out to.

package gitsrc

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// fixtureRepo is a throwaway git repository built fresh for one test under t.TempDir(). Every
// fixture is built fresh per test; no test reuses another's repository.
type fixtureRepo struct {
	t    testing.TB
	root string
}

// newFixtureRepo initialises a repository under a fresh temporary directory, with a fixed identity
// and a fixed default branch name so no machine's global git configuration can change the fixture's
// behaviour.
//
// It skips the whole test, with the reason, when no git binary is available on this machine: a
// machine without the tool is a normal state, following the skip-versus-fail asymmetry
// internal/engine/loomyard_test.go's own loomyardRepo already establishes for a missing checkout.
func newFixtureRepo(t testing.TB) *fixtureRepo {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not found on this machine")
	}

	f := &fixtureRepo{t: t, root: t.TempDir()}
	f.git("init", "--quiet", "--initial-branch=main")
	f.git("config", "user.name", "gitsrc-fixture")
	f.git("config", "user.email", "gitsrc-fixture@example.com")
	return f
}

// git runs one git invocation against the fixture's root, failing the test immediately on error: a
// fixture-construction step that fails is a broken test, never a normal state worth skipping.
func (f *fixtureRepo) git(args ...string) string {
	f.t.Helper()
	cmd := exec.Command("git", append([]string{"-C", f.root}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		f.t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

// write writes content to path, repository-relative, creating parent directories as needed. It
// does not stage the write.
func (f *fixtureRepo) write(path, content string) {
	f.t.Helper()
	full := filepath.Join(f.root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		f.t.Fatalf("mkdir %q: %v", filepath.Dir(full), err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		f.t.Fatalf("write %q: %v", full, err)
	}
}

// commit stages every pending change and commits it with message, returning the resulting
// commit's identifier.
func (f *fixtureRepo) commit(message string) string {
	f.git("add", "-A")
	f.git("commit", "--quiet", "-m", message)
	return f.git("rev-parse", "HEAD")
}

// writeAndCommit writes content to path and commits it in one step, returning the resulting
// commit's identifier.
func (f *fixtureRepo) writeAndCommit(path, content, message string) string {
	f.write(path, content)
	return f.commit(message)
}

// remove deletes path from the working tree. When staged is true the deletion is staged with
// "git rm"; otherwise it is a plain filesystem removal, leaving the deletion unstaged.
func (f *fixtureRepo) remove(path string, staged bool) {
	f.t.Helper()
	if staged {
		f.git("rm", "--quiet", path)
		return
	}
	full := filepath.Join(f.root, filepath.FromSlash(path))
	if err := os.Remove(full); err != nil {
		f.t.Fatalf("remove %q: %v", full, err)
	}
}

// rename renames a tracked path with "git mv", leaving the rename staged but not committed.
func (f *fixtureRepo) rename(oldPath, newPath string) {
	f.git("mv", oldPath, newPath)
}

// writeGitignore writes and commits a .gitignore holding the given pattern lines.
func (f *fixtureRepo) writeGitignore(patterns ...string) {
	f.writeAndCommit(".gitignore", strings.Join(patterns, "\n")+"\n", "add .gitignore")
}

// leaveUntracked writes content to path without staging or committing it, leaving it untracked.
func (f *fixtureRepo) leaveUntracked(path, content string) {
	f.write(path, content)
}

// makeUnmergedPath produces a conflicted path reachable on the working-tree side during a merge: it
// commits a base version of path, changes it on a side branch, changes it again on main, then
// attempts to merge the side branch into main and expects, rather than resolves, the resulting
// conflict.
func (f *fixtureRepo) makeUnmergedPath(path string) {
	f.t.Helper()
	f.writeAndCommit(path, "base content\n", "base commit for "+path)
	f.git("checkout", "--quiet", "-b", "gitsrc-fixture-conflict")
	f.writeAndCommit(path, "conflict branch content\n", "conflict branch change")
	f.git("checkout", "--quiet", "main")
	f.writeAndCommit(path, "main branch content\n", "main branch change")

	cmd := exec.Command("git", "-C", f.root, "merge", "--quiet", "--no-edit", "gitsrc-fixture-conflict")
	// The merge is expected to fail with a conflict on path; a merge that succeeds cleanly, leaving
	// no unmerged path at all, is the fixture-construction failure worth stopping the test over.
	if err := cmd.Run(); err == nil {
		f.t.Fatalf("merge of gitsrc-fixture-conflict into main succeeded cleanly; wanted a conflict on %q", path)
	}
}
