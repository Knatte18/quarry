package ladder

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// initGitRepo creates a bare-minimum git repository at dir, with an identity configured so a commit
// can be made against it -- mirroring the Python test suite's own _init_git_repo helper.
func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	for _, args := range [][]string{
		{"-C", dir, "init", "-q"},
		{"-C", dir, "config", "user.email", "test@example.com"},
		{"-C", dir, "config", "user.name", "test"},
	} {
		if _, err := RunGit(args...); err != nil {
			t.Fatalf("RunGit(%v) = _, %v; want nil error", args, err)
		}
	}
}

func TestRunGit_ReturnsOutputOnSuccess(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)

	output, err := RunGit("-C", dir, "status", "--porcelain")
	if err != nil {
		t.Fatalf("RunGit() = _, %v; want nil error", err)
	}
	if output != "" {
		t.Errorf("RunGit() = %q; want empty output for a clean repo", output)
	}
}

func TestRunGit_ErrorsAndCapturesOutputOnFailure(t *testing.T) {
	dir := t.TempDir()

	_, err := RunGit("-C", dir, "not-a-real-git-command")
	if err == nil {
		t.Fatal("RunGit() = _, nil; want an error for a non-zero exit")
	}
	gitErr, ok := err.(*RunGitError)
	if !ok {
		t.Fatalf("RunGit() error type = %T; want *RunGitError", err)
	}
	if gitErr.Output == "" {
		t.Error("RunGitError.Output is empty; want the child's captured output")
	}
	if !strings.Contains(gitErr.Error(), "not-a-real-git-command") {
		t.Errorf("RunGitError.Error() = %q; want it to name the failing args", gitErr.Error())
	}
}

func TestGateWorktreeNeutralised_FailsForEachAmbientContextEntry(t *testing.T) {
	for _, entry := range []string{"CLAUDE.md", "CONSTRAINTS.md", ".claude"} {
		t.Run(entry, func(t *testing.T) {
			worktree := t.TempDir()
			path := filepath.Join(worktree, entry)
			var err error
			if entry == ".claude" {
				err = os.Mkdir(path, 0o755)
			} else {
				err = os.WriteFile(path, []byte("x"), 0o644)
			}
			if err != nil {
				t.Fatalf("create %s: %v", entry, err)
			}

			findings := GateWorktreeNeutralised(worktree)
			if len(findings) != 1 || !findings[0].Fatal || findings[0].Gate != "worktree_neutralised" {
				t.Errorf("GateWorktreeNeutralised() = %+v; want one fatal worktree_neutralised finding", findings)
			}
		})
	}
}

func TestGateWorktreeNeutralised_PassesWhenNonePresent(t *testing.T) {
	worktree := t.TempDir()
	if findings := GateWorktreeNeutralised(worktree); len(findings) != 0 {
		t.Errorf("GateWorktreeNeutralised() = %+v; want empty", findings)
	}
}

func TestObserveWorktreeDirtied_NonFatalWhenClean(t *testing.T) {
	worktree := t.TempDir()
	initGitRepo(t, worktree)

	finding := ObserveWorktreeDirtied(worktree)
	if finding.Fatal {
		t.Error("ObserveWorktreeDirtied().Fatal = true; want false")
	}
	if finding.Gate != "worktree_dirtied" {
		t.Errorf("ObserveWorktreeDirtied().Gate = %q; want worktree_dirtied", finding.Gate)
	}
	if !strings.Contains(finding.Message, "false") {
		t.Errorf("ObserveWorktreeDirtied().Message = %q; want it to report false", finding.Message)
	}
}

func TestObserveWorktreeDirtied_NonFatalWhenDirty(t *testing.T) {
	worktree := t.TempDir()
	initGitRepo(t, worktree)
	if err := os.WriteFile(filepath.Join(worktree, "scratch.txt"), []byte("dirty"), 0o644); err != nil {
		t.Fatalf("write scratch file: %v", err)
	}

	finding := ObserveWorktreeDirtied(worktree)
	if finding.Fatal {
		t.Error("ObserveWorktreeDirtied().Fatal = true; want false")
	}
	if !strings.Contains(finding.Message, "true") {
		t.Errorf("ObserveWorktreeDirtied().Message = %q; want it to report true", finding.Message)
	}
}
