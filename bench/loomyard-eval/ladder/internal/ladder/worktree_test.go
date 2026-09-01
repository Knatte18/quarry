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

func TestNeutraliseWorktree_RemovesAmbientContextEntries(t *testing.T) {
	worktree := t.TempDir()
	if err := os.WriteFile(filepath.Join(worktree, "CLAUDE.md"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write CLAUDE.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(worktree, "CONSTRAINTS.md"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write CONSTRAINTS.md: %v", err)
	}
	if err := os.Mkdir(filepath.Join(worktree, ".claude"), 0o755); err != nil {
		t.Fatalf("mkdir .claude: %v", err)
	}

	if err := NeutraliseWorktree(worktree); err != nil {
		t.Fatalf("NeutraliseWorktree() error = %v", err)
	}

	if findings := GateWorktreeNeutralised(worktree); len(findings) != 0 {
		t.Errorf("GateWorktreeNeutralised() after NeutraliseWorktree() = %+v; want empty", findings)
	}
}

func TestNeutraliseWorktree_NoErrorWhenNothingPresent(t *testing.T) {
	worktree := t.TempDir()
	if err := NeutraliseWorktree(worktree); err != nil {
		t.Errorf("NeutraliseWorktree() error = %v; want nil for an already-clean worktree", err)
	}
}

func TestBuildWorktree_RefusesAnExistingDirectory(t *testing.T) {
	path := t.TempDir()
	var calls [][]string
	git := func(args ...string) (string, error) {
		calls = append(calls, args)
		return "", nil
	}

	err := BuildWorktree("/source", path, "deadbeef", git)
	if err == nil {
		t.Fatal("BuildWorktree() = nil; want a *HarnessError for a pre-existing directory")
	}
	if _, ok := err.(*HarnessError); !ok {
		t.Errorf("BuildWorktree() error type = %T; want *HarnessError", err)
	}
	if len(calls) != 0 {
		t.Errorf("git calls = %v; want none, since BuildWorktree must refuse before touching git", calls)
	}
}

func TestBuildWorktree_AddsAndNeutralisesAMissingWorktree(t *testing.T) {
	sourceRepo := t.TempDir()
	initGitRepo(t, sourceRepo)
	path := filepath.Join(t.TempDir(), "task-worktree")

	var calls [][]string
	git := func(args ...string) (string, error) {
		calls = append(calls, args)
		if args[0] == "-C" && len(args) > 2 && args[2] == "worktree" {
			if err := os.MkdirAll(path, 0o755); err != nil {
				return "", err
			}
			if err := os.WriteFile(filepath.Join(path, "CLAUDE.md"), []byte("x"), 0o644); err != nil {
				return "", err
			}
		}
		return "", nil
	}

	if err := BuildWorktree(sourceRepo, path, "deadbeef", git); err != nil {
		t.Fatalf("BuildWorktree() error = %v", err)
	}

	wantArgs := []string{"-C", sourceRepo, "worktree", "add", path, "deadbeef"}
	if len(calls) != 1 || strings.Join(calls[0], " ") != strings.Join(wantArgs, " ") {
		t.Errorf("git calls = %v; want exactly one call with args %v", calls, wantArgs)
	}
	if _, err := os.Stat(filepath.Join(path, "CLAUDE.md")); !os.IsNotExist(err) {
		t.Errorf("CLAUDE.md still present after BuildWorktree(); want it neutralised away")
	}
}

func TestRestoreWorktree_IssuesResetCleanThenNeutralises(t *testing.T) {
	path := t.TempDir()
	if err := os.WriteFile(filepath.Join(path, "CLAUDE.md"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write CLAUDE.md: %v", err)
	}

	var calls [][]string
	git := func(args ...string) (string, error) {
		calls = append(calls, args)
		return "", nil
	}

	if err := RestoreWorktree(path, git); err != nil {
		t.Fatalf("RestoreWorktree() error = %v", err)
	}

	wantCalls := [][]string{
		{"-C", path, "reset", "--hard"},
		{"-C", path, "clean", "-fdx"},
	}
	if len(calls) != len(wantCalls) {
		t.Fatalf("git calls = %v; want %v", calls, wantCalls)
	}
	for i := range wantCalls {
		if strings.Join(calls[i], " ") != strings.Join(wantCalls[i], " ") {
			t.Errorf("git call[%d] = %v; want %v", i, calls[i], wantCalls[i])
		}
	}
	if _, err := os.Stat(filepath.Join(path, "CLAUDE.md")); !os.IsNotExist(err) {
		t.Errorf("CLAUDE.md still present after RestoreWorktree(); want it re-neutralised")
	}
}

func TestRemoveWorktree_IssuesWorktreeRemoveForce(t *testing.T) {
	var calls [][]string
	git := func(args ...string) (string, error) {
		calls = append(calls, args)
		return "", nil
	}

	if err := RemoveWorktree("/source", "/task-worktree", git); err != nil {
		t.Fatalf("RemoveWorktree() error = %v", err)
	}

	want := []string{"-C", "/source", "worktree", "remove", "--force", "/task-worktree"}
	if len(calls) != 1 || strings.Join(calls[0], " ") != strings.Join(want, " ") {
		t.Errorf("git calls = %v; want exactly one call with args %v", calls, want)
	}
}

func realLadderWithOneTaskAt(worktree, pinnedSHA string) *Ladder {
	return &Ladder{
		SourceRepo: "/source",
		Tasks: map[string]TaskEntry{
			"t": {Worktree: worktree, PinnedSHA: pinnedSHA},
		},
	}
}

func TestEnsureTaskWorktrees_BuildsAMissingWorktree(t *testing.T) {
	missingPath := filepath.Join(t.TempDir(), "missing-worktree")
	l := realLadderWithOneTaskAt(missingPath, "deadbeef")

	var calls [][]string
	git := func(args ...string) (string, error) {
		calls = append(calls, args)
		return "", nil
	}

	worktrees, err := EnsureTaskWorktrees(l, l.SourceRepo, git)
	if err != nil {
		t.Fatalf("EnsureTaskWorktrees() error = %v", err)
	}
	if worktrees["t"] != missingPath {
		t.Errorf("worktrees[%q] = %q; want %q", "t", worktrees["t"], missingPath)
	}
	if _, err := os.Stat(missingPath); !os.IsNotExist(err) {
		t.Errorf("missingPath exists after EnsureTaskWorktrees() with a fake git that created nothing")
	}

	foundAdd := false
	for _, call := range calls {
		for _, arg := range call {
			if arg == "add" {
				foundAdd = true
			}
		}
	}
	if !foundAdd {
		t.Errorf("git calls = %v; want a worktree add call for the missing path", calls)
	}
}

func TestEnsureTaskWorktrees_AdoptsAnExistingWorktreeAtTheMatchingPin(t *testing.T) {
	existingPath := t.TempDir()
	pin := "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
	l := realLadderWithOneTaskAt(existingPath, pin)

	git := func(args ...string) (string, error) {
		for _, arg := range args {
			if arg == "rev-parse" {
				return pin + "\n", nil
			}
		}
		return "", nil
	}

	worktrees, err := EnsureTaskWorktrees(l, l.SourceRepo, git)
	if err != nil {
		t.Fatalf("EnsureTaskWorktrees() error = %v", err)
	}
	if worktrees["t"] != existingPath {
		t.Errorf("worktrees[%q] = %q; want %q", "t", worktrees["t"], existingPath)
	}
}

func TestEnsureTaskWorktrees_LeavesACorrectlyPinnedWorktreeAlone(t *testing.T) {
	existingPath := t.TempDir()
	pin := "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
	l := realLadderWithOneTaskAt(existingPath, pin)

	var calls [][]string
	git := func(args ...string) (string, error) {
		calls = append(calls, args)
		for _, arg := range args {
			if arg == "rev-parse" {
				return pin + "\n", nil
			}
		}
		return "", nil
	}

	if _, err := EnsureTaskWorktrees(l, l.SourceRepo, git); err != nil {
		t.Fatalf("EnsureTaskWorktrees() error = %v", err)
	}

	for _, call := range calls {
		for _, arg := range call {
			if arg == "add" {
				t.Errorf("git calls = %v; want no worktree add call for a correctly-pinned worktree", calls)
			}
		}
	}
}

func TestEnsureTaskWorktrees_RaisesWhenExistingWorktreeIsAtTheWrongPin(t *testing.T) {
	existingPath := t.TempDir()
	l := realLadderWithOneTaskAt(existingPath, "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef")

	git := func(args ...string) (string, error) {
		for _, arg := range args {
			if arg == "rev-parse" {
				return "cafecafecafecafecafecafecafecafecafecafe\n", nil
			}
		}
		return "", nil
	}

	_, err := EnsureTaskWorktrees(l, l.SourceRepo, git)
	if err == nil {
		t.Fatal("EnsureTaskWorktrees() = nil error; want a *HarnessError for a mispinned worktree")
	}
	if _, ok := err.(*HarnessError); !ok {
		t.Errorf("EnsureTaskWorktrees() error type = %T; want *HarnessError", err)
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
